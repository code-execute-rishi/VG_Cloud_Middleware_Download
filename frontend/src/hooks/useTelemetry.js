/**
 * useTelemetry Hook
 * 
 * This custom React hook manages the real-time telemetry state of the application.
 * It connects to the LiveKit Room via useRoomContext and updates the state based on incoming DataChannel data.
 * 
 * Features:
 * - Real-time state updates for Attitude, GPS, Battery.
 * - Calculates derived metrics: Distance Travelled, Bearing to Home, Flight Time.
 * - Manages a log of system messages.
 * 
 * @returns {Object} { telemetry, logs } - The current state and log history.
 */

import { useState, useEffect, useRef } from 'react';
import { useRoomContext } from '@livekit/components-react';

export const useTelemetry = () => {
  // Main Telemetry State
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
    // Derived Metrics
    home_position: null, // { lat, lon }
    distance_travelled: 0, // meters
    bearing_to_home: 0, // degrees
    flight_time: "00:00",
  });

  const [logs, setLogs] = useState([]);

  // Refs for calculations to avoid stale state in callbacks
  const stateRef = useRef({
    home: null,
    prevPos: null,
    distance: 0,
    startTime: null,
    armed: false
  });

  const room = useRoomContext();

  /**
   * Calculate Haversine Distance between two coordinates
   */
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

  /**
   * Calculate Bearing from Point A to Point B
   */
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

  // Effect: Flight Timer Logic
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

  // Effect: LiveKit Data Handling
  useEffect(() => {
    if (!room) return;

    const handleDataReceived = (payload, participant, kind, topic) => {
      const strData = new TextDecoder().decode(payload);
      try {
        const msg = JSON.parse(strData);

        // TASK 3 FIX 2: Explicit Type Checking
        if (msg.type === 'telemetry') {
          const data = msg.payload;

          setTelemetry(prev => {
            const newState = { ...prev, connected: true };

            // Process Flight Mode & Armed Status
            if (data.mode) {
              newState.flight_mode = data.mode;
            }
            if (typeof data.armed === 'boolean') {
              newState.armed = data.armed;
              stateRef.current.armed = data.armed; // Update ref for timer
            }

            // Process GPS Raw
            if (data.gps_raw_int) {
              newState.gps_fix = data.gps_raw_int.fix_type;
              newState.satellites = data.gps_raw_int.satellites_visible;
            }

            // Process Global Position (Extended)
            if (data.global_position_int) {
              const lat = data.global_position_int.lat;
              const lon = data.global_position_int.lon;

              newState.lat = lat;
              newState.lon = lon;
              newState.alt = data.global_position_int.alt; // Relative Alt
              newState.relative_alt = data.global_position_int.alt; // Mapping to same for now

              if (data.global_position_int.hdg) {
                newState.heading = data.global_position_int.hdg;
              }

              // Calculate Ground Speed
              if (data.global_position_int.vx !== undefined && data.global_position_int.vy !== undefined) {
                const vx = data.global_position_int.vx;
                const vy = data.global_position_int.vy;
                // cm/s to m/s
                newState.groundspeed = Math.sqrt(vx * vx + vy * vy) / 100.0;
              }

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
                  stateRef.current.prevPos = { lat, lon };
                } else {
                  const dist = getDistance(stateRef.current.prevPos.lat, stateRef.current.prevPos.lon, lat, lon);
                  if (dist > 0.5 && dist < 100) {
                    stateRef.current.distance += dist;
                  }
                  stateRef.current.prevPos = { lat, lon };
                }

                newState.distance_travelled = stateRef.current.distance;

                // 3. Calculate Bearing to Home
                if (stateRef.current.home) {
                  newState.bearing_to_home = getBearing(stateRef.current.home.lat, stateRef.current.home.lon, lat, lon);
                }
              }
            }

            // Process Attitude
            if (data.attitude) {
              newState.roll = data.attitude.roll;
              newState.pitch = data.attitude.pitch;
              newState.yaw = data.attitude.yaw;
            }

            // Process System Status
            if (data.sys_status) {
              newState.voltage = data.sys_status.voltage;
              newState.battery_remaining = data.sys_status.battery_remaining;
            }

            // Process Flight Mode
            if (data.mode) {
              newState.flight_mode = data.mode;
            }

            return newState;
          });
        } else {
          console.warn("Unknown message type:", msg.type);
        }
      } catch (e) {
        console.error("Failed to parse telemetry", e);
      }
    };

    room.on('dataReceived', handleDataReceived);

    // Connection state handling
    const handleConnected = () => setTelemetry(prev => ({ ...prev, connected: true }));
    const handleDisconnected = () => setTelemetry(prev => ({ ...prev, connected: false }));

    room.on('connected', handleConnected);
    room.on('disconnected', handleDisconnected);

    // Initial check
    if (room.state === 'connected') {
      setTelemetry(prev => ({ ...prev, connected: true }));
    }

    return () => {
      room.off('dataReceived', handleDataReceived);
      room.off('connected', handleConnected);
      room.off('disconnected', handleDisconnected);
    };
  }, [room]);

  const addLog = (text) => {
    setLogs(prev => {
      const newLogs = [`[${new Date().toLocaleTimeString()}] ${text}`, ...prev];
      return newLogs.slice(0, 50); // Keep last 50
    });
  };

  return { telemetry, logs };
};
