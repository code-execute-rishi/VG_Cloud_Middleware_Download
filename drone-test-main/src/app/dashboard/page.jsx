"use client";

import dynamic from "next/dynamic";

const DashboardPage = dynamic(
  () => import("@/components/DashboardClient"),
  { 
    ssr: false,
    loading: () => (
      <div className="flex h-screen w-full bg-gray-100 items-center justify-center">
        <div className="text-gray-900">Loading dashboard...</div>
      </div>
    ),
  }
);

export default DashboardPage;