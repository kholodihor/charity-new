package api

import (
	"database/sql"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgtype"
	db "github.com/kholodihor/charity/db/sqlc"
	"github.com/kholodihor/charity/token"
)

type createDonationRequest struct {
	GoalID      int64 `json:"goal_id" binding:"required"`
	Amount      int64 `json:"amount" binding:"required,min=1"`
	IsAnonymous bool  `json:"is_anonymous"`
}

type donationResponse struct {
	ID          int64  `json:"id"`
	UserID      *int64 `json:"user_id,omitempty"`
	GoalID      int64  `json:"goal_id"`
	Amount      int64  `json:"amount"`
	IsAnonymous bool   `json:"is_anonymous"`
	CreatedAt   string `json:"created_at"`
}

func newDonationResponse(donation db.Donation) donationResponse {
	response := donationResponse{
		ID:          donation.ID,
		GoalID:      donation.GoalID,
		Amount:      donation.Amount,
		IsAnonymous: donation.IsAnonymous,
		CreatedAt:   donation.CreatedAt.Format("2006-01-02T15:04:05Z"),
	}

	if donation.UserID.Valid && !donation.IsAnonymous {
		response.UserID = &donation.UserID.Int64
	}

	return response
}

// POST /donations
func (server *Server) createDonation(ctx *gin.Context) {
	var req createDonationRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	// Get user from token
	authPayload := ctx.MustGet(authorizationPayloadKey).(*token.Payload)

	// Check if goal exists
	_, err := server.store.GetGoal(ctx, req.GoalID)
	if err != nil {
		if err == sql.ErrNoRows {
			ctx.JSON(http.StatusNotFound, gin.H{"error": "goal not found"})
			return
		}
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	// Create donation using transaction
	arg := db.DonateToGoalTxParams{
		GoalID: req.GoalID,
		UserID: pgtype.Int8{
			Int64: authPayload.UserID,
			Valid: true,
		},
		Amount:      req.Amount,
		IsAnonymous: req.IsAnonymous,
	}

	result, err := server.store.DonateToGoalTx(ctx, arg)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	ctx.JSON(http.StatusCreated, newDonationResponse(result.Donation))
}

// GET /donations/:id
func (server *Server) getDonation(ctx *gin.Context) {
	idStr := ctx.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	donation, err := server.store.GetDonation(ctx, id)
	if err != nil {
		if err == sql.ErrNoRows {
			ctx.JSON(http.StatusNotFound, errorResponse(err))
			return
		}
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	ctx.JSON(http.StatusOK, newDonationResponse(donation))
}

// GET /donations
func (server *Server) listDonations(ctx *gin.Context) {
	limitStr := ctx.DefaultQuery("limit", "10")
	offsetStr := ctx.DefaultQuery("offset", "0")
	goalIDStr := ctx.Query("goal_id")

	limit, err := strconv.ParseInt(limitStr, 10, 32)
	if err != nil || limit <= 0 {
		limit = 10
	}

	offset, err := strconv.ParseInt(offsetStr, 10, 32)
	if err != nil || offset < 0 {
		offset = 0
	}

	var donations []db.Donation

	if goalIDStr != "" {
		goalID, err := strconv.ParseInt(goalIDStr, 10, 64)
		if err != nil {
			ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid goal_id"})
			return
		}

		arg := db.ListDonationsByGoalParams{
			GoalID: goalID,
			Limit:  int32(limit),
			Offset: int32(offset),
		}

		donations, err = server.store.ListDonationsByGoal(ctx, arg)
		if err != nil {
			ctx.JSON(http.StatusInternalServerError, errorResponse(err))
			return
		}
	} else {
		arg := db.ListDonationsParams{
			Limit:  int32(limit),
			Offset: int32(offset),
		}

		donations, err = server.store.ListDonations(ctx, arg)
		if err != nil {
			ctx.JSON(http.StatusInternalServerError, errorResponse(err))
			return
		}
	}

	response := make([]donationResponse, len(donations))
	for i, donation := range donations {
		response[i] = newDonationResponse(donation)
	}

	ctx.JSON(http.StatusOK, response)
}

// GET /users/me/donations
func (server *Server) listUserDonations(ctx *gin.Context) {
	authPayload := ctx.MustGet(authorizationPayloadKey).(*token.Payload)

	limitStr := ctx.DefaultQuery("limit", "10")
	offsetStr := ctx.DefaultQuery("offset", "0")

	limit, err := strconv.ParseInt(limitStr, 10, 32)
	if err != nil || limit <= 0 {
		limit = 10
	}

	offset, err := strconv.ParseInt(offsetStr, 10, 32)
	if err != nil || offset < 0 {
		offset = 0
	}

	arg := db.ListDonationsByUserParams{
		UserID: pgtype.Int8{
			Int64: authPayload.UserID,
			Valid: true,
		},
		Limit:  int32(limit),
		Offset: int32(offset),
	}

	donations, err := server.store.ListDonationsByUser(ctx, arg)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	response := make([]donationResponse, len(donations))
	for i, donation := range donations {
		response[i] = newDonationResponse(donation)
	}

	ctx.JSON(http.StatusOK, response)
}
