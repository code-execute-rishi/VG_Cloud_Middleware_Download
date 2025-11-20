import React from 'react';
import { BrowserRouter as Router, Routes, Route, NavLink } from 'react-router-dom';
import { useTelemetry } from './hooks/useTelemetry';
import Header from './components/Header';
import MissionLogs from './components/MissionLogs';
import Dashboard from './pages/Dashboard';
import Logs from './pages/Logs';
import { LayoutDashboard, FileText } from 'lucide-react';

function App() {
  const { telemetry, logs } = useTelemetry();

  return (
    <Router>
      <div className="h-screen w-screen bg-military-900 text-gray-200 font-sans overflow-hidden flex flex-col">
        
        {/* 1. Top Telemetry Bar */}
        <Header telemetry={telemetry} />

        {/* 2. Main Content Area */}
        <div className="flex-1 flex min-h-0 relative">
          
          {/* Sidebar Navigation */}
          <nav className="w-16 bg-military-800 border-r border-military-700 flex flex-col items-center py-4 gap-4 z-10">
            <NavLink 
              to="/" 
              className={({ isActive }) => `p-3 rounded-lg transition-colors ${isActive ? 'bg-military-accent text-military-900' : 'text-gray-400 hover:bg-military-700 hover:text-gray-200'}`}
              title="Dashboard"
            >
              <LayoutDashboard size={24} />
            </NavLink>
            <NavLink 
              to="/logs" 
              className={({ isActive }) => `p-3 rounded-lg transition-colors ${isActive ? 'bg-military-accent text-military-900' : 'text-gray-400 hover:bg-military-700 hover:text-gray-200'}`}
              title="System Logs"
            >
              <FileText size={24} />
            </NavLink>
          </nav>

          {/* Page Content */}
          <main className="flex-1 flex flex-col min-h-0 bg-military-900 relative overflow-hidden">
            <Routes>
              <Route path="/" element={<Dashboard telemetry={telemetry} />} />
              <Route path="/logs" element={<Logs logs={logs} />} />
            </Routes>
            <MissionLogs logs={logs} />
          </main>

        </div>
      </div>
    </Router>
  );
}

export default App;
