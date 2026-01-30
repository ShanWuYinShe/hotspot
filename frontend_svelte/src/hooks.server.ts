/**
 * 服务端钩子
 */

import type { Handle } from '@sveltejs/kit';

export const handle: Handle = async ({ event, resolve }) => {
	// 可以在这里添加服务端认证检查
	// 例如：从 cookie 中读取 token 并验证

	const response = await resolve(event);
	return response;
};
