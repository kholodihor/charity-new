package api

import (
	"database/sql"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgtype"
	db "github.com/kholodihor/charity/db/sqlc"
	"github.com/kholodihor/charity/token"
)

type createEventRequest struct {
	Name  string    `json:"name" binding:"required"`
	Place string    `json:"place" binding:"required"`
	Date  time.Time `json:"date" binding:"required"`
}

type updateEventRequest struct {
	Name  *string    `json:"name"`
	Place *string    `json:"place"`
	Date  *time.Time `json:"date"`
}

type eventResponse struct {
	ID        int64  `json:"id"`
	Name      string `json:"name"`
	Place     string `json:"place"`
	Date      string `json:"date"`
	CreatedAt string `json:"created_at"`
}

type eventBookingResponse struct {
	ID       int64  `json:"id"`
	UserID   int64  `json:"user_id"`
	EventID  int64  `json:"event_id"`
	BookedAt string `json:"booked_at"`
}

type eventBookingWithUserResponse struct {
	ID        int64  `json:"id"`
	UserID    int64  `json:"user_id"`
	EventID   int64  `json:"event_id"`
	BookedAt  string `json:"booked_at"`
	UserName  string `json:"user_name"`
	UserEmail string `json:"user_email"`
}

type userBookingWithEventResponse struct {
	ID         int64  `json:"id"`
	UserID     int64  `json:"user_id"`
	EventID    int64  `json:"event_id"`
	BookedAt   string `json:"booked_at"`
	EventName  string `json:"event_name"`
	EventPlace string `json:"event_place"`
	EventDate  string `json:"event_date"`
}

func newEventResponse(event db.Event) eventResponse {
	return eventResponse{
		ID:        event.ID,
		Name:      event.Name,
		Place:     event.Place,
		Date:      event.Date.Format("2006-01-02T15:04:05Z"),
		CreatedAt: event.CreatedAt.Format("2006-01-02T15:04:05Z"),
	}
}

func newEventBookingResponse(booking db.EventBooking) eventBookingResponse {
	return eventBookingResponse{
		ID:       booking.ID,
		UserID:   booking.UserID,
		EventID:  booking.EventID,
		BookedAt: booking.BookedAt.Format("2006-01-02T15:04:05Z"),
	}
}

func newEventBookingWithUserResponse(booking db.ListEventBookingsRow) eventBookingWithUserResponse {
	userName := ""
	if booking.UserName.Valid {
		userName = booking.UserName.String
	}
	return eventBookingWithUserResponse{
		ID:        booking.ID,
		UserID:    booking.UserID,
		EventID:   booking.EventID,
		BookedAt:  booking.BookedAt.Format("2006-01-02T15:04:05Z"),
		UserName:  userName,
		UserEmail: booking.UserEmail,
	}
}

func newUserBookingWithEventResponse(booking db.ListUserBookingsRow) userBookingWithEventResponse {
	return userBookingWithEventResponse{
		ID:         booking.ID,
		UserID:     booking.UserID,
		EventID:    booking.EventID,
		BookedAt:   booking.BookedAt.Format("2006-01-02T15:04:05Z"),
		EventName:  booking.EventName,
		EventPlace: booking.EventPlace,
		EventDate:  booking.EventDate.Format("2006-01-02T15:04:05Z"),
	}
}

// POST /events
func (server *Server) createEvent(ctx *gin.Context) {
	var req createEventRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	arg := db.CreateEventParams{
		Name:  req.Name,
		Place: req.Place,
		Date:  req.Date,
	}

	event, err := server.store.CreateEvent(ctx, arg)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	ctx.JSON(http.StatusCreated, newEventResponse(event))
}

// GET /events/:id
func (server *Server) getEvent(ctx *gin.Context) {
	idStr := ctx.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	event, err := server.store.GetEvent(ctx, id)
	if err != nil {
		if err == sql.ErrNoRows {
			ctx.JSON(http.StatusNotFound, errorResponse(err))
			return
		}
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	ctx.JSON(http.StatusOK, newEventResponse(event))
}

// GET /events
func (server *Server) listEvents(ctx *gin.Context) {
	limitStr := ctx.DefaultQuery("limit", "10")
	offsetStr := ctx.DefaultQuery("offset", "0")
	upcoming := ctx.Query("upcoming") == "true"

	limit, err := strconv.ParseInt(limitStr, 10, 32)
	if err != nil || limit <= 0 {
		limit = 10
	}

	offset, err := strconv.ParseInt(offsetStr, 10, 32)
	if err != nil || offset < 0 {
		offset = 0
	}

	var events []db.Event

	if upcoming {
		arg := db.ListUpcomingEventsParams{
			Limit:  int32(limit),
			Offset: int32(offset),
		}

		events, err = server.store.ListUpcomingEvents(ctx, arg)
	} else {
		arg := db.ListEventsParams{
			Limit:  int32(limit),
			Offset: int32(offset),
		}

		events, err = server.store.ListEvents(ctx, arg)
	}

	if err != nil {
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	response := make([]eventResponse, len(events))
	for i, event := range events {
		response[i] = newEventResponse(event)
	}

	ctx.JSON(http.StatusOK, response)
}

// PUT /events/:id
func (server *Server) updateEvent(ctx *gin.Context) {
	idStr := ctx.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	var req updateEventRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	arg := db.UpdateEventParams{
		ID: id,
	}

	if req.Name != nil {
		arg.Name = pgtype.Text{
			String: *req.Name,
			Valid:  true,
		}
	}

	if req.Place != nil {
		arg.Place = pgtype.Text{
			String: *req.Place,
			Valid:  true,
		}
	}

	if req.Date != nil {
		arg.Date = pgtype.Timestamptz{
			Time:  *req.Date,
			Valid: true,
		}
	}

	event, err := server.store.UpdateEvent(ctx, arg)
	if err != nil {
		if err == sql.ErrNoRows {
			ctx.JSON(http.StatusNotFound, errorResponse(err))
			return
		}
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	ctx.JSON(http.StatusOK, newEventResponse(event))
}

// DELETE /events/:id
func (server *Server) deleteEvent(ctx *gin.Context) {
	idStr := ctx.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	err = server.store.DeleteEvent(ctx, id)
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

// POST /events/:id/book
func (server *Server) bookEvent(ctx *gin.Context) {
	idStr := ctx.Param("id")
	eventID, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	authPayload := ctx.MustGet(authorizationPayloadKey).(*token.Payload)

	// Check if event exists
	_, err = server.store.GetEvent(ctx, eventID)
	if err != nil {
		if err == sql.ErrNoRows {
			ctx.JSON(http.StatusNotFound, gin.H{"error": "event not found"})
			return
		}
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	arg := db.BookEventParams{
		UserID:  authPayload.UserID,
		EventID: eventID,
	}

	booking, err := server.store.BookEvent(ctx, arg)
	if err != nil {
		// Check for unique constraint violation (user already booked this event)
		ctx.JSON(http.StatusConflict, gin.H{"error": "event already booked by user"})
		return
	}

	ctx.JSON(http.StatusCreated, newEventBookingResponse(booking))
}

// DELETE /events/:id/book
func (server *Server) cancelEventBooking(ctx *gin.Context) {
	idStr := ctx.Param("id")
	eventID, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	authPayload := ctx.MustGet(authorizationPayloadKey).(*token.Payload)

	arg := db.CancelEventBookingParams{
		UserID:  authPayload.UserID,
		EventID: eventID,
	}

	err = server.store.CancelEventBooking(ctx, arg)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	ctx.JSON(http.StatusNoContent, nil)
}

// GET /events/:id/bookings
func (server *Server) listEventBookings(ctx *gin.Context) {
	eventIDStr := ctx.Param("id")
	eventID, err := strconv.ParseInt(eventIDStr, 10, 64)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	limitStr := ctx.DefaultQuery("limit", "10")
	limit, err := strconv.Atoi(limitStr)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	offsetStr := ctx.DefaultQuery("offset", "0")
	offset, err := strconv.Atoi(offsetStr)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	arg := db.ListEventBookingsParams{
		EventID: eventID,
		Limit:   int32(limit),
		Offset:  int32(offset),
	}

	bookings, err := server.store.ListEventBookings(ctx, arg)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	response := make([]eventBookingWithUserResponse, len(bookings))
	for i, booking := range bookings {
		response[i] = newEventBookingWithUserResponse(booking)
	}

	ctx.JSON(http.StatusOK, response)
}

// GET /users/me/bookings
func (server *Server) listUserBookings(ctx *gin.Context) {
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

	arg := db.ListUserBookingsParams{
		UserID: authPayload.UserID,
		Limit:  int32(limit),
		Offset: int32(offset),
	}

	bookings, err := server.store.ListUserBookings(ctx, arg)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	response := make([]userBookingWithEventResponse, len(bookings))
	for i, booking := range bookings {
		response[i] = newUserBookingWithEventResponse(booking)
	}

	ctx.JSON(http.StatusOK, response)
}
