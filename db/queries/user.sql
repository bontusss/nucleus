-- name: CreateUser :one
INSERT INTO users (username, first_name, last_name, email, password_hash, gender, role_id, is_active, metadata)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
RETURNING *;

-- name: UpdateUser :one
UPDATE users
SET username   = COALESCE(sqlc.narg(username), username),
    first_name = COALESCE(sqlc.narg(first_name), first_name),
    last_name  = COALESCE(sqlc.narg(last_name), last_name),
    email      = COALESCE(sqlc.narg(email), email),
    gender     = COALESCE(sqlc.narg(gender), gender),
    role_id    = COALESCE(sqlc.narg(role_id), role_id),
    is_active  = COALESCE(sqlc.narg(is_active), is_active),
    metadata = COALESCE(sqlc.narg(metadata), metadata)
WHERE id = sqlc.arg(id)
RETURNING *;

-- name: DeleteUser :exec
DELETE FROM users WHERE id = $1;

-- name: GetUserByID :one
SELECT u.*, r.name as role_name FROM users u
JOIN roles r ON u.role_id = r.id
WHERE u.id = $1 LIMIT 1;

-- name: ListUsers :many
SELECT u.*, r.name as role_name FROM users u
JOIN roles r ON u.role_id = r.id
ORDER BY u.created_at DESC;

-- name: UpdateUserPassword :exec
UPDATE users
SET password_hash = $2, updated_at = CURRENT_TIMESTAMP
WHERE id = $1;

-- name: UpdateUserStatus :exec
UPDATE users
SET is_active = $2, updated_at = CURRENT_TIMESTAMP
WHERE id = $1;

-- name: CreateAdmin :one
INSERT INTO admins (username, email, first_name, last_name, password_hash, role_id, is_active)
VALUES ($1, $2, $3, $4, $5, $6, $7)
RETURNING *;

-- name: GetAdminByEmail :one
SELECT
    a.id,
    a.username,
    a.first_name,
    a.last_name,
    a.email,
    a.password_hash,
    a.is_active,
    a.email_verified,
    a.verification_code,
    a.verification_expires_at,
    a.reset_code,
    a.reset_code_expires_at,
    r.name as role_name
FROM admins a
JOIN roles r ON a.role_id = r.id
WHERE a.email = $1 LIMIT 1;


-- name: GetAdminByUsername :one
SELECT
    a.id,
    a.username,
    a.first_name,
    a.last_name,
    a.email,
    a.password_hash,
    a.is_active,
    a.email_verified,
    a.verification_code,
    a.verification_expires_at,
    a.reset_code,
    a.reset_code_expires_at,
    r.name as role_name
FROM admins a
JOIN roles r ON a.role_id = r.id
WHERE a.username = $1 LIMIT 1;

-- name: SetAdminEmailVerification :exec
UPDATE admins
SET verification_code = $2,
    verification_expires_at = $3,
    updated_at = NOW()
WHERE id = $1;

-- name: MarkAdminEmailVerified :exec
UPDATE admins
SET email_verified = $2,
    verification_code = NULL,
    verification_expires_at = NULL,
    updated_at = NOW()
WHERE id = $1;

-- name: UpdateAdminPassword :exec
UPDATE admins
SET password_hash = $2, updated_at = CURRENT_TIMESTAMP
WHERE id = $1;

-- name: ClearAdminResetCode :exec
UPDATE admins
SET reset_code = NULL,
    reset_code_expires_at = NULL,
    updated_at = NOW()
WHERE id = $1;

-- name: SetAdminResetCode :exec
UPDATE admins
SET reset_code = $2,
    reset_code_expires_at = $3,
    updated_at = NOW()
WHERE id = $1;

-- name: DeleteAdmin :exec
DELETE FROM users WHERE id = $1;


-- name: CreateRole :one
INSERT INTO roles (name, description)
VALUES ($1, $2)
RETURNING *;

-- name: UpdateRole :one
UPDATE roles
SET
    name = COALESCE($2, name),
    description = COALESCE($3, description)
WHERE id = $1
RETURNING *;

-- name: DeleteRole :exec
DELETE FROM roles WHERE id = $1;

-- name: GetRoleByID :one
SELECT * FROM roles WHERE id = $1 LIMIT 1;

-- name: ListRoles :many
SELECT * FROM roles ORDER BY name;

-- name: AddPermissionToRole :exec
INSERT INTO role_permissions (role_id, permission_id)
VALUES ($1, $2);

-- name: RemovePermissionFromRole :exec
DELETE FROM role_permissions
WHERE role_id = $1 AND permission_id = $2;

-- name: GetRolePermissions :many
SELECT p.* FROM permissions p
JOIN role_permissions rp ON p.id = rp.permission_id
WHERE rp.role_id = $1;

-- name: LogActivity :one
INSERT INTO activity_logs (user_id, action, details, entity_id, entity_type, ip_address, user_agent)
VALUES ($1, $2, $3, $4, $5, $6, $7)
RETURNING *;

-- name: GetActivityLogs :many
SELECT * FROM activity_logs
ORDER BY created_at DESC
LIMIT $1;

-- name: LogLoginAttempt :exec
INSERT INTO login_history (username_or_email, ip_address, user_agent, success, error_reason)
VALUES ($1, $2, $3, $4, $5)
RETURNING *;

-- name: GetLoginHistory :many
SELECT * FROM login_history
ORDER BY login_time DESC
LIMIT $1;

-- name: CreatePasswordResetToken :one
INSERT INTO password_reset_tokens (user_id, token, expires_at)
VALUES ($1, $2, $3)
RETURNING *;

-- name: GetPasswordResetToken :one
SELECT * FROM password_reset_tokens
WHERE token = $1 AND expires_at > NOW() AND used = FALSE
LIMIT 1;

-- name: MarkTokenAsUsed :exec
UPDATE password_reset_tokens
SET used = TRUE
WHERE id = $1;

-- name: GetUserByEmail :one
SELECT
    u.id,
    u.username,
    u.first_name,
    u.last_name,
    u.email,
    u.password_hash,
    u.gender,
    u.is_active,
    r.name as role_name
FROM users u
JOIN roles r ON u.role_id = r.id
WHERE u.email = $1 LIMIT 1;

-- name: GetUserPermissions :many
SELECT p.code
FROM permissions p
JOIN role_permissions rp ON p.id = rp.permission_id
JOIN roles r ON rp.role_id = r.id
JOIN users u ON u.role_id = r.id
WHERE u.id = $1;

-- name: GetAdminPermissions :many
SELECT p.code
FROM permissions p
JOIN role_permissions rp ON p.id = rp.permission_id
JOIN roles r ON rp.role_id = r.id
JOIN admins a ON a.role_id = r.id
WHERE a.id = $1;

-- name: GetUserByUsername :one
SELECT
    u.id,
    u.username,
    u.first_name,
    u.last_name,
    u.email,
    u.password_hash,
    u.gender,
    u.is_active,
    r.name as role_name
FROM users u
JOIN roles r ON u.role_id = r.id
WHERE u.username = $1 LIMIT 1;

-- name: CreateRefreshToken :one
INSERT INTO refresh_tokens (user_id, token, expires_at)
VALUES ($1, $2, $3)
RETURNING *;

-- name: GetRefreshToken :one
SELECT * FROM refresh_tokens
WHERE token = $1 AND expires_at > NOW() AND revoked = FALSE
LIMIT 1;

-- name: RevokeRefreshToken :exec
UPDATE refresh_tokens
SET revoked = TRUE, updated_at = CURRENT_TIMESTAMP
WHERE token = $1;

-- name: RevokeAllUserRefreshTokens :exec
UPDATE refresh_tokens
SET revoked = TRUE, updated_at = CURRENT_TIMESTAMP
WHERE user_id = $1;

-- name: CleanExpiredRefreshTokens :exec
DELETE FROM refresh_tokens
WHERE expires_at <= NOW() OR revoked = TRUE;
