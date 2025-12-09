# LiveKit Setup Guide

## Environment Variables

Add the following environment variables to your `.env.local` file:

```env
# LiveKit Configuration
LIVEKIT_API_KEY=your_livekit_api_key_here
LIVEKIT_API_SECRET=your_livekit_api_secret_here
NEXT_PUBLIC_LIVEKIT_URL=wss://your-livekit-server.com

# For local development with LiveKit server:
# LIVEKIT_API_KEY=devkey
# LIVEKIT_API_SECRET=secret
# NEXT_PUBLIC_LIVEKIT_URL=ws://localhost:7880
```

## Room Structure

Each drone has its own LiveKit room named: `drone-{DRONE_ID}`

For example:
- `drone-DRONE-001`
- `drone-DRONE-002`
- etc.

## Features Implemented

1. **Live Video Streaming**: Real-time video stream through WebRTC
2. **SSH Terminal**: Terminal access using xterm.js via LiveKit data channels
3. **Control Panel**: Dashboard for drone telemetry and control

## API Routes

### POST `/api/livekit/token`
Generates a LiveKit access token for connecting to a room.

**Request Body:**
```json
{
  "roomName": "drone-DRONE-001",
  "participantName": "user-123",
  "participantIdentity": "user-123"
}
```

**Response:**
```json
{
  "token": "eyJ...",
  "wsUrl": "wss://your-livekit-server.com"
}
```

## Backend Integration (Go)

When implementing the Go backend, you'll need to:

1. Replace the token generation endpoint (`/api/livekit/token`) with your Go backend
2. Handle data channels for SSH terminal communication
3. Publish video streams to LiveKit rooms from drone sources
4. Handle telemetry data through LiveKit data channels

## Required Variables for Go Backend

The frontend will send these to your backend:
- `roomName`: `drone-{DRONE_ID}`
- `participantName`: User identifier
- `participantIdentity`: User identity for token

Your Go backend should generate tokens with appropriate permissions for:
- `canPublish`: true (for publishing video/data)
- `canSubscribe`: true (for receiving streams)
- `canPublishData`: true (for SSH terminal)

