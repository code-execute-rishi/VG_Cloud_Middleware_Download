import React from 'react';
import { 
  Activity, 
  Battery, 
  Navigation, 
  Wifi, 
  WifiOff, 
  Satellite,
  Zap,
  Gauge,
  Cpu
} from 'lucide-react';

const Header = ({ telemetry }) => {
  // Helper to format battery color
  const getBatteryColor = (level) => {
    if (level > 70) return 'text-green-500';
    if (level > 30) return 'text-yellow-500';
    return 'text-red-500';
  };

  // Helper for GPS Fix Type
  const getGpsStatus = (fixType) => {
    switch(fixType) {
      case 0: return 'NO GPS';
      case 1: return 'NO FIX';
      case 2: return '2D FIX';
      case 3: return '3D FIX';
      case 4: return 'DGPS';
      case 5: return 'RTK FLT';
      case 6: return 'RTK FIX';
      default: return 'UNKNOWN';
    }
  };

  return (
    <header className="h-16 bg-military-800 border-b border-military-700 flex items-center justify-between px-4 shrink-0 z-20 shadow-md">
      {/* Left: Branding & Mode */}
      <div className="flex items-center gap-6">
        <div className="flex items-center gap-3">
          <div className="w-8 h-8 bg-military-accent rounded flex items-center justify-center text-military-900 font-bold">
            GCS
          </div>
          <h1 className="font-bold text-lg tracking-wider text-gray-100 hidden md:block">MISSION CONTROL</h1>
        </div>

        {/* Flight Mode & Arming */}
        <div className="flex flex-col border-l border-military-700 pl-6">
          <div className="flex items-baseline gap-2">
            <span className="font-mono font-bold text-xl text-blue-400">{telemetry.flight_mode}</span>
            <span className="text-gray-500">|</span>
            <span className={`font-bold text-sm tracking-wider ${telemetry.armed ? 'text-red-500 animate-pulse' : 'text-green-500'}`}>
              {telemetry.armed ? 'ARMED' : 'DISARMED'}
            </span>
          </div>
        </div>
      </div>

      {/* Right: Telemetry Grid */}
      <div className="flex items-center gap-4 text-xs md:text-sm">
        
        {/* GPS */}
        <div className="hidden md:flex items-center gap-2 bg-military-900/50 px-3 py-1.5 rounded border border-military-700">
          <Satellite size={16} className={telemetry.gps_fix >= 3 ? 'text-green-500' : 'text-red-500'} />
          <div>
            <div className="text-[10px] text-gray-500 uppercase leading-none">GPS</div>
            <div className="font-mono font-bold">
              {getGpsStatus(telemetry.gps_fix)} <span className="text-gray-400">({telemetry.satellites})</span>
            </div>
          </div>
        </div>

        {/* System Health (Radio/Link) */}
        <div className="hidden lg:flex items-center gap-2 bg-military-900/50 px-3 py-1.5 rounded border border-military-700">
          <Cpu size={16} className="text-gray-400" />
          <div>
            <div className="text-[10px] text-gray-500 uppercase leading-none">Link</div>
            <div className="font-mono font-bold">100%</div>
          </div>
        </div>

        {/* Mission Stats */}
        <div className="hidden xl:flex items-center gap-4 bg-military-900/50 px-4 py-1.5 rounded border border-military-700">
          {/* Distance */}
          <div className="flex flex-col items-center">
            <span className="text-[10px] text-gray-500 uppercase leading-none">Dist</span>
            <span className="font-mono font-bold text-blue-400">{telemetry.distance_travelled?.toFixed(0)}m</span>
          </div>
          <div className="w-px h-6 bg-military-700"></div>
          {/* Bearing */}
          <div className="flex flex-col items-center">
            <span className="text-[10px] text-gray-500 uppercase leading-none">Brng</span>
            <span className="font-mono font-bold text-yellow-400">{telemetry.bearing_to_home?.toFixed(0)}°</span>
          </div>
          <div className="w-px h-6 bg-military-700"></div>
          {/* Time */}
          <div className="flex flex-col items-center">
            <span className="text-[10px] text-gray-500 uppercase leading-none">Time</span>
            <span className="font-mono font-bold text-green-400">{telemetry.flight_time}</span>
          </div>
        </div>

        {/* Battery */}
        <div className="flex items-center gap-2 bg-military-900/50 px-3 py-1.5 rounded border border-military-700">
          <Battery size={16} className={getBatteryColor(telemetry.battery_remaining)} />
          <div>
            <div className="text-[10px] text-gray-500 uppercase leading-none">Batt 1</div>
            <div className="font-mono font-bold whitespace-nowrap">
              {telemetry.battery_remaining}% <span className="text-gray-500">/</span> {telemetry.voltage?.toFixed(1)}V <span className="text-gray-500">/</span> {telemetry.current?.toFixed(1)}A
            </div>
            {/* Est. Flight Time Calculation (Assuming 5000mAh battery for demo if capacity unknown, or just show N/A) */}
            {telemetry.current > 0 && (
               <div className="text-[10px] text-gray-400 font-mono">
                 Est: {((telemetry.battery_remaining / 100 * 5) / telemetry.current * 60).toFixed(0)} min
               </div>
            )}
          </div>
        </div>

        {/* Attitude & Speed */}
        <div className="hidden xl:flex items-center gap-4 bg-military-900/50 px-4 py-1.5 rounded border border-military-700">
          {/* Altitude */}
          <div className="flex flex-col items-center">
            <span className="text-[10px] text-gray-500 uppercase leading-none">Alt (m)</span>
            <span className="font-mono font-bold text-blue-400">{telemetry.relative_alt?.toFixed(1)}</span>
          </div>
          <div className="w-px h-6 bg-military-700"></div>
          {/* Speed */}
          <div className="flex flex-col items-center">
            <span className="text-[10px] text-gray-500 uppercase leading-none">GSpd (m/s)</span>
            <span className="font-mono font-bold text-purple-400">{telemetry.groundspeed?.toFixed(1)}</span>
          </div>
          <div className="w-px h-6 bg-military-700"></div>
          {/* Heading */}
          <div className="flex flex-col items-center">
            <span className="text-[10px] text-gray-500 uppercase leading-none">Hdg</span>
            <div className="flex items-center gap-1">
              <span className="font-mono font-bold text-yellow-400">{telemetry.heading?.toFixed(0)}°</span>
              <Navigation 
                size={14} 
                className="text-yellow-400 transition-transform duration-300" 
                style={{ transform: `rotate(${telemetry.heading}deg)` }} 
              />
            </div>
          </div>
        </div>

        {/* Connection Status */}
        <div className={`flex items-center gap-2 px-3 py-1.5 rounded border ${telemetry.connected ? 'bg-green-900/20 border-green-900/50 text-green-500' : 'bg-red-900/20 border-red-900/50 text-red-500'}`}>
          {telemetry.connected ? <Wifi size={16} /> : <WifiOff size={16} />}
          <span className="font-bold text-xs uppercase hidden sm:block">{telemetry.connected ? 'LINKED' : 'OFFLINE'}</span>
        </div>
      </div>
    </header>
  );
};

export default Header;
