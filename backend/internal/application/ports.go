package application

import (
	"context"
	"errors"

	"rainbet/internal/domain/account"
	"rainbet/internal/domain/mines"
)

var (
	ErrNotFound = errors.New("not found")
	ErrConflict = errors.New("concurrent update")
)

type AccountRepository interface {
	ByID(ctx context.Context, id int64) (*account.Account, error)
	ByUsername(ctx context.Context, username string) (*account.Account, error)
	Save(ctx context.Context, account *account.Account, expectedTransactionNumber int64) error
	AddBalance(ctx context.Context, username string, amountCents int64) error
}

type GameRepository interface {
	Create(ctx context.Context, game *mines.Game) error
	ByIDAndUser(ctx context.Context, gameID, userID int64) (*mines.Game, error)
	Save(ctx context.Context, game *mines.Game, expectedStatus mines.Status) error
}

type UnitOfWork interface {
	WithinTransaction(ctx context.Context, fn func(AccountRepository, GameRepository) error) error
}

type PasswordVerifier interface {
	Matches(passwordHash, password string) (bool, error)
}

type MineGenerator interface {
	Indexes(game *mines.Game) ([]int, error)
}
