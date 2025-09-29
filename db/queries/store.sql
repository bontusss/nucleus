-- name: CreateStore :one
INSERT INTO stores (
    name, description, branch_id, address, phone, email,
    is_active, store_code, store_type, assigned_user, manager_id, metadata
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12
)
RETURNING *;

-- name: GetStoreByID :one
SELECT * FROM stores WHERE id = $1 LIMIT 1;

-- name: GetStoresByBranch :many
SELECT * FROM stores WHERE branch_id = $1 ORDER BY name;

-- name: GetCentralStoreByBranch :one
SELECT * FROM stores WHERE branch_id = $1 AND store_type = 'central' LIMIT 1;

-- name: ListStores :many
SELECT * FROM stores ORDER BY created_at DESC;

-- name: UpdateStore :one
UPDATE stores
SET name = COALESCE(sqlc.narg(name), name),
    description = COALESCE(sqlc.narg(description), description),
    address = COALESCE(sqlc.narg(address), address),
    phone = COALESCE(sqlc.narg(phone), phone),
    email = COALESCE(sqlc.narg(email), email),
    is_active = COALESCE(sqlc.narg(is_active), is_active),
    store_code = COALESCE(sqlc.narg(store_code), store_code),
    store_type = COALESCE(sqlc.narg(store_type), store_type),
    assigned_user = COALESCE(sqlc.narg(assigned_user), assigned_user),
    manager_id = COALESCE(sqlc.narg(manager_id), manager_id),
    metadata = COALESCE(sqlc.narg(metadata), metadata),
    updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: DeleteStore :exec
DELETE FROM stores WHERE id = $1;

-- name: DeactivateStore :one
UPDATE stores
SET is_active = FALSE,
    updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: SearchStoresByName :many
SELECT * FROM stores
WHERE name ILIKE '%' || $1 || '%'
ORDER BY name;
