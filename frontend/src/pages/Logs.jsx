import React from 'react';
import { Terminal } from 'lucide-react';

const Logs = ({ logs }) => {
  return (
    <div className="flex-1 p-4 bg-military-900 overflow-hidden flex flex-col">
      <div className="flex items-center gap-2 mb-4 text-gray-400">
        <Terminal size={20} />
        <h2 className="text-lg font-bold uppercase tracking-wider">System Logs Console</h2>
      </div>
      
      <div className="flex-1 bg-military-800 rounded-lg border border-military-700 p-4 overflow-y-auto font-mono text-sm shadow-inner">
        {logs.length === 0 && <div className="text-gray-600 italic">No logs available...</div>}
        {logs.map((log, i) => (
          <div key={i} className="text-gray-300 border-b border-military-700/50 py-1 hover:bg-military-700/30 px-2 rounded">
            <span className="text-military-accent mr-3">➜</span>
            {log}
          </div>
        ))}
      </div>
    </div>
  );
};

export default Logs;
