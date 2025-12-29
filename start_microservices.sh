#!/bin/bash

echo "🚀 Starting Vyom Microservices..."

if [ "$EUID" -ne 0 ]; then 
  echo "❌ Please run as root (use sudo)"
  exit 1
fi

# Production Startup Script
BIN_DIR="/opt/vyom/bin"
LOG_DIR="/var/log/vyom"

mkdir -p $LOG_DIR

if [ ! -d "$BIN_DIR" ]; then
    echo "⚠️  Bin directory not found at $BIN_DIR. Using local './bin' for dev."
    BIN_DIR="./bin"
    LOG_DIR="./logs"
    mkdir -p $LOG_DIR
fi

echo "✅ Launching Services from $BIN_DIR..."

# cleanup old
pkill -f vyom-api
pkill -f vyom-zerotier
pkill -f vyom-telemetry
pkill -f vyom-livekit
pkill -f vyom-camera

# Start API Service (with auto-restart loop)
echo "🚀 Starting API Service..."
(
    while true; do
        $BIN_DIR/vyom-api >> $LOG_DIR/api.log 2>&1
        echo "⚠️ vyom-api crashed or restarted. Respawning in 1s..." >> $LOG_DIR/api.log
        sleep 1
    done
) &
PID_API=$!

# Start other services (Simple background)
$BIN_DIR/vyom-zerotier >> $LOG_DIR/zerotier.log 2>&1 &
PID_ZT=$!

$BIN_DIR/vyom-telemetry >> $LOG_DIR/telemetry.log 2>&1 &
PID_TLM=$!

$BIN_DIR/vyom-livekit >> $LOG_DIR/livekit.log 2>&1 &
PID_LK=$!

$BIN_DIR/vyom-camera >> $LOG_DIR/camera.log 2>&1 &
PID_CAM=$!

echo "🌟 All Services Started!"
echo "API at http://localhost:8085"

# Wait for Ctrl+C
trap "kill $PID_API $PID_ZT $PID_TLM $PID_LK $PID_CAM; exit" INT TERM
wait
