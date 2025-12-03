import React, { useState } from 'react';
import { useTracks } from '@livekit/components-react';
import { Track } from 'livekit-client';
import { Video, RefreshCw, Activity } from 'lucide-react';

const VideoFeed = () => {
    const [mode, setMode] = useState('live'); // 'live' or 'simulation'

    // Get Camera Tracks from LiveKit - TASK 3 FIX 1
    const tracks = useTracks(
        [
            { source: Track.Source.Camera, withPlaceholder: false }
        ],
        { onlySubscribed: true }
    );

    // Simulation video URL
    const SIM_VIDEO_URL = "https://media.w3.org/2010/05/sintel/trailer_hd.mp4";

    const renderLiveVideo = () => {
        const cameraTrack = tracks.find(t => t.source === Track.Source.Camera);

        if (cameraTrack && cameraTrack.publication.isSubscribed) {
            return (
                <video
                    ref={(el) => {
                        if (el) cameraTrack.publication.track?.attach(el);
                    }}
                    className="w-full h-full object-cover"
                    autoPlay
                    playsInline
                    muted
                />
            );
        }

        return (
            <div className="flex flex-col items-center justify-center h-full text-gray-500">
                <Activity className="animate-pulse mb-2" size={48} />
                <p>Waiting for Drone Video...</p>
            </div>
        );
    };

    return (
        <div className="h-full w-full flex flex-col bg-military-800 rounded-lg border border-military-700 overflow-hidden relative">
            {/* Header */}
            <div className="absolute top-0 left-0 right-0 p-3 bg-gradient-to-b from-black/70 to-transparent z-10 flex justify-between items-center">
                <div className="flex items-center gap-2">
                    <div className={`flex items-center gap-1 px-2 py-1 rounded text-xs font-bold ${mode === 'live' ? 'bg-red-500/20 text-red-500 border border-red-500/50' : 'bg-blue-500/20 text-blue-500 border border-blue-500/50'}`}>
                        {mode === 'live' ? <div className="w-2 h-2 rounded-full bg-red-500 animate-pulse" /> : <Video size={12} />}
                        {mode === 'live' ? 'LIVE FEED' : 'SIMULATION'}
                    </div>
                    <div className="text-[10px] text-gray-400 font-mono bg-black/50 px-2 py-1 rounded">
                        {mode === 'live' ? (tracks.length > 0 ? 'Connected' : 'Searching...') : 'Simulated'}
                    </div>
                </div>

                <div className="flex gap-2">
                    <button
                        onClick={() => setMode(mode === 'live' ? 'simulation' : 'live')}
                        className="text-xs bg-military-700 hover:bg-military-600 text-white px-3 py-1 rounded border border-military-600 transition-colors"
                    >
                        Switch to {mode === 'live' ? 'Sim' : 'Live'}
                    </button>
                </div>
            </div>

            {/* Video Area */}
            <div className="flex-1 bg-black relative flex items-center justify-center overflow-hidden">
                {mode === 'live' ? renderLiveVideo() : (
                    <video
                        src={SIM_VIDEO_URL}
                        className="w-full h-full object-cover"
                        autoPlay
                        loop
                        muted
                    />
                )}
            </div>
        </div>
    );
};

export default VideoFeed;
