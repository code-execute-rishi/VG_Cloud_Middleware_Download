import React from 'react';

const SystemHealth = ({ fcConnected, camConnected, lastHeartbeat }) => {
    return (
        <div className="status-grid glass-panel" style={{ marginTop: '1rem', padding: '1rem' }}>
            <h3 style={{ margin: '0 0 1rem 0', color: '#888', textTransform: 'uppercase', fontSize: '0.8rem', letterSpacing: '1px' }}>
                System Health
            </h3>

            {/* Flight Controller Row */}
            <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: '0.8rem' }}>
                <span style={{ color: '#ccc' }}>Flight Controller</span>
                <div style={{ display: 'flex', alignItems: 'center', gap: '0.5rem' }}>
                    <span style={{ fontSize: '0.8rem', color: fcConnected ? '#10b981' : '#ef4444' }}>
                        {fcConnected ? 'CONNECTED' : 'NO HEARTBEAT'}
                    </span>
                    <div className={`status-dot ${fcConnected ? 'pulsing' : 'red'}`} />
                </div>
            </div>

            {/* Camera Row */}
            <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
                <span style={{ color: '#ccc' }}>Camera Feed</span>
                <div style={{ display: 'flex', alignItems: 'center', gap: '0.5rem' }}>
                    <span style={{ fontSize: '0.8rem', color: camConnected ? '#10b981' : '#ef4444' }}>
                        {camConnected ? 'ACTIVE' : 'NOT DETECTED'}
                    </span>
                    <div className={`status-dot ${camConnected ? 'pulsing' : 'red'}`} />
                </div>
            </div>

            <style>{`
        .glass-panel {
          background: rgba(0, 0, 0, 0.4);
          backdrop-filter: blur(10px);
          border: 1px solid rgba(255, 255, 255, 0.05);
          border-radius: 12px;
        }
        .status-dot {
          width: 8px;
          height: 8px;
          border-radius: 50%;
        }
        .status-dot.pulsing {
          background-color: #10b981;
          box-shadow: 0 0 0 0 rgba(16, 185, 129, 0.7);
          animation: pulse-green 2s infinite;
        }
        .status-dot.red {
          background-color: #ef4444;
        }
        @keyframes pulse-green {
          0% { box-shadow: 0 0 0 0 rgba(16, 185, 129, 0.7); }
          70% { box-shadow: 0 0 0 6px rgba(16, 185, 129, 0); }
          100% { box-shadow: 0 0 0 0 rgba(16, 185, 129, 0); }
        }
      `}</style>
        </div>
    );
};

export default SystemHealth;
