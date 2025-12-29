import React from 'react';
import { NavLink } from 'react-router-dom';

// Helper to shorten UUIDs
const shortId = (id) => id ? `${id.substring(0, 8)}...` : '';

// Helper for Date Formatting
const formatDate = (timestamp) => {
    if (!timestamp) return 'Unknown';
    return new Date(timestamp * 1000).toLocaleDateString('en-US', {
        year: 'numeric',
        month: 'short',
        day: 'numeric'
    });
};

// Helper for Expiration Duration
const getDaysRemaining = (exp) => {
    if (!exp) return '';
    const now = Date.now() / 1000;
    const diff = exp - now;
    const days = Math.floor(diff / (60 * 60 * 24));
    if (days < 0) return 'Expired';
    if (days === 0) return 'Expires today';
    return `Valid for ${days} days`;
};

const Sidebar = ({ systemInfo, isClaimed, isConnected, userInfo }) => {
    return (
        <div className="sidebar">
            {/* Brand Section */}
            <div className="sidebar-header">
                <div className="brand-logo">
                    <svg width="28" height="28" viewBox="0 0 24 24" fill="none" stroke="#2563eb" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"><circle cx="12" cy="12" r="10" /><path d="M12 2a14.5 14.5 0 0 0 0 20 14.5 14.5 0 0 0 0-20" /><path d="M2 12h20" /></svg>
                </div>
                <div className="brand-name">
                    <h3>Vyom GCS</h3>
                    <span className={`status-badge ${isConnected ? 'online' : 'offline'}`}>
                        {isConnected ? 'ONLINE' : 'OFFLINE'}
                    </span>
                </div>
            </div>

            {/* Navigation Section */}
            <nav className="sidebar-nav">
                <NavLink to="/" className={({ isActive }) => `nav-item ${isActive ? 'active' : ''}`}>
                    <span className="label">Dashboard</span>
                </NavLink>
                <NavLink to="/camera" className={({ isActive }) => `nav-item ${isActive ? 'active' : ''}`}>
                    <span className="label">Settings & Video</span>
                </NavLink>
                <NavLink to="/logs" className={({ isActive }) => `nav-item ${isActive ? 'active' : ''}`}>
                    <span className="label">System Logs</span>
                </NavLink>
                <NavLink to="/flight-controller" className={({ isActive }) => `nav-item ${isActive ? 'active' : ''}`}>
                    <span className="label">Flight Controller</span>
                </NavLink>
            </nav>

            {/* Footer / Device Info */}
            <div className="sidebar-footer">
                {/* User Profile Section */}
                {userInfo && (
                    <div className="user-profile-card">
                        <div className="user-avatar">
                            <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"><path d="M20 21v-2a4 4 0 0 0-4-4H8a4 4 0 0 0-4 4v2" /><circle cx="12" cy="7" r="4" /></svg>
                        </div>
                        <div className="user-details">
                            <span className="user-name" title={userInfo.name}>{userInfo.name || "Unknown User"}</span>
                            <span className="user-email" title={userInfo.email}>{userInfo.email || "No Email"}</span>
                        </div>
                        <div className="login-tooltip">
                            <div>Logged in: {formatDate(userInfo.iat)}</div>
                            <div style={{ opacity: 0.8, fontSize: '0.7rem' }}>
                                {getDaysRemaining(userInfo.exp)} ({formatDate(userInfo.exp)})
                            </div>
                        </div>
                    </div>
                )}

                <div className="device-card">
                    <div className="device-icon">
                        <svg width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="#64748b" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"><rect x="4" y="4" width="16" height="16" rx="2" ry="2" /><rect x="9" y="9" width="6" height="6" /><line x1="9" y1="1" x2="9" y2="4" /><line x1="15" y1="1" x2="15" y2="4" /><line x1="9" y1="20" x2="9" y2="23" /><line x1="15" y1="20" x2="15" y2="23" /><line x1="20" y1="9" x2="23" y2="9" /><line x1="20" y1="14" x2="23" y2="14" /><line x1="1" y1="9" x2="4" y2="9" /><line x1="1" y1="14" x2="4" y2="14" /></svg>
                    </div>
                    <div className="device-info">
                        <span className="device-label">Active Device</span>
                        <span className="device-id" title={systemInfo?.device_id || "Loading..."}>
                            {systemInfo?.device_id ? shortId(systemInfo.device_id) : "..."}
                        </span>
                        <span className={`connection-dot ${isClaimed ? 'active' : ''}`}>
                            {isClaimed ? 'Claimed' : 'Unclaimed'}
                        </span>
                    </div>
                </div>
                <div className="version-info">
                    v2.5.0
                </div>
            </div>
        </div>
    );
};

export default Sidebar;
