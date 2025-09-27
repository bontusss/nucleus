-- Brand
-- name: CreateBrand :one
INSERT INTO brand (name, description, logo)
VALUES ($1, $2, $3)
RETURNING *;

-- name: GetBrand :one
SELECT * FROM brand WHERE id = $1 LIMIT 1;

-- name: ListBrands :many
SELECT * FROM brand ORDER BY name;

-- name: UpdateBrand :one
UPDATE brand
SET name = $2,
    description = $3,
    logo = $4,
    is_active = $5,
    updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: DeleteBrand :exec
DELETE FROM brand WHERE id = $1;


-- Category
-- name: CreateCategory :one
INSERT INTO category (name, parent_id, description)
VALUES ($1, $2, $3)
RETURNING *;

-- name: GetCategory :one
SELECT * FROM category WHERE id = $1 LIMIT 1;

-- name: ListCategories :many
SELECT * FROM category ORDER BY name;

-- name: UpdateCategory :one
UPDATE category
SET name = $2,
    parent_id = $3,
    description = $4,
    is_active = $5,
    updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: DeleteCategory :exec
DELETE FROM category WHERE id = $1;


-- Item
-- name: CreateItem :one
INSERT INTO item (brand_id, category_id, name, description, no_variants, item_type)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING *;

-- name: UpdateItem :one
UPDATE item
SET brand_id = COALESCE(sqlc.narg(brand_id), brand_id),
    category_id = COALESCE(sqlc.narg(category_id), category_id),
    name = COALESCE(sqlc.narg(name), name),
    description = COALESCE(sqlc.narg(description), description),
    no_variants = COALESCE(sqlc.narg(no_variants), no_variants),
    is_active = COALESCE(sqlc.narg(is_active), is_active),
    item_type = COALESCE(sqlc.narg(item_type), item_type),
    updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: GetItem :one
SELECT * FROM item WHERE id = $1 LIMIT 1;

-- name: ListItems :many
SELECT * FROM item ORDER BY name;

-- name: ListItemsByCategory :many
SELECT * FROM item WHERE category_id = $1 ORDER BY name;

-- name: DeleteItem :exec
DELETE FROM item WHERE id = $1;


-- Variation
-- name: CreateVariation :one
INSERT INTO variation (item_id, sku, name, unit_id, size, color_id, barcode, base_price, reorder_level, is_default)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
RETURNING *;

-- name: UpdateVariation :one
UPDATE variation
SET sku = $2,
    name = $3,
    unit_id = $4,
    size = $5,
    color_id = $6,
    barcode = $7,
    base_price = $8,
    reorder_level = $9,
    is_default = $10,
    is_active = $11,
    updated_at = NOW()
WHERE id = $1
RETURNING *;


-- name: GetVariation :one
SELECT * FROM variation WHERE id = $1 LIMIT 1;

-- name: ListVariationsByItem :many
SELECT * FROM variation WHERE item_id = $1 ORDER BY name;

-- name: DeleteVariation :exec
DELETE FROM variation WHERE id = $1;


-- Image
-- name: CreateItemImage :one
INSERT INTO item_image (item_id, variation_id, url, is_primary)
VALUES ($1, $2, $3, $4)
RETURNING *;

-- name: GetItemImagesByItem :many
SELECT * FROM item_image WHERE item_id = $1;

-- name: GetItemImagesByVariation :many
SELECT * FROM item_image WHERE variation_id = $1;

-- name: DeleteItemImage :exec
DELETE FROM item_image WHERE id = $1;


-- Inventory
-- name: UpsertInventory :one
-- name: UpsertInventory :one
INSERT INTO inventory (store_id, variation_id, quantity)
VALUES ($1, $2, $3)
ON CONFLICT (store_id, variation_id)
DO UPDATE SET
    quantity = EXCLUDED.quantity,
    last_updated = NOW()
RETURNING *;

-- name: UpdateInventoryQuantity :one
UPDATE inventory
SET quantity = $3,
    last_updated = NOW()
WHERE store_id = $1 AND variation_id = $2
RETURNING *;


-- name: GetInventoryByStore :many
SELECT * FROM inventory WHERE store_id = $1;

-- name: GetInventoryItem :one
SELECT * FROM inventory
WHERE store_id = $1 AND variation_id = $2
LIMIT 1;

-- name: DeleteInventory :exec
DELETE FROM inventory WHERE id = $1;

-- Units
-- name: CreateUnit :one
INSERT INTO unit (name, short_code)
VALUES ($1, $2)
RETURNING *;

-- name: GetUnitByID :one
SELECT * FROM unit
WHERE id = $1;

-- name: ListUnits :many
SELECT * FROM unit
ORDER BY id;

-- name: UpdateUnit :one
UPDATE unit
SET name = $1,
    short_code = $2,
    updated_at = CURRENT_TIMESTAMP
WHERE id = $3
RETURNING *;

-- name: DeleteUnit :exec
DELETE FROM unit
WHERE id = $1
RETURNING *;


-- Color
-- name: CreateColor :one
INSERT INTO color (name)
VALUES ($1)
RETURNING *;

-- name: GetColorByID :one
SELECT * FROM color
WHERE id = $1;

-- name: GetColorByName :one
SELECT * FROM color
WHERE name = $1;

-- name: ListColors :many
SELECT * FROM color
ORDER BY id;

-- name: UpdateColor :one
UPDATE color
SET name = $1,
    updated_at = CURRENT_TIMESTAMP
WHERE id = $2
RETURNING *;

-- name: DeleteColor :exec
DELETE FROM color
WHERE id = $1
RETURNING *;
