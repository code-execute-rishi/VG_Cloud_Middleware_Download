import React from 'react';
import MapView from '../components/MapView';
import VideoFeed from '../components/VideoFeed';

const Dashboard = ({ telemetry }) => {
  return (
    <div className="flex-1 p-2 grid grid-cols-12 gap-2 min-h-0">
      {/* Left Panel: Video Feed (40%) */}
      <div className="col-span-12 lg:col-span-5 flex flex-col min-h-0 h-[40vh] lg:h-auto">
        <div className="flex-1 min-h-0">
          <VideoFeed />
        </div>
      </div>

      {/* Right Panel: Map (60%) */}
      <div className="col-span-12 lg:col-span-7 flex flex-col min-h-0 h-[50vh] lg:h-auto">
        <div className="flex-1 min-h-0">
          <MapView telemetry={telemetry} />
        </div>
      </div>
    </div>
  );
};

export default Dashboard;
