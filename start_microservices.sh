#!/bin/bash

echo "🚀 Starting Vyom Microservices..."

if [ "$EUID" -ne 0 ]; then 
  echo "❌ Please run as root (use sudo)"
  exit 1
fi

mkdir -p bin

# Build everything
echo "🏗️  Building..."
make all

if [ $? -ne 0 ]; then
    echo "❌ Build Failed"
    exit 1
fi

echo "✅ Build Complete. Launching Services..."

# cleanup old
pkill -f vyom-api
pkill -f vyom-zerotier
pkill -f vyom-telemetry
pkill -f vyom-livekit
pkill -f vyom-camera

# Start services in background
./bin/vyom-api &
PID_API=$!

./bin/vyom-zerotier &
PID_ZT=$!

./bin/vyom-telemetry &
PID_TLM=$!

./bin/vyom-livekit &
PID_LK=$!

./bin/vyom-camera &
PID_CAM=$!

echo "🌟 All Services Started!"
echo "API at http://localhost:8085"

# Wait for Ctrl+C
trap "kill $PID_API $PID_ZT $PID_TLM $PID_LK $PID_CAM; exit" INT TERM
wait
