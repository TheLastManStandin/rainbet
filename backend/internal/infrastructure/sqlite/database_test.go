package sqlite

import (
	"database/sql"
	"path/filepath"
	"testing"
)

func TestOpenSeedsDefaultAccountAndSchema(t *testing.T) {
	db := openTestDatabase(t)
	var balance, nonce int64
	var seed string
	if err := db.QueryRow(
		"SELECT balanceDollars, serverSeed, transactionNumber FROM users WHERE username = ?",
		defaultUsername,
	).Scan(&balance, &seed, &nonce); err != nil {
		t.Fatalf("read default account: %v", err)
	}
	if balance != defaultBalanceCents || seed == "" || nonce != 0 {
		t.Fatalf("default account = balance:%d seed:%q nonce:%d", balance, seed, nonce)
	}
}

func TestOpenMigratesLegacySchema(t *testing.T) {
	path := filepath.Join(t.TempDir(), "rainbet.db")
	legacy, err := sql.Open("sqlite3", path)
	if err != nil {
		t.Fatalf("open legacy database: %v", err)
	}
	if _, err := legacy.Exec(`
		CREATE TABLE users (id INTEGER PRIMARY KEY AUTOINCREMENT, username TEXT NOT NULL UNIQUE, password_hash TEXT NOT NULL);
		INSERT INTO users (username, password_hash) VALUES ('user', 'legacy-hash');
		CREATE TABLE games (
			id INTEGER PRIMARY KEY AUTOINCREMENT, userId INTEGER NOT NULL,
			openedCells TEXT NOT NULL DEFAULT '[]', status TEXT NOT NULL DEFAULT 'inProcess',
			serverSeed TEXT NOT NULL, clientSeed TEXT NOT NULL, nonce BIGINT NOT NULL,
			startedAt DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		);
	`); err != nil {
		t.Fatalf("create legacy schema: %v", err)
	}
	if err := legacy.Close(); err != nil {
		t.Fatalf("close legacy database: %v", err)
	}
	db, err := Open(path)
	if err != nil {
		t.Fatalf("migrate database: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	var balance int64
	var seed string
	if err := db.QueryRow("SELECT balanceDollars, serverSeed FROM users WHERE username = 'user'").Scan(&balance, &seed); err != nil {
		t.Fatalf("read migrated account: %v", err)
	}
	if balance != defaultBalanceCents || seed == "" {
		t.Fatalf("migrated account = balance:%d seed:%q", balance, seed)
	}
	var columns int
	if err := db.QueryRow("SELECT COUNT(*) FROM pragma_table_info('games') WHERE name IN ('betAmount','gridSize','mines','demo')").Scan(&columns); err != nil {
		t.Fatalf("inspect migrated games: %v", err)
	}
	if columns != 4 {
		t.Fatalf("migrated game columns = %d, want 4", columns)
	}
}

func TestOpenConvertsLegacyMoneyOnce(t *testing.T) {
	path := filepath.Join(t.TempDir(), "rainbet.db")
	legacy, err := sql.Open("sqlite3", path)
	if err != nil {
		t.Fatalf("open legacy database: %v", err)
	}
	if _, err := legacy.Exec(`
		CREATE TABLE users (
			id INTEGER PRIMARY KEY, username TEXT UNIQUE, password_hash TEXT,
			balanceDollars BIGINT, serverSeed TEXT, transactionNumber BIGINT
		);
		CREATE TABLE games (
			id INTEGER PRIMARY KEY, userId INTEGER, betAmount BIGINT, gridSize INTEGER, mines INTEGER,
			demo BOOLEAN, openedCells TEXT, status TEXT, serverSeed TEXT, clientSeed TEXT,
			nonce BIGINT, startedAt DATETIME
		);
		INSERT INTO users VALUES (1, 'existing', 'hash', 100, 'seed', 0);
		INSERT INTO games VALUES (1, 1, 10, 25, 3, 0, '[]', 'inProcess', 'seed', 'client', 0, CURRENT_TIMESTAMP);
	`); err != nil {
		t.Fatalf("create money fixture: %v", err)
	}
	_ = legacy.Close()
	for iteration := 0; iteration < 2; iteration++ {
		db, err := Open(path)
		if err != nil {
			t.Fatalf("open iteration %d: %v", iteration, err)
		}
		var balance, bet int64
		if err := db.QueryRow("SELECT balanceDollars FROM users WHERE username = 'existing'").Scan(&balance); err != nil {
			t.Fatalf("read balance: %v", err)
		}
		if err := db.QueryRow("SELECT betAmount FROM games WHERE id = 1").Scan(&bet); err != nil {
			t.Fatalf("read bet: %v", err)
		}
		_ = db.Close()
		if balance != 10_000 || bet != 1_000 {
			t.Fatalf("iteration %d: balance = %d, bet = %d", iteration, balance, bet)
		}
	}
}

func openTestDatabase(t *testing.T) *sql.DB {
	t.Helper()
	db, err := Open(filepath.Join(t.TempDir(), "rainbet.db"))
	if err != nil {
		t.Fatalf("open test database: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	return db
}
