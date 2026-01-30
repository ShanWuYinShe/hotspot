/**
 * 客户端钩子 - 初始化 API 客户端
 */

import { initApiClient } from '$lib/api/client';
import { authStore } from '$lib/stores/auth.svelte';

// 初始化 API 客户端
// 使用 Vite 代理，无需指定 baseUrl
initApiClient({
	baseUrl: '', // 代理配置在 vite.config.ts 中
	getToken: () => authStore.token,
	setToken: (token, expiresAt) => authStore.updateToken(token, expiresAt),
	clearToken: () => authStore.clearToken()
});
