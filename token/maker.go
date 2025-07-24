package token

import (
	"time"
)

// Maker is an interface for managing tokens
type Maker interface {
	// CreateToken creates a new access token for a specific userID and duration
	CreateToken(userID int64, duration time.Duration) (string, *Payload, error)

	// CreateRefreshToken creates a new refresh token for a specific userID and duration
	CreateRefreshToken(userID int64, duration time.Duration) (string, *Payload, error)

	// VerifyToken checks if the token is valid or not
	VerifyToken(token string, tokenType TokenType) (*Payload, error)
}
