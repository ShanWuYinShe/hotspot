/**
 * SvelteKit 客户端钩子
 * 
 * 功能：
 * 1. 初始化 API 客户端
 * 2. 主动刷新 Token - 定时检查并刷新即将过期的 token
 * 3. WebSocket 自动管理 - websocketStore 内部自动监听认证状态
 */

import { browser, dev } from '$app/environment';
import { authStore } from '$lib/stores/auth.svelte';
import { configStore } from '$lib/stores/config.svelte';
import { initApiClient, ensureRefreshToken } from '$lib/api/client';
import { websocketStore } from '$lib/websocket';

/**
 * Token 刷新管理器
 * 
 * 与 client.ts 的被动刷新（401 触发）互补：
 * - 本管理器：主动定时检查，提前刷新即将过期的 Token
 * - client.ts：被动刷新，API 请求收到 Token 错误码时触发
 * - 两者共享同一个 ensureRefreshToken 锁，避免并发刷新
 */
class TokenRefreshManager {
	private checkInterval: ReturnType<typeof setInterval> | null = null;
	private readonly CHECK_INTERVAL = 60 * 1000; // 每分钟检查一次
	private readonly REFRESH_THRESHOLD = 3; // 提前 3 分钟刷新

	/**
	 * 启动定时检查
	 */
	start() {
		if (!browser || this.checkInterval) return;

		if (dev) console.log('[TokenRefresh] 启动定时检查器（每分钟检查一次）');

		// 立即检查一次
		this.checkAndRefresh();

		// 定时检查
		this.checkInterval = setInterval(() => {
			this.checkAndRefresh();
		}, this.CHECK_INTERVAL);
	}

	/**
	 * 停止定时检查
	 */
	stop() {
		if (this.checkInterval) {
			clearInterval(this.checkInterval);
			this.checkInterval = null;
			if (dev) console.log('[TokenRefresh] 停止定时检查器');
		}
	}

	/**
	 * 检查并刷新 Token
	 */
	private async checkAndRefresh() {
		if (!authStore.isAuthenticated) return;

		if (authStore.isTokenExpiringSoon(this.REFRESH_THRESHOLD)) {
			if (dev) console.log(`[TokenRefresh] Token 将在 ${this.REFRESH_THRESHOLD} 分钟内过期，开始刷新...`);
			// 复用 client.ts 的 ensureRefreshToken，共享并发锁
			const success = await ensureRefreshToken();
			if (dev) console.log(`[TokenRefresh] 刷新${success ? '成功' : '失败'}`);
		}
	}
}

// ==================== 单例 ====================

const tokenRefreshManager = new TokenRefreshManager();

// ==================== SvelteKit Hooks ====================

/**
 * 客户端初始化
 */
if (browser) {
	// 初始化 API 客户端
	initApiClient({
		baseUrl: '', // 使用相对路径（与前端同域）
		getToken: () => authStore.token,
		setToken: (token, expiresAt) => authStore.updateToken(token, expiresAt),
		clearToken: () => authStore.clearToken()
	});

	// 拉取服务端 ClientConfig（无需认证，尽早执行）
	configStore.fetch().then(() => {
		if (dev) console.log('[App] ClientConfig 已加载');
	});

	// 启动 Token 刷新管理器
	tokenRefreshManager.start();

	// 初始化 WebSocket Store（构造函数中自动管理生命周期）
	if (dev) console.log('[App] WebSocket Store 已初始化（自动管理连接）');

	// 页面卸载时清理
	window.addEventListener('beforeunload', () => {
		tokenRefreshManager.stop();
		websocketStore.disconnect();
	});
}

// 导出（供测试或手动控制）
export { tokenRefreshManager };
