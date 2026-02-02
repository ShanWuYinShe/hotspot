/**
 * 服务端钩子
 * 
 * 功能：
 * 1. 服务器端路由保护
 * 2. 认证检查（从 cookie 读取 token）
 * 3. 重定向未授权用户
 */

import type { Handle } from '@sveltejs/kit';
import { redirect } from '@sveltejs/kit';

/**
 * 检查路径是否需要认证
 */
function requiresAuth(pathname: string): boolean {
	// 受保护的路由（需要登录）
	const protectedPaths = [
		'/profile',
		'/dashboard',
		'/chat',
		'/groups',
		'/moments',
		'/notifications',
		'/settings',
		'/followers',
		'/follows',
		'/friends'
	];

	return protectedPaths.some(path => pathname.startsWith(path));
}

/**
 * 检查路径是否需要管理员权限
 */
function requiresAdmin(pathname: string): boolean {
	return pathname.startsWith('/admin');
}

export const handle: Handle = async ({ event, resolve }) => {
	const pathname = event.url.pathname;

	// 从 cookie 中读取 refresh_token（用于判断是否已登录）
	const refreshToken = event.cookies.get('refresh_token');
	const hasAuth = !!refreshToken;

	// 检查管理员路由
	if (requiresAdmin(pathname)) {
		if (!hasAuth) {
			// 未登录，重定向到首页
			throw redirect(302, '/?redirect=' + encodeURIComponent(pathname));
		}
		// 注意：这里只检查 token 是否存在，不验证管理员权限
		// 实际的权限验证由后端 API 处理
	}

	// 检查受保护路由
	if (requiresAuth(pathname)) {
		if (!hasAuth) {
			// 未登录，重定向到首页
			throw redirect(302, '/?redirect=' + encodeURIComponent(pathname));
		}
	}

	// 继续处理请求
	const response = await resolve(event);
	return response;
};
