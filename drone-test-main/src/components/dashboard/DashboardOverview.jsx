"use client";

import { useState, useEffect } from "react";
import { useRouter } from "next/navigation";
import { motion } from "framer-motion";
import { Plane, Video, Terminal, Monitor, Plus } from "lucide-react";
import { toast } from "sonner";
import { VideoStreamModal } from '@/components/livekit/VideoStreamModal';

const fleetStats = [
  {
    label: "Active Drones",
    value: "4",
    change: "+1 today",
    badge: "Alpha, Bravo, Delta"
  },
  {
    label: "Standby",
    value: "1",
    change: "Charlie ready",
    badge: "Hangar 2"
  },
  {
    label: "In Maintenance",
    value: "1",
    change: "Echo (battery)",
    badge: "ETA 4h"
  },
  {
    label: "Missions Today",
    value: "7",
    change: "+32% vs avg",
    badge: "3 BVLOS"
  },
];

const statusColors = {
  Airborne: "text-green-600 bg-green-50",
  "Airborne (Armed)": "text-green-600 bg-green-50",
  Standby: "text-amber-600 bg-amber-50",
  Offline: "text-red-600 bg-red-50",
  Maintenance: "text-gray-500 bg-gray-100",
};

const DashboardOverview = ({ user }) => {
  const router = useRouter();
  const [devices, setDevices] = useState([]);
  const [isLoadingDevices, setIsLoadingDevices] = useState(true);
  const [sshModalOpen, setSSHModalOpen] = useState(false);
  const [sshDroneId, setSSHDroneId] = useState(null);
  const [sshDroneName, setSSHDroneName] = useState(null);
  const [sshCommand, setSSHCommand] = useState("");
  const [sshLoading, setSSHLoading] = useState(false);
  const [videoModalOpen, setVideoModalOpen] = useState(false);
  const [videoDroneId, setVideoDroneId] = useState(null);
  const [videoDroneName, setVideoDroneName] = useState(null);

  const fetchDevices = async () => {
    try {
      setIsLoadingDevices(true);
      const response = await fetch('/api/devices');
      if (response.ok) {
        const data = await response.json();

        // Transform data to calculate real-time status (matching FleetTab logic)
        const transformedData = (data || []).map(device => {
          let displayStatus = device.status || "Standby";

          // Use real-time telemetry if available
          if (device.flight_mode) {
            displayStatus = device.flight_mode;
            if (device.armed) {
              // Use a simpler status for the small card, or keep full detail
              // "Airborne (Armed)" might be too long for the badge, but consistency is key.
              displayStatus += " (Armed)";
            }
          }

          return {
            ...device,
            status: displayStatus
          };
        });

        setDevices(transformedData);
      }
    } catch (error) {
      console.error('Error fetching devices:', error);
      toast.error('Failed to load devices');
    } finally {
      setIsLoadingDevices(false);
    }
  };

  useEffect(() => {
    fetchDevices();
  }, []);

  const handleSSHAccess = async (droneId, droneName) => {
    setSSHDroneId(droneId);
    setSSHDroneName(droneName);
    setSSHModalOpen(true);
    setSSHLoading(true);
    setSSHCommand("");

    try {
      const response = await fetch(`/api/devices/${droneId}/zerotier`);
      if (response.ok) {
        const data = await response.json();
        if (data.zerotier_ip) {
          setSSHCommand(`ssh drone@${data.zerotier_ip}`);
        } else {
          setSSHCommand("Device not connected to ZeroTier");
        }
      } else {
        setSSHCommand("Failed to fetch SSH info");
      }
    } catch (error) {
      console.error('Error fetching SSH info:', error);
      setSSHCommand("Error loading SSH info");
    } finally {
      setSSHLoading(false);
    }
  };

  const handleControlPanel = () => {
    router.push('/drone');
  };

  const handleLiveStream = (droneId, droneName) => {
    setVideoDroneId(droneId);
    setVideoDroneName(droneName);
    setVideoModalOpen(true);
  };

  const copyToClipboard = (text) => {
    navigator.clipboard.writeText(text);
    toast.success('Copied to clipboard');
  };

  if (isLoadingDevices) {
    return (
      <div className="space-y-6">
        <div>
          <h1 className="hidden md:block text-2xl md:text-3xl font-bold text-gray-900 mb-2">Dashboard</h1>
          <p className="text-gray-600 text-sm md:text-base">
            Loading your fleet...
          </p>
        </div>
        <div className="rounded-lg bg-white border border-gray-200 shadow-sm p-12 text-center">
          <div className="w-12 h-12 border-4 border-gray-200 border-t-gray-900 rounded-full animate-spin mx-auto mb-4" />
          <p className="text-gray-500">Loading devices...</p>
        </div>
      </div>
    );
  }

  if (devices.length === 0) {
    return (
      <div className="space-y-6">
        <div>
          <h1 className="hidden md:block text-2xl md:text-3xl font-bold text-gray-900 mb-2">Dashboard</h1>
          <p className="text-gray-600 text-sm md:text-base">
            Welcome back, {user.fullName || user.firstName || "Pilot"}
          </p>
        </div>
        <div className="rounded-lg bg-white border border-gray-200 shadow-sm p-12 text-center">
          <Plane className="w-24 h-24 text-gray-300 mx-auto mb-6" />
          <h3 className="text-xl font-semibold text-gray-900 mb-2">No Drones Yet</h3>
          <p className="text-gray-500 mb-6 max-w-md mx-auto">
            Get started by adding your first drone to the fleet. You'll need the device pairing code to begin.
          </p>
          <button
            onClick={() => router.push('/dashboard?tab=access')}
            className="inline-flex items-center gap-2 px-6 py-3 bg-gray-900 text-white rounded-lg font-semibold hover:bg-gray-800 transition"
          >
            <Plus className="w-5 h-5" />
            Add Your First Drone
          </button>
        </div>
      </div>
    );
  }

  return (
    <div className="space-y-6">
      <div>
        <h1 className="hidden md:block text-2xl md:text-3xl font-bold text-gray-900 mb-2">Dashboard</h1>
        <p className="text-gray-600 text-sm md:text-base">
          Welcome back, {user.fullName || user.firstName || "Pilot"} — fleet telemetry synced 8s ago.
        </p>
      </div>

      <div className="grid grid-cols-1 sm:grid-cols-2 xl:grid-cols-4 gap-3 md:gap-4">
        {fleetStats.map((item, index) => (
          <motion.div
            key={item.label}
            initial={{ opacity: 0, y: 20 }}
            animate={{ opacity: 1, y: 0 }}
            transition={{ duration: 0.3, delay: index * 0.1 }}
            className="rounded-lg bg-white border border-gray-200 shadow-sm p-4 flex flex-col gap-2"
          >
            <div className="text-xs uppercase tracking-wide text-gray-500">{item.label}</div>
            <div className="text-2xl font-semibold text-gray-900">{item.value}</div>
            <div className="text-sm text-emerald-600">{item.change}</div>
            <div className="text-xs text-gray-500">{item.badge}</div>
          </motion.div>
        ))}
      </div>

      <div className="grid grid-cols-1 md:grid-cols-2 xl:grid-cols-3 gap-6">
        {devices.map((drone, index) => (
          <motion.div
            key={drone.id}
            initial={{ opacity: 0, y: 20 }}
            animate={{ opacity: 1, y: 0 }}
            transition={{ duration: 0.3, delay: index * 0.1 }}
            className="rounded-lg bg-white border border-gray-200 shadow-sm overflow-hidden aspect-square flex flex-col"
          >
            <div className="relative bg-gradient-to-br from-gray-50 to-gray-100 flex-1 flex flex-col items-center justify-center p-8">
              {drone.status === "Airborne" && (
                <div className="absolute top-3 right-3 flex items-center gap-1.5 bg-red-600 text-white px-2.5 py-1 rounded-full text-xs font-semibold shadow-sm">
                  <div className="w-2 h-2 bg-white rounded-full animate-pulse" />
                  LIVE
                </div>
              )}
              <Plane className="h-28 w-28 md:h-36 md:w-36 text-gray-700 mb-4" />
              <div className="text-center">
                <div className="flex items-center justify-center gap-2 mb-1">
                  <h3 className="text-xl font-bold text-gray-900">{drone.name}</h3>
                  <span className={`text-xs font-semibold px-2.5 py-1 rounded-full ${statusColors[drone.status] || statusColors.Standby}`}>
                    {drone.status || "Standby"}
                  </span>
                </div>
                <p className="text-xs text-gray-500 font-medium">{drone.id.substring(0, 13)}...</p>
              </div>
            </div>

            <div className="bg-gray-50 border-t border-gray-200 p-3 flex-shrink-0">
              <div className="flex gap-2">
                <button
                  type="button"
                  onClick={handleControlPanel}
                  disabled={drone.status !== "Airborne"}
                  className="flex-1 flex items-center justify-center gap-1.5 px-2 py-2 text-xs font-semibold rounded-md border border-gray-300 bg-white text-gray-700 hover:bg-gray-50 hover:border-gray-400 disabled:opacity-50 disabled:cursor-not-allowed transition-all duration-200"
                >
                  <Monitor className="h-3.5 w-3.5" />
                  Control
                </button>
                <button
                  type="button"
                  onClick={() => handleLiveStream(drone.id, drone.name)}
                  disabled={drone.status !== "Airborne"}
                  className="flex-1 flex items-center justify-center gap-1.5 px-2 py-2 text-xs font-semibold rounded-md bg-gray-900 text-white hover:bg-gray-800 disabled:opacity-50 disabled:cursor-not-allowed transition-all duration-200 shadow-sm"
                >
                  <Video className="h-3.5 w-3.5" />
                  Stream
                </button>
                <button
                  type="button"
                  onClick={() => handleSSHAccess(drone.id, drone.name)}
                  className="flex-1 flex items-center justify-center gap-1.5 px-2 py-2 text-xs font-semibold rounded-md border border-gray-300 bg-white text-gray-700 hover:bg-gray-50 hover:border-gray-400 transition-all duration-200"
                >
                  <Terminal className="h-3.5 w-3.5" />
                  SSH
                </button>
              </div>
            </div>
          </motion.div>
        ))}
      </div>

      {sshModalOpen && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/40 px-4">
          <div className="w-full max-w-md rounded-2xl bg-white p-6 shadow-xl">
            <div className="mb-4">
              <h3 className="text-lg font-semibold text-gray-900">
                SSH Access - {sshDroneName}
              </h3>
              <p className="text-sm text-gray-500">
                Connect to your drone via SSH
              </p>
            </div>
            <div className="space-y-4">
              {sshLoading ? (
                <div className="flex items-center justify-center py-8">
                  <div className="w-8 h-8 border-4 border-gray-200 border-t-gray-900 rounded-full animate-spin" />
                </div>
              ) : (
                <>
                  <div className="bg-gray-900 text-green-400 p-4 rounded-lg font-mono text-sm">
                    {sshCommand}
                  </div>
                </>
              )}
            </div>
            <div className="flex items-center justify-end gap-2 mt-6">
              <button
                type="button"
                onClick={() => setSSHModalOpen(false)}
                className="rounded-lg border border-gray-200 px-4 py-2 text-sm font-semibold text-gray-700 hover:bg-gray-50"
              >
                Close
              </button>
              {!sshLoading && sshCommand && !sshCommand.includes("Failed") && !sshCommand.includes("Error") && (
                <button
                  type="button"
                  onClick={() => copyToClipboard(sshCommand)}
                  className="rounded-lg bg-gray-900 px-4 py-2 text-sm font-semibold text-white hover:bg-gray-800"
                >
                  Copy Command
                </button>
              )}
            </div>
          </div>
        </div>
      )}

      <VideoStreamModal
        droneId={videoDroneId}
        droneName={videoDroneName}
        isOpen={videoModalOpen}
        onClose={() => setVideoModalOpen(false)}
      />
    </div>
  );
};

export default DashboardOverview;