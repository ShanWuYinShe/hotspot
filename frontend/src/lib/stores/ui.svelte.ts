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

  /**
   * 登录成功后需要补做的动作（如点赞/关注/评论）。
   * 用模块级单例内存保存——SvelteKit 客户端导航不会重载模块，回调可在跨页面后保留。
   * 用户点「稍后」放弃时清空，避免误触发。
   */
  #pendingAction: (() => void) | null = null;

  showCreateSheet = $derived(this.#showCreateSheet);
  showCreateMoment = $derived(this.#showCreateMoment);
  showCreateGroup = $derived(this.#showCreateGroup);
  showSearchModal = $derived(this.#showSearchModal);
  showAuthPrompt = $derived(this.#showAuthPrompt);

  /**
   * 显示内联登录提示，用户手动点击后跳转到登录页
   * @param callback 登录成功后需要补做的动作（可选）
   */
  openLoginModal(callback?: () => void) {
    this.#pendingAction = callback ?? null;
    // 统一保存当前 URL 作为登录后返回目的地，免去每个调用方手动 saveRedirectUrl
    this.saveRedirectUrl();
    this.#showAuthPrompt = true;
  }

  openLoginModalWithRedirect(options: { url?: string; onSuccess?: () => void } = {}) {
    this.#pendingAction = options.onSuccess ?? null;
    // 消费 url：若调用方指定了目标页（如底栏 tab），用它覆盖默认当前页
    if (options.url) {
      this.saveRedirectUrl(options.url);
    } else {
      this.saveRedirectUrl();
    }
    this.#showAuthPrompt = true;
  }

  /** 关闭内联登录提示。用户点「稍后」放弃动作时清空待补做回调。 */
  closeAuthPrompt() {
    this.#showAuthPrompt = false;
    this.#pendingAction = null;
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

  /** 登录成功后跳转到之前保存的页面，并补做被登录中断的动作（如有） */
  onLoginSuccess() {
    if (!browser) return;
    const pending = this.#pendingAction;
    this.#pendingAction = null;
    const redirectUrl = this.hasRedirectUrl() ? this.getAndClearRedirectUrl() : null;
    if (redirectUrl) {
      const currentUrl = `${window.location.pathname}${window.location.search}${window.location.hash}`;
      if (redirectUrl !== currentUrl) {
        goto(redirectUrl);
      }
    }
    // 登录后自动补做原动作（点赞/关注/评论等）。回调内部已绑定目标数据，
    // 即便 redirect 跳转了新页面，回调在目标页的数据上下文中执行。
    if (pending) {
      // 微任务延后执行，确保重定向后的页面已挂载、authStore 已刷新
      queueMicrotask(() => {
        try {
          pending();
        } catch {
          // 补做失败不阻断登录流程，用户可手动重试
        }
      });
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
