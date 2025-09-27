#!/bin/bash

# Hotel ERP Deployment Script
# Handles zero-downtime deployments with graceful restarts

set -e

# Configuration
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"
APP_NAME="hotel-erp"
DEPLOY_USER="${DEPLOY_USER:-hotel-erp}"
DEPLOY_PATH="${DEPLOY_PATH:-/opt/hotel-erp}"
BACKUP_DIR="${BACKUP_DIR:-$DEPLOY_PATH/backups}"
SERVICE_NAME="${SERVICE_NAME:-hotel-erp}"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
NC='\033[0m' # No Color

# Logging function
log() {
    echo -e "$(date '+%Y-%m-%d %H:%M:%S') - $1"
}

# Error handling
error_exit() {
    log "${RED}ERROR: $1${NC}"
    exit 1
}

# Success message
success() {
    log "${GREEN}SUCCESS: $1${NC}"
}

# Warning message
warn() {
    log "${YELLOW}WARNING: $1${NC}"
}

# Info message
info() {
    log "${BLUE}INFO: $1${NC}"
}

# Check prerequisites
check_prerequisites() {
    info "Checking prerequisites..."

    # Check if running as correct user
    if [[ "$USER" != "$DEPLOY_USER" ]] && [[ "$USER" != "root" ]]; then
        error_exit "This script should be run as $DEPLOY_USER or root"
    fi

    # Check if Go is installed
    if ! command -v go &> /dev/null; then
        error_exit "Go is not installed"
    fi

    # Check if systemctl is available (for systemd)
    if ! command -v systemctl &> /dev/null; then
        warn "systemctl not available, will use process manager instead"
    fi

    # Check if required directories exist
    if [[ ! -d "$DEPLOY_PATH" ]]; then
        info "Creating deployment directory: $DEPLOY_PATH"
        sudo mkdir -p "$DEPLOY_PATH"
        sudo chown "$DEPLOY_USER:$DEPLOY_USER" "$DEPLOY_PATH"
    fi

    success "Prerequisites check completed"
}

# Create backup of current deployment
create_backup() {
    info "Creating backup of current deployment..."

    local timestamp=$(date +"%Y%m%d_%H%M%S")
    local backup_path="$BACKUP_DIR/$timestamp"

    mkdir -p "$backup_path"

    # Backup binary if it exists
    if [[ -f "$DEPLOY_PATH/bin/app" ]]; then
        cp "$DEPLOY_PATH/bin/app" "$backup_path/app"
        info "Binary backed up to $backup_path/app"
    fi

    # Backup configuration files
    if [[ -f "$DEPLOY_PATH/.env" ]]; then
        cp "$DEPLOY_PATH/.env" "$backup_path/.env"
    fi

    # Keep only last 5 backups
    cd "$BACKUP_DIR"
    ls -t | tail -n +6 | xargs -r rm -rf

    success "Backup created at $backup_path"
    echo "$backup_path" > "$DEPLOY_PATH/.last_backup"
}

# Build the application
build_application() {
    info "Building application..."

    cd "$PROJECT_DIR"

    # Clean previous builds
    rm -rf bin/

    # Download dependencies
    go mod download
    go mod tidy

    # Build for production
    CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo \
        -ldflags="-s -w -extldflags '-static'" \
        -o bin/app .

    if [[ ! -f "bin/app" ]]; then
        error_exit "Build failed - binary not created"
    fi

    success "Application built successfully"
}

# Run tests before deployment
run_tests() {
    info "Running tests..."

    cd "$PROJECT_DIR"

    # Run unit tests
    if ! go test ./... -v; then
        error_exit "Tests failed"
    fi

    success "All tests passed"
}

# Deploy the application
deploy_application() {
    info "Deploying application..."

    # Copy binary
    mkdir -p "$DEPLOY_PATH/bin"
    cp "$PROJECT_DIR/bin/app" "$DEPLOY_PATH/bin/app"
    chmod +x "$DEPLOY_PATH/bin/app"

    # Copy configuration files
    if [[ -f "$PROJECT_DIR/.env.production" ]]; then
        cp "$PROJECT_DIR/.env.production" "$DEPLOY_PATH/.env"
    elif [[ -f "$PROJECT_DIR/.env" ]]; then
        cp "$PROJECT_DIR/.env" "$DEPLOY_PATH/.env"
    fi

    # Copy scripts
    mkdir -p "$DEPLOY_PATH/scripts"
    cp -r "$PROJECT_DIR/scripts/"* "$DEPLOY_PATH/scripts/"
    chmod +x "$DEPLOY_PATH/scripts/"*.sh

    # Create necessary directories
    mkdir -p "$DEPLOY_PATH/tmp" "$DEPLOY_PATH/logs"

    # Set ownership
    if [[ "$USER" == "root" ]]; then
        chown -R "$DEPLOY_USER:$DEPLOY_USER" "$DEPLOY_PATH"
    fi

    success "Application deployed to $DEPLOY_PATH"
}

# Check if service is running
is_service_running() {
    if command -v systemctl &> /dev/null; then
        systemctl is-active --quiet "$SERVICE_NAME" 2>/dev/null
    else
        # Fallback to process manager
        "$DEPLOY_PATH/scripts/process-manager.sh" status >/dev/null 2>&1
    fi
}

# Start the service
start_service() {
    info "Starting service..."

    if command -v systemctl &> /dev/null; then
        sudo systemctl start "$SERVICE_NAME"
        sudo systemctl enable "$SERVICE_NAME"
    else
        cd "$DEPLOY_PATH"
        ./scripts/process-manager.sh start
    fi

    success "Service started"
}

# Stop the service
stop_service() {
    info "Stopping service..."

    if command -v systemctl &> /dev/null; then
        sudo systemctl stop "$SERVICE_NAME"
    else
        cd "$DEPLOY_PATH"
        ./scripts/process-manager.sh stop
    fi

    success "Service stopped"
}

# Perform graceful restart
graceful_restart() {
    info "Performing graceful restart..."

    if command -v systemctl &> /dev/null; then
        sudo systemctl reload "$SERVICE_NAME"
    else
        cd "$DEPLOY_PATH"
        ./scripts/process-manager.sh graceful-restart
    fi

    success "Graceful restart completed"
}

# Wait for service to be healthy
wait_for_health() {
    local max_attempts=30
    local attempt=1
    local health_url="http://localhost:${PORT:-9000}/health"

    info "Waiting for service to be healthy..."

    while [[ $attempt -le $max_attempts ]]; do
        if curl -sf "$health_url" >/dev/null 2>&1; then
            success "Service is healthy"
            return 0
        fi

        info "Health check attempt $attempt/$max_attempts failed, waiting..."
        sleep 2
        ((attempt++))
    done

    error_exit "Service failed to become healthy after $max_attempts attempts"
}

# Rollback to previous version
rollback() {
    warn "Initiating rollback..."

    if [[ ! -f "$DEPLOY_PATH/.last_backup" ]]; then
        error_exit "No backup found for rollback"
    fi

    local backup_path=$(cat "$DEPLOY_PATH/.last_backup")

    if [[ ! -d "$backup_path" ]]; then
        error_exit "Backup directory not found: $backup_path"
    fi

    # Stop current service
    if is_service_running; then
        stop_service
    fi

    # Restore from backup
    if [[ -f "$backup_path/app" ]]; then
        cp "$backup_path/app" "$DEPLOY_PATH/bin/app"
        chmod +x "$DEPLOY_PATH/bin/app"
    fi

    if [[ -f "$backup_path/.env" ]]; then
        cp "$backup_path/.env" "$DEPLOY_PATH/.env"
    fi

    # Start service
    start_service
    wait_for_health

    success "Rollback completed successfully"
}

# Perform zero-downtime deployment
zero_downtime_deploy() {
    info "Starting zero-downtime deployment..."

    # Pre-deployment checks
    check_prerequisites
    run_tests
    build_application

    # Create backup
    create_backup

    # Deploy new version
    deploy_application

    # Graceful restart or start
    if is_service_running; then
        graceful_restart
    else
        start_service
    fi

    # Health check
    wait_for_health

    success "Zero-downtime deployment completed successfully"
}

# Install systemd service
install_systemd_service() {
    info "Installing systemd service..."

    if [[ ! -f "$PROJECT_DIR/scripts/hotel-erp.service" ]]; then
        error_exit "Systemd service file not found"
    fi

    # Copy service file
    sudo cp "$PROJECT_DIR/scripts/hotel-erp.service" "/etc/systemd/system/$SERVICE_NAME.service"

    # Update paths in service file
    sudo sed -i "s|/opt/hotel-erp|$DEPLOY_PATH|g" "/etc/systemd/system/$SERVICE_NAME.service"
    sudo sed -i "s|User=hotel-erp|User=$DEPLOY_USER|g" "/etc/systemd/system/$SERVICE_NAME.service"
    sudo sed -i "s|Group=hotel-erp|Group=$DEPLOY_USER|g" "/etc/systemd/system/$SERVICE_NAME.service"

    # Reload systemd
    sudo systemctl daemon-reload
    sudo systemctl enable "$SERVICE_NAME"

    success "Systemd service installed"
}

# Show deployment status
status() {
    info "Deployment Status"
    echo "=================="
    echo "Project Directory: $PROJECT_DIR"
    echo "Deploy Path: $DEPLOY_PATH"
    echo "Service Name: $SERVICE_NAME"
    echo "Deploy User: $DEPLOY_USER"
    echo ""

    if command -v systemctl &> /dev/null; then
        echo "Service Status (systemd):"
        sudo systemctl status "$SERVICE_NAME" --no-pager -l
    else
        echo "Service Status (process manager):"
        cd "$DEPLOY_PATH" 2>/dev/null && ./scripts/process-manager.sh status || echo "Service not found"
    fi

    echo ""
    if [[ -f "$DEPLOY_PATH/.last_backup" ]]; then
        echo "Last Backup: $(cat "$DEPLOY_PATH/.last_backup")"
    fi
}

# Show usage
usage() {
    echo "Hotel ERP Deployment Script"
    echo "Usage: $0 {deploy|rollback|status|install-service|start|stop|restart|health}"
    echo ""
    echo "Commands:"
    echo "  deploy          Perform zero-downtime deployment"
    echo "  rollback        Rollback to previous version"
    echo "  status          Show deployment status"
    echo "  install-service Install systemd service"
    echo "  start           Start the service"
    echo "  stop            Stop the service"
    echo "  restart         Restart the service gracefully"
    echo "  health          Check service health"
    echo ""
    echo "Environment Variables:"
    echo "  DEPLOY_USER     User to run the service (default: hotel-erp)"
    echo "  DEPLOY_PATH     Deployment directory (default: /opt/hotel-erp)"
    echo "  SERVICE_NAME    Systemd service name (default: hotel-erp)"
    echo "  PORT            Application port (default: 9000)"
}

# Main command handling
case "${1:-}" in
    deploy)
        zero_downtime_deploy
        ;;
    rollback)
        rollback
        ;;
    status)
        status
        ;;
    install-service)
        install_systemd_service
        ;;
    start)
        start_service
        wait_for_health
        ;;
    stop)
        stop_service
        ;;
    restart)
        graceful_restart
        wait_for_health
        ;;
    health)
        wait_for_health
        ;;
    *)
        usage
        exit 1
        ;;
esac
