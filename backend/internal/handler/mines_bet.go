package handler

import (
	"encoding/json"
	"net/http"

	"rainbet/internal/response"
)

type createMinesBetRequest struct {
	BetAmount json.Number `json:"betAmount"`
	GridSize  int         `json:"gridSize"`
	Mines     int         `json:"mines"`
	Demo      bool        `json:"demo"`
}

func CreateMinesBet(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		response.JSON(w, http.StatusMethodNotAllowed, map[string]string{
			"error": "method not allowed",
		})
		return
	}

	var request createMinesBetRequest
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&request); err != nil {
		response.JSON(w, http.StatusBadRequest, map[string]string{
			"error": "request body must be valid JSON",
		})
		return
	}

	response.JSON(w, http.StatusNotImplemented, map[string]string{
		"error": "mines bet creation is not implemented",
	})
}
