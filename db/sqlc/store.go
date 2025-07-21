package db

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Store defines all functions to execute db queries and transactions
type Store interface {
	Querier
	DonateToGoalTx(ctx context.Context, arg DonateToGoalTxParams) (DonateToGoalTxResult, error)
}

// SQLStore provides all functions to execute SQL queries and transactions
type SQLStore struct {
	connPool *pgxpool.Pool
	*Queries
}

// NewStore creates a new store
func NewStore(connPool *pgxpool.Pool) Store {
	return &SQLStore{
		connPool: connPool,
		Queries:  New(connPool),
	}
}

// ExecTx executes a function within a database transaction
func (s *SQLStore) execTx(ctx context.Context, fn func(*Queries) error) error {
	tx, err := s.connPool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return err
	}

	q := New(tx)
	err = fn(q)
	if err != nil {
		if rbErr := tx.Rollback(ctx); rbErr != nil {
			return fmt.Errorf("tx err: %v, rb err: %v", err, rbErr)
		}
		return err
	}

	return tx.Commit(ctx)
}

// DonateToGoalTxParams contains the input parameters for the DonateToGoal transaction
type DonateToGoalTxParams struct {
	GoalID      int64       `json:"goal_id"`
	UserID      pgtype.Int8 `json:"user_id"`
	Amount      int64       `json:"amount"`
	IsAnonymous bool        `json:"is_anonymous"`
}

// DonateToGoalTxResult is the result of the DonateToGoal transaction
type DonateToGoalTxResult struct {
	Donation Donation `json:"donation"`
	Goal     Goal     `json:"goal"`
	User     User     `json:"user"`
}

// DonateToGoalTx performs a donation to a goal
func (store *SQLStore) DonateToGoalTx(ctx context.Context, arg DonateToGoalTxParams) (DonateToGoalTxResult, error) {
	var result DonateToGoalTxResult

	err := store.execTx(ctx, func(q *Queries) error {
		var err error

		// 1. Create donation record
		result.Donation, err = q.CreateDonation(ctx, CreateDonationParams{
			GoalID:      arg.GoalID,
			UserID:      arg.UserID,
			Amount:      arg.Amount,
			IsAnonymous: arg.IsAnonymous,
		})
		if err != nil {
			return fmt.Errorf("create donation: %w", err)
		}

		// 2. Get current goal to calculate new collected amount
		goal, err := q.GetGoal(ctx, arg.GoalID)
		if err != nil {
			return fmt.Errorf("get goal: %w", err)
		}

		// 3. Update goal's collected amount
		newCollectedAmount := goal.CollectedAmount + arg.Amount
		err = q.UpdateGoalCollectedAmount(ctx, UpdateGoalCollectedAmountParams{
			ID:              arg.GoalID,
			CollectedAmount: newCollectedAmount,
		})
		if err != nil {
			return fmt.Errorf("update goal collected amount: %w", err)
		}

		// 4. Get the updated goal
		result.Goal, err = q.GetGoal(ctx, arg.GoalID)
		if err != nil {
			return fmt.Errorf("get updated goal: %w", err)
		}

		// 5. Get the user who made the donation (only if not anonymous)
		if arg.UserID.Valid {
			user, err := q.GetUser(ctx, arg.UserID.Int64)
			if err != nil {
				return fmt.Errorf("get user: %w", err)
			}
			result.User = user
		}

		return nil
	})

	if err != nil {
		return DonateToGoalTxResult{}, fmt.Errorf("DonateToGoalTx: %w", err)
	}

	return result, nil
}
