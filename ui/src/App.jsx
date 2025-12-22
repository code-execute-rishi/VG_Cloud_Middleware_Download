import React, { useState, useEffect } from "react";
import "./App.css";
// import QRCode from "react-qr-code"; // Removed for v3
import MJPEGStream from './components/MJPEGStream';
import SystemHealth from './components/SystemHealth';

function App() {
  const [view, setView] = useState("dashboard"); // 'dashboard', 'camera', 'logs'
  const [status, setStatus] = useState(null);
  const [systemInfo, setSystemInfo] = useState(null);
  const [loading, setLoading] = useState(false);

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

  if (!systemInfo) return <div className="container center-content"><div className="spinner"></div></div>;

  const isConfigured = status?.is_configured;
  const isConnected = status?.is_connected;
  const isClaimed = status?.is_claimed;

  // NOTE: If device is not CLAIMED, we force the Bind Screen (Setup), 
  // even if it is technically configured (WiFi is set). 
  // This allows the user to see the Pairing Code easily if they Unbound/Forgot it.
  const showSetup = !isClaimed;

  if (showSetup) {
    // SETUP VIEW (Bind Flow)
    return (
      <div className="app-container">
        <header className="top-bar">
          <div className="logo-section">
            <div className="logo-icon">📡</div>
            <h1>Vyom Setup (v2.2)</h1>
          </div>
        </header>

        <main className="main-content">
          <div className="card bind-card">
            <h2>Connect Device</h2>
            <p>Link this device to your VG Cloud account</p>

            <div style={{ background: 'white', padding: '40px', borderRadius: '8px', margin: '20px 0', textAlign: 'center' }}>
              {status?.auth_status?.connect_url ? (
                <div style={{ display: 'flex', flexDirection: 'column', alignItems: 'center', gap: '20px' }}>
                  <div className="pulse-ring" style={{ width: '80px', height: '80px', background: '#e0f2fe', borderRadius: '50%', display: 'flex', alignItems: 'center', justifyContent: 'center', color: '#0284c7', fontSize: '2rem' }}>
                    🔗
                  </div>

                  <a
                    href={status.auth_status.connect_url}
                    target="_blank"
                    rel="noopener noreferrer"
                    className="btn-primary"
                    style={{
                      textDecoration: 'none',
                      fontSize: '1.2rem',
                      padding: '16px 32px',
                      background: 'linear-gradient(135deg, #2563eb 0%, #1d4ed8 100%)',
                      boxShadow: '0 4px 6px -1px rgba(0, 0, 0, 0.1), 0 2px 4px -1px rgba(0, 0, 0, 0.06)'
                    }}
                  >
                    🚀 Connect Device
                  </a>

                  <p style={{ fontSize: '0.9rem', color: '#64748b', maxWidth: '300px', lineHeight: '1.5' }}>
                    Clicking this will open the VG Cloud dashboard to authorize this device.
                  </p>
                </div>
              ) : (
                <div style={{ display: 'flex', flexDirection: 'column', alignItems: 'center' }}>
                  <div className="spinner"></div>
                  <p style={{ marginTop: '10px', color: '#666' }}>Fetching Setup URL...</p>
                </div>
              )}
            </div>

            <hr style={{ width: '100%', borderColor: '#eee', margin: '20px 0' }} />

            <h3>WiFi Settings</h3>
            <div className="form-group" style={{ width: '100%', textAlign: 'left' }}>
              <label>WiFi SSID</label>
              <input
                type="text" value={formData.ssid}
                onChange={(e) => setFormData({ ...formData, ssid: e.target.value })}
                placeholder="SSID"
              />
            </div>
            <div className="form-group" style={{ width: '100%', textAlign: 'left' }}>
              <label>Password</label>
              <input
                type="password" value={formData.password}
                onChange={(e) => setFormData({ ...formData, password: e.target.value })}
                placeholder="Password"
              />
            </div>

            <button className="btn-primary" onClick={handleBind} disabled={loading}>
              {loading ? "Saving..." : "Update WiFi"}
            </button>
          </div>
        </main>
      </div>
    );
  }

  // DASHBOARD VIEW (Claimed)
  return (
    <div className="app-container">
      <header className="top-bar">
        <div className="logo-section">
          <div className="logo-icon green">✔️</div>
          <div className="logo-text">
            <h1>Device Active</h1>
            <p className="subtitle">ID: {systemInfo.device_id}</p>
          </div>
        </div>
        <div className="nav-tabs">
          <button
            className={`nav-tab ${view === 'dashboard' ? 'active' : ''}`}
            onClick={() => setView('dashboard')}
          >
            Dashboard
          </button>
          <button
            className={`nav-tab ${view === 'camera' ? 'active' : ''}`}
            onClick={() => setView('camera')}
          >
            FC/Camera Settings
          </button>
          <button
            className={`nav-tab ${view === 'logs' ? 'active' : ''}`}
            onClick={() => setView('logs')}
          >
            System Logs
          </button>
        </div>
      </header>

      <main className="main-content">
        {view === 'dashboard' && (
          <div className="card">
            <h2>System Status</h2>
            <div className="status-grid">
              <StatusItem label="Internet" value="Online" active={true} />
              <StatusItem label="Cloud Link" value={isConnected ? "Connected" : "Connecting..."} active={isConnected} />
              <StatusItem label="Ownership" value="Claimed" active={true} />
            </div>
          </div>
        )}

        {view === 'dashboard' && (
          <NetworkStatusCard status={status} />
        )}

        {view === 'camera' && (
          <div className="card">

            <SystemHealth
              fcConnected={status?.hardware_status?.fc_connected}
              camConnected={status?.hardware_status?.cam_connected}
            />

            <h3>Local Camera Stream</h3>
            <div className="stream-container" style={{ marginBottom: '20px', background: '#000', borderRadius: '8px', overflow: 'hidden', minHeight: '300px' }}>
              <MJPEGStream
                src="/api/stream"
                alt="Local Stream"
              />
            </div>

            {/* Camera Config Section */}
            <h3>Camera Settings</h3>
            <div style={{ display: 'flex', gap: '15px', marginBottom: '0px' }}>
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
              <div className="form-group" style={{ flex: 1 }}>
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

            {/* Flight Controller Section */}
            <h3 style={{ marginTop: '2rem' }}>Flight Controller</h3>

            {/* FC Status Info */}
            <div className="card" style={{ padding: '1rem', marginBottom: '1.5rem', background: '#fbfbfd', border: '1px solid #eee' }}>
              <div className="info-row" style={{ justifyContent: 'space-between', marginBottom: '0.5rem' }}>
                <span className="info-label">Connection Status</span>
                {status?.hardware_status?.fc_connected ? (
                  <span className="tag connected">Connected</span>
                ) : (
                  <span className="tag" style={{ color: 'red', background: '#ffebeb' }}>Disconnected</span>
                )}
              </div>
              {status?.hardware_status?.fc_connected && (
                <>
                  <div className="info-row" style={{ justifyContent: 'space-between', marginBottom: '0.5rem' }}>
                    <span className="info-label">Autopilot Type</span>
                    <span className="info-value">{status?.hardware_status?.fc_type || "Unknown"}</span>
                  </div>
                  <div className="info-row" style={{ justifyContent: 'space-between', marginBottom: '0.5rem' }}>
                    <span className="info-label">Firmware</span>
                    <span className="info-value">{status?.hardware_status?.fc_firmware || "Unknown"}</span>
                  </div>
                  <div className="info-row" style={{ justifyContent: 'space-between' }}>
                    <span className="info-label">Current Port</span>
                    <span className="info-value code">{status?.hardware_status?.current_port}</span>
                  </div>
                </>
              )}
            </div>

            <div style={{ display: 'flex', gap: '15px' }}>
              <div className="form-group" style={{ flex: 2 }}>
                <label>Serial Port</label>
                <select
                  value={formData.fc_port}
                  onChange={(e) => setFormData({ ...formData, fc_port: e.target.value })}
                >
                  <option value="auto">Auto-Detect (Recommended)</option>
                  {serialPorts && serialPorts.map(port => (
                    <option key={port} value={port}>{port}</option>
                  ))}
                </select>
              </div>
              <div className="form-group" style={{ flex: 1 }}>
                <label>Baud Rate</label>
                <select
                  value={formData.fc_baud}
                  onChange={(e) => setFormData({ ...formData, fc_baud: parseInt(e.target.value) })}
                >
                  <option value="57600">57600 (Telemetry)</option>
                  <option value="115200">115200 (Default)</option>
                  <option value="921600">921600 (High Speed)</option>
                </select>
              </div>
            </div>

            <button className="btn-primary" onClick={handleUpdateConfig} disabled={loading}>
              {loading ? "Updating..." : "Update Settings"}
            </button>
          </div>
        )}

        {view === 'logs' && (
          <div className="card">
            <h2>System Logs</h2>
            <div style={{ display: 'flex', justifyContent: 'flex-end', marginBottom: '10px' }}>
              <a href="/api/logs?download=true" className="btn-small" target="_blank" rel="noopener noreferrer">
                Download Logs
              </a>
            </div>
            <LogsViewer />
          </div>
        )}

      </main>
    </div>
  );
}

const StatusItem = ({ label, value, active, warned }) => (
  <div className="status-item">
    <div className="status-label">{label}</div>
    <div className={`status-value ${active ? 'active' : ''} ${warned ? 'warn' : ''}`}>
      {value}
    </div>
  </div>
);

const LogsViewer = () => {
  const [logs, setLogs] = useState("Loading logs...");

  useEffect(() => {
    fetch("/api/logs")
      .then(res => {
        if (!res.ok) throw new Error("Failed to load");
        return res.text();
      })
      .then(text => setLogs(text))
      .catch(err => setLogs("Error loading logs: " + err.message));
  }, []);

  return (
    <div style={{
      background: '#1a1a1a',
      color: '#00ff00',
      padding: '15px',
      borderRadius: '8px',
      fontFamily: 'source-code-pro, Menlo, Monaco, Consolas, "Courier New", monospace',
      fontSize: '13px',
      lineHeight: '1.5',
      height: '500px',
      overflow: 'auto',
      whiteSpace: 'pre',
      border: '1px solid #333'
    }}>
      {logs}
    </div>
  );
};



const NetworkStatusCard = ({ status }) => {
  const lk = status?.livekit_status || {};
  const zt = status?.zerotier_status || {};

  return (
    <div className="card">
      <h2>Network Health</h2>

      {/* LiveKit Section */}
      <h3 style={{ marginTop: '1rem', fontSize: '1rem', color: '#666' }}>LiveKit (Video/Control)</h3>
      <div style={{ background: '#fbfbfd', padding: '1rem', borderRadius: '8px', border: '1px solid #eee' }}>
        <div className="info-row" style={{ justifyContent: 'space-between', marginBottom: '0.5rem' }}>
          <span className="info-label">Status</span>
          {lk.state === 'Connected' ? (
            <span className="tag connected">Connected</span>
          ) : (
            <span className="tag" style={{ color: 'red', background: '#ffebeb' }}>{lk.state || "Disconnected"}</span>
          )}
        </div>
        <div className="info-row" style={{ justifyContent: 'space-between', marginBottom: '0.5rem' }}>
          <span className="info-label">Room</span>
          <span className="info-value code">{lk.room_name || "-"}</span>
        </div>
        <div className="info-row" style={{ justifyContent: 'space-between', marginBottom: '0.5rem' }}>
          <span className="info-label">Peers</span>
          <span className="info-value">{lk.participants || 0}</span>
        </div>
        {lk.last_error && (
          <div className="info-row" style={{ justifyContent: 'space-between', color: 'red' }}>
            <span className="info-label" style={{ color: 'red' }}>Error</span>
            <span className="info-value">{lk.last_error}</span>
          </div>
        )}
      </div>

      {/* ZeroTier Section */}
      <h3 style={{ marginTop: '1.5rem', fontSize: '1rem', color: '#666' }}>ZeroTier (VPN)</h3>
      <div style={{ background: '#fbfbfd', padding: '1rem', borderRadius: '8px', border: '1px solid #eee' }}>
        <div className="info-row" style={{ justifyContent: 'space-between', marginBottom: '0.5rem' }}>
          <span className="info-label">Status</span>
          {zt.state === 'Connected' ? (
            <span className="tag connected">Connected</span>
          ) : (
            <span className="tag" style={{ color: 'red', background: '#ffebeb' }}>{zt.state || "Disconnected"}</span>
          )}
        </div>
        <div className="info-row" style={{ justifyContent: 'space-between', marginBottom: '0.5rem' }}>
          <span className="info-label">IP Address</span>
          <span className="info-value code">{zt.ip_address || "-"}</span>
        </div>
        {zt.last_error && (
          <div className="info-row" style={{ justifyContent: 'space-between', color: 'red' }}>
            <span className="info-label" style={{ color: 'red' }}>Error</span>
            <span className="info-value">{zt.last_error}</span>
          </div>
        )}
      </div>
    </div>
  );
};

export default App;
