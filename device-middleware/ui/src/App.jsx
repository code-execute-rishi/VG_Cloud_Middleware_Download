import { useState, useEffect } from 'react';
import './App.css';

function App() {
  const [data, setData] = useState({ pairing_code: 'loading...', status: 'CONNECTING', resolution: '640x480' });
  const [targetRes, setTargetRes] = useState('640x480');
  const [isChanging, setIsChanging] = useState(false);

  useEffect(() => {
    // Poll Status
    const interval = setInterval(() => {
      fetch('/api/status')
        .then(res => res.json())
        .then(json => {
          // If we are currently changing, don't overwrite resolution immediately
          // wait for it specifically or just show what backend says?
          // Let's show what backend says, but if we just clicked, we might see the old one for a sec.
          // That's fine, the button active state can reflect 'targetRes'.
          setData(json);
        })
        .catch(err => console.error("Poll failed", err));
    }, 1000);

    return () => clearInterval(interval);
  }, []);

  const handleConfigChange = (res) => {
    if (isChanging) return;
    setIsChanging(true);
    setTargetRes(res);

    // Determine bitrate based on resolution (simple logic)
    let bitrate = "500k";
    if (res === "1280x720") bitrate = "2000k";
    if (res === "1920x1080") bitrate = "4000k";

    fetch('/api/config', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ resolution: res, bitrate: bitrate })
    })
      .then(() => {
        // Optimistic update done, wait for poll to confirm
        setTimeout(() => setIsChanging(false), 2000);
      })
      .catch(err => {
        console.error("Config failed", err);
        setIsChanging(false);
      });
  };

  // Status Colors: Connected = Green, Waiting = Amber (but using class text-waiting defined in CSS)
  const statusColor = data.status === 'CONNECTED' ? 'text-green-500' : 'text-waiting';
  const statusBlink = data.status === 'WAITING' ? 'animate-pulse' : '';

  return (
    <div className="hud-container">
      {/* HUD Corners */}
      <div className="corner top-left"></div>
      <div className="corner top-right"></div>
      <div className="corner bottom-left"></div>
      <div className="corner bottom-right"></div>

      {/* Main Content */}
      <div className="hud-content">
        <header className="hud-header">
          <h1>VYOM GARUD <span className="text-xs">SYSTEMS</span></h1>
          <div className={`status-indicator ${statusColor} ${statusBlink}`}>
            ● {data.status}
          </div>
        </header>

        <main>
          <div className="data-block mt-10">
            <label>PAIRING CODE</label>
            <div className="pairing-code glitch" data-text={data.pairing_code}>
              {data.pairing_code}
            </div>
          </div>

          <div className="data-block mt-8">
            <label>CAMERA CONFIG</label>
            <div className="control-group">
              {['640x480', '1280x720', '1920x1080'].map((res) => (
                <button
                  key={res}
                  className={`hud-btn ${targetRes === res ? 'active' : ''}`}
                  onClick={() => handleConfigChange(res)}
                  disabled={isChanging}
                  style={{ opacity: isChanging && targetRes !== res ? 0.5 : 1 }}
                >
                  {res}
                </button>
              ))}
            </div>
            <div className="text-xs text-slate-500 mt-2">
              ACTIVE SOURCE: {data.resolution} {isChanging ? '(UPDATING...)' : ''}
            </div>
          </div>
        </main>

        <footer className="hud-footer">
          <div>LOC: 35.36, 149.16</div>
          <div>ALT: 10m</div>
        </footer>
      </div>
    </div>
  );
}

export default App;
