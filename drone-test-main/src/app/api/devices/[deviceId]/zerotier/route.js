import { auth } from '@clerk/nextjs/server';

export async function GET(request, { params }) {
  const { deviceId } = await params;
  
  const { getToken } = await auth();
  const token = await getToken();
  
  const backendUrl = `${process.env.BACKEND_URL}/api/v1/devices/${deviceId}/zerotier`;
  
  try {
    const response = await fetch(backendUrl, {
      headers: {
        'Authorization': `Bearer ${token}`,
      }
    });
    
    if (!response.ok) {
      const text = await response.text();
      console.error('Backend error:', response.status, text);
      return Response.json({ error: text }, { status: response.status });
    }
    
    const data = await response.json();
    return Response.json(data, { status: response.status });
  } catch (error) {
    console.error('Error:', error);
    return Response.json({ error: error.message }, { status: 500 });
  }
}