/**
 * WebSocket Store - 连接管理、鉴权、心跳
 * 严格遵循 @spec/api_spec_websocket.md
 *
 * 从 Svelte 5 迁移至 Zustand：
 *   $state       → Zustand state 字段
 *   $derived     → get() 计算
 *   $effect.root → initialize() 方法（从 useAppInit 调用）
 *   untrack()    → getState()（不带订阅的读取）
 */

import { create } from "zustand";
import { toast } from "sonner";
import { useAuthStore } from "@/lib/stores/auth";
import { ensureRefreshToken, TOKEN_INVALID_CODE } from "@/lib/api/client";
import type {
  ServerWebSocketEvent,
  ClientEvent,
  WebSocketEventMap,
  ConnectionEstablishedPayload,
  ErrorPayload,
} from "./types";

const isDev = import.meta.env.DEV;
const log = isDev ? console.log.bind(console) : () => {};
const logError = isDev ? console.error.bind(console) : () => {};

// ============ Spec 参考值（仅作 fallback）============

const SPEC_REFERENCE_PING_INTERVAL = 30000;
const SPEC_REFERENCE_PONG_TIMEOUT = 10000;

// 指数退避重连延迟（秒）
const RECONNECT_DELAYS = [1, 2, 4, 8, 16, 32, 60];
const MAX_DELAY = 60;
const JITTER_PERCENT = 0.2;

const MAX_RECONNECT_ATTEMPTS = 30;
const CONSECUTIVE_FAILURE_THRESHOLD = 10;

// ============ 连接状态 ============

export type ConnectionState =
  | "disconnected"
  | "connecting"
  | "authenticating"
  | "connected"
  | "reconnecting"
  | "failed";

// ============ Zustand State 接口 ============

interface WebSocketState {
  connectionState: ConnectionState;
  connectionId: string | null;
  lastError: string | null;
  lastDisconnectDurationMs: number | null;

  connect: (url?: string) => void;
  disconnect: () => void;
  send: (event: ClientEvent) => void;
  on: <K extends keyof WebSocketEventMap>(
    eventType: K,
    handler: (payload: WebSocketEventMap[K]) => void,
  ) => () => void;
  off: (eventType: string, handler?: (payload: unknown) => void) => void;
  destroy: () => void;
  initialize: () => void;
}

// ============ 模块级内部状态（不触发重渲染）============

let ws: WebSocket | null = null;
let wsUrl = "";
let serverConfig: {
  ping_interval: number;
  ping_timeout: number;
  max_subscriptions: { user: number; group: number };
} | null = null;

let reconnectAttempts = 0;
let reconnectTimer: ReturnType<typeof setTimeout> | null = null;
let shouldReconnect = true;
let userNotifiedUnstable = false;

let pingTimer: ReturnType<typeof setInterval> | null = null;
let pongTimer: ReturnType<typeof setTimeout> | null = null;
let lastPingTime = 0;

const eventListeners = new Map<string, Set<(payload: unknown) => void>>();
let pendingMessages: ClientEvent[] = [];

let connectingLock = false;
let lastConnectAttempt = 0;
const MIN_CONNECT_INTERVAL = 2000;

let lastServerTime: string | null = null;
let disconnectTime = 0;

let visibilityChangeHandler: (() => void) | null = null;
let onlineHandler: (() => void) | null = null;
let unsubscribeAuth: (() => void) | null = null;

// ============ Store 创建 ============

export const useWebSocketStore = create<WebSocketState>((set, get) => ({
  connectionState: "disconnected",
  connectionId: null,
  lastError: null,
  lastDisconnectDurationMs: null,

  initialize() {
    if (typeof window === "undefined") return;
    log("[WebSocket] 初始化自动生命周期管理");

    // 监听认证状态变化
    unsubscribeAuth = useAuthStore.subscribe((state, prevState) => {
      if (state.isAuthenticated === prevState.isAuthenticated) return;
      const { isAuthenticated } = state;
      log("[WebSocket] 认证状态变化:", isAuthenticated);

      const current = get().connectionState;
      if (isAuthenticated) {
        if (current === "disconnected" || current === "failed") {
          log("[WebSocket] 用户已登录，自动连接");
          get().connect();
        }
      } else {
        if (current !== "disconnected" && current !== "failed") {
          log("[WebSocket] 用户已登出，自动断开");
          get().disconnect();
        }
      }
    });

    // 页面可见性变化
    visibilityChangeHandler = () => {
      if (!document.hidden) {
        const { isAuthenticated } = useAuthStore.getState();
        const { connectionState } = get();
        if (
          isAuthenticated &&
          (connectionState === "failed" || connectionState === "disconnected")
        ) {
          log("[WebSocket] 页面显示，尝试恢复连接");
          reconnectAttempts = 0;
          userNotifiedUnstable = false;
          get().connect();
        }
      }
    };
    document.addEventListener("visibilitychange", visibilityChangeHandler);

    // 网络恢复
    onlineHandler = () => {
      log("[WebSocket] 网络恢复");
      const { isAuthenticated } = useAuthStore.getState();
      const { connectionState } = get();
      if (
        isAuthenticated &&
        (connectionState === "failed" || connectionState === "disconnected")
      ) {
        reconnectAttempts = 0;
        userNotifiedUnstable = false;
        get().connect();
      }
    };
    window.addEventListener("online", onlineHandler);

    // 开发环境调试工具
    if (isDev) {
      const dw = window as Window & {
        __debugWebSocket?: () => void;
        __wsStore?: typeof useWebSocketStore;
      };
      dw.__debugWebSocket = () => debug(get);
      dw.__wsStore = useWebSocketStore;
    }

    // 如果已经认证，立即连接
    if (useAuthStore.getState().isAuthenticated) {
      get().connect();
    }
  },

  connect(url?: string) {
    const { connectionState } = get();

    // 防护 1: 频率限制
    const now = Date.now();
    const timeSince = now - lastConnectAttempt;
    if (timeSince < MIN_CONNECT_INTERVAL) return;

    // 防护 2: 并发锁
    if (connectingLock) return;

    // 防护 3: 状态检查
    const isConnecting =
      connectionState === "connecting" ||
      connectionState === "authenticating" ||
      connectionState === "reconnecting";
    const isConnected = connectionState === "connected";
    if (isConnecting || isConnected) return;

    // 防护 4: 认证检查
    if (!useAuthStore.getState().isAuthenticated) return;

    // 防护 5: 清理残留连接
    if (ws?.readyState !== WebSocket.CLOSED && ws?.readyState !== undefined) {
      cleanup(true);
    }

    lastConnectAttempt = now;
    connectingLock = true;
    shouldReconnect = true;

    set({ lastError: null, lastDisconnectDurationMs: null });

    if (connectionState === "failed" || connectionState === "disconnected") {
      reconnectAttempts = 0;
    }

    if (url) {
      wsUrl = url;
    } else if (!wsUrl) {
      const wsOverride = (import.meta.env.VITE_WS_URL as string | undefined) ?? "";
      if (wsOverride) {
        wsUrl = wsOverride;
      } else {
        const protocol = window.location.protocol === "https:" ? "wss:" : "ws:";
        wsUrl = `${protocol}//${window.location.host}/ws/v1`;
      }
    }

    set({ connectionState: "connecting" });
    log(`[WebSocket] 连接到 ${wsUrl}`);

    try {
      ws = new WebSocket(wsUrl);
      setupWebSocketHandlers(set, get);
    } catch (error) {
      logError("[WebSocket] 连接失败:", error);
      connectingLock = false;
      handleConnectionError(set, get, "连接初始化失败");
    }
  },

  disconnect() {
    log("[WebSocket] 主动断开连接");
    shouldReconnect = false;
    reconnectAttempts = 0;
    connectingLock = false;
    pendingMessages = [];
    cleanup();
    set({ connectionState: "disconnected" });
  },

  send(event: ClientEvent) {
    const canSend = get().connectionState === "connected";
    if (!canSend) {
      pendingMessages.push(event);
      return;
    }
    try {
      ws?.send(JSON.stringify(event));
      log(`[WebSocket] ✓ 发送: ${event.type}`);
    } catch (error) {
      logError(`[WebSocket] ✗ 发送失败: ${event.type}`, error);
    }
  },

  on(eventType, handler) {
    if (!eventListeners.has(eventType)) {
      eventListeners.set(eventType, new Set());
    }
    const typedHandler = handler as (payload: unknown) => void;
    eventListeners.get(eventType)!.add(typedHandler);
    return () => {
      eventListeners.get(eventType)?.delete(typedHandler);
    };
  },

  off(eventType, handler) {
    if (!handler) {
      eventListeners.delete(eventType);
    } else {
      eventListeners.get(eventType)?.delete(handler);
    }
  },

  destroy() {
    log("[WebSocket] 彻底清理所有资源");
    shouldReconnect = false;
    pendingMessages = [];
    cleanup();

    if (visibilityChangeHandler) {
      document.removeEventListener("visibilitychange", visibilityChangeHandler);
      visibilityChangeHandler = null;
    }
    if (onlineHandler) {
      window.removeEventListener("online", onlineHandler);
      onlineHandler = null;
    }
    if (unsubscribeAuth) {
      unsubscribeAuth();
      unsubscribeAuth = null;
    }
    eventListeners.clear();
    log("[WebSocket] 资源清理完成");
  },
}));

// ============ 内部函数 ============

function setupWebSocketHandlers(
  set: (partial: Partial<WebSocketState>) => void,
  get: () => WebSocketState,
) {
  if (!ws) return;

  ws.onopen = () => {
    log("[WebSocket] 连接已建立，开始鉴权...");
    set({ connectionState: "authenticating" });
    reconnectAttempts = 0;
    userNotifiedUnstable = false;
    authenticate(set, get);
  };

  ws.onmessage = (event) => {
    try {
      const message: ServerWebSocketEvent = JSON.parse(event.data);
      handleServerEvent(set, get, message);
    } catch (error) {
      logError("[WebSocket] 解析消息失败:", error);
    }
  };

  ws.onerror = (error) => {
    logError("[WebSocket] 连接错误:", error);
  };

  ws.onclose = (event) => {
    log("[WebSocket] 连接已关闭:", event.code, event.reason);
    if (event.code === 1000 || event.code === 1001) {
      shouldReconnect = false;
    }
    handleConnectionClose(set, get, event.code, event.reason);
  };
}

function authenticate(
  _set: (partial: Partial<WebSocketState>) => void,
  get: () => WebSocketState,
) {
  const authEvent: ClientEvent = {
    type: "Auth",
    payload: { client_type: "web" },
  };
  try {
    ws?.send(JSON.stringify(authEvent));
    log("[WebSocket] 已发送鉴权请求");
  } catch (error) {
    logError("[WebSocket] 发送鉴权消息失败:", error);
    handleConnectionError(_set, get, "鉴权失败");
  }
}

function handleServerEvent(
  set: (partial: Partial<WebSocketState>) => void,
  get: () => WebSocketState,
  event: ServerWebSocketEvent,
) {
  log("[WebSocket] 收到事件:", event.type);

  switch (event.type) {
    case "ConnectionEstablished":
      handleConnectionEstablished(
        set,
        get,
        event.payload as ConnectionEstablishedPayload,
      );
      break;
    case "Pong":
      handlePong();
      break;
    case "Error":
      handleServerError(set, get, event.payload as ErrorPayload);
      break;
    default:
      emitEvent(event.type, event.payload);
  }
}

function handleConnectionEstablished(
  set: (partial: Partial<WebSocketState>) => void,
  get: () => WebSocketState,
  payload: ConnectionEstablishedPayload,
) {
  connectingLock = false;
  serverConfig = payload.config;

  // 计算断开时长
  let disconnectDurationMs: number | null = null;
  if (lastServerTime) {
    const lastServerMs = new Date(lastServerTime).getTime();
    const currentServerMs = new Date(payload.server_time).getTime();
    const diff = currentServerMs - lastServerMs;
    disconnectDurationMs = diff > 0 ? diff : null;
    if (diff > 10000) {
      log("[WebSocket] 断开超过 10 秒，建议补拉离线数据");
    }
  } else if (disconnectTime > 0) {
    const diff = Date.now() - disconnectTime;
    disconnectDurationMs = diff > 0 ? diff : null;
  }
  lastServerTime = payload.server_time;

  set({
    connectionState: "connected",
    connectionId: payload.connection_id ?? null,
    lastError: null,
    lastDisconnectDurationMs: disconnectDurationMs,
  });

  startHeartbeat();
  flushPendingMessages(get);
  emitEvent("ConnectionEstablished", payload);
}

function handlePong() {
  if (pongTimer) {
    clearTimeout(pongTimer);
    pongTimer = null;
  }
  const latency = Date.now() - lastPingTime;
  log("[WebSocket] 收到 Pong，延迟:", latency, "ms");
}

function handleServerError(
  set: (partial: Partial<WebSocketState>) => void,
  get: () => WebSocketState,
  payload: ErrorPayload,
) {
  logError("[WebSocket] 服务端错误:", payload);
  set({ lastError: payload.message ?? "未知错误" });
  emitEvent("Error", payload);

  switch (payload.severity) {
    case "fatal":
      shouldReconnect = false;
      cleanup();
      set({ connectionState: "failed" });
      toast.error(payload.message || "会话已失效，请重新登录");
      useAuthStore.getState().logout();
      break;

    case "recoverable":
      if (payload.code === 20000 || payload.code === TOKEN_INVALID_CODE) {
        shouldReconnect = true;
        cleanup();
        set({ connectionState: "reconnecting" });

        ensureRefreshToken()
          .then((success) => {
            if (success) {
              reconnectAttempts = 0;
              get().connect();
            } else {
              shouldReconnect = false;
              set({ connectionState: "failed" });
              useAuthStore.getState().logout();
            }
          })
          .catch((err) => {
            logError("[WebSocket] Token刷新异常", err);
          });
      }
      break;

    case "warning":
      break;
  }
}

function handleConnectionError(
  set: (partial: Partial<WebSocketState>) => void,
  get: () => WebSocketState,
  error: string,
) {
  set({ lastError: error, connectionState: "failed" });
  connectingLock = false;
  cleanup(false);
  scheduleReconnect(set, get);
}

function handleConnectionClose(
  set: (partial: Partial<WebSocketState>) => void,
  get: () => WebSocketState,
  _code: number,
  _reason: string,
) {
  disconnectTime = Date.now();
  connectingLock = false;
  cleanup(false);

  if (shouldReconnect) {
    scheduleReconnect(set, get);
  } else {
    set({ connectionState: "disconnected" });
  }
}

function startHeartbeat() {
  stopHeartbeat();

  const pingInterval = serverConfig?.ping_interval
    ? serverConfig.ping_interval * 1000
    : SPEC_REFERENCE_PING_INTERVAL;

  pingTimer = setInterval(() => {
    sendPing();
  }, pingInterval);

  log(`[WebSocket] 心跳已启动 (间隔: ${pingInterval}ms)`);
}

function stopHeartbeat() {
  if (pingTimer) {
    clearInterval(pingTimer);
    pingTimer = null;
  }
  if (pongTimer) {
    clearTimeout(pongTimer);
    pongTimer = null;
  }
}

function sendPing() {
  const state = useWebSocketStore.getState().connectionState;
  if (state !== "connected") return;

  lastPingTime = Date.now();
  useWebSocketStore.getState().send({ type: "Ping", payload: {} });

  const pongTimeout = serverConfig?.ping_timeout
    ? serverConfig.ping_timeout * 1000
    : SPEC_REFERENCE_PONG_TIMEOUT;

  pongTimer = setTimeout(() => {
    logError("[WebSocket] Pong 超时，连接异常");
    handleConnectionError(
      useWebSocketStore.setState.bind(useWebSocketStore),
      useWebSocketStore.getState,
      "Pong 超时",
    );
  }, pongTimeout);
}

function scheduleReconnect(
  set: (partial: Partial<WebSocketState>) => void,
  get: () => WebSocketState,
) {
  if (reconnectTimer) return;
  if (!shouldReconnect) {
    set({ connectionState: "failed" });
    connectingLock = false;
    return;
  }

  if (reconnectAttempts >= MAX_RECONNECT_ATTEMPTS) {
    shouldReconnect = false;
    set({ connectionState: "failed" });
    connectingLock = false;
    toast.error("连接失败，请检查网络后重试");
    return;
  }

  set({ connectionState: "reconnecting" });

  const delay = getReconnectDelay(reconnectAttempts);
  reconnectAttempts++;

  if (
    reconnectAttempts >= CONSECUTIVE_FAILURE_THRESHOLD &&
    !userNotifiedUnstable
  ) {
    userNotifiedUnstable = true;
    toast.warning("连接不稳定，正在重试…");
  }

  log(
    `[WebSocket] ${delay.toFixed(1)}秒后重连（第 ${reconnectAttempts}/${MAX_RECONNECT_ATTEMPTS} 次）`,
  );

  reconnectTimer = setTimeout(() => {
    reconnectTimer = null;
    connectingLock = false;
    get().connect();
  }, delay * 1000);
}

function getReconnectDelay(attempt: number): number {
  const baseDelay =
    RECONNECT_DELAYS[Math.min(attempt, RECONNECT_DELAYS.length - 1)] ??
    MAX_DELAY;
  const jitter = baseDelay * JITTER_PERCENT * (Math.random() * 2 - 1);
  return Math.max(Math.min(baseDelay + jitter, MAX_DELAY), 0.1);
}

function cleanup(closeWebSocket = true) {
  stopHeartbeat();
  if (reconnectTimer) {
    clearTimeout(reconnectTimer);
    reconnectTimer = null;
  }
  if (closeWebSocket && ws) {
    ws.close();
  }
  ws = null;
  connectingLock = false;
}

function flushPendingMessages(get: () => WebSocketState) {
  if (pendingMessages.length === 0) return;
  log("[WebSocket] 发送待发送队列:", pendingMessages.length);
  for (const msg of pendingMessages) {
    get().send(msg);
  }
  pendingMessages = [];
}

function emitEvent(eventType: string, payload: unknown) {
  const listeners = eventListeners.get(eventType);
  if (listeners) {
    for (const listener of listeners) {
      try {
        listener(payload);
      } catch (error) {
        logError(`[WebSocket] 事件监听器执行失败 (${eventType}):`, error);
      }
    }
  }
}

function debug(get: () => WebSocketState) {
  const state = get();
  console.group("🔍 WebSocket Debug Info");
  console.group("📡 连接状态");
  log("状态:", state.connectionState);
  log("连接 ID:", state.connectionId ?? "无");
  console.groupEnd();
  console.group("🔄 重连控制");
  log("重连次数:", `${reconnectAttempts}/${MAX_RECONNECT_ATTEMPTS}`);
  console.groupEnd();
  console.group("📨 消息队列");
  log("待发送消息:", pendingMessages.length);
  console.groupEnd();
  console.groupEnd();
  return state;
}
