-- name: CreateGoal :one
INSERT INTO goals (
  title,
  description,
  target_amount,
  collected_amount,
  is_active
) VALUES (
  $1, $2, $3, $4, $5
) RETURNING *;

-- name: GetGoal :one
SELECT * FROM goals
WHERE id = $1 LIMIT 1;

-- name: GetGoalForUpdate :one
SELECT * FROM goals
WHERE id = $1 LIMIT 1
FOR NO KEY UPDATE;

-- name: ListGoals :many
SELECT * FROM goals
ORDER BY id
LIMIT $1
OFFSET $2;

-- name: UpdateGoal :one
UPDATE goals
SET 
  target_amount = COALESCE($2, target_amount),
  is_active = COALESCE($3, is_active)
WHERE id = $1
RETURNING *;

-- name: DeleteGoal :exec
DELETE FROM goals
WHERE id = $1;
