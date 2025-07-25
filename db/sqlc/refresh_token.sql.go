// Code generated by sqlc. DO NOT EDIT.
// versions:
//   sqlc v1.29.0
// source: refresh_token.sql

package db

import (
	"context"
	"time"

	"github.com/google/uuid"
)

const cleanupExpiredRefreshTokens = `-- name: CleanupExpiredRefreshTokens :exec
DELETE FROM refresh_tokens
WHERE expires_at < now() OR revoked_at IS NOT NULL
`

func (q *Queries) CleanupExpiredRefreshTokens(ctx context.Context) error {
	_, err := q.db.Exec(ctx, cleanupExpiredRefreshTokens)
	return err
}

const createRefreshToken = `-- name: CreateRefreshToken :one
INSERT INTO refresh_tokens (
  user_id,
  token_id,
  expires_at
) VALUES (
  $1, $2, $3
) RETURNING id, user_id, token_id, expires_at, created_at, revoked_at
`

type CreateRefreshTokenParams struct {
	UserID    int64     `json:"user_id"`
	TokenID   uuid.UUID `json:"token_id"`
	ExpiresAt time.Time `json:"expires_at"`
}

func (q *Queries) CreateRefreshToken(ctx context.Context, arg CreateRefreshTokenParams) (RefreshToken, error) {
	row := q.db.QueryRow(ctx, createRefreshToken, arg.UserID, arg.TokenID, arg.ExpiresAt)
	var i RefreshToken
	err := row.Scan(
		&i.ID,
		&i.UserID,
		&i.TokenID,
		&i.ExpiresAt,
		&i.CreatedAt,
		&i.RevokedAt,
	)
	return i, err
}

const getRefreshToken = `-- name: GetRefreshToken :one
SELECT id, user_id, token_id, expires_at, created_at, revoked_at FROM refresh_tokens
WHERE token_id = $1 AND revoked_at IS NULL AND expires_at > now()
`

func (q *Queries) GetRefreshToken(ctx context.Context, tokenID uuid.UUID) (RefreshToken, error) {
	row := q.db.QueryRow(ctx, getRefreshToken, tokenID)
	var i RefreshToken
	err := row.Scan(
		&i.ID,
		&i.UserID,
		&i.TokenID,
		&i.ExpiresAt,
		&i.CreatedAt,
		&i.RevokedAt,
	)
	return i, err
}

const revokeAllUserRefreshTokens = `-- name: RevokeAllUserRefreshTokens :exec
UPDATE refresh_tokens
SET revoked_at = now()
WHERE user_id = $1 AND revoked_at IS NULL
`

func (q *Queries) RevokeAllUserRefreshTokens(ctx context.Context, userID int64) error {
	_, err := q.db.Exec(ctx, revokeAllUserRefreshTokens, userID)
	return err
}

const revokeRefreshToken = `-- name: RevokeRefreshToken :exec
UPDATE refresh_tokens
SET revoked_at = now()
WHERE token_id = $1
`

func (q *Queries) RevokeRefreshToken(ctx context.Context, tokenID uuid.UUID) error {
	_, err := q.db.Exec(ctx, revokeRefreshToken, tokenID)
	return err
}
