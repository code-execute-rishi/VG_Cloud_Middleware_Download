import React, { useState, useEffect, useRef } from 'react';
import Peer from 'peerjs';
import { Video, Radio, AlertCircle } from 'lucide-react';

const VideoFeed = () => {
    const [mode, setMode] = useState('live'); // 'live' or 'simulation'
    const [peerId, setPeerId] = useState(null);
    const [stream, setStream] = useState(null);
    const videoRef = useRef(null);
    const peerRef = useRef(null);

    // Simulation video URL (Big Buck Bunny or similar loop)
    const SIM_VIDEO_URL = "https://media.w3.org/2010/05/sintel/trailer_hd.mp4";

    useEffect(() => {
        if (mode === 'live') {
            // Initialize PeerJS
            // We use a random ID for the GCS client, but we connect to the known Drone ID
            const peer = new Peer(null, { 
                host: '98.92.19.117',
                port: 9000,
                path: '/'
            });

            peer.on('open', (id) => {
                console.log('My peer ID is: ' + id);
                setPeerId(id);

                // Create a dummy stream using a canvas to satisfy PeerJS requirements
                const canvas = document.createElement('canvas');
                canvas.width = 1;
                canvas.height = 1;
                const dummyStream = canvas.captureStream(10); // 10 FPS

                // Immediately call the drone
                console.log('Calling drone-cam-001...');
                const callObj = peer.call('drone-cam-001', dummyStream);

                if (callObj) {
                    callObj.on('stream', (remoteStream) => {
                        console.log('Received drone stream');
                        console.log('Stream tracks:', remoteStream.getTracks());
                        setStream(remoteStream);
                        if (videoRef.current) {
                            videoRef.current.srcObject = remoteStream;
                            // Force play when data loads
                            videoRef.current.onloadedmetadata = () => {
                                videoRef.current.play().catch(e => console.error("Autoplay Error:", e));
                            };
                        }
                    });

                    callObj.on('error', (err) => {
                        console.error("Call error:", err);
                    });
                } else {
                    console.error("Failed to initiate call. Drone might be offline.");
                }
            });

            peer.on('error', (err) => {
                console.error("Peer error:", err);
            });

            peerRef.current = peer;

            return () => {
                if (peerRef.current) {
                    peerRef.current.destroy();
                }
                setStream(null);
            };
        } else {
            // Simulation mode cleanup
            setStream(null);
            if (videoRef.current) {
                videoRef.current.srcObject = null;
            }
        }
    }, [mode]);

    return (
        <div className="h-full w-full flex flex-col bg-military-800 rounded-lg border border-military-700 overflow-hidden relative">
            {/* Header / Controls */}
            <div className="absolute top-0 left-0 right-0 p-3 bg-gradient-to-b from-black/70 to-transparent z-10 flex justify-between items-center">
                <div className="flex items-center gap-2">
                    <div className={`flex items-center gap-1 px-2 py-1 rounded text-xs font-bold ${mode === 'live' ? 'bg-red-500/20 text-red-500 border border-red-500/50' : 'bg-blue-500/20 text-blue-500 border border-blue-500/50'}`}>
                        {mode === 'live' ? <div className="w-2 h-2 rounded-full bg-red-500 animate-pulse" /> : <Video size={12} />}
                        {mode === 'live' ? 'LIVE FEED' : 'SIMULATION'}
                    </div>
                </div>
                
                <button 
                    onClick={() => setMode(mode === 'live' ? 'simulation' : 'live')}
                    className="text-xs bg-military-700 hover:bg-military-600 text-white px-3 py-1 rounded border border-military-600 transition-colors"
                >
                    Switch to {mode === 'live' ? 'Simulation' : 'Live'}
                </button>
            </div>

            {/* Video Area */}
            <div className="flex-1 bg-black relative flex items-center justify-center">
                {mode === 'live' ? (
                    stream ? (
                        <video 
                            ref={videoRef} 
                            className="w-full h-full object-cover" 
                            autoPlay 
                            playsInline 
                            muted // Muted to allow autoplay
                        />
                    ) : (
                        <div className="text-center text-gray-500 flex flex-col items-center gap-2">
                            <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-military-accent"></div>
                            <p className="text-sm">Waiting for Drone Stream...</p>
                            <p className="text-xs text-gray-600">Peer Server: :9000</p>
                        </div>
                    )
                ) : (
                    <video 
                        src={SIM_VIDEO_URL} 
                        className="w-full h-full object-cover" 
                        autoPlay 
                        loop 
                        muted 
                    />
                )}

                {/* Crosshair Overlay */}
                <div className="absolute inset-0 pointer-events-none opacity-30">
                    <div className="absolute top-1/2 left-1/2 -translate-x-1/2 -translate-y-1/2 w-12 h-12 border border-white/50 rounded-full"></div>
                    <div className="absolute top-1/2 left-1/2 -translate-x-1/2 -translate-y-1/2 w-1 h-1 bg-red-500 rounded-full"></div>
                    <div className="absolute top-1/2 left-0 right-0 h-px bg-white/20"></div>
                    <div className="absolute top-0 bottom-0 left-1/2 w-px bg-white/20"></div>
                </div>
            </div>
        </div>
    );
};

export default VideoFeed;
