'use client';

import { useState, useEffect, useRef } from 'react';
import dynamic from 'next/dynamic';
import { Room, RoomEvent } from 'livekit-client';
import { connectToRoom, disconnectFromRoom } from '@/lib/livekit';
import {
  HeadingIndicator,
  AttitudeIndicator
} from 'react-flight-indicators';
import { ZoomIn, ZoomOut, Link, Signal, Satellite, RotateCw, RotateCcw, ChevronUp, ChevronDown, RefreshCw } from 'lucide-react';
import { VideoStream } from '@/components/livekit/VideoStream';

const DroneMap = dynamic(() => import('../../components/DroneMap'), {
  ssr: false,
});

export default function Home() {
  const mapRef = useRef(null);
  const mapWidgetRef = useRef(null);
  const roomRef = useRef(null);
  const [connectionStatus, setConnectionStatus] = useState('Connecting...');

  // MAVLink message states - one state object per message type
  const [attitude, setAttitude] = useState(null);
  const [globalPositionInt, setGlobalPositionInt] = useState(null);
  const [vfrHud, setVfrHud] = useState(null);
  const [sysStatus, setSysStatus] = useState(null);
  const [powerStatus, setPowerStatus] = useState(null);
  const [memInfo, setMemInfo] = useState(null);
  const [navControllerOutput, setNavControllerOutput] = useState(null);
  const [missionCurrent, setMissionCurrent] = useState(null);
  const [servoOutputRaw, setServoOutputRaw] = useState(null);
  const [rcChannels, setRcChannels] = useState(null);
  const [rawImu, setRawImu] = useState(null);

  // UI state
  const [isModalOpen, setIsModalOpen] = useState(false);
  const [selectedDrone, setSelectedDrone] = useState(null);
  const [satelliteCount, setSatelliteCount] = useState(0);
  const [distToHome, setDistToHome] = useState(0.0);
  const [flightDist, setFlightDist] = useState(0.0);
  const [isVideoFullscreen, setIsVideoFullscreen] = useState(false);

  // Actual devices from backend
  const [devices, setDevices] = useState([]);
  const [loadingDevices, setLoadingDevices] = useState(true);

  // Fetch actual devices from backend
  useEffect(() => {
    const fetchDevices = async () => {
      try {
        setLoadingDevices(true);
        console.log('📡 Fetching devices from backend...');

        const response = await fetch('/api/devices', {
          method: 'GET',
          headers: {
            'Content-Type': 'application/json',
          },
          credentials: 'include', // Include cookies for Clerk auth
        });

        if (!response.ok) {
          throw new Error(`Failed to fetch devices: ${response.status}`);
        }

        const devicesData = await response.json();
        console.log('✅ Devices fetched:', devicesData);

        setDevices(devicesData);

        // Auto-select first active device if none selected
        if (!selectedDrone && devicesData.length > 0) {
          const firstActiveDevice = devicesData.find(d => d.status === 'active') || devicesData[0];
          setSelectedDrone(firstActiveDevice.id);
          console.log('🎯 Auto-selected device:', firstActiveDevice.id);
        }
      } catch (error) {
        console.error('❌ Failed to fetch devices:', error);
        setDevices([]);
      } finally {
        setLoadingDevices(false);
      }
    };

    fetchDevices();
  }, []);

  // Helper function to process telemetry messages
  const processTelemetryMessage = (message) => {
    if (!message || !message.type || !message.data) {
      console.warn('Invalid message format:', message);
      return;
    }

    const { type, data } = message;

    try {
      if (type === 'ATTITUDE') {
        setAttitude(data);

        if (data.yaw !== undefined && typeof window !== 'undefined' && window.updateDroneHeading) {
          const yawDeg = data.yaw * (180 / Math.PI);
          const normalizedHeading = ((yawDeg % 360) + 360) % 360;
          window.updateDroneHeading(normalizedHeading);
        }
      } else if (type === 'GLOBAL_POSITION_INT') {
        setGlobalPositionInt(data);

        if (data.lat !== undefined && data.lon !== undefined) {
          const lat = data.lat / 1e7;
          const lon = data.lon / 1e7;
          let heading = 0;

          if (data.hdg !== undefined) {
            heading = data.hdg / 100;
          }

          if (typeof window !== 'undefined' && window.updateDroneData) {
            window.updateDroneData(lat, lon, heading);
          } else if (typeof window !== 'undefined' && window.updateDronePosition) {
            window.updateDronePosition(lat, lon);
          }
        }
      } else if (type === 'VFR_HUD') {
        setVfrHud(data);

        if (data.heading !== undefined && typeof window !== 'undefined' && window.updateDroneHeading) {
          window.updateDroneHeading(data.heading);
        }
      } else if (type === 'SYS_STATUS') {
        setSysStatus(data);
      } else if (type === 'POWER_STATUS') {
        setPowerStatus(data);
      } else if (type === 'MEMINFO') {
        setMemInfo(data);
      } else if (type === 'NAV_CONTROLLER_OUTPUT') {
        setNavControllerOutput(data);
      } else if (type === 'MISSION_CURRENT') {
        setMissionCurrent(data);
      } else if (type === 'SERVO_OUTPUT_RAW') {
        setServoOutputRaw(data);
      } else if (type === 'RC_CHANNELS') {
        setRcChannels(data);
      } else if (type === 'RAW_IMU') {
        setRawImu(data);
      } else {
        console.log('Received unhandled message type:', type);
      }
    } catch (error) {
      console.error('Error processing telemetry message:', error, message);
    }
  };

  // Helper function to setup data listener for a room
  const setupRoomDataListener = (room) => {
    room.on(RoomEvent.DataReceived, (payload, participant, kind, topic) => {
      console.log('📦 Data received - Topic:', topic, 'Kind:', kind);

      if (topic === 'telemetry') {
        try {
          const messageStr = new TextDecoder().decode(payload);
          const message = JSON.parse(messageStr);
          console.log('📨 Telemetry message:', message);
          processTelemetryMessage(message);
        } catch (error) {
          console.error('Error processing telemetry message:', error);
        }
      }
    });
  };

  // LiveKit connection for telemetry data
  useEffect(() => {
    // Wait for devices to load and a drone to be selected
    if (loadingDevices) {
      console.log('⏳ Waiting for devices to load...');
      setConnectionStatus('Loading devices...');
      return;
    }

    if (!selectedDrone) {
      console.log('⏳ No drone selected');
      setConnectionStatus('No drone selected');
      return;
    }

    const deviceId = selectedDrone; // This is the actual UUID from backend

    console.log('🔌 Connecting to room for device UUID:', deviceId);

    const connectToLiveKit = async () => {
      try {
        setConnectionStatus('Connecting...');

        const room = await connectToRoom(deviceId);
        roomRef.current = room;
        setConnectionStatus('Connected');
        console.log('✅ Connected to LiveKit room for device:', deviceId);

        // Handle disconnection
        room.on(RoomEvent.Disconnected, () => {
          console.log('❌ Disconnected from LiveKit room:', deviceId);
          setConnectionStatus('Disconnected');
        });

        // Listen for telemetry data through LiveKit data channels
        setupRoomDataListener(room);

      } catch (error) {
        console.error('❌ Failed to connect to LiveKit:', error);
        setConnectionStatus('Connection failed');
      }
    };

    // Disconnect from previous room if exists
    if (roomRef.current) {
      console.log('🔄 Disconnecting from previous room before switching...');
      disconnectFromRoom(roomRef.current);
      roomRef.current = null;
    }

    connectToLiveKit();

    // Cleanup on unmount or when selectedDrone changes
    return () => {
      console.log('🧹 Cleaning up LiveKit connection for:', deviceId);
      if (roomRef.current) {
        disconnectFromRoom(roomRef.current);
        roomRef.current = null;
      }
    };
  }, [selectedDrone, loadingDevices]);

  // Calculate distance to home from relative altitude
  useEffect(() => {
    if (globalPositionInt?.relative_alt !== undefined) {
      // relative_alt is in mm, convert to meters
      const relativeAltM = Math.abs(globalPositionInt.relative_alt / 1000);
      setDistToHome(relativeAltM);
    }
  }, [globalPositionInt]);

  // Helper functions to extract and convert values with defaults
  const getHeading = () => {
    // Prefer VFR_HUD heading, then GLOBAL_POSITION_INT, then ATTITUDE yaw
    if (vfrHud?.heading !== undefined) {
      return vfrHud.heading;

    }
    if (globalPositionInt?.hdg !== undefined) {
      return globalPositionInt.hdg / 100; // centidegrees to degrees
    }
    if (attitude?.yaw !== undefined) {
      const yawDeg = attitude.yaw * (180 / Math.PI); // radians to degrees
      return ((yawDeg % 360) + 360) % 360; // normalize to 0-360
    }
    return 0;
  };

  const getRoll = () => {
    if (attitude?.roll !== undefined) {
      return attitude.roll * (180 / Math.PI); // radians to degrees
    }
    return 0;
  };

  const getPitch = () => {
    if (attitude?.pitch !== undefined) {
      return attitude.pitch * (180 / Math.PI); // radians to degrees
    }
    return 0;
  };

  const getAltitude = () => {
    // Prefer VFR_HUD alt, then GLOBAL_POSITION_INT
    if (vfrHud?.alt !== undefined) {
      return vfrHud.alt;
    }
    if (globalPositionInt?.alt !== undefined) {
      return globalPositionInt.alt / 1000; // mm to meters
    }
    return 0;
  };

  const getAirSpeed = () => {
    return vfrHud?.airspeed ?? 0.0;
  };

  const getGndSpeed = () => {
    return vfrHud?.groundspeed ?? 0.0;
  };

  const getThrottle = () => {
    return vfrHud?.throttle ?? 0;
  };

  const getVoltage = () => {
    if (sysStatus?.voltage_battery !== undefined) {
      return sysStatus.voltage_battery / 1000; // millivolts to volts
    }
    return 0;
  };

  const getSignalStrength = () => {
    if (rcChannels?.rssi !== undefined) {
      // RSSI is 0-255, convert to percentage (0-100)
      return Math.min(100, Math.max(0, (rcChannels.rssi / 255) * 100));
    }
    return 100; // default
  };

  const getWpDist = () => {
    return navControllerOutput?.wp_dist ?? 0;
  };

  // Helper for distance calculation (Haversine)
  const calculateDistance = (lat1, lon1, lat2, lon2) => {
    const R = 6371e3; // metres
    const φ1 = lat1 * Math.PI / 180;
    const φ2 = lat2 * Math.PI / 180;
    const Δφ = (lat2 - lat1) * Math.PI / 180;
    const Δλ = (lon2 - lon1) * Math.PI / 180;

    const a = Math.sin(Δφ / 2) * Math.sin(Δφ / 2) +
      Math.cos(φ1) * Math.cos(φ2) *
      Math.sin(Δλ / 2) * Math.sin(Δλ / 2);
    const c = 2 * Math.atan2(Math.sqrt(a), Math.sqrt(1 - a));

    return R * c;
  }

  // Ref to store last position for distance calc
  const lastPosRef = useRef(null);

  useEffect(() => {
    if (globalPositionInt?.lat && globalPositionInt?.lon) {
      const currentLat = globalPositionInt.lat / 1e7;
      const currentLon = globalPositionInt.lon / 1e7;

      if (lastPosRef.current) {
        const dist = calculateDistance(
          lastPosRef.current.lat,
          lastPosRef.current.lon,
          currentLat,
          currentLon
        );
        // Ignore jumps (e.g. init) or tiny noise
        if (dist > 0.5 && dist < 500) {
          setFlightDist(prev => prev + dist);
        }
      }
      lastPosRef.current = { lat: currentLat, lon: currentLon };
    }
  }, [globalPositionInt]);


  const getNextWaypoint = () => {
    if (missionCurrent?.seq !== undefined) {
      return `WP ${missionCurrent.seq}`;
    }
    return '--';
  };

  // Get computed values
  const heading = getHeading();
  const roll = getRoll();
  const pitch = getPitch();
  const altitude = getAltitude();
  const airSpeed = getAirSpeed();
  const gndSpeed = getGndSpeed();
  const throttle = getThrottle();
  const voltage = getVoltage();
  const signalStrength = getSignalStrength();
  const wpDist = getWpDist();
  const nextWaypoint = getNextWaypoint();

  const toggleVideoFullscreen = () => {
    setIsVideoFullscreen(!isVideoFullscreen);
  };

  // Get selected device info for display
  const selectedDevice = devices.find(d => d.id === selectedDrone);

  return (
    <div className="fixed top-0 left-0 h-screen w-screen overflow-hidden z-0">
      {/* Main View - Map (default) */}
      <div className={`absolute top-0 left-0 w-full h-full z-0 transition-all duration-300 ${isVideoFullscreen ? 'opacity-0 pointer-events-none' : 'opacity-100'}`}>
        <DroneMap ref={mapRef} />
      </div>

      {/* Main Video Stream - Fullscreen when toggled */}
      {roomRef.current && (
        <div
          className={`absolute top-0 left-0 w-full h-full z-[10] transition-all duration-300 bg-black ${isVideoFullscreen ? 'opacity-100' : 'opacity-0 pointer-events-none'}`}
          onDoubleClick={toggleVideoFullscreen}
          style={{ cursor: 'pointer' }}
        >
          <VideoStream room={roomRef.current} className="w-full h-full" />
        </div>
      )}

      {/* Small Video Widget - Bottom Left (shown when map is fullscreen) */}
      {roomRef.current && !isVideoFullscreen && (
        <div
          className="absolute bottom-5 left-5 z-[1000] w-[320px] h-[240px] transition-all duration-300"
          onDoubleClick={toggleVideoFullscreen}
          style={{ cursor: 'pointer' }}
        >
          <div className="relative w-full h-full bg-black rounded-lg overflow-hidden border-2 border-gray-700 shadow-2xl">
            <VideoStream room={roomRef.current} className="w-full h-full" />
            <div className="absolute top-2 left-2 bg-black/70 text-white text-xs px-2 py-1 rounded flex items-center gap-2 z-10">
              <div className="w-2 h-2 bg-red-500 rounded-full animate-pulse"></div>
              <span>LIVE</span>
            </div>
            <div className="absolute bottom-2 left-2 bg-black/70 text-white text-xs px-2 py-1 rounded z-10">
              Double-click to expand
            </div>
          </div>
        </div>
      )}

      {/* Small Map Widget - Bottom Left (shown when video is fullscreen) */}
      {isVideoFullscreen && (
        <div
          className="absolute bottom-5 left-5 z-[1000] w-[320px] h-[240px] transition-all duration-300"
          onDoubleClick={toggleVideoFullscreen}
          style={{ cursor: 'pointer' }}
        >
          <div className="relative w-full h-full bg-black rounded-lg overflow-hidden border-2 border-gray-700 shadow-2xl">
            <DroneMap ref={mapWidgetRef} />
            <div className="absolute bottom-2 left-2 bg-black/70 text-white text-xs px-2 py-1 rounded z-10">
              Double-click to expand map
            </div>
          </div>
        </div>
      )}

      {/* Controls */}
      <div className="absolute top-10 left-5 z-[1000] flex flex-col items-start gap-2.5">
        <div className="flex flex-col items-center gap-2">
          <button
            onClick={() => mapRef.current?.zoomIn()}
            className="bg-black/70 hover:bg-black/80 text-white p-2 rounded transition-colors"
            title="Zoom In"
          >
            <ZoomIn size={20} />
          </button>
          <button
            onClick={() => mapRef.current?.zoomOut()}
            className="bg-black/70 hover:bg-black/80 text-white p-2 rounded transition-colors"
            title="Zoom Out"
          >
            <ZoomOut size={20} />
          </button>
          <div className="w-full h-px bg-gray-600 my-1"></div>
          <button
            onClick={() => mapRef.current?.adjustPitch(10)}
            className="bg-black/70 hover:bg-black/80 text-white p-2 rounded transition-colors"
            title="Tilt Up"
          >
            <ChevronUp size={20} />
          </button>
          <button
            onClick={() => mapRef.current?.adjustPitch(-10)}
            className="bg-black/70 hover:bg-black/80 text-white p-2 rounded transition-colors"
            title="Tilt Down"
          >
            <ChevronDown size={20} />
          </button>
          <button
            onClick={() => mapRef.current?.adjustBearing(-15)}
            className="bg-black/70 hover:bg-black/80 text-white p-2 rounded transition-colors"
            title="Rotate Left"
          >
            <RotateCcw size={20} />
          </button>
          <button
            onClick={() => mapRef.current?.adjustBearing(15)}
            className="bg-black/70 hover:bg-black/80 text-white p-2 rounded transition-colors"
            title="Rotate Right"
          >
            <RotateCw size={20} />
          </button>
          <button
            onClick={() => mapRef.current?.resetView()}
            className="bg-black/70 hover:bg-black/80 text-white p-2 rounded transition-colors"
            title="Reset 3D View"
          >
            <RefreshCw size={20} />
          </button>
          <div className="w-full h-px bg-gray-600 my-1"></div>
          <button
            onClick={() => setIsModalOpen(!isModalOpen)}
            className="bg-black/70 hover:bg-black/80 text-white p-2 rounded transition-colors"
            title="Link"
          >
            <Link size={20} />
          </button>
        </div>
      </div>

      {/* Right Side - Flight Data Box */}
      <div className="absolute top-5 right-5 z-[1000]">
        <div className="bg-black/70 p-4 rounded-lg min-w-[320px] border border-gray-700">
          <div className="flex gap-4">
            {/* Left Side - Indicators */}
            <div className="flex flex-col items-center gap-3">
              <div className="flex flex-col items-center gap-1">
                <div className="flex items-center gap-1 mb-1">
                  <div className={`w-2 h-2 rounded-full ${connectionStatus === 'Connected' ? 'bg-green-500' :
                    connectionStatus === 'Connecting...' ? 'bg-yellow-500 animate-pulse' :
                      'bg-red-500'
                    }`}></div>
                  <span className="text-xs text-gray-400">{connectionStatus}</span>
                </div>
              </div>
              <div className="flex flex-col items-center gap-1">
                <div className="flex items-end gap-1 h-8">
                  <div className={`w-2 ${signalStrength > 20 ? 'bg-green-500' : 'bg-gray-500'} h-2`}></div>
                  <div className={`w-2 ${signalStrength > 40 ? 'bg-green-500' : 'bg-gray-500'} h-3`}></div>
                  <div className={`w-2 ${signalStrength > 60 ? 'bg-green-500' : 'bg-gray-500'} h-4`}></div>
                  <div className={`w-2 ${signalStrength > 80 ? 'bg-green-500' : 'bg-gray-500'} h-5`}></div>
                  <div className={`w-2 ${signalStrength > 90 ? 'bg-green-500' : 'bg-gray-500'} h-6`}></div>
                </div>
                <div className="text-green-500 text-xs font-bold">{Math.round(signalStrength)}%</div>
              </div>

              <div className="flex flex-col items-center gap-1">
                <Satellite className="text-green-500" size={24} />
                <div className="text-green-500 text-sm font-bold">{satelliteCount > 0 ? satelliteCount : '--'}</div>
              </div>
            </div>

            {/* Right Side - Data List */}
            <div className="flex-1 flex flex-col gap-1 text-xs text-white">
              <div className="flex justify-between">
                <span>Voltage:</span>
                <span className="font-mono">{sysStatus?.voltage_battery !== undefined ? voltage.toFixed(2) + ' v' : '--'}</span>
              </div>
              <div className="flex justify-between">
                <span>Throttle:</span>
                <span className="font-mono">{vfrHud ? Math.round(throttle) : '--'}</span>
              </div>
              <div className="flex justify-between">
                <span>Air Speed:</span>
                <span className="font-mono">{vfrHud ? airSpeed.toFixed(1) + ' m/s' : '--'}</span>
              </div>
              <div className="flex justify-between">
                <span>Gnd Speed:</span>
                <span className="font-mono">{vfrHud ? gndSpeed.toFixed(1) + ' m/s' : '--'}</span>
              </div>
              <div className="flex justify-between">
                <span>Altitude:</span>
                <span className="font-mono">{(vfrHud || globalPositionInt) ? altitude.toFixed(1) + ' m' : '--'}</span>
              </div>
              <div className="flex justify-between">
                <span>Dist To Home:</span>
                <span className="font-mono">{globalPositionInt?.relative_alt !== undefined ? distToHome.toFixed(1) + ' m' : '--'}</span>
              </div>
              <div className="flex justify-between">
                <span>Flight Dist:</span>
                <span className="font-mono">{flightDist > 0 ? flightDist.toFixed(1) + ' m' : '--'}</span>
              </div>
              <div className="flex justify-between">
                <span>Next Waypoint:</span>
                <span className="font-mono">{nextWaypoint}</span>
              </div>
              <div className="flex justify-between">
                <span>Next WP Dist:</span>
                <span className="font-mono">{navControllerOutput?.wp_dist !== undefined ? wpDist.toFixed(1) + ' m' : '--'}</span>
              </div>
            </div>
          </div>
        </div>
      </div>

      {/* Heading Indicator */}
      <div className="absolute bottom-60 right-5 z-[1000] flex flex-col items-start gap-2.5">
        <div className="bg-black/70 p-1 rounded-lg flex flex-col items-center min-w-[120px]">
          <div className=" origin-top">
            <HeadingIndicator heading={heading} showBox={false} size={200} />
          </div>
        </div>
      </div>

      {/* Attitude */}
      <div className="absolute bottom-5 right-5 z-[1000] flex flex-col items-start">
        <div className="bg-black/70 p-1 rounded-lg flex flex-col items-center min-w-[120px]">
          <div className=" origin-top">
            <AttitudeIndicator roll={roll} pitch={pitch} showBox={false} size={200} />
          </div>
        </div>
      </div>

      {/* Modal */}
      {isModalOpen && (
        <div className='absolute top-10 left-20 min-w-[300px] bg-black/60 p-4 rounded-lg z-[1500] text-white border border-gray-700'>
          <div className="mb-3">
            <h3 className="text-lg font-bold uppercase mb-2">Select Drone</h3>
          </div>

          {loadingDevices ? (
            <div className="text-center py-4">
              <div className="animate-spin w-6 h-6 border-2 border-white border-t-transparent rounded-full mx-auto"></div>
              <p className="text-sm text-gray-400 mt-2">Loading devices...</p>
            </div>
          ) : devices.length === 0 ? (
            <div className="text-center py-4">
              <p className="text-sm text-gray-400">No devices found</p>
            </div>
          ) : (
            <div className="flex flex-col gap-2">
              {devices.map((device) => (
                <button
                  key={device.id}
                  onClick={() => {
                    setSelectedDrone(device.id);
                    setIsModalOpen(false);
                  }}
                  className={`p-2 rounded text-center transition-colors ${selectedDrone === device.id
                    ? 'bg-green-500/30 border border-green-500'
                    : 'bg-black/40 hover:bg-black/60 border border-transparent'
                    }`}
                >
                  <div>
                    <div className="font-bold text-sm">{device.name || 'Unnamed Device'}</div>
                    <div className="text-xs text-gray-400">{device.status || 'unknown'}</div>
                  </div>
                </button>
              ))}
            </div>
          )}
        </div>
      )}
    </div>
  );
}