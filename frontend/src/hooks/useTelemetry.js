import { useState, useEffect, useRef } from 'react';
import { io } from 'socket.io-client';

const socket = io('http:172.25.119.168//:3001');
// ArduCopter Mode Mapping
const FLIGHT_MODES = {
  0: 'STABILIZE',
  1: 'ACRO',
  2: 'ALT_HOLD',
  3: 'AUTO',
  4: 'GUIDED',
  5: 'LOITER',
  6: 'RTL',
  7: 'CIRCLE',
  9: 'LAND',
  11: 'DRIFT',
  13: 'SPORT',
  14: 'FLIP',
  15: 'AUTOTUNE',
  16: 'POSHOLD',
  17: 'BRAKE',
  18: 'THROW',
  19: 'AVOID_ADSB',
  20: 'GUIDED_NOGPS',
  21: 'SMART_RTL',
  22: 'FLOWHOLD',
  23: 'FOLLOW',
  24: 'ZIGZAG',
  25: 'SYSTEMID',
  26: 'AUTOROTATE',
  27: 'AUTO_RTL'
};

export const useTelemetry = () => {
  const [telemetry, setTelemetry] = useState({
    lat: 0,
    lon: 0,
    alt: 0,
    relative_alt: 0,
    heading: 0,
    roll: 0,
    pitch: 0,
    yaw: 0,
    voltage: 0,
    current: 0,
    battery_remaining: 0,
    flight_mode: 'UNKNOWN',
    armed: false,
    gps_fix: 0,
    satellites: 0,
    airspeed: 0,
    groundspeed: 0,
    sensors: 0,
    connected: false,
    // New Metrics
    home_position: null, // { lat, lon }
    distance_travelled: 0, // meters
    bearing_to_home: 0, // degrees
    flight_time: "00:00",
  });

  const [logs, setLogs] = useState([]);
  
  // Refs for calculations to avoid stale state in socket callbacks
  const stateRef = useRef({
    home: null,
    prevPos: null,
    distance: 0,
    startTime: null,
    armed: false
  });

  // Helper: Haversine Distance (meters)
  const getDistance = (lat1, lon1, lat2, lon2) => {
    const R = 6371e3; // Earth radius in meters
    const φ1 = lat1 * Math.PI / 180;
    const φ2 = lat2 * Math.PI / 180;
    const Δφ = (lat2 - lat1) * Math.PI / 180;
    const Δλ = (lon2 - lon1) * Math.PI / 180;

    const a = Math.sin(Δφ / 2) * Math.sin(Δφ / 2) +
              Math.cos(φ1) * Math.cos(φ2) *
              Math.sin(Δλ / 2) * Math.sin(Δλ / 2);
    const c = 2 * Math.atan2(Math.sqrt(a), Math.sqrt(1 - a));
    return R * c;
  };

  // Helper: Bearing (degrees)
  const getBearing = (lat1, lon1, lat2, lon2) => {
    const φ1 = lat1 * Math.PI / 180;
    const φ2 = lat2 * Math.PI / 180;
    const Δλ = (lon2 - lon1) * Math.PI / 180;

    const y = Math.sin(Δλ) * Math.cos(φ2);
    const x = Math.cos(φ1) * Math.sin(φ2) -
              Math.sin(φ1) * Math.cos(φ2) * Math.cos(Δλ);
    const θ = Math.atan2(y, x);
    return (θ * 180 / Math.PI + 360) % 360;
  };

  // Flight Timer Effect
  useEffect(() => {
    const timer = setInterval(() => {
      if (stateRef.current.armed) {
        if (!stateRef.current.startTime) {
          stateRef.current.startTime = Date.now();
        }
        const diff = Math.floor((Date.now() - stateRef.current.startTime) / 1000);
        const mins = Math.floor(diff / 60).toString().padStart(2, '0');
        const secs = (diff % 60).toString().padStart(2, '0');
        
        setTelemetry(prev => ({
          ...prev,
          flight_time: `${mins}:${secs}`
        }));
      } else {
        stateRef.current.startTime = null;
      }
    }, 1000);
    return () => clearInterval(timer);
  }, []);

  useEffect(() => {
    const socket = io(SOCKET_URL);

    socket.on('connect', () => {
      console.log('Connected to backend');
      setTelemetry(prev => ({ ...prev, connected: true }));
      addLog('System connected to GCS Backend');
    });

    socket.on('disconnect', () => {
      console.log('Disconnected from backend');
      setTelemetry(prev => ({ ...prev, connected: false }));
      addLog('System disconnected from GCS Backend');
    });

    socket.on('telemetry', (data) => {
      setTelemetry(prev => {
        const newState = { ...prev };
        
        if (data.heartbeat) {
          newState.flight_mode = FLIGHT_MODES[data.heartbeat.custom_mode] || `MODE ${data.heartbeat.custom_mode}`;
          // Base mode bitmask: 128 = Safety Armed
          const isArmed = (data.heartbeat.base_mode & 128) === 128;
          newState.armed = isArmed;
          stateRef.current.armed = isArmed;
        }

        if (data.global_position_int) {
          const lat = data.global_position_int.lat;
          const lon = data.global_position_int.lon;
          
          newState.lat = lat;
          newState.lon = lon;
          newState.alt = data.global_position_int.alt;
          newState.relative_alt = data.global_position_int.relative_alt;
          newState.heading = data.global_position_int.heading;

          // Only process valid coordinates
          if (lat !== 0 && lon !== 0) {
            // 1. Set Home Position (First valid fix)
            if (!stateRef.current.home) {
              stateRef.current.home = { lat, lon };
              newState.home_position = { lat, lon };
              addLog(`Home Position Set: ${lat.toFixed(6)}, ${lon.toFixed(6)}`);
            }

            // 2. Calculate Distance Travelled
            if (!stateRef.current.prevPos) {
              // First valid point, just initialize
              stateRef.current.prevPos = { lat, lon };
            } else {
              // Calculate distance from previous point
              const dist = getDistance(stateRef.current.prevPos.lat, stateRef.current.prevPos.lon, lat, lon);
              
              // Filter noise and jumps: only add if moved > 0.5m AND < 100m (plausible)
              if (dist > 0.5 && dist < 100) {
                stateRef.current.distance += dist;
              }
              
              // Update previous position
              stateRef.current.prevPos = { lat, lon };
            }
            
            newState.distance_travelled = stateRef.current.distance;

            // 3. Calculate Bearing to Home
            if (stateRef.current.home) {
              newState.bearing_to_home = getBearing(stateRef.current.home.lat, stateRef.current.home.lon, lat, lon);
            }
          }
        }

        if (data.attitude) {
          newState.roll = data.attitude.roll;
          newState.pitch = data.attitude.pitch;
          newState.yaw = data.attitude.yaw;
        }

        if (data.sys_status) {
          newState.voltage = data.sys_status.voltage;
          newState.battery_remaining = data.sys_status.battery_remaining;
          newState.current = data.sys_status.current;
          newState.sensors = data.sys_status.sensors;
        }

        if (data.gps_raw_int) {
          newState.gps_fix = data.gps_raw_int.fix_type;
          newState.satellites = data.gps_raw_int.satellites_visible;
        }

        if (data.vfr_hud) {
          newState.airspeed = data.vfr_hud.airspeed;
          newState.groundspeed = data.vfr_hud.groundspeed;
        }

        return newState;
      });
    });

    socket.on('status-text', (text) => {
      addLog(text);
    });

    return () => {
      socket.disconnect();
    };
  }, []);

  const addLog = (text) => {
    setLogs(prev => {
      const newLogs = [`[${new Date().toLocaleTimeString()}] ${text}`, ...prev];
      return newLogs.slice(0, 50); // Keep last 50
    });
  };

  return { telemetry, logs };
};
