package httpapi

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"rainbet/internal/application"
	"rainbet/internal/domain/mines"
	"rainbet/internal/infrastructure/fairness"
	"rainbet/internal/infrastructure/password"
	"rainbet/internal/infrastructure/sqlite"
)

const (
	testUsername = "user"
	testPassword = "user"
)

type createdGameResponse struct {
	ID     int64        `json:"id"`
	Status mines.Status `json:"status"`
}

func TestCurrentUserAndPublicTopUp(t *testing.T) {
	handler, db := testHandler(t)
	assertBalance(t, handler, "100.00")

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/api/user/balance", bytes.NewBufferString(`{"username":"user","amount":100}`))
	handler.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusOK {
		t.Fatalf("top-up status = %d: %s", recorder.Code, recorder.Body.String())
	}
	assertBalance(t, handler, "200.00")
	if balance := userBalance(t, db); balance != 20_000 {
		t.Fatalf("stored balance = %d", balance)
	}
}

func TestCreateMoveAndCashOut(t *testing.T) {
	handler, db := testHandler(t)
	created := createGame(t, handler, `{"betAmount":10.50,"gridSize":25,"mines":3,"demo":false,"clientSeed":"route-test"}`)
	if created.Status != mines.StatusInProcess || userBalance(t, db) != 8_950 {
		t.Fatalf("created = %+v, balance = %d", created, userBalance(t, db))
	}

	safe := safeCell(t, db, created.ID)
	moveRecorder := httptest.NewRecorder()
	handler.ServeHTTP(moveRecorder, authenticatedRequest(
		http.MethodPost, fmt.Sprintf("/api/mines/bets/%d/moves", created.ID), fmt.Sprintf(`{"cellIndex":%d}`, safe),
	))
	if moveRecorder.Code != http.StatusOK {
		t.Fatalf("move status = %d: %s", moveRecorder.Code, moveRecorder.Body.String())
	}
	var move struct {
		Status      mines.Status `json:"status"`
		Result      string       `json:"result"`
		OpenedCells []int        `json:"openedCells"`
		Multiplier  string       `json:"multiplier"`
	}
	decode(t, moveRecorder, &move)
	if move.Status != mines.StatusInProcess || move.Result != "diamond" || move.Multiplier != "1.09" {
		t.Fatalf("move = %+v", move)
	}

	cashoutRecorder := httptest.NewRecorder()
	handler.ServeHTTP(cashoutRecorder, authenticatedRequest(
		http.MethodPost, fmt.Sprintf("/api/mines/bets/%d/cashout", created.ID), "",
	))
	if cashoutRecorder.Code != http.StatusOK {
		t.Fatalf("cashout status = %d: %s", cashoutRecorder.Code, cashoutRecorder.Body.String())
	}
	var cashout struct {
		Status     mines.Status `json:"status"`
		Payout     string       `json:"payout"`
		Multiplier string       `json:"multiplier"`
	}
	decode(t, cashoutRecorder, &cashout)
	if cashout.Status != mines.StatusCachedOut || cashout.Payout != "11.44" || cashout.Multiplier != "1.09" {
		t.Fatalf("cashout = %+v", cashout)
	}
	if userBalance(t, db) != 10_094 {
		t.Fatalf("balance after cashout = %d", userBalance(t, db))
	}
}

func TestBombAndInvalidRequests(t *testing.T) {
	handler, db := testHandler(t)
	created := createGame(t, handler, `{"betAmount":10,"gridSize":25,"mines":3,"demo":false,"clientSeed":"bomb-test"}`)
	bomb := mineIndexes(t, db, created.ID)[0]
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, authenticatedRequest(
		http.MethodPost, fmt.Sprintf("/api/mines/bets/%d/moves", created.ID), fmt.Sprintf(`{"cellIndex":%d}`, bomb),
	))
	if recorder.Code != http.StatusOK {
		t.Fatalf("bomb move status = %d: %s", recorder.Code, recorder.Body.String())
	}
	var move struct {
		Status mines.Status `json:"status"`
		Result string       `json:"result"`
	}
	decode(t, recorder, &move)
	if move.Status != mines.StatusFailed || move.Result != "bomb" {
		t.Fatalf("bomb move = %+v", move)
	}

	for _, body := range []string{
		`{"betAmount":0,"gridSize":25,"mines":3,"demo":false,"clientSeed":"test"}`,
		`{"betAmount":10.001,"gridSize":25,"mines":3,"demo":false,"clientSeed":"test"}`,
		`{"betAmount":10,"gridSize":24,"mines":3,"demo":false,"clientSeed":"test"}`,
	} {
		recorder := httptest.NewRecorder()
		handler.ServeHTTP(recorder, authenticatedRequest(http.MethodPost, "/api/mines/bets", body))
		if recorder.Code != http.StatusBadRequest {
			t.Fatalf("body %s: status = %d, response = %s", body, recorder.Code, recorder.Body.String())
		}
	}
}

func TestMinesRequiresAuthentication(t *testing.T) {
	handler, _ := testHandler(t)
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodPost, "/api/mines/bets", bytes.NewBufferString(`{}`)))
	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusUnauthorized)
	}
}

func testHandler(t *testing.T) (http.Handler, *sql.DB) {
	t.Helper()
	db, err := sqlite.Open(filepath.Join(t.TempDir(), "rainbet.db"))
	if err != nil {
		t.Fatalf("open test database: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	store := sqlite.NewStore(db)
	accounts := application.NewAccountService(store.Accounts(), password.Bcrypt{})
	games := application.NewMinesService(store, fairness.Generator{})
	return New(accounts, games), db
}

func createGame(t *testing.T, handler http.Handler, body string) createdGameResponse {
	t.Helper()
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, authenticatedRequest(http.MethodPost, "/api/mines/bets", body))
	if recorder.Code != http.StatusCreated {
		t.Fatalf("create status = %d: %s", recorder.Code, recorder.Body.String())
	}
	var created createdGameResponse
	decode(t, recorder, &created)
	if created.ID <= 0 {
		t.Fatalf("created game ID = %d", created.ID)
	}
	return created
}

func authenticatedRequest(method, path, body string) *http.Request {
	request := httptest.NewRequest(method, path, bytes.NewBufferString(body))
	request.SetBasicAuth(testUsername, testPassword)
	return request
}

func assertBalance(t *testing.T, handler http.Handler, want string) {
	t.Helper()
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, authenticatedRequest(http.MethodGet, "/api/user", ""))
	if recorder.Code != http.StatusOK {
		t.Fatalf("balance status = %d: %s", recorder.Code, recorder.Body.String())
	}
	var response struct {
		Balance string `json:"balance"`
	}
	decode(t, recorder, &response)
	if response.Balance != want {
		t.Fatalf("balance = %q, want %q", response.Balance, want)
	}
}

func userBalance(t *testing.T, db *sql.DB) int64 {
	t.Helper()
	var balance int64
	if err := db.QueryRow("SELECT balanceDollars FROM users WHERE username = ?", testUsername).Scan(&balance); err != nil {
		t.Fatalf("read balance: %v", err)
	}
	return balance
}

func mineIndexes(t *testing.T, db *sql.DB, gameID int64) []int {
	t.Helper()
	var options fairness.Options
	if err := db.QueryRow(`
		SELECT gridSize, mines, serverSeed, clientSeed, nonce FROM games WHERE id = ?
	`, gameID).Scan(&options.Tiles, &options.Mines, &options.ServerSeed, &options.ClientSeed, &options.TransactionNumber); err != nil {
		t.Fatalf("read fairness data: %v", err)
	}
	indexes, err := fairness.DetermineMineIndexes(options)
	if err != nil {
		t.Fatalf("determine mine indexes: %v", err)
	}
	return indexes
}

func safeCell(t *testing.T, db *sql.DB, gameID int64) int {
	t.Helper()
	minesByIndex := make(map[int]bool)
	for _, index := range mineIndexes(t, db, gameID) {
		minesByIndex[index] = true
	}
	for cell := 0; cell < 25; cell++ {
		if !minesByIndex[cell] {
			return cell
		}
	}
	t.Fatal("safe cell not found")
	return 0
}

func decode(t *testing.T, recorder *httptest.ResponseRecorder, target any) {
	t.Helper()
	if err := json.NewDecoder(recorder.Body).Decode(target); err != nil {
		t.Fatalf("decode response: %v", err)
	}
}
