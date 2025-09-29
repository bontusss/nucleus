-- name: CreateBusiness :one
INSERT INTO businesses (
    owner_id, name, motto, email, website, tax_id, vat_number,
    country, logo_url, metadata
) VALUES (
    $1, $2, $3, $4, $5, $6, $7,
    $8, $9, $10
) RETURNING *;

-- name: GetBusiness :one
SELECT *
FROM businesses
WHERE id = $1 AND owner_id = $2;

-- name: ListBusinesses :many
SELECT *
FROM businesses
WHERE owner_id = $1
ORDER BY created_at;

-- name: UpdateBusiness :one
UPDATE businesses SET
    name = COALESCE(sqlc.narg(name), name),
    motto = COALESCE(sqlc.narg(motto), motto),
    email = COALESCE(sqlc.narg(email), email),
    website = COALESCE(sqlc.narg(website), website),
    tax_id = COALESCE(sqlc.narg(tax_id), tax_id),
    vat_number = COALESCE(sqlc.narg(vat_number), vat_number),
    logo_url = COALESCE(sqlc.narg(logo_url), logo_url),
    country = COALESCE(sqlc.narg(country), country),
    metadata = COALESCE(sqlc.narg(metadata), metadata),
    updated_at = CURRENT_TIMESTAMP
WHERE id = sqlc.arg(id) AND owner_id = sqlc.arg(owner_id)
RETURNING *;

-- name: DeleteBusiness :one
DELETE FROM businesses
WHERE id = $1 AND owner_id = $2
RETURNING *;


-- name: CreateBranch :one
INSERT INTO branches (
    business_id, name, address, phone, email, metadata, is_active
) VALUES (
    $1, $2, $3, $4, $5, $6, $7
) RETURNING *;

-- name: GetBranch :one
SELECT * FROM branches
WHERE id = $1;

-- name: ListBranches :many
SELECT * FROM branches
WHERE business_id = $1
ORDER BY created_at DESC;

-- name: DeleteBranch :one
DELETE FROM branches
WHERE id = $1
RETURNING *;

-- name: UpdateBranch :one
UPDATE branches SET
    name = COALESCE(sqlc.narg(name), name),
    address = COALESCE(sqlc.narg(address), address),
    phone = COALESCE(sqlc.narg(phone), phone),
    email = COALESCE(sqlc.narg(email), email),
    metadata = COALESCE(sqlc.narg(metadata), metadata),
    is_active = COALESCE(sqlc.narg(is_active), is_active),
    updated_at = CURRENT_TIMESTAMP
WHERE id = sqlc.arg(id)
RETURNING *;

-- Taxes
-- -- name: CreateTax :one
INSERT INTO taxes (
    business_id, name, rate, code, is_active, note, metadata
) VALUES (
    $1, $2, $3, $4, COALESCE(sqlc.narg(is_active), TRUE), sqlc.narg(note), sqlc.narg(metadata)
)
RETURNING *;

-- name: GetTaxByID :one
SELECT * FROM taxes
WHERE id = $1;

-- name: ListTaxes :many
SELECT * FROM taxes
WHERE business_id = $1
ORDER BY created_at DESC;

-- name: UpdateTax :one
UPDATE taxes
SET
    name = COALESCE(sqlc.narg(name), name),
    rate = COALESCE(sqlc.narg(rate), rate),
    code = COALESCE(sqlc.narg(code), code),
    is_active = COALESCE(sqlc.narg(is_active), is_active),
    note = COALESCE(sqlc.narg(note), note),
    metadata = COALESCE(sqlc.narg(metadata), metadata),
    updated_at = CURRENT_TIMESTAMP
WHERE id = sqlc.arg(id) AND business_id = sqlc.arg(business_id)
RETURNING *;

-- name: DeleteTax :exec
DELETE FROM taxes
WHERE id = $1;
