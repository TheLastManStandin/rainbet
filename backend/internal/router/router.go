package router

import (
	"net/http"

	"rainbet/internal/handler"
	"rainbet/internal/middleware"
)

func New(authenticator middleware.Authenticator) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/mines/bets", handler.CreateMinesBet)
	return middleware.BasicAuth(authenticator)(mux)
}
