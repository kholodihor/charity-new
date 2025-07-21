package db

import (
	"context"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/stretchr/testify/require"
)


func TestCreateGoal(t *testing.T) {
	createRandomGoal(t, testStore)
}

func TestGetGoal(t *testing.T) {
	// Create a goal first
	goal1 := createRandomGoal(t, testStore)

	goal2, err := testStore.GetGoal(context.Background(), goal1.ID)
	require.NoError(t, err)
	require.NotEmpty(t, goal2)

	require.Equal(t, goal1.ID, goal2.ID)
	require.Equal(t, goal1.Title, goal2.Title)
	require.Equal(t, goal1.Description, goal2.Description)
	require.Equal(t, goal1.TargetAmount, goal2.TargetAmount)
	require.Equal(t, goal1.CollectedAmount, goal2.CollectedAmount)
	require.Equal(t, goal1.IsActive, goal2.IsActive)
	require.WithinDuration(t, goal1.CreatedAt, goal2.CreatedAt, time.Second)
}

func TestGetGoalForUpdate(t *testing.T) {
	// Create a goal first
	goal1 := createRandomGoal(t, testStore)

	goal2, err := testStore.GetGoalForUpdate(context.Background(), goal1.ID)
	require.NoError(t, err)
	require.NotEmpty(t, goal2)

	require.Equal(t, goal1.ID, goal2.ID)
	require.Equal(t, goal1.Title, goal2.Title)
}

func TestUpdateGoal(t *testing.T) {
	// Create a goal first
	goal1 := createRandomGoal(t, testStore)

	newTargetAmount := pgtype.Int8{Int64: 200000, Valid: true} // $2000.00
	newIsActive := false

	arg := UpdateGoalParams{
		ID:           goal1.ID,
		TargetAmount: newTargetAmount,
		IsActive:     newIsActive,
	}

	goal2, err := testStore.UpdateGoal(context.Background(), arg)
	require.NoError(t, err)
	require.NotEmpty(t, goal2)

	require.Equal(t, goal1.ID, goal2.ID)
	require.Equal(t, goal1.Title, goal2.Title) // Title should not change
	require.Equal(t, goal1.Description, goal2.Description) // Description should not change
	require.Equal(t, newTargetAmount, goal2.TargetAmount)
	require.Equal(t, goal1.CollectedAmount, goal2.CollectedAmount)
	require.Equal(t, newIsActive, goal2.IsActive)
	require.WithinDuration(t, goal1.CreatedAt, goal2.CreatedAt, time.Second)
}

func TestDeleteGoal(t *testing.T) {
	// Create a goal first
	goal1 := createRandomGoal(t, testStore)

	err := testStore.DeleteGoal(context.Background(), goal1.ID)
	require.NoError(t, err)

	goal2, err := testStore.GetGoal(context.Background(), goal1.ID)
	require.Error(t, err)
	require.Empty(t, goal2)
}

func TestListGoals(t *testing.T) {
	// Create multiple goals
	var goalIDs []int64
	for i := 0; i < 10; i++ {
		goal := createRandomGoal(t, testStore)
		goalIDs = append(goalIDs, goal.ID)
	}

	arg := ListGoalsParams{
		Limit:  3,
		Offset: 2,
	}

	goals, err := testStore.ListGoals(context.Background(), arg)
	require.NoError(t, err)
	require.Len(t, goals, 3)

	for _, goal := range goals {
		require.NotEmpty(t, goal)
	}

	// Clean up
	for _, id := range goalIDs {
		err := testStore.DeleteGoal(context.Background(), id)
		require.NoError(t, err)
	}
}

func TestUpdateGoalCollectedAmount(t *testing.T) {
	// Create a goal first
	goal := createRandomGoal(t, testStore)
	initialAmount := goal.CollectedAmount
	amountToAdd := int64(5000) // $50.00

	err := testStore.UpdateGoalCollectedAmount(context.Background(), UpdateGoalCollectedAmountParams{
		ID:              goal.ID,
		CollectedAmount: initialAmount + amountToAdd,
	})
	require.NoError(t, err)

	// Verify the update
	updatedGoal, err := testStore.GetGoal(context.Background(), goal.ID)
	require.NoError(t, err)
	require.Equal(t, initialAmount+amountToAdd, updatedGoal.CollectedAmount)
}
