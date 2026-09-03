-- name: CreateUser :one
WITH inserted AS (
    INSERT INTO users (role_id, email, password_hash)
    SELECT id, sqlc.arg(email), sqlc.arg(password_hash)
    FROM roles
    WHERE roles.name = sqlc.arg(role_name)
    RETURNING id, role_id, email, password_hash, is_active, created_at, updated_at, deleted_at
)
SELECT inserted.id, inserted.email, inserted.password_hash, roles.name AS role_name,
       inserted.is_active, inserted.created_at, inserted.updated_at, inserted.deleted_at
FROM inserted
JOIN roles ON roles.id = inserted.role_id;

-- name: GetUserByEmail :one
SELECT users.id, users.email, users.password_hash, roles.name AS role_name,
       users.is_active, users.created_at, users.updated_at, users.deleted_at
FROM users
JOIN roles ON roles.id = users.role_id
WHERE LOWER(users.email) = LOWER(sqlc.arg(email)) AND users.deleted_at IS NULL;

-- name: GetUserByID :one
SELECT users.id, users.email, users.password_hash, roles.name AS role_name,
       users.is_active, users.created_at, users.updated_at, users.deleted_at
FROM users
JOIN roles ON roles.id = users.role_id
WHERE users.id = sqlc.arg(id) AND users.deleted_at IS NULL;

-- name: CreateRefreshSession :exec
INSERT INTO refresh_sessions (user_id, token_id, token_hash, expires_at)
VALUES (sqlc.arg(user_id), sqlc.arg(token_id), sqlc.arg(token_hash), sqlc.arg(expires_at));

-- name: GetRefreshSessionByTokenID :one
SELECT id, user_id, token_id, token_hash, expires_at, revoked_at, replaced_by_token_id, created_at
FROM refresh_sessions
WHERE token_id = sqlc.arg(token_id);

-- name: RevokeRefreshSession :execrows
UPDATE refresh_sessions
SET revoked_at = COALESCE(revoked_at, NOW())
WHERE token_id = sqlc.arg(token_id);

-- name: RotateRefreshSession :execrows
UPDATE refresh_sessions
SET revoked_at = NOW(), replaced_by_token_id = sqlc.arg(replaced_by_token_id)
WHERE token_id = sqlc.arg(token_id) AND revoked_at IS NULL AND expires_at > NOW();
