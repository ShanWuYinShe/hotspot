import { dev } from '$app/environment';
import { json } from '@sveltejs/kit';
import type { RequestHandler } from './$types';
import { handleMockRequest } from '$lib/../mocks/handler';

export const POST: RequestHandler = async ({ request }) => {
	if (!dev) {
		return json({ code: -1, message: 'Not available in production' }, { status: 404 });
	}

	const response = await handleMockRequest(request);
	if (response) {
		return response;
	}

	return json({ code: 40400, message: 'No mock handler' }, { status: 404 });
};
