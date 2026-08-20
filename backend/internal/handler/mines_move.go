package handler

import (
	"encoding/json"
	"net/http"

	"rainbet/internal/game"
	"rainbet/internal/response"
)

type minesMoveRequest struct {
	CellIndex *int `json:"cellIndex"`
}

func MoveMinesBet(games *game.Store) http.HandlerFunc {
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

		var request minesMoveRequest
		decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20))
		decoder.DisallowUnknownFields()
		if err := decoder.Decode(&request); err != nil || request.CellIndex == nil {
			response.JSON(w, http.StatusBadRequest, map[string]string{
				"error": "cellIndex is required",
			})
			return
		}

		userID, ok := authenticatedUserID(w, r)
		if !ok {
			return
		}

		result, err := games.Move(r.Context(), game.MoveInput{
			UserID:    userID,
			GameID:    gameID,
			CellIndex: *request.CellIndex,
		})
		if err != nil {
			writeGameError(w, err)
			return
		}

		response.JSON(w, http.StatusOK, result)
	}
}
