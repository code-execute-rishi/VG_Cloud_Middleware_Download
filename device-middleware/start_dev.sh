#!/bin/bash
export BACKEND_URL="https://backend.internetlinkpro.vyomgarud.com"

echo "🚀 Starting Vyom Middleware (Development Mode)..."
echo "logs are being written to their respective log files (api.log, etc) and output below."

# Kill any existing instances to free ports (Try with sudo if needed)
echo "Cleaning up..."
sudo systemctl stop vyom-api 2>/dev/null
sudo systemctl stop vyom-telemetry 2>/dev/null
pkill -f "vyom-api"
pkill -f "vyom-telemetry"
pkill -f "main.go"
pkill -f "middleware-bin"

# Run the middleware as ROOT to allow ZeroTier management & Hardware Access
sudo -E go run main.go
