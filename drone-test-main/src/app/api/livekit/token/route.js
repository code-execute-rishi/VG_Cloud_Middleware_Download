import { auth } from '@clerk/nextjs/server';
import { NextResponse } from 'next/server';

export async function POST(request) {
  console.log('========================================');
  console.log('🟢 LiveKit Token API Route: REQUEST STARTED');
  console.log('🟢 Request URL:', request.url);
  console.log('🟢 Request Method:', request.method);
  console.log('========================================');

  try {
    console.log('🟢 Getting auth token...');
    const { getToken } = await auth();
    const token = await getToken();
    
    if (!token) {
      console.log('❌ No token available');
      return NextResponse.json(
        { error: 'Unauthorized - No token available' },
        { status: 401 }
      );
    }
    console.log('✅ Token obtained:', token.substring(0, 20) + '...');

    console.log('🟢 Parsing request body...');
    const body = await request.json();
    console.log('✅ Request body:', JSON.stringify(body, null, 2));

    if (!body.deviceId) {
      console.log('❌ deviceId is missing in request body');
      return NextResponse.json(
        { error: 'deviceId is required' },
        { status: 400 }
      );
    }

    const backendUrl = `${process.env.BACKEND_URL}/api/v1/livekit/tokens`;
    console.log('🟢 Backend URL:', backendUrl);
    console.log('🟢 BACKEND_URL env:', process.env.BACKEND_URL);
    console.log('🟢 deviceId:', body.deviceId);

    console.log('🟢 Making POST request to backend...');
    const response = await fetch(backendUrl, {
      method: 'POST',
      headers: {
        'Authorization': `Bearer ${token}`,
        'Content-Type': 'application/json',
      },
      body: JSON.stringify({
        device_id: body.deviceId
      })
    });

    console.log('✅ Backend response received');
    console.log('   - Status:', response.status);
    console.log('   - Status Text:', response.statusText);
    console.log('   - Headers:', Object.fromEntries(response.headers.entries()));

    let data;
    const contentType = response.headers.get('content-type');
    console.log('🟢 Response Content-Type:', contentType);

    if (contentType && contentType.includes('application/json')) {
      data = await response.json();
      console.log('✅ Response data (JSON):', JSON.stringify(data, null, 2));
    } else {
      const text = await response.text();
      console.log('⚠️ Response is not JSON. Raw text:', text);
      data = { error: 'Invalid response from server', details: text };
    }

    console.log('🟢 Returning response to client');
    console.log('   - Status:', response.status);
    console.log('   - Data:', data);
    console.log('========================================');
    
    return NextResponse.json(data, { status: response.status });

  } catch (error) {
    console.error('========================================');
    console.error('❌ LiveKit Token API Route: ERROR OCCURRED');
    console.error('❌ Error name:', error.name);
    console.error('❌ Error message:', error.message);
    console.error('❌ Error stack:', error.stack);
    console.error('========================================');

    return NextResponse.json(
      { 
        error: 'Failed to get LiveKit token',
        details: error.message,
        type: error.name
      },
      { status: 500 }
    );
  }
}