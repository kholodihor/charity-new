package db

import (
	"context"
	"testing"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/stretchr/testify/require"
)

func TestDonateToGoalTx(t *testing.T) {
	t.Run("Donate as authenticated user", func(t *testing.T) {
		testDonateToGoal(t, testStore, false)
	})

	t.Run("Donate anonymously", func(t *testing.T) {
		testDonateToGoal(t, testStore, true)
	})

	t.Run("Fails when goal doesn't exist", func(t *testing.T) {
		testDonateToGoalFailure(t, testStore, func(store Store, goalID int64, userID pgtype.Int8) error {
			_, err := store.DonateToGoalTx(context.Background(), DonateToGoalTxParams{
				GoalID:      999, // Non-existent goal ID
				UserID:      userID,
				Amount:      100,
				IsAnonymous: !userID.Valid,
			})
			return err
		})
	})

	t.Run("Fails with negative amount", func(t *testing.T) {
		testDonateToGoalFailure(t, testStore, func(store Store, goalID int64, userID pgtype.Int8) error {
			_, err := store.DonateToGoalTx(context.Background(), DonateToGoalTxParams{
				GoalID:      goalID,
				UserID:      userID,
				Amount:      -100, // Negative amount should fail
				IsAnonymous: !userID.Valid,
			})
			return err
		})
	})

	t.Run("Fails when updating invalid goal", func(t *testing.T) {
		// Create a goal but delete it immediately to simulate a race condition
		goal := createRandomGoal(t, testStore)
		err := testStore.DeleteGoal(context.Background(), goal.ID)
		require.NoError(t, err)

		testDonateToGoalFailure(t, testStore, func(store Store, _ int64, _ pgtype.Int8) error {
			_, err := store.DonateToGoalTx(context.Background(), DonateToGoalTxParams{
				GoalID:      goal.ID, // This goal was deleted
				UserID:      pgtype.Int8{Valid: false}, // Anonymous user
				Amount:      100,
				IsAnonymous: true,
			})
			return err
		})
	})
}

// testDonateToGoalFailure tests that a donation transaction fails and rolls back properly
func testDonateToGoalFailure(t *testing.T, store Store, testFunc func(Store, int64, pgtype.Int8) error) {
	// Create a new goal
	goal := createRandomGoal(t, store)
	initialAmount := goal.CollectedAmount

	// For anonymous donations, we don't need a user
	var user User
	var userID pgtype.Int8
	if !goal.IsActive {
		userID = pgtype.Int8{Valid: false}
	} else {
		user = createRandomUser(t, store)
		userID = pgtype.Int8{Int64: user.ID, Valid: true}
	}

	// Execute the test function which should fail
	err := testFunc(testStore, goal.ID, userID)
	require.Error(t, err, "Expected an error but got none")

	// Verify goal's collected amount didn't change
	updatedGoal, err := testStore.GetGoal(context.Background(), goal.ID)
	require.NoError(t, err)
	require.Equal(t, initialAmount, updatedGoal.CollectedAmount, "Goal's collected amount should not change")

	// Clean up test data
	err = testStore.DeleteGoal(context.Background(), goal.ID)
	require.NoError(t, err, "Failed to clean up test goal")

	if userID.Valid {
		err = testStore.DeleteUser(context.Background(), userID.Int64)
		require.NoError(t, err, "Failed to clean up test user")
	}
}

func testDonateToGoal(t *testing.T, store Store, anonymous bool) {
	// Create a new goal
	goal := createRandomGoal(t, store)
	initialAmount := goal.CollectedAmount

	// Ensure the goal is active
	if !goal.IsActive {
		goal, _ = testStore.UpdateGoal(context.Background(), UpdateGoalParams{
			ID:           goal.ID,
			IsActive:     true,
			TargetAmount: goal.TargetAmount,
		})
	}

	// For anonymous donations, we don't need a user
	var user User
	var userID pgtype.Int8
	if !anonymous {
		// Create a user for non-anonymous donations
		user = createRandomUser(t, store)
		userID = pgtype.Int8{Int64: user.ID, Valid: true}
	} else {
		userID = pgtype.Int8{Valid: false}
	}

	amount := int64(1000) // $10.00 in cents

	// Execute the transaction
	result, err := store.DonateToGoalTx(context.Background(), DonateToGoalTxParams{
		GoalID:      goal.ID,
		UserID:      userID,
		Amount:      amount,
		IsAnonymous: anonymous,
	})

	// Check for errors
	require.NoError(t, err)
	require.NotEmpty(t, result)

	// Check the donation
	require.Equal(t, goal.ID, result.Donation.GoalID)
	if !anonymous {
		require.Equal(t, userID, result.Donation.UserID, "UserID should be set for non-anonymous donations")
	} else {
		require.False(t, result.Donation.UserID.Valid, "UserID should be NULL for anonymous donations")
	}
	require.Equal(t, amount, result.Donation.Amount)
	require.Equal(t, anonymous, result.Donation.IsAnonymous, "IsAnonymous should match the input parameter")
	require.NotZero(t, result.Donation.ID)
	require.NotZero(t, result.Donation.CreatedAt)

	// Check the updated goal
	require.Equal(t, goal.ID, result.Goal.ID)
	require.Equal(t, goal.Title, result.Goal.Title)
	require.Equal(t, goal.Description, result.Goal.Description)
	require.Equal(t, goal.TargetAmount, result.Goal.TargetAmount)
	require.Equal(t, initialAmount+amount, result.Goal.CollectedAmount, "Goal collected amount should be updated correctly")

	// Note: We skip cleanup since there's no DeleteDonation method
	// and foreign key constraints prevent deleting goals/users with donations.
	// The test database will be cleaned up between test runs.
}
