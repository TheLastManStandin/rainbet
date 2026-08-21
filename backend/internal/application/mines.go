package application

import (
	"context"
	"errors"
	"fmt"

	"rainbet/internal/domain/mines"
)

var ErrGameNotFound = errors.New("game not found")

type CreateGameCommand struct {
	UserID     int64
	BetCents   int64
	GridSize   int
	Mines      int
	Demo       bool
	ClientSeed string
}

type CreatedGame struct {
	ID     int64
	Status mines.Status
}

type MoveCommand struct {
	UserID    int64
	GameID    int64
	CellIndex int
}

type MoveResult struct {
	Status               mines.Status
	Bomb                 bool
	OpenedCells          []int
	MultiplierHundredths int64
	MineIndexes          []int
}

type CashOutCommand struct {
	UserID int64
	GameID int64
}

type CashOutResult struct {
	Status               mines.Status
	PayoutCents          int64
	MultiplierHundredths int64
	MineIndexes          []int
}

type MinesService struct {
	unitOfWork UnitOfWork
	mines      MineGenerator
}

func NewMinesService(unitOfWork UnitOfWork, generator MineGenerator) *MinesService {
	return &MinesService{unitOfWork: unitOfWork, mines: generator}
}

func (service *MinesService) Create(ctx context.Context, command CreateGameCommand) (CreatedGame, error) {
	config := mines.Config{
		BetCents:   command.BetCents,
		GridSize:   command.GridSize,
		Mines:      command.Mines,
		Demo:       command.Demo,
		ClientSeed: command.ClientSeed,
	}
	if err := mines.ValidateConfig(config); err != nil {
		return CreatedGame{}, err
	}

	var created *mines.Game
	err := service.unitOfWork.WithinTransaction(ctx, func(accounts AccountRepository, games GameRepository) error {
		user, err := accounts.ByID(ctx, command.UserID)
		if errors.Is(err, ErrNotFound) {
			return ErrGameNotFound
		}
		if err != nil {
			return fmt.Errorf("load account: %w", err)
		}

		expectedNonce := user.TransactionNumber
		nonce, err := user.ReserveGame(command.BetCents, command.Demo)
		if err != nil {
			return err
		}
		created, err = mines.New(user.ID, config, user.ServerSeed, nonce)
		if err != nil {
			return err
		}
		if err := accounts.Save(ctx, user, expectedNonce); err != nil {
			return fmt.Errorf("reserve game funds: %w", err)
		}
		if err := games.Create(ctx, created); err != nil {
			return fmt.Errorf("store game: %w", err)
		}
		return nil
	})
	if err != nil {
		return CreatedGame{}, err
	}
	return CreatedGame{ID: created.ID, Status: created.Status}, nil
}

func (service *MinesService) Move(ctx context.Context, command MoveCommand) (MoveResult, error) {
	if command.UserID <= 0 || command.GameID <= 0 {
		return MoveResult{}, ErrGameNotFound
	}

	var result MoveResult
	err := service.unitOfWork.WithinTransaction(ctx, func(_ AccountRepository, games GameRepository) error {
		game, err := games.ByIDAndUser(ctx, command.GameID, command.UserID)
		if errors.Is(err, ErrNotFound) {
			return ErrGameNotFound
		}
		if err != nil {
			return fmt.Errorf("load game: %w", err)
		}
		mineIndexes, err := service.mines.Indexes(game)
		if err != nil {
			return fmt.Errorf("determine mine indexes: %w", err)
		}
		move, err := game.Open(command.CellIndex, mineIndexes)
		if err != nil {
			return err
		}
		if err := games.Save(ctx, game, mines.StatusInProcess); err != nil {
			if errors.Is(err, ErrConflict) {
				return mines.ErrGameFinished
			}
			return fmt.Errorf("store move: %w", err)
		}
		result = MoveResult{
			Status:               game.Status,
			Bomb:                 move.Bomb,
			OpenedCells:          move.OpenedCells,
			MultiplierHundredths: move.MultiplierHundredths,
			MineIndexes:          move.MineIndexes,
		}
		return nil
	})
	return result, err
}

func (service *MinesService) CashOut(ctx context.Context, command CashOutCommand) (CashOutResult, error) {
	if command.UserID <= 0 || command.GameID <= 0 {
		return CashOutResult{}, ErrGameNotFound
	}

	var result CashOutResult
	err := service.unitOfWork.WithinTransaction(ctx, func(accounts AccountRepository, games GameRepository) error {
		game, err := games.ByIDAndUser(ctx, command.GameID, command.UserID)
		if errors.Is(err, ErrNotFound) {
			return ErrGameNotFound
		}
		if err != nil {
			return fmt.Errorf("load game: %w", err)
		}
		mineIndexes, err := service.mines.Indexes(game)
		if err != nil {
			return fmt.Errorf("determine mine indexes: %w", err)
		}
		cashout, err := game.CashOut(mineIndexes)
		if err != nil {
			return err
		}
		if err := games.Save(ctx, game, mines.StatusInProcess); err != nil {
			if errors.Is(err, ErrConflict) {
				return mines.ErrGameFinished
			}
			return fmt.Errorf("finish game: %w", err)
		}
		if !game.Demo && cashout.PayoutCents > 0 {
			user, err := accounts.ByID(ctx, game.UserID)
			if errors.Is(err, ErrNotFound) {
				return ErrGameNotFound
			}
			if err != nil {
				return fmt.Errorf("load account: %w", err)
			}
			expectedNonce := user.TransactionNumber
			if err := user.Credit(cashout.PayoutCents); err != nil {
				return err
			}
			if err := accounts.Save(ctx, user, expectedNonce); err != nil {
				return fmt.Errorf("credit winnings: %w", err)
			}
		}
		result = CashOutResult{
			Status:               game.Status,
			PayoutCents:          cashout.PayoutCents,
			MultiplierHundredths: cashout.MultiplierHundredths,
			MineIndexes:          cashout.MineIndexes,
		}
		return nil
	})
	return result, err
}
