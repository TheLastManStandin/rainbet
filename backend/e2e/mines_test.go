package e2e_test

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"rainbet/internal/application"
	"rainbet/internal/delivery/httpapi"
	"rainbet/internal/domain/mines"
	"rainbet/internal/infrastructure/fairness"
	"rainbet/internal/infrastructure/password"
	"rainbet/internal/infrastructure/sqlite"
)

const (
	testUsername = "user"
	testPassword = "user"
)

type testBackend struct {
	db     *sql.DB
	server *httptest.Server
}

type createGameResponse struct {
	ID     int64  `json:"id"`
	Status string `json:"status"`
}

type moveResponse struct {
	Status      string `json:"status"`
	Result      string `json:"result"`
	OpenedCells []int  `json:"openedCells"`
	Multiplier  string `json:"multiplier"`
}

type cashoutResponse struct {
	Status     string `json:"status"`
	Payout     string `json:"payout"`
	Multiplier string `json:"multiplier"`
}

func TestMinesCashoutEndToEnd(t *testing.T) {
	backend := newTestBackend(t)
	createdGame := createGame(t, backend, `{
		"betAmount":10.50,
		"gridSize":25,
		"mines":3,
		"demo":false,
		"clientSeed":"e2e-cashout"
	}`)

	safeCell := safeCell(t, backend.db, createdGame.ID)
	status, body := post(t, backend, fmt.Sprintf("/api/mines/bets/%d/moves", createdGame.ID), fmt.Sprintf(`{"cellIndex":%d}`, safeCell))
	if status != http.StatusOK {
		t.Fatalf("move status = %d, want %d: %s", status, http.StatusOK, body)
	}

	var move moveResponse
	decodeJSON(t, body, &move)
	if move.Status != string(mines.StatusInProcess) || move.Result != "diamond" || len(move.OpenedCells) != 1 {
		t.Fatalf("move response = %+v", move)
	}

	status, body = post(t, backend, fmt.Sprintf("/api/mines/bets/%d/cashout", createdGame.ID), "")
	if status != http.StatusOK {
		t.Fatalf("cashout status = %d, want %d: %s", status, http.StatusOK, body)
	}

	var cashout cashoutResponse
	decodeJSON(t, body, &cashout)
	if cashout.Status != string(mines.StatusCachedOut) || cashout.Payout != "11.44" || cashout.Multiplier != "1.09" {
		t.Fatalf("cashout response = %+v", cashout)
	}
	if balance := userBalance(t, backend.db); balance != 10094 {
		t.Fatalf("balance = %d, want 10094", balance)
	}

	status, _ = post(t, backend, fmt.Sprintf("/api/mines/bets/%d/cashout", createdGame.ID), "")
	if status != http.StatusConflict {
		t.Fatalf("second cashout status = %d, want %d", status, http.StatusConflict)
	}
}

func TestMinesBombEndToEnd(t *testing.T) {
	backend := newTestBackend(t)
	createdGame := createGame(t, backend, `{
		"betAmount":10.00,
		"gridSize":25,
		"mines":3,
		"demo":false,
		"clientSeed":"e2e-bomb"
	}`)

	bombCell := mineIndexes(t, backend.db, createdGame.ID)[0]
	status, body := post(t, backend, fmt.Sprintf("/api/mines/bets/%d/moves", createdGame.ID), fmt.Sprintf(`{"cellIndex":%d}`, bombCell))
	if status != http.StatusOK {
		t.Fatalf("move status = %d, want %d: %s", status, http.StatusOK, body)
	}

	var move moveResponse
	decodeJSON(t, body, &move)
	if move.Status != string(mines.StatusFailed) || move.Result != "bomb" {
		t.Fatalf("move response = %+v", move)
	}
	if balance := userBalance(t, backend.db); balance != 9000 {
		t.Fatalf("balance = %d, want 9000", balance)
	}

	status, _ = post(t, backend, fmt.Sprintf("/api/mines/bets/%d/cashout", createdGame.ID), "")
	if status != http.StatusConflict {
		t.Fatalf("cashout after bomb status = %d, want %d", status, http.StatusConflict)
	}
}

func TestMinesDemoEndToEnd(t *testing.T) {
	backend := newTestBackend(t)
	createdGame := createGame(t, backend, `{
		"betAmount":0,
		"gridSize":25,
		"mines":3,
		"demo":true,
		"clientSeed":"e2e-demo"
	}`)

	safeCell := safeCell(t, backend.db, createdGame.ID)
	status, body := post(t, backend, fmt.Sprintf("/api/mines/bets/%d/moves", createdGame.ID), fmt.Sprintf(`{"cellIndex":%d}`, safeCell))
	if status != http.StatusOK {
		t.Fatalf("move status = %d, want %d: %s", status, http.StatusOK, body)
	}

	status, body = post(t, backend, fmt.Sprintf("/api/mines/bets/%d/cashout", createdGame.ID), "")
	if status != http.StatusOK {
		t.Fatalf("cashout status = %d, want %d: %s", status, http.StatusOK, body)
	}

	var cashout cashoutResponse
	decodeJSON(t, body, &cashout)
	if cashout.Status != string(mines.StatusCachedOut) || cashout.Payout != "0.00" {
		t.Fatalf("cashout response = %+v", cashout)
	}
	if balance := userBalance(t, backend.db); balance != 10000 {
		t.Fatalf("balance = %d, want 10000", balance)
	}
}

func TestMinesRequiresBasicAuthEndToEnd(t *testing.T) {
	backend := newTestBackend(t)
	request, err := http.NewRequest(http.MethodPost, backend.server.URL+"/api/mines/bets", strings.NewReader(`{}`))
	if err != nil {
		t.Fatalf("create request: %v", err)
	}

	response, err := backend.server.Client().Do(request)
	if err != nil {
		t.Fatalf("send request: %v", err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", response.StatusCode, http.StatusUnauthorized)
	}
}

func newTestBackend(t *testing.T) *testBackend {
	t.Helper()

	db, err := sqlite.Open(filepath.Join(t.TempDir(), "rainbet.db"))
	if err != nil {
		t.Fatalf("open test database: %v", err)
	}
	if _, err := db.Exec(`
		UPDATE users
		SET balanceDollars = ?, serverSeed = ?, transactionNumber = 0
		WHERE username = ?
	`, 10000, "e2e-server-seed", testUsername); err != nil {
		_ = db.Close()
		t.Fatalf("configure test user: %v", err)
	}

	store := sqlite.NewStore(db)
	accounts := application.NewAccountService(store.Accounts(), password.Bcrypt{})
	games := application.NewMinesService(store, fairness.Generator{})
	server := httptest.NewServer(httpapi.New(accounts, games))
	t.Cleanup(func() {
		server.Close()
		_ = db.Close()
	})

	return &testBackend{db: db, server: server}
}

func createGame(t *testing.T, backend *testBackend, body string) createGameResponse {
	t.Helper()

	status, responseBody := post(t, backend, "/api/mines/bets", body)
	if status != http.StatusCreated {
		t.Fatalf("create game status = %d, want %d: %s", status, http.StatusCreated, responseBody)
	}

	var createdGame createGameResponse
	decodeJSON(t, responseBody, &createdGame)
	if createdGame.ID <= 0 || createdGame.Status != string(mines.StatusInProcess) {
		t.Fatalf("create game response = %+v", createdGame)
	}

	return createdGame
}

func post(t *testing.T, backend *testBackend, path, body string) (int, []byte) {
	t.Helper()

	request, err := http.NewRequest(http.MethodPost, backend.server.URL+path, strings.NewReader(body))
	if err != nil {
		t.Fatalf("create request: %v", err)
	}
	if body != "" {
		request.Header.Set("Content-Type", "application/json")
	}
	request.SetBasicAuth(testUsername, testPassword)

	response, err := backend.server.Client().Do(request)
	if err != nil {
		t.Fatalf("send request: %v", err)
	}
	defer response.Body.Close()

	responseBody, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatalf("read response: %v", err)
	}

	return response.StatusCode, responseBody
}

func mineIndexes(t *testing.T, db *sql.DB, gameID int64) []int {
	t.Helper()

	var (
		gridSize   int
		mines      int
		serverSeed string
		clientSeed string
		nonce      int64
	)
	if err := db.QueryRow(`
		SELECT gridSize, mines, serverSeed, clientSeed, nonce
		FROM games
		WHERE id = ?
	`, gameID).Scan(&gridSize, &mines, &serverSeed, &clientSeed, &nonce); err != nil {
		t.Fatalf("read game fairness data: %v", err)
	}

	indexes, err := fairness.DetermineMineIndexes(fairness.Options{
		Tiles:             gridSize,
		Mines:             mines,
		ServerSeed:        serverSeed,
		ClientSeed:        clientSeed,
		TransactionNumber: nonce,
	})
	if err != nil {
		t.Fatalf("determine mine indexes: %v", err)
	}

	return indexes
}

func safeCell(t *testing.T, db *sql.DB, gameID int64) int {
	t.Helper()

	mines := mineIndexes(t, db, gameID)
	for cell := 0; cell < 25; cell++ {
		if !contains(mines, cell) {
			return cell
		}
	}

	t.Fatal("safe cell not found")
	return 0
}

func contains(values []int, target int) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}

	return false
}

func userBalance(t *testing.T, db *sql.DB) int64 {
	t.Helper()

	var balance int64
	if err := db.QueryRow("SELECT balanceDollars FROM users WHERE username = ?", testUsername).Scan(&balance); err != nil {
		t.Fatalf("read user balance: %v", err)
	}

	return balance
}

func decodeJSON(t *testing.T, body []byte, target any) {
	t.Helper()

	if err := json.Unmarshal(body, target); err != nil {
		t.Fatalf("decode JSON response %q: %v", body, err)
	}
}
