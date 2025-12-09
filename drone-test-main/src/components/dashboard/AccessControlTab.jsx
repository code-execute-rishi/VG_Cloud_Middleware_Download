"use client";

import { useState, useEffect } from "react";
import { motion } from "framer-motion";
import { Plus, RefreshCcw, Shield, ChevronDown, ChevronRight, Trash2, AlertTriangle } from "lucide-react";
import { toast } from "sonner";

const AccessControlTab = () => {
  const [isAddDroneOpen, setIsAddDroneOpen] = useState(false);
  const [isShareOpen, setIsShareOpen] = useState(false);
  const [isDeleteDeviceOpen, setIsDeleteDeviceOpen] = useState(false);
  const [isDeleteCollaboratorOpen, setIsDeleteCollaboratorOpen] = useState(false);
  const [deviceToDelete, setDeviceToDelete] = useState(null);
  const [collaboratorToDelete, setCollaboratorToDelete] = useState(null);
  const [deleteConfirmName, setDeleteConfirmName] = useState("");
  const [isDeleting, setIsDeleting] = useState(false);
  const [pairingCode, setPairingCode] = useState("");
  const [droneName, setDroneName] = useState("");
  const [isSubmitting, setIsSubmitting] = useState(false);
  const [isSharing, setIsSharing] = useState(false);
  const [sharePilotEmail, setSharePilotEmail] = useState("");
  const [shareDroneId, setShareDroneId] = useState("");
  const [expandedDrones, setExpandedDrones] = useState({});
  const [devices, setDevices] = useState([]);
  const [isLoadingDevices, setIsLoadingDevices] = useState(true);
  const [isRefreshing, setIsRefreshing] = useState(false);
  const [deletingCollaborator, setDeletingCollaborator] = useState(null);

  const fetchDevices = async (isRefresh = false) => {
    try {
      if (isRefresh) {
        setIsRefreshing(true);
      } else {
        setIsLoadingDevices(true);
      }

      const response = await fetch('/api/devices');
      if (response.ok) {
        const data = await response.json();
        setDevices(data || []);
        if (data && data.length > 0 && !shareDroneId) {
          setShareDroneId(data[0].id);
        }
      }
    } catch (error) {
      console.error('Error fetching devices:', error);
      toast.error('Failed to load devices');
    } finally {
      setIsLoadingDevices(false);
      setIsRefreshing(false);
    }
  };

  useEffect(() => {
    fetchDevices();
  }, []);

  const toggleDrone = (droneId) => {
    setExpandedDrones(prev => ({
      ...prev,
      [droneId]: !prev[droneId]
    }));
  };

  const handleAddDrone = async (event) => {
    event.preventDefault();
    setIsSubmitting(true);

    try {
      const response = await fetch("/api/devices/claim", {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
        },
        body: JSON.stringify({
          pairing_code: parseInt(pairingCode, 10),
          name: droneName
        })
      });

      const data = await response.json();

      if (response.ok) {
        toast.success(data.message || "Device claimed successfully!");
        setIsAddDroneOpen(false);
        setPairingCode("");
        setDroneName("");
        fetchDevices(true);
      } else {
        toast.error(data.error || "Failed to claim device");
      }
    } catch (error) {
      console.error("Error claiming device:", error);
      toast.error("Failed to claim device. Please try again.");
    } finally {
      setIsSubmitting(false);
    }
  };

  const handleShareDrone = async (event) => {
    event.preventDefault();
    setIsSharing(true);

    try {
      const response = await fetch(`/api/devices/${shareDroneId}/collaborators`, {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
        },
        body: JSON.stringify({
          email: sharePilotEmail
        })
      });

      const data = await response.json();

      if (response.ok) {
        toast.success("Drone access shared successfully!");
        setIsShareOpen(false);
        setSharePilotEmail("");
        if (devices.length > 0) {
          setShareDroneId(devices[0].id);
        }
        fetchDevices(true);
      } else {
        toast.error(data.error || "Failed to share drone access");
      }
    } catch (error) {
      console.error("Error sharing drone:", error);
      toast.error("Failed to share drone. Please try again.");
    } finally {
      setIsSharing(false);
    }
  };

  const openDeleteCollaboratorModal = (deviceId, collaboratorEmail) => {
    setCollaboratorToDelete({ deviceId, collaboratorEmail });
    setIsDeleteCollaboratorOpen(true);
  };

  const handleRemoveCollaborator = async () => {
    const { deviceId, collaboratorEmail } = collaboratorToDelete;
    setDeletingCollaborator(`${deviceId}-${collaboratorEmail}`);

    try {
      const response = await fetch(`/api/devices/${deviceId}/collaborators/${encodeURIComponent(collaboratorEmail)}`, {
        method: "DELETE"
      });

      if (response.ok) {
        toast.success("Collaborator removed successfully");
        setIsDeleteCollaboratorOpen(false);
        setCollaboratorToDelete(null);
        fetchDevices(true);
      } else {
        const data = await response.json();
        toast.error(data.error || "Failed to remove collaborator");
      }
    } catch (error) {
      console.error("Error removing collaborator:", error);
      toast.error("Failed to remove collaborator");
    } finally {
      setDeletingCollaborator(null);
    }
  };

  const openDeleteDeviceModal = (device) => {
    setDeviceToDelete(device);
    setDeleteConfirmName("");
    setIsDeleteDeviceOpen(true);
  };

  const handleDeleteDevice = async () => {
    if (deleteConfirmName !== deviceToDelete.name) {
      toast.error("Device name doesn't match");
      return;
    }

    setIsDeleting(true);

    try {
      const response = await fetch(`/api/devices/${deviceToDelete.id}`, {
        method: "DELETE"
      });

      if (response.ok) {
        toast.success("Device deleted successfully");
        setIsDeleteDeviceOpen(false);
        setDeviceToDelete(null);
        setDeleteConfirmName("");
        fetchDevices(true);
      } else {
        const data = await response.json();
        toast.error(data.error || "Failed to delete device");
      }
    } catch (error) {
      console.error("Error deleting device:", error);
      toast.error("Failed to delete device");
    } finally {
      setIsDeleting(false);
    }
  };

  const formatDate = (dateString) => {
    const date = new Date(dateString);
    return date.toLocaleDateString("en-US", {
      year: "numeric",
      month: "short",
      day: "numeric",
    });
  };

  return (
    <div className="space-y-6">
      <div className="flex flex-col md:flex-row md:items-center md:justify-between gap-3">
        <div>
          <h1 className="hidden md:block text-2xl md:text-3xl font-bold text-gray-900 mb-2">
            Access Control
          </h1>
          <p className="text-gray-600 text-sm md:text-base">
            Manage drone access permissions and shared links
          </p>
        </div>
        <div className="flex flex-col sm:flex-row gap-2 w-full sm:w-auto">
          <button
            type="button"
            onClick={() => fetchDevices(true)}
            disabled={isRefreshing}
            className="inline-flex items-center justify-center gap-2 rounded-lg border border-gray-300 text-gray-900 px-3 py-2 text-sm font-semibold shadow-sm hover:bg-gray-50 transition disabled:opacity-50"
          >
            <RefreshCcw className={`h-4 w-4 ${isRefreshing ? 'animate-spin' : ''}`} />
          </button>
          <button
            type="button"
            onClick={() => setIsShareOpen(true)}
            disabled={devices.length === 0}
            className="inline-flex items-center justify-center gap-2 rounded-lg border border-gray-300 text-gray-900 px-4 py-2 text-sm font-semibold shadow-sm hover:bg-gray-50 transition disabled:opacity-50"
          >
            <Shield className="h-4 w-4" />
            Share Drone
          </button>
          <button
            type="button"
            onClick={() => setIsAddDroneOpen(true)}
            className="inline-flex items-center justify-center gap-2 rounded-lg bg-gray-900 text-white px-4 py-2 text-sm font-semibold shadow hover:bg-gray-800 transition"
          >
            <Plus className="h-4 w-4" />
            Add Drone
          </button>
        </div>
      </div>

      {isLoadingDevices ? (
        <div className="rounded-lg bg-white border border-gray-200 shadow-sm p-8 text-center">
          <RefreshCcw className="w-8 h-8 animate-spin text-gray-400 mx-auto mb-2" />
          <p className="text-gray-500">Loading devices...</p>
        </div>
      ) : (
        <div className="grid grid-cols-1 lg:grid-cols-1 gap-4">
          {devices.length === 0 ? (
            <motion.div
              initial={{ opacity: 0, y: 20 }}
              animate={{ opacity: 1, y: 0 }}
              transition={{ duration: 0.3 }}
              className="lg:col-span-2 rounded-lg bg-white border border-gray-200 shadow-sm p-8 text-center"
            >
              <p className="text-gray-500">No devices found. Add a drone to get started.</p>
            </motion.div>
          ) : (
            devices.map((device, deviceIndex) => (
              <motion.div
                key={device.id}
                initial={{ opacity: 0, y: 20 }}
                animate={{ opacity: 1, y: 0 }}
                transition={{ duration: 0.3, delay: deviceIndex * 0.1 }}
                className="rounded-lg bg-white border border-gray-200 shadow-sm overflow-hidden"
              >
                <button
                  onClick={() => toggleDrone(device.id)}
                  className="w-full bg-gray-50 border-b border-gray-200 px-6 py-4 hover:bg-gray-100 transition-colors flex items-center justify-between"
                >
                  <div className="text-left flex-1">
                    <h3 className="text-lg font-semibold text-gray-900">
                      {device.name.length > 50 ? `${device.name.substring(0, 50)}...` : device.name}
                    </h3>
                    <p className="text-sm text-gray-400">
                      {device.id.length > 50 ? `${device.id.substring(0, 50)}...` : device.id}
                    </p>
                  </div>
                  <div className="flex items-center gap-3">
                    <div
                      onClick={(e) => {
                        e.stopPropagation();
                        openDeleteDeviceModal(device);
                      }}
                      className="p-2 rounded-lg text-red-600 hover:bg-red-50 transition-colors cursor-pointer"
                      title="Delete Device"
                    >
                      <Trash2 className="h-4 w-4" />
                    </div>
                    <span className="text-sm text-gray-500">
                      {device.collaborators?.length || 0} {device.collaborators?.length === 1 ? 'collaborators' : 'collaborators'}
                    </span>
                    {expandedDrones[device.id] ? (
                      <ChevronDown className="h-5 w-5 text-gray-500" />
                    ) : (
                      <ChevronRight className="h-5 w-5 text-gray-500" />
                    )}
                  </div>
                </button>

                {expandedDrones[device.id] && (
                  <div className="p-4 space-y-3">
                    {!device.collaborators || device.collaborators.length === 0 ? (
                      <p className="text-center text-gray-500 text-sm py-4">
                        No collaborators yet. Share this drone to add collaborators.
                      </p>
                    ) : (
                      device.collaborators.map((collaborator) => (
                        <div
                          key={collaborator.id}
                          className="border border-gray-200 rounded-lg p-4 hover:border-gray-300 transition-colors"
                        >
                          <div className="flex items-start justify-between mb-2">
                            <div className="flex-1">
                              <div className="flex items-center gap-2 mb-1">
                                <p className="text-sm font-semibold text-gray-900">
                                  {collaborator.email}
                                </p>
                                <span className="inline-flex px-2 py-0.5 text-xs font-semibold rounded-full text-green-600 bg-green-50">
                                  Active
                                </span>
                              </div>
                              <p className="text-xs text-gray-500">{collaborator.email}</p>
                            </div>
                            <button
                              onClick={() => openDeleteCollaboratorModal(device.id, collaborator.email)}
                              disabled={deletingCollaborator === `${device.id}-${collaborator.email}`}
                              className="p-2 rounded-lg text-red-600 hover:bg-red-50 transition-colors disabled:opacity-50"
                              title="Remove Collaborator"
                            >
                              <Trash2 className="h-4 w-4" />
                            </button>
                          </div>
                          <div className="flex flex-wrap gap-4 mt-3 pt-3 border-t border-gray-100">
                            <div>
                              <p className="text-xs text-gray-500 mb-0.5">Shared</p>
                              <p className="text-xs font-medium text-gray-900">
                                {formatDate(collaborator.added_at)}
                              </p>
                            </div>
                            <div>
                              <p className="text-xs text-gray-500 mb-0.5">Last Active</p>
                              <p className="text-xs font-medium text-gray-900">
                                Just now
                              </p>
                            </div>
                          </div>
                        </div>
                      ))
                    )}
                  </div>
                )}
              </motion.div>
            ))
          )}
        </div>
      )}

      {/* Add Drone Modal */}
      {isAddDroneOpen && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/40 px-4">
          <div className="w-full max-w-md rounded-2xl bg-white p-6 shadow-xl">
            <div className="mb-4">
              <h3 className="text-lg font-semibold text-gray-900">
                Register New Drone
              </h3>
              <p className="text-sm text-gray-500">
                Enter the unique pairing code to pair it with the console.
              </p>
            </div>
            <form onSubmit={handleAddDrone} className="space-y-4">
              <div>
                <label className="text-xs font-semibold text-gray-600 uppercase">
                  Pairing Code
                </label>
                <input
                  type="text"
                  value={pairingCode}
                  onChange={(e) => setPairingCode(e.target.value)}
                  placeholder="12345678"
                  className="mt-1 w-full rounded-lg border border-gray-300 px-3 py-2 text-sm text-gray-900 focus:border-gray-900 focus:outline-none focus:ring-1 focus:ring-gray-900"
                  required
                  disabled={isSubmitting}
                />
              </div>
              <div>
                <label className="text-xs font-semibold text-gray-600 uppercase">
                  Drone Name
                </label>
                <input
                  type="text"
                  value={droneName}
                  onChange={(e) => setDroneName(e.target.value)}
                  placeholder="My Drone"
                  className="mt-1 w-full rounded-lg border border-gray-300 px-3 py-2 text-sm text-gray-900 focus:border-gray-900 focus:outline-none focus:ring-1 focus:ring-gray-900"
                  required
                  disabled={isSubmitting}
                />
              </div>

              <div className="flex items-center justify-end gap-2">
                <button
                  type="button"
                  onClick={() => setIsAddDroneOpen(false)}
                  className="rounded-lg border border-gray-200 px-4 py-2 text-sm font-semibold text-gray-700 hover:bg-gray-50"
                  disabled={isSubmitting}
                >
                  Cancel
                </button>
                <button
                  type="submit"
                  className="inline-flex items-center gap-2 rounded-lg bg-gray-900 px-4 py-2 text-sm font-semibold text-white hover:bg-gray-800 disabled:opacity-50 disabled:cursor-not-allowed"
                  disabled={isSubmitting}
                >
                  <Plus className="h-4 w-4" />
                  {isSubmitting ? "Adding..." : "Add Drone"}
                </button>
              </div>
            </form>
          </div>
        </div>
      )}

      {/* Share Drone Modal */}
      {isShareOpen && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/40 px-4">
          <div className="w-full max-w-md rounded-2xl bg-white p-6 shadow-xl">
            <div className="mb-4">
              <h3 className="text-lg font-semibold text-gray-900">
                Share Drone Access
              </h3>
              <p className="text-sm text-gray-500">
                Select a drone and provide the pilot email to grant access.
              </p>
            </div>
            <form onSubmit={handleShareDrone} className="space-y-4">
              <div>
                <label className="text-xs font-semibold text-gray-600 uppercase">
                  Pilot Email
                </label>
                <input
                  type="email"
                  value={sharePilotEmail}
                  onChange={(e) => setSharePilotEmail(e.target.value)}
                  placeholder="pilot@example.com"
                  className="mt-1 w-full rounded-lg border border-gray-300 px-3 py-2 text-sm text-gray-900 focus:border-gray-900 focus:outline-none focus:ring-1 focus:ring-gray-900"
                  required
                  disabled={isSharing}
                />
              </div>
              <div>
                <label className="text-xs font-semibold text-gray-600 uppercase">
                  Select Drone
                </label>
                <select
                  value={shareDroneId}
                  onChange={(e) => setShareDroneId(e.target.value)}
                  className="mt-1 w-full rounded-lg border border-gray-300 px-3 py-2 text-sm text-gray-900 focus:border-gray-900 focus:outline-none focus:ring-1 focus:ring-gray-900"
                  disabled={isSharing}
                >
                  {devices.length === 0 ? (
                    <option value="">No devices available</option>
                  ) : (
                    devices.map((drone) => (
                      <option key={drone.id} value={drone.id}>
                        {drone.name} ({drone.id.substring(0, 8)}...)
                      </option>
                    ))
                  )}
                </select>
              </div>
              <div className="flex items-center justify-end gap-2">
                <button
                  type="button"
                  onClick={() => setIsShareOpen(false)}
                  className="rounded-lg border border-gray-200 px-4 py-2 text-sm font-semibold text-gray-700 hover:bg-gray-50"
                  disabled={isSharing}
                >
                  Cancel
                </button>
                <button
                  type="submit"
                  disabled={devices.length === 0 || isSharing}
                  className="inline-flex items-center gap-2 rounded-lg bg-gray-900 px-4 py-2 text-sm font-semibold text-white hover:bg-gray-800 disabled:opacity-50 disabled:cursor-not-allowed"
                >
                  <Shield className="h-4 w-4" />
                  {isSharing ? "Sharing..." : "Share Drone"}
                </button>
              </div>
            </form>
          </div>
        </div>
      )}

      {/* Delete Device Confirmation Modal */}
      {isDeleteDeviceOpen && deviceToDelete && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/40 px-4">
          <div className="w-full max-w-md rounded-2xl bg-white p-6 shadow-xl">
            <div className="mb-4 flex items-start gap-3">
              <div className="p-2 bg-red-100 rounded-lg">
                <AlertTriangle className="h-6 w-6 text-red-600" />
              </div>
              <div>
                <h3 className="text-lg font-semibold text-gray-900">
                  Delete Device
                </h3>
                <p className="text-sm text-gray-500 mt-1">
                  This action cannot be undone. This will permanently delete the device and all associated data.
                </p>
              </div>
            </div>

            <div className="mb-4 p-3 bg-gray-50 rounded-lg">
              <p className="text-sm text-gray-700 mb-2">
                Type <span className="font-mono font-semibold">{deviceToDelete.name}</span> to confirm:
              </p>
              <input
                type="text"
                value={deleteConfirmName}
                onChange={(e) => setDeleteConfirmName(e.target.value)}
                placeholder="Enter device name"
                className="w-full rounded-lg border border-gray-300 px-3 py-2 text-sm text-gray-900 focus:border-red-500 focus:outline-none focus:ring-1 focus:ring-red-500"
                disabled={isDeleting}
              />
            </div>

            <div className="flex items-center justify-end gap-2">
              <button
                type="button"
                onClick={() => {
                  setIsDeleteDeviceOpen(false);
                  setDeviceToDelete(null);
                  setDeleteConfirmName("");
                }}
                className="rounded-lg border border-gray-200 px-4 py-2 text-sm font-semibold text-gray-700 hover:bg-gray-50"
                disabled={isDeleting}
              >
                Cancel
              </button>
              <button
                type="button"
                onClick={handleDeleteDevice}
                disabled={deleteConfirmName !== deviceToDelete.name || isDeleting}
                className="inline-flex items-center gap-2 rounded-lg bg-red-600 px-4 py-2 text-sm font-semibold text-white hover:bg-red-700 disabled:opacity-50 disabled:cursor-not-allowed"
              >
                <Trash2 className="h-4 w-4" />
                {isDeleting ? "Deleting..." : "Delete Device"}
              </button>
            </div>
          </div>
        </div>
      )}

      {/* Delete Collaborator Confirmation Modal */}
      {isDeleteCollaboratorOpen && collaboratorToDelete && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/40 px-4">
          <div className="w-full max-w-md rounded-2xl bg-white p-6 shadow-xl">
            <div className="mb-4 flex items-start gap-3">
              <div className="p-2 bg-red-100 rounded-lg">
                <AlertTriangle className="h-6 w-6 text-red-600" />
              </div>
              <div>
                <h3 className="text-lg font-semibold text-gray-900">
                  Remove Collaborator
                </h3>
                <p className="text-sm text-gray-500 mt-1">
                  Are you sure you want to remove <span className="font-semibold">{collaboratorToDelete.collaboratorEmail}</span> from this device? They will lose access immediately.
                </p>
              </div>
            </div>

            <div className="flex items-center justify-end gap-2">
              <button
                type="button"
                onClick={() => {
                  setIsDeleteCollaboratorOpen(false);
                  setCollaboratorToDelete(null);
                }}
                className="rounded-lg border border-gray-200 px-4 py-2 text-sm font-semibold text-gray-700 hover:bg-gray-50"
                disabled={deletingCollaborator}
              >
                Cancel
              </button>
              <button
                type="button"
                onClick={handleRemoveCollaborator}
                disabled={deletingCollaborator}
                className="inline-flex items-center gap-2 rounded-lg bg-red-600 px-4 py-2 text-sm font-semibold text-white hover:bg-red-700 disabled:opacity-50 disabled:cursor-not-allowed"
              >
                <Trash2 className="h-4 w-4" />
                {deletingCollaborator ? "Removing..." : "Remove"}
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
};

export default AccessControlTab;