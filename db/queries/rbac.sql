-- RBAC System Database Queries
-- Queries for managing roles, permissions, and user assignments in the RBAC system
-- Focus: User roles under tenants (no tenant-level roles)

-- ========================================
-- TENANT ROLES QUERIES (User roles per tenant)
-- ========================================

-- name: CreateTenantRole :one
INSERT INTO tenant_roles (
    tenant_id, name, description, parent_role_id, is_default, metadata
) VALUES (
    $1, $2, $3, $4, $5, $6
) RETURNING *;

-- name: GetTenantRoleByID :one
SELECT * FROM tenant_roles
WHERE id = $1 AND tenant_id = $2;

-- name: GetTenantRoleByName :one
SELECT * FROM tenant_roles
WHERE name = $1 AND tenant_id = $2;

-- name: ListTenantRoles :many
SELECT * FROM tenant_roles
WHERE tenant_id = $1 AND is_active = true
ORDER BY level ASC, name ASC;

-- name: ListTenantRolesHierarchy :many
WITH RECURSIVE role_tree AS (
    -- Base case: root roles (no parent)
    SELECT tr.id, tr.tenant_id, tr.name, tr.description, tr.parent_role_id, tr.level, tr.is_default, tr.is_active, tr.metadata, tr.created_at, tr.updated_at, 0 as depth
    FROM tenant_roles tr
    WHERE tr.parent_role_id IS NULL AND tr.tenant_id = $1 AND tr.is_active = true

    UNION ALL

    -- Recursive case: child roles
    SELECT tr.id, tr.tenant_id, tr.name, tr.description, tr.parent_role_id, tr.level, tr.is_default, tr.is_active, tr.metadata, tr.created_at, tr.updated_at, rt.depth + 1
    FROM tenant_roles tr
    JOIN role_tree rt ON tr.parent_role_id = rt.id
    WHERE tr.is_active = true AND rt.depth < 10
)
SELECT * FROM role_tree
ORDER BY depth, level, name;

-- name: UpdateTenantRole :one
UPDATE tenant_roles
SET name = COALESCE(sqlc.narg('name'), name),
    description = COALESCE(sqlc.narg('description'), description),
    parent_role_id = COALESCE(sqlc.narg('parent_role_id'), parent_role_id),
    is_default = COALESCE(sqlc.narg('is_default'), is_default),
    is_active = COALESCE(sqlc.narg('is_active'), is_active),
    metadata = COALESCE(sqlc.narg('metadata'), metadata),
    updated_at = CURRENT_TIMESTAMP
WHERE id = sqlc.arg('id') AND tenant_id = sqlc.arg('tenant_id')
RETURNING *;

-- name: DeleteTenantRole :exec
DELETE FROM tenant_roles
WHERE id = $1 AND tenant_id = $2;

-- name: GetRoleHierarchy :many
WITH RECURSIVE role_hierarchy AS (
    -- Start from the given role
    SELECT tr.id, tr.tenant_id, tr.name, tr.parent_role_id, tr.level, 0 as depth, ARRAY[tr.id] as path
    FROM tenant_roles tr
    WHERE tr.id = $1 AND tr.tenant_id = $2

    UNION ALL

    -- Get parent roles recursively
    SELECT tr.id, tr.tenant_id, tr.name, tr.parent_role_id, tr.level, rh.depth + 1, rh.path || tr.id
    FROM tenant_roles tr
    JOIN role_hierarchy rh ON tr.id = rh.parent_role_id
    WHERE rh.depth < 10 AND NOT tr.id = ANY(rh.path) -- prevent cycles
)
SELECT * FROM role_hierarchy
ORDER BY depth;

-- name: GetChildRoles :many
WITH RECURSIVE child_roles AS (
    -- Start from the given role
    SELECT tr.id, tr.tenant_id, tr.name, tr.parent_role_id, tr.level, 0 as depth
    FROM tenant_roles tr
    WHERE tr.id = $1 AND tr.tenant_id = $2

    UNION ALL

    -- Get child roles recursively
    SELECT tr.id, tr.tenant_id, tr.name, tr.parent_role_id, tr.level, cr.depth + 1
    FROM tenant_roles tr
    JOIN child_roles cr ON tr.parent_role_id = cr.id
    WHERE cr.depth < 10
)
SELECT * FROM child_roles
WHERE depth > 0
ORDER BY depth, level;

-- ========================================
-- USER ROLE ASSIGNMENTS QUERIES
-- ========================================

-- name: AssignRoleToUser :one
INSERT INTO user_role_assignments (
    user_id, tenant_role_id, business_id, branch_id, assigned_by, expires_at, metadata
) VALUES (
    $1, $2, $3, $4, $5, $6, $7
) RETURNING *;

-- name: GetUserRoleAssignments :many
SELECT ura.*, tr.name as role_name, tr.description as role_description,
       b.name as business_name, br.name as branch_name
FROM user_role_assignments ura
JOIN tenant_roles tr ON ura.tenant_role_id = tr.id
LEFT JOIN businesses b ON ura.business_id = b.id
LEFT JOIN branches br ON ura.branch_id = br.id
WHERE ura.user_id = $1 AND ura.is_active = true
AND (ura.expires_at IS NULL OR ura.expires_at > NOW())
ORDER BY ura.assigned_at DESC;

-- name: GetRoleUsers :many
SELECT ura.*, u.email, u.first_name, u.last_name,
       b.name as business_name, br.name as branch_name
FROM user_role_assignments ura
JOIN users u ON ura.user_id = u.id
LEFT JOIN businesses b ON ura.business_id = b.id
LEFT JOIN branches br ON ura.branch_id = br.id
WHERE ura.tenant_role_id = $1 AND ura.is_active = true
AND (ura.expires_at IS NULL OR ura.expires_at > NOW())
ORDER BY u.email;

-- name: RemoveRoleFromUser :exec
UPDATE user_role_assignments
SET is_active = false
WHERE user_id = $1 AND tenant_role_id = $2
AND (business_id = $3 OR ($3::integer IS NULL AND business_id IS NULL))
AND (branch_id = $4 OR ($4::integer IS NULL AND branch_id IS NULL));

-- name: GetUserRoleAssignment :one
SELECT ura.*, tr.name as role_name
FROM user_role_assignments ura
JOIN tenant_roles tr ON ura.tenant_role_id = tr.id
WHERE ura.user_id = $1 AND ura.tenant_role_id = $2
AND (ura.business_id = $3 OR ($3::integer IS NULL AND ura.business_id IS NULL))
AND (ura.branch_id = $4 OR ($4::integer IS NULL AND ura.branch_id IS NULL))
AND ura.is_active = true;

-- ========================================
-- TENANT ROLE PERMISSIONS QUERIES
-- ========================================

-- name: GrantPermissionToRole :one
INSERT INTO tenant_role_permissions (
    tenant_role_id, permission_id, scope_type, scope_id, granted_by, metadata
) VALUES (
    $1, $2, $3, $4, $5, $6
) RETURNING *;

-- name: RevokePermissionFromRole :exec
UPDATE tenant_role_permissions
SET is_active = false
WHERE tenant_role_id = $1 AND permission_id = $2
AND scope_type = $3 AND (scope_id = $4 OR ($4::integer IS NULL AND scope_id IS NULL));

-- name: GetRolePermissions :many
SELECT trp.*, p.code, p.description, p.module, p.action, p.resource
FROM tenant_role_permissions trp
JOIN permissions p ON trp.permission_id = p.id
WHERE trp.tenant_role_id = $1 AND trp.is_active = true
ORDER BY p.module, p.action, p.resource;

-- name: GetPermissionRoles :many
SELECT trp.*, tr.name as role_name, tr.description as role_description
FROM tenant_role_permissions trp
JOIN tenant_roles tr ON trp.tenant_role_id = tr.id
WHERE trp.permission_id = $1 AND trp.is_active = true
ORDER BY tr.level, tr.name;

-- name: CheckRolePermission :one
SELECT COUNT(*) > 0 as has_permission
FROM tenant_role_permissions trp
WHERE trp.tenant_role_id = $1 AND trp.permission_id = $2
AND trp.scope_type = $3
AND (trp.scope_id = $4 OR ($4::integer IS NULL AND trp.scope_id IS NULL))
AND trp.is_active = true;

-- ========================================
-- PERMISSION QUERIES
-- ========================================

-- name: CreatePermission :one
INSERT INTO permissions (code, description, module, action, resource, scope)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING *;

-- name: GetPermissionByCode :one
SELECT * FROM permissions
WHERE code = $1 AND is_active = true;

-- name: GetPermissionByID :one
SELECT * FROM permissions
WHERE id = $1 AND is_active = true;

-- name: ListPermissions :many
SELECT * FROM permissions
WHERE is_active = true
ORDER BY module, action, resource;

-- name: ListPermissionsByModule :many
SELECT * FROM permissions
WHERE module = $1 AND is_active = true
ORDER BY action, resource;

-- name: UpdatePermission :one
UPDATE permissions
SET description = COALESCE(sqlc.narg('description'), description),
    module = COALESCE(sqlc.narg('module'), module),
    action = COALESCE(sqlc.narg('action'), action),
    resource = COALESCE(sqlc.narg('resource'), resource),
    scope = COALESCE(sqlc.narg('scope'), scope),
    is_active = COALESCE(sqlc.narg('is_active'), is_active)
WHERE id = sqlc.arg('id')
RETURNING *;

-- ========================================
-- USER PERMISSION QUERIES (AGGREGATED)
-- ========================================

-- name: GetUserPermissions :many
SELECT DISTINCT p.id, p.code, p.description, p.module, p.action, p.resource,
       trp.scope_type, trp.scope_id, tr.name as role_name
FROM users u
JOIN user_role_assignments ura ON u.id = ura.user_id
JOIN tenant_roles tr ON ura.tenant_role_id = tr.id
JOIN tenant_role_permissions trp ON tr.id = trp.tenant_role_id
JOIN permissions p ON trp.permission_id = p.id
WHERE u.id = $1 AND u.tenants_id = $2
AND ura.is_active = true AND trp.is_active = true AND p.is_active = true
AND (ura.expires_at IS NULL OR ura.expires_at > NOW())
ORDER BY p.module, p.action, p.resource;

-- name: GetUserPermissionsWithHierarchy :many
WITH RECURSIVE role_hierarchy AS (
    -- Get direct role assignments for the user
    SELECT DISTINCT tr.id as role_id, tr.name as role_name, tr.parent_role_id, 0 as level
    FROM tenant_roles tr
    JOIN user_role_assignments ura ON tr.id = ura.tenant_role_id
    WHERE ura.user_id = $1 AND tr.tenant_id = $2
    AND ura.is_active = TRUE
    AND (ura.expires_at IS NULL OR ura.expires_at > NOW())

    UNION ALL

    -- Get parent roles recursively
    SELECT tr.id as role_id, tr.name as role_name, tr.parent_role_id, rh.level + 1
    FROM tenant_roles tr
    JOIN role_hierarchy rh ON tr.id = rh.parent_role_id
    WHERE rh.level < 10 -- prevent infinite recursion
)
SELECT DISTINCT
    p.id, p.code, p.description, p.module, p.action, p.resource,
    trp.scope_type, trp.scope_id, rh.role_name
FROM role_hierarchy rh
JOIN tenant_role_permissions trp ON rh.role_id = trp.tenant_role_id
JOIN permissions p ON trp.permission_id = p.id
WHERE trp.is_active = TRUE AND p.is_active = TRUE
ORDER BY p.module, p.action, p.resource;

-- name: CheckUserPermission :one
WITH RECURSIVE role_hierarchy AS (
    -- Get direct role assignments for the user
    SELECT DISTINCT tr.id AS role_id, tr.parent_role_id
    FROM tenant_roles tr
    JOIN user_role_assignments ura ON tr.id = ura.tenant_role_id
    WHERE ura.user_id = $1 AND tr.tenant_id = $2
    AND ura.is_active = TRUE
    AND (ura.expires_at IS NULL OR ura.expires_at > NOW())
    AND (ura.business_id = $4 OR $4::integer IS NULL OR ura.business_id IS NULL)
    AND (ura.branch_id = $5 OR $5::integer IS NULL OR ura.branch_id IS NULL)

    UNION ALL

    -- Get parent roles recursively
    SELECT tr.id AS role_id, tr.parent_role_id
    FROM tenant_roles tr
    JOIN role_hierarchy rh ON tr.id = rh.parent_role_id
)
SELECT COUNT(*) > 0 AS has_permission
FROM role_hierarchy rh
JOIN tenant_role_permissions trp ON rh.role_id = trp.tenant_role_id
JOIN permissions p ON trp.permission_id = p.id
WHERE p.code = $3
AND trp.is_active = TRUE AND p.is_active = TRUE
AND (trp.scope_id = $4 OR $4::integer IS NULL OR trp.scope_id IS NULL);

-- ========================================
-- USER MANAGEMENT QUERIES (ENHANCED)
-- ========================================

-- name: CreateUser :one
INSERT INTO users (
    tenants_id, business_id, branch_id, email, password_hash,
    first_name, last_name, phone, is_active, metadata
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10
) RETURNING *;

-- name: GetUserByID :one
SELECT u.*, t.organization as tenant_name, b.name as business_name, br.name as branch_name
FROM users u
JOIN tenants t ON u.tenants_id = t.id
LEFT JOIN businesses b ON u.business_id = b.id
LEFT JOIN branches br ON u.branch_id = br.id
WHERE u.id = $1 AND u.tenants_id = $2;

-- name: GetUserByEmail :one
SELECT u.*, t.organization as tenant_name, b.name as business_name, br.name as branch_name
FROM users u
JOIN tenants t ON u.tenants_id = t.id
LEFT JOIN businesses b ON u.business_id = b.id
LEFT JOIN branches br ON u.branch_id = br.id
WHERE u.email = $1 AND u.tenants_id = $2;

-- name: ListTenantUsers :many
SELECT u.*, t.organization as tenant_name, b.name as business_name, br.name as branch_name
FROM users u
JOIN tenants t ON u.tenants_id = t.id
LEFT JOIN businesses b ON u.business_id = b.id
LEFT JOIN branches br ON u.branch_id = br.id
WHERE u.tenants_id = $1 AND u.is_active = true
ORDER BY u.created_at DESC
LIMIT $2 OFFSET $3;

-- name: ListBusinessUsers :many
SELECT u.*, t.organization as tenant_name, b.name as business_name, br.name as branch_name
FROM users u
JOIN tenants t ON u.tenants_id = t.id
LEFT JOIN businesses b ON u.business_id = b.id
LEFT JOIN branches br ON u.branch_id = br.id
WHERE u.tenants_id = $1 AND u.business_id = $2 AND u.is_active = true
ORDER BY u.created_at DESC;

-- name: ListBranchUsers :many
SELECT u.*, t.organization as tenant_name, b.name as business_name, br.name as branch_name
FROM users u
JOIN tenants t ON u.tenants_id = t.id
LEFT JOIN businesses b ON u.business_id = b.id
LEFT JOIN branches br ON u.branch_id = br.id
WHERE u.tenants_id = $1 AND u.branch_id = $2 AND u.is_active = true
ORDER BY u.created_at DESC;

-- name: UpdateUser :one
UPDATE users
SET first_name = COALESCE(sqlc.narg('first_name'), first_name),
    last_name = COALESCE(sqlc.narg('last_name'), last_name),
    email = COALESCE(sqlc.narg('email'), email),
    phone = COALESCE(sqlc.narg('phone'), phone),
    business_id = COALESCE(sqlc.narg('business_id'), business_id),
    branch_id = COALESCE(sqlc.narg('branch_id'), branch_id),
    avatar_url = COALESCE(sqlc.narg('avatar_url'), avatar_url),
    metadata = COALESCE(sqlc.narg('metadata'), metadata),
    updated_at = CURRENT_TIMESTAMP
WHERE id = sqlc.arg('id') AND tenants_id = sqlc.arg('tenant_id')
RETURNING *;

-- name: UpdateUserLastLogin :exec
UPDATE users
SET last_login_at = CURRENT_TIMESTAMP
WHERE id = $1;

-- name: DeactivateUser :one
UPDATE users
SET is_active = false, updated_at = CURRENT_TIMESTAMP
WHERE id = $1 AND tenants_id = $2
RETURNING *;

-- name: CountTenantUsers :one
SELECT COUNT(*) FROM users
WHERE tenants_id = $1 AND is_active = true;

-- ========================================
-- AUDIT LOG QUERIES
-- ========================================

-- name: CreateAuditLog :one
INSERT INTO rbac_audit_log (
    action, actor_id, target_user_id, target_role_id, permission_id,
    old_values, new_values, reason, ip_address, user_agent, tenant_id
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11
) RETURNING *;

-- name: GetAuditLogs :many
SELECT al.*,
       a.email as actor_email, a.first_name as actor_first_name, a.last_name as actor_last_name,
       tu.email as target_user_email, tu.first_name as target_first_name, tu.last_name as target_last_name,
       tr.name as target_role_name,
       p.code as permission_code
FROM rbac_audit_log al
LEFT JOIN users a ON al.actor_id = a.id
LEFT JOIN users tu ON al.target_user_id = tu.id
LEFT JOIN tenant_roles tr ON al.target_role_id = tr.id
LEFT JOIN permissions p ON al.permission_id = p.id
WHERE al.tenant_id = $1
ORDER BY al.created_at DESC
LIMIT $2 OFFSET $3;

-- name: GetUserAuditLogs :many
SELECT al.*,
       a.email as actor_email, a.first_name as actor_first_name, a.last_name as actor_last_name,
       tr.name as target_role_name,
       p.code as permission_code
FROM rbac_audit_log al
LEFT JOIN users a ON al.actor_id = a.id
LEFT JOIN tenant_roles tr ON al.target_role_id = tr.id
LEFT JOIN permissions p ON al.permission_id = p.id
WHERE al.target_user_id = $1 AND al.tenant_id = $2
ORDER BY al.created_at DESC
LIMIT $3 OFFSET $4;
