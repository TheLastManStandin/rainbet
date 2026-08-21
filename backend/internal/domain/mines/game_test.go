package mines

import (
	"errors"
	"math/big"
	"testing"
)

func TestMultiplierForSupportedBoards(t *testing.T) {
	tests := []struct {
		name                string
		grid, mines, opened int
		want                *big.Rat
	}{
		{"25 cells first diamond", 25, 15, 1, big.NewRat(24, 10)},
		{"25 cells second diamond", 25, 15, 2, big.NewRat(576, 90)},
		{"36 cells", 36, 10, 2, big.NewRat(3024, 1625)},
		{"49 cells", 49, 15, 2, big.NewRat(9408, 4675)},
		{"64 cells", 64, 24, 2, big.NewRat(4032, 1625)},
		{"all diamonds", 25, 23, 2, big.NewRat(288, 1)},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := multiplierFor(test.grid, test.mines, test.opened)
			if err != nil {
				t.Fatalf("multiplier: %v", err)
			}
			if got.Cmp(test.want) != 0 {
				t.Fatalf("multiplier = %s, want %s", got, test.want)
			}
		})
	}
}

func TestGameMoveAndCashOut(t *testing.T) {
	game, err := New(1, Config{BetCents: 1000, GridSize: 25, Mines: 3, ClientSeed: "client"}, "server", 0)
	if err != nil {
		t.Fatalf("new game: %v", err)
	}
	game.ID = 1

	move, err := game.Open(0, []int{1, 2, 3})
	if err != nil {
		t.Fatalf("open diamond: %v", err)
	}
	if move.Bomb || move.MultiplierHundredths != 109 || game.Status != StatusInProcess {
		t.Fatalf("move = %+v, status = %q", move, game.Status)
	}
	cashout, err := game.CashOut([]int{1, 2, 3})
	if err != nil {
		t.Fatalf("cash out: %v", err)
	}
	if cashout.PayoutCents != 1090 || game.Status != StatusCachedOut {
		t.Fatalf("cashout = %+v, status = %q", cashout, game.Status)
	}
	if _, err := game.Open(4, []int{1, 2, 3}); !errors.Is(err, ErrGameFinished) {
		t.Fatalf("move after cashout error = %v", err)
	}
}

func TestBombFailsGame(t *testing.T) {
	game, err := New(1, Config{BetCents: 0, GridSize: 25, Mines: 3, Demo: true, ClientSeed: "client"}, "server", 0)
	if err != nil {
		t.Fatalf("new game: %v", err)
	}
	move, err := game.Open(2, []int{1, 2, 3})
	if err != nil {
		t.Fatalf("open bomb: %v", err)
	}
	if !move.Bomb || game.Status != StatusFailed {
		t.Fatalf("move = %+v, status = %q", move, game.Status)
	}
}
