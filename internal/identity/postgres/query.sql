-- name: CreateUser :one
INSERT INTO users (phone)
VALUES ($1)
RETURNING id, phone, created_at, updated_at;

-- name: GetUserByID :one
SELECT id, phone, created_at, updated_at
FROM users
WHERE id = $1;

-- name: GetUserByPhone :one
SELECT id, phone, created_at, updated_at
FROM users
WHERE phone = $1;

-- name: CreateOTPChallenge :one
INSERT INTO otp_challenges (phone, code_hash, expires_at)
VALUES ($1, $2, $3)
RETURNING id, phone, code_hash, expires_at, consumed_at, created_at;

-- name: LatestActiveOTPByPhone :one
SELECT id, phone, code_hash, expires_at, consumed_at, created_at
FROM otp_challenges
WHERE phone = $1
  AND consumed_at IS NULL
  AND expires_at > $2
ORDER BY created_at DESC
LIMIT 1;

-- name: MarkOTPConsumed :execrows
UPDATE otp_challenges
SET consumed_at = $2
WHERE id = $1
  AND consumed_at IS NULL;