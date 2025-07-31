-- name: CreateRefreshToken :one
INSERT INTO refresh_tokens (tokens, created_at, updated_at, user_id, expires_at, revoked_at)
VALUES (
    $1,
    $2,
    $3,
    $4,
    $5,
    $6
) RETURNING *;

-- name: GetRefreshToken :one
SELECT * FROM refresh_tokens
WHERE tokens = $1;

-- name: RevokeToken :one
UPDATE refresh_tokens
SET revoked_at = $1
WHERE tokens = $2
RETURNING *;
