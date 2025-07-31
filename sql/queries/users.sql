-- name: CreateUser :one
INSERT INTO users (id, created_at, updated_at, email, password)
VALUES (
    $1,
    $2,
    $3,
    $4,
    $5
) RETURNING *;

-- name: DeleteAllUsers :one
DELETE FROM users
RETURNING *;

-- name: GetUser :one
SELECT * FROM users
WHERE email = $1;

-- name: GetUserByID :one
SELECT * FROM users
WHERE id = $1;

-- name: UpdatePassword :one
WITH updated_user AS (
    UPDATE users
    SET password = $1, updated_at = $2, email = $3
    WHERE id = $4
    RETURNING *
)
SELECT updated_user.id, updated_user.email, refresh_tokens.tokens, updated_user.updated_at, updated_user.created_at, updated_user.is_chirpy_red
FROM updated_user
LEFT JOIN refresh_tokens ON updated_user.id = refresh_tokens.user_id;

-- name: UpgradeUsertoRed :one
UPDATE users
SET is_chirpy_red = TRUE
WHERE id = $1
RETURNING *;
