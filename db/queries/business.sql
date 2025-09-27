-- name: CreateBusiness :one
INSERT INTO business (
    owner_id, name, motto, email, website, tax_id, tax_rate,
    country, logo_url, rounding, currency, timezone, language,
    low_stock_threshold, allow_overselling, metadata
) VALUES (
    $1, $2, $3, $4, $5, $6, $7,
    $8, $9, $10, $11, $12, $13,
    $14, $15, $16
) RETURNING *;

-- name: GetBusiness :one
SELECT *
FROM business
WHERE id = $1 AND owner_id = $2;

-- name: ListBusinesses :many
SELECT *
FROM business
WHERE owner_id = $1
ORDER BY created_at;

-- name: UpdateBusiness :one
UPDATE business SET
    name = COALESCE(sqlc.narg(name), name),
    motto = COALESCE(sqlc.narg(motto), motto),
    email = COALESCE(sqlc.narg(email), email),
    website = COALESCE(sqlc.narg(website), website),
    tax_id = COALESCE(sqlc.narg(tax_id), tax_id),
    tax_rate = COALESCE(sqlc.narg(tax_rate), tax_rate),
    logo_url = COALESCE(sqlc.narg(logo_url), logo_url),
    rounding = COALESCE(sqlc.narg(rounding), rounding),
    currency = COALESCE(sqlc.narg(currency), currency),
    timezone = COALESCE(sqlc.narg(timezone), timezone),
    language = COALESCE(sqlc.narg(language), language),
    low_stock_threshold = COALESCE(sqlc.narg(low_stock_threshold), low_stock_threshold),
    allow_overselling = COALESCE(sqlc.narg(allow_overselling), allow_overselling),
    country = COALESCE(sqlc.narg(country), country),
    metadata = COALESCE(sqlc.narg(metadata), metadata),
    updated_at = CURRENT_TIMESTAMP
WHERE id = sqlc.arg(id) AND owner_id = sqlc.arg(owner_id)
RETURNING *;

-- name: DeleteBusiness :one
DELETE FROM business
WHERE id = $1 AND owner_id = $2
RETURNING *;


-- name: CreateBranch :one
INSERT INTO branch (
    business_id, name, address_one, address_two, country, phone, email, city, state, zip_code, metadata
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11
) RETURNING *;

-- name: GetBranch :one
SELECT * FROM branch WHERE id = $1;

-- name: ListBranches :many
SELECT * FROM branch ORDER BY created_at DESC;

-- name: DeleteBranch :one
DELETE FROM branch WHERE id = $1
RETURNING *;

-- name: UpdateBranch :one
UPDATE branch SET
    name = COALESCE(sqlc.narg(name), name),
    address_one = COALESCE(sqlc.narg(name), name),
    address_two = COALESCE(sqlc.narg(address_two), address_two),
    country = COALESCE(sqlc.narg(country), country),
    phone = COALESCE(sqlc.narg(phone), phone),
    email = COALESCE(sqlc.narg(email), email),
    city = COALESCE(sqlc.narg(city), city),
    state = COALESCE(sqlc.narg(state), state),
    zip_code = COALESCE(sqlc.narg(zip_code), zip_code),
    metadata = COALESCE(sqlc.narg(metadata), metadata),
    updated_at = CURRENT_TIMESTAMP
WHERE id = $1
RETURNING *;
