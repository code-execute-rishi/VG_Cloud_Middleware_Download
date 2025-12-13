#!/bin/bash
export BACKEND_URL="http://4.247.135.200"
# Cleanup previous instances (Nuclear Option)
echo "Force cleaning up old processes..."
pkill -9 -f middleware-bin || true
pkill -9 -f ffmpeg || true
fuser -k -9 /dev/video0 2>/dev/null || true
time.Sleep 1 2>/dev/null || sleep 1

# Build and Run
echo "Building Middleware..."
go build -o middleware-bin . || exit 1

# Build UI if missing (Handles the local React app)
if [ ! -d "./ui/dist" ]; then
    echo "Building Local Dashboard UI..."
    cd ui
    npm install
    npm run build
    cd ..
fi

# Open browser if a display is detected
if [ -n "$DISPLAY" ]; then
    (sleep 5 && xdg-open http://localhost:8080) &
fi

# Run the binary
./middleware-bin "$@"
