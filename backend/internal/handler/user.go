package handler

import (
	"fmt"
	"net/http"

	"rainbet/internal/response"
	"rainbet/internal/user"
)

type currentUserResponse struct {
	Balance string `json:"balance"`
}

func CurrentUser(users *user.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.Header().Set("Allow", http.MethodGet)
			response.JSON(w, http.StatusMethodNotAllowed, map[string]string{
				"error": "method not allowed",
			})
			return
		}

		userID, ok := authenticatedUserID(w, r)
		if !ok {
			return
		}

		balance, err := users.Balance(r.Context(), userID)
		if err != nil {
			response.JSON(w, http.StatusInternalServerError, map[string]string{
				"error": "could not read user balance",
			})
			return
		}

		response.JSON(w, http.StatusOK, currentUserResponse{Balance: formatCents(balance)})
	}
}

func formatCents(cents int64) string {
	return fmt.Sprintf("%d.%02d", cents/100, cents%100)
}
