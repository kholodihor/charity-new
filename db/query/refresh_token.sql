-- name: CreateRefreshToken :one
INSERT INTO refresh_tokens (
  user_id,
  token_id,
  expires_at
) VALUES (
  $1, $2, $3
) RETURNING *;

-- name: GetRefreshToken :one
SELECT * FROM refresh_tokens
WHERE token_id = $1 AND revoked_at IS NULL AND expires_at > now();

-- name: RevokeRefreshToken :exec
UPDATE refresh_tokens
SET revoked_at = now()
WHERE token_id = $1;

-- name: RevokeAllUserRefreshTokens :exec
UPDATE refresh_tokens
SET revoked_at = now()
WHERE user_id = $1 AND revoked_at IS NULL;

-- name: CleanupExpiredRefreshTokens :exec
DELETE FROM refresh_tokens
WHERE expires_at < now() OR revoked_at IS NOT NULL;
