// Auth store — the single source of truth for "who is the current user".
// AuthContext（runes 模块单例；纯客户端 SPA，无服务端实例化问题）。
//
// Web 端通过 HttpOnly Cookie 认证（access_token / refresh_token），
// 本 store 不直接接触 token，只跟踪会话有效性（auth/session）与当前账号
// （users/account）。匿名浏览公开端点允许；写操作调用 requireAuth() 打开
// 登录弹层并在登录成功后重放该操作。

import {
  bootstrap as apiBootstrap,
  checkAuthStatus, getAccount, guest as apiGuest, login as apiLogin,
  logout as apiLogout, register as apiRegister, setAuthenticated,
  setKnownAccessExpiry,
  type ApiResponse,
} from '@/api';

import { resetAdmin } from "@/stores/admin.svelte";
import { resetPresence } from "@/stores/presence.svelte";
import { applyBootstrapUnread, resetUnread } from "@/stores/unread.svelte";

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

let status = $state<AuthStatus>('loading');
let user = $state<AuthUser | null>(null);
/** Authentication failure reason surfaced to the UI (e.g. ban message). */
let bannedMessage = $state<string | null>(null);
/** True when requireAuth opened the login prompt. */
let loginPromptOpen = $state(false);
/** 上次会话检查是否因传输失败（而非真正失效）落到匿名：此类匿名态可在
服务恢复后自愈（online/切回前台时静默重查），真正的失效不打扰用户。 */
let transportFailed = false;
let pendingAction = $state<PendingAction | null>(null);

/** 重新拉取当前账号（users/account）并同步登录态。供升级等改变 user_type 的操作后刷新。 */
export async function refreshUser(): Promise<void> {
  let res;
  try {
    res = await getAccount();
  } catch {
    // 传输失败（后端短暂不可达）：保持现有登录态，由调用方 toast/重试，
    // 不锁存匿名（此前 catch 即判匿名，部署滚动窗口内必掉线）。
    return;
  }
  if (res && res.code === 0 && res.data) {
    const d = res.data;
    user = {
      user_id: d.user_id,
      nickname: d.nickname,
      avatar_url: d.avatar_url ?? '',
      user_type: d.user_type,
    };
    status = 'authenticated';
    setAuthenticated(true);
  } else {
    // Account fetch failed (e.g. session revoked mid-flight) — drop to anonymous.
    user = null;
    status = 'anonymous';
  }
}

/** Boot: check session validity, then load account if authenticated. 由根布局调用一次。 */
export function initAuth(): () => void {
  void (async () => {
    // spec api_spec_base.md「Web 端 Cookie 契约」：bootstrap 是冷启动第一端点，
    // 服务端在此下发 device_id Cookie；auth/register、auth/guest、auth/signin 等
    // Web 端认证变更都从该 Cookie 读取设备标识，缺失即 10000 参数错误。
    // 失败不阻塞启动（匿名浏览公开端点不受影响）。
    const boot = await apiBootstrap().catch(() => null);
    // 传输失败退避重试（部署滚动窗口内冷启动）：只在真正的会话失效时落
    // 匿名，避免后端短暂不可用造成“发版即掉登录”且永不恢复。
    let res: Awaited<ReturnType<typeof checkAuthStatus>> | null = null;
    transportFailed = false;
    for (let attempt = 0; ; attempt++) {
      try {
        res = await checkAuthStatus();
        break;
      } catch {
        if (attempt >= 2) {
          transportFailed = true;
          break;
        }
        await new Promise<void>((r) => setTimeout(r, attempt === 0 ? 2000 : 5000));
      }
    }
    const data = res && res.code === 0 ? res.data : null;
    if (data) {
      // 冷启动未读校准：bootstrap 已携带服务端现算的 unread_summary，零额外请求
      applyBootstrapUnread(boot && boot.code === 0 ? boot.data?.unread_summary : null);
      setKnownAccessExpiry(data.expires_at);
      await refreshUser();
    } else {
      setKnownAccessExpiry(null);
      status = 'anonymous';
      user = null;
      setAuthenticated(false);
    }
  })();

  // Listen for auth-failed / auth-banned events dispatched by the API client.
  const onFailed = () => {
    transportFailed = false;
    setKnownAccessExpiry(null);
    status = 'anonymous';
    user = null;
    bannedMessage = null;
    resetPresence();
    resetUnread();
    // 会话失效必须连带清 admin 快照：isAdmin/permissions 与 loaded 残留会让
    // 同一标签页随后登录的其他账号沿用上一个账号的管理员入口与权限显示
    // （ensureAdminLoaded 因 loaded=true 永不重查）。
    resetAdmin();
    setAuthenticated(false);
  };
  const onBanned = (e: Event) => {
    transportFailed = false;
    setKnownAccessExpiry(null);
    const detail = (e as CustomEvent<AuthEvent>).detail;
    status = 'anonymous';
    user = null;
    bannedMessage = detail?.message ?? '账号已被封禁';
    resetPresence();
    resetUnread();
    resetAdmin(); // 同 onFailed：封禁后清 admin 快照，防跨账号权限残留
  };
  window.addEventListener('auth-failed', onFailed as EventListener);
  window.addEventListener('auth-banned', onBanned as EventListener);
  // 传输失败锁存的匿名态自愈：网络恢复/切回前台时静默重查一次，成功即
  // 恢复登录态（WS 生命周期跟随 status 自动重连）。仅 transportFailed 时
  // 触发，从未登录过的匿名用户与封禁用户不受打扰。
  const maybeRecover = () => {
    if (status === 'anonymous' && transportFailed && !bannedMessage) void recheckAuth();
  };
  window.addEventListener('online', maybeRecover);
  const onVisible = () => {
    if (document.visibilityState === 'visible') maybeRecover();
  };
  document.addEventListener('visibilitychange', onVisible);
  return () => {
    window.removeEventListener('auth-failed', onFailed as EventListener);
    window.removeEventListener('auth-banned', onBanned as EventListener);
    window.removeEventListener('online', maybeRecover);
    document.removeEventListener('visibilitychange', onVisible);
  };
}

/** 静默重查会话（自愈入口）：成功恢复登录态；仍失败则保持匿名。 */
export async function recheckAuth(): Promise<void> {
  let res;
  try {
    res = await checkAuthStatus();
  } catch {
    return;
  }
  if (res && res.code === 0 && res.data) {
    transportFailed = false;
    setKnownAccessExpiry(res.data.expires_at);
    await refreshUser();
  } else if (res) {
    // 服务端明确答复非成功：不是传输问题，清除自愈标记。
    transportFailed = false;
  }
}

/** Require auth before a write action; if anonymous, opens the login prompt. Returns true if allowed. */
export function requireAuth(action: PendingAction): boolean {
  if (status === 'authenticated') {
    return true;
  }
  // Store the action so the login prompt can replay it on success.
  // 注：不做 sessionStorage 持久化——读取/恢复逻辑从未实现，写入只会产生
  // 永不被消费的死数据；刷新页面后重走登录流程即可重新触发动作。
  pendingAction = action;
  loginPromptOpen = true;
  return false;
}

export function closeLoginPrompt(): void {
  loginPromptOpen = false;
  pendingAction = null;
}

/** Dismiss the ban screen; the session is already anonymous so browsing continues. */
export function dismissBanned(): void {
  bannedMessage = null;
}

function replayPending(): void {
  loginPromptOpen = false;
  const action = pendingAction;
  pendingAction = null;

  if (action) {
    // Defer so the prompt closes before any navigation/scroll it may trigger.
    setTimeout(() => { void action.run(); }, 0);
  }
}

async function onAuthSuccess(): Promise<void> {
  bannedMessage = null;
  await refreshUser();
  // 登录后习得 access 过期时间，驱动后续请求的主动续期（失败静默，被动链兜底）。
  const st = await checkAuthStatus().catch(() => null);
  if (st && st.code === 0 && st.data) setKnownAccessExpiry(st.data.expires_at);
  // 登录/注册/访客成功后重新 bootstrap：拉取该账号的 unread_summary 校准全局
  // 未读徽标（与冷启动同一来源；失败静默——后续由 WS 增量与页面回写收敛）。
  const boot = await apiBootstrap().catch(() => null);
  applyBootstrapUnread(boot && boot.code === 0 ? boot.data?.unread_summary : null);
  if (pendingAction) {
    replayPending();
  }
}

// Auth mutations — device_id 传 null（spec: Web 端由 bootstrap 经 Set-Cookie 下发
// device_id Cookie，服务端自行读取；App 端才需显式携带）。
export async function login(params: { login_id: string; password: string }): Promise<ApiResponse<unknown>> {
  const res = await apiLogin({ ...params, client_type: 'web', device_id: null });
  if (res.code === 0) await onAuthSuccess();
  return res;
}

export async function register(params: { login_id: string; nickname: string; password: string; code?: string | null }): Promise<ApiResponse<unknown>> {
  const { code, ...rest } = params;
  const res = await apiRegister({ ...rest, verification_code: code ?? null, client_type: 'web', device_id: null });
  if (res.code === 0) await onAuthSuccess();
  return res;
}

export async function guest(params: { nickname?: string | null } = {}): Promise<ApiResponse<unknown>> {
  const res = await apiGuest({ nickname: params.nickname ?? null, client_type: 'web', device_id: null });
  if (res.code === 0) await onAuthSuccess();
  return res;
}

export async function logout(): Promise<void> {
  await apiLogout().catch(() => {});
  // Web: cookies are cleared server-side; we only reset client state.
  // Per the anonymous-browsing strategy, stay on the current route.
  setKnownAccessExpiry(null);
  status = 'anonymous';
  user = null;
  bannedMessage = null;
  resetAdmin();
  resetPresence();
  resetUnread();
  setAuthenticated(false);
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
