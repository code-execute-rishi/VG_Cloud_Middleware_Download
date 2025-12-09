"use client"
import React from 'react'
import Navbar from '@/components/Navbar'
import { useUser } from '@clerk/nextjs'
import { HeroGeometric } from '@/components/ui/shape-landing-hero'
import FeatureSection from '@/components/FeatureSection'
import Card from '@/components/DisplayCard'
import Map from '@/components/Map'

const page = () => {
  const { user } = useUser();
  console.log(user);
  return (
    <div>
      <Navbar />
        <HeroGeometric badge="Kokonut UI"
            title1 = "Elevate Your"
            title2 = " Drone Operations" 
        />
        <FeatureSection />
        <Map />
        {/* <Card /> */}
    </div>
  )
}

export default page
