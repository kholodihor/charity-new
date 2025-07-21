package db

import (
	"context"
	"fmt"
	"testing"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/stretchr/testify/require"
)

func TestStoreDonateToGoalTx(t *testing.T) {
	// Create test users and goals
	user1 := createRandomUser(t, testStore)
	user2 := createRandomUser(t, testStore)
	goal1 := createRandomGoal(t, testStore)
	goal2 := createRandomGoal(t, testStore)

	// Print initial amounts
	fmt.Printf(">> before: goal1=%d, goal2=%d\n", goal1.CollectedAmount, goal2.CollectedAmount)

	n := 5
	amount := int64(1000) // $10.00

	errs := make(chan error)
	results := make(chan DonateToGoalTxResult)

	// Run n concurrent donation transactions
	for i := 0; i < n; i++ {
		// Alternate between users and goals
		userID := user1.ID
		goalID := goal1.ID
		isAnonymous := false

		if i%2 == 0 {
			userID = user2.ID
			goalID = goal2.ID
		}

		// Make some donations anonymous
		if i%3 == 0 {
			isAnonymous = true
		}

		go func(userID int64, goalID int64, isAnonymous bool) {
			var result DonateToGoalTxResult
			var err error

			if isAnonymous {
				result, err = testStore.DonateToGoalTx(context.Background(), DonateToGoalTxParams{
					GoalID:      goalID,
					UserID:      pgtype.Int8{Valid: false},
					Amount:      amount,
					IsAnonymous: true,
				})
			} else {
				result, err = testStore.DonateToGoalTx(context.Background(), DonateToGoalTxParams{
					GoalID:      goalID,
					UserID:      pgtype.Int8{Int64: userID, Valid: true},
					Amount:      amount,
					IsAnonymous: false,
				})
			}

			errs <- err
			results <- result
		}(userID, goalID, isAnonymous)
	}

	// Check results
	donatedToGoal1 := make(map[int64]bool)
	donatedToGoal2 := make(map[int64]bool)

	for i := 0; i < n; i++ {
		err := <-errs
		require.NoError(t, err)

		result := <-results
		require.NotEmpty(t, result)

		// Check donation
		donation := result.Donation
		require.NotEmpty(t, donation)
		require.NotZero(t, donation.ID)
		require.NotZero(t, donation.CreatedAt)
		require.Equal(t, amount, donation.Amount)

		// Check goal
		goal := result.Goal
		require.NotEmpty(t, goal)
		require.True(t, goal.ID == goal1.ID || goal.ID == goal2.ID)

		// Track which users donated to which goals
		if donation.UserID.Valid {
			if goal.ID == goal1.ID {
				donatedToGoal1[donation.UserID.Int64] = true
			} else {
				donatedToGoal2[donation.UserID.Int64] = true
			}
		}

		// Check user if not anonymous
		if !donation.IsAnonymous {
			user := result.User
			require.NotEmpty(t, user)
			require.Equal(t, donation.UserID.Int64, user.ID)
		}

		// Verify the donation was recorded in the goal's collected amount
		updatedGoal, err := testStore.GetGoal(context.Background(), goal.ID)
		require.NoError(t, err)
		require.GreaterOrEqual(t, updatedGoal.CollectedAmount, goal.CollectedAmount)
	}

	// Verify the final collected amounts
	finalGoal1, err := testStore.GetGoal(context.Background(), goal1.ID)
	require.NoError(t, err)
	finalGoal2, err := testStore.GetGoal(context.Background(), goal2.ID)
	require.NoError(t, err)

	fmt.Printf(">> after: goal1=%d, goal2=%d\n", finalGoal1.CollectedAmount, finalGoal2.CollectedAmount)
}

func TestStoreDonateToGoalTxDeadlock(t *testing.T) {
	// Test for deadlocks with concurrent donations to the same goal
	goal := createRandomGoal(t, testStore)

	n := 10
	amount := int64(100)
	errs := make(chan error)

	for i := 0; i < n; i++ {
		user := createRandomUser(t, testStore)

		go func(user User) {
			_, err := testStore.DonateToGoalTx(context.Background(), DonateToGoalTxParams{
				GoalID:      goal.ID,
				UserID:      pgtype.Int8{Int64: user.ID, Valid: true},
				Amount:      amount,
				IsAnonymous: false,
			})

			errs <- err
		}(user)
	}

	for i := 0; i < n; i++ {
		err := <-errs
		require.NoError(t, err)
	}

	// Verify the final collected amount
	updatedGoal, err := testStore.GetGoal(context.Background(), goal.ID)
	require.NoError(t, err)
	expectedAmount := goal.CollectedAmount + (amount * int64(n))
	require.Equal(t, expectedAmount, updatedGoal.CollectedAmount)
}
