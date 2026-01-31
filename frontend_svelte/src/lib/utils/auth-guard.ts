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

