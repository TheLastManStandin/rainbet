package user

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"golang.org/x/crypto/bcrypt"
)

type Store struct {
	db *sql.DB
}

func NewStore(db *sql.DB) *Store {
	return &Store{db: db}
}

func (s *Store) Authenticate(ctx context.Context, username, password string) (int64, bool, error) {
	var (
		userID       int64
		passwordHash string
	)
	err := s.db.QueryRowContext(
		ctx,
		"SELECT id, password_hash FROM users WHERE username = ?",
		username,
	).Scan(&userID, &passwordHash)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, false, nil
	}
	if err != nil {
		return 0, false, fmt.Errorf("find user: %w", err)
	}

	if err := bcrypt.CompareHashAndPassword([]byte(passwordHash), []byte(password)); err != nil {
		if errors.Is(err, bcrypt.ErrMismatchedHashAndPassword) {
			return 0, false, nil
		}
		return 0, false, fmt.Errorf("compare password: %w", err)
	}

	return userID, true, nil
}
