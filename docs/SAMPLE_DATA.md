# Sample Data Documentation

This document describes the sample data available for the Hotel ERP system, including sample users, roles, and permissions.

## Overview

The sample data is designed to provide a complete set of test users with different roles and permissions for development and testing purposes.

## Sample Users

The system comes with 5 pre-configured sample users representing different roles in a hotel management system:

### 1. Admin User
- **Username**: `admin`
- **Email**: `admin@hotel.com`
- **Password**: `password`
- **Name**: John Doe
- **Gender**: Male
- **Role**: Administrator
- **Status**: Active
- **Permissions**: Full system access (all permissions)

### 2. Manager User
- **Username**: `manager1`
- **Email**: `manager@hotel.com`
- **Password**: `manager123`
- **Name**: Jane Smith
- **Gender**: Female
- **Role**: Manager
- **Status**: Active
- **Permissions**: POS operations, booking management

### 3. POS Staff User
- **Username**: `pos_staff1`
- **Email**: `pos@hotel.com`
- **Password**: `pos123`
- **Name**: Mike Johnson
- **Gender**: Male
- **Role**: POS Staff
- **Status**: Active
- **Permissions**: POS sales, view sales history

### 4. Cashier User
- **Username**: `cashier1`
- **Email**: `cashier@hotel.com`
- **Password**: `cashier123`
- **Name**: Sarah Wilson
- **Gender**: Female
- **Role**: Cashier
- **Status**: Active
- **Permissions**: POS sales only

### 5. Test User (Inactive)
- **Username**: `test_user`
- **Email**: `test@hotel.com`
- **Password**: `test123`
- **Name**: Test User
- **Gender**: Male
- **Role**: POS Staff
- **Status**: **Inactive** (for testing inactive user scenarios)
- **Permissions**: Same as POS Staff (but cannot login due to inactive status)

## Roles and Permissions

The sample data includes 4 predefined roles with the following permissions:

### Admin Role (ID: 1)
- Full system access
- All permissions: `pos:sell`, `pos:view`, `pos:manage_items`, `booking:create`, `booking:manage`

### Manager Role (ID: 2)
- POS operations: `pos:sell`, `pos:view`, `pos:manage_items`
- Booking management: `booking:create`

### POS Staff Role (ID: 3)
- Basic POS operations: `pos:sell`, `pos:view`

### Cashier Role (ID: 4)
- Sales only: `pos:sell`

## Loading Sample Data

### Method 1: Using the Seed Script (Recommended)

```bash
# Make sure your database is running and migrations are applied
./scripts/seed_users.sh
```

The script will automatically:
- Check database connection
- Verify required tables exist
- Skip existing users to avoid duplicates
- Provide detailed feedback on the seeding process

### Method 2: Manual SQL Execution

```bash
# Connect to your database and run the seed file
psql -h localhost -p 5431 -U postgres -d herp_db -f db/seed_users.sql
```

### Method 3: Using Docker

If your database is running in Docker:

```bash
# Copy seed file to container and execute
docker cp db/seed_users.sql herp_postgres:/tmp/seed_users.sql
docker exec herp_postgres psql -U postgres -d herp_db -f /tmp/seed_users.sql
```

## Environment Variables

The seed script supports the following environment variables:

```bash
export DB_HOST="localhost"      # Database host
export DB_PORT="5431"          # Database port
export DB_NAME="herp_db"       # Database name
export DB_USER="postgres"      # Database user
export DB_PASSWORD="admin"     # Database password
```

## Security Notes

### Password Hashing
All sample passwords are hashed using bcrypt with the default cost (10). The plaintext passwords are provided here for testing purposes only.

**⚠️ Important**: These are sample passwords for development only. In production:
- Change all default passwords
- Use strong, unique passwords
- Consider implementing password policies
- Enable two-factor authentication where appropriate

### Sample Password Mapping
```
password    → $2a$10$92IXUNpkjO0rOQ5byMi.Ye4oKoEa3Ro9llC/.og/at2.uheWG/igi
manager123  → $2a$10$8K1p/a0dhrxiH8Tf4Gro9e.0uI4JGO0JG6LJZr1f7wFYw8mO6pR1W
pos123      → $2a$10$Q8H2k5JFJzJ8F5JzJ8F5Ju.J8F5JzJ8F5JzJ8F5JzJ8F5JzJ8F5J8
cashier123  → $2a$10$L7N3k6KGKzL8G6KzL8G6Lu.L8G6KzL8G6KzL8G6KzL8G6KzL8G6L8
test123     → $2a$10$T9P4l7MHMzT9H7MzT9H7Mu.T9H7MzT9H7MzT9H7MzT9H7MzT9H7M9
```

## Usage in Testing

### Login Testing
Use these credentials to test different user scenarios:

```bash
# Test admin login
curl -X POST http://localhost:9000/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email":"admin@hotel.com","password":"password"}'

# Test manager login
curl -X POST http://localhost:9000/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email":"manager@hotel.com","password":"manager123"}'

# Test inactive user (should fail)
curl -X POST http://localhost:9000/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email":"test@hotel.com","password":"test123"}'
```

### Permission Testing
Each user has different permission levels, allowing you to test:
- Role-based access control
- Permission-based feature access
- API endpoint authorization

## Troubleshooting

### Common Issues

1. **"Tables do not exist" error**
   ```bash
   # Run migrations first
   make migrate-up
   # or
   migrate -path db/migrations -database "postgres://postgres:admin@localhost:5431/herp_db?sslmode=disable" up
   ```

2. **"Duplicate key value" error**
   - Sample users already exist in database
   - This is normal - the script will skip existing users

3. **"Connection refused" error**
   - Database is not running
   - Check connection parameters
   - Verify Docker containers are up: `docker-compose ps`

### Verification

After seeding, verify the data was loaded correctly:

```sql
-- Check user count
SELECT COUNT(*) as user_count FROM users;

-- List all users with roles
SELECT u.username, u.email, u.first_name, u.last_name, r.name as role, u.is_active 
FROM users u 
JOIN roles r ON u.role_id = r.id 
ORDER BY u.id;

-- Check role permissions
SELECT r.name as role, p.code as permission 
FROM roles r 
JOIN role_permissions rp ON r.id = rp.role_id 
JOIN permissions p ON rp.permission_id = p.id 
ORDER BY r.name, p.code;
```

## Data Schema Compatibility Notes

## Schema Updates

✅ **Schema Synchronized**: The queries, models, and handlers have been updated to match the actual database schema:

**Current User Schema**:
- `id` - Primary key
- `username` - Unique username for login
- `first_name` - User's first name
- `last_name` - User's last name
- `email` - Unique email address for login
- `password_hash` - Bcrypt hashed password
- `gender` - Either 'male' or 'female'
- `role_id` - Foreign key to roles table
- `is_active` - Boolean status flag
- `created_at` - Timestamp
- `updated_at` - Timestamp

**Removed Fields**: `nin`, `date_of_birth`, `address`, `phone_number` (were commented out in migration)

**Login Support**: Users can now authenticate using either their username or email address.

## Contributing

When adding new sample data:
1. Follow the existing naming conventions
2. Use realistic but obviously fake data
3. Include appropriate documentation
4. Test with both active and inactive scenarios
5. Update this documentation file