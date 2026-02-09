/**
 * 认证状态管理 (Svelte 5 Runes)
 *
 * 特性：
 * - 多账户支持
 * - Token 自动刷新
 * - localStorage 持久化
 */

import { browser } from '$app/environment';
import type { AuthData, MyCard } from '$lib/types/api';

const STORAGE_KEY = 'hotspot_auth';

interface AccountInfo {
	user_id: string;
	nickname: string;
	avatar_url?: string;
	access_token: string;
	access_token_expires_at: string;
	refresh_token?: string;
	refresh_token_expires_at?: string;
	is_locked?: boolean;
}

interface AuthState {
	accounts: AccountInfo[];
	currentUserId: string | null;
}

function loadFromStorage(): AuthState {
	if (!browser) {
		return { accounts: [], currentUserId: null };
	}

	try {
		const stored = localStorage.getItem(STORAGE_KEY);
		if (stored) {
			return JSON.parse(stored);
		}
	} catch {
		// 忽略解析错误
	}

	return { accounts: [], currentUserId: null };
}

function saveToStorage(state: AuthState) {
	if (!browser) return;

	try {
		localStorage.setItem(STORAGE_KEY, JSON.stringify(state));
	} catch {
		// 忽略存储错误
	}
}

// ==================== Auth Store Class ====================

class AuthStore {
	#accounts = $state.raw<AccountInfo[]>([]);
	#currentUserId = $state<string | null>(null);
	#profile = $state<MyCard | null>(null);
	#initialized = $state(false);

	constructor() {
		// 在浏览器环境中初始化
		if (browser) {
			const stored = loadFromStorage();
			this.#accounts = stored.accounts;
			this.#currentUserId = stored.currentUserId;
			this.#initialized = true;
		}
	}

	// ==================== Derived State（所有公开属性）====================

	/** 所有账户列表 */
	accounts = $derived(this.#accounts);

	/** 当前用户 ID */
	currentUserId = $derived(this.#currentUserId);

	/** 用户资料卡片 */
	profile = $derived(this.#profile);

	/** 是否已初始化 */
	initialized = $derived(this.#initialized);

	/** 当前账户信息 */
	currentAccount = $derived(
		this.#accounts.find((a) => a.user_id === this.#currentUserId && !a.is_locked)
	);

	/** 当前 Token */
	token = $derived(
		this.currentAccount?.access_token ?? null
	);

	/** 是否已认证（Web 端 token 在 Cookie 中，通过 expires_at 判断） */
	isAuthenticated = $derived(
		!!this.currentAccount?.access_token || !!this.currentAccount?.access_token_expires_at
	);

	/** 可用账户列表（未锁定） */
	availableAccounts = $derived(
		this.#accounts.filter((a) => !a.is_locked)
	);

	/** 锁定账户列表 */
	lockedAccounts = $derived(
		this.#accounts.filter((a) => a.is_locked)
	);

	// ==================== Actions ====================

	/** 保存状态到 localStorage */
	#persist() {
		saveToStorage({
			accounts: this.#accounts,
			currentUserId: this.#currentUserId
		});
	}

	/** 登录/添加账户 */
	login(authData: AuthData) {
		const existingIndex = this.#accounts.findIndex((a) => a.user_id === authData.user_id);

		const accountInfo: AccountInfo = {
			user_id: authData.user_id,
			nickname: '', // Web 端 AuthData 不含 nickname，后续通过 setProfile 补充
			access_token: authData.access_token ?? '', // Web 端 token 在 Cookie 中
			access_token_expires_at: authData.access_token_expires_at,
			refresh_token: authData.refresh_token,
			refresh_token_expires_at: authData.refresh_token_expires_at,
			is_locked: false
		};

		if (existingIndex >= 0) {
			// 更新现有账户（不可变更新）
			this.#accounts = [
				...this.#accounts.slice(0, existingIndex),
				accountInfo,
				...this.#accounts.slice(existingIndex + 1)
			];
		} else {
			// 添加新账户
			this.#accounts = [...this.#accounts, accountInfo];
		}

		this.#currentUserId = authData.user_id;
		this.#persist();
	}

	/** 登出当前账户 */
	logout() {
		if (this.#currentUserId === null) return;

		this.#accounts = this.#accounts.filter((a) => a.user_id !== this.#currentUserId);
		this.#currentUserId = this.#accounts[0]?.user_id ?? null;
		this.#profile = null;
		this.#persist();
	}

	/** 登出所有账户 */
	logoutAll() {
		this.#accounts = [];
		this.#currentUserId = null;
		this.#profile = null;
		this.#persist();
	}

	/** 切换账户 */
	switchAccount(userId: string) {
		const account = this.#accounts.find((a) => a.user_id === userId && !a.is_locked);
		if (account) {
			this.#currentUserId = userId;
			this.#profile = null;
			this.#persist();
		}
	}

	/** 锁定账户（保留但不激活） */
	lockAccount(userId: string) {
		const index = this.#accounts.findIndex((a) => a.user_id === userId);
		if (index >= 0) {
			const account = this.#accounts[index]!;
			// 不可变更新
			this.#accounts = [
				...this.#accounts.slice(0, index),
				{ ...account, is_locked: true } as AccountInfo,
				...this.#accounts.slice(index + 1)
			];

			// 如果锁定的是当前账户，切换到其他账户
			if (this.#currentUserId === userId) {
				const nextAccount = this.#accounts.find((a) => !a.is_locked);
				this.#currentUserId = nextAccount?.user_id ?? null;
				this.#profile = null;
			}

			this.#persist();
		}
	}

	/** 解锁账户 */
	unlockAccount(userId: string) {
		const index = this.#accounts.findIndex((a) => a.user_id === userId);
		if (index >= 0) {
			const account = this.#accounts[index]!;
			// 不可变更新
			this.#accounts = [
				...this.#accounts.slice(0, index),
				{ ...account, is_locked: false } as AccountInfo,
				...this.#accounts.slice(index + 1)
			];
			this.#persist();
		}
	}

	/** 移除账户 */
	removeAccount(userId: string) {
		this.#accounts = this.#accounts.filter((a) => a.user_id !== userId);

		if (this.#currentUserId === userId) {
			this.#currentUserId = this.#accounts[0]?.user_id ?? null;
			this.#profile = null;
		}

		this.#persist();
	}

	/** 更新 Token */
	updateToken(accessToken: string, expiresAt: string) {
		if (this.#currentUserId === null) return;

		const index = this.#accounts.findIndex((a) => a.user_id === this.#currentUserId);
		if (index >= 0) {
			const account = this.#accounts[index]!;
			// 不可变更新
			this.#accounts = [
				...this.#accounts.slice(0, index),
				{
					...account,
					access_token: accessToken,
					access_token_expires_at: expiresAt
				} as AccountInfo,
				...this.#accounts.slice(index + 1)
			];
			this.#persist();
		}
	}

	/** 清除 Token（刷新失败时调用） */
	clearToken() {
		if (this.#currentUserId === null) return;

		const index = this.#accounts.findIndex((a) => a.user_id === this.#currentUserId);
		if (index >= 0) {
			const account = this.#accounts[index]!;
			// 不可变更新
			this.#accounts = [
				...this.#accounts.slice(0, index),
				{
					...account,
					access_token: '',
					access_token_expires_at: ''
				} as AccountInfo,
				...this.#accounts.slice(index + 1)
			];
			this.#persist();
		}
	}

	/** 设置用户资料 */
	setProfile(profile: MyCard | null) {
		this.#profile = profile;

		// 同步头像到账户信息
		if (profile && this.#currentUserId !== null) {
			const index = this.#accounts.findIndex((a) => a.user_id === this.#currentUserId);
			if (index >= 0) {
				const account = this.#accounts[index]!;
				// 不可变更新
				this.#accounts = [
					...this.#accounts.slice(0, index),
					{
						...account,
						nickname: profile.nickname,
						avatar_url: profile.avatar_url
					} as AccountInfo,
					...this.#accounts.slice(index + 1)
				];
				this.#persist();
			}
		}
	}

	/** 检查 Token 是否即将过期 */
	isTokenExpiringSoon(thresholdMinutes = 5): boolean {
		const account = this.currentAccount;
		if (!account?.access_token_expires_at) return true;

		const expiresAt = new Date(account.access_token_expires_at).getTime();
		const now = Date.now();
		const threshold = thresholdMinutes * 60 * 1000;

		return expiresAt - now < threshold;
	}
}

// 导出单例
export const authStore = new AuthStore();
