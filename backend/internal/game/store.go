package game

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
)

const (
	StatusInProcess = "inProcess"
	StatusCachedOut = "cachedOut"
	StatusFailed    = "failed"
)

var (
	ErrGameNotFound         = errors.New("game not found")
	ErrGameFinished         = errors.New("game is already finished")
	ErrInvalidCell          = errors.New("invalid cell")
	ErrCellAlreadyOpened    = errors.New("cell is already opened")
	ErrNoOpenedCells        = errors.New("game has no opened cells")
	ErrInsufficientBalance  = errors.New("insufficient balance")
	ErrInvalidConfiguration = errors.New("invalid game configuration")
)

type Store struct {
	db *sql.DB
}

type CreateInput struct {
	UserID     int64
	BetAmount  int64
	GridSize   int
	Mines      int
	Demo       bool
	ClientSeed string
}

type Game struct {
	ID     int64  `json:"id"`
	Status string `json:"status"`
}

func NewStore(db *sql.DB) *Store {
	return &Store{db: db}
}

func (s *Store) Create(ctx context.Context, input CreateInput) (Game, error) {
	if input.UserID <= 0 {
		return Game{}, fmt.Errorf("invalid user ID")
	}
	if err := validateConfiguration(input); err != nil {
		return Game{}, err
	}
	if input.ClientSeed == "" {
		return Game{}, fmt.Errorf("client seed is required")
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return Game{}, fmt.Errorf("begin game transaction: %w", err)
	}
	defer tx.Rollback()

	// Reserve the current nonce before inserting so concurrent game creations cannot reuse it.
	var result sql.Result
	if input.Demo {
		result, err = tx.ExecContext(
			ctx,
			"UPDATE users SET transactionNumber = transactionNumber + 1 WHERE id = ? AND transactionNumber >= 0",
			input.UserID,
		)
	} else {
		result, err = tx.ExecContext(
			ctx,
			`UPDATE users
			 SET transactionNumber = transactionNumber + 1, balanceDollars = balanceDollars - ?
			 WHERE id = ? AND transactionNumber >= 0 AND balanceDollars >= ?`,
			input.BetAmount,
			input.UserID,
			input.BetAmount,
		)
	}
	if err != nil {
		return Game{}, fmt.Errorf("increment user transaction number: %w", err)
	}
	updated, err := result.RowsAffected()
	if err != nil {
		return Game{}, fmt.Errorf("read updated user count: %w", err)
	}
	if updated != 1 {
		if input.Demo {
			return Game{}, ErrGameNotFound
		}
		return Game{}, ErrInsufficientBalance
	}

	var (
		serverSeed            string
		nextTransactionNumber int64
	)
	if err := tx.QueryRowContext(
		ctx,
		"SELECT serverSeed, transactionNumber FROM users WHERE id = ?",
		input.UserID,
	).Scan(&serverSeed, &nextTransactionNumber); err != nil {
		return Game{}, fmt.Errorf("read user fairness data: %w", err)
	}
	if serverSeed == "" || nextTransactionNumber <= 0 {
		return Game{}, fmt.Errorf("user has invalid fairness data")
	}

	result, err = tx.ExecContext(
		ctx,
		`INSERT INTO games (userId, betAmount, gridSize, mines, demo, status, serverSeed, clientSeed, nonce)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		input.UserID,
		input.BetAmount,
		input.GridSize,
		input.Mines,
		input.Demo,
		StatusInProcess,
		serverSeed,
		input.ClientSeed,
		nextTransactionNumber-1,
	)
	if err != nil {
		return Game{}, fmt.Errorf("create game: %w", err)
	}
	gameID, err := result.LastInsertId()
	if err != nil {
		return Game{}, fmt.Errorf("read created game ID: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return Game{}, fmt.Errorf("commit game transaction: %w", err)
	}

	return Game{ID: gameID, Status: StatusInProcess}, nil
}

func validateConfiguration(input CreateInput) error {
	switch input.GridSize {
	case 25, 36, 49, 64:
	default:
		return fmt.Errorf("%w: gridSize must be one of 25, 36, 49, or 64", ErrInvalidConfiguration)
	}
	if input.Mines <= 0 || input.Mines >= input.GridSize {
		return fmt.Errorf("%w: mines must be between 1 and gridSize - 1", ErrInvalidConfiguration)
	}
	if input.Demo && input.BetAmount != 0 {
		return fmt.Errorf("%w: demo games require betAmount 0", ErrInvalidConfiguration)
	}
	if !input.Demo && input.BetAmount <= 0 {
		return fmt.Errorf("%w: real games require a positive betAmount", ErrInvalidConfiguration)
	}

	return nil
}
