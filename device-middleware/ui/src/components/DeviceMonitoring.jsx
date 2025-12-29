import React, { useState, useEffect } from 'react';

// Reusing SystemHealth concepts but expanding it
const DeviceMonitoring = ({ status }) => {
    const [stats, setStats] = useState(null);
    const [tempStats, setTempStats] = useState({ min: 100, max: 0 });

    useEffect(() => {
        const fetchStats = async () => {
            try {
                const res = await fetch("/api/system-stats");
                if (res.ok) {
                    const data = await res.json();
                    setStats(data);

                    // Update Min/Max Temp
                    if (data.cpu_temp) {
                        setTempStats(prev => ({
                            min: Math.min(prev.min, data.cpu_temp),
                            max: Math.max(prev.max, data.cpu_temp)
                        }));
                    }
                }
            } catch (e) { console.error(e); }
        };
        fetchStats();
        const interval = setInterval(fetchStats, 2000);
        return () => clearInterval(interval);
    }, []);

    const formatBytes = (bytes) => {
        if (bytes === 0) return '0 B';
        const k = 1024;
        const sizes = ['B', 'KB', 'MB', 'GB', 'TB'];
        const i = Math.floor(Math.log(bytes) / Math.log(k));
        return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i];
    };

    const getUsageColor = (pct) => {
        if (pct > 90) return '#ef4444'; // Red
        if (pct > 75) return '#f59e0b'; // Orange
        return '#22c55e'; // Green
    };

    // Calculate generic percentage for Network (assuming 1MB/s max for visualization, just for UI feel)
    const getNetPct = (speedKB) => {
        const maxSpeed = 1024; // 1 MB/s reference
        return Math.min((speedKB / maxSpeed) * 100, 100);
    };

    return (
        <div style={{ display: 'flex', flexDirection: 'column', gap: '20px' }}>

            {/* 1. System Resources & Temperature */}
            <div className="card">
                <h2 style={{ fontSize: '1.25rem', marginBottom: '1.5rem', color: '#0f172a' }}>System Resources</h2>
                <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fit, minmax(280px, 1fr))', gap: '20px' }}>

                    {/* CPU Load */}
                    <div style={{ background: '#f8fafc', padding: '1.5rem', borderRadius: '12px', border: '1px solid #e2e8f0', textAlign: 'center' }}>
                        <div style={{ fontSize: '0.85rem', color: '#64748b', fontWeight: '600', textTransform: 'uppercase', marginBottom: '15px' }}>CPU Load</div>
                        <div style={{ position: 'relative', width: '140px', height: '140px', margin: '0 auto' }}>
                            <svg viewBox="0 0 36 36" style={{ width: '100%', height: '100%', transform: 'rotate(-90deg)' }}>
                                <path d="M18 2.0845 a 15.9155 15.9155 0 0 1 0 31.831 a 15.9155 15.9155 0 0 1 0 -31.831" fill="none" stroke="#e2e8f0" strokeWidth="3" />
                                <path d="M18 2.0845 a 15.9155 15.9155 0 0 1 0 31.831 a 15.9155 15.9155 0 0 1 0 -31.831" fill="none" stroke={getUsageColor(stats?.cpu_usage || 0)} strokeWidth="3" strokeDasharray={`${stats?.cpu_usage || 0}, 100`} />
                            </svg>
                            <div style={{ position: 'absolute', top: '50%', left: '50%', transform: 'translate(-50%, -50%)', textAlign: 'center' }}>
                                <div style={{ fontSize: '1.75rem', fontWeight: '700', color: '#0f172a' }}>{stats?.cpu_usage?.toFixed(1) || 0}%</div>
                            </div>
                        </div>
                    </div>

                    {/* RAM Usage */}
                    <div style={{ background: '#f8fafc', padding: '1.5rem', borderRadius: '12px', border: '1px solid #e2e8f0', textAlign: 'center', display: 'flex', flexDirection: 'column', justifyContent: 'center' }}>
                        <div style={{ fontSize: '0.85rem', color: '#64748b', fontWeight: '600', textTransform: 'uppercase', marginBottom: '15px' }}>RAM Usage</div>
                        <div style={{ display: 'flex', flexDirection: 'column', alignItems: 'center', gap: '5px' }}>
                            <div style={{ fontSize: '2rem', fontWeight: '700', color: '#0f172a' }}>
                                {stats ? formatBytes(stats.ram_used) : '-'}
                            </div>
                            <div style={{ fontSize: '0.9rem', color: '#64748b' }}>
                                of {stats ? formatBytes(stats.ram_total) : '-'}
                            </div>
                            {stats && (
                                <div style={{ width: '80%', height: '8px', background: '#e2e8f0', borderRadius: '4px', marginTop: '15px', overflow: 'hidden' }}>
                                    <div style={{ height: '100%', background: getUsageColor((stats.ram_used / stats.ram_total) * 100), width: `${(stats.ram_used / stats.ram_total) * 100}%` }}></div>
                                </div>
                            )}
                        </div>
                    </div>

                    {/* Temperature (NEW Separate Card) */}
                    <div style={{ background: '#f8fafc', padding: '1.5rem', borderRadius: '12px', border: '1px solid #e2e8f0', textAlign: 'center', display: 'flex', flexDirection: 'column', justifyContent: 'center' }}>
                        <div style={{ fontSize: '0.85rem', color: '#64748b', fontWeight: '600', textTransform: 'uppercase', marginBottom: '15px' }}>CPU Temperature</div>
                        <div style={{ marginBottom: '15px' }}>
                            <div style={{ fontSize: '2.5rem', fontWeight: '800', color: '#0f172a' }}>
                                {stats?.cpu_temp ? `${stats.cpu_temp.toFixed(1)}°C` : 'N/A'}
                            </div>
                            <div style={{ fontSize: '0.85rem', color: '#64748b' }}>Current Core Temp</div>
                        </div>
                        {/* Min / Max */}
                        <div style={{ display: 'flex', justifyContent: 'center', gap: '30px', marginTop: '10px', paddingTop: '15px', borderTop: '1px solid #e2e8f0' }}>
                            <div>
                                <div style={{ fontSize: '0.75rem', color: '#64748b', textTransform: 'uppercase' }}>Min</div>
                                <div style={{ fontWeight: '700', color: '#0f172a' }}>{tempStats.min !== 100 ? tempStats.min.toFixed(1) : '-'}°C</div>
                            </div>
                            <div>
                                <div style={{ fontSize: '0.75rem', color: '#64748b', textTransform: 'uppercase' }}>Max</div>
                                <div style={{ fontWeight: '700', color: '#ef4444' }}>{tempStats.max !== 0 ? tempStats.max.toFixed(1) : '-'}°C</div>
                            </div>
                        </div>
                    </div>

                </div>
            </div>

            {/* 2. Network Load (Redesigned with Circular Gauges) */}
            <div className="card">
                <h2 style={{ fontSize: '1.25rem', marginBottom: '1.5rem', color: '#0f172a' }}>Network Traffic</h2>
                <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fit, minmax(280px, 1fr))', gap: '20px' }}>

                    {/* Download */}
                    <div style={{ padding: '1.5rem', background: '#f8fafc', borderRadius: '12px', border: '1px solid #e2e8f0', textAlign: 'center' }}>
                        <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'center', gap: '10px', marginBottom: '15px' }}>
                            <div style={{ background: '#dbeafe', color: '#2563eb', padding: '6px', borderRadius: '6px' }}>
                                <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"><path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4" /><polyline points="7 10 12 15 17 10" /><line x1="12" y1="15" x2="12" y2="3" /></svg>
                            </div>
                            <span style={{ fontWeight: '600', color: '#64748b', textTransform: 'uppercase', fontSize: '0.85rem' }}>Download</span>
                        </div>

                        <div style={{ position: 'relative', width: '140px', height: '140px', margin: '0 auto' }}>
                            <svg viewBox="0 0 36 36" style={{ width: '100%', height: '100%', transform: 'rotate(-90deg)' }}>
                                <path d="M18 2.0845 a 15.9155 15.9155 0 0 1 0 31.831 a 15.9155 15.9155 0 0 1 0 -31.831" fill="none" stroke="#e2e8f0" strokeWidth="3" />
                                <path d="M18 2.0845 a 15.9155 15.9155 0 0 1 0 31.831 a 15.9155 15.9155 0 0 1 0 -31.831" fill="none" stroke="#3b82f6" strokeWidth="3" strokeDasharray={`${getNetPct(stats?.net_rx_speed || 0)}, 100`} />
                            </svg>
                            <div style={{ position: 'absolute', top: '50%', left: '50%', transform: 'translate(-50%, -50%)', textAlign: 'center', width: '100%' }}>
                                <div style={{ fontSize: '1.5rem', fontWeight: '700', color: '#0f172a' }}>{stats?.net_rx_speed ? stats.net_rx_speed.toFixed(1) : '0.0'}</div>
                                <div style={{ fontSize: '0.8rem', color: '#64748b' }}>KB/s</div>
                            </div>
                        </div>
                    </div>

                    {/* Upload */}
                    <div style={{ padding: '1.5rem', background: '#f8fafc', borderRadius: '12px', border: '1px solid #e2e8f0', textAlign: 'center' }}>
                        <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'center', gap: '10px', marginBottom: '15px' }}>
                            <div style={{ background: '#fee2e2', color: '#dc2626', padding: '6px', borderRadius: '6px' }}>
                                <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"><path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4" /><polyline points="17 8 12 3 7 8" /><line x1="12" y1="3" x2="12" y2="15" /></svg>
                            </div>
                            <span style={{ fontWeight: '600', color: '#64748b', textTransform: 'uppercase', fontSize: '0.85rem' }}>Upload</span>
                        </div>

                        <div style={{ position: 'relative', width: '140px', height: '140px', margin: '0 auto' }}>
                            <svg viewBox="0 0 36 36" style={{ width: '100%', height: '100%', transform: 'rotate(-90deg)' }}>
                                <path d="M18 2.0845 a 15.9155 15.9155 0 0 1 0 31.831 a 15.9155 15.9155 0 0 1 0 -31.831" fill="none" stroke="#e2e8f0" strokeWidth="3" />
                                <path d="M18 2.0845 a 15.9155 15.9155 0 0 1 0 31.831 a 15.9155 15.9155 0 0 1 0 -31.831" fill="none" stroke="#ef4444" strokeWidth="3" strokeDasharray={`${getNetPct(stats?.net_tx_speed || 0)}, 100`} />
                            </svg>
                            <div style={{ position: 'absolute', top: '50%', left: '50%', transform: 'translate(-50%, -50%)', textAlign: 'center', width: '100%' }}>
                                <div style={{ fontSize: '1.5rem', fontWeight: '700', color: '#0f172a' }}>{stats?.net_tx_speed ? stats.net_tx_speed.toFixed(1) : '0.0'}</div>
                                <div style={{ fontSize: '0.8rem', color: '#64748b' }}>KB/s</div>
                            </div>
                        </div>
                    </div>

                </div>
            </div>

        </div>
    );
};

export default DeviceMonitoring;
