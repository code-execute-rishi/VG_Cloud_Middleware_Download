import React, { useEffect } from 'react';
import { MapContainer, TileLayer, Marker, Popup, useMap } from 'react-leaflet';
import L from 'leaflet';
import 'leaflet/dist/leaflet.css';

// Fix for default Leaflet marker icons in React
import icon from 'leaflet/dist/images/marker-icon.png';
import iconShadow from 'leaflet/dist/images/marker-shadow.png';

let DefaultIcon = L.icon({
    iconUrl: icon,
    shadowUrl: iconShadow,
    iconSize: [25, 41],
    iconAnchor: [12, 41]
});

L.Marker.prototype.options.icon = DefaultIcon;

// Component to auto-center map
const RecenterMap = ({ lat, lon }) => {
    const map = useMap();
    useEffect(() => {
        if (lat !== 0 && lon !== 0) {
            map.setView([lat, lon]);
        }
    }, [lat, lon, map]);
    return null;
};

const MapView = ({ telemetry }) => {
    const { lat, lon, heading } = telemetry;

    // Custom Rotated Icon
    const createRotatedIcon = (heading) => {
        return L.divIcon({
            className: 'custom-drone-icon',
            html: `
                <div style="
                    transform: rotate(${heading}deg);
                    width: 40px;
                    height: 40px;
                    display: flex;
                    align-items: center;
                    justify-content: center;
                ">
                    <svg viewBox="0 0 24 24" fill="none" stroke="#ef4444" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" style="width: 100%; height: 100%; filter: drop-shadow(0 0 4px rgba(0,0,0,0.5));">
                        <path d="M12 2L2 22l10-3 10 3L12 2z"/>
                    </svg>
                </div>
            `,
            iconSize: [40, 40],
            iconAnchor: [20, 20],
        });
    };

    // Default position (Canberra, Australia - common SITL location) if 0,0
    const displayLat = lat || -35.363261;
    const displayLon = lon || 149.165230;

    return (
        <div className="h-full w-full rounded-lg overflow-hidden border border-military-700 shadow-lg bg-military-800 relative z-0">
            <MapContainer 
                center={[displayLat, displayLon]} 
                zoom={18} 
                style={{ height: '100%', width: '100%' }}
                zoomControl={false}
            >
                <TileLayer
                    attribution='&copy; <a href="https://www.openstreetmap.org/copyright">OpenStreetMap</a> contributors'
                    url="https://{s}.tile.openstreetmap.org/{z}/{x}/{y}.png"
                    className="map-tiles"
                />
                
                <Marker 
                    position={[displayLat, displayLon]} 
                    icon={createRotatedIcon(heading)}
                >
                    <Popup>
                        Drone Location<br />
                        Lat: {displayLat.toFixed(6)}<br />
                        Lon: {displayLon.toFixed(6)}
                    </Popup>
                </Marker>

                <RecenterMap lat={displayLat} lon={displayLon} />
            </MapContainer>
            
            {/* Overlay Info */}
            <div className="absolute top-4 right-4 bg-military-900/80 backdrop-blur p-2 rounded border border-military-700 text-xs text-gray-300 z-[1000]">
                <div>HDG: {heading.toFixed(1)}°</div>
                <div>LAT: {displayLat.toFixed(6)}</div>
                <div>LON: {displayLon.toFixed(6)}</div>
            </div>
        </div>
    );
};

export default MapView;
