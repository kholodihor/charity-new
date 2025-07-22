package db

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5/pgtype"
)

// DonateToGoalTxParams contains the input parameters of the donation transaction
type DonateToGoalTxParams struct {
	UserID      pgtype.Int8 `json:"user_id"`
	GoalID      int64       `json:"goal_id"`
	Amount      int64       `json:"amount"`
	IsAnonymous bool        `json:"is_anonymous"`
}

// DonateToGoalTxResult is the result of the donation transaction
type DonateToGoalTxResult struct {
	Donation    Donation `json:"donation"`
	User        User     `json:"user"`
	Goal        Goal     `json:"goal"`
	FromBalance int64    `json:"from_balance"`
	ToBalance   int64    `json:"to_balance"`
}

// DonateToGoalTx performs a donation from a user to a goal.
// It creates the donation, updates the user's balance, and updates the goal's collected amount within a database transaction
func (store *SQLStore) DonateToGoalTx(ctx context.Context, arg DonateToGoalTxParams) (DonateToGoalTxResult, error) {
	var result DonateToGoalTxResult

	err := store.execTx(ctx, func(q *Queries) error {
		var err error

		// Validate donation amount
		if arg.Amount <= 0 {
			return errors.New("donation amount must be positive")
		}

		// Get goal and check if it's active
		goal, err := q.GetGoal(ctx, arg.GoalID)
		if err != nil {
			return err
		}
		if !goal.IsActive {
			return errors.New("cannot donate to inactive goal")
		}

		// Check user balance if not anonymous
		if arg.UserID.Valid {
			user, err := q.GetUser(ctx, arg.UserID.Int64)
			if err != nil {
				return err
			}
			if user.Balance < arg.Amount {
				return errors.New("insufficient balance")
			}

			// Calculate new balance
			newBalance := user.Balance - arg.Amount
			
			// Update user balance
			result.User, err = q.UpdateUserBalance(ctx, UpdateUserBalanceParams{
				ID:      arg.UserID.Int64,
				Balance: newBalance,
			})
			if err != nil {
				return err
			}
			result.FromBalance = user.Balance
			result.ToBalance = result.User.Balance
		}

		// Create donation
		result.Donation, err = q.CreateDonation(ctx, CreateDonationParams{
			UserID:      arg.UserID,
			GoalID:      arg.GoalID,
			Amount:      arg.Amount,
			IsAnonymous: arg.IsAnonymous,
		})
		if err != nil {
			return err
		}

		// Update goal collected amount (SQL query adds to current amount)
		err = q.UpdateGoalCollectedAmount(ctx, UpdateGoalCollectedAmountParams{
			ID:              arg.GoalID,
			CollectedAmount: arg.Amount, // SQL will add this to current collected_amount
		})
		if err != nil {
			return err
		}
		
		// Get updated goal
		result.Goal, err = q.GetGoal(ctx, arg.GoalID)
		if err != nil {
			return err
		}

		return nil
	})

	return result, err
}
