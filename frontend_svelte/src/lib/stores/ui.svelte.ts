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
	#showCreateSheet = $state(false);
	#showCreateMoment = $state(false);
	#showCreateGroup = $state(false);
	#showSearchModal = $state(false);
	#showAuthPrompt = $state(false);

	showCreateSheet = $derived(this.#showCreateSheet);
	showCreateMoment = $derived(this.#showCreateMoment);
	showCreateGroup = $derived(this.#showCreateGroup);
	showSearchModal = $derived(this.#showSearchModal);
	showAuthPrompt = $derived(this.#showAuthPrompt);

	/**
	 * 显示内联登录提示，用户手动点击后跳转到登录页
	 */
	openLoginModal(_callback?: () => void) {
		this.#showAuthPrompt = true;
	}

	openLoginModalWithRedirect(_options: { url?: string; onSuccess?: () => void } = {}) {
		this.#showAuthPrompt = true;
	}

	/** 关闭内联登录提示 */
	closeAuthPrompt() {
		this.#showAuthPrompt = false;
	}

	/** 手动点击登录按钮后，跳转到登录页 */
	navigateToAuth(redirectUrl?: string) {
		if (!browser) return;
		this.#showAuthPrompt = false;
		const fallbackUrl = `${window.location.pathname}${window.location.search}${window.location.hash}`;
		const target = normalizeRedirectPath(redirectUrl ?? fallbackUrl) ?? '/';
		const authUrl = `/auth?redirect=${encodeURIComponent(target)}`;
		goto(authUrl);
	}

	/** 登录成功后跳转到之前保存的页面 */
	onLoginSuccess() {
		if (!browser) return;
		const redirectUrl = this.hasRedirectUrl() ? this.getAndClearRedirectUrl() : null;
		if (redirectUrl) {
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
		openSearchModal() {
			this.#showSearchModal = true;
		}

		closeSearchModal() {
			this.#showSearchModal = false;
		}
}

export const uiStore = new UIStore();
