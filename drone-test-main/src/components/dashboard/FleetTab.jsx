"use client";

import { useState, useEffect } from "react";
import { ChevronDown, RefreshCw, AlertCircle } from "lucide-react";
import { toast } from "sonner";

const statusOptions = ["Airborne", "StandBy", "Maintenance"];

const FleetTab = () => {
  const [drones, setDrones] = useState([]);
  const [openDropdown, setOpenDropdown] = useState(null);
  const [isLoading, setIsLoading] = useState(true);
  const [isRefreshing, setIsRefreshing] = useState(false);
  const [error, setError] = useState(null);

  const fetchDrones = async (isRefresh = false) => {
    try {
      if (isRefresh) {
        setIsRefreshing(true);
      } else {
        setIsLoading(true);
      }
      setError(null);

      const response = await fetch('/api/devices');

      if (!response.ok) {
        throw new Error(`Failed to fetch devices: ${response.status}`);
      }

      const data = await response.json();

      const transformedDrones = data.map(device => ({
        id: device.id,
        name: device.name || "Unnamed Device",
        status: device.status || "StandBy",
        batteryPercentage: device.battery,
        altitude: device.altitude ? `${device.altitude.toFixed(1)} m` : "--",
        location: device.latitude && device.longitude
          ? `${Math.abs(device.latitude).toFixed(4)}°${device.latitude >= 0 ? 'N' : 'S'}, ${Math.abs(device.longitude).toFixed(4)}°${device.longitude >= 0 ? 'E' : 'W'}`
          : "--",
        speed: device.speed ? `${device.speed.toFixed(1)} m/s` : "--",
        heading: device.heading ? `${device.heading.toFixed(0)}°` : "--",
        signal: device.signal_strength ? `${device.signal_strength}%` : "--"
      }));

      setDrones(transformedDrones);
    } catch (err) {
      console.error('Error fetching drones:', err);
      setError(err.message);
    } finally {
      setIsLoading(false);
      setIsRefreshing(false);
    }
  };

  useEffect(() => {
    fetchDrones();
  }, []);

  const handleStatusChange = async (droneId, newStatus) => {
    const previousDrones = [...drones];
    const previousStatus = drones.find(d => d.id === droneId)?.status;

    setDrones(drones.map(drone =>
      drone.id === droneId ? { ...drone, status: newStatus } : drone
    ));
    setOpenDropdown(null);

    try {
      const response = await fetch(`/api/devices/${droneId}`, {
        method: 'PATCH',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify({ status: newStatus })
      });

      if (!response.ok) {
        throw new Error('Failed to update status');
      }

      toast.success(`Status updated to ${newStatus}`);
    } catch (error) {
      console.error('Error updating status:', error);
      setDrones(previousDrones);
      toast.error(`Failed to update status. Reverted to ${previousStatus}`);
    }
  };

  const toggleDropdown = (droneId) => {
    setOpenDropdown(openDropdown === droneId ? null : droneId);
  };

  const handleRefresh = () => {
    fetchDrones(true);
  };

  if (isLoading) {
    return (
      <div className="space-y-6 p-6">
        <div className="flex justify-center items-center h-64">
          <div className="flex flex-col items-center gap-4">
            <RefreshCw className="w-8 h-8 animate-spin text-gray-400" />
            <p className="text-gray-500">Loading fleet data...</p>
          </div>
        </div>
      </div>
    );
  }

  if (error) {
    return (
      <div className="space-y-6 p-6">
        <div className="flex justify-center items-center h-64">
          <div className="flex flex-col items-center gap-4 text-center">
            <AlertCircle className="w-12 h-12 text-red-500" />
            <div>
              <p className="text-red-600 font-semibold">Failed to load fleet data</p>
              <p className="text-gray-500 text-sm mt-1">{error}</p>
            </div>
            <button
              onClick={() => fetchDrones()}
              className="mt-4 px-4 py-2 bg-gray-800 text-white rounded-lg hover:bg-gray-900 transition-colors"
            >
              Try Again
            </button>
          </div>
        </div>
      </div>
    );
  }

  return (
    <div className="space-y-6 p-6">
      <div className="flex justify-between items-start">
        <div>
          <h1 className="hidden md:block text-2xl md:text-3xl font-bold text-gray-900 mb-2">
            Fleet Overview
          </h1>
          <p className="text-gray-600 mb-4 md:mb-8 text-sm md:text-base">
            Live state of every airframe, including health, location, and readiness.
          </p>
        </div>
        <button
          onClick={handleRefresh}
          disabled={isRefreshing}
          className="flex items-center gap-2 px-4 py-2 bg-gray-800 text-white rounded-lg hover:bg-gray-900 transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
        >
          <RefreshCw className={`w-4 h-4 ${isRefreshing ? 'animate-spin' : ''}`} />
          <span className="hidden sm:inline">Refresh</span>
        </button>
      </div>

      {drones.length === 0 ? (
        <div className="rounded-xl border border-gray-200 bg-white shadow-sm p-12 text-center">
          <p className="text-gray-500">No devices found. Claim a device to get started.</p>
        </div>
      ) : (
        <div className="rounded-xl border border-gray-200 bg-white shadow-sm overflow-visible">
          <div className="grid grid-cols-6 text-xs font-semibold uppercase tracking-wide text-gray-500 border-b border-gray-100 px-6 py-3">
            <span>Drone</span>
            <span>Status</span>
            <span>Coordinates</span>
            <span>Speed / Altitude</span>
            <span>Heading</span>
            <span>Battery</span>
          </div>
          <div className="divide-y divide-gray-100 overflow-visible">
            {drones.map((drone) => (
              <div
                key={drone.id}
                className="grid grid-cols-6 items-center px-6 py-4 text-sm text-gray-700"
              >
                <div>
                  <p className="font-semibold text-gray-900">
                    {drone.name.length > 16 ? `${drone.name.substring(0, 16)}...` : drone.name}
                  </p>
                  <p className="text-xs text-gray-500">{drone.id.substring(0, 13)}...</p>
                </div>

                <div className="relative">
                  <button
                    onClick={() => toggleDropdown(drone.id)}
                    className="text-gray-600 flex gap-2 items-center hover:text-gray-900 transition-colors"
                  >
                    {drone.status}
                    <ChevronDown className={`text-gray-400 font-extralight transition-transform ${openDropdown === drone.id ? 'rotate-180' : ''}`} />
                  </button>

                  {openDropdown === drone.id && (
                    <>
                      <div
                        className="fixed inset-0 z-10"
                        onClick={() => setOpenDropdown(null)}
                      />
                      <div className="absolute left-0 mt-2 w-40 bg-white rounded-lg shadow-lg border border-gray-200 z-20 overflow-hidden bottom-auto top-full">
                        {statusOptions.map((status) => (
                          <button
                            key={status}
                            onClick={() => handleStatusChange(drone.id, status)}
                            className={`w-full text-left px-4 py-2.5 text-sm hover:bg-gray-50 transition-colors ${drone.status === status ? 'bg-blue-50 text-blue-600 font-medium' : 'text-gray-700'
                              }`}
                          >
                            {status}
                          </button>
                        ))}
                      </div>
                    </>
                  )}
                </div>

                <p className="text-gray-600">{drone.location}</p>
                <div className="text-gray-600 space-y-1">
                  <p>Speed: <span className="font-medium text-gray-900">{drone.speed}</span></p>
                  <p>Altitude: <span className="font-medium text-gray-900">{drone.altitude}</span></p>
                </div>
                <p className="text-gray-600">{drone.heading}</p>
                <div className="flex items-center gap-2">
                  <div className="w-full h-2 bg-gray-100 rounded-full">
                    <div
                      className={`h-full rounded-full ${drone.batteryPercentage >= 70
                          ? "bg-emerald-500"
                          : drone.batteryPercentage >= 40
                            ? "bg-amber-500"
                            : "bg-rose-500"
                        }`}
                      style={{ width: `${drone.batteryPercentage || 0}%` }}
                    />
                  </div>
                  <span className="text-xs text-gray-500">
                    {drone.batteryPercentage !== null ? `${drone.batteryPercentage}%` : "—"}
                  </span>
                </div>
              </div>
            ))}
          </div>
        </div>
      )}
    </div>
  );
};

export default FleetTab;