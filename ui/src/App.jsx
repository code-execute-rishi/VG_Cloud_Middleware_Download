import React, { useState, useEffect } from "react";
import "./App.css";

function App() {
  const [activeTab, setActiveTab] = useState("device");
  const [rebooting, setRebooting] = useState(false);
  const [systemInfo, setSystemInfo] = useState(null);

  const [formData, setFormData] = useState({
    ssid: "",
    password: "",
    resolution: "640x480",
  });

  useEffect(() => {
    const fetchInfo = async () => {
      try {
        const res = await fetch("/api/system-info");
        const data = await res.json();
        setSystemInfo(data);
      } catch (err) {
        console.error("Failed to fetch info", err);
      }
    };
    fetchInfo();
  }, []);

  const handleInputChange = (e) => {
    const { name, value } = e.target;
    setFormData((prev) => ({
      ...prev,
      [name]: value,
    }));
  };

  const handleBindDevice = async () => {
    try {
      setRebooting(true);
      const res = await fetch("/api/save-config", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(formData),
      });

      if (!res.ok) throw new Error("Failed to save");
    } catch (err) {
      alert("Error saving configuration: " + err.message);
      setRebooting(false);
    }
  };

  if (rebooting) {
    return (
      <div className="container center-content">
        <div className="card reboot-card">
          <div className="spinner"></div>
          <h2>Rebooting...</h2>
          <p>Your device is restarting to apply changes.</p>
          <p className="sub-text">Please reconnect to the new network.</p>
        </div>
      </div>
    );
  }

  return (
    <div className="app-container">
      {/* Header */}
      <header className="top-bar">
        <div className="logo-section">
          <div className="logo-icon">📡</div>
          <div className="logo-text">
            <h1>Vyom Device Setup</h1>
            <p className="subtitle">Bind this device to your cloud account</p>
          </div>
        </div>
      </header>

      {/* Tabs */}
      <nav className="tabs">
        <button
          className={`tab-btn ${activeTab === "device" ? "active" : ""}`}
          onClick={() => setActiveTab("device")}
        >
          <span className="icon">📱</span> Device
        </button>
        <button
          className={`tab-btn ${activeTab === "wifi" ? "active" : ""}`}
          onClick={() => setActiveTab("wifi")}
        >
          <span className="icon">📶</span> WiFi
        </button>
        <button
          className={`tab-btn ${activeTab === "net" ? "active" : ""}`}
          onClick={() => setActiveTab("net")}
        >
          <span className="icon">🌐</span> Net
        </button>
      </nav>

      {/* Content */}
      <main className="main-content">

        {activeTab === "device" && (
          <div className="card bind-card">
            <div className="icon-large">↪️</div>
            <h2>Bind Device to Your Account</h2>
            <p>
              To start using your Vyom device, you need to bind it to your account.
              This will enable all features and remote management.
            </p>

            <div className="info-row">
              <span className="info-label">Device ID:</span>
              <span className="info-value">{systemInfo?.device_id || "Loading..."}</span>
            </div>
            <div className="info-row">
              <span className="info-label">Pairing Code:</span>
              <span className="info-value code">{systemInfo?.pairing_code || "---"}</span>
            </div>

            <div className="status-badge">
              Requires internet connection
            </div>

            <button className="btn-primary" onClick={handleBindDevice}>
              <span className="btn-icon">↪️</span> Bind Device Now
            </button>

            <div className="help-link">Having trouble binding?</div>
            <div className="destructive-link">🗑️ Clear Device Credentials</div>
          </div>
        )}

        {activeTab === "wifi" && (
          <div className="card form-card">
            <div className="card-header">
              <h3>WiFi Configuration</h3>
              <button className="btn-small">⚙️ Configure</button>
            </div>
            <p className="card-desc">Configure local network settings for internet access.</p>

            <div className="form-group">
              <label>SSID (Network Name)</label>
              <input
                type="text"
                name="ssid"
                value={formData.ssid}
                onChange={handleInputChange}
                placeholder="Enter WiFi Name"
              />
            </div>

            <div className="form-group">
              <label>Password</label>
              <input
                type="password"
                name="password"
                value={formData.password}
                onChange={handleInputChange}
                placeholder="Enter WiFi Password"
              />
            </div>

            <div className="form-group">
              <label>Camera Resolution</label>
              <select name="resolution" value={formData.resolution} onChange={handleInputChange}>
                <option value="640x480">480p</option>
                <option value="1280x720">720p</option>
                <option value="1920x1080">1080p</option>
              </select>
            </div>
          </div>
        )}

        {activeTab === "net" && (
          <div className="card list-card">
            <div className="card-header">
              <h3>Network Interfaces</h3>
              <span className="badge">3 devices</span>
            </div>

            <div className="list-item">
              <div className="item-icon">💻</div>
              <div className="item-details">
                <span className="item-title">lo</span>
                <span className="item-sub">127.0.0.1</span>
              </div>
              <span className="tag connected">Connected</span>
            </div>

            <div className="list-item">
              <div className="item-icon">🔌</div>
              <div className="item-details">
                <span className="item-title">eth0</span>
                <span className="item-sub">192.168.1.5</span>
              </div>
              <span className="tag connected">Connected</span>
            </div>

            <div className="list-item">
              <div className="item-icon">📡</div>
              <div className="item-details">
                <span className="item-title">wlan0</span>
                <span className="item-sub">Scanning...</span>
              </div>
              <span className="tag">Disconnected</span>
            </div>
          </div>
        )}

      </main>
    </div>
  );
}

export default App;
