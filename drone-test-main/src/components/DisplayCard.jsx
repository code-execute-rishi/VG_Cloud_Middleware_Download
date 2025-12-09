"use client";

import DisplayCards from "@/components/ui/display-cards";
import { Sparkles } from "lucide-react";

const defaultCards = [
  {
    icon: <Sparkles className="size-4 text-white/80" />,
    title: "Live Link Quality",
    description: `{
  "signal_strength": "Good",
  "network_type": "4G/LTE",
  "packet_loss": "0.8%",
  "latency": "92ms"
}`,
    date: "Real-time",
    iconClassName: "text-white",
    titleClassName: "text-white",
    className:
      "[grid-area:stack] hover:-translate-y-6 before:absolute before:w-[100%] before:rounded-xl before:h-[100%] before:content-[''] before:bg-white/5 hover:before:bg-white/10 before:transition before:duration-500",
  },
  {
    icon: <Sparkles className="size-4 text-white/80" />,
    title: "Flight Status",
    description: `{
  "battery": "76%",
  "flight_mode": "AUTO",
  "altitude": "122m",
  "speed": "14.2 m/s"
}`,
    date: "Continuously updating",
    iconClassName: "text-white",
    titleClassName: "text-white",
    className:
      "[grid-area:stack] translate-x-12 translate-y-10 hover:-translate-y-2 before:absolute before:w-[100%] before:rounded-xl before:h-[100%] before:content-[''] before:bg-white/5 hover:before:bg-white/10 before:transition before:duration-500",
  },
  {
    icon: <Sparkles className="size-4 text-white/80" />,
    title: "System Health",
    description: `{
  "cpu_usage": "43%",
  "temperature": "58°C",
  "storage": "68% free",
  "memory": "54% used"
}`,
    date: "Monitored",
    iconClassName: "text-white",
    titleClassName: "text-white",
    className:
      "[grid-area:stack] translate-x-24 translate-y-20 hover:translate-y-10 before:absolute before:w-[100%] before:rounded-xl before:h-[100%] before:content-[''] before:bg-white/5 hover:before:bg-white/10 before:transition before:duration-500",
  },
];

const Card = () => {
  return (
    <div className="flex min-h-[400px] w-full items-center justify-center py-20 bg-black text-white">
      <div className="w-full max-w-4xl">
        <DisplayCards cards={defaultCards} />
      </div>
    </div>
  );
};

export default Card;
