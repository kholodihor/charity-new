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
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	mockdb "github.com/kholodihor/charity/db/mock"
	db "github.com/kholodihor/charity/db/sqlc"
)

func TestRefreshTokenAPI(t *testing.T) {
	user, _ := randomUser(t)

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	store := mockdb.NewMockStore(ctrl)
	server := newTestServer(t, store)

	// Create a valid refresh token
	refreshToken, refreshPayload, err := server.tokenMaker.CreateRefreshToken(user.ID, time.Hour)
	require.NoError(t, err)

	// Test successful refresh
	t.Run("OK", func(t *testing.T) {
		dbRefreshToken := db.RefreshToken{
			ID:        uuid.New(),
			UserID:    user.ID,
			TokenID:   refreshPayload.ID,
			ExpiresAt: refreshPayload.ExpiredAt,
			CreatedAt: time.Now(),
		}

		store.EXPECT().
			GetRefreshToken(gomock.Any(), refreshPayload.ID).
			Times(1).
			Return(dbRefreshToken, nil)

		store.EXPECT().
			RevokeRefreshToken(gomock.Any(), refreshPayload.ID).
			Times(1).
			Return(nil)

		store.EXPECT().
			CreateRefreshToken(gomock.Any(), gomock.Any()).
			Times(1).
			Return(dbRefreshToken, nil)

		recorder := httptest.NewRecorder()
		body := gin.H{
			"refresh_token": refreshToken,
		}
		data, err := json.Marshal(body)
		require.NoError(t, err)

		url := "/auth/refresh"
		request, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(data))
		require.NoError(t, err)

		server.router.ServeHTTP(recorder, request)
		require.Equal(t, http.StatusOK, recorder.Code)

		var response refreshTokenResponse
		err = json.Unmarshal(recorder.Body.Bytes(), &response)
		require.NoError(t, err)
		require.NotEmpty(t, response.AccessToken)
		require.NotEmpty(t, response.RefreshToken)
		require.NotZero(t, response.AccessTokenExpiresAt)
		require.NotZero(t, response.RefreshTokenExpiresAt)
	})
}
