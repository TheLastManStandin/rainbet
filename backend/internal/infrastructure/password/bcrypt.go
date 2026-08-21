package password

import (
	"errors"

	"golang.org/x/crypto/bcrypt"
)

type Bcrypt struct{}

func (Bcrypt) Matches(passwordHash, password string) (bool, error) {
	err := bcrypt.CompareHashAndPassword([]byte(passwordHash), []byte(password))
	if errors.Is(err, bcrypt.ErrMismatchedHashAndPassword) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}
