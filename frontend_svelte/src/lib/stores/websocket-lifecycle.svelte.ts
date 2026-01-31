/**
 * WebSocket 生命周期管理
 * 在应用层面管理 WebSocket 连接的生命周期
 */

import { authStore } from './auth.svelte';
import { websocketStore } from '$lib/websocket';
import { browser } from '$app/environment';

/**
 * 初始化 WebSocket 连接（在应用启动时调用）
 */
export function initializeWebSocket() {
	if (!browser) return;

	// 监听登录状态变化
	$effect(() => {
		if (authStore.isAuthenticated && authStore.currentAccount?.access_token) {
			// 用户已登录，连接 WebSocket
			if (websocketStore.connectionState === 'disconnected') {
				console.log('[App] 用户已登录，连接 WebSocket');
				websocketStore.connect();
			}
		} else {
			// 用户未登录，断开 WebSocket
			if (websocketStore.connectionState !== 'disconnected') {
				console.log('[App] 用户已登出，断开 WebSocket');
				websocketStore.disconnect();
			}
		}
	});
}
