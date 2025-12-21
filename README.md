# VyomGarud GCS 🚁

![React](https://img.shields.io/badge/React-20232A?style=for-the-badge&logo=react&logoColor=61DAFB)
![Go](https://img.shields.io/badge/Go-00ADD8?style=for-the-badge&logo=go&logoColor=white)
![LiveKit](https://img.shields.io/badge/LiveKit-38B2AC?style=for-the-badge&logo=webrtc&logoColor=white)
![Vite](https://img.shields.io/badge/Vite-B73BFE?style=for-the-badge&logo=vite&logoColor=FFD62E)

**VyomGarud GCS** is a professional Ground Control Station (GCS) designed for monitoring and controlling UAVs. It features real-time telemetry visualization, low-latency video streaming via LiveKit, and an interactive map interface.

## 🌟 Features

- **Real-time Telemetry**: Live monitoring of Attitude (Roll, Pitch, Yaw), GPS Position, Battery Status, and Flight Modes.
- **Low-Latency Video**: Integrated WebRTC video feed using **LiveKit** for real-time situational awareness.
- **Network Health Dashboard**: Built-in monitoring for **LiveKit** (Room/Peers) and **ZeroTier** (VPN IP) connectivity.
- **Web-Based Setup**: Onboard configuration UI for easy WiFi provisioning and device binding.
- **Secure Authentication**: Ed25519-based device authentication and claiming system.
- **Interactive Map**: Leaflet-based map visualization tracking the drone's path.

## 🏗️ Architecture

- **Backend**: Go (Golang) server handling device authentication, claiming, and LiveKit token generation (Deployed on Render).
- **Device Middleware**: Go application running on the drone (or edge) that bridges MAVLink telemetry to LiveKit.
- **Frontend**: React application using LiveKit SDK to display video and telemetry.

## 🚀 Getting Started

### Prerequisites
- **Go** (v1.21 or higher)
- **Node.js** (v18 or higher)
- **LiveKit Cloud Account** (API Key, Secret, and URL)

### 1. Configuration (Backend)

The backend is configured to run on Render. Ensure your environment variables (LIVEKIT keys, POSTGRES_URL) are set in your deployment provider.

### 2. Run the Device Middleware (Drone Side)

The middleware now includes a web-based setup wizard.

```bash
cd device-middleware
# Build the binary (includes UI)
go build -o middleware-bin .
# Run
./middleware-bin
```

1.  Connect to the same network as the device.
2.  Open **`http://<DEVICE_IP>:8085`** in your browser.
3.  Enter the **Pairing Code** shown in the Web UI into your Cloud Dashboard to claim the device.

### 3. Run the Cloud Frontend (GCS Dashboard)

```bash
cd frontend
npm install
npm run dev
```

Open your browser to the local or deployed URL (e.g., `https://middleware-gcs-assigment.vercel.app`). You should see the live video feed and telemetry from the middleware.

## 🛠️ Tech Stack

- **Frontend**: React, Vite, Tailwind CSS, LiveKit React SDK
- **Backend**: Go (Standard Lib + Gorilla Mux/RS Cors)
- **Middleware**: Go, Gomavlib, LiveKit Go SDK
- **Communication**: LiveKit (WebRTC Data Channels for Telemetry & Video)

---
*Developed for Senior Internship Project*
