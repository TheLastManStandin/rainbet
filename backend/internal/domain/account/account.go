package account

import "errors"

var (
	ErrInvalidAccount      = errors.New("invalid account")
	ErrInsufficientBalance = errors.New("insufficient balance")
)

// Account is the domain representation of a Rainbet user. Monetary values are
// stored as integer cents so the domain never relies on floating-point money.
type Account struct {
	ID                int64
	Username          string
	PasswordHash      string
	BalanceCents      int64
	ServerSeed        string
	TransactionNumber int64
}

// ReserveGame advances the provably-fair nonce and, for a real game, reserves
// the stake. It returns the nonce that belongs to the newly created game.
func (a *Account) ReserveGame(betCents int64, demo bool) (int64, error) {
	if a.ID <= 0 || a.ServerSeed == "" || a.TransactionNumber < 0 {
		return 0, ErrInvalidAccount
	}
	if demo {
		if betCents != 0 {
			return 0, ErrInvalidAccount
		}
	} else {
		if betCents <= 0 {
			return 0, ErrInvalidAccount
		}
		if a.BalanceCents < betCents {
			return 0, ErrInsufficientBalance
		}
		a.BalanceCents -= betCents
	}

	nonce := a.TransactionNumber
	a.TransactionNumber++
	return nonce, nil
}

func (a *Account) Credit(amountCents int64) error {
	if a.ID <= 0 || amountCents < 0 {
		return ErrInvalidAccount
	}
	a.BalanceCents += amountCents
	return nil
}
