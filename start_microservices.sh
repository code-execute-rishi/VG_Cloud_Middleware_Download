#!/bin/bash

echo "🚀 Starting Vyom Microservices..."

if [ "$EUID" -ne 0 ]; then 
  echo "❌ Please run as root (use sudo)"
  exit 1
fi

# Production Startup Script
BIN_DIR="/opt/vyom/bin"

if [ ! -d "$BIN_DIR" ]; then
    echo "⚠️  Bin directory not found at $BIN_DIR. Using local './bin' for dev."
    BIN_DIR="./bin"
fi

echo "✅ Launching Services from $BIN_DIR..."

# cleanup old
pkill -f vyom-api
pkill -f vyom-zerotier
pkill -f vyom-telemetry
pkill -f vyom-livekit
pkill -f vyom-camera

# Start services in background
# Start services in background
$BIN_DIR/vyom-api &
PID_API=$!

$BIN_DIR/vyom-zerotier &
PID_ZT=$!

$BIN_DIR/vyom-telemetry &
PID_TLM=$!

$BIN_DIR/vyom-livekit &
PID_LK=$!

$BIN_DIR/vyom-camera &
PID_CAM=$!

echo "🌟 All Services Started!"
echo "API at http://localhost:8085"

# Wait for Ctrl+C
trap "kill $PID_API $PID_ZT $PID_TLM $PID_LK $PID_CAM; exit" INT TERM
wait
