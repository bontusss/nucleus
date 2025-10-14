# Nucleus ERP API

This repository provides the Nucleus ERP API, a modular, API-first business suite for managing tenants, users, businesses, branches, inventory, POS, webhooks, and more. It includes JWT auth, API key auth, rate limiting, activity logging, and generated API documentation (Swagger and Redoc).

## Table of Contents

1. Overview
2. Getting Started
3. Authentication
4. API Documentation
5. Available Endpoints (high-level)
6. Request/Response Examples
7. Error Handling
8. Development
9. Sample Data
10. Deployment

## Overview

- Language: Go
- Database: PostgreSQL 15+
- Cache: Redis (optional, used for rate limiting and session support)
- Docs: Swagger/OpenAPI + Redoc
- Framework: Gin
- Migrations: golang-migrate
- SQL access: sqlc

### API Specs

- Version: 1.0.0
- Base URL: `http://localhost:9000/api/v1`
- Authentication: JWT (Bearer), API Keys
- Content Type: `application/json`

## Getting Started

### Prerequisites

- Go 1.24.3+
- PostgreSQL 15+
- Redis (optional)
- Docker (optional)

### Install

```bash
git clone <repository-url>
cd nucleus

go mod download
```

### Environment

Create a `.env` file with at least the following variables (see `internal/config/config.go` for full list and defaults):

```bash
# Server
PORT=9000
GIN_MODE=release
API_VERSION=v1.0.0

# Database
DATABASE_URL=postgres://postgres:admin@localhost:5431/nucleus_db?sslmode=disable

# JWT
JWT_SECRET=your_very_strong_encypted_secret
JWT_EXPIRY=15
JWT_REFRESH_SECRET=your_very_strong_encypted_secret
JWT_REFRESH_EXPIRY=720

# Redis
REDIS_HOST=localhost
REDIS_PORT=6379
REDIS_PASSWORD=

# Rate limiting
LOGIN_RATE_LIMIT=5
LOGIN_RATE_WINDOW=15
LOGIN_BLOCK_DURATION=30
IP_RATE_LIMIT=50

# Optional integrations
PLUNK_BASE_URL=
PLUNK_SECRET_KEY=
PAPERTRAIL_ADDR=
PAPERTRAIL_APPNAME=
```

### Start Dependencies (Docker)

```bash
make s_up   # Start PostgreSQL and Redis using docker-compose.services.yml
```

- Postgres: exposed on `localhost:5431`
- Redis: exposed on `localhost:6379`

### Migrations

You can migrate in two ways:

- Let the API run migrations automatically on startup (default behavior in `main.go`), or
- Use the CLI:

```bash
# Apply all pending migrations (count=0 applies all)
make m_up count=0

# Show current version
make m_version

# Roll back one step
make m_down count=1
```

### Run the API

```bash
# Dev with hot reload (requires air)
make start

# Or build and run
make build && ./bin/app
```

### Docker (App + Services)

```bash
docker compose up -d
```

- API: `http://localhost:9000`
- Health: `http://localhost:9000/api/v1/health`
- Swagger UI: `http://localhost:9000/docs/swagger/index.html`
- Redoc: `http://localhost:9000/redoc`

## Authentication

The API exposes tenant-scoped auth endpoints under `/api/v1/tenant/*` and supports both JWT and API Key flows.

- JWT login: `POST /api/v1/tenant/login`
- Register tenant: `POST /api/v1/tenant/register`
- Verify email: `POST /api/v1/tenant/verify-email`
- Forgot password: `POST /api/v1/tenant/forgot-password`
- Reset password: `POST /api/v1/tenant/reset-password`

API key management (requires JWT):

- Generate API key: `POST /api/v1/key/generate`
- Use header `X-API-Key: <key>` for API-key-protected routes

## API Documentation

- Swagger UI: `http://localhost:9000/docs/swagger/index.html`
- Redoc: `http://localhost:9000/redoc`
- OpenAPI JSON: `http://localhost:9000/docs/swagger/doc.json`

Regenerate docs (requires `swag`):

```bash
make docs-install   # installs swag tool
make docs-generate  # generates docs into docs/swagger/
```

## Available Endpoints (high-level)

- Auth (tenant): `/api/v1/tenant/*`
- Health: `GET /api/v1/health`
- Users: secured routes registered via `internal/core/user`
- Business/Branches: secured routes in `internal/core/business`
- Inventory: secured routes in `internal/core/inventory`
- Store: secured routes in `internal/core/store`
- Logs: secured routes in `internal/core/ilogs`
- POS: secured routes in `internal/pos`
- API Keys: secured routes in `internal/key`
- Webhooks: secured routes in `internal/webhook`

Note: Many routes require JWT and API Key authentication and are grouped under `/api/v1`.

## Request/Response Examples

### Login

```http
POST /api/v1/tenant/login
Content-Type: application/json

{
  "email": "owner@acme.test",
  "password": "password"
}
```

Then call protected routes with:

```http
Authorization: Bearer <token>
```

### Create Sale (example)

```http
POST /api/v1/pos/sales
Authorization: Bearer <token>
Content-Type: application/json

{
  "customer_id": 1,
  "items": [
    { "item_id": 1, "quantity": 2, "price": 25.99 }
  ],
  "discount": 10.5,
  "tax_rate": 8.25
}
```

## Error Handling

Errors follow standard HTTP codes and JSON bodies:

```json
{ "error": "Error message description" }
```

Common statuses: 200, 201, 400, 401, 403, 404, 500.

## Development

- Build: `make build`
- Tests: `make test`
- SQL to Go (sqlc): `make sqlc`
- Process manager helpers: `make pm-*` (see makefile)
- Docs: `make docs-generate`

## Sample Data

Seed data is embedded in the initial migration `db/migrations/000001_users.up.sql`. On first migration it creates:

- Tenant: `Acme Inc` (email: `owner@acme.test`)
- Business + Branch
- Roles (Owner, Admin, Manager, Cashier, POS)
- Users (all with password `password`):
  - `owner@acme.test`
  - `admin@acme.test`
  - `manager@acme.test`
  - `cashier@acme.test`
  - `pos@acme.test`
- Role assignments and permission grants (using the permissions inserted earlier in the migration)

No additional seed step is required. The legacy script `scripts/seed_users.sh` and `db/seed_users.sql` are kept for reference but do not match the new schema and are not needed.

## Deployment

### Environment

Use the `.env` file for configuration (see above). The API reads from environment variables on startup.

### Docker Compose

- `docker-compose.yml` runs the API (`9000`) plus Postgres (`5431`) and Redis (`6379`).
- Health checks are configured for containers.

### Production Build

```bash
make build-prod
```

Then run the resulting `bin/app` with appropriate environment variables set.

### Health

- `GET /api/v1/health` returns JSON with basic service status, database, redis, version, and uptime.

## License

MIT
