'use client';

import { useState, useEffect } from 'react';
import { Room } from 'livekit-client';
import { connectToRoom, disconnectFromRoom } from '@/lib/livekit';
import { SSHTerminal } from './SSHTerminal';
import { X, Wifi } from 'lucide-react';

export function SSHModal({ droneId, droneName, isOpen, onClose }) {
  const [room, setRoom] = useState(null);
  const [isConnecting, setIsConnecting] = useState(false);
  const [connectionStatus, setConnectionStatus] = useState('Disconnected');

  useEffect(() => {
    if (!isOpen || !droneId) return;

    const connect = async () => {
      try {
        setIsConnecting(true);
        setConnectionStatus('Connecting...');
        
        const participantName = `ssh-${Date.now()}`;
        const roomName = `drone-${droneId}`;
        
        const connectedRoom = await connectToRoom(roomName, participantName, participantName);
        setRoom(connectedRoom);
        setConnectionStatus('Connected');
        setIsConnecting(false);

        connectedRoom.on('disconnected', () => {
          setConnectionStatus('Disconnected');
        });
      } catch (error) {
        console.error('Failed to connect to SSH:', error);
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
      <div className="bg-white rounded-lg shadow-xl w-full max-w-6xl h-[80vh] flex flex-col">
        <div className="flex items-center justify-between p-4 border-b border-gray-200">
          <div className="flex items-center gap-3">
            <div>
              <h3 className="text-lg font-semibold text-gray-900">
                SSH Terminal - {droneName || droneId}
              </h3>
              <div className="flex items-center gap-2 mt-1">
                <div className={`w-2 h-2 rounded-full ${
                  connectionStatus === 'Connected' ? 'bg-green-500' : 
                  connectionStatus === 'Connecting...' ? 'bg-yellow-500 animate-pulse' : 
                  'bg-red-500'
                }`} />
                <span className="text-sm text-gray-600">{connectionStatus}</span>
              </div>
            </div>
          </div>
          <div className="flex items-center gap-2">
            <Wifi className="h-4 w-4 text-gray-400" />
            <button
              onClick={onClose}
              className="p-2 hover:bg-gray-100 rounded-lg transition"
            >
              <X className="h-5 w-5 text-gray-600" />
            </button>
          </div>
        </div>
        <div className="flex-1 p-4 overflow-hidden">
          {room ? (
            <SSHTerminal room={room} className="h-full" />
          ) : (
            <div className="w-full h-full flex items-center justify-center bg-[#1e1e1e] rounded-lg">
              <div className="text-center text-gray-400">
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

