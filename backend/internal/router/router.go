package router

import (
	"net/http"

	"rainbet/internal/handler"
	"rainbet/internal/middleware"
)

func New(username, password string) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/mines/bets", handler.CreateMinesBet)
	return middleware.BasicAuth(username, password)(mux)
}
