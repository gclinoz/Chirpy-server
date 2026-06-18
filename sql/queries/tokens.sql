-- name: CreateRefTk :one
INSERT INTO refresh_tokens (token, created_at, updated_at, user_id, expires_at)
VALUES (
    $1,
    NOW(),
    NOW(),
    $2,
    NOW() + INTERVAL '60 days'
)
RETURNING *;

-- name: GetUserFromRefreshToken :one
SELECT * FROM refresh_tokens
WHERE token = $1 LIMIT 1;

-- name: UpdateRevokeAt :exec
UPDATE refresh_tokens
SET updated_at = $2, revoked_at = $2
WHERE token = $1;
