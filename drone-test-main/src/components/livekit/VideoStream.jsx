'use client';

import { useEffect, useRef, useState } from 'react';
import { Room, RoomEvent } from 'livekit-client';

export function VideoStream({ room, className = '' }) {
  const videoRef = useRef(null);
  const [isStreaming, setIsStreaming] = useState(false);

  useEffect(() => {
    if (!room) return;

    const checkActiveStreams = () => {
      let isActive = false;
      room.remoteParticipants.forEach((participant) => {
        participant.videoTrackPublications.forEach((publication) => {
          if (publication.track && publication.isSubscribed) {
            if (videoRef.current && publication.track.kind === 'video') {
              publication.track.attach(videoRef.current);
              isActive = true;
            }
          }
        });
      });
      setIsStreaming(isActive);
    };

    const handleTrackSubscribed = (track, publication, participant) => {
      checkActiveStreams();
    };

    const handleTrackUnsubscribed = (track, publication, participant) => {
      track.detach();
      checkActiveStreams();
    };

    room.on(RoomEvent.TrackSubscribed, handleTrackSubscribed);
    room.on(RoomEvent.TrackUnsubscribed, handleTrackUnsubscribed);
    room.on(RoomEvent.TrackMuted, checkActiveStreams);
    room.on(RoomEvent.TrackUnmuted, checkActiveStreams);

    // Initial check
    checkActiveStreams();

    return () => {
      room.off(RoomEvent.TrackSubscribed, handleTrackSubscribed);
      room.off(RoomEvent.TrackUnsubscribed, handleTrackUnsubscribed);
    };
  }, [room]);

  return (
    <div className={`relative w-full h-full bg-black rounded-lg overflow-hidden ${className}`}>
      <video
        ref={videoRef}
        className="w-full h-full object-contain"
        autoPlay
        playsInline
      />
      {!isStreaming && (
        <div className="absolute inset-0 flex items-center justify-center bg-gray-900">
          <div className="text-center text-white">
            <div className="animate-pulse mb-2">●</div>
            <p className="text-sm">Waiting for stream...</p>
          </div>
        </div>
      )}
    </div>
  );
}

