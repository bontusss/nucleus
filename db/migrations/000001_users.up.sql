-- Roles table
CREATE TABLE roles (
    id SERIAL PRIMARY KEY,
    name VARCHAR(50) UNIQUE NOT NULL,
    description TEXT
);

-- Users table
CREATE TABLE users (
    id SERIAL PRIMARY KEY,
    username VARCHAR(50) NOT NULL UNIQUE,
    first_name VARCHAR(50) NOT NULL,
    last_name VARCHAR(50) NOT NULL,
    email VARCHAR(100) UNIQUE,
    password_hash VARCHAR(255) NOT NULL,
    gender VARCHAR CHECK (gender IN ('male', 'female')),
    role_id INTEGER,
    is_active BOOLEAN DEFAULT TRUE,
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (role_id) REFERENCES roles(id)
);

-- Admin table
CREATE TABLE admins (
    id SERIAL PRIMARY KEY,
    first_name VARCHAR(50) NOT NULL,
    last_name VARCHAR(50) NOT NULL,
    username VARCHAR(50) UNIQUE NOT NULL,
    email VARCHAR(100) UNIQUE NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    role_id INTEGER NOT NULL,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    email_verified BOOLEAN NOT NULL DEFAULT FALSE,
    verification_code TEXT,
    verification_expires_at TIMESTAMP,
    reset_code TEXT,
    reset_code_expires_at TIMESTAMP,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (role_id) REFERENCES roles(id)
);

-- Refresh tokens table
CREATE TABLE refresh_tokens (
    id SERIAL PRIMARY KEY,
    user_id INTEGER NOT NULL,
    token VARCHAR(255) UNIQUE NOT NULL,
    expires_at TIMESTAMP NOT NULL,
    revoked BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);

-- Index for faster lookups
CREATE INDEX idx_refresh_tokens_token ON refresh_tokens(token);
CREATE INDEX idx_refresh_tokens_user_id ON refresh_tokens(user_id);

-- Permissions table
CREATE TABLE permissions (
    id SERIAL PRIMARY KEY,
    code VARCHAR(50) UNIQUE NOT NULL,
    description TEXT
);

-- Role-Permissions junction table
CREATE TABLE role_permissions (
    role_id INTEGER NOT NULL,
    permission_id INTEGER NOT NULL,
    PRIMARY KEY (role_id, permission_id),
    FOREIGN KEY (role_id) REFERENCES roles(id),
    FOREIGN KEY (permission_id) REFERENCES permissions(id)
);

-- Sample data
INSERT INTO roles (name, description) VALUES
('admin', 'System administrator with full access'),
('manager', 'Hotel manager with broad access'),
('cashier', 'Cashier with limited POS access'),
('pos_staff', 'POS system user');

INSERT INTO permissions (code, description) VALUES
-- admin
('admin:manage', 'Manage admin settings'),

-- buisness
('business:create', 'Create business'),
('business:view', 'View business'),
('business:delete', 'Delete business'),
('business:update', 'Update business'),

-- supplier
('supplier:create', 'Create supplier'),
('logs:view', 'View activity logs'),

-- Inventory
('inventory:create', 'Create inventory items'),
('inventory:update', 'Update inventory items'),
('inventory:delete', 'Delete inventory items'),
('inventory:view', 'View inventory items'),


-- sales
-- ('pos:sell', 'Create new sales in POS'),
-- ('sale:view', 'View sales history in POS'),
-- ('sale:manage_items', 'Manage POS items'),
-- ('sale:create', 'Create new sales'),
-- ('sale:update', 'Update sales'),
-- ('sale:delete', 'Delete sales'),
-- ('sale:cancel', 'Cancel sales'),
-- ('sale:refund', 'Refund sales'),
-- ('sale:print', 'Print sales receipts'),


-- users
('user:create', 'Create new users'),
('user:update', 'Update user information'),
('user:delete', 'Delete users'),
('user:view', 'View user information');

-- Role
-- ('role:create', 'Create new roles'),
-- ('role:update', 'Update role information'),
-- ('role:delete', 'Delete roles'),
-- ('role:view', 'View role information'),

-- General settings
-- ('setting:create', 'Create new settings'),
-- ('setting:update', 'Update settings'),
-- ('setting:delete', 'Delete settings'),
-- ('setting:view', 'View settings');

-- Assign permissions to roles
INSERT INTO role_permissions (role_id, permission_id) VALUES
(1, 1), (1, 2), (1, 3), (1, 4), (1, 5), (1, 6), (1, 7), (1, 8), (1, 9), (1, 10), (1, 11), (1, 12), (1, 13), (1, 14), -- admin has all permissions
(2, 2), (2, 3), (2, 4),         -- manager has all POS and booking permissions
(3, 2),                          -- pos_staff can sell and view
(4, 2);                                  -- cashier can only sell

-- Sample user with admin role (role_id = 1)
-- Password: password (hashed with bcrypt)
INSERT INTO admins (username, first_name, last_name, email, password_hash, role_id, is_active)
VALUES (
    'admin',
    'John',
    'Doe',
    'admin@hotel.com',
    '$2a$10$92IXUNpkjO0rOQ5byMi.Ye4oKoEa3Ro9llC/.og/at2.uheWG/igi',
    1,
    true
);

-- Sample user with manager role (role_id = 2)
-- Password: manager123 (hashed with bcrypt)
INSERT INTO users (username, first_name, last_name, email, password_hash, gender, role_id, is_active)
VALUES (
    'manager1',
    'Jane',
    'Smith',
    'manager@hotel.com',
    '$2a$12$6j0pw3VAHhGFlorLUuPo3eOaH52EwMkjXwCusaUkeXAE0sLaRLpv.',
    'female',
    2,
    true
);

-- Sample user with POS staff role (role_id = 3)
-- Password: pos123 (hashed with bcrypt)
INSERT INTO users (username, first_name, last_name, email, password_hash, gender, role_id, is_active)
VALUES (
    'pos_staff1',
    'Mike',
    'Johnson',
    'pos@hotel.com',
    '$2a$12$meqnUUw6yDWSVJRDplVSXOd/FEHqC5xahR.KgLiLjgju2bYr4gfba',
    'male',
    3,
    true
);

-- Sample user with cashier role (role_id = 4)
-- Password: cashier123 (hashed with bcrypt)
INSERT INTO users (username, first_name, last_name, email, password_hash, gender, role_id, is_active)
VALUES (
    'cashier1',
    'Sarah',
    'Wilson',
    'cashier@hotel.com',
    '$2a$12$BcqItePCDoPiZF1LsjIjzOPlYFNdevaHd5k0.0Z6QZOgfY5ib8Jya',
    'female',
    4,
    true
);

-- Additional sample user - inactive user for testing
-- Password: test123 (hashed with bcrypt)
INSERT INTO users (username, first_name, last_name, email, password_hash, gender, role_id, is_active)
VALUES (
    'test_user',
    'Test',
    'User',
    'test@hotel.com',
    '$2a$12$H9GUybbl5PKLbhzhmh176.f7StMp35a3SRvQ9iALihfaUeBhNVHmK',
    'male',
    3,
    false
);
