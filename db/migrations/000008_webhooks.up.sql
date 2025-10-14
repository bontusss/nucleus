-- Webhooks table
CREATE TABLE webhooks (
    id SERIAL PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    description TEXT,
    url VARCHAR(500) NOT NULL,
    secret VARCHAR(64) NOT NULL, -- For signing webhook payloads
    events TEXT[] NOT NULL DEFAULT '{}', -- Array of event types to listen for
    tenant_id INTEGER NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
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
