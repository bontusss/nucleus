/*
======================================================
HERP INVENTORY & POS SCHEMA
======================================================

Hierarchy:
-----------
 Brand:       Coca-Cola, Johnnie Walker, Samsung
 Category:    Drinks → Soft Drinks → Cola
 Item:        Coca-Cola
 Variation:   Coca-Cola 500ml Bottle, Coca-Cola 1L Bottle
 Inventory:   Branch 1 Central Store has 50 of Coca-Cola 500ml

Diagram:
----------
 Brand ──┐
         │
 Category ──┐
            │
         Item ──> Variation ──> Inventory (per store)
            │         │
            │         └─> Store-specific pricing
            │
            └─> Item images (either item-level or variation-level)

Notes:
- Every Item MUST have at least one Variation (even if “default”).
- Inventory is always tracked at the Variation level.
- Prices can be set globally (base_price) or per store (store_price).
- item_type defines behavior in ERP (for sale, consumable, raw material, fixed).

======================================================
*/

-- Brand: Coca-Cola, Johnnie Walker, Samsung.
CREATE TABLE brands (
    id SERIAL PRIMARY KEY,
    name VARCHAR(100) NOT NULL UNIQUE,
    business_id INTEGER NOT NULL REFERENCES businesses(id),
    description TEXT,
    logo VARCHAR(255),
    is_active BOOLEAN DEFAULT TRUE,
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Category: Drinks → Soft Drinks → Cola.
CREATE TABLE categories (
    id SERIAL PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    business_id INTEGER NOT NULL REFERENCES businesses(id),
    parent_id INT REFERENCES categories(id) ON DELETE SET NULL,
    description TEXT,
    is_active BOOLEAN DEFAULT TRUE,
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE (name, parent_id) -- avoids duplicate names under same parent
);

-- Item: Coca-Cola (abstract, not directly stocked).
CREATE TABLE items (
    id SERIAL PRIMARY KEY,
    business_id INTEGER NOT NULL REFERENCES businesses(id),
    brand_id INT REFERENCES brands(id) ON DELETE SET NULL,
    category_id INT NOT NULL REFERENCES categories(id) ON DELETE SET NULL,
    name VARCHAR(150) NOT NULL,
    description TEXT,
    item_type VARCHAR(20) NOT NULL CHECK(item_type IN ('fixed', 'consumable', 'raw_material', 'for_sale')),
    -- Defines business rules:
    -- 'fixed'       = Furniture, AC units (not stock-tracked, not sold)
    -- 'consumable'  = Internal use items (soap, cleaning supplies)
    -- 'raw_material'= Ingredients (flour, sugar) for production
    -- 'for_sale'    = Products sold to customers (drinks, gadgets)
    is_active BOOLEAN DEFAULT TRUE,
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Supporting tables for standard GENERATED ALWAYS AS (quantity * unit_price) STOREDized attributes
CREATE TABLE units (
    id SERIAL PRIMARY KEY,
    business_id INTEGER NOT NULL REFERENCES businesses(id),
    name VARCHAR(20) NOT NULL,  -- e.g. Kilogram, Liter, Piece
    short_code VARCHAR(10),     -- e.g. kg, L, pcs
    metadata JSONB DEFAULT '{}',
    is_active BOOLEAN,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE unit_conversions (
    id SERIAL PRIMARY KEY,
    business_id INTEGER NOT NULL REFERENCES businesses(id),
    from_unit_id INT NOT NULL REFERENCES units(id),
    to_unit_id INT NOT NULL REFERENCES units(id),
    factor NUMERIC(12,4) NOT NULL -- e.g., 1 carton = 12 pieces → 12
);

CREATE TABLE colors (
    id SERIAL PRIMARY KEY,
    business_id INTEGER NOT NULL REFERENCES businesses(id),
    name VARCHAR(20) NOT NULL,  -- e.g. Red, Blue, Black
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Variation: Coca-Cola 500ml Bottle, Coca-Cola 1L Bottle.
CREATE TABLE variations (
    id SERIAL PRIMARY KEY,
    item_id INT NOT NULL REFERENCES items(id) ON DELETE CASCADE,
    sku VARCHAR(50) NOT NULL UNIQUE,   -- Stock Keeping Unit
    name VARCHAR(100) NOT NULL,        -- e.g. '500ml Bottle'
    unit_id INT NOT NULL REFERENCES units(id) ON DELETE SET NULL,
    size VARCHAR(50),                  -- e.g. '500', 'Large'
    color_id INT REFERENCES colors(id) ON DELETE SET NULL,
    barcode VARCHAR(50) UNIQUE,        -- Retail barcode
    cost_price DECIMAL(12,2),
    base_price DECIMAL(12,2) NOT NULL, -- Default/global price
    reorder_level INT DEFAULT 5,       -- Minimum stock before reordering
    is_active BOOLEAN DEFAULT TRUE,
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE variation_taxes (
    id SERIAL PRIMARY KEY,
    variation_id INT NOT NULL REFERENCES variations(id) ON DELETE CASCADE,
    tax_id INT NOT NULL REFERENCES taxes(id) ON DELETE CASCADE,
    UNIQUE (variation_id, tax_id)
);

-- Store-specific pricing (overrides base_price if exists)
CREATE TABLE store_prices (
    id SERIAL PRIMARY KEY,
    store_id INT NOT NULL REFERENCES stores(id) ON DELETE CASCADE,
    variation_id INT NOT NULL REFERENCES variations(id) ON DELETE CASCADE,
    price NUMERIC(12,2) NOT NULL,
    metadata JSONB DEFAULT '{}',
    UNIQUE (store_id, variation_id)
);

-- Images: can belong to either Item or Variation (not both).
CREATE TABLE item_images (
    id SERIAL PRIMARY KEY,
    item_id INT REFERENCES items(id) ON DELETE CASCADE,
    variation_id INT REFERENCES variations(id) ON DELETE CASCADE,
    url TEXT NOT NULL,
    is_primary BOOLEAN DEFAULT FALSE,
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    -- Enforce either item_id OR variation_id, never both
    CONSTRAINT item_or_variation CHECK (
        (item_id IS NOT NULL AND variation_id IS NULL)
        OR (item_id IS NULL AND variation_id IS NOT NULL)
    )
);

-- Inventory: “Branch 1 Central Store has 50 of Coca-Cola 500ml.”
CREATE TABLE inventories (
    id SERIAL PRIMARY KEY,
    store_id INT NOT NULL REFERENCES stores(id) ON DELETE CASCADE,
    variation_id INT NOT NULL REFERENCES variations(id) ON DELETE CASCADE,
    quantity INT NOT NULL DEFAULT 0,
    metadata JSONB DEFAULT '{}',
    last_updated TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE (store_id, variation_id)
);

-- Brand search
CREATE INDEX idx_brand_name ON brands(name);

-- Category search
CREATE INDEX idx_category_name ON categories(name);

-- Item search (by name, type, brand, category)
CREATE INDEX idx_item_name ON items(name);
CREATE INDEX idx_item_type ON items(item_type);
CREATE INDEX idx_item_brand_id ON items(brand_id);
CREATE INDEX idx_item_category_id ON items(category_id);

-- Variation search (by SKU, barcode, name)
CREATE INDEX idx_variation_sku ON variations(sku);
CREATE INDEX idx_variation_barcode ON variations(barcode);
CREATE INDEX idx_variation_name ON variations(name);

-- Store price lookup
CREATE INDEX idx_store_price_store_variation ON store_prices(store_id, variation_id);

-- Inventory lookup (store + variation)
CREATE INDEX idx_inventory_store_variation ON inventories(store_id, variation_id);


/*
======================================================
USAGE EXAMPLES
======================================================

1. Adding a new item without variations:
   - Insert into item (name='Coca-Cola', item_type='for_sale')
   - Auto-generate default variation (is_default = TRUE)

2. Inventory:
   - Variation “Coca-Cola 500ml Bottle”
   - Store “Hotel Bar”
   - Inventory row: store_id=1, variation_id=101, quantity=50

3. Pricing:
   - Global price = 200 (base_price in variation)
   - Store A overrides to 220 in store_price
   - Store B uses default 200

======================================================
*/
