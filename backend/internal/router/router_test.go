package router

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"rainbet/internal/database"
	"rainbet/internal/game"
	"rainbet/internal/provablyfair"
	"rainbet/internal/user"
)

const (
	testUsername = "user"
	testPassword = "user"
)

func TestCreateMinesBetCreatesRealGameAndDebitsBalance(t *testing.T) {
	handler, db := testHandler(t)
	createdGame := createGame(t, handler, `{"betAmount":10.50,"gridSize":25,"mines":3,"demo":false,"clientSeed":"mine-test"}`)

	if createdGame.Status != game.StatusInProcess {
		t.Fatalf("game status = %q, want %q", createdGame.Status, game.StatusInProcess)
	}

	var (
		userID            int64
		balance           int64
		userServerSeed    string
		transactionNumber int64
		betAmount         int64
		gridSize          int
		mines             int
		demo              bool
		openedCells       string
		gameStatus        string
		gameServerSeed    string
		clientSeed        string
		nonce             int64
	)
	if err := db.QueryRow(
		"SELECT id, balanceDollars, serverSeed, transactionNumber FROM users WHERE username = ?",
		testUsername,
	).Scan(&userID, &balance, &userServerSeed, &transactionNumber); err != nil {
		t.Fatalf("read user: %v", err)
	}
	if err := db.QueryRow(`
		SELECT betAmount, gridSize, mines, demo, openedCells, status, serverSeed, clientSeed, nonce
		FROM games
		WHERE id = ? AND userId = ?
	`, createdGame.ID, userID).Scan(
		&betAmount,
		&gridSize,
		&mines,
		&demo,
		&openedCells,
		&gameStatus,
		&gameServerSeed,
		&clientSeed,
		&nonce,
	); err != nil {
		t.Fatalf("read game: %v", err)
	}

	if balance != 8950 {
		t.Fatalf("balance = %d, want 8950", balance)
	}
	if transactionNumber != 1 {
		t.Fatalf("transaction number = %d, want 1", transactionNumber)
	}
	if betAmount != 1050 || gridSize != 25 || mines != 3 || demo {
		t.Fatalf("stored game config = bet:%d grid:%d mines:%d demo:%t", betAmount, gridSize, mines, demo)
	}
	if openedCells != "[]" {
		t.Fatalf("opened cells = %q, want %q", openedCells, "[]")
	}
	if gameStatus != game.StatusInProcess {
		t.Fatalf("stored game status = %q, want %q", gameStatus, game.StatusInProcess)
	}
	if gameServerSeed != userServerSeed {
		t.Fatal("game server seed does not match the user's server seed")
	}
	if clientSeed != "mine-test" || nonce != 0 {
		t.Fatalf("stored fairness data = clientSeed:%q nonce:%d", clientSeed, nonce)
	}
}

func TestCreateMinesBetAllowsDemoWithZeroBet(t *testing.T) {
	handler, db := testHandler(t)
	createdGame := createGame(t, handler, `{"betAmount":0,"gridSize":25,"mines":3,"demo":true,"clientSeed":"mine-test"}`)

	if balance := userBalance(t, db); balance != 10000 {
		t.Fatalf("balance = %d, want 10000", balance)
	}

	var (
		betAmount int64
		demo      bool
	)
	if err := db.QueryRow("SELECT betAmount, demo FROM games WHERE id = ?", createdGame.ID).Scan(&betAmount, &demo); err != nil {
		t.Fatalf("read demo game: %v", err)
	}
	if betAmount != 0 || !demo {
		t.Fatalf("demo game config = bet:%d demo:%t", betAmount, demo)
	}
}

func TestCreateMinesBetRejectsInvalidBetForMode(t *testing.T) {
	handler, db := testHandler(t)

	for _, body := range []string{
		`{"betAmount":0,"gridSize":25,"mines":3,"demo":false,"clientSeed":"mine-test"}`,
		`{"betAmount":10,"gridSize":25,"mines":3,"demo":true,"clientSeed":"mine-test"}`,
		`{"betAmount":10.001,"gridSize":25,"mines":3,"demo":false,"clientSeed":"mine-test"}`,
	} {
		recorder := httptest.NewRecorder()
		handler.ServeHTTP(recorder, authenticatedRequest(http.MethodPost, "/api/mines/bets", body))
		if recorder.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want %d", recorder.Code, http.StatusBadRequest)
		}
	}

	var gameCount int
	if err := db.QueryRow("SELECT COUNT(*) FROM games").Scan(&gameCount); err != nil {
		t.Fatalf("count games: %v", err)
	}
	if gameCount != 0 {
		t.Fatalf("game count = %d, want 0", gameCount)
	}
}

func TestCreateMinesBetRejectsUnsupportedGridSize(t *testing.T) {
	handler, db := testHandler(t)

	for _, gridSize := range []int{24, 26, 35, 37, 48, 50, 63, 65} {
		recorder := httptest.NewRecorder()
		handler.ServeHTTP(
			recorder,
			authenticatedRequest(
				http.MethodPost,
				"/api/mines/bets",
				fmt.Sprintf(`{"betAmount":10,"gridSize":%d,"mines":3,"demo":false,"clientSeed":"mine-test"}`, gridSize),
			),
		)
		if recorder.Code != http.StatusBadRequest {
			t.Fatalf("grid size %d: status = %d, want %d", gridSize, recorder.Code, http.StatusBadRequest)
		}
	}

	if gameCount(t, db) != 0 {
		t.Fatalf("game count = %d, want 0", gameCount(t, db))
	}
}

func TestMinesMoveAndCashOut(t *testing.T) {
	handler, db := testHandler(t)
	createdGame := createGame(t, handler, `{"betAmount":10,"gridSize":25,"mines":3,"demo":false,"clientSeed":"mine-test"}`)
	safeCell := safeCell(t, db, createdGame.ID)

	moveRecorder := httptest.NewRecorder()
	handler.ServeHTTP(
		moveRecorder,
		authenticatedRequest(
			http.MethodPost,
			fmt.Sprintf("/api/mines/bets/%d/moves", createdGame.ID),
			fmt.Sprintf(`{"cellIndex":%d}`, safeCell),
		),
	)
	if moveRecorder.Code != http.StatusOK {
		t.Fatalf("move status = %d, want %d", moveRecorder.Code, http.StatusOK)
	}

	var move game.MoveResult
	if err := json.NewDecoder(moveRecorder.Body).Decode(&move); err != nil {
		t.Fatalf("decode move response: %v", err)
	}
	if move.Status != game.StatusInProcess || move.Result != "diamond" {
		t.Fatalf("move result = status:%q result:%q", move.Status, move.Result)
	}
	if len(move.OpenedCells) != 1 || move.OpenedCells[0] != safeCell {
		t.Fatalf("opened cells = %v, want [%d]", move.OpenedCells, safeCell)
	}
	if move.Multiplier != "1.09090909" {
		t.Fatalf("multiplier = %q, want %q", move.Multiplier, "1.09090909")
	}

	cashoutRecorder := httptest.NewRecorder()
	handler.ServeHTTP(
		cashoutRecorder,
		authenticatedRequest(http.MethodPost, fmt.Sprintf("/api/mines/bets/%d/cashout", createdGame.ID), ""),
	)
	if cashoutRecorder.Code != http.StatusOK {
		t.Fatalf("cashout status = %d, want %d", cashoutRecorder.Code, http.StatusOK)
	}

	var cashout game.CashOutResult
	if err := json.NewDecoder(cashoutRecorder.Body).Decode(&cashout); err != nil {
		t.Fatalf("decode cashout response: %v", err)
	}
	if cashout.Status != game.StatusCachedOut || cashout.Payout != "10.90" || cashout.Multiplier != "1.09090909" {
		t.Fatalf("cashout = %+v", cashout)
	}
	if balance := userBalance(t, db); balance != 10090 {
		t.Fatalf("balance after cashout = %d, want 10090", balance)
	}

	secondCashoutRecorder := httptest.NewRecorder()
	handler.ServeHTTP(
		secondCashoutRecorder,
		authenticatedRequest(http.MethodPost, fmt.Sprintf("/api/mines/bets/%d/cashout", createdGame.ID), ""),
	)
	if secondCashoutRecorder.Code != http.StatusConflict {
		t.Fatalf("second cashout status = %d, want %d", secondCashoutRecorder.Code, http.StatusConflict)
	}

	moveAfterCashoutRecorder := httptest.NewRecorder()
	handler.ServeHTTP(
		moveAfterCashoutRecorder,
		authenticatedRequest(
			http.MethodPost,
			fmt.Sprintf("/api/mines/bets/%d/moves", createdGame.ID),
			fmt.Sprintf(`{"cellIndex":%d}`, safeCell),
		),
	)
	if moveAfterCashoutRecorder.Code != http.StatusConflict {
		t.Fatalf("move after cashout status = %d, want %d", moveAfterCashoutRecorder.Code, http.StatusConflict)
	}
}

func TestMinesBombFailsGameAndPreventsCashOut(t *testing.T) {
	handler, db := testHandler(t)
	createdGame := createGame(t, handler, `{"betAmount":10,"gridSize":25,"mines":3,"demo":false,"clientSeed":"mine-test"}`)
	bombCell := mineIndexes(t, db, createdGame.ID)[0]

	moveRecorder := httptest.NewRecorder()
	handler.ServeHTTP(
		moveRecorder,
		authenticatedRequest(
			http.MethodPost,
			fmt.Sprintf("/api/mines/bets/%d/moves", createdGame.ID),
			fmt.Sprintf(`{"cellIndex":%d}`, bombCell),
		),
	)
	if moveRecorder.Code != http.StatusOK {
		t.Fatalf("move status = %d, want %d", moveRecorder.Code, http.StatusOK)
	}

	var move game.MoveResult
	if err := json.NewDecoder(moveRecorder.Body).Decode(&move); err != nil {
		t.Fatalf("decode move response: %v", err)
	}
	if move.Status != game.StatusFailed || move.Result != "bomb" {
		t.Fatalf("move result = status:%q result:%q", move.Status, move.Result)
	}
	if balance := userBalance(t, db); balance != 9000 {
		t.Fatalf("balance after bomb = %d, want 9000", balance)
	}

	cashoutRecorder := httptest.NewRecorder()
	handler.ServeHTTP(
		cashoutRecorder,
		authenticatedRequest(http.MethodPost, fmt.Sprintf("/api/mines/bets/%d/cashout", createdGame.ID), ""),
	)
	if cashoutRecorder.Code != http.StatusConflict {
		t.Fatalf("cashout status = %d, want %d", cashoutRecorder.Code, http.StatusConflict)
	}
}

func TestMinesCashOutRequiresDiamond(t *testing.T) {
	handler, db := testHandler(t)
	createdGame := createGame(t, handler, `{"betAmount":10,"gridSize":25,"mines":3,"demo":false,"clientSeed":"mine-test"}`)

	cashoutRecorder := httptest.NewRecorder()
	handler.ServeHTTP(
		cashoutRecorder,
		authenticatedRequest(http.MethodPost, fmt.Sprintf("/api/mines/bets/%d/cashout", createdGame.ID), ""),
	)
	if cashoutRecorder.Code != http.StatusConflict {
		t.Fatalf("cashout status = %d, want %d", cashoutRecorder.Code, http.StatusConflict)
	}
	if balance := userBalance(t, db); balance != 9000 {
		t.Fatalf("balance = %d, want 9000", balance)
	}

	var status string
	if err := db.QueryRow("SELECT status FROM games WHERE id = ?", createdGame.ID).Scan(&status); err != nil {
		t.Fatalf("read game status: %v", err)
	}
	if status != game.StatusInProcess {
		t.Fatalf("game status = %q, want %q", status, game.StatusInProcess)
	}
}

func TestDemoMinesCashOutDoesNotChangeBalance(t *testing.T) {
	handler, db := testHandler(t)
	createdGame := createGame(t, handler, `{"betAmount":0,"gridSize":25,"mines":3,"demo":true,"clientSeed":"mine-test"}`)
	safeCell := safeCell(t, db, createdGame.ID)

	moveRecorder := httptest.NewRecorder()
	handler.ServeHTTP(
		moveRecorder,
		authenticatedRequest(
			http.MethodPost,
			fmt.Sprintf("/api/mines/bets/%d/moves", createdGame.ID),
			fmt.Sprintf(`{"cellIndex":%d}`, safeCell),
		),
	)
	if moveRecorder.Code != http.StatusOK {
		t.Fatalf("move status = %d, want %d", moveRecorder.Code, http.StatusOK)
	}

	cashoutRecorder := httptest.NewRecorder()
	handler.ServeHTTP(
		cashoutRecorder,
		authenticatedRequest(http.MethodPost, fmt.Sprintf("/api/mines/bets/%d/cashout", createdGame.ID), ""),
	)
	if cashoutRecorder.Code != http.StatusOK {
		t.Fatalf("cashout status = %d, want %d", cashoutRecorder.Code, http.StatusOK)
	}

	var cashout game.CashOutResult
	if err := json.NewDecoder(cashoutRecorder.Body).Decode(&cashout); err != nil {
		t.Fatalf("decode cashout response: %v", err)
	}
	if cashout.Payout != "0.00" {
		t.Fatalf("demo payout = %q, want %q", cashout.Payout, "0.00")
	}
	if balance := userBalance(t, db); balance != 10000 {
		t.Fatalf("demo balance = %d, want 10000", balance)
	}
}

func TestCreateMinesBetRequiresAuthentication(t *testing.T) {
	handler, _ := testHandler(t)
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/api/mines/bets", bytes.NewBufferString(`{}`))
	handler.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusUnauthorized)
	}
}

func TestCreateMinesBetRequiresClientSeed(t *testing.T) {
	handler, db := testHandler(t)
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(
		recorder,
		authenticatedRequest(http.MethodPost, "/api/mines/bets", `{"betAmount":10,"gridSize":25,"mines":3,"demo":false}`),
	)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusBadRequest)
	}
	if gameCount(t, db) != 0 {
		t.Fatalf("game count = %d, want 0", gameCount(t, db))
	}
}

func createGame(t *testing.T, handler http.Handler, body string) game.Game {
	t.Helper()

	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, authenticatedRequest(http.MethodPost, "/api/mines/bets", body))
	if recorder.Code != http.StatusCreated {
		t.Fatalf("create game status = %d, want %d: %s", recorder.Code, http.StatusCreated, recorder.Body.String())
	}

	var createdGame game.Game
	if err := json.NewDecoder(recorder.Body).Decode(&createdGame); err != nil {
		t.Fatalf("decode create game response: %v", err)
	}
	if createdGame.ID <= 0 {
		t.Fatalf("game ID = %d, want a positive ID", createdGame.ID)
	}

	return createdGame
}

func authenticatedRequest(method, path, body string) *http.Request {
	request := httptest.NewRequest(method, path, bytes.NewBufferString(body))
	request.SetBasicAuth(testUsername, testPassword)
	return request
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
	if err := db.QueryRow(
		"SELECT gridSize, mines, serverSeed, clientSeed, nonce FROM games WHERE id = ?",
		gameID,
	).Scan(&gridSize, &mines, &serverSeed, &clientSeed, &nonce); err != nil {
		t.Fatalf("read game fairness data: %v", err)
	}

	indexes, err := provablyfair.DetermineMineIndexes(provablyfair.MinesOptions{
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

func gameCount(t *testing.T, db *sql.DB) int {
	t.Helper()

	var count int
	if err := db.QueryRow("SELECT COUNT(*) FROM games").Scan(&count); err != nil {
		t.Fatalf("count games: %v", err)
	}

	return count
}

func testHandler(t *testing.T) (http.Handler, *sql.DB) {
	t.Helper()

	db, err := database.OpenSQLite(filepath.Join(t.TempDir(), "rainbet.db"))
	if err != nil {
		t.Fatalf("open test database: %v", err)
	}
	t.Cleanup(func() {
		_ = db.Close()
	})

	return New(user.NewStore(db), game.NewStore(db)), db
}
