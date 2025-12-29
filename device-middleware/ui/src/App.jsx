import React, { useState, useEffect } from "react";
import { HashRouter as Router, Routes, Route, useLocation } from "react-router-dom";
import "./App.css";
import MJPEGStream from './components/MJPEGStream';
import SystemHealth from './components/SystemHealth';
import FlightController from './components/FlightController';
import Sidebar from './components/Sidebar';
import DeviceMonitoring from './components/DeviceMonitoring';
import LogsViewer from './components/LogsViewer';
import ZeroTierVPN from './components/ZeroTierVPN';

function App() {
  return (
    <Router>
      <AppContent />
    </Router>
  );
}

function AppContent() {
  const [status, setStatus] = useState(null);
  const [systemInfo, setSystemInfo] = useState(null);
  const [loading, setLoading] = useState(false);

  // No longer needed for sidebar logic, but keeping for reference if needed
  // const location = useLocation(); 

  // Form State
  const [formData, setFormData] = useState({
    ssid: "",
    password: "",
    resolution: "",
    camera_type: "auto",
    camera_device: "auto",
    fc_port: "auto",
    fc_baud: 57600
  });

  const [cameras, setCameras] = useState([]);
  const [serialPorts, setSerialPorts] = useState([]);

  // Poll Status
  useEffect(() => {
    const poll = async () => {
      try {
        const res = await fetch("/api/status");
        if (res.ok) {
          const data = await res.json();
          setStatus(data);
          // Sync form only if not set (Initial Load)
          setFormData(prev => {
            if (prev.resolution === "" && data.camera_config?.resolution) {
              return {
                ...prev,
                resolution: data.camera_config.resolution,
                camera_type: data.camera_config.camera_type || "auto",
                camera_device: data.camera_config.camera_device || "auto",
                fc_port: data.camera_config.fc_port || "auto",
                fc_baud: data.camera_config.fc_baud || 57600
              };
            }
            return prev;
          });
        }
      } catch (e) {
        console.error("Poll failed", e);
      }
    };

    // Fetch System Info & Cameras once
    fetch("/api/system-info").then(r => r.json()).then(setSystemInfo);
    fetch("/api/cameras").then(r => r.json()).then(setCameras).catch(console.error);
    fetch("/api/serial-ports").then(r => r.json()).then(setSerialPorts).catch(console.error);

    const interval = setInterval(poll, 2000);
    poll();
    return () => clearInterval(interval);
  }, []);

  const handleBind = async () => {
    try {
      setLoading(true);
      const res = await fetch("/api/save-config", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ ssid: formData.ssid, password: formData.password }),
      });
      if (!res.ok) throw new Error("Result not OK");
      alert("✅ WiFi Configuration Saved!\n\nNext Step: Go to your Cloud Dashboard and enter the Pairing Code to complete the binding.");
    } catch (err) {
      alert("Bind Failed: " + err.message);
    } finally {
      setLoading(false);
    }
  };

  const handleUpdateConfig = async () => {
    try {
      setLoading(true);
      await fetch("/api/update-config", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          resolution: formData.resolution,
          camera_type: formData.camera_type,
          camera_device: formData.camera_device,
          fc_port: formData.fc_port,
          fc_baud: parseInt(formData.fc_baud)
        }),
      });
      alert("Settings Updated!");
    } catch (e) {
      alert("Failed to update");
    } finally {
      setLoading(false);
    }
  };

  if (!systemInfo) return <div style={{ display: 'flex', height: '100vh', alignItems: 'center', justifyContent: 'center' }}><div className="spinner"></div></div>;

  const isClaimed = status?.is_claimed;
  const showSetup = !isClaimed;

  if (showSetup) {
    // Setup View (No Sidebar) - Refined Premium Design
    return (
      <div className="app-container" style={{ justifyContent: 'center', alignItems: 'center', background: '#f8fafc', height: '100vh', width: '100vw' }}>
        <div style={{ maxWidth: '440px', width: '100%', padding: '0 20px' }}>

          <header style={{ marginBottom: '32px', textAlign: 'center' }}>
            <h1 style={{ fontSize: '1.75rem', fontWeight: '700', color: '#0f172a', marginBottom: '8px', letterSpacing: '-0.025em' }}>Vyom Setup</h1>
            <p style={{ color: '#64748b', fontSize: '1rem' }}>Connect your device to the cloud</p>
          </header>

          <div className="card bind-card" style={{ padding: '0', overflow: 'hidden', boxShadow: '0 4px 6px -1px rgb(0 0 0 / 0.1), 0 2px 4px -2px rgb(0 0 0 / 0.1)' }}>

            {/* Top Blue Accent Bar */}
            <div style={{ height: '4px', background: 'linear-gradient(to right, #2563eb, #3b82f6)' }}></div>

            <div style={{ padding: '32px' }}>
              <div style={{ marginBottom: '24px', textAlign: 'center' }}>
                <h2 style={{ fontSize: '1.25rem', color: '#1e293b', marginBottom: '8px', border: 'none', padding: 0 }}>Connection</h2>
                <div style={{ background: '#f8fafc', padding: '24px', borderRadius: '12px', border: '1px solid #e2e8f0', marginTop: '16px' }}>

                  {status?.auth_status?.connect_url ? (
                    <div style={{ display: 'flex', flexDirection: 'column', alignItems: 'center', gap: '16px' }}>
                      <a
                        href={status.auth_status.connect_url}
                        target="_blank"
                        rel="noopener noreferrer"
                        className="btn-primary"
                        style={{
                          textDecoration: 'none',
                          fontSize: '1rem',
                          padding: '12px 24px',
                          display: 'flex',
                          alignItems: 'center',
                          justifyContent: 'center',
                          width: '100%',
                          fontWeight: '600'
                        }}
                      >
                        Connect Device
                      </a>
                      <p style={{ fontSize: '0.875rem', color: '#64748b', margin: 0 }}>
                        Click to authorize via Cloud Dashboard
                      </p>
                    </div>
                  ) : (
                    <div style={{ display: 'flex', flexDirection: 'column', alignItems: 'center', padding: '20px' }}>
                      <div className="spinner"></div>
                      <p style={{ marginTop: '12px', color: '#64748b', fontSize: '0.875rem' }}>Preparing Setup...</p>
                    </div>
                  )}
                </div>
              </div>

              <div style={{ display: 'flex', alignItems: 'center', gap: '12px', margin: '24px 0' }}>
                <div style={{ height: '1px', flex: 1, background: '#e2e8f0' }}></div>
                <span style={{ fontSize: '0.75rem', color: '#94a3b8', fontWeight: '600', textTransform: 'uppercase' }}>Config</span>
                <div style={{ height: '1px', flex: 1, background: '#e2e8f0' }}></div>
              </div>

              <div className="form-group custom-input" style={{ marginBottom: '16px' }}>
                <label style={{ fontSize: '0.875rem', fontWeight: '600', color: '#334155', marginBottom: '6px' }}>WiFi SSID</label>
                <input
                  type="text" value={formData.ssid}
                  onChange={(e) => setFormData({ ...formData, ssid: e.target.value })}
                  placeholder="Network Name"
                  style={{ width: '100%', padding: '10px 12px', borderRadius: '6px', border: '1px solid #cbd5e1', fontSize: '0.95rem' }}
                />
              </div>
              <div className="form-group custom-input" style={{ marginBottom: '24px' }}>
                <label style={{ fontSize: '0.875rem', fontWeight: '600', color: '#334155', marginBottom: '6px' }}>Password</label>
                <input
                  type="password" value={formData.password}
                  onChange={(e) => setFormData({ ...formData, password: e.target.value })}
                  placeholder="Network Password"
                  style={{ width: '100%', padding: '10px 12px', borderRadius: '6px', border: '1px solid #cbd5e1', fontSize: '0.95rem' }}
                />
              </div>

              <button className="btn-secondary" onClick={handleBind} disabled={loading} style={{ width: '100%', padding: '10px', fontWeight: '600', color: '#334155', background: '#f1f5f9', border: '1px solid #cbd5e1' }}>
                {loading ? "Saving Configuration..." : "Save WiFi Settings"}
              </button>
            </div>
          </div>

          <div style={{ textAlign: 'center', marginTop: '24px' }}>
            <p style={{ color: '#94a3b8', fontSize: '0.8rem' }}>Vyom Device Middleware v2.5.0</p>
          </div>
        </div>
      </div>
    );
  }

  // APP VIEW (Claimed) - Sidebar Layout
  return (
    <div className="app-container">
      <Sidebar
        systemInfo={systemInfo}
        isClaimed={status?.is_claimed}
        isConnected={status?.is_connected}
        userInfo={status?.user_info}
      />

      <main className="main-content">
        <Routes>
          <Route path="/" element={<DashboardView status={status} isConnected={status?.is_connected} />} />
          <Route path="/camera" element={<CameraSettingsView status={status} formData={formData} setFormData={setFormData} cameras={cameras} serialPorts={serialPorts} handleUpdateConfig={handleUpdateConfig} loading={loading} />} />
          <Route path="/monitoring" element={<DeviceMonitoring status={status} />} />
          <Route path="/flight-controller" element={<FlightController />} />
          <Route path="/zerotier" element={<ZeroTierVPN />} />
        </Routes>
      </main>
    </div>
  );
}

// VIEW COMPONENTS

const DashboardView = ({ status, isConnected }) => {
  const lk = status?.livekit_status || {};
  const zt = status?.zerotier_status || {};
  const sysStats = useSystemStats();

  return (
    <div style={{ display: 'flex', flexDirection: 'column', gap: '20px' }}>

      {/* 1. KPI Stats Row */}
      <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fit, minmax(240px, 1fr))', gap: '20px' }}>
        <div className="card" style={{ padding: '1.2rem', display: 'flex', alignItems: 'center', gap: '15px' }}>
          <div style={{ width: '48px', height: '48px', borderRadius: '12px', background: '#dcfce7', display: 'flex', alignItems: 'center', justifyContent: 'center', color: '#16a34a' }}>
            <svg width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"><circle cx="12" cy="12" r="10" /><line x1="2" y1="12" x2="22" y2="12" /><path d="M12 2a15.3 15.3 0 0 1 4 10 15.3 15.3 0 0 1-4 10 15.3 15.3 0 0 1-4-10 15.3 15.3 0 0 1 4-10z" /></svg>
          </div>
          <div>
            <div style={{ fontSize: '0.85rem', color: '#64748b', fontWeight: 'bold', textTransform: 'uppercase' }}>Internet</div>
            <div style={{ fontSize: '1.2rem', fontWeight: '700', color: '#0f172a' }}>Online</div>
          </div>
        </div>

        <div className="card" style={{ padding: '1.2rem', display: 'flex', alignItems: 'center', gap: '15px' }}>
          <div style={{ width: '48px', height: '48px', borderRadius: '12px', background: isConnected ? '#e0f2fe' : '#fef2f2', display: 'flex', alignItems: 'center', justifyContent: 'center', color: isConnected ? '#0284c7' : '#ef4444' }}>
            <svg width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"><path d="M17.5 19c0-3.037-2.463-5.5-5.5-5.5S6.5 15.963 6.5 19" /><path d="M19 16c2.5-1 3.2-3.8 2.5-6.3-.5-1.9-2.3-3.2-4.2-3.1C16.9 3.8 14.3 2 11.5 2.5c-2.4.4-4.2 2.3-4.6 4.7C4.6 7.6 2.6 9 1 9" /></svg>
          </div>
          <div>
            <div style={{ fontSize: '0.85rem', color: '#64748b', fontWeight: 'bold', textTransform: 'uppercase' }}>Cloud Link</div>
            <div style={{ fontSize: '1.2rem', fontWeight: '700', color: '#0f172a' }}>{isConnected ? "Connected" : "Disconnected"}</div>
          </div>
        </div>

        <div className="card" style={{ padding: '1.2rem', display: 'flex', alignItems: 'center', gap: '15px' }}>
          <div style={{ width: '48px', height: '48px', borderRadius: '12px', background: '#f1f5f9', display: 'flex', alignItems: 'center', justifyContent: 'center', color: '#475569' }}>
            <svg width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"><path d="M12 22s8-4 8-10V5l-8-3-8 3v7c0 6 8 10 8 10z" /></svg>
          </div>
          <div>
            <div style={{ fontSize: '0.85rem', color: '#64748b', fontWeight: 'bold', textTransform: 'uppercase' }}>Ownership</div>
            <div style={{ fontSize: '1.2rem', fontWeight: '700', color: '#0f172a' }}>Claimed</div>
          </div>
        </div>
      </div>

      {/* 2. RESTORED: Network Health & System Status (Old Style) */}
      <h2 style={{ fontSize: '1rem', color: '#64748b', textTransform: 'uppercase', margin: '0 0 -10px 5px', letterSpacing: '0.05em' }}>Overview</h2>
      <div style={{ display: 'grid', gridTemplateColumns: 'minmax(0, 2fr) minmax(0, 1fr)', gap: '20px' }}>

        {/* Network Health (LiveKit + ZeroTier) */}
        <div className="card">
          <h2>Network Health</h2>
          <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: '20px' }}>

            {/* LiveKit */}
            <div style={{ background: '#f8fafc', padding: '1rem', borderRadius: '8px', border: '1px solid #e2e8f0' }}>
              <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: '10px' }}>
                <h3 style={{ margin: 0, fontSize: '0.95rem', color: '#334155' }}>LiveKit Video</h3>
                <span className={`tag ${lk.state === 'Connected' ? 'connected' : 'disconnected'}`}>{lk.state || 'Disconnected'}</span>
              </div>
              <div className="info-row" style={{ display: 'flex', justifyContent: 'space-between' }}>
                <span className="info-label">Room Name</span>
                <span className="code">{lk.room_name || "-"}</span>
              </div>
              <div className="info-row" style={{ display: 'flex', justifyContent: 'space-between', marginTop: '5px' }}>
                <span className="info-label">Participants</span>
                <span style={{ fontWeight: '700' }}>{lk.participants || 0}</span>
              </div>
              {lk.last_error && <div style={{ color: '#ef4444', fontSize: '0.85rem', marginTop: '5px' }}>{lk.last_error}</div>}
            </div>

            {/* ZeroTier */}
            <div style={{ background: '#f8fafc', padding: '1rem', borderRadius: '8px', border: '1px solid #e2e8f0' }}>
              <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: '10px' }}>
                <h3 style={{ margin: 0, fontSize: '0.95rem', color: '#334155' }}>ZeroTier VPN</h3>
                <span className={`tag ${zt.state === 'Connected' ? 'connected' : 'disconnected'}`}>{zt.state || 'Disconnected'}</span>
              </div>
              <div className="info-row" style={{ display: 'flex', justifyContent: 'space-between' }}>
                <span className="info-label">IP Address</span>
                <span className="code">{zt.ip_address || "-"}</span>
              </div>
              <div className="info-row" style={{ display: 'flex', justifyContent: 'space-between', marginTop: '5px' }}>
                <span className="info-label">Net ID</span>
                <span className="code">{zt.network_id || "-"}</span>
              </div>
            </div>

          </div>
        </div>

        {/* System Status (Old Style - Simplified) */}
        <div className="card" style={{ display: 'flex', flexDirection: 'column' }}>
          <h2>System Status</h2>
          <div style={{ flex: 1 }}>
            <SystemHealth
              fcConnected={status?.hardware_status?.fc_connected}
              camConnected={status?.hardware_status?.cam_connected}
            />
          </div>
        </div>
      </div>

      {/* 3. NEW: Connection Status (APN) & Hardware Status (Detailed) */}
      <h2 style={{ fontSize: '1rem', color: '#64748b', textTransform: 'uppercase', margin: '10px 0 -10px 5px', letterSpacing: '0.05em' }}>Telemetry & Connection</h2>
      <div style={{ display: 'grid', gridTemplateColumns: 'minmax(0, 1fr) minmax(0, 1fr)', gap: '20px' }}>

        {/* Connection Status: JUST APN/Interface (Removed ZT/LK to avoid duplication) */}
        <div className="card">
          <h2>Connection Status</h2>
          <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: '10px' }}>
            <h3 style={{ margin: 0, fontSize: '0.95rem', color: '#475569' }}>Network Interface</h3>
            <span className="tag connected" style={{ background: '#f1f5f9', color: '#475569' }}>{sysStats?.interfaces?.some(i => i.startsWith('wla') || i.startsWith('wl')) ? 'WiFi' : 'Ethernet'}</span>
          </div>
          <div className="info-row" style={{ display: 'flex', justifyContent: 'space-between', padding: '15px', background: '#f8fafc', borderRadius: '8px', border: '1px solid #e2e8f0' }}>
            <div style={{ display: 'flex', flexDirection: 'column', gap: '5px' }}>
              <span className="info-label" style={{ fontSize: '1rem', color: '#334155' }}>APN/Modem</span>
              <div style={{ fontSize: '0.75rem', color: '#94a3b8' }}>Detected: {sysStats?.interfaces?.join(', ') || '-'}</div>
            </div>
            <span style={{ fontWeight: '700', fontSize: '1.1rem', color: sysStats?.interfaces?.some(i => i.startsWith('wwan') || i.startsWith('ppp') || i.startsWith('usb')) ? '#22c55e' : '#94a3b8' }}>
              {sysStats?.interfaces?.some(i => i.startsWith('wwan') || i.startsWith('ppp') || i.startsWith('usb')) ? 'Active' : 'Not Detected'}
            </span>
          </div>
        </div>

        {/* Hardware Status: Detailed FC Stats */}
        <div className="card" style={{ display: 'flex', flexDirection: 'column' }}>
          <h2>Hardware Status</h2>
          <div style={{ flex: 1, display: 'flex', flexDirection: 'column', gap: '15px' }}>
            <div style={{ background: '#f8fafc', padding: '1.2rem', borderRadius: '8px', border: '1px solid #e2e8f0', display: 'flex', flexDirection: 'column', justifyContent: 'center', minHeight: '100px' }}>
              <div className="info-row" style={{ display: 'flex', justifyContent: 'space-between', marginBottom: '8px' }}>
                <span className="info-label" style={{ fontSize: '1rem' }}>FC Mode</span>
                <span style={{ fontWeight: '700', fontSize: '1.1rem' }}>{status?.telemetry?.system?.mode || "UNKNOWN"}</span>
              </div>
              <div className="info-row" style={{ display: 'flex', justifyContent: 'space-between' }}>
                <span className="info-label" style={{ fontSize: '1rem' }}>Arming</span>
                <span style={{ fontWeight: '700', fontSize: '1.1rem', color: status?.telemetry?.system?.armed ? '#ef4444' : '#22c55e' }}>
                  {status?.telemetry?.system?.armed ? "ARMED" : "DISARMED"}
                </span>
              </div>
            </div>
          </div>
        </div>

      </div>
    </div>
  );
};

// Helper Hook (can be extracted later, but putting here for context scope in this generic replace)
const useSystemStats = () => {
  const [stats, setStats] = React.useState(null);
  React.useEffect(() => {
    const fetchStats = () => fetch("/api/system-stats").then(r => r.json()).then(setStats).catch(() => { });
    fetchStats();
    const interval = setInterval(fetchStats, 2000);
    return () => clearInterval(interval);
  }, []);
  return stats;
};

const CameraSettingsView = ({ status, formData, setFormData, cameras, serialPorts, handleUpdateConfig, loading }) => (
  <div style={{ display: 'flex', flexDirection: 'column', gap: '20px' }}>
    {/* Top Section: Health & Stream */}
    <div style={{ display: 'flex', flexDirection: 'column', gap: '20px' }}>
      {/* System Health Full Width */}
      <div className="card" style={{ padding: '1rem' }}>
        <h2 style={{ border: 'none', padding: 0, marginBottom: '0.5rem', fontSize: '1rem', textTransform: 'uppercase', color: '#64748b' }}>System Overview</h2>
        <SystemHealth
          fcConnected={status?.hardware_status?.fc_connected}
          camConnected={status?.hardware_status?.cam_connected}
        />
      </div>

      {/* Stream & Settings Grid */}
      <div style={{ display: 'grid', gridTemplateColumns: 'minmax(0, 1.5fr) minmax(0, 1fr)', gap: '20px' }}>
        <div className="card" style={{ display: 'flex', flexDirection: 'column' }}>
          <h2>Local Camera Stream</h2>
          <div className="stream-container" style={{ flex: 1, background: '#000', borderRadius: '8px', overflow: 'hidden', minHeight: '300px', display: 'flex', alignItems: 'center', justifyContent: 'center' }}>
            <MJPEGStream src="/api/stream" alt="Local Stream" />
          </div>
        </div>

        <div className="card">
          <h2>Configuration</h2>
          <h3>Camera Settings</h3>
          <div style={{ display: 'flex', gap: '15px', marginBottom: '15px' }}>
            <div className="form-group" style={{ flex: 1 }}>
              <label>Camera Device</label>
              <select
                value={formData.camera_device}
                onChange={(e) => setFormData({ ...formData, camera_device: e.target.value })}
              >
                <option value="auto">Auto Detection</option>
                {cameras && cameras.map(cam => (
                  <option key={cam} value={cam}>{cam}</option>
                ))}
              </select>
            </div>
          </div>
          <div className="form-group">
            <label>Camera Type</label>
            <select
              value={formData.camera_type}
              onChange={(e) => setFormData({ ...formData, camera_type: e.target.value })}
            >
              <option value="auto">Auto Detect</option>
              <option value="csi">CSI Ribbon Cable (RPi)</option>
              <option value="usb">USB Camera</option>
            </select>
          </div>

          <div className="form-group">
            <label>Camera Resolution</label>
            <select
              value={formData.resolution}
              onChange={(e) => setFormData({ ...formData, resolution: e.target.value })}
            >
              <option value="640x480">480p</option>
              <option value="1280x720">720p</option>
              <option value="1920x1080">1080p</option>
            </select>
          </div>

          <div style={{ textAlign: 'right', marginTop: '20px' }}>
            <button className="btn-primary" onClick={handleUpdateConfig} disabled={loading}>
              {loading ? "Updating..." : "Update Settings"}
            </button>
          </div>
        </div>
      </div>
    </div>

    {/* 3. Camera Logs */}
    <LogsViewer service="camera" />
  </div>
);

// LogsViewer extracted to components/LogsViewer.jsx

const NetworkStatusCard = ({ status }) => {
  const lk = status?.livekit_status || {};
  const zt = status?.zerotier_status || {};

  return (
    <div className="card">
      <h2>Network Health</h2>

      <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: '20px' }}>
        {/* LiveKit Section */}
        <div>
          <h3 style={{ marginTop: '0', fontSize: '0.9rem', color: '#64748b', textTransform: 'uppercase' }}>LiveKit (Video/Control)</h3>
          <div style={{ background: '#f8fafc', padding: '1rem', borderRadius: '8px', border: '1px solid #e2e8f0' }}>
            <div className="info-row" style={{ display: 'flex', justifyContent: 'space-between', marginBottom: '0.5rem' }}>
              <span className="info-label" style={{ fontSize: '0.85rem', color: '#64748b' }}>Status</span>
              {lk.state === 'Connected' ? (
                <span className="tag connected">Connected</span>
              ) : (
                <span className="tag" style={{ color: '#ef4444', background: '#fee2e2' }}>{lk.state || "Disconnected"}</span>
              )}
            </div>
            <div className="info-row" style={{ display: 'flex', justifyContent: 'space-between', marginBottom: '0.5rem' }}>
              <span className="info-label" style={{ fontSize: '0.85rem', color: '#64748b' }}>Room</span>
              <span className="info-value code">{lk.room_name || "-"}</span>
            </div>
            <div className="info-row" style={{ display: 'flex', justifyContent: 'space-between', marginBottom: '0.5rem' }}>
              <span className="info-label" style={{ fontSize: '0.85rem', color: '#64748b' }}>Peers</span>
              <span className="info-value" style={{ fontWeight: 'bold' }}>{lk.participants || 0}</span>
            </div>
            {lk.last_error && (
              <div className="info-row" style={{ display: 'flex', justifyContent: 'space-between', color: '#ef4444' }}>
                <span className="info-label" style={{ color: '#ef4444' }}>Error</span>
                <span className="info-value">{lk.last_error}</span>
              </div>
            )}
          </div>
        </div>

        {/* ZeroTier Section */}
        <div>
          <h3 style={{ marginTop: '0', fontSize: '0.9rem', color: '#64748b', textTransform: 'uppercase' }}>ZeroTier (VPN)</h3>
          <div style={{ background: '#f8fafc', padding: '1rem', borderRadius: '8px', border: '1px solid #e2e8f0' }}>
            <div className="info-row" style={{ display: 'flex', justifyContent: 'space-between', marginBottom: '0.5rem' }}>
              <span className="info-label" style={{ fontSize: '0.85rem', color: '#64748b' }}>Status</span>
              {zt.state === 'Connected' ? (
                <span className="tag connected">Connected</span>
              ) : (
                <span className="tag" style={{ color: '#ef4444', background: '#fee2e2' }}>{zt.state || "Disconnected"}</span>
              )}
            </div>
            <div className="info-row" style={{ display: 'flex', justifyContent: 'space-between', marginBottom: '0.5rem' }}>
              <span className="info-label" style={{ fontSize: '0.85rem', color: '#64748b' }}>IP Address</span>
              <span className="info-value code">{zt.ip_address || "-"}</span>
            </div>
            {zt.last_error && (
              <div className="info-row" style={{ display: 'flex', justifyContent: 'space-between', color: '#ef4444' }}>
                <span className="info-label" style={{ color: '#ef4444' }}>Error</span>
                <span className="info-value">{zt.last_error}</span>
              </div>
            )}
          </div>
        </div>
      </div>
    </div>
  );
};

export default App;
