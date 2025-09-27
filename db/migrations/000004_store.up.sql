-- Migration to create the store table and related indexes
CREATE TABLE store (
    id SERIAL PRIMARY KEY,
    name VARCHAR(50) NOT NULL UNIQUE,
    description VARCHAR(255), -- describe what this store is for
    branch_id INTEGER NOT NULL,
    address VARCHAR(50) NOT NULL,
    phone VARCHAR(15) NOT NULL,
    email VARCHAR(50) NOT NULL,
    is_active BOOLEAN DEFAULT TRUE,
    store_type VARCHAR(20) NOT NULL CHECK (store_type IN ('central', 'sub-store')),
    store_code VARCHAR(10) NOT NUll UNIQUE,
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    assigned_user INTEGER REFERENCES users(id) ON DELETE SET NULL,
    manager_id INTEGER REFERENCES users(id) ON DELETE SET NULL,
    FOREIGN KEY (branch_id) REFERENCES branch(id) ON DELETE CASCADE
);

-- Ensure only one central store per branch
CREATE UNIQUE INDEX unique_central_store_per_branch
ON store(branch_id)
WHERE store_type = 'central';

-- Indexes for faster lookups
CREATE INDEX idx_store_branch_id ON store (branch_id);
CREATE INDEX idx_store_name ON store (name);
