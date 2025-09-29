-- Webhooks table
CREATE TABLE webhooks (
    id SERIAL PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    description TEXT,
    url VARCHAR(500) NOT NULL,
    secret VARCHAR(64) NOT NULL, -- For signing webhook payloads
    events TEXT[] NOT NULL DEFAULT '{}', -- Array of event types to listen for
    user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    is_active BOOLEAN DEFAULT TRUE,
    retry_count INTEGER DEFAULT 0,
    max_retries INTEGER DEFAULT 3,
    timeout_ms INTEGER DEFAULT 5000, -- 5 seconds default timeout
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Webhook deliveries table (for tracking sent webhooks)
CREATE TABLE webhook_deliveries (
    id SERIAL PRIMARY KEY,
    webhook_id INTEGER NOT NULL REFERENCES webhooks(id) ON DELETE CASCADE,
    event_type VARCHAR(100) NOT NULL,
    payload JSONB NOT NULL,
    response_status INTEGER,
    response_body TEXT,
    error_message TEXT,
    attempt_number INTEGER DEFAULT 1,
    delivered_at TIMESTAMP,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Webhook events registry
CREATE TABLE webhook_events (
    id SERIAL PRIMARY KEY,
    event_type VARCHAR(100) UNIQUE NOT NULL,
    description TEXT,
    module VARCHAR(50) NOT NULL, -- Which module emits this event
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Insert common webhook events
INSERT INTO webhook_events (event_type, description, module) VALUES
-- Auth events
('user.created', 'User account created', 'auth'),
('user.updated', 'User account updated', 'auth'),
('user.deleted', 'User account deleted', 'auth'),
('user.logged_in', 'User logged in', 'auth'),
('user.logged_out', 'User logged out', 'auth'),

-- POS events
('sale.created', 'New sale created', 'pos'),
('sale.updated', 'Sale updated', 'pos'),
('sale.refunded', 'Sale refunded', 'pos'),
('sale.cancelled', 'Sale cancelled', 'pos'),

-- Inventory events
('inventory.low_stock', 'Inventory item below reorder level', 'inventory'),
('inventory.out_of_stock', 'Inventory item out of stock', 'inventory'),
('inventory.adjusted', 'Stock level adjusted', 'inventory'),
('inventory.transfer_created', 'Stock transfer created', 'inventory'),
('inventory.transfer_completed', 'Stock transfer completed', 'inventory'),

-- System events
('api_key.created', 'API key created', 'system'),
('api_key.revoked', 'API key revoked', 'system'),
('settings.updated', 'System settings updated', 'system');

-- Add permissions for webhook management
INSERT INTO permissions (code, description) VALUES
('webhooks:manage', 'Manage webhooks'),
('webhooks:view', 'View webhooks and deliveries');

-- Grant permissions to admin role
INSERT INTO role_permissions (role_id, permission_id) VALUES
(1, 16), (1, 17); -- admin gets webhook permissions
