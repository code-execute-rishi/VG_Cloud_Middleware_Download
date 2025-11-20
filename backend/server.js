const dgram = require('dgram');
const express = require('express');
const http = require('http');
const { Server } = require('socket.io');
const cors = require('cors');
const { PeerServer } = require('peer');
const { Readable } = require('stream');
const { MavLinkPacketSplitter, MavLinkPacketParser } = require('node-mavlink');
const { common, minimal } = require('mavlink-mappings');

// Handle BigInt serialization
BigInt.prototype.toJSON = function() { return this.toString(); };

const app = express();
app.use(cors());

const server = http.createServer(app);
const io = new Server(server, {
  cors: {
    origin: "*",
    methods: ["GET", "POST"]
  }
});

const PORT = 3001;
const UDP_PORT = 14550;
const PEER_PORT = 9000;

// PeerJS Server
const peerServer = PeerServer({ port: PEER_PORT, path: '/' });
console.log(`PeerJS server running on port ${PEER_PORT}`);

// Create UDP socket for telemetry
const udpSocket = dgram.createSocket('udp4');

// Create a registry of MAVLink messages we want to parse
const REGISTRY = {
  ...minimal.REGISTRY,
  ...common.REGISTRY,
};

// Create a stream to feed UDP data into the MAVLink parser
const udpStream = new Readable({
  read() {}
});

// Set up the MAVLink parser pipeline
const reader = udpStream
  .pipe(new MavLinkPacketSplitter())
  .pipe(new MavLinkPacketParser());

reader.on('data', (packet) => {
  const clazz = REGISTRY[packet.header.msgid];
  if (clazz) {
    const message = packet.protocol.data(packet.payload, clazz);
    
    // Log Heartbeat
    if (message instanceof minimal.Heartbeat) {
      // console.log('Heartbeat received');
      // Emit heartbeat data for mode/arming
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

    if (message instanceof common.GlobalPositionInt) {
      data.global_position_int = {
        lat: message.lat / 1E7,
        lon: message.lon / 1E7,
        alt: message.alt / 1000,
        relative_alt: message.relativeAlt / 1000, // Fixed camelCase
        heading: message.hdg / 100
      };
      shouldEmit = true;
    } else if (message instanceof common.Attitude) {
      data.attitude = {
        roll: message.roll,
        pitch: message.pitch,
        yaw: message.yaw
      };
      shouldEmit = true;
    } else if (message instanceof common.SysStatus) {
      data.sys_status = {
        voltage: message.voltageBattery / 1000,
        battery_remaining: message.batteryRemaining,
        current: (message.currentBattery || 0) / 100, // cA to A
        sensors: message.onboardControlSensorsHealth
      };
      shouldEmit = true;
    } else if (message instanceof common.GpsRawInt) {
      data.gps_raw_int = {
        fix_type: message.fixType,
        satellites_visible: message.satellitesVisible
      };
      shouldEmit = true;
    } else if (message instanceof common.VfrHud) {
      data.vfr_hud = {
        airspeed: message.airspeed,
        groundspeed: message.groundspeed,
        heading: message.heading,
        throttle: message.throttle,
        alt: message.alt
      };
      shouldEmit = true;
    } else if (message instanceof common.StatusText) {
      // Handle STATUSTEXT for logs
      // The 'text' field is usually a char array (buffer) or string depending on the parser
      // We need to ensure it's a clean string
      let logText = message.text;
      if (typeof logText !== 'string') {
         // If it's a buffer/array, convert to string
         logText = String.fromCharCode(...message.text).replace(/\0/g, '');
      }
      io.emit('status-text', logText);
      // We don't set shouldEmit=true because we emitted a separate event
    }

    if (shouldEmit) {
      io.emit('telemetry', data);
    }
  }
});

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
