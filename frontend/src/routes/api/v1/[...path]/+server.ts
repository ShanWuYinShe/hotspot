import { dev } from '$app/environment';
import type { RequestHandler } from './$types';
import { env } from '$env/dynamic/private';

export const POST: RequestHandler = async ({ request, url }) => {
  if (dev && env['USE_MOCK'] === 'true') {
    // 显式设置 USE_MOCK=true 时启用 mock handler，默认不拦截（避免脏数据）
    // mock 是动态 import，USE_MOCK 未设置时构建不打包 mocks/ 内容
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
