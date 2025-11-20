import React, { useState, useEffect, useRef } from 'react';
import Peer from 'peerjs';
import { Video, RefreshCw, Play, Activity } from 'lucide-react';

const VideoFeed = () => {
    const [mode, setMode] = useState('live'); // 'live' or 'simulation'
    const [connectionStatus, setConnectionStatus] = useState('Initializing...');
    const [videoState, setVideoState] = useState({
        readyState: 0,
        paused: true,
        muted: true,
        hasStream: false
    });
    
    const videoRef = useRef(null);
    const peerRef = useRef(null);
    const [stream, setStream] = useState(null);

    // Simulation video URL
    const SIM_VIDEO_URL = "https://media.w3.org/2010/05/sintel/trailer_hd.mp4";

    const updateVideoState = () => {
        if (videoRef.current) {
            setVideoState({
                readyState: videoRef.current.readyState,
                paused: videoRef.current.paused,
                muted: videoRef.current.muted,
                hasStream: !!videoRef.current.srcObject
            });
        }
    };

    useEffect(() => {
        let peer = null;

        if (mode === 'live') {
            setConnectionStatus('Connecting to PeerServer...');
            
            // Initialize PeerJS
            peer = new Peer(null, { 
                host: '98.92.19.117',
                port: 9000,
                path: '/'
            });

            peer.on('open', (id) => {
                console.log('My peer ID is: ' + id);
                setConnectionStatus(`Connected (ID: ${id}). Calling Drone...`);

                // Create dummy stream
                const canvas = document.createElement('canvas');
                canvas.width = 1; canvas.height = 1;
                const dummyStream = canvas.captureStream(10);

                // Call drone
                const callObj = peer.call('drone-cam-001', dummyStream);

                if (callObj) {
                    setConnectionStatus('Calling...');
                    
                    callObj.on('stream', (remoteStream) => {
                        console.log('Received drone stream', remoteStream);
                        setConnectionStatus('Stream Received. Attempting Playback...');
                        setStream(remoteStream);
                        
                        if (videoRef.current) {
                            videoRef.current.srcObject = remoteStream;
                            videoRef.current.muted = true;
                            
                            const playPromise = videoRef.current.play();
                            if (playPromise !== undefined) {
                                playPromise
                                    .then(() => setConnectionStatus('Playing Stream'))
                                    .catch(e => {
                                        console.error("Autoplay Error:", e);
                                        setConnectionStatus(`Autoplay Blocked: ${e.message}`);
                                    });
                            }
                        }
                        updateVideoState();
                    });

                    callObj.on('error', (err) => {
                        console.error("Call error:", err);
                        setConnectionStatus(`Call Error: ${err.type}`);
                    });
                    
                    callObj.on('close', () => {
                         setConnectionStatus('Call Closed');
                         setStream(null);
                    });
                } else {
                    setConnectionStatus('Failed to initiate call');
                }
            });

            peer.on('error', (err) => {
                console.error("Peer error:", err);
                setConnectionStatus(`Peer Error: ${err.type}`);
            });

            peerRef.current = peer;
        } else {
            setConnectionStatus('Simulation Mode');
            setStream(null);
        }

        return () => {
            if (peer) peer.destroy();
            setStream(null);
        };
    }, [mode]);

    // Periodic state update for debugging
    useEffect(() => {
        const interval = setInterval(updateVideoState, 1000);
        return () => clearInterval(interval);
    }, []);

    const handleForcePlay = () => {
        if (videoRef.current) {
            videoRef.current.muted = true;
            videoRef.current.play()
                .then(() => console.log("Manual play success"))
                .catch(e => console.error("Manual play error", e));
        }
    };

    const handleReload = () => {
        setMode('simulation');
        setTimeout(() => setMode('live'), 100);
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
                        {connectionStatus}
                    </div>
                </div>
                
                <div className="flex gap-2">
                    <button 
                        onClick={handleReload}
                        className="p-1 bg-military-700 hover:bg-military-600 text-white rounded border border-military-600"
                        title="Reload Connection"
                    >
                        <RefreshCw size={14} />
                    </button>
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
                {mode === 'live' ? (
                    <video 
                        ref={videoRef} 
                        className="w-full h-full object-cover" 
                        autoPlay 
                        playsInline 
                        muted
                        onLoadedMetadata={updateVideoState}
                        onCanPlay={updateVideoState}
                        onPause={updateVideoState}
                        onPlay={updateVideoState}
                        onTimeUpdate={updateVideoState}
                    />
                ) : (
                    <video 
                        src={SIM_VIDEO_URL} 
                        className="w-full h-full object-cover" 
                        autoPlay 
                        loop 
                        muted 
                    />
                )}

                {/* Debug Overlay */}
                <div className="absolute bottom-4 left-4 bg-black/80 text-green-400 p-2 rounded border border-green-900 font-mono text-[10px] pointer-events-none z-20">
                    <div>STATUS: {connectionStatus}</div>
                    <div>READY_STATE: {videoState.readyState}</div>
                    <div>PAUSED: {videoState.paused ? 'YES' : 'NO'}</div>
                    <div>MUTED: {videoState.muted ? 'YES' : 'NO'}</div>
                    <div>HAS_STREAM: {videoState.hasStream ? 'YES' : 'NO'}</div>
                </div>

                {/* Force Play Button */}
                {mode === 'live' && videoState.paused && (
                    <button 
                        onClick={handleForcePlay}
                        className="absolute z-30 bg-red-600 hover:bg-red-700 text-white px-6 py-3 rounded-full font-bold flex items-center gap-2 shadow-lg animate-bounce"
                    >
                        <Play size={24} fill="currentColor" />
                        FORCE PLAY
                    </button>
                )}
            </div>
        </div>
    );
};

export default VideoFeed;
