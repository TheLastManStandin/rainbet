package httpapi

import (
	"net/http"

	"rainbet/internal/application"
)

func New(accounts *application.AccountService, games *application.MinesService) http.Handler {
	users := userHandler{accounts: accounts}
	mines := minesHandler{games: games}

	public := http.NewServeMux()
	public.HandleFunc("/api/user/balance", users.topUp)

	protected := http.NewServeMux()
	protected.HandleFunc("/api/user", users.current)
	protected.HandleFunc("/api/mines/bets", mines.create)
	protected.HandleFunc("/api/mines/bets/{gameID}/moves", mines.move)
	protected.HandleFunc("/api/mines/bets/{gameID}/cashout", mines.cashOut)

	public.Handle("/api/", basicAuth(accounts)(protected))
	return public
}
