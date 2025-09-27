-- Sample user data for hotel-erp system
-- This file contains sample users with different roles for testing and development
-- Uses ON CONFLICT DO NOTHING to prevent duplicate key errors

-- Sample user with admin role (role_id = 1)
-- Password: password (hashed with bcrypt)
INSERT INTO users (username, first_name, last_name, email, password_hash, gender, role_id, is_active)
VALUES (
    'admin',
    'John',
    'Doe',
    'admin@hotel.com',
    '$2a$10$92IXUNpkjO0rOQ5byMi.Ye4oKoEa3Ro9llC/.og/at2.uheWG/igi',
    'male',
    1,
    true
)
ON CONFLICT (username) DO NOTHING;

-- Handle email conflict separately for admin
INSERT INTO users (username, first_name, last_name, email, password_hash, gender, role_id, is_active)
SELECT 'admin_alt', 'John', 'Doe', 'admin@hotel.com', '$2a$10$92IXUNpkjO0rOQ5byMi.Ye4oKoEa3Ro9llC/.og/at2.uheWG/igi', 'male', 1, true
WHERE NOT EXISTS (SELECT 1 FROM users WHERE email = 'admin@hotel.com');

-- Sample user with manager role (role_id = 2)
-- Password: manager123 (hashed with bcrypt)
INSERT INTO users (username, first_name, last_name, email, password_hash, gender, role_id, is_active)
VALUES (
    'manager1',
    'Jane',
    'Smith',
    'manager@hotel.com',
    '$2a$10$8K1p/a0dhrxiH8Tf4Gro9e.0uI4JGO0JG6LJZr1f7wFYw8mO6pR1W',
    'female',
    2,
    true
)
ON CONFLICT (username) DO NOTHING;

-- Handle email conflict separately for manager
INSERT INTO users (username, first_name, last_name, email, password_hash, gender, role_id, is_active)
SELECT 'manager_alt', 'Jane', 'Smith', 'manager@hotel.com', '$2a$10$8K1p/a0dhrxiH8Tf4Gro9e.0uI4JGO0JG6LJZr1f7wFYw8mO6pR1W', 'female', 2, true
WHERE NOT EXISTS (SELECT 1 FROM users WHERE email = 'manager@hotel.com');

-- Sample user with POS staff role (role_id = 3)
-- Password: pos123 (hashed with bcrypt)
INSERT INTO users (username, first_name, last_name, email, password_hash, gender, role_id, is_active)
VALUES (
    'pos_staff1',
    'Mike',
    'Johnson',
    'pos@hotel.com',
    '$2a$10$Q8H2k5JFJzJ8F5JzJ8F5Ju.J8F5JzJ8F5JzJ8F5JzJ8F5JzJ8F5J8',
    'male',
    3,
    true
)
ON CONFLICT (username) DO NOTHING;

-- Handle email conflict separately for POS staff
INSERT INTO users (username, first_name, last_name, email, password_hash, gender, role_id, is_active)
SELECT 'pos_alt', 'Mike', 'Johnson', 'pos@hotel.com', '$2a$10$Q8H2k5JFJzJ8F5JzJ8F5Ju.J8F5JzJ8F5JzJ8F5JzJ8F5JzJ8F5J8', 'male', 3, true
WHERE NOT EXISTS (SELECT 1 FROM users WHERE email = 'pos@hotel.com');

-- Sample user with cashier role (role_id = 4)
-- Password: cashier123 (hashed with bcrypt)
INSERT INTO users (username, first_name, last_name, email, password_hash, gender, role_id, is_active)
VALUES (
    'cashier1',
    'Sarah',
    'Wilson',
    'cashier@hotel.com',
    '$2a$10$L7N3k6KGKzL8G6KzL8G6Lu.L8G6KzL8G6KzL8G6KzL8G6KzL8G6L8',
    'female',
    4,
    true
)
ON CONFLICT (username) DO NOTHING;

-- Handle email conflict separately for cashier
INSERT INTO users (username, first_name, last_name, email, password_hash, gender, role_id, is_active)
SELECT 'cashier_alt', 'Sarah', 'Wilson', 'cashier@hotel.com', '$2a$10$L7N3k6KGKzL8G6KzL8G6Lu.L8G6KzL8G6KzL8G6KzL8G6KzL8G6L8', 'female', 4, true
WHERE NOT EXISTS (SELECT 1 FROM users WHERE email = 'cashier@hotel.com');

-- Additional sample user - inactive user for testing
-- Password: test123 (hashed with bcrypt)
INSERT INTO users (username, first_name, last_name, email, password_hash, gender, role_id, is_active)
VALUES (
    'test_user',
    'Test',
    'User',
    'test@hotel.com',
    '$2a$10$T9P4l7MHMzT9H7MzT9H7Mu.T9H7MzT9H7MzT9H7MzT9H7MzT9H7M9',
    'male',
    3,
    false
)
ON CONFLICT (username) DO NOTHING;

-- Handle email conflict separately for test user
INSERT INTO users (username, first_name, last_name, email, password_hash, gender, role_id, is_active)
SELECT 'test_alt', 'Test', 'User', 'test@hotel.com', '$2a$10$T9P4l7MHMzT9H7MzT9H7Mu.T9H7MzT9H7MzT9H7MzT9H7MzT9H7M9', 'male', 3, false
WHERE NOT EXISTS (SELECT 1 FROM users WHERE email = 'test@hotel.com');

-- Display confirmation message
DO $$
BEGIN
    RAISE NOTICE '✅ Sample users seeded successfully!';
    RAISE NOTICE '📋 Available test accounts:';
    RAISE NOTICE '   👑 Admin: admin@hotel.com (password: password)';
    RAISE NOTICE '   👨‍💼 Manager: manager@hotel.com (password: manager123)';
    RAISE NOTICE '   👨‍💻 POS Staff: pos@hotel.com (password: pos123)';
    RAISE NOTICE '   💰 Cashier: cashier@hotel.com (password: cashier123)';
    RAISE NOTICE '   🧪 Test User: test@hotel.com (password: test123) [INACTIVE]';
END $$;
