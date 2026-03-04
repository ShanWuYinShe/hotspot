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
import { initApiClient, ensureRefreshToken, isSuccess } from '$lib/api/client';
import { authService } from '$lib/api/services/auth';
import { usersService } from '$lib/api/services/users';
import { websocketStore } from '$lib/websocket';
import { initKeyboardFocusManager } from '$lib/utils/keyboard-focus';

/**
 * Token 刷新管理器
 * 
 * 与 client.ts 的被动刷新（code=10100 触发）互补：
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

// ==================== 会话恢复 ====================

async function restoreSessionFromServer() {
	authStore.startRestoring();

	try {
		let statusRes = await authService.checkStatus();
		// check_status 是 token_optional，access_token 过期时可能直接返回 is_valid=false，
		// 不会触发 request 层 code=10100 的自动刷新。恢复阶段显式补一次 refresh。
		if ((!isSuccess(statusRes) || !statusRes.data?.is_valid) && authStore.currentUserId) {
			const refreshed = await ensureRefreshToken();
			if (refreshed) {
				statusRes = await authService.checkStatus();
			}
		}
		if (!isSuccess(statusRes) || !statusRes.data?.is_valid) {
			// 仅标记会话失效，保留本地账户列表，避免“临时用户直接丢失”
			authStore.invalidateSession();
			return;
		}

		const profileRes = await usersService.getMyProfile();
		if (!isSuccess(profileRes) || !profileRes.data) {
			authStore.invalidateSession();
			return;
		}

		const expiresAt = statusRes.data.expires_at ?? new Date(Date.now() + 15 * 60 * 1000).toISOString();

		authStore.login({
			user_id: profileRes.data.user_id,
			user_type: profileRes.data.user_type,
			access_token_expires_at: expiresAt
		});
		authStore.setProfile(profileRes.data);
	} catch (error) {
		if (dev) console.warn('[App] 自动恢复登录状态失败:', error);
		authStore.invalidateSession();
	} finally {
		authStore.finishRestoring();
	}
}

// ==================== 单例 ====================

const tokenRefreshManager = new TokenRefreshManager();

// ==================== SvelteKit Hooks ====================

/**
 * 客户端初始化
 */
if (browser) {
  initKeyboardFocusManager(window);
  // 初始化 API 客户端
  // spec: Web 端禁止使用 Authorization Header，token 通过 HttpOnly Cookie 传递
  initApiClient({
		baseUrl: '', // 使用相对路径（与前端同域）
		onAccountBanned: (message) => {
			// spec 10201: 强制登出 + 显示封禁页
			authStore.logoutAll();
			// 使用 setTimeout 避免在请求处理中同步导航
			setTimeout(() => {
				window.location.href = `/banned?reason=${encodeURIComponent(message)}`;
			}, 0);
		},
		onRefreshSuccess: () => {
			// Token 已通过 Set-Cookie 更新，获取新的过期时间
			authService.checkStatus().then((res) => {
				if (isSuccess(res) && res.data.expires_at) {
					authStore.updateExpiresAt(res.data.expires_at);
				} else {
					// checkStatus 返回非成功码，降级：用当前时间 + 15 分钟作为预估过期时间
					if (dev) console.warn('[App] checkStatus 返回失败，使用预估过期时间');
					const fallbackExpiry = new Date(Date.now() + 15 * 60 * 1000).toISOString();
					authStore.updateExpiresAt(fallbackExpiry);
				}
			}).catch((error) => {
				// 网络错误等，降级：用当前时间 + 15 分钟作为预估过期时间
				if (dev) console.warn('[App] checkStatus 请求失败，使用预估过期时间:', error);
				const fallbackExpiry = new Date(Date.now() + 15 * 60 * 1000).toISOString();
				authStore.updateExpiresAt(fallbackExpiry);
			});
		},
		onRefreshFail: () => {
			// 刷新失败时仅清空会话态，保留账户元数据，便于重新登录恢复
			authStore.invalidateSession();
		}
	});

	// 拉取服务端 ClientConfig（无需认证，尽早执行）
	configStore.fetch().then(() => {
		if (dev) console.log('[App] ClientConfig 已加载');
	}).catch((error) => {
		if (dev) console.error('[App] ClientConfig 加载失败:', error);
	});

	// 恢复已保存的登录态（需要服务端确认 Cookie 有效）
	restoreSessionFromServer();

	// 启动 Token 刷新管理器
	tokenRefreshManager.start();

	// 初始化 WebSocket Store（构造函数中自动管理生命周期）
	if (dev) console.log('[App] WebSocket Store 已初始化（自动管理连接）');

	// 页面卸载时彻底清理所有资源
	// destroy() 会移除全局事件监听、销毁 $effect.root、清空所有 WebSocket 事件监听器
	window.addEventListener('beforeunload', () => {
		tokenRefreshManager.stop();
		websocketStore.destroy();
	});
}

// 导出（供测试或手动控制）
export { tokenRefreshManager };
