package httpapi

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"rainbet/internal/application"
	"rainbet/internal/domain/account"
	"rainbet/internal/domain/mines"
)

type minesHandler struct{ games *application.MinesService }

type createMinesBetRequest struct {
	BetAmount  json.Number `json:"betAmount"`
	GridSize   int         `json:"gridSize"`
	Mines      int         `json:"mines"`
	Demo       bool        `json:"demo"`
	ClientSeed string      `json:"clientSeed"`
}

func (handler minesHandler) create(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}
	var request createMinesBetRequest
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&request); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "request body must be valid JSON"})
		return
	}
	if request.ClientSeed == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "clientSeed is required"})
		return
	}
	betCents, err := parseDollarsToCents(request.BetAmount)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "betAmount must be a non-negative amount with at most two decimal places"})
		return
	}
	if (request.Demo && betCents != 0) || (!request.Demo && betCents == 0) {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "demo games require betAmount 0 and real games require a positive betAmount"})
		return
	}
	userID, ok := authenticatedUserID(w, r)
	if !ok {
		return
	}
	created, err := handler.games.Create(r.Context(), application.CreateGameCommand{
		UserID: userID, BetCents: betCents, GridSize: request.GridSize,
		Mines: request.Mines, Demo: request.Demo, ClientSeed: request.ClientSeed,
	})
	if err != nil {
		writeGameError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, struct {
		ID     int64        `json:"id"`
		Status mines.Status `json:"status"`
	}{ID: created.ID, Status: created.Status})
}

type minesMoveRequest struct {
	CellIndex *int `json:"cellIndex"`
}

func (handler minesHandler) move(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
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
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "cellIndex is required"})
		return
	}
	userID, ok := authenticatedUserID(w, r)
	if !ok {
		return
	}
	result, err := handler.games.Move(r.Context(), application.MoveCommand{
		UserID: userID, GameID: gameID, CellIndex: *request.CellIndex,
	})
	if err != nil {
		writeGameError(w, err)
		return
	}
	response := struct {
		Status      mines.Status `json:"status"`
		Result      string       `json:"result"`
		OpenedCells []int        `json:"openedCells"`
		Multiplier  string       `json:"multiplier,omitempty"`
		MineIndexes []int        `json:"mineIndexes,omitempty"`
	}{Status: result.Status, Result: "diamond", OpenedCells: result.OpenedCells}
	if result.Bomb {
		response.Result = "bomb"
		response.MineIndexes = result.MineIndexes
	} else {
		response.Multiplier = formatHundredths(result.MultiplierHundredths)
	}
	writeJSON(w, http.StatusOK, response)
}

func (handler minesHandler) cashOut(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
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
	result, err := handler.games.CashOut(r.Context(), application.CashOutCommand{UserID: userID, GameID: gameID})
	if err != nil {
		writeGameError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, struct {
		Status      mines.Status `json:"status"`
		Payout      string       `json:"payout"`
		Multiplier  string       `json:"multiplier"`
		MineIndexes []int        `json:"mineIndexes,omitempty"`
	}{
		Status: result.Status, Payout: formatCents(result.PayoutCents),
		Multiplier: formatHundredths(result.MultiplierHundredths), MineIndexes: result.MineIndexes,
	})
}

func gameIDFromRequest(r *http.Request) (int64, error) {
	gameID, err := strconv.ParseInt(r.PathValue("gameID"), 10, 64)
	if err != nil || gameID <= 0 {
		return 0, application.ErrGameNotFound
	}
	return gameID, nil
}

func writeGameError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, application.ErrGameNotFound):
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "game not found"})
	case errors.Is(err, mines.ErrGameFinished):
		writeJSON(w, http.StatusConflict, map[string]string{"error": "game is already finished"})
	case errors.Is(err, mines.ErrCellAlreadyOpened):
		writeJSON(w, http.StatusConflict, map[string]string{"error": "cell is already opened"})
	case errors.Is(err, mines.ErrNoOpenedCells):
		writeJSON(w, http.StatusConflict, map[string]string{"error": "open a diamond before cashout"})
	case errors.Is(err, mines.ErrInvalidCell), errors.Is(err, mines.ErrInvalidConfiguration):
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
	case errors.Is(err, account.ErrInsufficientBalance):
		writeJSON(w, http.StatusConflict, map[string]string{"error": "insufficient balance"})
	default:
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "game operation failed"})
	}
}
