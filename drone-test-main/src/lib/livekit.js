import axios from 'axios';
import { Room, RoomEvent } from 'livekit-client';

export async function getLiveKitToken(deviceId) {
  try {
    console.log('🔐 Requesting LiveKit token for deviceId:', deviceId);
    
    const response = await axios.post('/api/livekit/token', {
      deviceId: deviceId,
    });

    console.log('✅ Token response received:', response.data);
    return response.data;
  } catch (error) {
    console.error('❌ Error fetching LiveKit token:', error);
    const errorMessage = error.response?.data?.error || error.response?.data?.details || error.message || 'Failed to get LiveKit token';
    throw new Error(errorMessage);
  }
}

export async function connectToRoom(deviceId) {
  try {
    console.log('🔐 Requesting LiveKit token for device:', deviceId);
    
    const { token, url, roomName } = await getLiveKitToken(deviceId);
    
    console.log('✅ Token received');
    console.log('✅ Room name:', roomName);
    console.log('✅ Connecting to:', url);
    
    const room = new Room({
      adaptiveStream: true,
      dynacast: true,
      videoCaptureDefaults: {
        resolution: {
          width: 1280,
          height: 720,
        },
      },
    });

    room.on(RoomEvent.Connected, () => {
      console.log('✅ Room connected successfully:', roomName);
    });

    // room.on(RoomEvent.ParticipantConnected, (participant) => {
    //   console.log('✅ Participant connected:', participant?.identity || 'unknown');
    // });

    // room.on(RoomEvent.ParticipantDisconnected, (participant) => {
    //   console.log('⚠️ Participant disconnected:', participant?.identity || 'unknown');
    // });

    room.on(RoomEvent.Disconnected, (reason) => {
      console.log('⚠️ Room disconnected:', reason);
    });

    room.on(RoomEvent.Reconnecting, () => {
      console.log('🔄 Reconnecting to room...');
    });

    room.on(RoomEvent.Reconnected, () => {
      console.log('✅ Reconnected to room');
    });

    room.on(RoomEvent.ConnectionQualityChanged, (quality) => {
      console.log('📊 Connection quality changed:', quality);
    });

    await room.connect(url, token);
    console.log('✅ Successfully connected to LiveKit room:', roomName);

    return room;
  } catch (error) {
    console.error('❌ Failed to connect to room:', error);
    throw error;
  }
}

export function disconnectFromRoom(room) {
  if (room) {
    console.log('🔌 Disconnecting from room...');
    room.disconnect();
  }
}