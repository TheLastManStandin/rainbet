package database

import (
	"database/sql"
	"fmt"

	_ "github.com/mattn/go-sqlite3"
	"golang.org/x/crypto/bcrypt"
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
			balanceDollars BIGINT NOT NULL DEFAULT 0
		)
	`); err != nil {
		return fmt.Errorf("create users table: %w", err)
	}

	balanceColumnAdded, err := ensureBalanceDollarsColumn(db)
	if err != nil {
		return err
	}

	passwordHash, err := bcrypt.GenerateFromPassword([]byte(defaultPassword), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("hash default user password: %w", err)
	}

	if _, err := db.Exec(`
		INSERT INTO users (username, password_hash, balanceDollars)
		VALUES (?, ?, ?)
		ON CONFLICT(username) DO NOTHING
	`, defaultUsername, string(passwordHash), defaultBalanceDollars); err != nil {
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

	return nil
}

func ensureBalanceDollarsColumn(db *sql.DB) (bool, error) {
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
		if name == "balanceDollars" {
			return false, nil
		}
	}
	if err := rows.Err(); err != nil {
		return false, fmt.Errorf("iterate users table columns: %w", err)
	}

	if _, err := db.Exec("ALTER TABLE users ADD COLUMN balanceDollars BIGINT NOT NULL DEFAULT 0"); err != nil {
		return false, fmt.Errorf("add users balance column: %w", err)
	}

	return true, nil
}
