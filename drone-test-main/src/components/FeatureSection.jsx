import React from 'react'
import { FeaturesSectionWithHoverEffects } from './feature-section-with-hover-effects'

const FeatureSection = () => {
    return (
        <div className="min-h-screen w-full bg-[#040303] flex flex-col justify-center items-center py-20 pt-30 text-white">
            <div className='py-5'>
                <div className='text-5xl max-w-5xl mb-5 text-center font-bold'>Complete Platform for Drone Monitoring & Remote Operations</div>
                <div className='text-center text-xl'>Live location, MAVLink data, network health, and secure remote management — all in one interface.</div>
            </div>
            <div className="w-full">
                <FeaturesSectionWithHoverEffects />
            </div>
        </div>
    )
}

export default FeatureSection
