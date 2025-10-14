-- API Keys table
CREATE TABLE api_keys (
    id SERIAL PRIMARY KEY,
    key_name VARCHAR(100) NOT NULL, 
    api_key VARCHAR(64) UNIQUE NOT NULL,
    api_secret VARCHAR(64) NOT NULL, -- For HMAC signing
    tenant_id INTEGER NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    allowed_modules TEXT[] NOT NULL DEFAULT '{}', -- Array of module names
    monthly_limit INTEGER NOT NULL DEFAULT 1000,
    current_month_requests INTEGER NOT NULL DEFAULT 0,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    expires_at TIMESTAMP NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- API Key usage tracking
CREATE TABLE api_key_usages (
    id SERIAL PRIMARY KEY,
    api_key_id INTEGER NOT NULL REFERENCES api_keys(id) ON DELETE CASCADE,
    module_name VARCHAR(50) NOT NULL,
    endpoint VARCHAR(100) NOT NULL,
    request_method VARCHAR(10) NOT NULL,
    request_size INTEGER NOT NULL DEFAULT 0,
    response_status INTEGER NOT NULL,
    ip_address TEXT NOT NULL,
    user_agent TEXT NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Indexes for performance
CREATE INDEX idx_api_keys_key ON api_keys(api_key);
CREATE INDEX idx_api_keys_tenant ON api_keys(tenant_id);
CREATE INDEX idx_api_key_usage_key_date ON api_key_usages(api_key_id, created_at);
CREATE INDEX idx_api_key_usage_month ON api_key_usages(created_at);

-- Add permissions for API key management
INSERT INTO permissions (code, description) VALUES
('api_keys:manage', 'Manage API keys'),
('api_keys:view', 'View API keys and usage');
