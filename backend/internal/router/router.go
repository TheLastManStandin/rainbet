package router

import (
	"net/http"

	"rainbet/internal/game"
	"rainbet/internal/handler"
	"rainbet/internal/middleware"
)

func New(authenticator middleware.Authenticator, games *game.Store) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/mines/bets", handler.CreateMinesBet(games))
	mux.HandleFunc("/api/mines/bets/{gameID}/moves", handler.MoveMinesBet(games))
	mux.HandleFunc("/api/mines/bets/{gameID}/cashout", handler.CashOutMinesBet(games))
	return middleware.BasicAuth(authenticator)(mux)
}
