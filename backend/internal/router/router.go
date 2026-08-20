package router

import (
	"net/http"

	"rainbet/internal/game"
	"rainbet/internal/handler"
	"rainbet/internal/middleware"
	"rainbet/internal/user"
)

func New(users *user.Store, games *game.Store) http.Handler {
	public := http.NewServeMux()
	public.HandleFunc("/api/user/balance", handler.TopUpBalance(users))

	protected := http.NewServeMux()
	protected.HandleFunc("/api/user", handler.CurrentUser(users))
	protected.HandleFunc("/api/mines/bets", handler.CreateMinesBet(games))
	protected.HandleFunc("/api/mines/bets/{gameID}/moves", handler.MoveMinesBet(games))
	protected.HandleFunc("/api/mines/bets/{gameID}/cashout", handler.CashOutMinesBet(games))

	public.Handle("/api/", middleware.BasicAuth(users)(protected))
	return public
}
