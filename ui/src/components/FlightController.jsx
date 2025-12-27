import { useState, useEffect } from 'react';

function FlightController() {
    const [status, setStatus] = useState(null);
    const [serialPorts, setSerialPorts] = useState([]);
    const [config, setConfig] = useState({
        fc_port: "auto",
        fc_baud: 57600
    });

    useEffect(() => {
        const fetchStatus = async () => {
            try {
                const res = await fetch("/api/status");
                const data = await res.json();
                setStatus(data);

                // Initialize Config state from server once if needed
                if (data && data.is_configured && !config.loaded) {
                    // Placeholder for future config sync
                }

            } catch (e) {
                console.error("Poll Error", e);
            }
        };

        const fetchPorts = async () => {
            try {
                const res = await fetch("/api/serial-ports");
                const data = await res.json();
                setSerialPorts(data);
            } catch (e) {
                console.error(e);
            }
        };

        fetchStatus();
        fetchPorts();
        const interval = setInterval(fetchStatus, 500); // 2Hz Poll
        return () => clearInterval(interval);
    }, []);

    const [gcsEndpoints, setGcsEndpoints] = useState([]);
    const [newEndpoint, setNewEndpoint] = useState({ name: "", ip: "", port: 14550 });

    const fetchGCSEndpoints = async () => {
        try {
            const res = await fetch("/api/gcs/endpoints");
            if (res.ok) {
                setGcsEndpoints(await res.json());
            }
        } catch (e) { console.error(e); }
    };

    const addGCSEndpoint = async () => {
        if (!newEndpoint.ip || !newEndpoint.port) return;
        try {
            await fetch("/api/gcs/endpoints", {
                method: "POST",
                headers: { "Content-Type": "application/json" },
                body: JSON.stringify({ ...newEndpoint, enable_telemetry: true })
            });
            setNewEndpoint({ name: "", ip: "", port: 14550 });
            fetchGCSEndpoints();
        } catch (e) { console.error(e); }
    };

    const deleteGCSEndpoint = async (id) => {
        try {
            await fetch(`/api/gcs/endpoints/delete?id=${id}`, { method: "POST" }); // Using POST/GET as defined in main.go? main.go uses DELETE handler on GET path? No, main.go handlers check method?
            // Wait, main.go handlers:
            // handleGCSEndpointDelete -> r.URL.Query().Get("id"). Method check?
            // "mux.HandleFunc" -> func(w,r). main.go didn't check method for delete/toggle explicitly? 
            // It just executes. So GET/POST works. I'll use POST for safety or GET.
            // Let's use GET or POST.
            // Actually I should check main.go implementation.
            // handleGCSEndpointDelete just calls RemoveEndpoint.
            fetchGCSEndpoints();
        } catch (e) { console.error(e); }
    };

    const toggleGCSEndpoint = async (id, current) => {
        try {
            await fetch(`/api/gcs/endpoints/toggle?id=${id}&enabled=${!current}`);
            fetchGCSEndpoints();
        } catch (e) { console.error(e); }
    };

    const updateConfig = async (key, value) => {
        const newConfig = { ...config, [key]: value };
        setConfig(newConfig);
        try {
            // Mock Save
            console.log("Saving Config:", newConfig);
            // In real app replace URL with actual endpoint
            // await fetch("/api/save-config", ...);
        } catch (e) { console.error(e) }
    };

    useEffect(() => {
        const fetchStatus = async () => {
            try {
                const res = await fetch("/api/status");
                const data = await res.json();
                setStatus(data);

                // Initialize Config state from server once if needed
                if (data && data.is_configured && !config.loaded) {
                    // Placeholder for future config sync
                }

            } catch (e) {
                console.error("Poll Error", e);
            }
        };

        const fetchPorts = async () => {
            try {
                const res = await fetch("/api/serial-ports");
                const data = await res.json();
                setSerialPorts(data);
            } catch (e) {
                console.error(e);
            }
        };

        fetchStatus();
        fetchPorts();
        fetchGCSEndpoints(); // Initial Load
        const interval = setInterval(fetchStatus, 500); // 2Hz Poll
        // Poll GCS Endpoints too? Or just on change.
        const gcsInterval = setInterval(fetchGCSEndpoints, 2000);
        return () => { clearInterval(interval); clearInterval(gcsInterval); };
    }, []);

    if (!status) return <div className="container center-content"><div className="spinner"></div></div>;

    // Helper to get value or "N/A"
    const val = (v, suffix = "") => (v !== undefined && v !== null) ? `${v}${suffix}` : "N/A";
    const fval = (v, d = 2) => (v !== undefined && v !== null) ? v.toFixed(d) : "N/A";

    return (
        <div style={{ display: 'flex', flexDirection: 'column', gap: '20px' }}>
            {/* Header / Status Bar */}
            <div className="card" style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', padding: '1.5rem' }}>
                <div style={{ display: 'flex', alignItems: 'center', gap: '15px' }}>
                    <div style={{ width: '40px', height: '40px', borderRadius: '10px', background: '#e0f2fe', display: 'flex', alignItems: 'center', justifyContent: 'center', color: '#0284c7' }}>
                        <svg width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"><circle cx="12" cy="12" r="3" /><path d="M19.4 15a1.65 1.65 0 0 0 .33 1.82l.06.06a2 2 0 0 1 0 2.83 2 2 0 0 1-2.83 0l-.06-.06a1.65 1.65 0 0 0-1.82-.33 1.65 1.65 0 0 0-1 1.51V21a2 2 0 0 1-2 2 2 2 0 0 1-2-2v-.09A1.65 1.65 0 0 0 9 19.4a1.65 1.65 0 0 0-1.82.33l-.06.06a2 2 0 0 1-2.83 0 2 2 0 0 1 0-2.83l.06-.06a1.65 1.65 0 0 0 .33-1.82 1.65 1.65 0 0 0-1.51-1H3a2 2 0 0 1-2-2 2 2 0 0 1 2-2h.09A1.65 1.65 0 0 0 4.6 9a1.65 1.65 0 0 0-.33-1.82l-.06-.06a2 2 0 0 1 0-2.83 2 2 0 0 1 2.83 0l.06.06a1.65 1.65 0 0 0 1.82.33H9a1.65 1.65 0 0 0 1-1.51V3a2 2 0 0 1 2-2 2 2 0 0 1 2 2v.09a1.65 1.65 0 0 0 1 1.51 1.65 1.65 0 0 0 1.82-.33l.06-.06a2 2 0 0 1 2.83 0 2 2 0 0 1 0 2.83l-.06.06a1.65 1.65 0 0 0-.33 1.82V9a1.65 1.65 0 0 0 1.51 1H21a2 2 0 0 1 2 2 2 2 0 0 1-2 2h-.09a1.65 1.65 0 0 0-1.51 1z" /></svg>
                    </div>
                    <div>
                        <h1 style={{ margin: 0, fontSize: '1.25rem', color: '#0f172a' }}>Flight Controller</h1>
                        <p style={{ margin: '5px 0 0 0', color: '#64748b', fontSize: '0.9rem' }}>Advanced Telemetry & Configuration</p>
                    </div>
                </div>
                <div style={{ display: 'flex', gap: '10px' }}>
                    {status.hardware_status?.fc_connected ? (
                        <span className="tag connected">FC Connected</span>
                    ) : (
                        <span className="tag disconnected">No FC Connection</span>
                    )}
                </div>
            </div>

            <div style={{ display: 'grid', gridTemplateColumns: 'minmax(0, 1fr) minmax(0, 2fr)', gap: '20px' }}>

                {/* CONFIGURATION CARD */}
                <div className="card">
                    <h2>Connection Settings</h2>
                    <div className="form-group">
                        <label>Serial Port</label>
                        <select
                            value={config.fc_port}
                            onChange={(e) => updateConfig('fc_port', e.target.value)}
                        >
                            <option value="auto">Auto-Detect (Recommended)</option>
                            {serialPorts.map(p => <option key={p} value={p}>{p}</option>)}
                            <option value="udp:127.0.0.1:14550">UDP Local (SITL)</option>
                        </select>
                    </div>

                    <div className="form-group">
                        <label>Baud Rate</label>
                        <select
                            value={config.fc_baud}
                            onChange={(e) => updateConfig('fc_baud', parseInt(e.target.value))}
                        >
                            <option value={57600}>57600 (Telemetry)</option>
                            <option value={115200}>115200</option>
                            <option value={921600}>921600</option>
                        </select>
                    </div>

                    <div style={{ display: 'flex', gap: '10px', marginTop: '20px' }}>

                        <button className="btn-primary" style={{ flex: 1 }}>
                            Connect
                        </button>
                        <button className="btn-primary" style={{ flex: 1, background: '#ef4444' }}>
                            Disconnect
                        </button>
                    </div>
                </div>

                {/* GROUND CONTROL STATIONS CARD */}
                <div className="card">
                    <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', borderBottom: '1px solid #e2e8f0', paddingBottom: '1rem', marginBottom: '1.5rem' }}>
                        <h2 style={{ border: 'none', padding: 0, margin: 0 }}>Ground Control Stations</h2>
                        <span className="tag" style={{ background: '#f0fdf4', color: '#16a34a' }}>FORWARDING ACTIVE</span>
                    </div>

                    {/* Add New Endpoint Form REMOVED as per Cloud-Master workflow */}
                    <div style={{ padding: '15px', background: '#f8fafc', borderRadius: '8px', border: '1px solid #e2e8f0', marginBottom: '20px', color: '#64748b', fontSize: '0.9rem', textAlign: 'center' }}>
                        Manage Ground Control Stations via the <strong>Cloud Dashboard</strong>.
                        <br />
                        Endpoints added there will automatically appear below.
                    </div>

                    {/* Active Endpoints List */}
                    <div style={{ display: 'flex', flexDirection: 'column', gap: '10px' }}>
                        {gcsEndpoints.length === 0 && (
                            <div style={{ padding: '20px', textAlign: 'center', color: '#94a3b8', fontStyle: 'italic', border: '1px dashed #cbd5e1', borderRadius: '8px' }}>
                                No active forwarding endpoints.
                            </div>
                        )}
                        {gcsEndpoints.map(ep => (
                            <div key={ep.id} style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', padding: '12px 15px', background: 'white', border: '1px solid #e2e8f0', borderRadius: '8px', boxShadow: '0 1px 2px rgba(0,0,0,0.05)' }}>
                                <div style={{ display: 'flex', alignItems: 'center', gap: '15px' }}>
                                    <div style={{ width: '10px', height: '10px', borderRadius: '50%', background: ep.enabled ? '#22c55e' : '#cbd5e1' }}></div>
                                    <div>
                                        <div style={{ fontWeight: '600', color: '#0f172a' }}>{ep.name || "Unnamed Endpoint"}</div>
                                        <div style={{ fontSize: '0.85rem', color: '#64748b', fontFamily: 'monospace' }}>{ep.ip}:{ep.port}</div>
                                    </div>
                                </div>
                                <div style={{ display: 'flex', alignItems: 'center', gap: '10px' }}>
                                    <button
                                        onClick={() => toggleGCSEndpoint(ep.id, ep.enabled)}
                                        style={{ padding: '6px 12px', borderRadius: '6px', border: '1px solid #cbd5e1', background: 'white', color: '#475569', cursor: 'pointer', fontSize: '0.85rem' }}
                                    >
                                        {ep.enabled ? "Disable" : "Enable"}
                                    </button>
                                    <button
                                        onClick={() => deleteGCSEndpoint(ep.id)}
                                        style={{ padding: '6px', borderRadius: '6px', border: 'none', background: 'transparent', color: '#ef4444', cursor: 'pointer' }}
                                        title="Remove Endpoint"
                                    >
                                        <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"><polyline points="3 6 5 6 21 6"></polyline><path d="M19 6v14a2 2 0 0 1-2 2H7a2 2 0 0 1-2-2V6m3 0V4a2 2 0 0 1 2-2h4a2 2 0 0 1 2 2v2"></path></svg>
                                    </button>
                                </div>
                            </div>
                        ))}
                    </div>
                </div>

                {/* TELEMETRY CARD */}
                <div className="card">
                    <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', borderBottom: '1px solid #e2e8f0', paddingBottom: '1rem', marginBottom: '1.5rem' }}>
                        <h2 style={{ border: 'none', padding: 0, margin: 0 }}>Telemetry Inspector</h2>
                        <span className="tag" style={{ background: '#f1f5f9', color: '#64748b' }}>LIVE DATA</span>
                    </div>

                    <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fit, minmax(180px, 1fr))', gap: '20px' }}>
                        {/* GPS */}
                        <div style={{ background: '#f8fafc', padding: '1rem', borderRadius: '8px', border: '1px solid #e2e8f0' }}>
                            <div style={{ fontSize: '0.75rem', color: '#64748b', textTransform: 'uppercase', fontWeight: '600', marginBottom: '0.5rem' }}>GPS Satellites</div>
                            <div style={{ fontSize: '1.5rem', fontWeight: '700', color: '#0f172a' }}>{val(status.telemetry?.gps?.satellites_visible)}</div>
                        </div>
                        <div style={{ background: '#f8fafc', padding: '1rem', borderRadius: '8px', border: '1px solid #e2e8f0' }}>
                            <div style={{ fontSize: '0.75rem', color: '#64748b', textTransform: 'uppercase', fontWeight: '600', marginBottom: '0.5rem' }}>GPS Fix</div>
                            <div style={{ fontSize: '1.25rem', fontWeight: '700', color: status.telemetry?.gps?.fix_type >= 3 ? '#16a34a' : '#ef4444' }}>
                                {status.telemetry?.gps?.fix_type >= 3 ? "3D FIX" : "NO FIX"}
                            </div>
                        </div>

                        {/* Battery */}
                        <div style={{ background: '#f8fafc', padding: '1rem', borderRadius: '8px', border: '1px solid #e2e8f0' }}>
                            <div style={{ fontSize: '0.75rem', color: '#64748b', textTransform: 'uppercase', fontWeight: '600', marginBottom: '0.5rem' }}>Battery</div>
                            <div style={{ display: 'flex', alignItems: 'baseline', gap: '5px' }}>
                                <span style={{ fontSize: '1.5rem', fontWeight: '700', color: '#d97706' }}>{fval(status.telemetry?.battery?.voltage)}</span>
                                <span style={{ fontSize: '0.9rem', color: '#64748b' }}>V</span>
                            </div>
                        </div>
                        <div style={{ background: '#f8fafc', padding: '1rem', borderRadius: '8px', border: '1px solid #e2e8f0' }}>
                            <div style={{ fontSize: '0.75rem', color: '#64748b', textTransform: 'uppercase', fontWeight: '600', marginBottom: '0.5rem' }}>Capacity</div>
                            <div style={{ display: 'flex', alignItems: 'baseline', gap: '5px' }}>
                                <span style={{ fontSize: '1.5rem', fontWeight: '700', color: '#d97706' }}>{val(status.telemetry?.battery?.battery_remaining)}</span>
                                <span style={{ fontSize: '0.9rem', color: '#64748b' }}>%</span>
                            </div>
                        </div>

                        {/* HUD */}
                        <div style={{ background: '#f8fafc', padding: '1rem', borderRadius: '8px', border: '1px solid #e2e8f0' }}>
                            <div style={{ fontSize: '0.75rem', color: '#64748b', textTransform: 'uppercase', fontWeight: '600', marginBottom: '0.5rem' }}>Heading</div>
                            <div style={{ fontSize: '1.5rem', fontWeight: '700', color: '#6366f1' }}>{val(status.telemetry?.hud?.heading)}°</div>
                        </div>
                        <div style={{ background: '#f8fafc', padding: '1rem', borderRadius: '8px', border: '1px solid #e2e8f0' }}>
                            <div style={{ fontSize: '0.75rem', color: '#64748b', textTransform: 'uppercase', fontWeight: '600', marginBottom: '0.5rem' }}>Altitude</div>
                            <div style={{ fontSize: '1.5rem', fontWeight: '700', color: '#6366f1' }}>{fval(status.telemetry?.hud?.alt, 1)} m</div>
                        </div>
                    </div>

                    {/* Flight Mode Banner */}
                    <div style={{ marginTop: '20px', padding: '20px', background: '#f8fafc', borderRadius: '12px', border: '1px solid #e2e8f0', display: 'flex', alignItems: 'center', justifyContent: 'center', flexDirection: 'column' }}>
                        <span style={{ fontSize: '0.85rem', color: '#64748b', textTransform: 'uppercase', letterSpacing: '1px', marginBottom: '5px' }}>Current Flight Mode</span>
                        <div style={{ fontSize: '2rem', fontWeight: '800', color: '#0f172a', letterSpacing: '0.5px' }}>
                            {status.telemetry?.system?.mode || "UNKNOWN"}
                        </div>
                    </div>
                </div>

            </div>
        </div>
    );
}

export default FlightController;
