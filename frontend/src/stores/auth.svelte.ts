// Auth store — the single source of truth for "who is the current user".
// AuthContext（runes 模块单例；纯客户端 SPA，无服务端实例化问题）。
//
// Web 端通过 HttpOnly Cookie 认证（access_token / refresh_token），
// 本 store 不直接接触 token，只跟踪会话有效性（auth/check_status）与当前账号
// （users/get_account）。匿名浏览公开端点允许；写操作调用 requireAuth() 打开
// 登录弹层并在登录成功后重放该操作。

import {
  checkAuthStatus, getAccount, guest as apiGuest, login as apiLogin,
  logout as apiLogout, register as apiRegister, type ApiResponse,
} from '@/api';

export type AuthStatus = 'loading' | 'anonymous' | 'authenticated';

export interface AuthUser {
  user_id: string;
  nickname: string;
  avatar_url: string;
  user_type?: string;
}

export interface AuthEvent {
  type: 'auth-failed' | 'auth-banned';
  message?: string;
}

/** A deferred write-action (like/follow/post) gated behind login. */
export interface PendingAction {
  action: string;          // human label, e.g. "点赞" / "发帖"
  targetId?: string;       // optional subject (moment/group/user id)
  run: () => void | Promise<void>;
}

const DEVICE_ID_KEY = 'hotspot:device-id';

/** spec: DeviceId is UUID v4, persisted, not cleared on logout (10y cookie equivalent). */
export function getDeviceId(): string {
  try {
    const existing = localStorage.getItem(DEVICE_ID_KEY);
    if (existing) return existing;
    const id = uuidv4();
    localStorage.setItem(DEVICE_ID_KEY, id);
    return id;
  } catch {
    return uuidv4();
  }
}

function uuidv4(): string {
  if (typeof crypto !== 'undefined' && typeof crypto.randomUUID === 'function') {
    return crypto.randomUUID();
  }
  // RFC4122 v4 fallback for older browsers.
  return 'xxxxxxxx-xxxx-4xxx-yxxx-xxxxxxxxxxxx'.replace(/[xy]/g, (c) => {
    const r = (Math.random() * 16) | 0;
    const v = c === 'x' ? r : (r & 0x3) | 0x8;
    return v.toString(16);
  });
}

let status = $state<AuthStatus>('loading');
let user = $state<AuthUser | null>(null);
/** Authentication failure reason surfaced to the UI (e.g. ban message). */
let bannedMessage = $state<string | null>(null);
/** True when requireAuth opened the login prompt. */
let loginPromptOpen = $state(false);
let pendingAction = $state<PendingAction | null>(null);

async function refreshUser(): Promise<void> {
  const res = await getAccount().catch(() => null);
  if (res && res.code === 0 && res.data) {
    const d = res.data;
    user = {
      user_id: d.user_id,
      nickname: d.nickname,
      avatar_url: d.avatar_url ?? '',
      user_type: d.user_type,
    };
    status = 'authenticated';
  } else {
    // Account fetch failed (e.g. session revoked mid-flight) — drop to anonymous.
    user = null;
    status = 'anonymous';
  }
}

/** Boot: check session validity, then load account if authenticated. 由根布局调用一次。 */
export function initAuth(): () => void {
  void (async () => {
    const res = await checkAuthStatus().catch(() => null);
    const data = res && res.code === 0 ? res.data : null;
    if (data?.is_valid) {
      await refreshUser();
    } else {
      status = 'anonymous';
      user = null;
    }
  })();

  // Listen for auth-failed / auth-banned events dispatched by the API client.
  const onFailed = () => {
    status = 'anonymous';
    user = null;
    bannedMessage = null;
  };
  const onBanned = (e: Event) => {
    const detail = (e as CustomEvent<AuthEvent>).detail;
    status = 'anonymous';
    user = null;
    bannedMessage = detail?.message ?? '账号已被封禁';
  };
  window.addEventListener('auth-failed', onFailed as EventListener);
  window.addEventListener('auth-banned', onBanned as EventListener);
  return () => {
    window.removeEventListener('auth-failed', onFailed as EventListener);
    window.removeEventListener('auth-banned', onBanned as EventListener);
  };
}

/** Require auth before a write action; if anonymous, opens the login prompt. Returns true if allowed. */
export function requireAuth(action: PendingAction): boolean {
  if (status === 'authenticated') {
    return true;
  }
  // Store the action so the login prompt can replay it on success.
  // Persist to sessionStorage so a page refresh during login still recovers it.
  try {
    sessionStorage.setItem(
      'hotspot:pending-action',
      JSON.stringify({ action, targetId: action.targetId }),
    );
  } catch {}
  pendingAction = action;
  loginPromptOpen = true;
  return false;
}

export function closeLoginPrompt(): void {
  loginPromptOpen = false;
  pendingAction = null;
  try { sessionStorage.removeItem('hotspot:pending-action'); } catch {}
}

/** Dismiss the ban screen; the session is already anonymous so browsing continues. */
export function dismissBanned(): void {
  bannedMessage = null;
}

function replayPending(): void {
  loginPromptOpen = false;
  const action = pendingAction;
  pendingAction = null;
  try { sessionStorage.removeItem('hotspot:pending-action'); } catch {}
  if (action) {
    // Defer so the prompt closes before any navigation/scroll it may trigger.
    setTimeout(() => { void action.run(); }, 0);
  }
}

async function onAuthSuccess(): Promise<void> {
  bannedMessage = null;
  await refreshUser();
  if (pendingAction) {
    replayPending();
  }
}

// Auth mutations — inject device_id (spec-required) and update state.
export async function login(params: { login_id: string; password: string }): Promise<ApiResponse<unknown>> {
  const res = await apiLogin({ ...params, client_type: 'web', device_id: getDeviceId() });
  if (res.code === 0) await onAuthSuccess();
  return res;
}

export async function register(params: { login_id: string; nickname: string; password: string }): Promise<ApiResponse<unknown>> {
  const res = await apiRegister({ ...params, client_type: 'web', device_id: getDeviceId() });
  if (res.code === 0) await onAuthSuccess();
  return res;
}

export async function guest(params: { nickname: string }): Promise<ApiResponse<unknown>> {
  const res = await apiGuest({ ...params, client_type: 'web', device_id: getDeviceId() });
  if (res.code === 0) await onAuthSuccess();
  return res;
}

export async function logout(): Promise<void> {
  await apiLogout().catch(() => {});
  // Web: cookies are cleared server-side; we only reset client state.
  // Per the anonymous-browsing strategy, stay on the current route.
  status = 'anonymous';
  user = null;
  bannedMessage = null;
}

// Expose reactive state via getters（组件内直接读取即响应式）
export const auth = {
  get status() { return status; },
  get user() { return user; },
  get bannedMessage() { return bannedMessage; },
  get loginPromptOpen() { return loginPromptOpen; },
  get pendingAction() { return pendingAction; },
  get isAuthenticated() { return status === 'authenticated'; },
};
