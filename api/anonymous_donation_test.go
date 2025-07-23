package api

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang/mock/gomock"
	"github.com/jackc/pgx/v5/pgtype"
	mockdb "github.com/kholodihor/charity/db/mock"
	db "github.com/kholodihor/charity/db/sqlc"
	"github.com/kholodihor/charity/util"
	"github.com/stretchr/testify/require"
)

func TestCreateAnonymousDonationAPI(t *testing.T) {
	goal := randomGoal()
	donation := randomDonation(0, goal.ID) // Use 0 for userID, will be overridden
	donation.UserID = pgtype.Int8{Valid: false} // Anonymous donation
	donation.IsAnonymous = true
	donation.GoalID = goal.ID

	testCases := []struct {
		name          string
		buildStubs    func(store *mockdb.MockStore)
		checkResponse func(recorder *httptest.ResponseRecorder)
	}{
		{
			name: "OK",
			buildStubs: func(store *mockdb.MockStore) {
				// First expect GetGoal call to validate goal exists
				store.EXPECT().
					GetGoal(gomock.Any(), gomock.Eq(goal.ID)).
					Times(1).
					Return(goal, nil)

				arg := db.DonateToGoalTxParams{
					UserID: pgtype.Int8{
						Valid: false, // Anonymous donation
					},
					GoalID:      goal.ID,
					Amount:      donation.Amount,
					IsAnonymous: true,
				}

				result := db.DonateToGoalTxResult{
					Donation: donation,
					Goal:     db.Goal{ID: goal.ID, CollectedAmount: goal.CollectedAmount + donation.Amount},
				}

				store.EXPECT().
					DonateToGoalTx(gomock.Any(), gomock.Eq(arg)).
					Times(1).
					Return(result, nil)
			},
			checkResponse: func(recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusCreated, recorder.Code)
				requireBodyMatchDonation(t, recorder.Body, donation)
			},
		},
		{
			name: "GoalNotFound",
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().
					GetGoal(gomock.Any(), gomock.Eq(goal.ID)).
					Times(1).
					Return(db.Goal{}, sql.ErrNoRows)
			},
			checkResponse: func(recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusNotFound, recorder.Code)
			},
		},
		{
			name: "InternalError",
			buildStubs: func(store *mockdb.MockStore) {
				// First expect GetGoal call to validate goal exists
				store.EXPECT().
					GetGoal(gomock.Any(), gomock.Eq(goal.ID)).
					Times(1).
					Return(goal, nil)

				store.EXPECT().
					DonateToGoalTx(gomock.Any(), gomock.Any()).
					Times(1).
					Return(db.DonateToGoalTxResult{}, sql.ErrConnDone)
			},
			checkResponse: func(recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusInternalServerError, recorder.Code)
			},
		},
		{
			name: "InvalidData",
			buildStubs: func(store *mockdb.MockStore) {
				// No expectations since request validation should fail
			},
			checkResponse: func(recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusBadRequest, recorder.Code)
			},
		},
	}

	for i := range testCases {
		tc := testCases[i]

		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			store := mockdb.NewMockStore(ctrl)
			tc.buildStubs(store)

			// Create server with higher rate limit for testing
			config := util.Config{
				TokenSymmetricKey:     util.RandomString(32),
				AccessTokenDuration:   time.Minute,
				MaxAnonymousDonation:  1000000,  // $10,000
				MaxRegisteredDonation: 5000000,  // $50,000
				RateLimitPerMinute:    100,      // High limit for testing
			}
			server, err := NewServer(config, store)
			require.NoError(t, err)
			recorder := httptest.NewRecorder()

			// Prepare request body
			var data gin.H
			if tc.name == "InvalidData" {
				data = gin.H{
					"goal_id": goal.ID,
					"amount":  -100, // Invalid negative amount
				}
			} else {
				data = gin.H{
					"goal_id": goal.ID,
					"amount":  donation.Amount,
				}
			}

			body, err := json.Marshal(data)
			require.NoError(t, err)

			url := "/donations/anonymous"
			request, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(body))
			require.NoError(t, err)

			server.router.ServeHTTP(recorder, request)
			tc.checkResponse(recorder)
		})
	}
}
