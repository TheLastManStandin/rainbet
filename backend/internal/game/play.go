package game

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"

	"rainbet/internal/provablyfair"
)

type MoveInput struct {
	UserID    int64
	GameID    int64
	CellIndex int
}

type MoveResult struct {
	Status      string `json:"status"`
	Result      string `json:"result"`
	OpenedCells []int  `json:"openedCells"`
	Multiplier  string `json:"multiplier,omitempty"`
}

type CashOutInput struct {
	UserID int64
	GameID int64
}

type CashOutResult struct {
	Status     string `json:"status"`
	Payout     string `json:"payout"`
	Multiplier string `json:"multiplier"`
}

type storedGame struct {
	ID          int64
	UserID      int64
	BetAmount   int64
	GridSize    int
	Mines       int
	Demo        bool
	OpenedCells []int
	Status      string
	ServerSeed  string
	ClientSeed  string
	Nonce       int64
}

func (s *Store) Move(ctx context.Context, input MoveInput) (MoveResult, error) {
	if input.UserID <= 0 || input.GameID <= 0 {
		return MoveResult{}, ErrGameNotFound
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return MoveResult{}, fmt.Errorf("begin move transaction: %w", err)
	}
	defer tx.Rollback()

	game, err := loadGame(ctx, tx, input.GameID, input.UserID)
	if err != nil {
		return MoveResult{}, err
	}
	if game.Status != StatusInProcess {
		return MoveResult{}, ErrGameFinished
	}
	if input.CellIndex < 0 || input.CellIndex >= game.GridSize {
		return MoveResult{}, ErrInvalidCell
	}
	if contains(game.OpenedCells, input.CellIndex) {
		return MoveResult{}, ErrCellAlreadyOpened
	}

	mineIndexes, err := provablyfair.DetermineMineIndexes(provablyfair.MinesOptions{
		Tiles:             game.GridSize,
		Mines:             game.Mines,
		ClientSeed:        game.ClientSeed,
		ServerSeed:        game.ServerSeed,
		TransactionNumber: game.Nonce,
	})
	if err != nil {
		return MoveResult{}, fmt.Errorf("determine mine indexes: %w", err)
	}

	openedCells := append(game.OpenedCells, input.CellIndex)
	openedCellsJSON, err := json.Marshal(openedCells)
	if err != nil {
		return MoveResult{}, fmt.Errorf("encode opened cells: %w", err)
	}

	isMine := contains(mineIndexes, input.CellIndex)
	status := StatusInProcess
	var multiplier *big.Rat
	if isMine {
		status = StatusFailed
	} else {
		multiplier, err = multiplierFor(game.GridSize, game.Mines, len(openedCells))
		if err != nil {
			return MoveResult{}, err
		}
	}
	result, err := tx.ExecContext(
		ctx,
		`UPDATE games
		 SET openedCells = ?, status = ?
		 WHERE id = ? AND userId = ? AND status = ?`,
		string(openedCellsJSON),
		status,
		game.ID,
		game.UserID,
		StatusInProcess,
	)
	if err != nil {
		return MoveResult{}, fmt.Errorf("save game move: %w", err)
	}
	updated, err := result.RowsAffected()
	if err != nil {
		return MoveResult{}, fmt.Errorf("read moved game count: %w", err)
	}
	if updated != 1 {
		return MoveResult{}, ErrGameFinished
	}

	if err := tx.Commit(); err != nil {
		return MoveResult{}, fmt.Errorf("commit move transaction: %w", err)
	}

	if isMine {
		return MoveResult{
			Status:      StatusFailed,
			Result:      "bomb",
			OpenedCells: openedCells,
		}, nil
	}

	return MoveResult{
		Status:      StatusInProcess,
		Result:      "diamond",
		OpenedCells: openedCells,
		Multiplier:  multiplier.FloatString(8),
	}, nil
}

func (s *Store) CashOut(ctx context.Context, input CashOutInput) (CashOutResult, error) {
	if input.UserID <= 0 || input.GameID <= 0 {
		return CashOutResult{}, ErrGameNotFound
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return CashOutResult{}, fmt.Errorf("begin cashout transaction: %w", err)
	}
	defer tx.Rollback()

	game, err := loadGame(ctx, tx, input.GameID, input.UserID)
	if err != nil {
		return CashOutResult{}, err
	}
	if game.Status != StatusInProcess {
		return CashOutResult{}, ErrGameFinished
	}
	if len(game.OpenedCells) == 0 {
		return CashOutResult{}, ErrNoOpenedCells
	}

	multiplier, err := multiplierFor(game.GridSize, game.Mines, len(game.OpenedCells))
	if err != nil {
		return CashOutResult{}, err
	}
	payout, err := payoutFor(game.BetAmount, multiplier)
	if err != nil {
		return CashOutResult{}, err
	}

	result, err := tx.ExecContext(
		ctx,
		"UPDATE games SET status = ? WHERE id = ? AND userId = ? AND status = ?",
		StatusCachedOut,
		game.ID,
		game.UserID,
		StatusInProcess,
	)
	if err != nil {
		return CashOutResult{}, fmt.Errorf("finish game: %w", err)
	}
	updated, err := result.RowsAffected()
	if err != nil {
		return CashOutResult{}, fmt.Errorf("read finished game count: %w", err)
	}
	if updated != 1 {
		return CashOutResult{}, ErrGameFinished
	}

	if !game.Demo && payout > 0 {
		result, err = tx.ExecContext(
			ctx,
			"UPDATE users SET balanceDollars = balanceDollars + ? WHERE id = ?",
			payout,
			game.UserID,
		)
		if err != nil {
			return CashOutResult{}, fmt.Errorf("credit user balance: %w", err)
		}
		updated, err = result.RowsAffected()
		if err != nil {
			return CashOutResult{}, fmt.Errorf("read credited user count: %w", err)
		}
		if updated != 1 {
			return CashOutResult{}, ErrGameNotFound
		}
	}

	if err := tx.Commit(); err != nil {
		return CashOutResult{}, fmt.Errorf("commit cashout transaction: %w", err)
	}

	return CashOutResult{
		Status:     StatusCachedOut,
		Payout:     formatDollars(payout),
		Multiplier: multiplier.FloatString(8),
	}, nil
}

func loadGame(ctx context.Context, tx *sql.Tx, gameID, userID int64) (storedGame, error) {
	var (
		game            storedGame
		openedCellsJSON string
	)
	err := tx.QueryRowContext(
		ctx,
		`SELECT id, userId, betAmount, gridSize, mines, demo, openedCells, status, serverSeed, clientSeed, nonce
		 FROM games
		 WHERE id = ? AND userId = ?`,
		gameID,
		userID,
	).Scan(
		&game.ID,
		&game.UserID,
		&game.BetAmount,
		&game.GridSize,
		&game.Mines,
		&game.Demo,
		&openedCellsJSON,
		&game.Status,
		&game.ServerSeed,
		&game.ClientSeed,
		&game.Nonce,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return storedGame{}, ErrGameNotFound
	}
	if err != nil {
		return storedGame{}, fmt.Errorf("load game: %w", err)
	}
	if err := json.Unmarshal([]byte(openedCellsJSON), &game.OpenedCells); err != nil {
		return storedGame{}, fmt.Errorf("decode opened cells: %w", err)
	}
	if err := validateConfiguration(CreateInput{
		BetAmount: game.BetAmount,
		GridSize:  game.GridSize,
		Mines:     game.Mines,
		Demo:      game.Demo,
	}); err != nil {
		return storedGame{}, err
	}

	return game, nil
}

func multiplierFor(gridSize, mines, openedCells int) (*big.Rat, error) {
	diamonds := gridSize - mines
	if openedCells <= 0 || openedCells > diamonds {
		return nil, fmt.Errorf("invalid opened cell count")
	}

	numerator := new(big.Int).Mul(big.NewInt(int64(gridSize)), big.NewInt(96))
	denominator := new(big.Int).Mul(big.NewInt(int64(diamonds)), big.NewInt(100))
	multiplier := new(big.Rat).SetFrac(numerator, denominator)

	for openedCount := 1; openedCount < openedCells; openedCount++ {
		multiplier.Mul(multiplier, new(big.Rat).SetFrac64(
			int64(gridSize-openedCount),
			int64(diamonds-openedCount),
		))
	}

	return multiplier, nil
}

func payoutFor(betAmount int64, multiplier *big.Rat) (int64, error) {
	if betAmount < 0 {
		return 0, fmt.Errorf("invalid bet amount")
	}

	numerator := new(big.Int).Mul(big.NewInt(betAmount), multiplier.Num())
	payout := new(big.Int).Quo(numerator, multiplier.Denom())
	if !payout.IsInt64() {
		return 0, fmt.Errorf("payout exceeds balance range")
	}

	return payout.Int64(), nil
}

func formatDollars(cents int64) string {
	return fmt.Sprintf("%d.%02d", cents/100, cents%100)
}

func contains(values []int, target int) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}

	return false
}
