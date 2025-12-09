import { auth } from '@clerk/nextjs/server';
import { NextResponse } from 'next/server';

export async function POST(request) {
  try {
    const { getToken } = await auth();
    const token = await getToken();

    if (!token) {
      return NextResponse.json(
        { error: 'Unauthorized - No token available' },
        { status: 401 }
      );
    }

    const body = await request.json();

    const backendUrl = process.env.BACKEND_URL;
    const response = await fetch(`${backendUrl}/api/v1/devices/claim`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        'Authorization': `Bearer ${token}`,
      },
      body: JSON.stringify(body),
    });


    
    let data;
    const contentType = response.headers.get('content-type');
    
    if (contentType && contentType.includes('application/json')) {
      data = await response.json();
    } else {
      const text = await response.text();
      data = { error: 'Invalid response from server', details: text };
    }



    return NextResponse.json(data, { status: response.status });

  } catch (error) {
    console.error('API route error:', error);
    console.error('Error details:', {
      message: error.message,
      name: error.name,
      stack: error.stack
    });

    return NextResponse.json(
      { 
        error: 'Failed to claim device',
        details: error.message,
        type: error.name
      },
      { status: 500 }
    );
  }
}