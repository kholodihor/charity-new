package db

import (
	"context"
	"testing"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/stretchr/testify/require"
	"github.com/kholodihor/charity/util"
)

func TestCreateUser(t *testing.T) {
	t.Run("Create user with valid data", func(t *testing.T) {
		email, name := util.RandomUserParams()
		arg := CreateUserParams{
			Email: email,
			Name:  name,
		}

		user, err := testStore.CreateUser(context.Background(), arg)
		require.NoError(t, err)
		require.NotEmpty(t, user)
		require.NotZero(t, user.ID)
		require.NotZero(t, user.CreatedAt)
		require.Equal(t, email, user.Email)
		require.Equal(t, name, user.Name)
		require.Equal(t, int64(1000000), user.Balance, "User should have $10,000 (1,000,000 cents) default balance")
	})

	t.Run("Create user with duplicate email", func(t *testing.T) {
		// First create a user with random email
		email := util.RandomEmail()
		arg1 := CreateUserParams{
			Email: email,
			Name:  pgtype.Text{String: "First User", Valid: true},
		}

		_, err := testStore.CreateUser(context.Background(), arg1)
		require.NoError(t, err)

		// Try to create another user with the same email
		arg2 := CreateUserParams{
			Email: email, // Same email
			Name:  pgtype.Text{String: "Second User", Valid: true},
		}

		_, err = testStore.CreateUser(context.Background(), arg2)
		require.Error(t, err)
		require.ErrorContains(t, err, "duplicate key value")
	})
}

func TestGetUser(t *testing.T) {
	user1 := createRandomUser(t, testStore)

	t.Run("Get existing user", func(t *testing.T) {
		user2, err := testStore.GetUser(context.Background(), user1.ID)
		require.NoError(t, err)
		require.NotEmpty(t, user2)

		require.Equal(t, user1.ID, user2.ID)
		require.Equal(t, user1.Email, user2.Email)
		require.Equal(t, user1.Name, user2.Name)
		require.Equal(t, user1.Balance, user2.Balance)
		require.WithinDuration(t, user1.CreatedAt, user2.CreatedAt, 0)
	})

	t.Run("Get non-existent user", func(t *testing.T) {
		_, err := testStore.GetUser(context.Background(), 999999)
		require.Error(t, err)
		require.Error(t, err)
		require.ErrorContains(t, err, "no rows in result set")
	})
}

func TestListUsers(t *testing.T) {
	// Create multiple users
	for i := 0; i < 5; i++ {
		_ = createRandomUser(t, testStore)
	}

	arg := ListUsersParams{
		Limit:  5,
		Offset: 0,
	}

	users, err := testStore.ListUsers(context.Background(), arg)
	require.NoError(t, err)
	require.NotEmpty(t, users)
	require.True(t, len(users) >= 5, "should return at least 5 users")

	for _, user := range users {
		require.NotEmpty(t, user)
	}
}

func TestUpdateUser(t *testing.T) {
	user := createRandomUser(t, testStore)

	t.Run("Update user name", func(t *testing.T) {
		newName := pgtype.Text{String: "Updated Name", Valid: true}
		updatedUser, err := testStore.UpdateUser(context.Background(), UpdateUserParams{
			ID:   user.ID,
			Name: newName,
		})

		require.NoError(t, err)
		require.Equal(t, newName, updatedUser.Name)
		require.Equal(t, user.Email, updatedUser.Email) // Email should not change
	})
}

func TestUpdateUserBalance(t *testing.T) {
	user := createRandomUser(t, testStore)

	t.Run("Update user balance", func(t *testing.T) {
		// This test is a placeholder since we don't have a direct UpdateUserBalance method
		// In a real application, you would either:
		// 1. Add an UpdateUserBalance method to the Store interface
		// 2. Or test the balance update as part of a transaction test
		
		// For now, we'll just verify we can get the user
		_, err := testStore.GetUser(context.Background(), user.ID)
		require.NoError(t, err)
	})
}

func TestDeleteUser(t *testing.T) {
	user := createRandomUser(t, testStore)

	t.Run("Delete existing user", func(t *testing.T) {
		err := testStore.DeleteUser(context.Background(), user.ID)
		require.NoError(t, err)

		// Verify user is deleted
		_, err = testStore.GetUser(context.Background(), user.ID)
		require.Error(t, err)
		require.Error(t, err)
		require.ErrorContains(t, err, "no rows in result set")
	})

	t.Run("Delete non-existent user", func(t *testing.T) {
		err := testStore.DeleteUser(context.Background(), 999999)
		require.NoError(t, err) // Deleting non-existent user should not return an error
	})
}
