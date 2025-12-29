import React, { useState, useEffect } from 'react';

const ZeroTierVPN = () => {
    const [status, setStatus] = useState(null); // Full System Status (contains ZeroTier)
    const [gcsEndpoints, setGcsEndpoints] = useState([]);

    // Poll System Status (for ZT details)
    useEffect(() => {
        const fetchStatus = async () => {
            try {
                const res = await fetch("/api/status");
                if (res.ok) setStatus(await res.json());
            } catch (e) { console.error(e); }
        };
        fetchStatus();
        const interval = setInterval(fetchStatus, 2000);
        return () => clearInterval(interval);
    }, []);

    // --- GCS Logic (Moved from FlightController) ---
    const fetchGCSEndpoints = async () => {
        try {
            const res = await fetch("/api/gcs/endpoints");
            if (res.ok) setGcsEndpoints(await res.json());
        } catch (e) { console.error(e); }
    };

    const toggleGCSEndpoint = async (id, current) => {
        try {
            await fetch(`/api/gcs/endpoints/toggle?id=${id}&enabled=${!current}`);
            fetchGCSEndpoints();
        } catch (e) { console.error(e); }
    };

    const deleteGCSEndpoint = async (id) => {
        try {
            await fetch(`/api/gcs/endpoints/delete?id=${id}`, { method: "POST" });
            fetchGCSEndpoints();
        } catch (e) { console.error(e); }
    };

    useEffect(() => {
        fetchGCSEndpoints();
        const interval = setInterval(fetchGCSEndpoints, 2000);
        return () => clearInterval(interval);
    }, []);

    const zt = status?.zerotier_status || {};
    const peers = Array.isArray(zt.peers) ? zt.peers : [];

    console.log("Rendering ZeroTierVPN. Status:", status, "Peers:", peers, "GCS:", gcsEndpoints);

    if (!status && !gcsEndpoints.length) {
        return <div className="p-4 text-center">Loading Network Status...</div>;
    }

    return (
        <div style={{ display: 'flex', flexDirection: 'column', gap: '20px' }}>

            {/* Header */}
            <div className="card" style={{ display: 'flex', alignItems: 'center', gap: '15px', padding: '1.5rem' }}>
                <div style={{ width: '45px', height: '45px', borderRadius: '10px', background: '#e0f2fe', display: 'flex', alignItems: 'center', justifyContent: 'center', color: '#0284c7' }}>
                    <svg width="26" height="26" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"><circle cx="12" cy="12" r="10" /><line x1="2" y1="12" x2="22" y2="12" /><path d="M12 2a15.3 15.3 0 0 1 4 10 15.3 15.3 0 0 1-4 10 15.3 15.3 0 0 1-4-10 15.3 15.3 0 0 1 4-10z" /></svg>
                </div>
                <div>
                    <h1 style={{ margin: 0, fontSize: '1.25rem', color: '#0f172a' }}>ZeroTier & VPN</h1>
                    <p style={{ margin: '5px 0 0 0', color: '#64748b', fontSize: '0.9rem' }}>Secure Mesh Network Management</p>
                </div>
            </div>

            <div style={{ display: 'grid', gridTemplateColumns: 'minmax(0, 1.2fr) minmax(0, 1fr)', gap: '20px' }}>

                {/* 1. Network Status (ZeroTier) */}
                <div className="card" style={{ display: 'flex', flexDirection: 'column' }}>
                    <h2>Network Status</h2>
                    <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: '20px', marginBottom: '20px' }}>
                        <div>
                            <div style={{ fontSize: '0.75rem', color: '#64748b', textTransform: 'uppercase', marginBottom: '4px' }}>Status</div>
                            <div className={`tag ${zt.state === 'Connected' ? 'connected' : 'disconnected'}`} style={{ display: 'inline-block' }}>
                                {zt.state || 'Unknown'}
                            </div>
                        </div>
                        <div>
                            <div style={{ fontSize: '0.75rem', color: '#64748b', textTransform: 'uppercase', marginBottom: '4px' }}>Managed IP</div>
                            <div style={{ fontWeight: '600', color: '#0f172a', fontFamily: 'monospace', fontSize: '1.1rem' }}>{zt.ip_address || "N/A"}</div>
                        </div>
                        <div style={{ gridColumn: 'span 2' }}>
                            <div style={{ fontSize: '0.75rem', color: '#64748b', textTransform: 'uppercase', marginBottom: '4px' }}>Network ID</div>
                            <div style={{ fontWeight: '600', color: '#0f172a', fontFamily: 'monospace', background: '#f1f5f9', padding: '8px 12px', borderRadius: '6px' }}>
                                {zt.network_id || "Not Configured"}
                            </div>
                        </div>
                    </div>
                </div>

                {/* 2. Ground Control Stations (Moved) */}
                <div className="card">
                    <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', borderBottom: '1px solid #e2e8f0', paddingBottom: '1rem', marginBottom: '1rem' }}>
                        <h2 style={{ border: 'none', padding: 0, margin: 0 }}>Ground Control Stations</h2>
                        <span className="tag" style={{ background: '#f0fdf4', color: '#16a34a' }}>FORWARDING ACTIVE</span>
                    </div>

                    <div style={{ padding: '12px', background: '#f8fafc', borderRadius: '8px', border: '1px solid #e2e8f0', marginBottom: '15px', color: '#64748b', fontSize: '0.85rem', textAlign: 'center' }}>
                        Managed via <strong>Cloud Dashboard</strong>
                    </div>

                    <div style={{ display: 'flex', flexDirection: 'column', gap: '10px' }}>
                        {gcsEndpoints.length === 0 && (
                            <div style={{ padding: '20px', textAlign: 'center', color: '#94a3b8', fontStyle: 'italic', border: '1px dashed #cbd5e1', borderRadius: '8px' }}>
                                No active endpoints.
                            </div>
                        )}
                        {gcsEndpoints.map(ep => (
                            <div key={ep.id} style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', padding: '10px 12px', background: 'white', border: '1px solid #e2e8f0', borderRadius: '8px' }}>
                                <div style={{ display: 'flex', alignItems: 'center', gap: '12px' }}>
                                    <div style={{ width: '8px', height: '8px', borderRadius: '50%', background: ep.enabled ? '#22c55e' : '#cbd5e1' }}></div>
                                    <div>
                                        <div style={{ fontWeight: '600', color: '#0f172a', fontSize: '0.9rem' }}>{ep.name || "Unnamed"}</div>
                                        <div style={{ fontSize: '0.8rem', color: '#64748b', fontFamily: 'monospace' }}>{ep.ip}:{ep.port}</div>
                                    </div>
                                </div>
                                <div style={{ display: 'flex', gap: '8px' }}>
                                    <button onClick={() => toggleGCSEndpoint(ep.id, ep.enabled)} style={{ fontSize: '0.75rem', padding: '4px 8px', borderRadius: '4px', border: '1px solid #cbd5e1', background: 'white' }}>
                                        {ep.enabled ? "Disable" : "Enable"}
                                    </button>
                                    <button onClick={() => deleteGCSEndpoint(ep.id)} style={{ border: 'none', background: 'transparent', color: '#ef4444', cursor: 'pointer', padding: '4px' }}>
                                        <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2"><polyline points="3 6 5 6 21 6"></polyline><path d="M19 6v14a2 2 0 0 1-2 2H7a2 2 0 0 1-2-2V6m3 0V4a2 2 0 0 1 2-2h4a2 2 0 0 1 2 2v2"></path></svg>
                                    </button>
                                </div>
                            </div>
                        ))}
                    </div>
                </div>
            </div>

            {/* 3. Network List (Peers) */}
            <div className="card">
                <h2>Network List (Peers)</h2>
                {peers.length === 0 ? (
                    <div style={{ padding: '30px', textAlign: 'center', color: '#94a3b8' }}>
                        No direct peers connected.
                    </div>
                ) : (
                    <div style={{ overflowX: 'auto' }}>
                        <table style={{ width: '100%', borderCollapse: 'collapse', fontSize: '0.9rem' }}>
                            <thead>
                                <tr style={{ borderBottom: '2px solid #e2e8f0', color: '#64748b', textAlign: 'left' }}>
                                    <th style={{ padding: '12px' }}>Address</th>
                                    <th style={{ padding: '12px' }}>Role</th>
                                    <th style={{ padding: '12px' }}>Latency</th>
                                    <th style={{ padding: '12px' }}>Version</th>
                                </tr>
                            </thead>
                            <tbody>
                                {peers.map((p, i) => (
                                    <tr key={i} style={{ borderBottom: '1px solid #f1f5f9' }}>
                                        <td style={{ padding: '12px', fontFamily: 'monospace', color: '#0f172a' }}>{p.address}</td>
                                        <td style={{ padding: '12px' }}>
                                            <span className="tag" style={{ background: p.role === 'PLANET' ? '#e0e7ff' : '#f1f5f9', color: p.role === 'PLANET' ? '#4338ca' : '#475569' }}>
                                                {p.role}
                                            </span>
                                        </td>
                                        <td style={{ padding: '12px' }}>
                                            <span style={{ color: p.latency < 50 ? '#16a34a' : p.latency < 150 ? '#d97706' : '#dc2626', fontWeight: '600' }}>
                                                {p.latency} ms
                                            </span>
                                        </td>
                                        <td style={{ padding: '12px', color: '#64748b' }}>{p.version}</td>
                                    </tr>
                                ))}
                            </tbody>
                        </table>
                    </div>
                )}
            </div>
        </div>
    );
};

export default ZeroTierVPN;
