package router

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"rainbet/internal/database"
	"rainbet/internal/user"
)

const (
	testUsername = "user"
	testPassword = "user"
)

func TestCreateMinesBetAcceptsRequest(t *testing.T) {
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(
		http.MethodPost,
		"/api/mines/bets",
		bytes.NewBufferString(`{"betAmount":10,"gridSize":5,"mines":3,"demo":true,"clientSeed":"mine-test"}`),
	)
	request.SetBasicAuth(testUsername, testPassword)

	testHandler(t).ServeHTTP(recorder, request)

	if recorder.Code != http.StatusNotImplemented {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusNotImplemented)
	}

	if contentType := recorder.Header().Get("Content-Type"); contentType != "application/json" {
		t.Fatalf("Content-Type = %q, want %q", contentType, "application/json")
	}
}

func TestCreateMinesBetRejectsOtherMethods(t *testing.T) {
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/api/mines/bets", nil)
	request.SetBasicAuth(testUsername, testPassword)

	testHandler(t).ServeHTTP(recorder, request)

	if recorder.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusMethodNotAllowed)
	}

	if allow := recorder.Header().Get("Allow"); allow != http.MethodPost {
		t.Fatalf("Allow = %q, want %q", allow, http.MethodPost)
	}
}

func TestCreateMinesBetRequiresAuthentication(t *testing.T) {
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/api/mines/bets", nil)

	testHandler(t).ServeHTTP(recorder, request)

	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusUnauthorized)
	}

	if challenge := recorder.Header().Get("WWW-Authenticate"); challenge == "" {
		t.Fatal("WWW-Authenticate header is missing")
	}
}

func TestCreateMinesBetRejectsInvalidAuthentication(t *testing.T) {
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/api/mines/bets", nil)
	request.SetBasicAuth(testUsername, "wrong-password")

	testHandler(t).ServeHTTP(recorder, request)

	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusUnauthorized)
	}
}

func testHandler(t *testing.T) http.Handler {
	t.Helper()

	db, err := database.OpenSQLite(filepath.Join(t.TempDir(), "rainbet.db"))
	if err != nil {
		t.Fatalf("open test database: %v", err)
	}
	t.Cleanup(func() {
		_ = db.Close()
	})

	return New(user.NewStore(db))
}
