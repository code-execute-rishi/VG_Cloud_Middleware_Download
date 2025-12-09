import { auth } from '@clerk/nextjs/server';
import { NextResponse } from 'next/server';

export async function GET() {
  try {
    const { getToken } = await auth();
    const token = await getToken();

    if (!token) {
      return NextResponse.json(
        { error: 'Unauthorized - No token available' },
        { status: 401 }
      );
    }


    const backendUrl = process.env.BACKEND_URL;
    const response = await fetch(`${backendUrl}/api/v1/devices`, {
      method: 'GET',
      headers: {
        'Authorization': `Bearer ${token}`,
        'Content-Type': 'application/json',
      },
    });

    
    let data;
    const contentType = response.headers.get('content-type');
    
    if (contentType && contentType.includes('application/json')) {
      data = await response.json();
    } else {
      const text = await response.text();
      data = { error: 'Invalid response from server', details: text };
    }


    // Handle null response - return empty array instead
    if (data === null) {
      return NextResponse.json([], { status: 200 });
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
        error: 'Failed to fetch devices',
        details: error.message,
        type: error.name
      },
      { status: 500 }
    );
  }
}