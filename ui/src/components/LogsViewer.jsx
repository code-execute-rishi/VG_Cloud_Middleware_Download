import React, { useState, useEffect } from 'react';

const LogsViewer = ({ service = "" }) => {
    const [logs, setLogs] = useState("Loading logs...");

    const fetchLogs = () => {
        let url = "/api/logs";
        if (service) url += `?service=${service}`;

        fetch(url)
            .then(res => {
                if (!res.ok) throw new Error("Failed to load");
                return res.json();
            })
            .then(lines => {
                if (Array.isArray(lines)) {
                    setLogs(lines.join("\n"));
                } else {
                    setLogs(lines); // Fallback if text
                }
            })
            .catch(err => setLogs("Error loading logs: " + err.message));
    };

    useEffect(() => {
        fetchLogs();
        const interval = setInterval(fetchLogs, 5000); // Poll every 5s
        return () => clearInterval(interval);
    }, [service]);

    return (
        <div style={{ marginTop: '20px' }}>
            <h3 style={{ fontSize: '1rem', color: '#64748b', textTransform: 'uppercase', marginBottom: '10px' }}>
                {service ? `${service} Logs` : 'System Logs'}
            </h3>
            <div style={{
                background: '#0f172a',
                color: '#22c55e',
                padding: '15px',
                borderRadius: '8px',
                fontFamily: 'source-code-pro, Menlo, Monaco, Consolas, "Courier New", monospace',
                fontSize: '12px',
                lineHeight: '1.4',
                height: '300px',
                overflow: 'auto',
                whiteSpace: 'pre-wrap',
                border: '1px solid #334155'
            }}>
                {logs}
            </div>
        </div>
    );
};

export default LogsViewer;
