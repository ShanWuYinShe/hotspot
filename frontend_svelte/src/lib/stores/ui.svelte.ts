/**
 * UI 状态管理
 * 管理模态框、跳转状态等
 */

import { browser } from '$app/environment';
import { goto } from '$app/navigation';

const REDIRECT_KEY = 'hotspot_redirect_url';
const DEFAULT_REDIRECT_PATH = '/';

function safelyDecodeRedirectPath(value: string): string {
	try {
		return decodeURIComponent(value);
	} catch {
		return value;
	}
}

function isAuthPath(pathname: string): boolean {
	return pathname === '/auth' || pathname.startsWith('/auth/');
}

export function normalizeRedirectPath(url?: string | null): string | null {
	if (!url) return null;

	const trimmed = url.trim();
	if (!trimmed) return null;
	if (/[^\S ]|[\u0000-\u001F\u007F]/.test(trimmed)) return null;

	const decoded = safelyDecodeRedirectPath(trimmed).trim();
	if (!decoded.startsWith('/')) return null;
	if (decoded.startsWith('//') || decoded.startsWith('/\\')) return null;
	if (decoded.includes('\\')) return null;

	const lowered = decoded.toLowerCase();
	if (lowered.startsWith('/javascript:')) return null;
	if (lowered.startsWith('/http:') || lowered.startsWith('/https:')) return null;

	try {
		const normalized = new URL(decoded, 'https://hotspot.local');
		if (normalized.origin !== 'https://hotspot.local') return null;
		if (isAuthPath(normalized.pathname)) return null;
		return `${normalized.pathname}${normalized.search}${normalized.hash}`;
	} catch {
		return null;
	}
}

class UIStore {
	// 登录模态框状态
	#showLoginModal = $state(false);
	#loginModalCallback = $state<(() => void) | null>(null);
	#showCreateSheet = $state(false);
	#showCreateMoment = $state(false);
	#showCreateGroup = $state(false);

	// ==================== 登录模态框 ====================

	/** 是否显示登录模态框 */
	showLoginModal = $derived(this.#showLoginModal);

	/** 登录成功后的回调函数 */
	loginModalCallback = $derived(this.#loginModalCallback);
	showCreateSheet = $derived(this.#showCreateSheet);
	showCreateMoment = $derived(this.#showCreateMoment);
	showCreateGroup = $derived(this.#showCreateGroup);
	
	/**
	 * 打开登录模态框
	 * @param callback 登录成功后的回调
	 */
	openLoginModal(callback?: () => void) {
		this.#showLoginModal = true;
		this.#loginModalCallback = callback ?? null;
	}

	openLoginModalWithRedirect(options: { url?: string; onSuccess?: () => void } = {}) {
		this.saveRedirectUrl(options.url);
		this.openLoginModal(options.onSuccess);
	}

	saveCurrentRedirectUrl() {
		if (!browser) return;
		this.saveRedirectUrl(
			`${window.location.pathname}${window.location.search}${window.location.hash}`
		);
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
			const currentUrl = `${window.location.pathname}${window.location.search}${window.location.hash}`;
			if (redirectUrl !== currentUrl) {
				goto(redirectUrl);
			}
		}
	}
	
	// ==================== 重定向管理 ====================
	
	/** 保存当前 URL，用于登录后返回 */
	saveRedirectUrl(url?: string) {
		if (!browser) return;
		const fallbackUrl = `${window.location.pathname}${window.location.search}${window.location.hash}`;
		const targetUrl = normalizeRedirectPath(url ?? fallbackUrl);
		if (!targetUrl) {
			this.clearRedirectUrl();
			return;
		}
		try {
			sessionStorage.setItem(REDIRECT_KEY, targetUrl);
		} catch {
			// 隐私模式或配额满
		}
	}

	/** 清除保存的重定向 URL */
	clearRedirectUrl() {
		if (!browser) return;
		try {
			sessionStorage.removeItem(REDIRECT_KEY);
		} catch {
			// 忽略存储错误
		}
	}

	/** 获取并清除保存的重定向 URL */
	getAndClearRedirectUrl(): string {
		if (!browser) return DEFAULT_REDIRECT_PATH;
		try {
			const url = sessionStorage.getItem(REDIRECT_KEY);
			sessionStorage.removeItem(REDIRECT_KEY);
			return normalizeRedirectPath(url) ?? DEFAULT_REDIRECT_PATH;
		} catch {
			return DEFAULT_REDIRECT_PATH;
		}
	}

	/** 检查是否有保存的重定向 URL */
	hasRedirectUrl(): boolean {
		if (!browser) return false;
		try {
			return normalizeRedirectPath(sessionStorage.getItem(REDIRECT_KEY)) !== null;
		} catch {
			return false;
		}
	}

	openCreateSheet() {
		this.#showCreateSheet = true;
	}

	closeCreateSheet() {
		this.#showCreateSheet = false;
	}

	openCreateMoment() {
		this.#showCreateSheet = false;
		this.#showCreateMoment = true;
	}

	closeCreateMoment() {
		this.#showCreateMoment = false;
	}

	openCreateGroup() {
		this.#showCreateSheet = false;
		this.#showCreateGroup = true;
	}

	closeCreateGroup() {
		this.#showCreateGroup = false;
	}
}

export const uiStore = new UIStore();
