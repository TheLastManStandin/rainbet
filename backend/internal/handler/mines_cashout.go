package handler

import (
	"net/http"

	"rainbet/internal/game"
	"rainbet/internal/response"
)

func CashOutMinesBet(games *game.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.Header().Set("Allow", http.MethodPost)
			response.JSON(w, http.StatusMethodNotAllowed, map[string]string{
				"error": "method not allowed",
			})
			return
		}

		gameID, err := gameIDFromRequest(r)
		if err != nil {
			writeGameError(w, err)
			return
		}

		userID, ok := authenticatedUserID(w, r)
		if !ok {
			return
		}

		result, err := games.CashOut(r.Context(), game.CashOutInput{
			UserID: userID,
			GameID: gameID,
		})
		if err != nil {
			writeGameError(w, err)
			return
		}

		response.JSON(w, http.StatusOK, result)
	}
}
