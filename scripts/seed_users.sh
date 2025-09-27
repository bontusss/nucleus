#!/bin/bash

# Script to seed sample user data into the hotel-erp database
# This script will load sample users for development and testing

set -e

# Default database connection parameters
DB_HOST=${DB_HOST:-"localhost"}
DB_PORT=${DB_PORT:-"5431"}
DB_NAME=${DB_NAME:-"herp_db"}
DB_USER=${DB_USER:-"postgres"}
DB_PASSWORD=${DB_PASSWORD:-"admin"}

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo -e "${YELLOW}🌱 Seeding user data for hotel-erp...${NC}"

# Check if psql is installed
if ! command -v psql &> /dev/null; then
    echo -e "${RED}❌ Error: psql is not installed. Please install PostgreSQL client tools.${NC}"
    exit 1
fi

# Get the directory of this script
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_DIR="$(dirname "$SCRIPT_DIR")"
SEED_FILE="$PROJECT_DIR/db/seed_users.sql"

# Check if seed file exists
if [ ! -f "$SEED_FILE" ]; then
    echo -e "${RED}❌ Error: Seed file not found at $SEED_FILE${NC}"
    exit 1
fi

echo -e "${YELLOW}📊 Using database connection:${NC}"
echo -e "  Host: $DB_HOST"
echo -e "  Port: $DB_PORT"
echo -e "  Database: $DB_NAME"
echo -e "  User: $DB_USER"
echo ""

# Test database connection
echo -e "${YELLOW}🔍 Testing database connection...${NC}"
export PGPASSWORD="$DB_PASSWORD"

if ! psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d "$DB_NAME" -c "SELECT 1;" &> /dev/null; then
    echo -e "${RED}❌ Error: Cannot connect to database. Please check your connection parameters and ensure the database is running.${NC}"
    exit 1
fi

echo -e "${GREEN}✅ Database connection successful${NC}"

# Check if tables exist
echo -e "${YELLOW}🔍 Checking if required tables exist...${NC}"
TABLES_EXIST=$(psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d "$DB_NAME" -t -c "SELECT COUNT(*) FROM information_schema.tables WHERE table_name IN ('users', 'roles');")

if [ "$TABLES_EXIST" -ne 2 ]; then
    echo -e "${RED}❌ Error: Required tables (users, roles) do not exist. Please run migrations first.${NC}"
    echo -e "${YELLOW}💡 Hint: Run 'make migrate-up' or your migration command first.${NC}"
    exit 1
fi

echo -e "${GREEN}✅ Required tables found${NC}"

# Check if users already exist
echo -e "${YELLOW}🔍 Checking if sample users already exist...${NC}"
EXISTING_USERS=$(psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d "$DB_NAME" -t -c "SELECT COUNT(*) FROM users WHERE email IN ('admin@hotel.com', 'manager@hotel.com', 'pos@hotel.com', 'cashier@hotel.com', 'test@hotel.com');")

if [ "$EXISTING_USERS" -gt 0 ]; then
    echo -e "${YELLOW}⚠️  Warning: Some sample users already exist in the database.${NC}"
    echo -e "${YELLOW}   This script will skip inserting duplicate users.${NC}"
fi

# Execute the seed file
echo -e "${YELLOW}🌱 Inserting sample user data...${NC}"

if psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d "$DB_NAME" -f "$SEED_FILE"; then
    echo -e "${GREEN}✅ Sample users seeded successfully!${NC}"
    echo ""
    echo -e "${YELLOW}📋 Sample user accounts created:${NC}"
    echo -e "  👑 Admin: admin@hotel.com (password: password)"
    echo -e "  👨‍💼 Manager: manager@hotel.com (password: manager123)"
    echo -e "  👨‍💻 POS Staff: pos@hotel.com (password: pos123)"
    echo -e "  💰 Cashier: cashier@hotel.com (password: cashier123)"
    echo -e "  🧪 Test User: test@hotel.com (password: test123) [INACTIVE]"
    echo ""
    echo -e "${GREEN}🎉 Seeding completed successfully!${NC}"
else
    echo -e "${RED}❌ Error: Failed to seed user data${NC}"
    exit 1
fi

# Display user count
USER_COUNT=$(psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d "$DB_NAME" -t -c "SELECT COUNT(*) FROM users;")
echo -e "${YELLOW}📊 Total users in database: $USER_COUNT${NC}"
