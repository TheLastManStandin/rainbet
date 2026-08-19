package database

import (
	"database/sql"
	"fmt"

	_ "github.com/mattn/go-sqlite3"
	"golang.org/x/crypto/bcrypt"
)

const (
	defaultUsername = "user"
	defaultPassword = "user"
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
			password_hash TEXT NOT NULL
		)
	`); err != nil {
		return fmt.Errorf("create users table: %w", err)
	}

	passwordHash, err := bcrypt.GenerateFromPassword([]byte(defaultPassword), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("hash default user password: %w", err)
	}

	if _, err := db.Exec(`
		INSERT INTO users (username, password_hash)
		VALUES (?, ?)
		ON CONFLICT(username) DO NOTHING
	`, defaultUsername, string(passwordHash)); err != nil {
		return fmt.Errorf("seed default user: %w", err)
	}

	return nil
}
