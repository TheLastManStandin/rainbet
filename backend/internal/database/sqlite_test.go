package database

import (
	"database/sql"
	"path/filepath"
	"testing"
)

func TestOpenSQLiteSeedsDefaultUserBalance(t *testing.T) {
	db := openTestDatabase(t)

	var (
		balance           int64
		serverSeed        string
		transactionNumber int64
	)
	if err := db.QueryRow(
		"SELECT balanceDollars, serverSeed, transactionNumber FROM users WHERE username = ?",
		defaultUsername,
	).Scan(&balance, &serverSeed, &transactionNumber); err != nil {
		t.Fatalf("read default user: %v", err)
	}

	if balance != defaultBalanceCents {
		t.Fatalf("balance = %d, want %d", balance, defaultBalanceCents)
	}
	if serverSeed == "" {
		t.Fatal("server seed is empty")
	}
	if transactionNumber != 0 {
		t.Fatalf("transaction number = %d, want 0", transactionNumber)
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
	if _, err := legacyDB.Exec(
		"INSERT INTO users (username, password_hash) VALUES (?, ?)",
		"existing-user",
		"legacy-password-hash",
	); err != nil {
		t.Fatalf("seed existing legacy user: %v", err)
	}
	if _, err := legacyDB.Exec(`
		CREATE TABLE games (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			userId INTEGER NOT NULL,
			openedCells TEXT NOT NULL DEFAULT '[]',
			status TEXT NOT NULL DEFAULT 'inProcess',
			serverSeed TEXT NOT NULL,
			clientSeed TEXT NOT NULL,
			nonce BIGINT NOT NULL,
			startedAt DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		)
	`); err != nil {
		t.Fatalf("create legacy games table: %v", err)
	}
	if _, err := legacyDB.Exec(`
		INSERT INTO games (userId, serverSeed, clientSeed, nonce)
		VALUES (?, ?, ?, ?)
	`, 1, "legacy-server-seed", "legacy-client-seed", 0); err != nil {
		t.Fatalf("seed legacy game: %v", err)
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

	var (
		balance           int64
		serverSeed        string
		transactionNumber int64
	)
	if err := db.QueryRow(
		"SELECT balanceDollars, serverSeed, transactionNumber FROM users WHERE username = ?",
		defaultUsername,
	).Scan(&balance, &serverSeed, &transactionNumber); err != nil {
		t.Fatalf("read migrated user: %v", err)
	}

	if balance != defaultBalanceCents {
		t.Fatalf("balance = %d, want %d", balance, defaultBalanceCents)
	}
	if serverSeed == "" {
		t.Fatal("migrated server seed is empty")
	}
	if transactionNumber != 0 {
		t.Fatalf("transaction number = %d, want 0", transactionNumber)
	}

	var usersWithoutFairnessData int
	if err := db.QueryRow(`
		SELECT COUNT(*)
		FROM users
		WHERE serverSeed IS NULL OR serverSeed = '' OR transactionNumber IS NULL
	`).Scan(&usersWithoutFairnessData); err != nil {
		t.Fatalf("count users without fairness data: %v", err)
	}
	if usersWithoutFairnessData != 0 {
		t.Fatalf("users without fairness data = %d, want 0", usersWithoutFairnessData)
	}

	var (
		legacyGameBetAmount int64
		legacyGameGridSize  int
		legacyGameMines     int
		legacyGameDemo      bool
	)
	if err := db.QueryRow("SELECT betAmount, gridSize, mines, demo FROM games LIMIT 1").Scan(
		&legacyGameBetAmount,
		&legacyGameGridSize,
		&legacyGameMines,
		&legacyGameDemo,
	); err != nil {
		t.Fatalf("read migrated game config: %v", err)
	}
	if legacyGameBetAmount != 0 || legacyGameGridSize != 0 || legacyGameMines != 0 || legacyGameDemo {
		t.Fatalf(
			"migrated game config = bet:%d grid:%d mines:%d demo:%t",
			legacyGameBetAmount,
			legacyGameGridSize,
			legacyGameMines,
			legacyGameDemo,
		)
	}
}

func TestOpenSQLiteCreatesGamesTable(t *testing.T) {
	db := openTestDatabase(t)

	var userID int64
	if err := db.QueryRow("SELECT id FROM users WHERE username = ?", defaultUsername).Scan(&userID); err != nil {
		t.Fatalf("read default user ID: %v", err)
	}

	result, err := db.Exec(`
		INSERT INTO games (userId, betAmount, gridSize, mines, demo, serverSeed, clientSeed, nonce)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`, userID, 1000, 25, 3, false, "server-seed", "client-seed", 0)
	if err != nil {
		t.Fatalf("create game: %v", err)
	}
	gameID, err := result.LastInsertId()
	if err != nil {
		t.Fatalf("read game ID: %v", err)
	}

	var (
		openedCells  string
		status       string
		betAmount    int64
		gridSize     int
		mines        int
		demo         bool
		nonce        int64
		hasStartedAt int
	)
	if err := db.QueryRow(`
		SELECT openedCells, status, betAmount, gridSize, mines, demo, nonce, startedAt IS NOT NULL
		FROM games
		WHERE id = ?
	`, gameID).Scan(&openedCells, &status, &betAmount, &gridSize, &mines, &demo, &nonce, &hasStartedAt); err != nil {
		t.Fatalf("read game: %v", err)
	}

	if openedCells != "[]" {
		t.Fatalf("opened cells = %q, want %q", openedCells, "[]")
	}
	if status != "inProcess" {
		t.Fatalf("status = %q, want %q", status, "inProcess")
	}
	if betAmount != 1000 {
		t.Fatalf("bet amount = %d, want 1000", betAmount)
	}
	if gridSize != 25 || mines != 3 || demo {
		t.Fatalf("game config = grid:%d mines:%d demo:%t", gridSize, mines, demo)
	}
	if nonce != 0 {
		t.Fatalf("nonce = %d, want 0", nonce)
	}
	if hasStartedAt != 1 {
		t.Fatalf("startedAt is set = %d, want 1", hasStartedAt)
	}

	if _, err := db.Exec(`
		INSERT INTO games (userId, betAmount, serverSeed, clientSeed, nonce)
		VALUES (?, ?, ?, ?, ?)
	`, -1, 1000, "server-seed", "client-seed", 0); err == nil {
		t.Fatal("game without a user was accepted")
	}

	if _, err := db.Exec(`
		INSERT INTO games (userId, betAmount, status, serverSeed, clientSeed, nonce)
		VALUES (?, ?, ?, ?, ?, ?)
	`, userID, 1000, "unknown", "server-seed", "client-seed", 0); err == nil {
		t.Fatal("game with an invalid status was accepted")
	}
}

func TestOpenSQLiteConvertsExistingMoneyToCentsOnce(t *testing.T) {
	path := filepath.Join(t.TempDir(), "rainbet.db")
	legacyDB, err := sql.Open("sqlite3", path)
	if err != nil {
		t.Fatalf("open legacy database: %v", err)
	}

	if _, err := legacyDB.Exec(`
		CREATE TABLE users (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			username TEXT NOT NULL UNIQUE,
			password_hash TEXT NOT NULL,
			balanceDollars BIGINT NOT NULL,
			serverSeed TEXT NOT NULL,
			transactionNumber BIGINT NOT NULL
		)
	`); err != nil {
		t.Fatalf("create legacy users table: %v", err)
	}
	if _, err := legacyDB.Exec(`
		CREATE TABLE games (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			userId INTEGER NOT NULL,
			betAmount BIGINT NOT NULL,
			gridSize INTEGER NOT NULL,
			mines INTEGER NOT NULL,
			demo BOOLEAN NOT NULL,
			openedCells TEXT NOT NULL,
			status TEXT NOT NULL,
			serverSeed TEXT NOT NULL,
			clientSeed TEXT NOT NULL,
			nonce BIGINT NOT NULL,
			startedAt DATETIME NOT NULL
		)
	`); err != nil {
		t.Fatalf("create legacy games table: %v", err)
	}
	if _, err := legacyDB.Exec(`
		INSERT INTO users (username, password_hash, balanceDollars, serverSeed, transactionNumber)
		VALUES (?, ?, ?, ?, ?)
	`, "existing-user", "password-hash", 100, "server-seed", 0); err != nil {
		t.Fatalf("seed legacy user: %v", err)
	}
	if _, err := legacyDB.Exec(`
		INSERT INTO games (userId, betAmount, gridSize, mines, demo, openedCells, status, serverSeed, clientSeed, nonce, startedAt)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, CURRENT_TIMESTAMP)
	`, 1, 10, 25, 3, false, "[]", "inProcess", "server-seed", "client-seed", 0); err != nil {
		t.Fatalf("seed legacy game: %v", err)
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

	assertExistingMoneyInCents(t, db)

	dbAgain, err := OpenSQLite(path)
	if err != nil {
		t.Fatalf("reopen migrated database: %v", err)
	}
	t.Cleanup(func() {
		_ = dbAgain.Close()
	})
	assertExistingMoneyInCents(t, dbAgain)
}

func assertExistingMoneyInCents(t *testing.T, db *sql.DB) {
	t.Helper()

	var balance, betAmount int64
	if err := db.QueryRow("SELECT balanceDollars FROM users WHERE username = ?", "existing-user").Scan(&balance); err != nil {
		t.Fatalf("read migrated balance: %v", err)
	}
	if err := db.QueryRow("SELECT betAmount FROM games LIMIT 1").Scan(&betAmount); err != nil {
		t.Fatalf("read migrated bet amount: %v", err)
	}
	if balance != 10000 || betAmount != 1000 {
		t.Fatalf("migrated money = balance:%d bet:%d", balance, betAmount)
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
