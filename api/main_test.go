package api

import (
	"fmt"
	"net/http"
	"reflect"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang/mock/gomock"
	"github.com/jackc/pgx/v5/pgtype"

	db "github.com/kholodihor/charity/db/sqlc"
	"github.com/kholodihor/charity/token"
	"github.com/kholodihor/charity/util"
	"github.com/stretchr/testify/require"
)

func newTestServer(t *testing.T, store db.Store) *Server {
	config := util.Config{
		TokenSymmetricKey:   util.RandomString(32),
		AccessTokenDuration: time.Minute,
	}

	server, err := NewServer(config, store)
	require.NoError(t, err)

	return server
}



func addAuthorization(
	t *testing.T,
	request *http.Request,
	tokenMaker token.Maker,
	authorizationType string,
	userID int64,
	duration time.Duration,
) {
	token, payload, err := tokenMaker.CreateToken(userID, duration)
	require.NoError(t, err)
	require.NotEmpty(t, payload)

	authorizationHeader := fmt.Sprintf("%s %s", authorizationType, token)
	request.Header.Set(authorizationHeaderKey, authorizationHeader)
}

// EqCreateUserParams is a custom matcher for CreateUserParams
type eqCreateUserParamsMatcher struct {
	arg      db.CreateUserParams
	password string
}

func (e eqCreateUserParamsMatcher) Matches(x interface{}) bool {
	arg, ok := x.(db.CreateUserParams)
	if !ok {
		return false
	}

	err := util.CheckPassword(e.password, arg.HashedPassword.String)
	if err != nil {
		return false
	}

	e.arg.HashedPassword = arg.HashedPassword
	return reflect.DeepEqual(e.arg, arg)
}

func (e eqCreateUserParamsMatcher) String() string {
	return fmt.Sprintf("matches arg %v and password %v", e.arg, e.password)
}

func EqCreateUserParams(arg db.CreateUserParams, password string) gomock.Matcher {
	return eqCreateUserParamsMatcher{arg, password}
}

// Test data generators
func randomGoal() db.Goal {
	// Use a fixed time to avoid timezone issues in tests
	fixedTime, _ := time.Parse(time.RFC3339, "2023-01-01T12:00:00Z")
	return db.Goal{
		ID:    util.RandomInt(1, 1000),
		Title: util.RandomString(10),
		Description: pgtype.Text{
			String: util.RandomString(50),
			Valid:  true,
		},
		TargetAmount: pgtype.Int8{
			Int64: util.RandomMoney(),
			Valid: true,
		},
		CollectedAmount: util.RandomMoney(),
		IsActive:        true,
		CreatedAt:       fixedTime,
	}
}

func randomEvent() db.Event {
	// Use a fixed time to avoid timezone issues in tests
	fixedTime, _ := time.Parse(time.RFC3339, "2023-01-01T12:00:00Z")
	return db.Event{
		ID:        util.RandomInt(1, 1000),
		Name:      util.RandomString(10),
		Place:     util.RandomString(15),
		Date:      fixedTime.Add(time.Hour * 24),
		CreatedAt: fixedTime,
	}
}

func randomDonation(userID, goalID int64) db.Donation {
	// Use a fixed time to avoid timezone issues in tests
	fixedTime, _ := time.Parse(time.RFC3339, "2023-01-01T12:00:00Z")
	return db.Donation{
		ID: util.RandomInt(1, 1000),
		UserID: pgtype.Int8{
			Int64: userID,
			Valid: true,
		},
		GoalID:      goalID,
		Amount:      util.RandomMoney(),
		IsAnonymous: util.RandomBool(),
		CreatedAt:   fixedTime,
	}
}

func randomEventBooking(userID, eventID int64) db.EventBooking {
	// Use a fixed time to avoid timezone issues in tests
	fixedTime, _ := time.Parse(time.RFC3339, "2023-01-01T12:00:00Z")
	return db.EventBooking{
		ID:       util.RandomInt(1, 1000),
		UserID:   userID,
		EventID:  eventID,
		BookedAt: fixedTime,
	}
}

// parseTime parses time string from JSON response
func parseTime(t *testing.T, timeStr string) time.Time {
	parsedTime, err := time.Parse(time.RFC3339, timeStr)
	require.NoError(t, err)
	// Convert to UTC for consistent comparison
	return parsedTime.UTC()
}

func init() {
	gin.SetMode(gin.TestMode)
}
