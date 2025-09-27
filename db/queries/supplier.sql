-- name: CreateSupplier :one
INSERT INTO suppliers (
    name, phone, email, address, metadata, business_id
) VALUES (
    $1, $2, $3, $4, $5, $6
) RETURNING *;

-- name: GetSupplier :one
SELECT * FROM suppliers WHERE id = $1 AND business_id = $2;

-- name: ListSuppliers :many
SELECT * FROM suppliers WHERE business_id = $1
ORDER BY id DESC;

-- name: DeleteSupplier :one
DELETE FROM suppliers
WHERE id = $1 AND business_id = $2
RETURNING *;

-- name: UpdateSupplier :one
UPDATE suppliers
SET
    name = COALESCE(sqlc.narg('name'), name),
    phone = COALESCE(sqlc.narg('phone'), phone),
    email = COALESCE(sqlc.narg('email'), email),
    address = COALESCE(sqlc.narg('address'), address),
    metadata = COALESCE(sqlc.narg('metadata'), metadata),
    updated_at = NOW()
WHERE id = sqlc.arg('id') AND business_id = sqlc.arg('business_id')
RETURNING *;
