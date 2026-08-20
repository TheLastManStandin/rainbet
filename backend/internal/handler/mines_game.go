package handler

import (
	"errors"
	"net/http"
	"strconv"

	"rainbet/internal/game"
	"rainbet/internal/middleware"
	"rainbet/internal/response"
)

func gameIDFromRequest(r *http.Request) (int64, error) {
	gameID, err := strconv.ParseInt(r.PathValue("gameID"), 10, 64)
	if err != nil || gameID <= 0 {
		return 0, game.ErrGameNotFound
	}

	return gameID, nil
}

func authenticatedUserID(w http.ResponseWriter, r *http.Request) (int64, bool) {
	userID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		response.JSON(w, http.StatusInternalServerError, map[string]string{
			"error": "authenticated user is missing",
		})
	}

	return userID, ok
}

func writeGameError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, game.ErrGameNotFound):
		response.JSON(w, http.StatusNotFound, map[string]string{"error": "game not found"})
	case errors.Is(err, game.ErrGameFinished):
		response.JSON(w, http.StatusConflict, map[string]string{"error": "game is already finished"})
	case errors.Is(err, game.ErrCellAlreadyOpened):
		response.JSON(w, http.StatusConflict, map[string]string{"error": "cell is already opened"})
	case errors.Is(err, game.ErrNoOpenedCells):
		response.JSON(w, http.StatusConflict, map[string]string{"error": "open a diamond before cashout"})
	case errors.Is(err, game.ErrInvalidCell), errors.Is(err, game.ErrInvalidConfiguration):
		response.JSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
	case errors.Is(err, game.ErrInsufficientBalance):
		response.JSON(w, http.StatusConflict, map[string]string{"error": "insufficient balance"})
	default:
		response.JSON(w, http.StatusInternalServerError, map[string]string{"error": "game operation failed"})
	}
}
