-- name: CreateEvent :one
INSERT INTO events (
  name,
  place,
  date
) VALUES (
  $1, $2, $3
) RETURNING *;

-- name: GetEvent :one
SELECT * FROM events
WHERE id = $1 LIMIT 1;

-- name: ListEvents :many
SELECT * FROM events
ORDER BY date ASC
LIMIT $1
OFFSET $2;

-- name: ListUpcomingEvents :many
SELECT * FROM events
WHERE date > NOW()
ORDER BY date ASC
LIMIT $1
OFFSET $2;

-- name: UpdateEvent :one
UPDATE events
SET 
  name = COALESCE(sqlc.narg(name), name),
  place = COALESCE(sqlc.narg(place), place),
  date = COALESCE(sqlc.narg(date), date)
WHERE id = sqlc.arg(id)
RETURNING *;

-- name: DeleteEvent :exec
DELETE FROM events
WHERE id = $1;

-- name: BookEvent :one
INSERT INTO event_bookings (
  user_id,
  event_id
) VALUES (
  $1, $2
) RETURNING *;

-- name: CancelEventBooking :exec
DELETE FROM event_bookings
WHERE user_id = $1 AND event_id = $2;

-- name: GetEventBooking :one
SELECT * FROM event_bookings
WHERE user_id = $1 AND event_id = $2 LIMIT 1;

-- name: ListUserBookings :many
SELECT 
  eb.id,
  eb.user_id,
  eb.event_id,
  eb.booked_at,
  e.name as event_name,
  e.place as event_place,
  e.date as event_date
FROM event_bookings eb
JOIN events e ON eb.event_id = e.id
WHERE eb.user_id = $1
ORDER BY eb.booked_at DESC
LIMIT $2
OFFSET $3;

-- name: ListEventBookings :many
SELECT 
  eb.id,
  eb.user_id,
  eb.event_id,
  eb.booked_at,
  u.name as user_name,
  u.email as user_email
FROM event_bookings eb
JOIN users u ON eb.user_id = u.id
WHERE eb.event_id = $1
ORDER BY eb.booked_at DESC
LIMIT $2
OFFSET $3;

-- name: IsEventBooked :one
SELECT EXISTS(
  SELECT 1 FROM event_bookings
  WHERE user_id = $1 AND event_id = $2
) as is_booked;
