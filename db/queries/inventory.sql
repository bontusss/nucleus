-- Brand
-- name: CreateBrand :one
INSERT INTO brands (name, description, logo, business_id)
VALUES ($1, $2, $3, $4)
RETURNING *;

-- name: GetBrand :one
SELECT * FROM brands
WHERE id = $1
LIMIT 1;

-- name: ListBrandsByBusiness :many
SELECT * FROM brands
WHERE business_id = $1
ORDER BY name;

-- name: UpdateBrand :one
UPDATE brands
SET name = $2,
    description = COALESCE(sqlc.narg(description), description),
    logo = COALESCE(sqlc.narg(logo), logo),
    is_active = COALESCE(sqlc.narg(is_active), is_active),
    updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: DeleteBrand :one
DELETE FROM brands
WHERE id = $1
RETURNING *;


-- Category
-- name: CreateCategory :one
INSERT INTO categories (name, parent_id, description, business_id)
VALUES ($1, $2, $3, $4)
RETURNING *;

-- name: GetCategory :one
SELECT * FROM categories
WHERE id = $1
LIMIT 1;

-- name: ListCategoriesByBusiness :many
SELECT * FROM categories
WHERE business_id = $1
ORDER BY name;

-- name: UpdateCategory :one
UPDATE categories
SET name = $2,
    parent_id = COALESCE(sqlc.narg(parent_id), parent_id),
    description = COALESCE(sqlc.narg(description), description),
    is_active = COALESCE(sqlc.narg(is_active), is_active),
    updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: DeleteCategory :exec
DELETE FROM categories
WHERE id = $1
RETURNING *;


-- Item
-- name: CreateItem :one
INSERT INTO items (brand_id, category_id, name, description, item_type, business_id)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING *;

-- name: UpdateItem :one
UPDATE items
SET brand_id = COALESCE(sqlc.narg(brand_id), brand_id),
    category_id = COALESCE(sqlc.narg(category_id), category_id),
    name = COALESCE(sqlc.narg(name), name),
    description = COALESCE(sqlc.narg(description), description),
    is_active = COALESCE(sqlc.narg(is_active), is_active),
    item_type = COALESCE(sqlc.narg(item_type), item_type),
    updated_at = NOW()
WHERE id = $1 AND business_id = $2
RETURNING *;

-- name: GetItem :one
SELECT * FROM items
WHERE id = $1
LIMIT 1;

-- name: ListItemsByBusiness :many
SELECT * FROM items
WHERE business_id = $1
ORDER BY name;

-- name: ListItemsByCategory :many
SELECT * FROM items
WHERE category_id = $1
ORDER BY name;

-- name: DeleteItem :one
DELETE FROM items
WHERE id = $1
RETURNING *;


-- Variation
-- name: CreateVariation :one
INSERT INTO variations (
item_id, sku, name, unit_id, size, color_id, barcode,cost_price, base_price, reorder_level, metadata, is_active)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
RETURNING *;

-- name: UpdateVariation :one
UPDATE variations
SET sku = COALESCE(sqlc.narg(sku), sku),
    name = COALESCE(sqlc.narg(name), name),
    unit_id = COALESCE(sqlc.narg(unit_id), unit_id),
    size = COALESCE(sqlc.narg(size), size),
    color_id = COALESCE(sqlc.narg(color_id), color_id),
    barcode = COALESCE(sqlc.narg(barcode), barcode),
    base_price = COALESCE(sqlc.narg(base_price), base_price),
    reorder_level = COALESCE(sqlc.narg(reorder_level), reorder_level),
    is_active = COALESCE(sqlc.narg(is_active), is_active),
    updated_at = NOW()
WHERE id = $1 AND item_id = $2
RETURNING *;


-- name: GetVariation :one
SELECT * FROM variations
WHERE id = $1
LIMIT 1;

-- name: ListVariationsByItem :many
SELECT * FROM variations
WHERE item_id = $1
ORDER BY name;

-- name: DeleteVariation :one
DELETE FROM variations
WHERE id = $1
RETURNING *;


-- Image
-- name: CreateItemImage :one
INSERT INTO item_images (item_id, variation_id, url, is_primary)
VALUES ($1, $2, $3, $4)
RETURNING *;

-- name: GetItemImagesByItem :many
SELECT * FROM item_images WHERE item_id = $1;

-- name: GetItemImagesByVariation :many
SELECT * FROM item_images WHERE variation_id = $1;

-- name: DeleteItemImage :exec
DELETE FROM item_images WHERE id = $1;


-- Inventory
-- name: UpsertInventory :one
-- name: UpsertInventory :one
INSERT INTO inventories (store_id, variation_id, quantity)
VALUES ($1, $2, $3)
ON CONFLICT (store_id, variation_id)
DO UPDATE SET
    quantity = EXCLUDED.quantity,
    last_updated = NOW()
RETURNING *;

-- name: UpdateInventoryQuantity :one
UPDATE inventories
SET quantity = $3,
    last_updated = NOW()
WHERE store_id = $1 AND variation_id = $2
RETURNING *;


-- name: GetInventoryByStore :many
SELECT * FROM inventories WHERE store_id = $1;

-- name: GetInventoryItem :one
SELECT * FROM inventories
WHERE store_id = $1 AND variation_id = $2
LIMIT 1;

-- name: DeleteInventory :exec
DELETE FROM inventories WHERE id = $1;

-- Units
-- name: CreateUnit :one
INSERT INTO units (name, short_code)
VALUES ($1, $2)
RETURNING *;

-- name: GetUnitByID :one
SELECT * FROM units
WHERE id = $1;

-- name: ListUnits :many
SELECT * FROM units
ORDER BY id;

-- name: UpdateUnit :one
UPDATE units
SET name = $1,
    short_code = $2,
    updated_at = CURRENT_TIMESTAMP
WHERE id = $3
RETURNING *;

-- name: DeleteUnit :exec
DELETE FROM units
WHERE id = $1
RETURNING *;


-- Color
-- name: CreateColor :one
INSERT INTO colors (name)
VALUES ($1)
RETURNING *;

-- name: GetColorByID :one
SELECT * FROM colors
WHERE id = $1;

-- name: GetColorByName :one
SELECT * FROM colors
WHERE name = $1;

-- name: ListColors :many
SELECT * FROM colors
ORDER BY id;

-- name: UpdateColor :one
UPDATE colors
SET name = $1,
    updated_at = CURRENT_TIMESTAMP
WHERE id = $2
RETURNING *;

-- name: DeleteColor :exec
DELETE FROM colors
WHERE id = $1
RETURNING *;
