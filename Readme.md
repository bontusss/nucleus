# Hotel ERP API Documentation

This document provides comprehensive information about the Hotel ERP API, including setup, usage, and available endpoints.

## Table of Contents

1. [Overview](#overview)
2. [Getting Started](#getting-started)
3. [Authentication](#authentication)
4. [API Documentation Formats](#api-documentation-formats)
5. [Available Endpoints](#available-endpoints)
6. [Request/Response Examples](#requestresponse-examples)
7. [Error Handling](#error-handling)
8. [Development](#development)
9. [Sample Data](#sample-data)
10. [Deployment](#deployment)

## Overview

The Hotel ERP API is a RESTful API that provides endpoints for managing hotel operations including:

- Authentication and authorization
- Point of Sale (POS) operations
- Inventory management
- Sales reporting
- User management

### API Specifications

- **Version**: 1.0.0
- **Base URL**: `http://localhost:9000/api`
- **Authentication**: JWT Bearer tokens
- **Content Type**: `application/json`
- **Documentation**: Swagger/OpenAPI 3.0

## Getting Started

### Prerequisites

- Go 1.24.3 or higher
- PostgreSQL 15+
- Redis (optional, for caching)

### Installation

1. Clone the repository:
```bash
git clone <repository-url>
cd hotel-erp
```

2. Install dependencies:
```bash
go mod download
```

3. Set up environment variables:
```bash
cp .env.example .env
# Edit .env with your configuration
```

4. Start the services:
```bash
make s_up  # Start PostgreSQL and Redis containers
make m_up  # Run database migrations
```

5. Start the API server:
```bash
make start  # Development mode with hot reload
# OR
make build && ./bin/app  # Production mode
```

6. Load sample data (optional):
```bash
./scripts/seed_users.sh  # Load sample users for testing
```

### Accessing API Documentation

Once the server is running, you can access the API documentation at:

- **Swagger UI**: http://localhost:9000/docs/swagger/index.html
- **Redocly**: http://localhost:9000/redoc
- **OpenAPI JSON**: http://localhost:9000/docs/swagger/doc.json

## Authentication

The API uses JWT (JSON Web Tokens) for authentication. All protected endpoints require a valid JWT token in the Authorization header.

### Login

To authenticate, send a POST request to `/api/v1/auth/login`:

```http
POST /api/v1/auth/login
Content-Type: application/json

{
  "username": "admin",
  "password": "password"
}
```

Or login with email:

```http
POST /api/v1/auth/login
Content-Type: application/json

{
  "email": "admin@hotel.com",
  "password": "password"
}
```

Response:
```json
{
  "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."
}
```

### Using the Token

Include the token in the Authorization header for protected endpoints:

```http
Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...
```

## API Documentation Formats

### Swagger UI

Interactive API documentation with the ability to test endpoints directly in the browser.

**URL**: http://localhost:7000/docs/swagger/index.html

Features:
- Interactive endpoint testing
- Request/response examples
- Schema validation
- Authentication testing

### Redocly

Clean, modern API documentation interface.

**URL**: http://localhost:7000/redoc

Features:
- Clean, readable interface
- Code examples in multiple languages
- Detailed schema documentation
- Mobile-friendly design

### OpenAPI JSON

Raw OpenAPI specification in JSON format for integration with other tools.

**URL**: http://localhost:7000/docs/swagger/doc.json

## Available Endpoints

### Authentication
- `POST /api/v1/auth/login` - User authentication

### Health Check
- `GET /health` - API health status

### Point of Sale (POS)
- `POST /api/v1/pos/sales` - Create a new sale
- `GET /api/v1/pos/sales/history` - Get sales history
- `POST /api/v1/pos/items` - Create a new item

### Documentation
- `GET /docs/` - Redirect to Swagger UI
- `GET /docs/swagger/*` - Swagger UI interface
- `GET /redoc` - Redocly documentation
- `GET /docs/health` - Documentation service health

## Request/Response Examples

### Create Sale

**Request**:
```http
POST /api/v1/pos/sales
Authorization: Bearer <token>
Content-Type: application/json

{
  "customer_id": 1,
  "items": [
    {
      "item_id": 1,
      "quantity": 2,
      "price": 25.99
    }
  ],
  "discount": 10.5,
  "tax_rate": 8.25
}
```

**Response**:
```json
{
  "id": 1,
  "customer_id": 1,
  "total_amount": 56.23,
  "tax_amount": 4.27,
  "discount_amount": 10.5,
  "items": [
    {
      "item_id": 1,
      "quantity": 2,
      "price": 25.99
    }
  ],
  "created_at": "2024-01-15T10:30:00Z"
}
```

### Get Sales History

**Request**:
```http
GET /api/v1/pos/sales/history?page=1&limit=20&start_date=2024-01-01&end_date=2024-01-31
Authorization: Bearer <token>
```

**Response**:
```json
{
  "sales": [
    {
      "id": 1,
      "customer_id": 1,
      "total_amount": 56.23,
      "tax_amount": 4.27,
      "discount_amount": 10.5,
      "items": [
        {
          "item_id": 1,
          "quantity": 2,
          "price": 25.99
        }
      ],
      "created_at": "2024-01-15T10:30:00Z"
    }
  ],
  "pagination": {
    "page": 1,
    "limit": 20,
    "total": 1,
    "pages": 1
  }
}
```

## Error Handling

The API uses standard HTTP status codes and returns errors in the following format:

```json
{
  "error": "Error message description"
}
```

### Common Status Codes

- `200 OK` - Request successful
- `201 Created` - Resource created successfully
- `400 Bad Request` - Invalid request data
- `401 Unauthorized` - Authentication required
- `403 Forbidden` - Insufficient permissions
- `404 Not Found` - Resource not found
- `500 Internal Server Error` - Server error

### Authentication Errors

- `401 Unauthorized` - Missing or invalid token
- `403 Forbidden` - Valid token but insufficient permissions

Example error response:
```json
{
  "error": "Invalid credentials"
}
```

## Sample Data

The system includes sample user data for development and testing purposes. This data includes users with different roles and permissions to help you test various scenarios.

### Quick Setup

Load sample users with a single command:

```bash
./scripts/seed_users.sh
```

### Sample User Accounts

The following test accounts are available after loading sample data:

| Role | Username | Email | Password | Permissions |
|------|----------|--------|----------|-------------|
| Admin | admin | admin@hotel.com | password | Full system access |
| Manager | manager1 | manager@hotel.com | manager123 | POS operations, booking management |
| POS Staff | pos_staff1 | pos@hotel.com | pos123 | POS sales, view history |
| Cashier | cashier1 | cashier@hotel.com | cashier123 | POS sales only |
| Test User | test_user | test@hotel.com | test123 | Inactive (for testing) |


### Comprehenive Permissions list

| Permission              | Description                        |
|--------------------------|------------------------------------|
| **Admin**               |                                    |
| `admin:manage`          | Manage admin settings              |
| **Sales**               |                                    |
| `pos:sell`              | Create new sales in POS            |
| `sale:view`             | View sales history in POS          |
| `sale:manage_items`     | Manage POS items                   |
| `sale:create`           | Create new sales                   |
| `sale:update`           | Update sales                       |
| `sale:delete`           | Delete sales                       |
| `sale:cancel`           | Cancel sales                       |
| `sale:refund`           | Refund sales                       |
| `sale:print`            | Print sales receipts               |
| **Inventory**           |                                    |
| `item:create`           | Create new inventory items         |
| `item:update`           | Update inventory items             |
| `item:delete`           | Delete inventory items             |
| `item:view`             | View inventory items               |
| `item_request:create`   | Create new inventory item requests |
| `item_request:update`   | Update inventory item requests     |
| `item_request:delete`   | Delete inventory item requests     |
| `item_request:view`     | View inventory item requests       |
| `item_request:approve`  | Approve inventory item requests    |
| `item_request:reject`   | Reject inventory item requests     |
| **Users**               |                                    |
| `user:create`           | Create new users                   |
| `user:update`           | Update user information            |
| `user:delete`           | Delete users                       |
| `user:view`             | View user information              |
| **Role**                |                                    |
| `role:create`           | Create new roles                   |
| `role:update`           | Update role information            |
| `role:delete`           | Delete roles                       |
| `role:view`             | View role information              |
| **General settings**    |                                    |
| `setting:create`        | Create new settings                |
| `setting:update`        | Update settings                    |
| `setting:delete`        | Delete settings                    |
| `setting:view`          | View settings                      |


### Testing Authentication

Test login with sample users:

```bash
# Test admin login with email
curl -X POST http://localhost:9000/api/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email":"admin@hotel.com","password":"password"}'

# Test admin login with username
curl -X POST http://localhost:9000/api/auth/login \
  -H "Content-Type: application/json" \
  -d '{"username":"admin","password":"password"}'

# Test manager login
curl -X POST http://localhost:9000/api/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email":"manager@hotel.com","password":"manager123"}'
```

### Manual Setup

If you prefer manual setup, you can run the SQL file directly:

```bash
psql -h localhost -p 5431 -U postgres -d herp_db -f db/seed_users.sql
```

### Environment Variables

The seed script supports custom database connection parameters:

```bash
export DB_HOST="localhost"
export DB_PORT="5431"
export DB_NAME="herp_db"
export DB_USER="postgres"
export DB_PASSWORD="admin"
```

### Security Note

⚠️ **Important**: Sample passwords are for development only. Change all default passwords in production environments.

For detailed information about sample data, see [docs/SAMPLE_DATA.md](docs/SAMPLE_DATA.md).

## Development

### Adding New Endpoints

1. **Create handler functions** with proper Swagger annotations:
```go
// CreateUser godoc
// @Summary Create user
// @Description Create a new user account
// @Tags users
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param body body CreateUserRequest true "User details"
// @Success 201 {object} UserResponse "User created successfully"
// @Failure 400 {object} ErrorResponse "Bad request"
// @Router /users [post]
func CreateUser(c *gin.Context) {
    // Implementation
}
```

2. **Register routes** in the appropriate package
3. **Update documentation** by running:
```bash
make docs-generate
```

### Documentation Generation

The API documentation is automatically generated from code annotations. To regenerate:

```bash
# Install swag tool
go install github.com/swaggo/swag/cmd/swag@latest

# Generate documentation
swag init

# Or use the Makefile
make docs-generate
```

### Swagger Annotations

Use these annotations in your handler functions:

- `@Summary` - Brief description
- `@Description` - Detailed description
- `@Tags` - Group endpoints by functionality
- `@Accept` - Request content type
- `@Produce` - Response content type
- `@Security` - Authentication requirements
- `@Param` - Request parameters
- `@Success` - Success responses
- `@Failure` - Error responses
- `@Router` - Endpoint path and method

### Request/Response Models

Define models with proper JSON tags and examples:

```go
type CreateUserRequest struct {
    Username string `json:"username" binding:"required" example:"john_doe"`
    Email    string `json:"email" binding:"required" example:"john@example.com"`
    Password string `json:"password" binding:"required" example:"password123"`
}
```

## Deployment

### Environment Configuration

Set the following environment variables for documentation:

```bash
# Enable/disable documentation
DOCS_ENABLED=true

# Documentation host (for production)
DOCS_HOST=api.hotel-erp.com

# API version
API_VERSION=1.0.0
```

### Production Deployment

1. **Build the application**:
```bash
make build-prod
```

2. **Deploy using the deployment script**:
```bash
./scripts/deploy.sh deploy
```

3. **Access documentation**:
- Production: https://api.hotel-erp.com/docs/swagger/index.html
- Staging: https://staging-api.hotel-erp.com/docs/swagger/index.html

### Docker Deployment

The API documentation is included in the Docker image:

```bash
# Build and run with Docker Compose
docker-compose up -d

# Access documentation
# http://localhost:9000/docs/swagger/index.html
```

### Security Considerations

For production deployments:

1. **Disable documentation** in production if not needed:
```bash
DOCS_ENABLED=false
```

2. **Secure documentation endpoints** with authentication if public access is not desired

3. **Use HTTPS** for all documentation URLs

4. **Rate limit** documentation endpoints to prevent abuse

## Support

For API documentation issues or questions:

- **Email**: support@hotel-erp.com
- **Documentation Health**: http://localhost:9000/docs/health
- **API Health**: http://localhost:9000/health

## License

This API documentation is licensed under the MIT License.