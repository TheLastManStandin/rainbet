package database

import (
	"database/sql"
	"fmt"

	_ "github.com/mattn/go-sqlite3"
	"golang.org/x/crypto/bcrypt"

	"rainbet/internal/provablyfair"
)

const (
	defaultUsername       = "user"
	defaultPassword       = "user"
	defaultBalanceDollars = 100
)

func OpenSQLite(path string) (*sql.DB, error) {
	db, err := sql.Open("sqlite3", path)
	if err != nil {
		return nil, fmt.Errorf("open SQLite database: %w", err)
	}

	if err := migrate(db); err != nil {
		_ = db.Close()
		return nil, err
	}

	return db, nil
}

func migrate(db *sql.DB) error {
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

	balanceColumnAdded, err := ensureColumn(db, "balanceDollars", "balanceDollars BIGINT NOT NULL DEFAULT 0")
	if err != nil {
		return err
	}
	if _, err := ensureColumn(db, "serverSeed", "serverSeed TEXT NOT NULL DEFAULT ''"); err != nil {
		return err
	}
	if _, err := ensureColumn(db, "transactionNumber", "transactionNumber BIGINT NOT NULL DEFAULT 0"); err != nil {
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
	`, defaultUsername, string(passwordHash), defaultBalanceDollars, serverSeed); err != nil {
		return fmt.Errorf("seed default user: %w", err)
	}

	if balanceColumnAdded {
		if _, err := db.Exec(
			"UPDATE users SET balanceDollars = ? WHERE username = ?",
			defaultBalanceDollars,
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

func ensureColumn(db *sql.DB, columnName, definition string) (bool, error) {
	rows, err := db.Query("PRAGMA table_info(users)")
	if err != nil {
		return false, fmt.Errorf("inspect users table: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var id, notNull, primaryKey int
		var name, dataType string
		var defaultValue any
		if err := rows.Scan(&id, &name, &dataType, &notNull, &defaultValue, &primaryKey); err != nil {
			return false, fmt.Errorf("read users table columns: %w", err)
		}
		if name == columnName {
			return false, nil
		}
	}
	if err := rows.Err(); err != nil {
		return false, fmt.Errorf("iterate users table columns: %w", err)
	}

	if _, err := db.Exec("ALTER TABLE users ADD COLUMN " + definition); err != nil {
		return false, fmt.Errorf("add users %s column: %w", columnName, err)
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
