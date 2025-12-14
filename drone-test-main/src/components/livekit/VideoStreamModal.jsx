'use client';

import { useState, useEffect } from 'react';
import { Room, RoomEvent } from 'livekit-client';
import { connectToRoom, disconnectFromRoom } from '@/lib/livekit';
import { VideoStream } from './VideoStream';
import { X, Wifi, Cpu, Camera } from 'lucide-react';

export function VideoStreamModal({ droneId, droneName, isOpen, onClose }) {
  const [room, setRoom] = useState(null);
  const [isConnecting, setIsConnecting] = useState(false);
  const [connectionStatus, setConnectionStatus] = useState('Disconnected');
  const [sysStatus, setSysStatus] = useState(null);

  useEffect(() => {
    if (!isOpen || !droneId) return;

    const connect = async () => {
      try {
        setIsConnecting(true);
        setConnectionStatus('Connecting...');


        const roomName = `${droneId}`;

        const connectedRoom = await connectToRoom(roomName);
        setRoom(connectedRoom);
        setConnectionStatus('Connected');
        setIsConnecting(false);

        // Data Listener
        connectedRoom.on(RoomEvent.DataReceived, (payload, participant, kind, topic) => {
          if (topic === 'telemetry') {
            try {
              const str = new TextDecoder().decode(payload);
              const msg = JSON.parse(str);
              if (msg.type === 'SYS_STATUS') {
                setSysStatus(msg.data);
              }
            } catch (e) {
              console.error('Telemetry parse error', e);
            }
          }
        });

        connectedRoom.on('disconnected', () => {
          setConnectionStatus('Disconnected');
          setSysStatus(null);
        });
      } catch (error) {
        console.error('Failed to connect to live stream:', error);
        setConnectionStatus('Connection failed');
        setIsConnecting(false);
      }
    };

    connect();

    return () => {
      if (room) {
        disconnectFromRoom(room);
        setRoom(null);
      }
    };
  }, [isOpen, droneId]);

  useEffect(() => {
    if (!isOpen && room) {
      disconnectFromRoom(room);
      setRoom(null);
    }
  }, [isOpen, room]);

  if (!isOpen) return null;

  return (
    <div className="fixed inset-0 z-50 bg-black/80 flex items-center justify-center p-4">
      <div className="bg-white rounded-lg shadow-xl w-full max-w-6xl aspect-video flex flex-col">
        <div className="flex items-center justify-between p-4 border-b border-gray-200">
          <div className="flex items-center gap-3">
            <div>
              <h3 className="text-lg font-semibold text-gray-900">
                Live Stream - {droneName || droneId}
              </h3>
              <div className="flex items-center gap-2 mt-1">
                <div className={`w-2 h-2 rounded-full ${connectionStatus === 'Connected' ? 'bg-green-500' :
                  connectionStatus === 'Connecting...' ? 'bg-yellow-500 animate-pulse' :
                    'bg-red-500'
                  }`} />
                <span className="text-sm text-gray-600">{connectionStatus}</span>
              </div>

            </div>
          </div>

          <div className="flex items-center gap-4">
            {/* Status Icons */}
            <div className="flex items-center gap-3">
              {sysStatus && sysStatus.fc_connected === false && (
                <div className="flex items-center gap-1.5 px-2 py-1 bg-red-100 text-red-700 rounded-md border border-red-200" title="Flight Controller Disconnected">
                  <Cpu size={16} />
                  <span className="text-xs font-bold">FC</span>
                </div>
              )}
              {sysStatus && sysStatus.cam_connected === false && (
                <div className="flex items-center gap-1.5 px-2 py-1 bg-red-100 text-red-700 rounded-md border border-red-200" title="Camera Disconnected">
                  <Camera size={16} />
                  <span className="text-xs font-bold">CAM</span>
                </div>
              )}
            </div>

            {/* Controls */}
            <div className="flex items-center gap-2 pl-4 border-l border-gray-200">
              <Wifi className={`h-4 w-4 ${connectionStatus === 'Connected' ? 'text-green-500' : 'text-gray-400'}`} />
              <button
                onClick={onClose}
                className="p-2 hover:bg-gray-100 rounded-lg transition"
              >
                <X className="h-5 w-5 text-gray-600" />
              </button>
            </div>
          </div>
        </div>

        <div className="flex-1 p-4 overflow-hidden relative bg-gray-100 rounded-b-lg">
          {room ? (
            <VideoStream room={room} className="w-full h-full object-contain rounded-lg shadow-sm" />
          ) : (
            <div className="w-full h-full flex items-center justify-center bg-gray-900 rounded-lg">
              <div className="text-center text-white">
                <div className="animate-pulse mb-2">●</div>
                <p>{connectionStatus}</p>
              </div>
            </div>
          )}
        </div>
      </div>
    </div>

  );
}

