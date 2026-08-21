package httpapi

import (
	"encoding/json"
	"net/http"

	"rainbet/internal/application"
)

type userHandler struct{ accounts *application.AccountService }

func (handler userHandler) current(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}
	userID, ok := authenticatedUserID(w, r)
	if !ok {
		return
	}
	balance, err := handler.accounts.Balance(r.Context(), userID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "could not read user balance"})
		return
	}
	writeJSON(w, http.StatusOK, struct {
		Balance string `json:"balance"`
	}{Balance: formatCents(balance)})
}

type topUpBalanceRequest struct {
	Username string  `json:"username"`
	Amount   float64 `json:"amount"`
}

// topUp is intentionally unauthenticated and unrestricted for test funds.
func (handler userHandler) topUp(w http.ResponseWriter, r *http.Request) {
	var request topUpBalanceRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "request body must be valid JSON"})
		return
	}
	if err := handler.accounts.AddBalance(r.Context(), request.Username, int64(request.Amount*100)); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "could not update user balance"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}
