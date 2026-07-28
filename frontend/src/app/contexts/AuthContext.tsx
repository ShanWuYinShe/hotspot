import React, { createContext, useCallback, useContext, useEffect, useMemo, useState } from "react";
import {
  checkAuthStatus, getAccount, guest as apiGuest, login as apiLogin,
  logout as apiLogout, register as apiRegister, type ApiResponse,
} from "../../api";

// AuthContext — the single source of truth for "who is the current user".
//
// The web client authenticates via HttpOnly cookies (access_token / refresh_token),
// so this context never touches tokens directly. It only tracks whether a valid
// session exists (auth/check_status) and, when authenticated, the current account
// (users/get_account). Anonymous browsing of public endpoints is allowed; write
// actions call requireAuth() which opens a login prompt and replays the action.

export type AuthStatus = "loading" | "anonymous" | "authenticated";

export interface AuthUser {
  user_id: string;
  nickname: string;
  avatar_url: string;
  user_type?: string;
}

export interface AuthEvent {
  type: "auth-failed" | "auth-banned";
  message?: string;
}

/** A deferred write-action (like/follow/post) gated behind login. */
export interface PendingAction {
  action: string;          // human label, e.g. "点赞" / "发帖"
  targetId?: string;       // optional subject (moment/group/user id)
  run: () => void | Promise<void>;
}

const DEVICE_ID_KEY = "hotspot:device-id";

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
  if (typeof crypto !== "undefined" && typeof crypto.randomUUID === "function") {
    return crypto.randomUUID();
  }
  // RFC4122 v4 fallback for older browsers.
  return "xxxxxxxx-xxxx-4xxx-yxxx-xxxxxxxxxxxx".replace(/[xy]/g, (c) => {
    const r = (Math.random() * 16) | 0;
    const v = c === "x" ? r : (r & 0x3) | 0x8;
    return v.toString(16);
  });
}

interface AuthContextValue {
  status: AuthStatus;
  user: AuthUser | null;
  /** Authentication failure reason surfaced to the UI (e.g. ban message). */
  bannedMessage: string | null;
  /** True when requireAuth opened the login prompt. */
  loginPromptOpen: boolean;
  pendingAction: PendingAction | null;
  /** Require auth before a write action; if anonymous, opens the login prompt. Returns true if allowed. */
  requireAuth: (action: PendingAction) => boolean;
  closeLoginPrompt: () => void;
  /** Dismiss the ban screen; the session is already anonymous so browsing continues. */
  dismissBanned: () => void;
  // Auth mutations — inject device_id (spec-required) and update state.
  login: (params: { login_id: string; password: string }) => Promise<ApiResponse<unknown>>;
  register: (params: { login_id: string; nickname: string; password: string }) => Promise<ApiResponse<unknown>>;
  guest: (params: { nickname: string }) => Promise<ApiResponse<unknown>>;
  logout: () => Promise<void>;
  refreshUser: () => Promise<void>;
}

const AuthContext = createContext<AuthContextValue | null>(null);

export function AuthProvider({ children }: { children: React.ReactNode }) {
  const [status, setStatus] = useState<AuthStatus>("loading");
  const [user, setUser] = useState<AuthUser | null>(null);
  const [bannedMessage, setBannedMessage] = useState<string | null>(null);
  const [loginPromptOpen, setLoginPromptOpen] = useState(false);
  const [pendingAction, setPendingAction] = useState<PendingAction | null>(null);

  const refreshUser = useCallback(async () => {
    const res = await getAccount().catch(() => null);
    if (res && res.code === 0 && res.data) {
      const d = res.data as any;
      setUser({
        user_id: d.user_id ?? d.id ?? "",
        nickname: d.nickname ?? d.name ?? "",
        avatar_url: d.avatar_url ?? d.avatar ?? "",
        user_type: d.user_type,
      });
      setStatus("authenticated");
    } else {
      // Account fetch failed (e.g. session revoked mid-flight) — drop to anonymous.
      setUser(null);
      setStatus("anonymous");
    }
  }, []);

  // Boot: check session validity, then load account if authenticated.
  useEffect(() => {
    let cancelled = false;
    (async () => {
      const res = await checkAuthStatus().catch(() => null);
      if (cancelled) return;
      const data = res && res.code === 0 ? (res.data as any) : null;
      if (data?.is_valid) {
        await refreshUser();
      } else {
        setStatus("anonymous");
        setUser(null);
      }
    })();
    return () => { cancelled = true; };
  }, [refreshUser]);

  // Listen for auth-failed / auth-banned events dispatched by the API client.
  useEffect(() => {
    const onFailed = () => {
      setStatus("anonymous");
      setUser(null);
      setBannedMessage(null);
    };
    const onBanned = (e: Event) => {
      const detail = (e as CustomEvent<AuthEvent>).detail;
      setStatus("anonymous");
      setUser(null);
      setBannedMessage(detail?.message ?? "账号已被封禁");
    };
    window.addEventListener("auth-failed", onFailed as EventListener);
    window.addEventListener("auth-banned", onBanned as EventListener);
    return () => {
      window.removeEventListener("auth-failed", onFailed as EventListener);
      window.removeEventListener("auth-banned", onBanned as EventListener);
    };
  }, []);

  const requireAuth = useCallback((action: PendingAction): boolean => {
    if (status === "authenticated") {
      return true;
    }
    // Store the action so the login prompt can replay it on success.
    // Persist to sessionStorage so a page refresh during login still recovers it.
    try {
      sessionStorage.setItem(
        "hotspot:pending-action",
        JSON.stringify({ action, targetId: action.targetId }),
      );
    } catch {}
    setPendingAction(action);
    setLoginPromptOpen(true);
    return false;
  }, [status]);

  const closeLoginPrompt = useCallback(() => {
    setLoginPromptOpen(false);
    setPendingAction(null);
    try { sessionStorage.removeItem("hotspot:pending-action"); } catch {}
  }, []);

  const dismissBanned = useCallback(() => {
    setBannedMessage(null);
  }, []);

  const replayPending = useCallback(() => {
    setLoginPromptOpen(false);
    const action = pendingAction;
    setPendingAction(null);
    try { sessionStorage.removeItem("hotspot:pending-action"); } catch {}
    if (action) {
      // Defer so the prompt closes before any navigation/scroll it may trigger.
      setTimeout(() => { void action.run(); }, 0);
    }
  }, [pendingAction]);

  const onAuthSuccess = useCallback(async () => {
    setBannedMessage(null);
    await refreshUser();
    if (pendingAction) {
      replayPending();
    }
  }, [pendingAction, refreshUser, replayPending]);

  const login = useCallback(async (params: { login_id: string; password: string }) => {
    const res = await apiLogin({ ...params, client_type: "web", device_id: getDeviceId() });
    if (res.code === 0) await onAuthSuccess();
    return res;
  }, [onAuthSuccess]);

  const register = useCallback(async (params: { login_id: string; nickname: string; password: string }) => {
    const res = await apiRegister({ ...params, client_type: "web", device_id: getDeviceId() });
    if (res.code === 0) await onAuthSuccess();
    return res;
  }, [onAuthSuccess]);

  const guest = useCallback(async (params: { nickname: string }) => {
    const res = await apiGuest({ ...params, client_type: "web", device_id: getDeviceId() });
    if (res.code === 0) await onAuthSuccess();
    return res;
  }, [onAuthSuccess]);

  const logout = useCallback(async () => {
    await apiLogout().catch(() => {});
    // Web: cookies are cleared server-side; we only reset client state.
    // Per the anonymous-browsing strategy, stay on the current route.
    setStatus("anonymous");
    setUser(null);
    setBannedMessage(null);
  }, []);

  const value = useMemo<AuthContextValue>(() => ({
    status, user, bannedMessage, loginPromptOpen, pendingAction,
    requireAuth, closeLoginPrompt, dismissBanned,
    login, register, guest, logout, refreshUser,
  }), [
    status, user, bannedMessage, loginPromptOpen, pendingAction,
    requireAuth, closeLoginPrompt, dismissBanned,
    login, register, guest, logout, refreshUser,
  ]);

  return <AuthContext.Provider value={value}>{children}</AuthContext.Provider>;
}

export function useAuth() {
  const ctx = useContext(AuthContext);
  if (!ctx) throw new Error("useAuth must be used within AuthProvider");
  return ctx;
}
