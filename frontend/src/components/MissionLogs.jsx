import React, { useRef, useEffect } from 'react';
import { Download, Terminal } from 'lucide-react';

const MissionLogs = ({ logs }) => {
  const scrollRef = useRef(null);

  // Auto-scroll to bottom
  useEffect(() => {
    if (scrollRef.current) {
      scrollRef.current.scrollTop = scrollRef.current.scrollHeight;
    }
  }, [logs]);

  const downloadCSV = () => {
    if (!logs || logs.length === 0) return;

    const headers = ['Timestamp', 'Event'];
    const csvRows = [headers.join(',')];

    logs.forEach(log => {
      // Log format: "[HH:MM:SS AM/PM] Message"
      // Regex to capture timestamp and the rest
      const match = log.match(/^\[(.*?)\]\s*(.*)$/);
      
      let timestamp = '';
      let message = log;

      if (match) {
        timestamp = match[1];
        message = match[2];
      }

      // Escape quotes in message and wrap in quotes
      const escapedMessage = `"${message.replace(/"/g, '""')}"`;
      csvRows.push(`${timestamp},${escapedMessage}`);
    });

    const csvString = csvRows.join('\n');
    const blob = new Blob([csvString], { type: 'text/csv' });
    const url = URL.createObjectURL(blob);
    const link = document.createElement('a');
    
    const dateStr = new Date().toISOString().replace(/[:.]/g, '-');
    link.href = url;
    link.download = `flight_logs_${dateStr}.csv`;
    document.body.appendChild(link);
    link.click();
    document.body.removeChild(link);
  };

  return (
    <div className="h-48 bg-military-900 border-t border-military-700 flex flex-col shrink-0">
      {/* Header */}
      <div className="flex items-center justify-between px-4 py-2 bg-military-800 border-b border-military-700">
        <div className="flex items-center gap-2 text-gray-400">
          <Terminal size={16} />
          <h3 className="text-xs font-bold uppercase tracking-wider">Mission Logs</h3>
        </div>
        <button 
          onClick={downloadCSV}
          className="flex items-center gap-1 text-xs text-gray-400 hover:text-white transition-colors"
          title="Download CSV"
        >
          <Download size={14} />
          <span>CSV</span>
        </button>
      </div>

      {/* Log List */}
      <div 
        ref={scrollRef}
        className="flex-1 overflow-y-auto p-2 font-mono text-xs space-y-1"
      >
        {logs.length === 0 && <div className="text-gray-600 italic px-2">System ready. No logs yet.</div>}
        {logs.map((log, i) => (
          <div key={i} className="text-gray-300 hover:bg-military-800/50 px-2 rounded flex">
            <span className="text-military-accent mr-2">➜</span>
            <span className="break-all">{log}</span>
          </div>
        ))}
      </div>
    </div>
  );
};

export default MissionLogs;
