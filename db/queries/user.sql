-- name: CreateTenant :one
INSERT INTO tenants (email, password_hash, is_active, organization, plan)
VALUES ($1, $2, true, $3, $4)
RETURNING *;


-- name: UpdateTenant :one
UPDATE tenants
SET organization   = COALESCE(sqlc.narg(organization), organization),
    email      = COALESCE(sqlc.narg(email), email),
    is_active  = COALESCE(sqlc.narg(is_active), is_active)
WHERE id = sqlc.arg(id)
RETURNING *;

-- name: GetTenantByID :one
SELECT * FROM tenants
WHERE id = $1 LIMIT 1;

-- name: UpdateTenantPassword :exec
UPDATE tenants
SET password_hash = $2, updated_at = CURRENT_TIMESTAMP
WHERE id = $1;

-- name: UpdateTenantStatus :exec
UPDATE tenants
SET is_active = $2, updated_at = CURRENT_TIMESTAMP
WHERE id = $1;


-- name: GetTenantByEmail :one
SELECT
    a.id,
    a.organization,
    a.email,
    a.password_hash,
    a.is_active,
    a.plan,
    a.email_verified,
    a.verification_code,
    a.verification_expires_at,
    a.reset_code,
    a.reset_code_expires_at
FROM tenants a
WHERE a.email = $1 LIMIT 1;


-- name: GetTenantByOrganization :one
SELECT
    a.id,
    a.organization,
    a.email,
    a.password_hash,
    a.is_active,
    a.plan,
    a.email_verified,
    a.verification_code,
    a.verification_expires_at,
    a.reset_code,
    a.reset_code_expires_at
FROM tenants a
WHERE a.organization = $1 LIMIT 1;

-- name: SetTenantEmailVerification :exec
UPDATE tenants
SET verification_code = $2,
    verification_expires_at = $3,
    updated_at = NOW()
WHERE id = $1;

-- name: MarkTenantEmailVerified :exec
UPDATE tenants
SET email_verified = $2,
    verification_code = NULL,
    verification_expires_at = NULL,
    updated_at = NOW()
WHERE id = $1;

-- name: ClearTenantResetCode :exec
UPDATE tenants
SET reset_code = NULL,
    reset_code_expires_at = NULL,
    updated_at = NOW()
WHERE id = $1;

-- name: SetTenantResetCode :exec
UPDATE tenants
SET reset_code = $2,
    reset_code_expires_at = $3,
    updated_at = NOW()
WHERE id = $1;

-- Legacy user creation for compatibility
-- name: CreateLegacyUser :one
INSERT INTO users(email, business_id, tenants_id, branch_id, password_hash, is_active)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING *;




-- Legacy permission creation for compatibility
-- name: CreateLegacyPermission :one
INSERT INTO permissions (code, description)
VALUES ($1, $2)
RETURNING *;

-- name: GetLegacyPermissionByCode :one
SELECT * FROM permissions
WHERE code = $1;

-- name: ListLegacyPermissions :many
SELECT * FROM permissions
ORDER BY id;

-- name: CreateBusinessRole :one
INSERT INTO business_roles (business_id, name, description)
VALUES ($1, $2, $3)
RETURNING *;

-- name: GetBusinessRoleByName :one
SELECT * FROM business_roles
WHERE business_id = $1 AND name = $2;

-- name: ListBusinessRoles :many
SELECT * FROM business_roles
WHERE business_id = $1
ORDER BY created_at DESC;

-- name: UpdateBusinessRole :one
UPDATE business_roles
SET name = COALESCE(sqlc.narg('name'), name),
    description = COALESCE(sqlc.narg('description'), description),
    updated_at = CURRENT_TIMESTAMP
WHERE id = $1
RETURNING *;

-- name: DeleteBusinessRole :exec
DELETE FROM business_roles
WHERE id = $1;

-- name: AddPermissionToRole :exec
INSERT INTO role_permissions (role_id, permission_id, business_id)
VALUES ($1, $2, $3)
ON CONFLICT (role_id, permission_id) DO NOTHING;

-- name: RemovePermissionFromRole :exec
DELETE FROM role_permissions
WHERE role_id = $1 AND permission_id = $2;

-- name: GetLegacyRolePermissions :many
SELECT p.* FROM permissions p
JOIN role_permissions rp ON p.id = rp.permission_id
WHERE rp.role_id = $1
ORDER BY p.code;

-- ACTIVITY LOGS

-- name: CreateActivityLog :one
INSERT INTO activity_logs (
    business_id, user_id, action, details, entity_id, entity_type, ip_address, user_agent, metadata
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, COALESCE($9, '{}')
)
RETURNING *;

-- name: GetActivityLog :one
SELECT * FROM activity_logs
WHERE id = $1
  AND business_id = $2
LIMIT 1;

-- name: ListActivityLogs :many
SELECT * FROM activity_logs
WHERE business_id = $1
ORDER BY created_at DESC;

-- name: ListActivityLogsByUser :many
SELECT * FROM activity_logs
WHERE business_id = $1
  AND user_id = $2
ORDER BY created_at DESC;

-- name: DeleteActivityLog :exec
DELETE FROM activity_logs
WHERE id = $1
  AND business_id = $2;

-- name: DeleteActivityLogs :exec
DELETE FROM activity_logs
WHERE business_id = $1
  AND id = ANY($2::int[]);


-- name: LogLoginAttempt :exec
INSERT INTO login_histories (username_or_email, ip_address, user_agent, success, error_reason)
VALUES ($1, $2, $3, $4, $5)
RETURNING *;

-- name: GetLoginHistory :many
SELECT * FROM login_histories
ORDER BY login_time DESC
LIMIT $1;

-- name: CreatePasswordResetToken :one
INSERT INTO password_reset_tokens (developer_id, token, expires_at)
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
