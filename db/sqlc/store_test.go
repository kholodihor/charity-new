package db

import (
	"context"
	"fmt"
	"testing"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/kholodihor/charity/util"
	"github.com/stretchr/testify/require"
)

// Test helper functions following the example pattern
func createRandomUser(t *testing.T, store Store) User {
	email, name := util.RandomUserParams()
	arg := CreateUserParams{
		Email: email,
		Name:  name,
	}

	user, err := store.CreateUser(context.Background(), arg)
	require.NoError(t, err)
	require.NotEmpty(t, user)

	require.Equal(t, arg.Email, user.Email)
	require.Equal(t, arg.Name, user.Name)
	require.Equal(t, int64(1000000), user.Balance, "User should have $10,000 default balance")
	require.NotZero(t, user.ID)
	require.NotZero(t, user.CreatedAt)

	return user
}

func createRandomGoal(t *testing.T, store Store) Goal {
	title, description, targetAmount, isActive := util.RandomGoalParams()
	arg := CreateGoalParams{
		Title:        title,
		Description:  description,
		TargetAmount: targetAmount,
		IsActive:     isActive,
	}

	goal, err := store.CreateGoal(context.Background(), arg)
	require.NoError(t, err)
	require.NotEmpty(t, goal)

	require.Equal(t, arg.Title, goal.Title)
	require.Equal(t, arg.Description, goal.Description)
	require.Equal(t, arg.TargetAmount, goal.TargetAmount)
	require.Zero(t, goal.CollectedAmount)
	require.Equal(t, arg.IsActive, goal.IsActive)
	require.NotZero(t, goal.ID)
	require.NotZero(t, goal.CreatedAt)

	return goal
}



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
	initialAmount := goal.CollectedAmount

	n := 10
	amount := int64(100)
	errs := make(chan error, n)

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

	// Wait for all goroutines to complete
	for i := 0; i < n; i++ {
		err := <-errs
		require.NoError(t, err, "concurrent donations should not deadlock")
	}

	// Verify the final state
	updatedGoal, err := testStore.GetGoal(context.Background(), goal.ID)
	require.NoError(t, err)
	expectedAmount := initialAmount + int64(n)*amount
	require.Equal(t, expectedAmount, updatedGoal.CollectedAmount, "all donations should be processed")
}

// TestDonateToGoalTx_Concurrent verifies that concurrent donations from multiple users to the same goal work correctly
func TestDonateToGoalTx_Concurrent(t *testing.T) {
	// Create a test goal
	goal := createRandomGoal(t, testStore)
	n := 5                // Number of concurrent donations
	amount := int64(1000) // $10.00

	// Channel to collect results
	errs := make(chan error, n)

	// Run concurrent donations from different users
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

	// Check results
	for i := 0; i < n; i++ {
		err := <-errs
		require.NoError(t, err, "concurrent donations should succeed")
	}

	// Verify the final state
	updatedGoal, err := testStore.GetGoal(context.Background(), goal.ID)
	require.NoError(t, err)
	require.Equal(t, int64(n)*amount, updatedGoal.CollectedAmount, "all donations should be processed")
}

// TestDonateToGoalTx_InsufficientBalance verifies that donations fail when user has insufficient balance
func TestDonateToGoalTx_InsufficientBalance(t *testing.T) {
	// Create a test goal and user with default balance
	user := createRandomUser(t, testStore)
	goal := createRandomGoal(t, testStore)

	// Try to donate more than the user's balance
	_, err := testStore.DonateToGoalTx(context.Background(), DonateToGoalTxParams{
		GoalID:      goal.ID,
		UserID:      pgtype.Int8{Int64: user.ID, Valid: true},
		Amount:      2000000, // $20,000.00 - more than the user's $10,000 default balance
		IsAnonymous: false,
	})

	require.Error(t, err)
	require.Contains(t, err.Error(), "insufficient balance")
}

// TestDonateToGoalTx_InactiveGoal verifies that donations to inactive goals are rejected
func TestDonateToGoalTx_InactiveGoal(t *testing.T) {
	// Create a test goal and mark it as inactive
	user := createRandomUser(t, testStore)
	goal := createRandomGoal(t, testStore)
	_, err := testStore.UpdateGoal(context.Background(), UpdateGoalParams{
		ID:       goal.ID,
		IsActive: false,
	})
	require.NoError(t, err)

	// Try to donate to the inactive goal
	_, err = testStore.DonateToGoalTx(context.Background(), DonateToGoalTxParams{
		GoalID:      goal.ID,
		UserID:      pgtype.Int8{Int64: user.ID, Valid: true},
		Amount:      1000, // $10.00
		IsAnonymous: false,
	})

	require.Error(t, err)
	require.Contains(t, err.Error(), "cannot donate to inactive goal")
}

// TestDonateToGoalTx_Anonymous verifies that anonymous donations work correctly
func TestDonateToGoalTx_Anonymous(t *testing.T) {
	// Create a test goal
	goal := createRandomGoal(t, testStore)

	// Make an anonymous donation
	result, err := testStore.DonateToGoalTx(context.Background(), DonateToGoalTxParams{
		GoalID:      goal.ID,
		UserID:      pgtype.Int8{}, // No user ID for anonymous donation
		Amount:      1000,          // $10.00
		IsAnonymous: true,
	})

	require.NoError(t, err)
	require.NotEmpty(t, result.Donation)
	require.True(t, result.Donation.IsAnonymous)
	require.False(t, result.Donation.UserID.Valid) // User ID should be NULL for anonymous donations

	// Verify the goal's collected amount was updated
	updatedGoal, err := testStore.GetGoal(context.Background(), goal.ID)
	require.NoError(t, err)
	require.Equal(t, int64(1000), updatedGoal.CollectedAmount)
}
