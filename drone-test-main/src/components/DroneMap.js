'use client';

import { useEffect, useRef, useState, forwardRef, useImperativeHandle } from 'react';
import mapboxgl from 'mapbox-gl';
import 'mapbox-gl/dist/mapbox-gl.css';

// Set access token globally
if (typeof window !== 'undefined') {
  const token = process.env.NEXT_PUBLIC_MAPBOX_TOKEN;
  if (token) {
    mapboxgl.accessToken = token;
  }
}

const DroneMap = forwardRef((props, ref) => {
  const mapContainer = useRef(null);
  const map = useRef(null);
  const marker = useRef(null);
  const markerEl = useRef(null);
  const [mapLoaded, setMapLoaded] = useState(false);
  const directionRef = useRef(45);

  const [dronePosition, setDronePosition] = useState({
    lat: 23.210837,
    lng: 77.410526,
  });

  const [droneHeading, setDroneHeading] = useState(0);

  const startPosition = { lat: 23.210837, lng: 77.410526 };

  const createDroneMarker = (rotation = 0, lat = 0, lng = 0) => {
    if (markerEl.current) {
      const iconEl = markerEl.current.querySelector('.drone-icon-inner');
      const coordsEl = markerEl.current.querySelector('.drone-coords');
      
      if (iconEl) {
        iconEl.style.transform = `rotate(${rotation - 90}deg)`;
      }
      if (coordsEl) {
        coordsEl.textContent = `${lat.toFixed(5)}, ${lng.toFixed(5)}`;
      }
      return;
    }

    const el = document.createElement('div');
    el.className = 'drone-marker-wrapper';
    el.innerHTML = `
      <div style="
        display: flex;
        flex-direction: column;
        align-items: center;
      ">
        <div class="drone-icon-inner" style="
          width: 30px;
          height: 30px;
          font-size: 30px;
          transform: rotate(${rotation - 90}deg);
          filter: drop-shadow(0 0 3px rgba(255,255,255,0.8));
          text-align: center;
          line-height: 30px;
        ">✈</div>
        <div class="drone-coords" style="
          background: rgba(0, 0, 0, 0.7);
          color: white;
          padding: 2px 6px;
          border-radius: 4px;
          font-size: 10px;
          font-family: monospace;
          white-space: nowrap;
          margin-top: 2px;
          box-shadow: 0 2px 4px rgba(0,0,0,0.3);
        ">${lat.toFixed(5)}, ${lng.toFixed(5)}</div>
      </div>
    `;
    markerEl.current = el;
  };

  useEffect(() => {
    if (map.current) return;

    // Wait for container to be ready
    const initMap = () => {
      if (!mapContainer.current) {
        setTimeout(initMap, 50);
        return;
      }

      const mapboxToken = process.env.NEXT_PUBLIC_MAPBOX_TOKEN || mapboxgl.accessToken;
      if (!mapboxToken) {
        console.error('Mapbox token not found');
        return;
      }

      mapboxgl.accessToken = mapboxToken;

      try {
        map.current = new mapboxgl.Map({
          container: mapContainer.current,
          style: 'mapbox://styles/mapbox/satellite-v9',
          center: [startPosition.lng, startPosition.lat],
          zoom: 15,
          pitch: 60,
          bearing: 0,
          antialias: true,
        });

        map.current.on('load', () => {
          map.current.addSource('mapbox-dem', {
            type: 'raster-dem',
            url: 'mapbox://mapbox.mapbox-terrain-dem-v1',
            tileSize: 512,
            maxzoom: 14
          });

          map.current.setTerrain({ 
            source: 'mapbox-dem', 
            exaggeration: 1.5 
          });

          map.current.addLayer({
            id: '3d-buildings',
            source: 'composite',
            'source-layer': 'building',
            filter: ['==', ['get', 'extrude'], 'true'],
            type: 'fill-extrusion',
            minzoom: 14,
            paint: {
              'fill-extrusion-color': '#aaa',
              'fill-extrusion-height': [
                'interpolate',
                ['linear'],
                ['zoom'],
                15, 0,
                15.05, ['get', 'height']
              ],
              'fill-extrusion-base': [
                'interpolate',
                ['linear'],
                ['zoom'],
                15, 0,
                15.05, ['get', 'min_height']
              ],
              'fill-extrusion-opacity': 0.6
            }
          });

          setMapLoaded(true);
          createDroneMarker(droneHeading || directionRef.current, dronePosition.lat, dronePosition.lng);

          marker.current = new mapboxgl.Marker({
            element: markerEl.current,
            anchor: 'center',
          })
            .setLngLat([dronePosition.lng, dronePosition.lat])
            .addTo(map.current);
        });

        map.current.on('error', (e) => {
          console.error('Map error:', e);
        });
      } catch (error) {
        console.error('Error initializing map:', error);
      }
    };

    initMap();

    return () => {
      if (map.current) {
        map.current.remove();
        map.current = null;
      }
    };
  }, []);

  // Update marker when position or heading changes (from real MAVLink data)
  useEffect(() => {
    if (!mapLoaded || !marker.current) return;

    if (marker.current) {
      marker.current.setLngLat([dronePosition.lng, dronePosition.lat]);
      createDroneMarker(droneHeading, dronePosition.lat, dronePosition.lng);
      directionRef.current = droneHeading;
    }

    // Smoothly pan map to follow drone (only if map is loaded)
    if (map.current) {
      map.current.easeTo({
        center: [dronePosition.lng, dronePosition.lat],
        duration: 500,
        easing: (t) => t * (2 - t),
      });
    }
  }, [dronePosition, droneHeading, mapLoaded]);

  const updateDronePosition = (lat, lng) => {
    setDronePosition({ lat, lng });
  };

  const updateDroneHeading = (heading) => {
    setDroneHeading(heading);
  };

  const updateDroneData = (lat, lng, heading) => {
    setDronePosition({ lat, lng });
    setDroneHeading(heading);
  };

  useEffect(() => {
    if (typeof window !== 'undefined') {
      window.updateDronePosition = updateDronePosition;
      window.updateDroneHeading = updateDroneHeading;
      window.updateDroneData = updateDroneData;
    }
  }, []);

  // Expose zoom and 3D control methods to parent component
  useImperativeHandle(ref, () => ({
    zoomIn: () => {
      if (map.current) {
        map.current.zoomIn({ duration: 300 });
      }
    },
    zoomOut: () => {
      if (map.current) {
        map.current.zoomOut({ duration: 300 });
      }
    },
    adjustPitch: (delta) => {
      if (map.current) {
        const currentPitch = map.current.getPitch();
        const newPitch = Math.max(0, Math.min(85, currentPitch + delta));
        map.current.easeTo({ pitch: newPitch, duration: 300 });
      }
    },
    adjustBearing: (delta) => {
      if (map.current) {
        const currentBearing = map.current.getBearing();
        const newBearing = currentBearing + delta;
        map.current.easeTo({ bearing: newBearing, duration: 300 });
      }
    },
    resetView: () => {
      if (map.current) {
        map.current.easeTo({ pitch: 60, bearing: 0, duration: 500 });
      }
    },
  }));

  return (
    <div style={{ 
      position: 'absolute',
      top: 0,
      left: 0,
      width: '100%',
      height: '100%'
    }}>
      <div 
        ref={mapContainer} 
        style={{ 
          height: '100%', 
          width: '100%',
          position: 'absolute',
          top: 0,
          left: 0,
          backgroundColor: '#000'
        }} 
      />
    </div>
  );
});

DroneMap.displayName = 'DroneMap';

export default DroneMap;
