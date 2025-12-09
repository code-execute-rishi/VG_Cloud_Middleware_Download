"use client"

import { LiveKitRoom, useLocalParticipant, useTracks } from '@livekit/components-react';
import '@livekit/components-styles';
import { useState, useEffect, useRef } from 'react';
import { Track } from 'livekit-client';

export default function MobileStream() {
  const [token, setToken] = useState('');
  const [url, setUrl] = useState('');
  const [isConnected, setIsConnected] = useState(false);

  const handleConnect = () => {
    if (token && url) {
      setIsConnected(true);
    }
  };

  if (!isConnected) {
    return (
      <div style={{ padding: '20px', maxWidth: '400px', margin: '0 auto' }}>
        <h2>📹 Mobile Camera Publisher</h2>
        
        <div style={{ marginBottom: '15px' }}>
          <label>LiveKit URL:</label>
          <input
            type="text"
            placeholder="wss://your-livekit.com"
            value={url}
            onChange={(e) => setUrl(e.target.value)}
            style={{ width: '100%', padding: '8px', marginTop: '5px' }}
          />
        </div>

        <div style={{ marginBottom: '15px' }}>
          <label>Token:</label>
          <textarea
            placeholder="eyJhbGc..."
            value={token}
            onChange={(e) => setToken(e.target.value)}
            rows={4}
            style={{ width: '100%', padding: '8px', marginTop: '5px' }}
          />
        </div>

        <button 
          onClick={handleConnect}
          style={{ 
            width: '100%', 
            padding: '12px', 
            background: '#007bff', 
            color: 'white', 
            border: 'none',
            borderRadius: '4px',
            fontSize: '16px'
          }}
        >
          Start Streaming
        </button>
      </div>
    );
  }

  return (
    <LiveKitRoom
      token={token}
      serverUrl={url}
      connect={true}
      video={true}
      audio={false}
      options={{
        adaptiveStream: true,
        dynacast: true,
      }}
    >
      <PublisherView />
    </LiveKitRoom>
  );
}

function PublisherView() {
  const { localParticipant, room } = useLocalParticipant();
  const videoRef = useRef(null);
  const [messageCount, setMessageCount] = useState(0);
  
  const cameraTracks = useTracks([Track.Source.Camera], {
    participant: localParticipant,
  });

  console.log('🔍 PublisherView Debug Info:');
  console.log('🔍 localParticipant:', localParticipant?.identity);
  console.log('🔍 cameraTracks found:', cameraTracks.length);
  
  // Check for video track publications
  useEffect(() => {
    if (localParticipant) {
      const videoPubs = Array.from(localParticipant.videoTrackPublications.values());
      console.log('📹 Video Publications:', videoPubs.length);
      videoPubs.forEach((pub, i) => {
        console.log(`📹 Publication ${i}:`, {
          trackSid: pub.trackSid,
          isMuted: pub.isMuted,
          isEnabled: pub.isEnabled,
          kind: pub.kind,
          source: pub.source,
        });
      });
    }
  }, [localParticipant]);

  // Attach the camera track to video element
  useEffect(() => {
    console.log('🎥 useEffect triggered');
    
    if (videoRef.current && cameraTracks.length > 0) {
      const track = cameraTracks[0];
      console.log('🎥 Track found:', track);
      
      if (track.publication?.track?.mediaStreamTrack) {
        console.log('🎥 Attaching mediaStreamTrack to video element');
        const mediaStreamTrack = track.publication.track.mediaStreamTrack;
        
        const stream = new MediaStream([mediaStreamTrack]);
        videoRef.current.srcObject = stream;
        
        videoRef.current.play().catch(error => {
          console.error('❌ Video play error:', error);
        });
      } else {
        console.warn('⚠️ No mediaStreamTrack available');
      }
    }
  }, [cameraTracks]);

  // DATA SENDER - Sends telemetry in exact format frontend expects
  useEffect(() => {
    if (!localParticipant) {
      console.log('⏳ Waiting for localParticipant...');
      return;
    }
    
    console.log('📤 Starting telemetry data sender...');
    
    let heading = 0;
    let altitude = 10.0;
    
    const interval = setInterval(() => {
      // Simulate changing values
      heading = (heading + 5) % 360;
      altitude = 10 + Math.sin(Date.now() / 2000) * 5;
      
      // ATTITUDE message
      const attitudeMsg = {
        type: 'ATTITUDE',
        data: {
          roll: Math.sin(Date.now() / 1000) * 0.2,  // radians
          pitch: Math.cos(Date.now() / 1000) * 0.1, // radians
          yaw: heading * (Math.PI / 180)            // radians
        }
      };
      
      // GLOBAL_POSITION_INT message
      const gpsMsg = {
        type: 'GLOBAL_POSITION_INT',
        data: {
          lat: -353632620,      // lat * 1e7 (int32)
          lon: 1491652370,      // lon * 1e7 (int32)
          alt: altitude * 1000, // mm (int32)
          relative_alt: altitude * 1000, // mm (int32)
          hdg: heading * 100,   // centidegrees (uint16)
          vx: 100,              // cm/s (int16)
          vy: 50,               // cm/s (int16)
          vz: -10               // cm/s (int16)
        }
      };
      
      // SYS_STATUS message
      const sysMsg = {
        type: 'SYS_STATUS',
        data: {
          voltage_battery: 12600 + Math.random() * 100, // millivolts
          battery_remaining: 85 - Math.floor(Date.now() / 10000) % 10
        }
      };
      
      // VFR_HUD message
      const vfrMsg = {
        type: 'VFR_HUD',
        data: {
          airspeed: 12.5 + Math.random() * 2,
          groundspeed: 11.8 + Math.random() * 2,
          heading: heading,
          throttle: 65,
          alt: altitude,
          climb: Math.sin(Date.now() / 3000) * 0.5
        }
      };
      
      // GPS_RAW_INT message
      const gpsRawMsg = {
        type: 'GPS_RAW_INT',
        data: {
          fix_type: 3,          // 3D Fix
          satellites_visible: 12
        }
      };
      
      // NAV_CONTROLLER_OUTPUT message
      const navMsg = {
        type: 'NAV_CONTROLLER_OUTPUT',
        data: {
          wp_dist: 150 - (Date.now() / 1000) % 150
        }
      };
      
      // MISSION_CURRENT message
      const missionMsg = {
        type: 'MISSION_CURRENT',
        data: {
          seq: 3
        }
      };
      
      // RC_CHANNELS message
      const rcMsg = {
        type: 'RC_CHANNELS',
        data: {
          rssi: 200 + Math.floor(Math.random() * 50) // 0-255
        }
      };
      
      // Send all messages
      const messages = [
        attitudeMsg, 
        gpsMsg, 
        sysMsg, 
        vfrMsg, 
        gpsRawMsg,
        navMsg,
        missionMsg,
        rcMsg
      ];
      
      messages.forEach(msg => {
        const encoded = new TextEncoder().encode(JSON.stringify(msg));
        try {
          localParticipant.publishData(encoded, { 
            reliable: true, 
            topic: 'telemetry' 
          });
        } catch (error) {
          console.error('❌ Error sending data:', error);
        }
      });
      
      setMessageCount(prev => prev + messages.length);
      
      if (messageCount % 40 === 0) { // Log every 5 intervals
        console.log(`📤 Sent ${messages.length} messages, total: ${messageCount}`);
      }
      
    }, 5000); // Send every 500ms
    
    return () => {
      console.log('🛑 Stopping data sender');
      clearInterval(interval);
    };
  }, [localParticipant, messageCount]);

  return (
    <div style={{ width: '100vw', height: '100vh', background: '#000' }}>
      <div style={{ padding: '10px', background: 'rgba(0,0,0,0.7)', color: 'white' }}>
        <h3>📹 Streaming Active</h3>
        <p>Status: {localParticipant ? '✅ Connected' : '⏳ Connecting...'}</p>
        <p>Participant: {localParticipant?.identity || 'None'}</p>
        <p>Camera Tracks: {cameraTracks.length}</p>
        <p style={{ color: '#00ff00' }}>📤 Messages Sent: {messageCount}</p>
      </div>

      <div style={{ 
        width: '100%', 
        height: 'calc(100% - 140px)',
        display: 'flex',
        alignItems: 'center',
        justifyContent: 'center',
        backgroundColor: '#000'
      }}>
        {cameraTracks.length > 0 ? (
          <video
            ref={videoRef}
            autoPlay
            playsInline
            muted
            style={{
              width: '100%',
              height: '100%',
              objectFit: 'contain',
              transform: 'scaleX(-1)',
            }}
            onLoadedData={() => console.log('✅ Video data loaded')}
            onCanPlay={() => console.log('✅ Video can play')}
            onPlaying={() => console.log('▶️ Video is playing')}
            onError={(e) => console.error('❌ Video error:', e)}
          />
        ) : (
          <div style={{ color: 'white', textAlign: 'center' }}>
            <p>Waiting for camera feed...</p>
            <p>Camera tracks detected: {cameraTracks.length}</p>
            {localParticipant && (
              <p>Camera enabled: {localParticipant.isCameraEnabled ? 'Yes' : 'No'}</p>
            )}
          </div>
        )}
      </div>
    </div>
  );
}