package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang/mock/gomock"
	mockdb "github.com/kholodihor/charity/db/mock"
	db "github.com/kholodihor/charity/db/sqlc"
	"github.com/kholodihor/charity/token"
	"github.com/kholodihor/charity/util"
	"github.com/stretchr/testify/require"
)

func TestDonationLimits(t *testing.T) {
	user, _ := randomUser(t)
	goal := randomGoal()

	testCases := []struct {
		name          string
		donationType  string // "registered" or "anonymous"
		amount        int64
		setupAuth     func(t *testing.T, request *http.Request, tokenMaker token.Maker)
		buildStubs    func(store *mockdb.MockStore)
		checkResponse func(recorder *httptest.ResponseRecorder)
	}{
		{
			name:         "RegisteredDonationWithinLimit",
			donationType: "registered",
			amount:       100000, // $1,000 - within limit
			setupAuth: func(t *testing.T, request *http.Request, tokenMaker token.Maker) {
				addAuthorization(t, request, tokenMaker, authorizationTypeBearer, user.ID, time.Minute)
			},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().
					GetGoal(gomock.Any(), gomock.Eq(goal.ID)).
					Times(1).
					Return(goal, nil)

				store.EXPECT().
					DonateToGoalTx(gomock.Any(), gomock.Any()).
					Times(1).
					Return(db.DonateToGoalTxResult{}, nil)
			},
			checkResponse: func(recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusCreated, recorder.Code)
			},
		},
		{
			name:         "RegisteredDonationExceedsLimit",
			donationType: "registered",
			amount:       6000000, // $60,000 - exceeds $50,000 limit
			setupAuth: func(t *testing.T, request *http.Request, tokenMaker token.Maker) {
				addAuthorization(t, request, tokenMaker, authorizationTypeBearer, user.ID, time.Minute)
			},
			buildStubs: func(store *mockdb.MockStore) {
				// No database calls expected since validation should fail first
			},
			checkResponse: func(recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusBadRequest, recorder.Code)
				
				var response map[string]interface{}
				err := json.Unmarshal(recorder.Body.Bytes(), &response)
				require.NoError(t, err)
				require.Contains(t, response["error"], "exceeds maximum limit")
				require.Equal(t, float64(5000000), response["max_amount"])
			},
		},
		{
			name:         "AnonymousDonationWithinLimit",
			donationType: "anonymous",
			amount:       500000, // $5,000 - within limit
			setupAuth:    func(t *testing.T, request *http.Request, tokenMaker token.Maker) {
				// No auth for anonymous donations
			},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().
					GetGoal(gomock.Any(), gomock.Eq(goal.ID)).
					Times(1).
					Return(goal, nil)

				store.EXPECT().
					DonateToGoalTx(gomock.Any(), gomock.Any()).
					Times(1).
					Return(db.DonateToGoalTxResult{}, nil)
			},
			checkResponse: func(recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusCreated, recorder.Code)
			},
		},
		{
			name:         "AnonymousDonationExceedsLimit",
			donationType: "anonymous",
			amount:       1500000, // $15,000 - exceeds $10,000 limit
			setupAuth:    func(t *testing.T, request *http.Request, tokenMaker token.Maker) {
				// No auth for anonymous donations
			},
			buildStubs: func(store *mockdb.MockStore) {
				// No database calls expected since validation should fail first
			},
			checkResponse: func(recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusBadRequest, recorder.Code)
				
				var response map[string]interface{}
				err := json.Unmarshal(recorder.Body.Bytes(), &response)
				require.NoError(t, err)
				require.Contains(t, response["error"], "exceeds maximum limit")
				require.Equal(t, float64(1000000), response["max_amount"])
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
			data := gin.H{
				"goal_id": goal.ID,
				"amount":  tc.amount,
			}

			// Add is_anonymous for registered donations
			if tc.donationType == "registered" {
				data["is_anonymous"] = false
			}

			body, err := json.Marshal(data)
			require.NoError(t, err)

			// Choose endpoint based on donation type
			var url string
			if tc.donationType == "anonymous" {
				url = "/donations/anonymous"
			} else {
				url = "/donations"
			}

			request, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(body))
			require.NoError(t, err)

			tc.setupAuth(t, request, server.tokenMaker)

			server.router.ServeHTTP(recorder, request)
			tc.checkResponse(recorder)
		})
	}
}

func TestRateLimiting(t *testing.T) {
	_, _ = randomUser(t) // Not used in this test
	goal := randomGoal()

	// Create a server with very low rate limit for testing
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	store := mockdb.NewMockStore(ctrl)
	config := util.Config{
		TokenSymmetricKey:     util.RandomString(32),
		AccessTokenDuration:   time.Minute,
		MaxAnonymousDonation:  1000000,
		MaxRegisteredDonation: 5000000,
		RateLimitPerMinute:    2, // Very low limit for testing
	}

	server, err := NewServer(config, store)
	require.NoError(t, err)

	// Mock successful donation calls
	store.EXPECT().
		GetGoal(gomock.Any(), gomock.Eq(goal.ID)).
		AnyTimes().
		Return(goal, nil)

	store.EXPECT().
		DonateToGoalTx(gomock.Any(), gomock.Any()).
		AnyTimes().
		Return(db.DonateToGoalTxResult{}, nil)

	// Prepare request body
	data := gin.H{
		"goal_id": goal.ID,
		"amount":  50000,
	}

	body, err := json.Marshal(data)
	require.NoError(t, err)

	// First request should succeed
	recorder1 := httptest.NewRecorder()
	request1, err := http.NewRequest(http.MethodPost, "/donations/anonymous", bytes.NewReader(body))
	require.NoError(t, err)

	server.router.ServeHTTP(recorder1, request1)
	require.Equal(t, http.StatusCreated, recorder1.Code)

	// Second request should succeed
	recorder2 := httptest.NewRecorder()
	request2, err := http.NewRequest(http.MethodPost, "/donations/anonymous", bytes.NewReader(body))
	require.NoError(t, err)

	server.router.ServeHTTP(recorder2, request2)
	require.Equal(t, http.StatusCreated, recorder2.Code)

	// Third request should be rate limited
	recorder3 := httptest.NewRecorder()
	request3, err := http.NewRequest(http.MethodPost, "/donations/anonymous", bytes.NewReader(body))
	require.NoError(t, err)

	server.router.ServeHTTP(recorder3, request3)
	require.Equal(t, http.StatusTooManyRequests, recorder3.Code)

	var response map[string]interface{}
	err = json.Unmarshal(recorder3.Body.Bytes(), &response)
	require.NoError(t, err)
	require.Equal(t, "rate limit exceeded", response["error"])
}
