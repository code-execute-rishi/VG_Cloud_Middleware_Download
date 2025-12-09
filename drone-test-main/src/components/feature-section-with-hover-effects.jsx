'use client'
import { cn } from "@/lib/utils";
import {
  IconBolt,
  IconCommand,
  IconDeviceDesktop,
  IconShieldLock,
  IconShare2,
  IconUsers,
  IconWifi,
  IconTerminal2,
} from "@tabler/icons-react";

export function FeaturesSectionWithHoverEffects() {
  const features = [
    {
       title: "WebRTC Ultra-Low Latency",
    description:
      "Experience 100–300ms latency with real-time control and monitoring powered by WebRTC.",
    icon: <IconBolt />,
  },
  {
    title: "Professional Ground Station",
    description:
      "Desktop control with MAVLink proxy on TCP 5180. Connect Mission Planner or QGroundControl with full mission control capabilities.",
    icon: <IconDeviceDesktop />,
  },
  {
    title: "4G/LTE Global Coverage",
    description:
      "Fly anywhere with cellular coverage. Automatic network failover ensures smooth, stable low-latency streams.",
    icon: <IconWifi />,
  },
  {
    title: "Enterprise Security",
    description:
      "End-to-end WebRTC DTLS encryption, secure SSH tunneling, and 99.7% uptime with automatic reconnection.",
    icon: <IconShieldLock />,
  },
  
  
  {
    title: "Unlimited Multi-User Access",
    description:
      "Share live streams publicly or privately with unlimited viewers. Perfect for teams and clients.",
    icon: <IconUsers />,
  },
  {
    title: "Remote SSH Management",
    description:
      "Securely access and manage your drone’s onboard computer (e.g., Raspberry Pi) from anywhere in the world.",
    icon: <IconTerminal2 />,
  },
  {
    title: "Live Demo Sharing",
    description:
      "Generate secure public links to share live drone operations in real-time without accounts.",
    icon: <IconShare2 />,
  },
  {
    title: "MAVLink Telemetry Integration",
    description:
      "Stream flight data alongside video—monitor altitude, GPS, battery, and more with full MAVLink support.",
    icon: <IconCommand />,
  },
];
  return (
    <div
      className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4  relative z-10 py-10 max-w-7xl mx-auto">
      {features.map((feature, index) => (
        <Feature key={feature.title} {...feature} index={index} />
      ))}
    </div>
  );
}

const Feature = ({
  title,
  description,
  icon,
  index
}) => {
  return (
    <div
      className={cn(
        "flex flex-col lg:border-r py-10 relative group/feature border-neutral-800 transition-all duration-300 hover:bg-neutral-800/30 hover:scale-[1.02] cursor-pointer",
        (index === 0 || index === 4) && "lg:border-l border-neutral-800",
        index < 4 && "lg:border-b border-neutral-800"
      )}>
      {index < 4 && (
        <div
          className="opacity-0 group-hover/feature:opacity-100 transition duration-300 absolute inset-0 h-full w-full bg-gradient-to-t from-neutral-800 to-transparent pointer-events-none" />
      )}
      {index >= 4 && (
        <div
          className="opacity-0 group-hover/feature:opacity-100 transition duration-300 absolute inset-0 h-full w-full bg-gradient-to-b from-neutral-800 to-transparent pointer-events-none" />
      )}
      <div
        className="mb-4 relative z-10 px-10 text-neutral-400 transition-all duration-300 group-hover/feature:text-blue-400 group-hover/feature:scale-110">
        {icon}
      </div>
      <div className="text-lg font-bold mb-2 relative z-10 px-10">
        <div
          className="absolute left-0 inset-y-0 h-6 group-hover/feature:h-8 w-1 rounded-tr-full rounded-br-full bg-neutral-700 group-hover/feature:bg-blue-500 transition-all duration-300 origin-center group-hover/feature:shadow-lg group-hover/feature:shadow-blue-500/50" />
        <span
          className="group-hover/feature:translate-x-2 transition duration-300 inline-block text-neutral-100 group-hover/feature:text-blue-400">
          {title}
        </span>
      </div>
      <p
        className="text-sm text-neutral-300 max-w-xs relative z-10 px-10 transition-colors duration-300 group-hover/feature:text-neutral-200">
        {description}
      </p>
    </div>
  );
};
