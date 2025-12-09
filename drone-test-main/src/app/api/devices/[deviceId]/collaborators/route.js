import { auth } from '@clerk/nextjs/server';

export async function POST(request, { params }) {
  const { getToken } = await auth();
  const token = await getToken();
  const { deviceId } = await params;
  const body = await request.json();
  
  const response = await fetch(`${process.env.BACKEND_URL}/api/v1/devices/${deviceId}/collaborators`, {
    method: 'POST',
    headers: {
      'Authorization': `Bearer ${token}`,
      'Content-Type': 'application/json',
    },
    body: JSON.stringify(body)
  });
  
  const data = await response.json();
  return Response.json(data, { status: response.status });
}