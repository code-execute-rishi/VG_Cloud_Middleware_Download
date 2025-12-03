# VyomGarud GCS 🚁

![React](https://img.shields.io/badge/React-20232A?style=for-the-badge&logo=react&logoColor=61DAFB)
![Go](https://img.shields.io/badge/Go-00ADD8?style=for-the-badge&logo=go&logoColor=white)
![LiveKit](https://img.shields.io/badge/LiveKit-38B2AC?style=for-the-badge&logo=webrtc&logoColor=white)
![Vite](https://img.shields.io/badge/Vite-B73BFE?style=for-the-badge&logo=vite&logoColor=FFD62E)

**VyomGarud GCS** is a professional Ground Control Station (GCS) designed for monitoring and controlling UAVs. It features real-time telemetry visualization, low-latency video streaming via LiveKit, and an interactive map interface.

## 🌟 Features

- **Real-time Telemetry**: Live monitoring of Attitude (Roll, Pitch, Yaw), GPS Position, Battery Status, and Flight Modes.
- **Low-Latency Video**: Integrated WebRTC video feed using **LiveKit** for real-time situational awareness.
- **Secure Authentication**: Ed25519-based device authentication and claiming system.
- **Interactive Map**: Leaflet-based map visualization tracking the drone's path.
- **Responsive Design**: Modern, dark-themed UI built with Tailwind CSS.

## 🏗️ Architecture

- **Backend**: Go (Golang) server handling device authentication, claiming, and LiveKit token generation.
- **Device Middleware**: Go application running on the drone (or edge) that bridges MAVLink telemetry to LiveKit.
- **Frontend**: React application using LiveKit SDK to display video and telemetry.

## 🚀 Getting Started

### Prerequisites
- **Go** (v1.21 or higher)
- **Node.js** (v18 or higher)
- **LiveKit Cloud Account** (API Key, Secret, and URL)

### 1. Configuration

Create a `.env` file in the `backend/` directory with your LiveKit credentials:

```env
LIVEKIT_API_KEY=your_api_key
LIVEKIT_API_SECRET=your_api_secret
LIVEKIT_URL=wss://your-project.livekit.cloud
```

### 2. Run the Backend

The backend handles authentication and token generation.

```bash
cd backend
go mod tidy
go run main.go
```
*Server runs on port 8080.*

### 3. Run the Device Middleware (Drone Side)

This simulates the drone. It generates a pairing code and connects to LiveKit.

```bash
cd device-middleware
go mod tidy
go run main.go
```

1.  Copy the **Pairing Code** displayed in the terminal (e.g., `7f9387a6`).
2.  Copy the **Public Key** (if needed for manual claiming, though the UI/Curl handles this).

### 4. Claim the Device

Register the device with the backend using the pairing code.

```bash
# Replace PAIRING_CODE and PUBLIC_KEY with values from middleware output
curl -X POST http://localhost:8080/api/v1/devices/claim \
  -H "Content-Type: application/json" \
  -d '{"pairing_code": "PAIRING_CODE", "public_key": "PUBLIC_KEY"}'
```

*Once claimed, the middleware will automatically connect to LiveKit.*

### 5. Run the Frontend (GCS Dashboard)

```bash
cd frontend
npm install
npm run dev
```

Open your browser to `http://localhost:5173`. You should see the live video feed and telemetry from the middleware.

## 🛠️ Tech Stack

- **Frontend**: React, Vite, Tailwind CSS, LiveKit React SDK
- **Backend**: Go (Standard Lib + Gorilla Mux/RS Cors)
- **Middleware**: Go, Gomavlib, LiveKit Go SDK
- **Communication**: LiveKit (WebRTC Data Channels for Telemetry & Video)

---
*Developed for Senior Internship Project*
