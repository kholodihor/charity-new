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

func TestCreateEventAPI(t *testing.T) {
	user, _ := randomUser(t)
	event := randomEvent()

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
				"name":  event.Name,
				"place": event.Place,
				"date":  event.Date.Format("2006-01-02T15:04:05Z"),
			},
			setupAuth: func(t *testing.T, request *http.Request, tokenMaker token.Maker) {
				addAuthorization(t, request, tokenMaker, authorizationTypeBearer, user.ID, time.Minute)
			},
			buildStubs: func(store *mockdb.MockStore) {
				arg := db.CreateEventParams{
					Name:  event.Name,
					Place: event.Place,
					Date:  event.Date,
				}

				store.EXPECT().
					CreateEvent(gomock.Any(), gomock.Eq(arg)).
					Times(1).
					Return(event, nil)
			},
			checkResponse: func(recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusCreated, recorder.Code)
				requireBodyMatchEvent(t, recorder.Body, event)
			},
		},
		{
			name: "NoAuthorization",
			body: gin.H{
				"name":  event.Name,
				"place": event.Place,
				"date":  event.Date.Format("2006-01-02T15:04:05Z"),
			},
			setupAuth: func(t *testing.T, request *http.Request, tokenMaker token.Maker) {
			},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().
					CreateEvent(gomock.Any(), gomock.Any()).
					Times(0)
			},
			checkResponse: func(recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusUnauthorized, recorder.Code)
			},
		},
		{
			name: "InternalError",
			body: gin.H{
				"name":  event.Name,
				"place": event.Place,
				"date":  event.Date.Format("2006-01-02T15:04:05Z"),
			},
			setupAuth: func(t *testing.T, request *http.Request, tokenMaker token.Maker) {
				addAuthorization(t, request, tokenMaker, authorizationTypeBearer, user.ID, time.Minute)
			},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().
					CreateEvent(gomock.Any(), gomock.Any()).
					Times(1).
					Return(db.Event{}, sql.ErrConnDone)
			},
			checkResponse: func(recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusInternalServerError, recorder.Code)
			},
		},
		{
			name: "InvalidData",
			body: gin.H{
				"name":  "", // empty name
				"place": event.Place,
				"date":  event.Date.Format("2006-01-02T15:04:05Z"),
			},
			setupAuth: func(t *testing.T, request *http.Request, tokenMaker token.Maker) {
				addAuthorization(t, request, tokenMaker, authorizationTypeBearer, user.ID, time.Minute)
			},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().
					CreateEvent(gomock.Any(), gomock.Any()).
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

			url := "/events"
			request, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(data))
			require.NoError(t, err)

			tc.setupAuth(t, request, server.tokenMaker)
			server.router.ServeHTTP(recorder, request)
			tc.checkResponse(recorder)
		})
	}
}

func TestBookEventAPI(t *testing.T) {
	user, _ := randomUser(t)
	event := randomEvent()
	booking := randomEventBooking(user.ID, event.ID)

	testCases := []struct {
		name          string
		eventID       int64
		setupAuth     func(t *testing.T, request *http.Request, tokenMaker token.Maker)
		buildStubs    func(store *mockdb.MockStore)
		checkResponse func(recorder *httptest.ResponseRecorder)
	}{
		{
			name:    "OK",
			eventID: event.ID,
			setupAuth: func(t *testing.T, request *http.Request, tokenMaker token.Maker) {
				addAuthorization(t, request, tokenMaker, authorizationTypeBearer, user.ID, time.Minute)
			},
			buildStubs: func(store *mockdb.MockStore) {
				arg := db.BookEventParams{
					UserID:  user.ID,
					EventID: event.ID,
				}

				store.EXPECT().
					BookEvent(gomock.Any(), gomock.Eq(arg)).
					Times(1).
					Return(booking, nil)
			},
			checkResponse: func(recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusCreated, recorder.Code)
			},
		},
		{
			name:    "NoAuthorization",
			eventID: event.ID,
			setupAuth: func(t *testing.T, request *http.Request, tokenMaker token.Maker) {
			},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().
					BookEvent(gomock.Any(), gomock.Any()).
					Times(0)
			},
			checkResponse: func(recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusUnauthorized, recorder.Code)
			},
		},
		{
			name:    "InternalError",
			eventID: event.ID,
			setupAuth: func(t *testing.T, request *http.Request, tokenMaker token.Maker) {
				addAuthorization(t, request, tokenMaker, authorizationTypeBearer, user.ID, time.Minute)
			},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().
					BookEvent(gomock.Any(), gomock.Any()).
					Times(1).
					Return(db.EventBooking{}, sql.ErrConnDone)
			},
			checkResponse: func(recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusInternalServerError, recorder.Code)
			},
		},
		{
			name:    "InvalidID",
			eventID: 0,
			setupAuth: func(t *testing.T, request *http.Request, tokenMaker token.Maker) {
				addAuthorization(t, request, tokenMaker, authorizationTypeBearer, user.ID, time.Minute)
			},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().
					BookEvent(gomock.Any(), gomock.Any()).
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

			url := fmt.Sprintf("/events/%d/book", tc.eventID)
			request, err := http.NewRequest(http.MethodPost, url, nil)
			require.NoError(t, err)

			tc.setupAuth(t, request, server.tokenMaker)
			server.router.ServeHTTP(recorder, request)
			tc.checkResponse(recorder)
		})
	}
}

func TestListEventBookingsAPI(t *testing.T) {
	event := randomEvent()
	n := 5
	bookings := make([]db.ListEventBookingsRow, n)
	for i := 0; i < n; i++ {
		user, _ := randomUser(t)
		bookings[i] = db.ListEventBookingsRow{
			ID:       util.RandomInt(1, 1000),
			UserID:   user.ID,
			EventID:  event.ID,
			BookedAt: time.Now(),
			UserName: pgtype.Text{
				String: user.Name.String,
				Valid:  true,
			},
			UserEmail: user.Email,
		}
	}

	type Query struct {
		pageID   int
		pageSize int
	}

	testCases := []struct {
		name          string
		eventID       int64
		query         Query
		buildStubs    func(store *mockdb.MockStore)
		checkResponse func(recorder *httptest.ResponseRecorder)
	}{
		{
			name:    "OK",
			eventID: event.ID,
			query: Query{
				pageID:   1,
				pageSize: n,
			},
			buildStubs: func(store *mockdb.MockStore) {
				arg := db.ListEventBookingsParams{
					EventID: event.ID,
					Limit:   int32(n),
					Offset:  0,
				}

				store.EXPECT().
					ListEventBookings(gomock.Any(), gomock.Eq(arg)).
					Times(1).
					Return(bookings, nil)
			},
			checkResponse: func(recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusOK, recorder.Code)
				requireBodyMatchEventBookings(t, recorder.Body, bookings)
			},
		},
		{
			name:    "InternalError",
			eventID: event.ID,
			query: Query{
				pageID:   1,
				pageSize: n,
			},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().
					ListEventBookings(gomock.Any(), gomock.Any()).
					Times(1).
					Return([]db.ListEventBookingsRow{}, sql.ErrConnDone)
			},
			checkResponse: func(recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusInternalServerError, recorder.Code)
			},
		},
		{
			name:    "InvalidID",
			eventID: 0,
			query: Query{
				pageID:   1,
				pageSize: n,
			},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().
					ListEventBookings(gomock.Any(), gomock.Any()).
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

			url := fmt.Sprintf("/events/%d/bookings", tc.eventID)
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

func requireBodyMatchEvent(t *testing.T, body *bytes.Buffer, event db.Event) {
	data, err := io.ReadAll(body)
	require.NoError(t, err)

	var gotEvent eventResponse
	err = json.Unmarshal(data, &gotEvent)
	require.NoError(t, err)

	require.Equal(t, event.ID, gotEvent.ID)
	require.Equal(t, event.Name, gotEvent.Name)
	require.Equal(t, event.Place, gotEvent.Place)
	require.WithinDuration(t, event.Date, parseTime(t, gotEvent.Date), time.Second)
	require.WithinDuration(t, event.CreatedAt, parseTime(t, gotEvent.CreatedAt), time.Second)
}

func requireBodyMatchEventBookings(t *testing.T, body *bytes.Buffer, bookings []db.ListEventBookingsRow) {
	data, err := io.ReadAll(body)
	require.NoError(t, err)

	var gotBookings []eventBookingWithUserResponse
	err = json.Unmarshal(data, &gotBookings)
	require.NoError(t, err)

	require.Len(t, gotBookings, len(bookings))
	for i, booking := range bookings {
		require.Equal(t, booking.ID, gotBookings[i].ID)
		require.Equal(t, booking.UserID, gotBookings[i].UserID)
		require.Equal(t, booking.EventID, gotBookings[i].EventID)
		require.Equal(t, booking.UserEmail, gotBookings[i].UserEmail)
		if booking.UserName.Valid {
			require.Equal(t, booking.UserName.String, gotBookings[i].UserName)
		}
	}
}
