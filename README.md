# Vyom Device Middleware

This repository contains the **Device Middleware** for the Vyom GCS, refactored into a microservices architecture. It runs on the drone's companion computer (e.g., Raspberry Pi) and manages telemetry, video streaming, and cloud connectivity.

## Architecture 🏗️

The middleware is split into 5 independent services for better stability and fault tolerance:

1.  **`vyom-api`** (Port `:8085`):
    *   **The Brain.** Manages device identity, configuration, and status.
    *   Hosts the local UI (React app).
    *   Orchestrates configuration for other services.
2.  **`vyom-telemetry`**:
    *   **The Pilot.** Connects to the Flight Controller (Serial/USB or UDP/SITL).
    *   Parses MAVLink data (using `gomavlib`).
    *   Forwards telemetry to the Cloud (LiveKit) and local GCS apps (UDP Proxy).
3.  **`vyom-livekit`** (UDP `:5000` Ingress):
    *   **The Uplink.** Connects to the LiveKit Cloud Server via WebRTC.
    *   Relays Video (H.264) and Telemetry (Data Channel) to the dashboard.
4.  **`vyom-camera`**:
    *   **The Eye.** Captures video from USB Webcams or CSI cameras (via GStreamer).
    *   Streams H.264 video to `vyom-livekit`.
    *   Serves a local MJPEG stream for the Local UI.
5.  **`vyom-zerotier`**:
    *   **The Network.** Monitors ZeroTier connection status.
    *   Reports public IP and Network ID to the Cloud.

## Getting Started 🚀

### Prerequisites
*   Go 1.25+
*   `make`
*   `libgstreamer1.0-dev` (for camera)

### Build
To build all microservices:
```bash
cd device-middleware
make all
```
This produces binaries in the `bin/` directory.

### Run
To start all services with the orchestrator script:
```bash
sudo ./start_microservices.sh
```
*Note: `sudo` is often required for Serial port detection and ZeroTier interaction.*

## Configuration ⚙️

Configuration is managed via the **Local UI** (http://localhost:8085) or the Cloud Dashboard.
*   **Identity:** Stored in `device.json`.
*   **Telemetry:** Auto-detects `/dev/ttyACM*` or `/dev/ttyUSB*`. Falls back to UDP `:14550` for SITL.

## Development 🛠️

*   **Logs:** Each service logs to stdout. The start script pipes them to `vyom-*.log`.
*   **Mock Mode:** Telemetry will auto-connect to SITL if no hardware is found.
