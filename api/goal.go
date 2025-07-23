package api

import (
	"database/sql"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgtype"
	db "github.com/kholodihor/charity/db/sqlc"
)

type createGoalRequest struct {
	Title       string `json:"title" binding:"required"`
	Description string `json:"description"`
	TargetAmount int64 `json:"target_amount" binding:"required,min=1"`
}

type updateGoalRequest struct {
	Title       *string `json:"title"`
	Description *string `json:"description"`
	TargetAmount *int64 `json:"target_amount" binding:"omitempty,min=1"`
	IsActive    *bool   `json:"is_active"`
}

type goalResponse struct {
	ID              int64  `json:"id"`
	Title           string `json:"title"`
	Description     string `json:"description"`
	TargetAmount    int64  `json:"target_amount"`
	CollectedAmount int64  `json:"collected_amount"`
	IsActive        bool   `json:"is_active"`
	CreatedAt       string `json:"created_at"`
}

func newGoalResponse(goal db.Goal) goalResponse {
	description := ""
	if goal.Description.Valid {
		description = goal.Description.String
	}
	
	targetAmount := int64(0)
	if goal.TargetAmount.Valid {
		targetAmount = goal.TargetAmount.Int64
	}

	return goalResponse{
		ID:              goal.ID,
		Title:           goal.Title,
		Description:     description,
		TargetAmount:    targetAmount,
		CollectedAmount: goal.CollectedAmount,
		IsActive:        goal.IsActive,
		CreatedAt:       goal.CreatedAt.Format("2006-01-02T15:04:05Z"),
	}
}

// POST /goals
func (server *Server) createGoal(ctx *gin.Context) {
	var req createGoalRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	arg := db.CreateGoalParams{
		Title: req.Title,
		Description: pgtype.Text{
			String: req.Description,
			Valid:  req.Description != "",
		},
		TargetAmount: pgtype.Int8{
			Int64: req.TargetAmount,
			Valid: true,
		},
		CollectedAmount: 0,
		IsActive:        true,
	}

	goal, err := server.store.CreateGoal(ctx, arg)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	ctx.JSON(http.StatusCreated, newGoalResponse(goal))
}

// GET /goals/:id
func (server *Server) getGoal(ctx *gin.Context) {
	idStr := ctx.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	goal, err := server.store.GetGoal(ctx, id)
	if err != nil {
		if err == sql.ErrNoRows {
			ctx.JSON(http.StatusNotFound, errorResponse(err))
			return
		}
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	ctx.JSON(http.StatusOK, newGoalResponse(goal))
}

// GET /goals
func (server *Server) listGoals(ctx *gin.Context) {
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

	arg := db.ListGoalsParams{
		Limit:  int32(limit),
		Offset: int32(offset),
	}

	goals, err := server.store.ListGoals(ctx, arg)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	response := make([]goalResponse, len(goals))
	for i, goal := range goals {
		response[i] = newGoalResponse(goal)
	}

	ctx.JSON(http.StatusOK, response)
}

// PUT /goals/:id
func (server *Server) updateGoal(ctx *gin.Context) {
	idStr := ctx.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	var req updateGoalRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	arg := db.UpdateGoalParams{
		ID: id,
	}

	if req.TargetAmount != nil {
		arg.TargetAmount = pgtype.Int8{
			Int64: *req.TargetAmount,
			Valid: true,
		}
	}

	if req.IsActive != nil {
		arg.IsActive = *req.IsActive
	}

	goal, err := server.store.UpdateGoal(ctx, arg)
	if err != nil {
		if err == sql.ErrNoRows {
			ctx.JSON(http.StatusNotFound, errorResponse(err))
			return
		}
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	ctx.JSON(http.StatusOK, newGoalResponse(goal))
}

// DELETE /goals/:id
func (server *Server) deleteGoal(ctx *gin.Context) {
	idStr := ctx.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	err = server.store.DeleteGoal(ctx, id)
	if err != nil {
		if err == sql.ErrNoRows {
			ctx.JSON(http.StatusNotFound, errorResponse(err))
			return
		}
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	ctx.JSON(http.StatusNoContent, nil)
}
