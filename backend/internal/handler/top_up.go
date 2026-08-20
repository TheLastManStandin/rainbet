package handler

import (
	"encoding/json"
	"net/http"

	"rainbet/internal/response"
	"rainbet/internal/user"
)

type topUpBalanceRequest struct {
	Username string  `json:"username"`
	Amount   float64 `json:"amount"`
}

// TopUpBalance is intentionally unauthenticated and unrestricted for test funds.
func TopUpBalance(users *user.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var request topUpBalanceRequest
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			response.JSON(w, http.StatusBadRequest, map[string]string{
				"error": "request body must be valid JSON",
			})
			return
		}

		if err := users.AddBalance(r.Context(), request.Username, int64(request.Amount*100)); err != nil {
			response.JSON(w, http.StatusInternalServerError, map[string]string{
				"error": "could not update user balance",
			})
			return
		}

		response.JSON(w, http.StatusOK, map[string]bool{"ok": true})
	}
}
