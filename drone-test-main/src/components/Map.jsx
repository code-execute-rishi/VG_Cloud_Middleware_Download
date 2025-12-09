"use client"
import React from 'react'
import { WorldMap } from './ui/world-map'

const Map = () => {
    const mapDots = [{
            start: {
              lat: 64.2008,
              lng: -149.4937,
            }, // Alaska (Fairbanks)
            end: {
              lat: 34.0522,
              lng: -118.2437,
            }, // Los Angeles
          },
          {
            start: { lat: 64.2008, lng: -149.4937 }, // Alaska (Fairbanks)
            end: { lat: -15.7975, lng: -47.8919 }, // Brazil (Brasília)
          },
          {
            start: { lat: -15.7975, lng: -47.8919 }, // Brazil (Brasília)
            end: { lat: 38.7223, lng: -9.1393 }, // Lisbon
          },
          {
            start: { lat: 51.5074, lng: -0.1278 }, // London
            end: { lat: 28.6139, lng: 77.209 }, // New Delhi
          },
          {
            start: { lat: 28.6139, lng: 77.209 }, // New Delhi
            end: { lat: 43.1332, lng: 131.9113 }, // Vladivostok
          },
          {
            start: { lat: 28.6139, lng: 77.209 }, // New Delhi
            end: { lat: -1.2921, lng: 36.8219 }, // Nairobi
          },
        ]
  return (
    <div className="w-full bg-white py-20 px-4 h-screen">
          <div className="container mx-auto max-w-6xl">
            <div className="mb-12 text-center">
              <h2 className="text-4xl md:text-5xl font-bold mb-4">
                <span className="text-gray-900">Remote</span>{" "}
                <span className="text-gray-500">Connectivity</span>
              </h2>
              <p className="text-gray-600 text-lg max-w-2xl mx-auto leading-relaxed">
                Break free from traditional boundaries. Work from anywhere, at the comfort of your own studio apartment. Perfect for Nomads and Travellers.
              </p>
            </div>
            <div className="w-full">
              <WorldMap dots={mapDots} lineColor="#0ea5e9" />
            </div>
          </div>
        </div>
  )
}

export default Map
