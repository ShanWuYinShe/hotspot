import { dev } from '$app/environment';
import type { RequestHandler } from './$types';
import { env } from '$env/dynamic/private';

export const POST: RequestHandler = async ({ request, url }) => {
  if (dev) {
    // 动态 import：mock 仅在 dev 分支加载，生产构建不打包 mocks/（避免假数据进入线上 SSR bundle）
    const { handleMockRequest } = await import('$lib/../mocks/handler');
    const response = await handleMockRequest(request);
    if (response) {
      return response;
    }
  }

  const backendUrl = env['BACKEND_URL'] || 'http://backend:8081';
  const target = new URL(url.pathname + url.search, backendUrl);
  const headers = new Headers(request.headers);
  headers.delete('host');
  headers.set('Content-Type', 'application/json');
  headers.set('X-Proxy-Debug', `dev=${dev}, backend=${backendUrl}`);

  const body = await request.text();

  const resp = await fetch(target.toString(), {
    method: 'POST',
    headers,
    body
  });

  return new Response(resp.body, {
    status: resp.status,
    headers: resp.headers
  });
};
