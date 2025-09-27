#!/bin/bash

APP_NAME="erp"
BINARY_NAME="erp"
SERVICE_NAME="erp"

echo "🚀 Deploying $APP_NAME..."

# Step 1: Pull latest main repo and submodules
echo "📥 Pulling latest code..."
git pull origin main || { echo "❌ Git pull failed"; exit 1; }

# Step 3: Build backend and drop binary in root
echo "⚙️ Building backend..."
go build -o "$BINARY_NAME" . || { echo "❌ Backend build failed"; exit 1; }

# Step 4: Restart systemd service
echo "🔁 Restarting $SERVICE_NAME service..."
sudo systemctl restart "$SERVICE_NAME"

# Step 5: Check service status
echo "📋 Status:"
sudo systemctl status "$SERVICE_NAME" --no-pager

echo "✅ Deployment complete."
