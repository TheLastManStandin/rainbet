package account

import (
	"errors"
	"testing"
)

func TestReserveGame(t *testing.T) {
	user := Account{ID: 1, BalanceCents: 10_000, ServerSeed: "seed", TransactionNumber: 7}
	nonce, err := user.ReserveGame(1_050, false)
	if err != nil {
		t.Fatalf("reserve game: %v", err)
	}
	if nonce != 7 || user.TransactionNumber != 8 || user.BalanceCents != 8_950 {
		t.Fatalf("account after reservation = %+v, nonce = %d", user, nonce)
	}
}

func TestReserveGameRejectsInsufficientBalance(t *testing.T) {
	user := Account{ID: 1, BalanceCents: 100, ServerSeed: "seed"}
	_, err := user.ReserveGame(101, false)
	if !errors.Is(err, ErrInsufficientBalance) {
		t.Fatalf("error = %v, want %v", err, ErrInsufficientBalance)
	}
	if user.BalanceCents != 100 || user.TransactionNumber != 0 {
		t.Fatalf("rejected reservation changed account: %+v", user)
	}
}
