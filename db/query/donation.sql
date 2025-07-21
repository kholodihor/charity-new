-- name: CreateDonation :one
INSERT INTO donations (
  goal_id,
  user_id,
  amount,
  is_anonymous
) VALUES (
  $1, $2, $3, $4
) RETURNING *;

-- name: UpdateGoalCollectedAmount :exec
UPDATE goals
SET collected_amount = collected_amount + $2
WHERE id = $1;
