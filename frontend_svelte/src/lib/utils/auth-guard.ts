/**
 * 认证守卫工具
 */

import { browser } from '$app/environment';
import { goto } from '$app/navigation';
import { authStore } from '$lib/stores/auth.svelte';
import { uiStore } from '$lib/stores/ui.svelte';

/**
 * 检查是否需要认证
 * @param redirectTo 未认证时重定向的路径
 */
export function requireAuth(redirectTo?: string): boolean {
	if (!browser) return true; // SSR 时跳过检查

	if (!authStore.isAuthenticated) {
		if (redirectTo) {
			uiStore.saveRedirectUrl(redirectTo);
		} else {
			uiStore.saveRedirectUrl();
		}
		uiStore.openLoginModal();
		return false;
	}

	return true;
}

/**
 * 检查是否需要访客状态
 * @param redirectTo 已认证时重定向的路径
 */
export function requireGuest(redirectTo = '/'): boolean {
	if (!browser) return true;

	if (authStore.isAuthenticated) {
		goto(redirectTo);
		return false;
	}

	return true;
}

/**
 * 获取重定向 URL（带 redirect 参数）
 */
export function getAuthRedirectUrl(currentPath: string): string {
	return currentPath;
}
