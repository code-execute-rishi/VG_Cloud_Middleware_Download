#!/bin/bash
export BACKEND_URL="http://98.92.19.117:8080"
# Open browser in background after short delay
(sleep 2 && xdg-open http://localhost:8081) &
go run main.go
