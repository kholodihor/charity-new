package api

import (
	"fmt"

	"github.com/gin-gonic/gin"
	db "github.com/kholodihor/charity/db/sqlc"
	"github.com/kholodihor/charity/token"
	"github.com/kholodihor/charity/util"
)

// Server serves HTTP requests for our banking service.
type Server struct {
	config     util.Config
	store      db.Store
	tokenMaker token.Maker
	router     *gin.Engine
}

// NewServer creates a new HTTP server and set up routing.
func NewServer(config util.Config, store db.Store) (*Server, error) {
	tokenMaker, err := token.NewJWTMaker(config.TokenSymmetricKey)
	if err != nil {
		return nil, fmt.Errorf("cannot create token maker: %w", err)
	}

	server := &Server{
		config:     config,
		store:      store,
		tokenMaker: tokenMaker,
	}

	server.setupRouter()
	return server, nil
}

func (server *Server) setupRouter() {
	router := gin.Default()

	// Public routes
	router.POST("/users", server.createUser)
	router.POST("/users/login", server.loginUser)

	// Public goal routes (read-only)
	router.GET("/goals", server.listGoals)
	router.GET("/goals/:id", server.getGoal)

	// Public event routes (read-only)
	router.GET("/events", server.listEvents)
	router.GET("/events/:id", server.getEvent)

	// Public donation routes (read-only)
	router.GET("/donations", server.listDonations)
	router.GET("/donations/:id", server.getDonation)

	// Public user routes (read-only)
	router.GET("/users", server.listUsers)
	router.GET("/users/:id", server.getUser)

	// Protected routes (require authentication)
	authRoutes := router.Group("/").Use(authMiddleware(server.tokenMaker))

	// User profile management
	authRoutes.GET("/users/me", server.getCurrentUser)
	authRoutes.PUT("/users/me", server.updateCurrentUser)
	authRoutes.GET("/users/me/donations", server.listUserDonations)
	authRoutes.GET("/users/me/bookings", server.listUserBookings)

	// Goal management (admin/authenticated users)
	authRoutes.POST("/goals", server.createGoal)
	authRoutes.PUT("/goals/:id", server.updateGoal)
	authRoutes.DELETE("/goals/:id", server.deleteGoal)

	// Donation management
	authRoutes.POST("/donations", server.createDonation)

	// Event management (admin/authenticated users)
	authRoutes.POST("/events", server.createEvent)
	authRoutes.PUT("/events/:id", server.updateEvent)
	authRoutes.DELETE("/events/:id", server.deleteEvent)

	// Event booking management
	authRoutes.POST("/events/:id/book", server.bookEvent)
	authRoutes.DELETE("/events/:id/book", server.cancelEventBooking)
	authRoutes.GET("/events/:id/bookings", server.listEventBookings)

	server.router = router
}

// Start runs the HTTP server on a specific address.
func (server *Server) Start(address string) error {
	return server.router.Run(address)
}

func errorResponse(err error) gin.H {
	return gin.H{"error": err.Error()}
}
