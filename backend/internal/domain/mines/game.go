package mines

import (
	"errors"
	"fmt"
	"math/big"
)

type Status string

const (
	StatusInProcess Status = "inProcess"
	StatusCachedOut Status = "cachedOut"
	StatusFailed    Status = "failed"
)

var (
	ErrGameFinished         = errors.New("game is already finished")
	ErrInvalidCell          = errors.New("invalid cell")
	ErrCellAlreadyOpened    = errors.New("cell is already opened")
	ErrNoOpenedCells        = errors.New("game has no opened cells")
	ErrInvalidConfiguration = errors.New("invalid game configuration")
)

const maxMultiplierHundredths = int64(100000000)

type Config struct {
	BetCents   int64
	GridSize   int
	Mines      int
	Demo       bool
	ClientSeed string
}

type Game struct {
	ID          int64
	UserID      int64
	BetCents    int64
	GridSize    int
	Mines       int
	Demo        bool
	OpenedCells []int
	Status      Status
	ServerSeed  string
	ClientSeed  string
	Nonce       int64
}

type Move struct {
	Bomb                 bool
	OpenedCells          []int
	MultiplierHundredths int64
	MineIndexes          []int
}

type CashOut struct {
	PayoutCents          int64
	MultiplierHundredths int64
	MineIndexes          []int
}

func New(userID int64, config Config, serverSeed string, nonce int64) (*Game, error) {
	if userID <= 0 || serverSeed == "" || nonce < 0 {
		return nil, fmt.Errorf("%w: invalid fairness data", ErrInvalidConfiguration)
	}
	if err := ValidateConfig(config); err != nil {
		return nil, err
	}

	return &Game{
		UserID:      userID,
		BetCents:    config.BetCents,
		GridSize:    config.GridSize,
		Mines:       config.Mines,
		Demo:        config.Demo,
		OpenedCells: []int{},
		Status:      StatusInProcess,
		ServerSeed:  serverSeed,
		ClientSeed:  config.ClientSeed,
		Nonce:       nonce,
	}, nil
}

func ValidateConfig(config Config) error {
	switch config.GridSize {
	case 25, 36, 49, 64:
	default:
		return fmt.Errorf("%w: gridSize must be one of 25, 36, 49, or 64", ErrInvalidConfiguration)
	}
	if config.Mines <= 0 || config.Mines >= config.GridSize {
		return fmt.Errorf("%w: mines must be between 1 and gridSize - 1", ErrInvalidConfiguration)
	}
	if config.ClientSeed == "" {
		return fmt.Errorf("%w: client seed is required", ErrInvalidConfiguration)
	}
	if config.Demo && config.BetCents != 0 {
		return fmt.Errorf("%w: demo games require betAmount 0", ErrInvalidConfiguration)
	}
	if !config.Demo && config.BetCents <= 0 {
		return fmt.Errorf("%w: real games require a positive betAmount", ErrInvalidConfiguration)
	}
	return nil
}

func (g *Game) Validate() error {
	if g.ID <= 0 || g.UserID <= 0 || g.ServerSeed == "" || g.Nonce < 0 {
		return fmt.Errorf("%w: invalid persisted game", ErrInvalidConfiguration)
	}
	return ValidateConfig(Config{
		BetCents:   g.BetCents,
		GridSize:   g.GridSize,
		Mines:      g.Mines,
		Demo:       g.Demo,
		ClientSeed: g.ClientSeed,
	})
}

func (g *Game) Open(cellIndex int, mineIndexes []int) (Move, error) {
	if g.Status != StatusInProcess {
		return Move{}, ErrGameFinished
	}
	if cellIndex < 0 || cellIndex >= g.GridSize {
		return Move{}, ErrInvalidCell
	}
	if contains(g.OpenedCells, cellIndex) {
		return Move{}, ErrCellAlreadyOpened
	}

	g.OpenedCells = append(g.OpenedCells, cellIndex)
	move := Move{OpenedCells: append([]int(nil), g.OpenedCells...)}
	if contains(mineIndexes, cellIndex) {
		g.Status = StatusFailed
		move.Bomb = true
		move.MineIndexes = append([]int(nil), mineIndexes...)
		return move, nil
	}

	multiplier, err := multiplierFor(g.GridSize, g.Mines, len(g.OpenedCells))
	if err != nil {
		return Move{}, err
	}
	move.MultiplierHundredths = truncateMultiplier(multiplier)
	return move, nil
}

func (g *Game) CashOut(mineIndexes []int) (CashOut, error) {
	if g.Status != StatusInProcess {
		return CashOut{}, ErrGameFinished
	}
	if len(g.OpenedCells) == 0 {
		return CashOut{}, ErrNoOpenedCells
	}

	multiplier, err := multiplierFor(g.GridSize, g.Mines, len(g.OpenedCells))
	if err != nil {
		return CashOut{}, err
	}
	hundredths := truncateMultiplier(multiplier)
	payout, err := payoutFor(g.BetCents, hundredths)
	if err != nil {
		return CashOut{}, err
	}
	g.Status = StatusCachedOut

	return CashOut{
		PayoutCents:          payout,
		MultiplierHundredths: hundredths,
		MineIndexes:          append([]int(nil), mineIndexes...),
	}, nil
}

func multiplierFor(gridSize, mines, openedCells int) (*big.Rat, error) {
	diamonds := gridSize - mines
	if openedCells <= 0 || openedCells > diamonds {
		return nil, errors.New("invalid opened cell count")
	}

	multiplier := new(big.Rat).SetFrac64(int64(gridSize*96), int64(diamonds*100))
	for opened := 1; opened < openedCells; opened++ {
		multiplier.Mul(multiplier, new(big.Rat).SetFrac64(
			int64(gridSize-opened),
			int64(diamonds-opened),
		))
	}
	return multiplier, nil
}

func truncateMultiplier(multiplier *big.Rat) int64 {
	hundredths := new(big.Rat).Mul(multiplier, big.NewRat(100, 1))
	units := new(big.Int).Quo(hundredths.Num(), hundredths.Denom())
	if units.Cmp(big.NewInt(maxMultiplierHundredths)) > 0 {
		return maxMultiplierHundredths
	}
	return units.Int64()
}

func payoutFor(betCents, multiplierHundredths int64) (int64, error) {
	if betCents < 0 || multiplierHundredths < 0 {
		return 0, errors.New("invalid payout input")
	}
	value := new(big.Int).Mul(big.NewInt(betCents), big.NewInt(multiplierHundredths))
	value.Quo(value, big.NewInt(100))
	if !value.IsInt64() {
		return 0, errors.New("payout exceeds balance range")
	}
	return value.Int64(), nil
}

func contains(values []int, target int) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}
