import React from 'react';

const SystemHealth = ({ fcConnected, camConnected }) => {
  return (
    <div style={{ padding: '0', width: '100%', boxSizing: 'border-box' }}>
      {/* Flight Controller Row */}
      <div style={{ background: '#f8fafc', padding: '1rem', borderRadius: '8px', border: '1px solid #e2e8f0', marginBottom: '1rem', display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
        <div style={{ display: 'flex', alignItems: 'center', gap: '10px' }}>
          <div style={{ color: '#64748b' }}>
            <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"><path d="M22 2l-7 20-4-9-9-4 20-7z"></path></svg>
          </div>
          <span style={{ fontSize: '0.9rem', color: '#64748b', fontWeight: '500' }}>Flight Controller</span>
        </div>
        <div style={{ display: 'flex', alignItems: 'center', gap: '0.5rem' }}>
          <span className={`tag ${fcConnected ? 'connected' : 'disconnected'}`}>
            {fcConnected ? 'CONNECTED' : 'DISCONNECTED'}
          </span>
        </div>
      </div>

      {/* Camera Row */}
      <div style={{ background: '#f8fafc', padding: '1rem', borderRadius: '8px', border: '1px solid #e2e8f0', display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
        <div style={{ display: 'flex', alignItems: 'center', gap: '10px' }}>
          <div style={{ color: '#64748b' }}>
            <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"><path d="M23 19a2 2 0 0 1-2 2H3a2 2 0 0 1-2-2V8a2 2 0 0 1 2-2h4l2-3h6l2 3h4a2 2 0 0 1 2 2z" /><circle cx="12" cy="13" r="4" /></svg>
          </div>
          <span style={{ fontSize: '0.9rem', color: '#64748b', fontWeight: '500' }}>Camera Feed</span>
        </div>
        <div style={{ display: 'flex', alignItems: 'center', gap: '0.5rem' }}>
          <span className={`tag ${camConnected ? 'connected' : 'disconnected'}`}>
            {camConnected ? 'ACTIVE' : 'INACTIVE'}
          </span>
        </div>
      </div>
    </div>
  );
};

export default SystemHealth;
