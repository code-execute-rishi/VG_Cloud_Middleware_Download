# VyomGarud GCS 🚁

![Go](https://img.shields.io/badge/Go-00ADD8?style=for-the-badge&logo=go&logoColor=white)
![LiveKit](https://img.shields.io/badge/LiveKit-38B2AC?style=for-the-badge&logo=webrtc&logoColor=white)
![Vite](https://img.shields.io/badge/Vite-B73BFE?style=for-the-badge&logo=vite&logoColor=FFD62E)

**VyomGarud GCS** is a professional Ground Control Station (GCS) designed for monitoring and controlling UAVs. It features real-time telemetry visualization, low-latency video streaming via LiveKit, and an interactive map interface.

## 🌟 Features

- **Real-time Telemetry**: Live monitoring of Attitude (Roll, Pitch, Yaw), GPS Position, Battery Status, and Flight Modes.
- **Low-Latency Video**: Integrated WebRTC video feed using **LiveKit** for real-time situational awareness.
- **Network Health Dashboard**: Built-in monitoring for **LiveKit** (Room/Peers) and **ZeroTier** (VPN IP) connectivity.
- **QR Code Activation**: Seamless device pairing using **Clerk Device Flow**. Simply scan to link.
- **Secure Authentication**: Ed25519-based device authentication and Identity Management.
- **Factory Reset**: Remote "Kill Switch" to wipe device identity and configuration.

## 🏗️ Architecture

- **Backend**: Go (Golang) server handling device authentication, claiming, and LiveKit token generation (Deployed on Render).
- **Device Middleware**: Go application running on the drone (or edge) that bridges MAVLink telemetry to LiveKit.
- **Frontend**: Next.js application using LiveKit SDK to display video and telemetry.

## 🚀 Getting Started

### Prerequisites
- **Go** (v1.21 or higher)
- **Node.js** (v18 or higher)
- **LiveKit Cloud Account** (API Key, Secret, and URL)

### 1. Configuration (Backend)

The backend is configured to run on Render. Ensure your environment variables (LIVEKIT keys, POSTGRES_URL, CLERK_KEYS) are set.

### 2. Run the Device Middleware (Drone Side)

The middleware uses a Display/CLI based setup wizard.

```bash
cd vyom-gcs/device-middleware

# Build the binary
go build -o middleware-bin .

# Run
./middleware-bin
```

1.  On first boot, the device will enter **Setup Mode**.
2.  A **QR Code** (and a backup text code) will appear on the screen/terminal.
3.  **Scan the QR Code** with your phone or visit the link displayed.
4.  Log in with your Clerk account and click **"Connect Device"**.
5.  The device will automatically authorize, generate a secure identity, and reboot into **Mission Mode**.

> **Note**: The device's Identity (`identity.json`) and Config (`config.json`) are stored locally.

### 3. Run the Cloud Frontend (GCS Dashboard)

```bash
cd VG-Cloud-GCS-Frontend-main
npm install
npm run dev
```

Open your browser to the local or deployed URL (e.g., `https://middleware-gcs-assigment.vercel.app`).
- **Dashboard**: View your claimed devices.
- **Activation**: `/activate` handles the QR code linking.

## 🛠️ Tech Stack

- **Frontend**: Next.js, Tailwind CSS, LiveKit React SDK, Clerk Auth
- **Backend**: Go (Standard Lib + Gorilla Mux), PostgreSQL, Clerk SDK
- **Middleware**: Go, Gomavlib, LiveKit Go SDK, GStreamer (Video Pipeline)
- **Communication**: LiveKit (WebRTC Data Channels for Telemetry & Video)

## ⚠️ Troubleshooting / Factory Reset

If you need to move the device to a new account or fix a bad configuration:

1.  Go to the **Frontend Dashboard**.
2.  Find the device card.
3.  Click the **Delete (Trash Icon)** button.
4.  The device will receive a command to **Self-Destruct Identity**.
5.  It will stop the camera, wipe `identity.json` and `config.json`, and **Reboot** back to the QR Code screen.

---
*Developed for Senior Internship Project*
