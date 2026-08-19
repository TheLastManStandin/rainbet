package router

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"
)

const (
	testUsername = "test-user"
	testPassword = "test-password"
)

func TestCreateMinesBetAcceptsRequest(t *testing.T) {
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(
		http.MethodPost,
		"/api/mines/bets",
		bytes.NewBufferString(`{"betAmount":10,"gridSize":5,"mines":3,"demo":true}`),
	)
	request.SetBasicAuth(testUsername, testPassword)

	New(testUsername, testPassword).ServeHTTP(recorder, request)

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

	New(testUsername, testPassword).ServeHTTP(recorder, request)

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

	New(testUsername, testPassword).ServeHTTP(recorder, request)

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

	New(testUsername, testPassword).ServeHTTP(recorder, request)

	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusUnauthorized)
	}
}
