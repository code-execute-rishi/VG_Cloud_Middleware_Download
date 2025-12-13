import React, { useState, useEffect } from "react";
import "./App.css";

function App() {
  const [view, setView] = useState("dashboard"); // 'dashboard', 'camera'
  const [status, setStatus] = useState(null);
  const [systemInfo, setSystemInfo] = useState(null);
  const [loading, setLoading] = useState(false);

  // Form State
  const [formData, setFormData] = useState({
    ssid: "",
    password: "",
    resolution: "",
  });

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
              return { ...prev, resolution: data.camera_config.resolution };
            }
            return prev;
          });
        }
      } catch (e) {
        console.error("Poll failed", e);
      }
    };

    // Fetch System Info once
    fetch("/api/system-info").then(r => r.json()).then(setSystemInfo);

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
        body: JSON.stringify(formData),
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
        body: JSON.stringify({ resolution: formData.resolution }),
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
            <h2>Bind Device</h2>
            <p>Pairing Code: <span className="code">{systemInfo.pairing_code}</span></p>

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
              {loading ? "Saving..." : "Save WiFi Settings"}
            </button>
            <p className="help-link" style={{ marginTop: '15px' }}>
              ℹ️ To finish setup, enter code <b>{systemInfo.pairing_code}</b> in your Cloud Dashboard.
            </p>
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
            Camera & Settings
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

        {view === 'camera' && (
          <div className="card">
            <h3>Local Camera Stream</h3>
            <div className="stream-container" style={{ marginBottom: '20px', background: '#000', borderRadius: '8px', overflow: 'hidden', minHeight: '300px' }}>
              <img
                src="/api/stream"
                alt="Local Stream"
                style={{ width: '100%', display: 'block' }}
              />
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
            <button className="btn-primary" onClick={handleUpdateConfig} disabled={loading}>
              {loading ? "Updating..." : "Update Settings"}
            </button>
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

export default App;
