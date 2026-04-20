import { dev } from '$app/environment';
import { json } from '@sveltejs/kit';
import type { RequestHandler } from './$types';
import { handleMockRequest } from '$lib/../mocks/handler';
import { env } from '$env/dynamic/private';

export const POST: RequestHandler = async ({ request, url }) => {
	if (dev) {
		const response = await handleMockRequest(request);
		if (response) {
			return response;
		}
		return json({ code: 40400, message: 'No mock handler' }, { status: 404 });
	}

	const backendUrl = env.BACKEND_URL || 'http://backend:8081';
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
