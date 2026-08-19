package database

import (
	"database/sql"
	"path/filepath"
	"testing"
)

func TestOpenSQLiteSeedsDefaultUserBalance(t *testing.T) {
	db := openTestDatabase(t)

	var balance int64
	if err := db.QueryRow(
		"SELECT balanceDollars FROM users WHERE username = ?",
		defaultUsername,
	).Scan(&balance); err != nil {
		t.Fatalf("read default user balance: %v", err)
	}

	if balance != defaultBalanceDollars {
		t.Fatalf("balance = %d, want %d", balance, defaultBalanceDollars)
	}
}

func TestOpenSQLiteMigratesDefaultUserBalance(t *testing.T) {
	path := filepath.Join(t.TempDir(), "rainbet.db")
	legacyDB, err := sql.Open("sqlite3", path)
	if err != nil {
		t.Fatalf("open legacy database: %v", err)
	}

	if _, err := legacyDB.Exec(`
		CREATE TABLE users (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			username TEXT NOT NULL UNIQUE,
			password_hash TEXT NOT NULL
		)
	`); err != nil {
		t.Fatalf("create legacy users table: %v", err)
	}
	if _, err := legacyDB.Exec(
		"INSERT INTO users (username, password_hash) VALUES (?, ?)",
		defaultUsername,
		"legacy-password-hash",
	); err != nil {
		t.Fatalf("seed legacy user: %v", err)
	}
	if err := legacyDB.Close(); err != nil {
		t.Fatalf("close legacy database: %v", err)
	}

	db, err := OpenSQLite(path)
	if err != nil {
		t.Fatalf("migrate database: %v", err)
	}
	t.Cleanup(func() {
		_ = db.Close()
	})

	var balance int64
	if err := db.QueryRow(
		"SELECT balanceDollars FROM users WHERE username = ?",
		defaultUsername,
	).Scan(&balance); err != nil {
		t.Fatalf("read migrated user balance: %v", err)
	}

	if balance != defaultBalanceDollars {
		t.Fatalf("balance = %d, want %d", balance, defaultBalanceDollars)
	}
}

func openTestDatabase(t *testing.T) *sql.DB {
	t.Helper()

	db, err := OpenSQLite(filepath.Join(t.TempDir(), "rainbet.db"))
	if err != nil {
		t.Fatalf("open test database: %v", err)
	}
	t.Cleanup(func() {
		_ = db.Close()
	})

	return db
}
