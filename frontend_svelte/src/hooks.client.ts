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
import { authService } from '$lib/api/services/auth';
import { initApiClient, isSuccess } from '$lib/api/client';
import { websocketStore } from '$lib/websocket';

/**
 * Token 刷新管理器
 */
class TokenRefreshManager {
	private checkInterval: ReturnType<typeof setInterval> | null = null;
	private readonly CHECK_INTERVAL = 60 * 1000; // 每分钟检查一次
	private readonly REFRESH_THRESHOLD = 3; // 提前 3 分钟刷新（30 分钟过期时间下更合理）
	private isRefreshing = false;

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
		// 未登录或正在刷新，跳过
		if (!authStore.isAuthenticated || this.isRefreshing) {
			return;
		}

		// 检查 Token 是否即将过期
		if (authStore.isTokenExpiringSoon(this.REFRESH_THRESHOLD)) {
			if (dev) console.log(`[TokenRefresh] Token 将在 ${this.REFRESH_THRESHOLD} 分钟内过期，开始刷新...`);
			await this.refreshToken();
		}
	}

	/**
	 * 刷新 Token
	 */
	private async refreshToken() {
		if (this.isRefreshing) return;

		this.isRefreshing = true;

		try {
			const response = await authService.refreshToken({
				client_type: 'web'
				// Web 端不传 refresh_token（从 HttpOnly Cookie 读取）
			});

			if (isSuccess(response) && response.data) {
				// 更新 Token
				authStore.updateToken(
					response.data.access_token,
					response.data.access_token_expires_at
				);
				if (dev) console.log('[TokenRefresh] Token 刷新成功');
			} else {
				if (dev) console.error('[TokenRefresh] Token 刷新失败:', response.message);
				// 刷新失败，清除 Token 并登出
				authStore.clearToken();
			}
		} catch (error) {
			if (dev) console.error('[TokenRefresh] Token 刷新异常:', error);
			authStore.clearToken();
		} finally {
			this.isRefreshing = false;
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
