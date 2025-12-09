"use client";
import React, { useState, createContext, useContext, useEffect } from "react";
import { motion, AnimatePresence } from "framer-motion";
import { Menu, X } from "lucide-react";

const SidebarContext = createContext(undefined);

export const useSidebar = () => {
  const context = useContext(SidebarContext);
  if (!context) {
    throw new Error("useSidebar must be used within a SidebarProvider");
  }
  return context;
};

export const SidebarProvider = ({ children, open: openProp, setOpen: setOpenProp }) => {
  const [openState, setOpenState] = useState(false);
  const [isMobile, setIsMobile] = useState(false);

  useEffect(() => {
    const checkMobile = () => {
      const mobile = window.innerWidth < 768;
      setIsMobile(mobile);
      if (mobile) {
        setOpenState(false);
      }
    };
    checkMobile();
    window.addEventListener("resize", checkMobile);
    return () => window.removeEventListener("resize", checkMobile);
  }, []);

  const open = openProp !== undefined ? openProp : openState;
  const setOpen = setOpenProp !== undefined ? setOpenProp : setOpenState;

  return (
    <SidebarContext.Provider value={{ open, setOpen, isMobile }}>
      {children}
    </SidebarContext.Provider>
  );
};

export const Sidebar = ({ children, open, setOpen }) => {
  return (
    <SidebarProvider open={open} setOpen={setOpen}>
      {children}
    </SidebarProvider>
  );
};

export const SidebarBody = (props) => {
  return (
    <>
      <DesktopSidebar {...props} />
      <MobileSidebar {...props} />
    </>
  );
};

export const DesktopSidebar = ({ className, children, ...props }) => {
  return (
    <div
      className={`h-full px-4 py-4 hidden md:flex md:flex-col border-r border-gray-200 overflow-hidden bg-white w-[15%] flex-shrink-0 ${className || ""}`}
      {...props}
    >
      <div className="overflow-y-auto overflow-x-hidden flex-1">
        {children}
      </div>
    </div>
  );
};

export const MobileSidebar = ({ className, children, ...props }) => {
  const { open, setOpen } = useSidebar();
  
  return (
    <AnimatePresence>
        {open && (
          <>
            <motion.div
              initial={{ opacity: 0 }}
              animate={{ opacity: 1 }}
              exit={{ opacity: 0 }}
              transition={{ duration: 0.2 }}
              className="fixed inset-0 bg-black/30 z-[99] md:hidden"
              onClick={() => setOpen(false)}
            />
            <motion.div
              initial={{ x: "-100%" }}
              animate={{ x: 0 }}
              exit={{ x: "-100%" }}
              transition={{ duration: 0.3, ease: "easeInOut" }}
              className={`fixed h-full w-80 max-w-[85vw] inset-y-0 left-0 bg-white p-6 z-[100] flex flex-col shadow-xl overflow-hidden ${className || ""}`}
            >
              <div className="flex justify-between items-center mb-8 flex-shrink-0">
                <motion.div
                  initial={{ opacity: 0, x: -20 }}
                  animate={{ opacity: 1, x: 0 }}
                  transition={{ delay: 0.1 }}
                  className="text-gray-900 font-semibold text-lg"
                >
                  Menu
                </motion.div>
                <motion.button
                  onClick={() => setOpen(false)}
                  className="text-gray-500 hover:text-gray-900"
                  whileHover={{ scale: 1.1 }}
                  whileTap={{ scale: 0.9 }}
                >
                  <X className="h-5 w-5" />
                </motion.button>
              </div>
              <div className="flex-1 overflow-y-auto overflow-x-hidden">{children}</div>
            </motion.div>
          </>
        )}
    </AnimatePresence>
  );
};

export const SidebarTab = ({ tab, isActive, onClick, className, ...props }) => {
  const { open, isMobile } = useSidebar();
  const isOpen = !isMobile || open;
  
  if (!tab) return null;
  
  return (
    <button
      onClick={onClick}
      className={`flex items-center gap-3 py-2.5 px-3 rounded-lg transition-all duration-200 w-full text-left ${
        isActive
          ? "bg-gray-900 text-white shadow-md"
          : "text-gray-700 hover:bg-gray-100 hover:text-gray-900"
      } ${className || ""}`}
      {...props}
    >
      <motion.div
        whileHover={{ scale: 1.1 }}
        transition={{ duration: 0.2 }}
        className="flex-shrink-0"
      >
        {tab.icon}
      </motion.div>
      {isOpen && (
        <span className="text-sm whitespace-nowrap overflow-hidden font-medium">
          {tab.label}
        </span>
      )}
    </button>
  );
};
