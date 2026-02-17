/**
 * 服务端钩子
 *
 * 注意：
 * Spec 规定 access_token 的 Cookie Path=/api、refresh_token 的 Path=/api/v1/auth/refresh_token，
 * 因此页面请求（如 /profile、/admin）不会携带这两个 Cookie，服务端无法判断登录状态。
 *
 * 认证保护完全由客户端实现：
 * - (protected)/+layout.svelte：检查 authStore.isAuthenticated
 * - (admin)/+layout.svelte：检查 authStore + 管理员权限
 */

import type { Handle } from '@sveltejs/kit';

export const handle: Handle = async ({ event, resolve }) => {
	const response = await resolve(event);
	return response;
};
