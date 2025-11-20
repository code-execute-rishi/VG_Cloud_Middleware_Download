/**
 * VyomGarud GCS Backend Server
 * 
 * This server acts as the central telemetry hub for the Ground Control Station.
 * It performs three main functions:
 * 1. Listens for UDP packets from the drone (via ZeroTier VPN) on port 14550.
 * 2. Parses raw binary MAVLink v2 packets into JSON objects.
 * 3. Broadcasts telemetry data to the React Frontend via Socket.io.
 * 
 * Additionally, it hosts a PeerJS server to facilitate WebRTC signaling for video.
 */

const dgram = require('dgram');
const express = require('express');
const http = require('http');
const { Server } = require('socket.io');
const cors = require('cors');
const { PeerServer } = require('peer');
const { Readable } = require('stream');
const { MavLinkPacketSplitter, MavLinkPacketParser } = require('node-mavlink');
const { common, minimal } = require('mavlink-mappings');

// Handle BigInt serialization for JSON (MAVLink uses uint64_t)
BigInt.prototype.toJSON = function() { return this.toString(); };

const app = express();
app.use(cors());

// Configure Socket.io with CORS to allow Public IP access
const server = http.createServer(app);
const io = new Server(server, {
  cors: {
    origin: ["http://98.92.19.117", "http://localhost:5173"],
    methods: ["GET", "POST"]
  }
});

const PORT = 3001;
const UDP_PORT = 14550;
const PEER_PORT = 9000;

/**
 * PeerJS Server for WebRTC Signaling
 * Facilitates P2P connection between Drone Camera and GCS Dashboard.
 */
const peerServer = PeerServer({ port: PEER_PORT, path: '/' });
console.log(`PeerJS server running on port ${PEER_PORT}`);

/**
 * UDP Socket for Telemetry
 * Receives MAVLink packets forwarded by ZeroTier from the drone.
 */
const udpSocket = dgram.createSocket('udp4');

// Registry of MAVLink messages to parse (Minimal + Common set)
const REGISTRY = {
  ...minimal.REGISTRY,
  ...common.REGISTRY,
};

// Stream to feed UDP buffer into MAVLink parser
const udpStream = new Readable({
  read() {}
});

// MAVLink Parser Pipeline: Splitter -> Parser
const reader = udpStream
  .pipe(new MavLinkPacketSplitter())
  .pipe(new MavLinkPacketParser());

/**
 * MAVLink Packet Handler
 * Decodes binary packets and emits relevant data to the frontend.
 */
reader.on('data', (packet) => {
  const clazz = REGISTRY[packet.header.msgid];
  if (clazz) {
    const message = packet.protocol.data(packet.payload, clazz);
    
    // Handle Heartbeat (System Status & Mode)
    if (message instanceof minimal.Heartbeat) {
      io.emit('telemetry', {
        timestamp: new Date().toISOString(),
        heartbeat: {
          custom_mode: message.customMode,
          base_mode: message.baseMode,
          system_status: message.systemStatus,
          mavlink_version: message.mavlinkVersion
        }
      });
      return;
    }

    let data = {
      timestamp: new Date().toISOString()
    };
    let shouldEmit = false;

    // Handle Global Position (Lat/Lon/Alt)
    if (message instanceof common.GlobalPositionInt) {
      data.global_position_int = {
        lat: message.lat / 1E7,
        lon: message.lon / 1E7,
        alt: message.alt / 1000,
        relative_alt: message.relativeAlt / 1000,
        heading: message.hdg / 100
      };
      shouldEmit = true;
    } 
    // Handle Attitude (Roll/Pitch/Yaw)
    else if (message instanceof common.Attitude) {
      data.attitude = {
        roll: message.roll,
        pitch: message.pitch,
        yaw: message.yaw
      };
      shouldEmit = true;
    } 
    // Handle System Status (Battery/Sensors)
    else if (message instanceof common.SysStatus) {
      data.sys_status = {
        voltage: message.voltageBattery / 1000,
        battery_remaining: message.batteryRemaining,
        current: (message.currentBattery || 0) / 100, // cA to A
        sensors: message.onboardControlSensorsHealth
      };
      shouldEmit = true;
    } 
    // Handle GPS Raw (Fix Type/Satellites)
    else if (message instanceof common.GpsRawInt) {
      data.gps_raw_int = {
        fix_type: message.fixType,
        satellites_visible: message.satellitesVisible
      };
      shouldEmit = true;
    } 
    // Handle HUD Data (Speed/Throttle)
    else if (message instanceof common.VfrHud) {
      data.vfr_hud = {
        airspeed: message.airspeed,
        groundspeed: message.groundspeed,
        heading: message.heading,
        throttle: message.throttle,
        alt: message.alt
      };
      shouldEmit = true;
    } 
    // Handle Status Text (Logs/Messages)
    else if (message instanceof common.StatusText) {
      let logText = message.text;
      // Convert buffer to string if necessary and remove null bytes
      if (typeof logText !== 'string') {
         logText = String.fromCharCode(...message.text).replace(/\0/g, '');
      }
      io.emit('status-text', logText);
    }

    if (shouldEmit) {
      io.emit('telemetry', data);
    }
  }
});

// UDP Event Listeners
udpSocket.on('listening', () => {
  const address = udpSocket.address();
  console.log(`UDP socket listening on ${address.address}:${address.port}`);
});

udpSocket.on('message', (msg, rinfo) => {
  udpStream.push(msg);
});

udpSocket.bind(UDP_PORT);

server.listen(PORT, () => {
  console.log(`Server is running on http://localhost:${PORT}`);
});

io.on('connection', (socket) => {
  console.log('A user connected to Socket.io');
  socket.on('disconnect', () => {
    console.log('User disconnected');
  });
});
