package database

import (
	"database/sql"
	"errors"
	"fmt"
	"strings"

	_ "github.com/mattn/go-sqlite3"
	"golang.org/x/crypto/bcrypt"

	"rainbet/internal/provablyfair"
)

const (
	defaultUsername     = "user"
	defaultPassword     = "user"
	defaultBalanceCents = 10_000
	moneyCentsMigration = "money_cents_v1"
)

func OpenSQLite(path string) (*sql.DB, error) {
	db, err := sql.Open("sqlite3", sqliteDSN(path))
	if err != nil {
		return nil, fmt.Errorf("open SQLite database: %w", err)
	}

	if err := migrate(db); err != nil {
		_ = db.Close()
		return nil, err
	}

	return db, nil
}

func sqliteDSN(path string) string {
	if strings.Contains(path, "?") {
		return path + "&_foreign_keys=on"
	}

	return path + "?_foreign_keys=on"
}

func migrate(db *sql.DB) error {
	if _, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS schema_migrations (
			name TEXT PRIMARY KEY
		)
	`); err != nil {
		return fmt.Errorf("create schema migrations table: %w", err)
	}
	if _, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS users (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			username TEXT NOT NULL UNIQUE,
			password_hash TEXT NOT NULL,
			balanceDollars BIGINT NOT NULL DEFAULT 0,
			serverSeed TEXT NOT NULL,
			transactionNumber BIGINT NOT NULL DEFAULT 0
		)
	`); err != nil {
		return fmt.Errorf("create users table: %w", err)
	}
	if _, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS games (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			userId INTEGER NOT NULL,
			betAmount BIGINT NOT NULL DEFAULT 0,
			gridSize INTEGER NOT NULL DEFAULT 0,
			mines INTEGER NOT NULL DEFAULT 0,
			demo BOOLEAN NOT NULL DEFAULT 0,
			openedCells TEXT NOT NULL DEFAULT '[]',
			status TEXT NOT NULL DEFAULT 'inProcess' CHECK (status IN ('inProcess', 'cachedOut', 'failed')),
			serverSeed TEXT NOT NULL,
			clientSeed TEXT NOT NULL,
			nonce BIGINT NOT NULL,
			startedAt DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (userId) REFERENCES users(id)
		)
	`); err != nil {
		return fmt.Errorf("create games table: %w", err)
	}
	if _, err := db.Exec("CREATE INDEX IF NOT EXISTS games_user_id_idx ON games(userId)"); err != nil {
		return fmt.Errorf("create games user index: %w", err)
	}

	if _, err := ensureColumn(db, "games", "betAmount", "betAmount BIGINT NOT NULL DEFAULT 0"); err != nil {
		return err
	}
	if _, err := ensureColumn(db, "games", "gridSize", "gridSize INTEGER NOT NULL DEFAULT 0"); err != nil {
		return err
	}
	if _, err := ensureColumn(db, "games", "mines", "mines INTEGER NOT NULL DEFAULT 0"); err != nil {
		return err
	}
	if _, err := ensureColumn(db, "games", "demo", "demo BOOLEAN NOT NULL DEFAULT 0"); err != nil {
		return err
	}

	balanceColumnAdded, err := ensureColumn(db, "users", "balanceDollars", "balanceDollars BIGINT NOT NULL DEFAULT 0")
	if err != nil {
		return err
	}
	if _, err := ensureColumn(db, "users", "serverSeed", "serverSeed TEXT NOT NULL DEFAULT ''"); err != nil {
		return err
	}
	if _, err := ensureColumn(db, "users", "transactionNumber", "transactionNumber BIGINT NOT NULL DEFAULT 0"); err != nil {
		return err
	}
	if err := migrateMoneyToCents(db); err != nil {
		return err
	}

	passwordHash, err := bcrypt.GenerateFromPassword([]byte(defaultPassword), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("hash default user password: %w", err)
	}
	serverSeed, err := provablyfair.GenerateServerSeed()
	if err != nil {
		return err
	}

	if _, err := db.Exec(`
		INSERT INTO users (username, password_hash, balanceDollars, serverSeed, transactionNumber)
		VALUES (?, ?, ?, ?, 0)
		ON CONFLICT(username) DO NOTHING
	`, defaultUsername, string(passwordHash), defaultBalanceCents, serverSeed); err != nil {
		return fmt.Errorf("seed default user: %w", err)
	}

	if balanceColumnAdded {
		if _, err := db.Exec(
			"UPDATE users SET balanceDollars = ? WHERE username = ?",
			defaultBalanceCents,
			defaultUsername,
		); err != nil {
			return fmt.Errorf("set default user balance: %w", err)
		}
	}
	if err := backfillServerSeeds(db); err != nil {
		return err
	}
	if _, err := db.Exec("UPDATE users SET transactionNumber = 0 WHERE transactionNumber IS NULL"); err != nil {
		return fmt.Errorf("set missing transaction numbers: %w", err)
	}

	return nil
}

func migrateMoneyToCents(db *sql.DB) error {
	tx, err := db.Begin()
	if err != nil {
		return fmt.Errorf("begin money migration: %w", err)
	}
	defer tx.Rollback()

	var name string
	err = tx.QueryRow("SELECT name FROM schema_migrations WHERE name = ?", moneyCentsMigration).Scan(&name)
	if err == nil {
		if err := tx.Commit(); err != nil {
			return fmt.Errorf("commit checked money migration: %w", err)
		}
		return nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return fmt.Errorf("check money migration: %w", err)
	}

	if _, err := tx.Exec("UPDATE users SET balanceDollars = balanceDollars * 100"); err != nil {
		return fmt.Errorf("convert user balances to cents: %w", err)
	}
	if _, err := tx.Exec("UPDATE games SET betAmount = betAmount * 100"); err != nil {
		return fmt.Errorf("convert game bets to cents: %w", err)
	}
	if _, err := tx.Exec("INSERT INTO schema_migrations (name) VALUES (?)", moneyCentsMigration); err != nil {
		return fmt.Errorf("record money migration: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit money migration: %w", err)
	}

	return nil
}

func ensureColumn(db *sql.DB, tableName, columnName, definition string) (bool, error) {
	rows, err := db.Query("PRAGMA table_info(" + tableName + ")")
	if err != nil {
		return false, fmt.Errorf("inspect %s table: %w", tableName, err)
	}
	defer rows.Close()

	for rows.Next() {
		var id, notNull, primaryKey int
		var name, dataType string
		var defaultValue any
		if err := rows.Scan(&id, &name, &dataType, &notNull, &defaultValue, &primaryKey); err != nil {
			return false, fmt.Errorf("read %s table columns: %w", tableName, err)
		}
		if name == columnName {
			return false, nil
		}
	}
	if err := rows.Err(); err != nil {
		return false, fmt.Errorf("iterate %s table columns: %w", tableName, err)
	}

	if _, err := db.Exec("ALTER TABLE " + tableName + " ADD COLUMN " + definition); err != nil {
		return false, fmt.Errorf("add %s %s column: %w", tableName, columnName, err)
	}

	return true, nil
}

func backfillServerSeeds(db *sql.DB) error {
	rows, err := db.Query("SELECT id FROM users WHERE serverSeed IS NULL OR serverSeed = ''")
	if err != nil {
		return fmt.Errorf("find users without server seeds: %w", err)
	}
	defer rows.Close()

	var userIDs []int64
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return fmt.Errorf("read user without server seed: %w", err)
		}
		userIDs = append(userIDs, id)
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("iterate users without server seeds: %w", err)
	}
	if err := rows.Close(); err != nil {
		return fmt.Errorf("close users without server seeds: %w", err)
	}

	for _, id := range userIDs {
		serverSeed, err := provablyfair.GenerateServerSeed()
		if err != nil {
			return err
		}
		if _, err := db.Exec("UPDATE users SET serverSeed = ? WHERE id = ?", serverSeed, id); err != nil {
			return fmt.Errorf("set user server seed: %w", err)
		}
	}

	return nil
}
