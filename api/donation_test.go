package api

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang/mock/gomock"
	"github.com/jackc/pgx/v5/pgtype"
	mockdb "github.com/kholodihor/charity/db/mock"
	db "github.com/kholodihor/charity/db/sqlc"
	"github.com/kholodihor/charity/token"
	"github.com/kholodihor/charity/util"
	"github.com/stretchr/testify/require"
)

func TestDonateToGoalAPI(t *testing.T) {
	user, _ := randomUser(t)
	goal := randomGoal()
	donation := randomDonation(user.ID, goal.ID)

	testCases := []struct {
		name          string
		body          gin.H
		setupAuth     func(t *testing.T, request *http.Request, tokenMaker token.Maker)
		buildStubs    func(store *mockdb.MockStore)
		checkResponse func(recorder *httptest.ResponseRecorder)
	}{
		{
			name: "OK",
			body: gin.H{
				"goal_id":      goal.ID,
				"amount":       donation.Amount,
				"is_anonymous": donation.IsAnonymous,
			},
			setupAuth: func(t *testing.T, request *http.Request, tokenMaker token.Maker) {
				addAuthorization(t, request, tokenMaker, authorizationTypeBearer, user.ID, time.Minute)
			},
			buildStubs: func(store *mockdb.MockStore) {
				// First expect GetGoal call to validate goal exists
				store.EXPECT().
					GetGoal(gomock.Any(), gomock.Eq(goal.ID)).
					Times(1).
					Return(goal, nil)

				arg := db.DonateToGoalTxParams{
					UserID: pgtype.Int8{
						Int64: user.ID,
						Valid: true,
					},
					GoalID:      goal.ID,
					Amount:      donation.Amount,
					IsAnonymous: donation.IsAnonymous,
				}

				result := db.DonateToGoalTxResult{
					Donation: donation,
					User:     db.User{ID: user.ID, Balance: user.Balance - donation.Amount},
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
			name: "NoAuthorization",
			body: gin.H{
				"goal_id":      goal.ID,
				"amount":       donation.Amount,
				"is_anonymous": donation.IsAnonymous,
			},
			setupAuth: func(t *testing.T, request *http.Request, tokenMaker token.Maker) {
			},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().
					DonateToGoalTx(gomock.Any(), gomock.Any()).
					Times(0)
			},
			checkResponse: func(recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusUnauthorized, recorder.Code)
			},
		},
		{
			name: "InternalError",
			body: gin.H{
				"goal_id":      goal.ID,
				"amount":       donation.Amount,
				"is_anonymous": donation.IsAnonymous,
			},
			setupAuth: func(t *testing.T, request *http.Request, tokenMaker token.Maker) {
				addAuthorization(t, request, tokenMaker, authorizationTypeBearer, user.ID, time.Minute)
			},
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
			body: gin.H{
				"goal_id": goal.ID,
				"amount":  -100, // negative amount
			},
			setupAuth: func(t *testing.T, request *http.Request, tokenMaker token.Maker) {
				addAuthorization(t, request, tokenMaker, authorizationTypeBearer, user.ID, time.Minute)
			},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().
					DonateToGoalTx(gomock.Any(), gomock.Any()).
					Times(0)
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

			server := newTestServer(t, store)
			recorder := httptest.NewRecorder()

			// Marshal body data to JSON
			data, err := json.Marshal(tc.body)
			require.NoError(t, err)

			url := "/donations"
			request, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(data))
			require.NoError(t, err)

			tc.setupAuth(t, request, server.tokenMaker)
			server.router.ServeHTTP(recorder, request)
			tc.checkResponse(recorder)
		})
	}
}

func TestListDonationsAPI(t *testing.T) {
	n := 5
	donations := make([]db.Donation, n)
	for i := 0; i < n; i++ {
		user, _ := randomUser(t)
		goal := randomGoal()
		donations[i] = db.Donation{
			ID:     util.RandomInt(1, 1000),
			Amount: util.RandomMoney(),
			UserID: pgtype.Int8{
				Int64: user.ID,
				Valid: true,
			},
			GoalID:      goal.ID,
			IsAnonymous: util.RandomBool(),
			CreatedAt:   time.Now(),
		}
	}

	type Query struct {
		pageID   int
		pageSize int
	}

	testCases := []struct {
		name          string
		query         Query
		buildStubs    func(store *mockdb.MockStore)
		checkResponse func(recorder *httptest.ResponseRecorder)
	}{
		{
			name: "OK",
			query: Query{
				pageID:   1,
				pageSize: n,
			},
			buildStubs: func(store *mockdb.MockStore) {
				arg := db.ListDonationsParams{
					Limit:  int32(n),
					Offset: 0,
				}

				store.EXPECT().
					ListDonations(gomock.Any(), gomock.Eq(arg)).
					Times(1).
					Return(donations, nil)
			},
			checkResponse: func(recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusOK, recorder.Code)
				requireBodyMatchDonations(t, recorder.Body, donations)
			},
		},
		{
			name: "InternalError",
			query: Query{
				pageID:   1,
				pageSize: n,
			},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().
					ListDonations(gomock.Any(), gomock.Any()).
					Times(1).
					Return([]db.Donation{}, sql.ErrConnDone)
			},
			checkResponse: func(recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusInternalServerError, recorder.Code)
			},
		},
		{
			name: "InvalidPageID",
			query: Query{
				pageID:   -1,
				pageSize: n,
			},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().
					ListDonations(gomock.Any(), gomock.Any()).
					Times(0)
			},
			checkResponse: func(recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusBadRequest, recorder.Code)
			},
		},
		{
			name: "InvalidPageSize",
			query: Query{
				pageID:   1,
				pageSize: 100000,
			},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().
					ListDonations(gomock.Any(), gomock.Any()).
					Times(0)
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

			server := newTestServer(t, store)
			recorder := httptest.NewRecorder()

			url := "/donations"
			request, err := http.NewRequest(http.MethodGet, url, nil)
			require.NoError(t, err)

			// Add query parameters
			q := request.URL.Query()
			q.Add("page_id", fmt.Sprintf("%d", tc.query.pageID))
			q.Add("page_size", fmt.Sprintf("%d", tc.query.pageSize))
			request.URL.RawQuery = q.Encode()

			server.router.ServeHTTP(recorder, request)
			tc.checkResponse(recorder)
		})
	}
}

func TestListUserDonationsAPI(t *testing.T) {
	user, _ := randomUser(t)
	n := 5
	donations := make([]db.Donation, n)
	for i := 0; i < n; i++ {
		goal := randomGoal()
		donations[i] = db.Donation{
			ID:     util.RandomInt(1, 1000),
			Amount: util.RandomMoney(),
			UserID: pgtype.Int8{
				Int64: user.ID,
				Valid: true,
			},
			GoalID:      goal.ID,
			IsAnonymous: util.RandomBool(),
			CreatedAt:   time.Now(),
		}
	}

	type Query struct {
		pageID   int
		pageSize int
	}

	testCases := []struct {
		name          string
		userID        int64
		query         Query
		setupAuth     func(t *testing.T, request *http.Request, tokenMaker token.Maker)
		buildStubs    func(store *mockdb.MockStore)
		checkResponse func(recorder *httptest.ResponseRecorder)
	}{
		{
			name:   "OK",
			userID: user.ID,
			query: Query{
				pageID:   1,
				pageSize: n,
			},
			setupAuth: func(t *testing.T, request *http.Request, tokenMaker token.Maker) {
				addAuthorization(t, request, tokenMaker, authorizationTypeBearer, user.ID, time.Minute)
			},
			buildStubs: func(store *mockdb.MockStore) {
				arg := db.ListDonationsByUserParams{
					UserID: pgtype.Int8{
						Int64: user.ID,
						Valid: true,
					},
					Limit:  int32(n),
					Offset: 0,
				}

				store.EXPECT().
					ListDonationsByUser(gomock.Any(), gomock.Eq(arg)).
					Times(1).
					Return(donations, nil)
			},
			checkResponse: func(recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusOK, recorder.Code)
				requireBodyMatchUserDonations(t, recorder.Body, donations)
			},
		},
		{
			name:   "NoAuthorization",
			userID: user.ID,
			query: Query{
				pageID:   1,
				pageSize: n,
			},
			setupAuth: func(t *testing.T, request *http.Request, tokenMaker token.Maker) {
			},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().
					ListDonationsByUser(gomock.Any(), gomock.Any()).
					Times(0)
			},
			checkResponse: func(recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusUnauthorized, recorder.Code)
			},
		},
		{
			name:   "InternalError",
			userID: user.ID,
			query: Query{
				pageID:   1,
				pageSize: n,
			},
			setupAuth: func(t *testing.T, request *http.Request, tokenMaker token.Maker) {
				addAuthorization(t, request, tokenMaker, authorizationTypeBearer, user.ID, time.Minute)
			},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().
					ListDonationsByUser(gomock.Any(), gomock.Any()).
					Times(1).
					Return([]db.Donation{}, sql.ErrConnDone)
			},
			checkResponse: func(recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusInternalServerError, recorder.Code)
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

			server := newTestServer(t, store)
			recorder := httptest.NewRecorder()

			url := fmt.Sprintf("/users/%d/donations", tc.userID)
			request, err := http.NewRequest(http.MethodGet, url, nil)
			require.NoError(t, err)

			// Add query parameters
			q := request.URL.Query()
			q.Add("page_id", fmt.Sprintf("%d", tc.query.pageID))
			q.Add("page_size", fmt.Sprintf("%d", tc.query.pageSize))
			request.URL.RawQuery = q.Encode()

			tc.setupAuth(t, request, server.tokenMaker)
			server.router.ServeHTTP(recorder, request)
			tc.checkResponse(recorder)
		})
	}
}

func requireBodyMatchDonation(t *testing.T, body *bytes.Buffer, donation db.Donation) {
	data, err := io.ReadAll(body)
	require.NoError(t, err)

	var gotDonation donationResponse
	err = json.Unmarshal(data, &gotDonation)
	require.NoError(t, err)

	require.Equal(t, donation.ID, gotDonation.ID)
	require.Equal(t, donation.Amount, gotDonation.Amount)
	require.Equal(t, donation.GoalID, gotDonation.GoalID)
	require.Equal(t, donation.IsAnonymous, gotDonation.IsAnonymous)
	require.WithinDuration(t, donation.CreatedAt, parseTime(t, gotDonation.CreatedAt), time.Second)
}

func requireBodyMatchDonations(t *testing.T, body *bytes.Buffer, donations []db.Donation) {
	data, err := io.ReadAll(body)
	require.NoError(t, err)

	var gotDonations []donationResponse
	err = json.Unmarshal(data, &gotDonations)
	require.NoError(t, err)

	require.Len(t, gotDonations, len(donations))
	for i, donation := range donations {
		require.Equal(t, donation.ID, gotDonations[i].ID)
		require.Equal(t, donation.Amount, gotDonations[i].Amount)
		require.Equal(t, donation.GoalID, gotDonations[i].GoalID)
		require.Equal(t, donation.IsAnonymous, gotDonations[i].IsAnonymous)
	}
}

func requireBodyMatchUserDonations(t *testing.T, body *bytes.Buffer, donations []db.Donation) {
	data, err := io.ReadAll(body)
	require.NoError(t, err)

	var gotDonations []donationResponse
	err = json.Unmarshal(data, &gotDonations)
	require.NoError(t, err)

	require.Len(t, gotDonations, len(donations))
	for i, donation := range donations {
		require.Equal(t, donation.ID, gotDonations[i].ID)
		require.Equal(t, donation.Amount, gotDonations[i].Amount)
		require.Equal(t, donation.GoalID, gotDonations[i].GoalID)
		require.Equal(t, donation.IsAnonymous, gotDonations[i].IsAnonymous)
	}
}
