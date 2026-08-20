package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"

	"rainbet/internal/game"
	"rainbet/internal/middleware"
	"rainbet/internal/response"
)

type createMinesBetRequest struct {
	BetAmount  json.Number `json:"betAmount"`
	GridSize   int         `json:"gridSize"`
	Mines      int         `json:"mines"`
	Demo       bool        `json:"demo"`
	ClientSeed string      `json:"clientSeed"`
}

func CreateMinesBet(games *game.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
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
		if request.ClientSeed == "" {
			response.JSON(w, http.StatusBadRequest, map[string]string{
				"error": "clientSeed is required",
			})
			return
		}
		betAmount, err := parseDollarsToCents(request.BetAmount)
		if err != nil {
			response.JSON(w, http.StatusBadRequest, map[string]string{
				"error": "betAmount must be a non-negative amount with at most two decimal places",
			})
			return
		}
		if (request.Demo && betAmount != 0) || (!request.Demo && betAmount == 0) {
			response.JSON(w, http.StatusBadRequest, map[string]string{
				"error": "demo games require betAmount 0 and real games require a positive betAmount",
			})
			return
		}

		userID, ok := middleware.UserIDFromContext(r.Context())
		if !ok {
			response.JSON(w, http.StatusInternalServerError, map[string]string{
				"error": "authenticated user is missing",
			})
			return
		}

		createdGame, err := games.Create(r.Context(), game.CreateInput{
			UserID:     userID,
			BetAmount:  betAmount,
			GridSize:   request.GridSize,
			Mines:      request.Mines,
			Demo:       request.Demo,
			ClientSeed: request.ClientSeed,
		})
		if err != nil {
			if errors.Is(err, game.ErrInsufficientBalance) {
				response.JSON(w, http.StatusConflict, map[string]string{
					"error": "insufficient balance",
				})
				return
			}
			if errors.Is(err, game.ErrInvalidConfiguration) {
				response.JSON(w, http.StatusBadRequest, map[string]string{
					"error": err.Error(),
				})
				return
			}
			response.JSON(w, http.StatusInternalServerError, map[string]string{
				"error": "could not create game",
			})
			return
		}

		response.JSON(w, http.StatusCreated, createdGame)
	}
}

func parseDollarsToCents(value json.Number) (int64, error) {
	raw := string(value)
	if raw == "" || strings.HasPrefix(raw, "-") {
		return 0, errors.New("invalid amount")
	}

	parts := strings.Split(raw, ".")
	if len(parts) > 2 || parts[0] == "" {
		return 0, errors.New("invalid amount")
	}
	dollars, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil || dollars < 0 {
		return 0, errors.New("invalid amount")
	}

	cents := int64(0)
	if len(parts) == 2 {
		if len(parts[1]) > 2 {
			return 0, errors.New("amount has more than two decimal places")
		}
		if parts[1] != "" {
			cents, err = strconv.ParseInt(parts[1], 10, 64)
			if err != nil || cents < 0 {
				return 0, errors.New("invalid amount")
			}
			if len(parts[1]) == 1 {
				cents *= 10
			}
		}
	}

	const maxInt64 = int64(1<<63 - 1)
	if dollars > (maxInt64-cents)/100 {
		return 0, errors.New("amount is too large")
	}

	return dollars*100 + cents, nil
}
