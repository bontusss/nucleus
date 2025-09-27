# Build the application
build:
	@echo "Building Hotel ERP..."
	@mkdir -p bin
	@go build -o bin/app -ldflags="-s -w" .
	@echo "Build completed: bin/app"

# Build for production with optimizations
build-prod:
	@echo "Building Hotel ERP for production..."
	@mkdir -p bin
	@CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -ldflags="-s -w -extldflags '-static'" -o bin/app .
	@echo "Production build completed: bin/app"

# Development server with hot reload
start:
	air \
		--build.cmd "go build -o bin/app" \
		--build.bin "./bin/app" \
		--build.exclude_dir "vendor" \
		--build.exclude_dir "static" \
		--build.include_ext "go" \
		--build.kill_delay "0.5s" \
		--build.poll "2s"

# Process management commands
pm-start: build
	@./scripts/process-manager.sh start

pm-stop:
	@./scripts/process-manager.sh stop

pm-restart:
	@./scripts/process-manager.sh restart

pm-graceful-restart:
	@./scripts/process-manager.sh graceful-restart

pm-status:
	@./scripts/process-manager.sh status

pm-logs:
	@./scripts/process-manager.sh logs

pm-follow-logs:
	@./scripts/process-manager.sh follow-logs

pm-health:
	@./scripts/process-manager.sh health

pm-cleanup:
	@./scripts/process-manager.sh cleanup

pm-monitor:
	@./scripts/process-manager.sh monitor

# Documentation commands
docs-generate:
	@echo "Generating API documentation..."
	@swag init --generalInfo main.go --output docs/swagger
	@echo "Documentation generated in docs/swagger/"

docs-install:
	@echo "Installing swag tool..."
	@go install github.com/swaggo/swag/cmd/swag@latest
	@echo "Swag tool installed"

docs-serve:
	@echo "Starting documentation server..."
	@echo "Swagger UI: http://localhost:9000/docs/swagger/index.html"
	@echo "Redocly: http://localhost:9000/redoc"
	@echo "OpenAPI JSON: http://localhost:9000/docs/swagger/doc.json"

docs-validate:
	@echo "Validating API documentation..."
	@swag init --generalInfo main.go --output docs/swagger --parseVendor
	@echo "Documentation validation completed"

docs-clean:
	@echo "Cleaning generated documentation..."
	@rm -rf docs/swagger/
	@echo "Documentation cleaned"

seed:
	@echo "Seeding database..."
	@./scripts/seed_users.sh
	@echo "users seeded"

c_m: # create-migration: create migration of name=<migration_name>
	migrate create -ext sql -dir db/migrations -seq $(name)

count ?= 1
version ?= 1
db_username ?= postgres
db_password ?= admin
db_host ?= localhost
db_port ?= 5431
db_name ?= herp_db
ssl_mode ?= disable

# Run database migrations up to apply pending changes
m_up: # migrate-up
	migrate -path db/migrations -database "postgres://${db_username}:${db_password}@${db_host}:${db_port}/${db_name}?sslmode=${ssl_mode}" up $(count)

# Fix dirty database state by forcing to previous clean version
m_fix: # migrate-fix: fix dirty database state
	migrate -path db/migrations -database "postgres://${db_username}:${db_password}@${db_host}:${db_port}/${db_name}?sslmode=${ssl_mode}" force $(version)

# Check current migration version number
m_version: # migrate-version
	migrate -path db/migrations -database "postgres://${db_username}:${db_password}@${db_host}:${db_port}/${db_name}?sslmode=${ssl_mode}" version

# Force database migration version without running migrations
m_fup: # migreate-force up
	migrate -path db/migrations -database "postgres://${db_username}:${db_password}@${db_host}:${db_port}/${db_name}?sslmode=${ssl_mode}" force $(count)

# Roll back database migrations
m_down: # migrate-down
	migrate -path db/migrations -database "postgres://${db_username}:${db_password}@${db_host}:${db_port}/${db_name}?sslmode=${ssl_mode}" down $(count)

# Start services - PostgreSQL | Redis containers in detached mode
s_up: #
	docker compose -f docker-compose.services.yml up -d

# Stop and remove services - PostgreSQL | Bitgo | Redis container
s_down: #
	docker compose -f docker-compose.services.yml down

container_name ?= herp_postgres

# Create a new PostgreSQL database
db_up: # database-up: create a new database
	docker exec -it ${container_name} createdb --username=${db_username} --owner=${db_username} ${db_name}

# Create a full backup of the database
db_backup:
	docker exec -it ${container_name} pg_dump --username=${db_username} ${db_name} > db_backup.sql

# Restore database from a full backup
db_restore:
	docker exec -i ${container_name} psql --username=${db_username} ${db_name} < db_backup.sql

# Backup specific tables from the database
# Usage: make db_backup_specific tables="table1 table2 table3"
db_backup_specific:
	docker exec -it ${container_name} pg_dump --username=${db_username} ${db_name} --table=$(subst $(space),$(,),$(tables)) > db_backup_specific.sql

# Restore specific tables from a backup
# Usage: make db_restore_specific tables="table1 table2 table3"
db_restore_specific:
	docker exec -i ${container_name} psql --username=${db_username} ${db_name} < db_backup_specific.sql

# Drop/delete the database
db_down: # database-down: drop a database
	docker exec -it ${container_name} dropdb --username=${db_username} ${db_name}

# Generate Go code from SQL using sqlc
sqlc: # sqlc-generate
	sqlc generate

# Run all tests
test:
	go test ./...
