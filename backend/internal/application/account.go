package application

import (
	"context"
	"errors"
	"fmt"
)

type AccountService struct {
	accounts  AccountRepository
	passwords PasswordVerifier
}

func NewAccountService(accounts AccountRepository, passwords PasswordVerifier) *AccountService {
	return &AccountService{accounts: accounts, passwords: passwords}
}

func (service *AccountService) Authenticate(ctx context.Context, username, password string) (int64, bool, error) {
	user, err := service.accounts.ByUsername(ctx, username)
	if errors.Is(err, ErrNotFound) {
		return 0, false, nil
	}
	if err != nil {
		return 0, false, fmt.Errorf("find account: %w", err)
	}

	matches, err := service.passwords.Matches(user.PasswordHash, password)
	if err != nil {
		return 0, false, fmt.Errorf("verify password: %w", err)
	}
	if !matches {
		return 0, false, nil
	}
	return user.ID, true, nil
}

func (service *AccountService) Balance(ctx context.Context, userID int64) (int64, error) {
	user, err := service.accounts.ByID(ctx, userID)
	if err != nil {
		return 0, fmt.Errorf("find account: %w", err)
	}
	return user.BalanceCents, nil
}

func (service *AccountService) AddBalance(ctx context.Context, username string, amountCents int64) error {
	if err := service.accounts.AddBalance(ctx, username, amountCents); err != nil {
		return fmt.Errorf("add account balance: %w", err)
	}
	return nil
}
