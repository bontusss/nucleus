-- Tenants table (organizations/companies) 
CREATE TABLE tenants (
    id SERIAL PRIMARY KEY,
    organization VARCHAR(100) NOT NULL,
    email VARCHAR(100) UNIQUE NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    plan VARCHAR(50) NOT NULL DEFAULT 'free',
    email_verified BOOLEAN NOT NULL DEFAULT FALSE,
    verification_code TEXT,
    verification_expires_at TIMESTAMP,
    reset_code TEXT,
    reset_code_expires_at TIMESTAMP,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Create the business table
CREATE TABLE businesses (
    id SERIAL PRIMARY KEY,
    tenants_id INTEGER NOT NULL REFERENCES tenants(id),
    name VARCHAR(50) NOT NULL UNIQUE,
    motto VARCHAR(50),
    email VARCHAR(50) UNIQUE,
    website VARCHAR(20),
    tax_id VARCHAR(100),
    vat_number VARCHAR(100),
    country VARCHAR(100) NOT NULL,
    logo_url VARCHAR(255),
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_business_name ON businesses (name);
CREATE INDEX idx_business_email ON businesses (email);
CREATE INDEX idx_business_tax_id ON businesses (tax_id);
CREATE INDEX idx_business_tenants_id ON businesses (tenants_id);

CREATE TABLE branches (
    id SERIAL PRIMARY KEY,
    business_id INTEGER NOT NULL,
    name VARCHAR(50) NOT NULL,
    address VARCHAR(20),
    phone VARCHAR(15),
    email VARCHAR(20),
    metadata JSONB DEFAULT '{}',
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (business_id) REFERENCES businesses(id) ON DELETE CASCADE
);

CREATE INDEX idx_branch_business_id ON branches (business_id);
CREATE INDEX idx_branch_name ON branches (name);

-- Permissions table (global permissions for all modules)
CREATE TABLE permissions (
    id SERIAL PRIMARY KEY,
    code VARCHAR(50) NOT NULL UNIQUE,
    description TEXT,
    module VARCHAR(50) NOT NULL DEFAULT 'core',
    action VARCHAR(50) NOT NULL DEFAULT 'read',
    resource VARCHAR(50),
    scope VARCHAR(20) NOT NULL DEFAULT 'global' CHECK (scope IN ('global', 'tenant', 'business', 'branch')),
    is_active BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Tenant roles table for hierarchical role management per tenant
CREATE TABLE tenant_roles (
    id SERIAL PRIMARY KEY,
    tenant_id INTEGER NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    name VARCHAR(50) NOT NULL,
    description TEXT,
    parent_role_id INTEGER REFERENCES tenant_roles(id) ON DELETE CASCADE,
    level INTEGER NOT NULL DEFAULT 0, -- 0 = highest level, higher numbers = lower levels
    is_default BOOLEAN DEFAULT FALSE, -- default role for new users
    is_active BOOLEAN DEFAULT TRUE,
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE (tenant_id, name),
    CHECK (level >= 0),
    CHECK (parent_role_id IS NULL OR parent_role_id <> id) -- prevent self-reference
);

-- Create indexes for tenant roles
CREATE INDEX idx_tenant_roles_tenant_id ON tenant_roles(tenant_id);
CREATE INDEX idx_tenant_roles_parent_role_id ON tenant_roles(parent_role_id);
CREATE INDEX idx_tenant_roles_level ON tenant_roles(level);

-- User table with enhanced fields
CREATE TABLE users (
    id SERIAL PRIMARY KEY,
    business_id INTEGER REFERENCES businesses(id) ON DELETE CASCADE,
    tenants_id INTEGER NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    branch_id INTEGER REFERENCES branches(id) ON DELETE CASCADE,
    email VARCHAR(100) NOT NULL UNIQUE,
    password_hash VARCHAR(255) NOT NULL,
    first_name VARCHAR(50),
    last_name VARCHAR(50),
    phone VARCHAR(20),
    avatar_url VARCHAR(255),
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    last_login_at TIMESTAMP,
    email_verified BOOLEAN DEFAULT FALSE,
    verification_code TEXT,
    verification_expires_at TIMESTAMP,
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Add indexes for users table
CREATE INDEX idx_users_tenant_id ON users(tenants_id);
CREATE INDEX idx_users_business_id ON users(business_id);
CREATE INDEX idx_users_branch_id ON users(branch_id);
CREATE INDEX idx_users_email ON users(email);
CREATE INDEX idx_users_is_active ON users(is_active);

-- User role assignments table (many-to-many relationship)
CREATE TABLE user_role_assignments (
    id SERIAL PRIMARY KEY,
    user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    tenant_role_id INTEGER NOT NULL REFERENCES tenant_roles(id) ON DELETE CASCADE,
    business_id INTEGER REFERENCES businesses(id) ON DELETE CASCADE, -- scope to specific business
    branch_id INTEGER REFERENCES branches(id) ON DELETE CASCADE,     -- scope to specific branch
    assigned_by INTEGER REFERENCES users(id), -- who assigned this role
    assigned_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    expires_at TIMESTAMP, -- optional expiration
    is_active BOOLEAN DEFAULT TRUE,
    metadata JSONB DEFAULT '{}'
);

-- Create indexes for user role assignments
CREATE INDEX idx_user_role_assignments_user_id ON user_role_assignments(user_id);
CREATE INDEX idx_user_role_assignments_tenant_role_id ON user_role_assignments(tenant_role_id);
CREATE INDEX idx_user_role_assignments_business_id ON user_role_assignments(business_id);
CREATE INDEX idx_user_role_assignments_branch_id ON user_role_assignments(branch_id);
CREATE UNIQUE INDEX unique_active_user_role_assignments
ON user_role_assignments (
  user_id,
  tenant_role_id,
  COALESCE(business_id, 0),
  COALESCE(branch_id, 0)
)
WHERE is_active = TRUE;



-- Tenant role permissions table
CREATE TABLE tenant_role_permissions (
    id SERIAL PRIMARY KEY,
    tenant_role_id INTEGER NOT NULL REFERENCES tenant_roles(id) ON DELETE CASCADE,
    permission_id INTEGER NOT NULL REFERENCES permissions(id) ON DELETE CASCADE,
    scope_type VARCHAR(20) DEFAULT 'tenant' CHECK (scope_type IN ('global', 'tenant', 'business', 'branch')),
    scope_id INTEGER, -- references tenant, business, or branch id when applicable
    granted_by INTEGER REFERENCES users(id),
    granted_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    is_active BOOLEAN DEFAULT TRUE,
    metadata JSONB DEFAULT '{}',
    UNIQUE (tenant_role_id, permission_id, scope_type, scope_id)
);

-- Create indexes for tenant role permissions
CREATE INDEX idx_tenant_role_permissions_role_id ON tenant_role_permissions(tenant_role_id);
CREATE INDEX idx_tenant_role_permissions_permission_id ON tenant_role_permissions(permission_id);
CREATE INDEX idx_tenant_role_permissions_scope ON tenant_role_permissions(scope_type, scope_id);

-- Legacy business roles table (for backward compatibility)
CREATE TABLE business_roles (
    id SERIAL PRIMARY KEY,
    business_id INTEGER REFERENCES businesses(id) ON DELETE CASCADE,
    name VARCHAR(50) NOT NULL,
    description TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE (business_id, name)
);

-- Legacy role permissions table (for backward compatibility)
CREATE TABLE role_permissions (
    role_id INTEGER NOT NULL,
    permission_id INTEGER NOT NULL,
    business_id INTEGER REFERENCES businesses(id),
    PRIMARY KEY (role_id, permission_id),
    FOREIGN KEY (role_id) REFERENCES business_roles(id),
    FOREIGN KEY (permission_id) REFERENCES permissions(id)
);

-- Insert default global permissions for core modules
INSERT INTO permissions (code, description, module, action, resource, scope) VALUES
-- Core user management permissions
('user:create', 'Create new users', 'user', 'create', 'users', 'tenant'),
('user:read', 'View user information', 'user', 'read', 'users', 'tenant'),
('user:update', 'Update user information', 'user', 'update', 'users', 'tenant'),
('user:delete', 'Delete users', 'user', 'delete', 'users', 'tenant'),
('user:assign_roles', 'Assign roles to users', 'user', 'assign', 'roles', 'tenant'),

-- Role management permissions
('role:create', 'Create new roles', 'role', 'create', 'roles', 'tenant'),
('role:read', 'View role information', 'role', 'read', 'roles', 'tenant'),
('role:update', 'Update role information', 'role', 'update', 'roles', 'tenant'),
('role:delete', 'Delete roles', 'role', 'delete', 'roles', 'tenant'),
('role:assign_permissions', 'Assign permissions to roles', 'role', 'assign', 'permissions', 'tenant'),

-- Business management permissions
('business:create', 'Create new businesses', 'business', 'create', 'businesses', 'tenant'),
('business:read', 'View business information', 'business', 'read', 'businesses', 'tenant'),
('business:update', 'Update business information', 'business', 'update', 'businesses', 'tenant'),
('business:delete', 'Delete businesses', 'business', 'delete', 'businesses', 'tenant'),

-- Branch management permissions
('branch:create', 'Create new branches', 'branch', 'create', 'branches', 'business'),
('branch:read', 'View branch information', 'branch', 'read', 'branches', 'business'),
('branch:update', 'Update branch information', 'branch', 'update', 'branches', 'business'),
('branch:delete', 'Delete branches', 'branch', 'delete', 'branches', 'business'),

-- Inventory permissions
('inventory:create', 'Create inventory items', 'inventory', 'create', 'items', 'business'),
('inventory:read', 'View inventory items', 'inventory', 'read', 'items', 'business'),
('inventory:update', 'Update inventory items', 'inventory', 'update', 'items', 'business'),
('inventory:delete', 'Delete inventory items', 'inventory', 'delete', 'items', 'business'),

-- POS permissions
('pos:create', 'Create sales transactions', 'pos', 'create', 'sales', 'branch'),
('pos:read', 'View sales transactions', 'pos', 'read', 'sales', 'branch'),
('pos:update', 'Update sales transactions', 'pos', 'update', 'sales', 'branch'),
('pos:delete', 'Delete sales transactions', 'pos', 'delete', 'sales', 'branch'),

-- Analytics and reporting
('analytics:read', 'View analytics and reports', 'analytics', 'read', 'reports', 'business'),
('analytics:export', 'Export analytics data', 'analytics', 'export', 'reports', 'business'),

-- System administration (tenant level)
('tenant:manage', 'Manage tenant settings', 'tenant', 'manage', 'settings', 'tenant'),
('tenant:billing', 'View and manage billing', 'tenant', 'manage', 'billing', 'tenant'),
('logs:read', 'View system logs', 'logs', 'read', 'logs', 'tenant')
ON CONFLICT (code) DO NOTHING;

-- Create a function to ensure role hierarchy integrity
CREATE OR REPLACE FUNCTION check_role_hierarchy() RETURNS TRIGGER AS $$
BEGIN
    -- Prevent circular references in role hierarchy
    IF NEW.parent_role_id IS NOT NULL THEN
        -- Check if creating this relationship would create a cycle
        WITH RECURSIVE role_ancestors AS (
            SELECT id, parent_role_id, 1 as level
            FROM tenant_roles
            WHERE id = NEW.parent_role_id

            UNION ALL

            SELECT tr.id, tr.parent_role_id, ra.level + 1
            FROM tenant_roles tr
            JOIN role_ancestors ra ON tr.id = ra.parent_role_id
            WHERE ra.level < 10 -- prevent infinite recursion
        )
        SELECT INTO NEW.level COALESCE(MAX(level), 0) + 1
        FROM role_ancestors;

        -- Check for circular reference
        IF EXISTS (
            WITH RECURSIVE role_ancestors AS (
                SELECT id, parent_role_id
                FROM tenant_roles
                WHERE id = NEW.parent_role_id

                UNION ALL

                SELECT tr.id, tr.parent_role_id
                FROM tenant_roles tr
                JOIN role_ancestors ra ON tr.id = ra.parent_role_id
            )
            SELECT 1 FROM role_ancestors WHERE id = NEW.id
        ) THEN
            RAISE EXCEPTION 'Circular reference detected in role hierarchy';
        END IF;
    ELSE
        NEW.level := 0; -- Root level role
    END IF;

    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Create trigger for role hierarchy validation
CREATE TRIGGER trigger_check_role_hierarchy
    BEFORE INSERT OR UPDATE ON tenant_roles
    FOR EACH ROW
    EXECUTE FUNCTION check_role_hierarchy();

-- Create function to get all user permissions (including inherited from role hierarchy)
CREATE OR REPLACE FUNCTION get_user_permissions(user_id_param INTEGER)
RETURNS TABLE (
    permission_code VARCHAR(50),
    permission_description TEXT,
    module VARCHAR(50),
    action VARCHAR(50),
    resource VARCHAR(50),
    scope VARCHAR(20),
    scope_id INTEGER,
    role_name VARCHAR(50)
) AS $$
BEGIN
    RETURN QUERY
    WITH RECURSIVE role_hierarchy AS (
        -- Get direct role assignments for the user
        SELECT DISTINCT tr.id as role_id, tr.name as role_name, tr.parent_role_id, 0 as level
        FROM tenant_roles tr
        JOIN user_role_assignments ura ON tr.id = ura.tenant_role_id
        WHERE ura.user_id = user_id_param
        AND ura.is_active = TRUE
        AND (ura.expires_at IS NULL OR ura.expires_at > NOW())

        UNION ALL

        -- Get parent roles recursively
        SELECT tr.id as role_id, tr.name as role_name, tr.parent_role_id, rh.level + 1
        FROM tenant_roles tr
        JOIN role_hierarchy rh ON tr.id = rh.parent_role_id
        WHERE rh.level < 10 -- prevent infinite recursion
    )
    SELECT DISTINCT
        p.code as permission_code,
        p.description as permission_description,
        p.module,
        p.action,
        p.resource,
        trp.scope_type as scope,
        trp.scope_id,
        rh.role_name
    FROM role_hierarchy rh
    JOIN tenant_role_permissions trp ON rh.role_id = trp.tenant_role_id
    JOIN permissions p ON trp.permission_id = p.id
    WHERE trp.is_active = TRUE
    AND p.is_active = TRUE
    ORDER BY permission_code, scope, scope_id;
END;
$$ LANGUAGE plpgsql;

-- Create audit log for role and permission changes
CREATE TABLE rbac_audit_log (
    id SERIAL PRIMARY KEY,
    action VARCHAR(50) NOT NULL, -- 'role_assigned', 'role_removed', 'permission_granted', etc.
    actor_id INTEGER NOT NULL REFERENCES users(id), -- who performed the action
    target_user_id INTEGER REFERENCES users(id), -- target user (if applicable)
    target_role_id INTEGER REFERENCES tenant_roles(id), -- target role (if applicable)
    permission_id INTEGER REFERENCES permissions(id), -- target permission (if applicable)
    old_values JSONB,
    new_values JSONB,
    reason TEXT,
    ip_address TEXT NOT NULL,
    user_agent TEXT NOT NULL,
    tenant_id INTEGER NOT NULL REFERENCES tenants(id),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Create indexes for audit log
CREATE INDEX idx_rbac_audit_log_actor_id ON rbac_audit_log(actor_id);
CREATE INDEX idx_rbac_audit_log_target_user_id ON rbac_audit_log(target_user_id);
CREATE INDEX idx_rbac_audit_log_tenant_id ON rbac_audit_log(tenant_id);
CREATE INDEX idx_rbac_audit_log_created_at ON rbac_audit_log(created_at);
CREATE INDEX idx_rbac_audit_log_action ON rbac_audit_log(action);

-- Seed: tenant, business, branch, roles, users, role-permissions, assignments
-- Idempotent: uses WHERE NOT EXISTS and selects to avoid duplicates

WITH upsert_tenant AS (
    INSERT INTO tenants (organization, email, password_hash, is_active, plan, email_verified)
    SELECT 'Acme Inc', 'owner@acme.test',
           -- bcrypt("password")
           '$2a$10$92IXUNpkjO0rOQ5byMi.Ye4oKoEa3Ro9llC/.og/at2.uheWG/igi',
           TRUE, 'free', TRUE
    WHERE NOT EXISTS (SELECT 1 FROM tenants WHERE email = 'owner@acme.test')
    RETURNING id
), tenant_row AS (
    SELECT id FROM upsert_tenant
    UNION ALL
    SELECT t.id FROM tenants t WHERE t.email = 'owner@acme.test'
),
upsert_business AS (
    INSERT INTO businesses (tenants_id, name, motto, email, country, is_active, metadata)
    SELECT tr.id, 'Acme Retail', 'We sell everything', 'contact@acme.test', 'US', TRUE, '{}'::jsonb
    FROM tenant_row tr
    WHERE NOT EXISTS (
        SELECT 1 FROM businesses b WHERE b.tenants_id = tr.id AND b.name = 'Acme Retail'
    )
    RETURNING id
), business_row AS (
    SELECT id FROM upsert_business
    UNION ALL
    SELECT b.id
    FROM businesses b
    JOIN tenant_row tr ON b.tenants_id = tr.id
    WHERE b.name = 'Acme Retail'
),
upsert_branch AS (
    INSERT INTO branches (business_id, name, address, phone, email, is_active, metadata)
    SELECT b.id, 'Main Branch', '123 Market St', '+1-555-0100', 'main@acme.test', TRUE, '{}'::jsonb
    FROM business_row b
    WHERE NOT EXISTS (
        SELECT 1 FROM branches br WHERE br.business_id = b.id AND br.name = 'Main Branch'
    )
    RETURNING id
), branch_row AS (
    SELECT id FROM upsert_branch
    UNION ALL
    SELECT br.id
    FROM branches br
    JOIN business_row b ON br.business_id = b.id
    WHERE br.name = 'Main Branch'
),

-- Roles (per-tenant)
ins_owner AS (
    INSERT INTO tenant_roles (tenant_id, name, description, is_default, metadata)
    SELECT tr.id, 'Owner', 'Full access to all features', FALSE, '{}'::jsonb
    FROM tenant_row tr
    WHERE NOT EXISTS (SELECT 1 FROM tenant_roles r WHERE r.tenant_id = tr.id AND r.name = 'Owner')
    RETURNING id
), owner_role AS (
    SELECT id FROM ins_owner
    UNION ALL
    SELECT r.id FROM tenant_roles r JOIN tenant_row t ON r.tenant_id = t.id WHERE r.name = 'Owner'
),
ins_admin AS (
    INSERT INTO tenant_roles (tenant_id, name, description, is_default, metadata)
    SELECT tr.id, 'Admin', 'Administrative access', FALSE, '{}'::jsonb
    FROM tenant_row tr
    WHERE NOT EXISTS (SELECT 1 FROM tenant_roles r WHERE r.tenant_id = tr.id AND r.name = 'Admin')
    RETURNING id
), admin_role AS (
    SELECT id FROM ins_admin
    UNION ALL
    SELECT r.id FROM tenant_roles r JOIN tenant_row t ON r.tenant_id = t.id WHERE r.name = 'Admin'
),
ins_manager AS (
    INSERT INTO tenant_roles (tenant_id, name, description, is_default, metadata)
    SELECT tr.id, 'Manager', 'Manage operations', FALSE, '{}'::jsonb
    FROM tenant_row tr
    WHERE NOT EXISTS (SELECT 1 FROM tenant_roles r WHERE r.tenant_id = tr.id AND r.name = 'Manager')
    RETURNING id
), manager_role AS (
    SELECT id FROM ins_manager
    UNION ALL
    SELECT r.id FROM tenant_roles r JOIN tenant_row t ON r.tenant_id = t.id WHERE r.name = 'Manager'
),
ins_cashier AS (
    INSERT INTO tenant_roles (tenant_id, name, description, is_default, metadata)
    SELECT tr.id, 'Cashier', 'POS cashier operations', FALSE, '{}'::jsonb
    FROM tenant_row tr
    WHERE NOT EXISTS (SELECT 1 FROM tenant_roles r WHERE r.tenant_id = tr.id AND r.name = 'Cashier')
    RETURNING id
), cashier_role AS (
    SELECT id FROM ins_cashier
    UNION ALL
    SELECT r.id FROM tenant_roles r JOIN tenant_row t ON r.tenant_id = t.id WHERE r.name = 'Cashier'
),
ins_pos AS (
    INSERT INTO tenant_roles (tenant_id, name, description, is_default, metadata)
    SELECT tr.id, 'POS', 'Point-of-sale staff', TRUE, '{}'::jsonb
    FROM tenant_row tr
    WHERE NOT EXISTS (SELECT 1 FROM tenant_roles r WHERE r.tenant_id = tr.id AND r.name = 'POS')
    RETURNING id
), pos_role AS (
    SELECT id FROM ins_pos
    UNION ALL
    SELECT r.id FROM tenant_roles r JOIN tenant_row t ON r.tenant_id = t.id WHERE r.name = 'POS'
),

-- Users
ins_owner_user AS (
    INSERT INTO users (tenants_id, business_id, branch_id, email, password_hash,
                       first_name, last_name, phone, is_active, email_verified, metadata)
    SELECT t.id, b.id, br.id, 'owner@acme.test',
           '$2a$10$92IXUNpkjO0rOQ5byMi.Ye4oKoEa3Ro9llC/.og/at2.uheWG/igi', -- "password"
           'Olivia', 'Owner', '+1-555-1000', TRUE, TRUE, '{}'::jsonb
    FROM tenant_row t, business_row b, branch_row br
    WHERE NOT EXISTS (SELECT 1 FROM users u WHERE u.email = 'owner@acme.test')
    RETURNING id
), owner_user AS (
    SELECT id FROM ins_owner_user
    UNION ALL SELECT u.id FROM users u WHERE u.email = 'owner@acme.test'
),
ins_admin_user AS (
    INSERT INTO users (tenants_id, business_id, branch_id, email, password_hash,
                       first_name, last_name, phone, is_active, email_verified, metadata)
    SELECT t.id, b.id, br.id, 'admin@acme.test',
           '$2a$10$92IXUNpkjO0rOQ5byMi.Ye4oKoEa3Ro9llC/.og/at2.uheWG/igi',
           'Ava', 'Admin', '+1-555-1001', TRUE, TRUE, '{}'::jsonb
    FROM tenant_row t, business_row b, branch_row br
    WHERE NOT EXISTS (SELECT 1 FROM users u WHERE u.email = 'admin@acme.test')
    RETURNING id
), admin_user AS (
    SELECT id FROM ins_admin_user
    UNION ALL SELECT u.id FROM users u WHERE u.email = 'admin@acme.test'
),
ins_manager_user AS (
    INSERT INTO users (tenants_id, business_id, branch_id, email, password_hash,
                       first_name, last_name, phone, is_active, email_verified, metadata)
    SELECT t.id, b.id, br.id, 'manager@acme.test',
           '$2a$10$92IXUNpkjO0rOQ5byMi.Ye4oKoEa3Ro9llC/.og/at2.uheWG/igi',
           'Mia', 'Manager', '+1-555-1002', TRUE, TRUE, '{}'::jsonb
    FROM tenant_row t, business_row b, branch_row br
    WHERE NOT EXISTS (SELECT 1 FROM users u WHERE u.email = 'manager@acme.test')
    RETURNING id
), manager_user AS (
    SELECT id FROM ins_manager_user
    UNION ALL SELECT u.id FROM users u WHERE u.email = 'manager@acme.test'
),
ins_cashier_user AS (
    INSERT INTO users (tenants_id, business_id, branch_id, email, password_hash,
                       first_name, last_name, phone, is_active, email_verified, metadata)
    SELECT t.id, b.id, br.id, 'cashier@acme.test',
           '$2a$10$92IXUNpkjO0rOQ5byMi.Ye4oKoEa3Ro9llC/.og/at2.uheWG/igi',
           'Charlotte', 'Cashier', '+1-555-1003', TRUE, TRUE, '{}'::jsonb
    FROM tenant_row t, business_row b, branch_row br
    WHERE NOT EXISTS (SELECT 1 FROM users u WHERE u.email = 'cashier@acme.test')
    RETURNING id
), cashier_user AS (
    SELECT id FROM ins_cashier_user
    UNION ALL SELECT u.id FROM users u WHERE u.email = 'cashier@acme.test'
),
ins_pos_user AS (
    INSERT INTO users (tenants_id, business_id, branch_id, email, password_hash,
                       first_name, last_name, phone, is_active, email_verified, metadata)
    SELECT t.id, b.id, br.id, 'pos@acme.test',
           '$2a$10$92IXUNpkjO0rOQ5byMi.Ye4oKoEa3Ro9llC/.og/at2.uheWG/igi',
           'Noah', 'POS', '+1-555-1004', TRUE, TRUE, '{}'::jsonb
    FROM tenant_row t, business_row b, branch_row br
    WHERE NOT EXISTS (SELECT 1 FROM users u WHERE u.email = 'pos@acme.test')
    RETURNING id
), pos_user AS (
    SELECT id FROM ins_pos_user
    UNION ALL SELECT u.id FROM users u WHERE u.email = 'pos@acme.test'
),

-- Assign roles to users (idempotent)
assign_owner AS (
    INSERT INTO user_role_assignments (user_id, tenant_role_id, business_id, branch_id, assigned_by, metadata)
    SELECT ou.id, orl.id, b.id, br.id, ou.id, '{}'::jsonb
    FROM owner_user ou, owner_role orl, business_row b, branch_row br
    WHERE NOT EXISTS (
        SELECT 1 FROM user_role_assignments x
        WHERE x.user_id = ou.id AND x.tenant_role_id = orl.id
          AND x.business_id = b.id AND x.branch_id = br.id AND x.is_active = TRUE
    )
    RETURNING 1
),
assign_admin AS (
    INSERT INTO user_role_assignments (user_id, tenant_role_id, business_id, branch_id, assigned_by, metadata)
    SELECT au.id, ar.id, b.id, br.id, (SELECT id FROM owner_user), '{}'::jsonb
    FROM admin_user au, admin_role ar, business_row b, branch_row br
    WHERE NOT EXISTS (
        SELECT 1 FROM user_role_assignments x
        WHERE x.user_id = au.id AND x.tenant_role_id = ar.id
          AND x.business_id = b.id AND x.branch_id = br.id AND x.is_active = TRUE
    )
    RETURNING 1
),
assign_manager AS (
    INSERT INTO user_role_assignments (user_id, tenant_role_id, business_id, branch_id, assigned_by, metadata)
    SELECT mu.id, mr.id, b.id, br.id, (SELECT id FROM owner_user), '{}'::jsonb
    FROM manager_user mu, manager_role mr, business_row b, branch_row br
    WHERE NOT EXISTS (
        SELECT 1 FROM user_role_assignments x
        WHERE x.user_id = mu.id AND x.tenant_role_id = mr.id
          AND x.business_id = b.id AND x.branch_id = br.id AND x.is_active = TRUE
    )
    RETURNING 1
),
assign_cashier AS (
    INSERT INTO user_role_assignments (user_id, tenant_role_id, business_id, branch_id, assigned_by, metadata)
    SELECT cu.id, cr.id, b.id, br.id, (SELECT id FROM owner_user), '{}'::jsonb
    FROM cashier_user cu, cashier_role cr, business_row b, branch_row br
    WHERE NOT EXISTS (
        SELECT 1 FROM user_role_assignments x
        WHERE x.user_id = cu.id AND x.tenant_role_id = cr.id
          AND x.business_id = b.id AND x.branch_id = br.id AND x.is_active = TRUE
    )
    RETURNING 1
),
assign_pos AS (
    INSERT INTO user_role_assignments (user_id, tenant_role_id, business_id, branch_id, assigned_by, metadata)
    SELECT pu.id, pr.id, b.id, br.id, (SELECT id FROM owner_user), '{}'::jsonb
    FROM pos_user pu, pos_role pr, business_row b, branch_row br
    WHERE NOT EXISTS (
        SELECT 1 FROM user_role_assignments x
        WHERE x.user_id = pu.id AND x.tenant_role_id = pr.id
          AND x.business_id = b.id AND x.branch_id = br.id AND x.is_active = TRUE
    )
    RETURNING 1
)
SELECT 1;

-- Grant permissions
-- Owner: grant all active permissions available at migration time
INSERT INTO tenant_role_permissions (tenant_role_id, permission_id, scope_type, scope_id, granted_by, metadata)
SELECT
    (SELECT tr.id
     FROM tenant_roles tr
     JOIN tenants t ON t.id = tr.tenant_id
     WHERE tr.name = 'Owner' AND t.email = 'owner@acme.test'),
    p.id,
    p.scope,
    NULL,
    (SELECT u.id FROM users u WHERE u.email = 'owner@acme.test'),
    '{}'::jsonb
FROM permissions p
WHERE p.is_active = TRUE
AND NOT EXISTS (
    SELECT 1 FROM tenant_role_permissions trp
    WHERE trp.tenant_role_id = (SELECT tr.id
                                FROM tenant_roles tr
                                JOIN tenants t ON t.id = tr.tenant_id
                                WHERE tr.name = 'Owner' AND t.email = 'owner@acme.test')
      AND trp.permission_id = p.id
);

-- Admin: grant all active permissions (adjust here if you want stricter than Owner)
INSERT INTO tenant_role_permissions (tenant_role_id, permission_id, scope_type, scope_id, granted_by, metadata)
SELECT
    (SELECT tr.id
     FROM tenant_roles tr
     JOIN tenants t ON t.id = tr.tenant_id
     WHERE tr.name = 'Admin' AND t.email = 'owner@acme.test'),
    p.id,
    p.scope,
    NULL,
    (SELECT u.id FROM users u WHERE u.email = 'owner@acme.test'),
    '{}'::jsonb
FROM permissions p
WHERE p.is_active = TRUE
AND NOT EXISTS (
    SELECT 1 FROM tenant_role_permissions trp
    WHERE trp.tenant_role_id = (SELECT tr.id
                                FROM tenant_roles tr
                                JOIN tenants t ON t.id = tr.tenant_id
                                WHERE tr.name = 'Admin' AND t.email = 'owner@acme.test')
      AND trp.permission_id = p.id
);

-- Manager: inventory/pos read+create+update, analytics read/export, branch read, business read
INSERT INTO tenant_role_permissions (tenant_role_id, permission_id, scope_type, scope_id, granted_by, metadata)
SELECT
    (SELECT tr.id
     FROM tenant_roles tr
     JOIN tenants t ON t.id = tr.tenant_id
     WHERE tr.name = 'Manager' AND t.email = 'owner@acme.test'),
    p.id,
    p.scope,
    NULL,
    (SELECT u.id FROM users u WHERE u.email = 'owner@acme.test'),
    '{}'::jsonb
FROM permissions p
WHERE p.is_active = TRUE
AND (
    (p.module IN ('inventory','pos') AND p.action IN ('read','create','update')) OR
    (p.module = 'analytics' AND p.action IN ('read','export')) OR
    (p.module = 'branch' AND p.action = 'read') OR
    (p.module = 'business' AND p.action = 'read')
)
AND NOT EXISTS (
    SELECT 1 FROM tenant_role_permissions trp
    WHERE trp.tenant_role_id = (SELECT tr.id
                                FROM tenant_roles tr
                                JOIN tenants t ON t.id = tr.tenant_id
                                WHERE tr.name = 'Manager' AND t.email = 'owner@acme.test')
      AND trp.permission_id = p.id
);

-- Cashier: POS read/create
INSERT INTO tenant_role_permissions (tenant_role_id, permission_id, scope_type, scope_id, granted_by, metadata)
SELECT
    (SELECT tr.id
     FROM tenant_roles tr
     JOIN tenants t ON t.id = tr.tenant_id
     WHERE tr.name = 'Cashier' AND t.email = 'owner@acme.test'),
    p.id,
    p.scope,
    NULL,
    (SELECT u.id FROM users u WHERE u.email = 'owner@acme.test'),
    '{}'::jsonb
FROM permissions p
WHERE p.is_active = TRUE
AND p.module = 'pos' AND p.action IN ('read','create')
AND NOT EXISTS (
    SELECT 1 FROM tenant_role_permissions trp
    WHERE trp.tenant_role_id = (SELECT tr.id
                                FROM tenant_roles tr
                                JOIN tenants t ON t.id = tr.tenant_id
                                WHERE tr.name = 'Cashier' AND t.email = 'owner@acme.test')
      AND trp.permission_id = p.id
);

-- POS: POS read/create
INSERT INTO tenant_role_permissions (tenant_role_id, permission_id, scope_type, scope_id, granted_by, metadata)
SELECT
    (SELECT tr.id
     FROM tenant_roles tr
     JOIN tenants t ON t.id = tr.tenant_id
     WHERE tr.name = 'POS' AND t.email = 'owner@acme.test'),
    p.id,
    p.scope,
    NULL,
    (SELECT u.id FROM users u WHERE u.email = 'owner@acme.test'),
    '{}'::jsonb
FROM permissions p
WHERE p.is_active = TRUE
AND p.module = 'pos' AND p.action IN ('read','create')
AND NOT EXISTS (
    SELECT 1 FROM tenant_role_permissions trp
    WHERE trp.tenant_role_id = (SELECT tr.id
                                FROM tenant_roles tr
                                JOIN tenants t ON t.id = tr.tenant_id
                                WHERE tr.name = 'POS' AND t.email = 'owner@acme.test')
      AND trp.permission_id = p.id
);

DO $$
BEGIN
    RAISE NOTICE '✅ Seeded tenant/business/branch, roles, users, role-permissions, and assignments.';
    RAISE NOTICE 'Users: owner@acme.test, admin@acme.test, manager@acme.test, cashier@acme.test, pos@acme.test (password for all: password)';
END $$;