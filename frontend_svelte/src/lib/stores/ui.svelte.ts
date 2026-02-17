/**
 * UI 状态管理
 * 管理模态框、跳转状态等
 */

import { browser } from '$app/environment';
import { goto } from '$app/navigation';

const REDIRECT_KEY = 'hotspot_redirect_url';

class UIStore {
	// 登录模态框状态
	#showLoginModal = $state(false);
	#loginModalCallback = $state<(() => void) | null>(null);
	
	// ==================== 登录模态框 ====================
	
	/** 是否显示登录模态框 */
	showLoginModal = $derived(this.#showLoginModal);
	
	/** 登录成功后的回调函数 */
	loginModalCallback = $derived(this.#loginModalCallback);
	
	/**
	 * 打开登录模态框
	 * @param callback 登录成功后的回调
	 */
	openLoginModal(callback?: () => void) {
		this.#showLoginModal = true;
		this.#loginModalCallback = callback ?? null;
	}
	
	closeLoginModal() {
		this.#showLoginModal = false;
		this.#loginModalCallback = null;
	}
	
	/** 登录成功后执行回调 */
	onLoginSuccess() {
		const callback = this.#loginModalCallback;
		const redirectUrl = this.hasRedirectUrl() ? this.getAndClearRedirectUrl() : null;
		this.closeLoginModal();
		if (callback) {
			callback();
		}
		if (redirectUrl && browser) {
			const currentUrl = window.location.pathname + window.location.search;
			if (redirectUrl !== currentUrl) {
				goto(redirectUrl);
			}
		}
	}
	
	// ==================== 重定向管理 ====================
	
	/** 保存当前 URL，用于登录后返回 */
	saveRedirectUrl(url?: string) {
		if (!browser) return;
		const targetUrl = url ?? window.location.pathname + window.location.search;
		// 不保存认证相关页面
		if (!targetUrl.startsWith('/auth')) {
			try { sessionStorage.setItem(REDIRECT_KEY, targetUrl); } catch { /* 隐私模式或配额满 */ }
		}
	}
	
	/** 获取并清除保存的重定向 URL */
	getAndClearRedirectUrl(): string {
		if (!browser) return '/';
		try {
			const url = sessionStorage.getItem(REDIRECT_KEY);
			sessionStorage.removeItem(REDIRECT_KEY);
			return url || '/';
		} catch {
			return '/';
		}
	}
	
	/** 检查是否有保存的重定向 URL */
	hasRedirectUrl(): boolean {
		if (!browser) return false;
		try { return !!sessionStorage.getItem(REDIRECT_KEY); } catch { return false; }
	}
}

export const uiStore = new UIStore();
