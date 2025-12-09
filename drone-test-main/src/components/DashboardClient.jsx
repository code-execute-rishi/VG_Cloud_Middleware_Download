"use client";

import React, { useState, useEffect } from "react";
import { Sidebar, SidebarBody, SidebarTab } from "@/components/ui/sidebar";
import { LayoutDashboard, UserCog, Settings, Menu, Plane, Shield } from "lucide-react";
import { useUser, UserButton } from "@clerk/nextjs";
import { motion, AnimatePresence } from "framer-motion";
import DashboardOverview from "@/components/dashboard/DashboardOverview";
import ProfileTab from "@/components/dashboard/ProfileTab";
import SettingsTab from "@/components/dashboard/SettingsTab";
import FleetTab from "@/components/dashboard/FleetTab";
import AccessControlTab from "@/components/dashboard/AccessControlTab";

function DashboardClient() {
  const { user, isLoaded } = useUser();
  const [open, setOpen] = useState(false);
  const [isMobile, setIsMobile] = useState(false);
  const [activeTab, setActiveTab] = useState("dashboard");

  useEffect(() => {
    const checkMobile = () => {
      const mobile = window.innerWidth < 768;
      setIsMobile(mobile);
      if (mobile) {
        setOpen(false);
      }
    };
    checkMobile();
    window.addEventListener("resize", checkMobile);
    return () => window.removeEventListener("resize", checkMobile);
  }, []);

  if (!isLoaded) {
    return (
      <div className="flex h-screen w-full bg-gray-100 items-center justify-center">
        <motion.div
          initial={{ opacity: 0 }}
          animate={{ opacity: 1 }}
          className="text-gray-900"
        >
          Loading...
        </motion.div>
      </div>
    );
  }

  if (!user) {
    return (
      <div className="flex h-screen w-full bg-gray-100 items-center justify-center">
        <div className="text-gray-900">Please sign in.</div>
      </div>
    );
  }

  const tabs = [
    { 
      id: "dashboard", 
      label: "Dashboard", 
      icon: <LayoutDashboard className="h-5 w-5" /> 
    },
    { 
      id: "fleet", 
      label: "Fleet", 
      icon: <Plane className="h-5 w-5" /> 
    },
    { 
      id: "access-control", 
      label: "Access Control", 
      icon: <Shield className="h-5 w-5" /> 
    },
    { 
      id: "profile", 
      label: "Profile", 
      icon: <UserCog className="h-5 w-5" /> 
    },
    { 
      id: "settings", 
      label: "Settings", 
      icon: <Settings className="h-5 w-5" /> 
    },
  ];

  const renderTabContent = () => {
    switch (activeTab) {
      case "dashboard":
        return <DashboardOverview user={user} />;
      case "fleet":
        return <FleetTab />;
      case "access-control":
        return <AccessControlTab />;
      case "profile":
        return <ProfileTab user={user} />;
      case "settings":
        return <SettingsTab />;
      default:
        return null;
    }
  };

  return (
    <div className="flex h-screen w-full bg-gray-50 overflow-hidden">
      <Sidebar open={open} setOpen={setOpen}>
        <SidebarBody className="justify-between bg-white border-r border-gray-200">
          <div className="flex flex-col flex-1 overflow-y-auto overflow-x-hidden">
            <motion.div
              className="mb-6"
              initial={{ opacity: 0, y: -10 }}
              animate={{ opacity: 1, y: 0 }}
              transition={{ duration: 0.3 }}
            >
              <div className="text-gray-900 font-semibold text-lg">
                Console
              </div>
            </motion.div>
            <div className="flex flex-col gap-1">
              {tabs.map((tab, idx) => (
                <motion.div
                  key={tab.id}
                  initial={{ opacity: 0, x: -20 }}
                  animate={{ opacity: 1, x: 0 }}
                  transition={{ duration: 0.3, delay: idx * 0.1 }}
                >
                  <SidebarTab
                    tab={tab}
                    isActive={activeTab === tab.id}
                    onClick={() => {
                      setActiveTab(tab.id);
                      if (isMobile) setOpen(false);
                    }}
                  />
                </motion.div>
              ))}
            </div>
          </div>

          <motion.div
            className="border-t border-gray-200 pt-4 flex-shrink-0"
            initial={{ opacity: 0, y: 20 }}
            animate={{ opacity: 1, y: 0 }}
            transition={{ duration: 0.3, delay: 0.4 }}
          >
            <div className="flex items-center gap-3 px-3 py-2">
              <UserButton
                appearance={{
                  elements: {
                    avatarBox: "h-10 w-10",
                  },
                }}
              />
              <div className="flex-1 min-w-0 overflow-hidden">
                <p className="text-sm font-medium text-gray-900 truncate">
                  {user.fullName || user.firstName || user.emailAddresses[0]?.emailAddress || "User"}
                </p>
                <p className="text-xs text-gray-500 truncate">
                  {user.emailAddresses[0]?.emailAddress || ""}
                </p>
              </div>
            </div>
          </motion.div>
        </SidebarBody>
      </Sidebar>

      <motion.div
        className="flex flex-1 overflow-hidden bg-gray-50"
        initial={{ opacity: 0 }}
        animate={{ opacity: 1 }}
        transition={{ duration: 0.3 }}
      >
        <div className="flex flex-1 flex-col overflow-y-auto">
          <div className="md:hidden h-14 px-4 flex items-center justify-between bg-white border-b border-gray-200 sticky top-0 z-50">
            <motion.button
              onClick={() => setOpen(!open)}
              className="text-gray-700 p-2 rounded-lg hover:bg-gray-100"
              whileTap={{ scale: 0.9 }}
            >
              <Menu className="h-5 w-5" />
            </motion.button>
            <h1 className="text-lg font-semibold text-gray-900 capitalize">{activeTab}</h1>
            <div className="w-9" />
          </div>

          <div className="p-4 md:p-8 flex-1 pb-20 md:pb-8">
            <AnimatePresence mode="wait">
              <motion.div
                key={activeTab}
                initial={{ opacity: 0, y: 20 }}
                animate={{ opacity: 1, y: 0 }}
                exit={{ opacity: 0, y: -20 }}
                transition={{ duration: 0.3 }}
              >
                {renderTabContent()}
              </motion.div>
            </AnimatePresence>
          </div>
        </div>
      </motion.div>
    </div>
  );
}

export default DashboardClient;