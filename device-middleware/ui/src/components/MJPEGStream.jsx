import React, { useState, useEffect, useRef } from 'react';

const MJPEGStream = ({ src, alt, style }) => {
    const [streamUrl, setStreamUrl] = useState(src);
    const [error, setError] = useState(false);
    const [retryCount, setRetryCount] = useState(0);
    const timeoutRef = useRef(null);

    useEffect(() => {
        // When prop src changes (e.g. resolution update), reset everything
        setStreamUrl(`${src}?t=${Date.now()}`);
        setError(false);
        setRetryCount(0);

        return () => {
            if (timeoutRef.current) clearTimeout(timeoutRef.current);
        };
    }, [src]);

    const handleError = () => {
        console.log("Stream Disconnected. Retrying...");
        setError(true);

        // Exponential backoff or simple retry
        const delay = Math.min(1000 * (retryCount + 1), 5000);

        timeoutRef.current = setTimeout(() => {
            setRetryCount(c => c + 1);
            setStreamUrl(`${src}?t=${Date.now()}`); // Force reload
            setError(false); // Try to show image again
        }, delay);
    };

    const handleLoad = () => {
        // Stream connected successfully
        setError(false);
        setRetryCount(0);
    };

    return (
        <div style={{ position: 'relative', width: '100%', height: '100%', minHeight: '300px', background: '#000', ...style }}>
            {error && (
                <div style={{
                    position: 'absolute', top: 0, left: 0, right: 0, bottom: 0,
                    display: 'flex', alignItems: 'center', justifyContent: 'center',
                    color: 'white', flexDirection: 'column', gap: '10px'
                }}>
                    <div className="spinner" style={{
                        width: '30px', height: '30px', border: '3px solid rgba(255,255,255,0.3)',
                        borderTopColor: '#fff', borderRadius: '50%', animation: 'spin 1s linear infinite'
                    }} />
                    <span style={{ fontSize: '0.9rem', opacity: 0.8 }}>Reconnecting...</span>
                </div>
            )}

            <img
                src={streamUrl}
                alt={alt}
                style={{ width: '100%', height: '100%', objectFit: 'contain', opacity: error ? 0 : 1 }}
                onError={handleError}
                onLoad={handleLoad}
            />
            <style>{`
        @keyframes spin { to { transform: rotate(360deg); } }
      `}</style>
        </div>
    );
};

export default MJPEGStream;
