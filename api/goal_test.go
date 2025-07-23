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

	"github.com/stretchr/testify/require"
)

func TestCreateGoalAPI(t *testing.T) {
	user, _ := randomUser(t)
	goal := randomGoal()

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
				"title":         goal.Title,
				"description":   goal.Description.String,
				"target_amount": goal.TargetAmount.Int64,
			},
			setupAuth: func(t *testing.T, request *http.Request, tokenMaker token.Maker) {
				addAuthorization(t, request, tokenMaker, authorizationTypeBearer, user.ID, time.Minute)
			},
			buildStubs: func(store *mockdb.MockStore) {
				arg := db.CreateGoalParams{
					Title: goal.Title,
					Description: pgtype.Text{
						String: goal.Description.String,
						Valid:  true,
					},
					TargetAmount: pgtype.Int8{
						Int64: goal.TargetAmount.Int64,
						Valid: true,
					},
					CollectedAmount: 0,
					IsActive:        true,
				}

				store.EXPECT().
					CreateGoal(gomock.Any(), gomock.Eq(arg)).
					Times(1).
					Return(goal, nil)
			},
			checkResponse: func(recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusCreated, recorder.Code)
				requireBodyMatchGoal(t, recorder.Body, goal)
			},
		},
		{
			name: "NoAuthorization",
			body: gin.H{
				"title":         goal.Title,
				"description":   goal.Description.String,
				"target_amount": goal.TargetAmount.Int64,
			},
			setupAuth: func(t *testing.T, request *http.Request, tokenMaker token.Maker) {
			},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().
					CreateGoal(gomock.Any(), gomock.Any()).
					Times(0)
			},
			checkResponse: func(recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusUnauthorized, recorder.Code)
			},
		},
		{
			name: "InternalError",
			body: gin.H{
				"title":         goal.Title,
				"description":   goal.Description.String,
				"target_amount": goal.TargetAmount.Int64,
			},
			setupAuth: func(t *testing.T, request *http.Request, tokenMaker token.Maker) {
				addAuthorization(t, request, tokenMaker, authorizationTypeBearer, user.ID, time.Minute)
			},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().
					CreateGoal(gomock.Any(), gomock.Any()).
					Times(1).
					Return(db.Goal{}, sql.ErrConnDone)
			},
			checkResponse: func(recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusInternalServerError, recorder.Code)
			},
		},
		{
			name: "InvalidData",
			body: gin.H{
				"title":       "", // empty title
				"description": goal.Description.String,
			},
			setupAuth: func(t *testing.T, request *http.Request, tokenMaker token.Maker) {
				addAuthorization(t, request, tokenMaker, authorizationTypeBearer, user.ID, time.Minute)
			},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().
					CreateGoal(gomock.Any(), gomock.Any()).
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

			url := "/goals"
			request, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(data))
			require.NoError(t, err)

			tc.setupAuth(t, request, server.tokenMaker)
			server.router.ServeHTTP(recorder, request)
			tc.checkResponse(recorder)
		})
	}
}

func TestGetGoalAPI(t *testing.T) {
	goal := randomGoal()

	testCases := []struct {
		name          string
		goalID        int64
		buildStubs    func(store *mockdb.MockStore)
		checkResponse func(t *testing.T, recorder *httptest.ResponseRecorder)
	}{
		{
			name:   "OK",
			goalID: goal.ID,
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().
					GetGoal(gomock.Any(), gomock.Eq(goal.ID)).
					Times(1).
					Return(goal, nil)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusOK, recorder.Code)
				requireBodyMatchGoal(t, recorder.Body, goal)
			},
		},
		{
			name:   "NotFound",
			goalID: goal.ID,
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().
					GetGoal(gomock.Any(), gomock.Eq(goal.ID)).
					Times(1).
					Return(db.Goal{}, sql.ErrNoRows)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusNotFound, recorder.Code)
			},
		},
		{
			name:   "InternalError",
			goalID: goal.ID,
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().
					GetGoal(gomock.Any(), gomock.Eq(goal.ID)).
					Times(1).
					Return(db.Goal{}, sql.ErrConnDone)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusInternalServerError, recorder.Code)
			},
		},
		{
			name:   "InvalidID",
			goalID: 0,
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().
					GetGoal(gomock.Any(), gomock.Any()).
					Times(0)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
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

			url := fmt.Sprintf("/goals/%d", tc.goalID)
			request, err := http.NewRequest(http.MethodGet, url, nil)
			require.NoError(t, err)

			server.router.ServeHTTP(recorder, request)
			tc.checkResponse(t, recorder)
		})
	}
}

func TestListGoalsAPI(t *testing.T) {
	n := 5
	goals := make([]db.Goal, n)
	for i := 0; i < n; i++ {
		goals[i] = randomGoal()
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
				arg := db.ListGoalsParams{
					Limit:  int32(n),
					Offset: 0,
				}

				store.EXPECT().
					ListGoals(gomock.Any(), gomock.Eq(arg)).
					Times(1).
					Return(goals, nil)
			},
			checkResponse: func(recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusOK, recorder.Code)
				requireBodyMatchGoals(t, recorder.Body, goals)
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
					ListGoals(gomock.Any(), gomock.Any()).
					Times(1).
					Return([]db.Goal{}, sql.ErrConnDone)
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
					ListGoals(gomock.Any(), gomock.Any()).
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
					ListGoals(gomock.Any(), gomock.Any()).
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

			url := "/goals"
			request, err := http.NewRequest(http.MethodGet, url, nil)
			require.NoError(t, err)

			// Add query parameters
			q := request.URL.Query()
			q.Add("limit", fmt.Sprintf("%d", tc.query.pageSize))
			q.Add("offset", fmt.Sprintf("%d", (tc.query.pageID-1)*tc.query.pageSize))
			request.URL.RawQuery = q.Encode()

			server.router.ServeHTTP(recorder, request)
			tc.checkResponse(recorder)
		})
	}
}

func requireBodyMatchGoal(t *testing.T, body *bytes.Buffer, goal db.Goal) {
	data, err := io.ReadAll(body)
	require.NoError(t, err)

	var gotGoal goalResponse
	err = json.Unmarshal(data, &gotGoal)
	require.NoError(t, err)

	require.Equal(t, goal.ID, gotGoal.ID)
	require.Equal(t, goal.Title, gotGoal.Title)
	require.Equal(t, goal.Description.String, gotGoal.Description)
	require.Equal(t, goal.TargetAmount.Int64, gotGoal.TargetAmount)
	require.Equal(t, goal.CollectedAmount, gotGoal.CollectedAmount)
	require.Equal(t, goal.IsActive, gotGoal.IsActive)
	require.WithinDuration(t, goal.CreatedAt.UTC(), parseTime(t, gotGoal.CreatedAt), time.Second)
}

func requireBodyMatchGoals(t *testing.T, body *bytes.Buffer, goals []db.Goal) {
	data, err := io.ReadAll(body)
	require.NoError(t, err)

	var gotGoals []goalResponse
	err = json.Unmarshal(data, &gotGoals)
	require.NoError(t, err)

	require.Len(t, gotGoals, len(goals))
	for i, goal := range goals {
		require.Equal(t, goal.ID, gotGoals[i].ID)
		require.Equal(t, goal.Title, gotGoals[i].Title)
		require.Equal(t, goal.Description.String, gotGoals[i].Description)
		require.Equal(t, goal.TargetAmount.Int64, gotGoals[i].TargetAmount)
		require.Equal(t, goal.CollectedAmount, gotGoals[i].CollectedAmount)
		require.Equal(t, goal.IsActive, gotGoals[i].IsActive)
	}
}
