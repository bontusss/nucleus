#!/bin/bash

# Hotel ERP Process Manager
# Handles graceful shutdown, restart, and monitoring of the application

set -e

# Configuration
APP_NAME="herp"
APP_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
PID_FILE="$APP_DIR/tmp/$APP_NAME.pid"
LOG_FILE="$APP_DIR/tmp/$APP_NAME.log"
APP_BINARY="$APP_DIR/bin/app"
RESTART_DELAY=5

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Ensure tmp directory exists
mkdir -p "$APP_DIR/tmp"

# Logging function
log() {
    echo -e "$(date '+%Y-%m-%d %H:%M:%S') - $1" | tee -a "$LOG_FILE"
}

# Check if process is running
is_running() {
    if [[ -f "$PID_FILE" ]]; then
        local pid=$(cat "$PID_FILE")
        if kill -0 "$pid" 2>/dev/null; then
            return 0
        else
            rm -f "$PID_FILE"
            return 1
        fi
    fi
    return 1
}

# Get process PID
get_pid() {
    if [[ -f "$PID_FILE" ]]; then
        cat "$PID_FILE"
    fi
}

# Start the application
start() {
    if is_running; then
        log "${YELLOW}$APP_NAME is already running (PID: $(get_pid))${NC}"
        return 1
    fi

    log "${BLUE}Starting $APP_NAME...${NC}"

    # Build the application first
    cd "$APP_DIR"
    if ! make_binary; then
        log "${RED}Failed to build $APP_NAME${NC}"
        return 1
    fi

    # Start the application in background
    nohup "$APP_BINARY" >> "$LOG_FILE" 2>&1 &
    local pid=$!

    # Save PID
    echo "$pid" > "$PID_FILE"

    # Wait a moment and check if it's still running
    sleep 2
    if kill -0 "$pid" 2>/dev/null; then
        log "${GREEN}$APP_NAME started successfully (PID: $pid)${NC}"
        return 0
    else
        log "${RED}$APP_NAME failed to start${NC}"
        rm -f "$PID_FILE"
        return 1
    fi
}

# Stop the application gracefully
stop() {
    if ! is_running; then
        log "${YELLOW}$APP_NAME is not running${NC}"
        return 1
    fi

    local pid=$(get_pid)
    log "${BLUE}Stopping $APP_NAME gracefully (PID: $pid)...${NC}"

    # Send SIGTERM for graceful shutdown
    kill -TERM "$pid"

    # Wait for graceful shutdown (max 30 seconds)
    local count=0
    while kill -0 "$pid" 2>/dev/null && [[ $count -lt 30 ]]; do
        sleep 1
        ((count++))
    done

    # Check if process is still running
    if kill -0 "$pid" 2>/dev/null; then
        log "${YELLOW}Graceful shutdown timeout, forcing stop...${NC}"
        kill -KILL "$pid"
        sleep 2
    fi

    # Clean up PID file
    rm -f "$PID_FILE"
    log "${GREEN}$APP_NAME stopped${NC}"
    return 0
}

# Restart the application gracefully
restart() {
    log "${BLUE}Restarting $APP_NAME...${NC}"

    if is_running; then
        stop
        sleep "$RESTART_DELAY"
    fi

    start
}

# Graceful restart using SIGUSR1 (if supported by app)
graceful_restart() {
    if ! is_running; then
        log "${YELLOW}$APP_NAME is not running, starting instead...${NC}"
        start
        return $?
    fi

    local pid=$(get_pid)
    log "${BLUE}Sending graceful restart signal to $APP_NAME (PID: $pid)...${NC}"

    # Send SIGUSR1 for graceful restart
    if kill -USR1 "$pid"; then
        log "${GREEN}Graceful restart signal sent${NC}"

        # Wait for the process to handle the restart
        sleep "$RESTART_DELAY"

        # Check if we need to start a new process
        if ! is_running; then
            log "${BLUE}Process terminated, starting new instance...${NC}"
            start
        fi
    else
        log "${RED}Failed to send graceful restart signal${NC}"
        return 1
    fi
}

# Build the application binary
make_binary() {
    log "${BLUE}Building $APP_NAME...${NC}"
    if make -s build 2>/dev/null || go build -o "$APP_BINARY"; then
        log "${GREEN}Build successful${NC}"
        return 0
    else
        log "${RED}Build failed${NC}"
        return 1
    fi
}

# Show application status
status() {
    if is_running; then
        local pid=$(get_pid)
        log "${GREEN}$APP_NAME is running (PID: $pid)${NC}"

        # Show additional process info
        if ps -p "$pid" -o pid,ppid,cmd,start,time > /dev/null 2>&1; then
            echo
            ps -p "$pid" -o pid,ppid,cmd,start,time
        fi

        # Show recent logs
        if [[ -f "$LOG_FILE" ]]; then
            echo
            echo "Recent logs:"
            tail -10 "$LOG_FILE"
        fi
    else
        log "${RED}$APP_NAME is not running${NC}"
        return 1
    fi
}

# Monitor the application and restart if needed
monitor() {
    log "${BLUE}Starting monitor for $APP_NAME...${NC}"

    while true; do
        if ! is_running; then
            log "${YELLOW}$APP_NAME is not running, attempting to start...${NC}"
            start
        fi
        sleep 30
    done
}

# Show logs
logs() {
    local lines=${1:-50}
    if [[ -f "$LOG_FILE" ]]; then
        tail -n "$lines" "$LOG_FILE"
    else
        echo "No log file found at $LOG_FILE"
    fi
}

# Follow logs
follow_logs() {
    if [[ -f "$LOG_FILE" ]]; then
        tail -f "$LOG_FILE"
    else
        echo "No log file found at $LOG_FILE"
        echo "Starting tail on empty log file..."
        touch "$LOG_FILE"
        tail -f "$LOG_FILE"
    fi
}

# Health check
health() {
    if ! is_running; then
        echo "Service is not running"
        return 1
    fi

    # Try to hit the health endpoint
    local health_url="http://localhost:${PORT:-9000}/health"
    if command -v curl >/dev/null 2>&1; then
        if curl -sf "$health_url" >/dev/null 2>&1; then
            echo "Service is healthy"
            return 0
        else
            echo "Service is running but health check failed"
            return 1
        fi
    else
        echo "Service is running (curl not available for health check)"
        return 0
    fi
}

# Clean up old files
cleanup() {
    log "${BLUE}Cleaning up old files...${NC}"

    # Remove old PID file if process is not running
    if [[ -f "$PID_FILE" ]] && ! is_running; then
        rm -f "$PID_FILE"
        log "Removed stale PID file"
    fi

    # Rotate logs if they're too large (>100MB)
    if [[ -f "$LOG_FILE" ]] && [[ $(stat -f%z "$LOG_FILE" 2>/dev/null || stat -c%s "$LOG_FILE" 2>/dev/null || echo 0) -gt 104857600 ]]; then
        mv "$LOG_FILE" "$LOG_FILE.old"
        touch "$LOG_FILE"
        log "Rotated large log file"
    fi

    log "${GREEN}Cleanup completed${NC}"
}

# Show usage information
usage() {
    echo "Usage: $0 {start|stop|restart|graceful-restart|status|monitor|logs|follow-logs|health|cleanup}"
    echo
    echo "Commands:"
    echo "  start             Start the application"
    echo "  stop              Stop the application gracefully"
    echo "  restart           Stop and start the application"
    echo "  graceful-restart  Send graceful restart signal (SIGUSR1)"
    echo "  status            Show application status"
    echo "  monitor           Monitor and auto-restart if needed"
    echo "  logs [N]          Show last N lines of logs (default: 50)"
    echo "  follow-logs       Follow logs in real-time"
    echo "  health            Check application health"
    echo "  cleanup           Clean up old files and rotate logs"
    echo
    echo "Environment variables:"
    echo "  PORT              Application port (default: 9000)"
    echo
    echo "Files:"
    echo "  PID file:         $PID_FILE"
    echo "  Log file:         $LOG_FILE"
    echo "  Binary:           $APP_BINARY"
}

# Main command handling
case "${1:-}" in
    start)
        start
        ;;
    stop)
        stop
        ;;
    restart)
        restart
        ;;
    graceful-restart|reload)
        graceful_restart
        ;;
    status)
        status
        ;;
    monitor)
        monitor
        ;;
    logs)
        logs "${2:-50}"
        ;;
    follow-logs|tail)
        follow_logs
        ;;
    health)
        health
        ;;
    cleanup)
        cleanup
        ;;
    *)
        usage
        exit 1
        ;;
esac
