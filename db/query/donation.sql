-- name: CreateDonation :one
INSERT INTO donations (
  goal_id,
  user_id,
  amount,
  is_anonymous
) VALUES (
  $1, $2, $3, $4
) RETURNING *;

-- name: GetDonation :one
SELECT * FROM donations
WHERE id = $1 LIMIT 1;

-- name: ListDonations :many
SELECT * FROM donations
ORDER BY created_at DESC
LIMIT $1
OFFSET $2;

-- name: ListDonationsByGoal :many
SELECT * FROM donations
WHERE goal_id = $1
ORDER BY created_at DESC
LIMIT $2
OFFSET $3;

-- name: ListDonationsByUser :many
SELECT * FROM donations
WHERE user_id = $1
ORDER BY created_at DESC
LIMIT $2
OFFSET $3;

-- name: UpdateGoalCollectedAmount :exec
UPDATE goals
SET collected_amount = collected_amount + $2
WHERE id = $1;
