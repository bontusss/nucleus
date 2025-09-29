CREATE TABLE suppliers(
    id SERIAL PRIMARY KEY,
    name VARCHAR(100) NOT NULL UNIQUE,
    phone VARCHAR(50),
    email VARCHAR(50),
    address VARCHAR(100),
    metadata JSONB DEFAULT '{}',
    business_id INTEGER NOT NULL REFERENCES businesses(id),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- A purchase order will be tied to a branch.
CREATE TABLE purchases (
    id SERIAL PRIMARY KEY,
    supplier_id INTEGER NOT NULL REFERENCES suppliers(id),
    branch_id INTEGER NOT NULL REFERENCES branches(id),
    invoice_number VARCHAR(20),
    purchase_order_number VARCHAR(20) NOT NULL,
    total_amount DECIMAL(12,2),
    note TEXT,
    status VARCHAR(20) NOT NULL CHECK(status IN ('pending', 'ordered', 'delivered', 'cancelled')),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE purchase_items (
    id SERIAL PRIMARY KEY,
    product_id INTEGER NOT NULL REFERENCES variations(id),
    purchase_id INTEGER NOT NULL REFERENCES purchases(id),
    quantity INTEGER NOT NULL,
    unit_price DECIMAL(12,2) NOT NULL,
    tax_id INTEGER REFERENCES taxes(id),
    total_price DECIMAL(12,2) NOT NULL,
    metadata JSONB
);

CREATE INDEX idx_purchase_supplier ON purchases(supplier_id);
CREATE INDEX idx_purchase_branch ON purchases(branch_id);
CREATE INDEX idx_purchase_items_purchase ON purchase_items(purchase_id);
