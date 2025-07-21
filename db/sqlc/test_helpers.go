package db

import (
	"context"
	"testing"

	"github.com/kholodihor/charity/util"
	"github.com/stretchr/testify/require"
)

// createRandomUser creates a random user for testing
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
	require.Zero(t, user.Balance)
	require.NotZero(t, user.ID)
	require.NotZero(t, user.CreatedAt)

	return user
}

// createRandomGoal creates a random goal for testing
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

// createRandomDonation creates a random donation for testing
func createRandomDonation(t *testing.T, store Store, userID int64, goalID int64) Donation {
	donorID, amount, isAnonymous := util.RandomDonationParams(userID, goalID)
	arg := CreateDonationParams{
		UserID:      donorID,
		GoalID:      goalID,
		Amount:      amount,
		IsAnonymous: isAnonymous,
	}

	donation, err := store.CreateDonation(context.Background(), arg)
	require.NoError(t, err)
	require.NotEmpty(t, donation)

	require.Equal(t, arg.UserID, donation.UserID)
	require.Equal(t, arg.GoalID, donation.GoalID)
	require.Equal(t, arg.Amount, donation.Amount)
	require.Equal(t, arg.IsAnonymous, donation.IsAnonymous)
	require.NotZero(t, donation.ID)
	require.NotZero(t, donation.CreatedAt)

	return donation
}
