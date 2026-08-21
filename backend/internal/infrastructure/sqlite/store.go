package sqlite

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"

	"rainbet/internal/application"
	"rainbet/internal/domain/account"
	"rainbet/internal/domain/mines"
)

type Store struct {
	db *sql.DB
}

type querier interface {
	ExecContext(context.Context, string, ...any) (sql.Result, error)
	QueryRowContext(context.Context, string, ...any) *sql.Row
}

type accountRepository struct{ queries querier }
type gameRepository struct{ queries querier }

func NewStore(db *sql.DB) *Store { return &Store{db: db} }

func (store *Store) Accounts() application.AccountRepository {
	return &accountRepository{queries: store.db}
}

func (store *Store) WithinTransaction(ctx context.Context, fn func(application.AccountRepository, application.GameRepository) error) error {
	tx, err := store.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback()
	if err := fn(&accountRepository{queries: tx}, &gameRepository{queries: tx}); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit transaction: %w", err)
	}
	return nil
}

func (repository *accountRepository) ByID(ctx context.Context, id int64) (*account.Account, error) {
	return repository.find(ctx, "WHERE id = ?", id)
}

func (repository *accountRepository) ByUsername(ctx context.Context, username string) (*account.Account, error) {
	return repository.find(ctx, "WHERE username = ?", username)
}

func (repository *accountRepository) find(ctx context.Context, where string, argument any) (*account.Account, error) {
	var user account.Account
	err := repository.queries.QueryRowContext(ctx, `
		SELECT id, username, password_hash, balanceDollars, serverSeed, transactionNumber
		FROM users `+where, argument).Scan(
		&user.ID, &user.Username, &user.PasswordHash, &user.BalanceCents,
		&user.ServerSeed, &user.TransactionNumber,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, application.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("query account: %w", err)
	}
	return &user, nil
}

func (repository *accountRepository) Save(ctx context.Context, user *account.Account, expectedTransactionNumber int64) error {
	result, err := repository.queries.ExecContext(ctx, `
		UPDATE users
		SET balanceDollars = ?, transactionNumber = ?
		WHERE id = ? AND transactionNumber = ?
	`, user.BalanceCents, user.TransactionNumber, user.ID, expectedTransactionNumber)
	if err != nil {
		return fmt.Errorf("update account: %w", err)
	}
	updated, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("read updated account count: %w", err)
	}
	if updated != 1 {
		return application.ErrConflict
	}
	return nil
}

func (repository *accountRepository) AddBalance(ctx context.Context, username string, amountCents int64) error {
	if _, err := repository.queries.ExecContext(ctx,
		"UPDATE users SET balanceDollars = balanceDollars + ? WHERE username = ?",
		amountCents, username,
	); err != nil {
		return fmt.Errorf("add account balance: %w", err)
	}
	return nil
}

func (repository *gameRepository) Create(ctx context.Context, game *mines.Game) error {
	openedCells, err := json.Marshal(game.OpenedCells)
	if err != nil {
		return fmt.Errorf("encode opened cells: %w", err)
	}
	result, err := repository.queries.ExecContext(ctx, `
		INSERT INTO games (userId, betAmount, gridSize, mines, demo, openedCells, status, serverSeed, clientSeed, nonce)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, game.UserID, game.BetCents, game.GridSize, game.Mines, game.Demo, string(openedCells),
		string(game.Status), game.ServerSeed, game.ClientSeed, game.Nonce)
	if err != nil {
		return fmt.Errorf("insert game: %w", err)
	}
	game.ID, err = result.LastInsertId()
	if err != nil {
		return fmt.Errorf("read inserted game ID: %w", err)
	}
	return nil
}

func (repository *gameRepository) ByIDAndUser(ctx context.Context, gameID, userID int64) (*mines.Game, error) {
	var game mines.Game
	var openedCells, status string
	err := repository.queries.QueryRowContext(ctx, `
		SELECT id, userId, betAmount, gridSize, mines, demo, openedCells, status, serverSeed, clientSeed, nonce
		FROM games WHERE id = ? AND userId = ?
	`, gameID, userID).Scan(
		&game.ID, &game.UserID, &game.BetCents, &game.GridSize, &game.Mines, &game.Demo,
		&openedCells, &status, &game.ServerSeed, &game.ClientSeed, &game.Nonce,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, application.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("query game: %w", err)
	}
	game.Status = mines.Status(status)
	if err := json.Unmarshal([]byte(openedCells), &game.OpenedCells); err != nil {
		return nil, fmt.Errorf("decode opened cells: %w", err)
	}
	if err := game.Validate(); err != nil {
		return nil, err
	}
	return &game, nil
}

func (repository *gameRepository) Save(ctx context.Context, game *mines.Game, expectedStatus mines.Status) error {
	openedCells, err := json.Marshal(game.OpenedCells)
	if err != nil {
		return fmt.Errorf("encode opened cells: %w", err)
	}
	result, err := repository.queries.ExecContext(ctx, `
		UPDATE games SET openedCells = ?, status = ?
		WHERE id = ? AND userId = ? AND status = ?
	`, string(openedCells), string(game.Status), game.ID, game.UserID, string(expectedStatus))
	if err != nil {
		return fmt.Errorf("update game: %w", err)
	}
	updated, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("read updated game count: %w", err)
	}
	if updated != 1 {
		return application.ErrConflict
	}
	return nil
}
