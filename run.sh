#!/bin/bash
export BACKEND_URL="http://4.247.135.200:8080"
# Build first for performance
go build -o middleware-bin main.go || exit 1

# Open browser if a display is detected
if [ -n "$DISPLAY" ]; then
    (sleep 5 && xdg-open http://localhost:8080) &
fi

# Run the binary
./middleware-bin
