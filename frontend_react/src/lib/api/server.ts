import { cookies } from 'next/headers';

import { ApiResponse } from '@/types/api';

type ServerRequestConfig = {
  skipAuth?: boolean;
  tags?: string[];
  revalidate?: number;
};

// Server-side API base URL - must be absolute for SSR
const API_BASE_URL = process.env.API_BASE_URL || 'http://localhost:8888';

const getAccessToken = async () => {
  const cookieStore = await cookies();
  return cookieStore.get('access_token')?.value;
};

export async function serverPost<T>(
  endpoint: string,
  body?: unknown,
  config: ServerRequestConfig = {},
): Promise<ApiResponse<T>> {
  const headers: Record<string, string> = {
    'Content-Type': 'application/json',
  };

  if (!config.skipAuth) {
    const token = await getAccessToken();
    if (token) {
      headers.Authorization = `Bearer ${token}`;
    }
  }

  // Use absolute URL for server-side fetches
  const url = endpoint.startsWith('http') ? endpoint : `${API_BASE_URL}${endpoint}`;

  const response = await fetch(url, {
    method: 'POST',
    headers,
    body: JSON.stringify(body ?? {}),
    cache: 'no-store',
    next: config.tags || config.revalidate ? { tags: config.tags, revalidate: config.revalidate } : undefined,
  });

  try {
    return await response.json();
  } catch {
    throw new Error('Invalid JSON response');
  }
}

export const isServerSuccess = <T,>(response: ApiResponse<T>) => response.code === 0;
