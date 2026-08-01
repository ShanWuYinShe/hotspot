// WebSocket 客户端 — 原生 WebSocket，零框架依赖，模块单例。
// 对齐 src/api/client.ts 模式：薄封装 + window CustomEvent 事件驱动。
//
// 契约（spec/api_spec_websocket.md）：
// - 连接成功后 5 秒内发送 Auth{client_type:'web'}（Cookie 自动鉴权，不传 token）
// - ConnectionEstablished 下发心跳配置（ping_interval / ping_timeout）与 resume_token
// - Ping 后 ping_timeout 内未收到 Pong 视为断线，触发重连
// - 重连：指数退避 1s/2s/4s…最大 30s + 随机 jitter；携带 resume_token
// - close code 1000/1008/4001 与手动 disconnect 不自动重连
// - 重连成功后用 set 操作全量重放订阅（本地订阅状态为权威）
// - SessionRevoked/AccountBanned → auth-failed/auth-banned CustomEvent（与 client.ts 一致）
// - 通知类/私聊类事件 → ws:* CustomEvent 供 UI 消费

import type {
  AuthPayload,
  ClientEventType,
  ClientWsEvent,
  ConnectionEstablishedPayload,
  ServerEventPayloadMap,
  ServerEventType,
  ServerWsEvent,
  SetSubscriptionsPayload,
  SubscribeAckPayload,
  SubscriptionItem,
  SubscriptionOperation,
  WsEvent,
} from './types';

export type WsStatus = 'idle' | 'connecting' | 'connected' | 'disconnected';

export interface WsClientOptions {
  /** 覆盖连接 URL（测试注入用）；默认基于 location.origin 推导 `${origin}/ws/v1` */
  url?: string;
  /** 重连退避基数（ms），默认 1000 */
  baseBackoffMs?: number;
  /** 重连退避上限（ms），默认 30000 */
  maxBackoffMs?: number;
}

// 全局 CustomEvent 名称（与 src/api/client.ts 的 auth-failed/auth-banned 对齐）
export const WS_AUTH_FAILED_EVENT = 'auth-failed';
export const WS_AUTH_BANNED_EVENT = 'auth-banned';
export const WS_NOTIFY_EVENT = 'ws:notify';
export const WS_PRIVATE_MESSAGE_EVENT = 'ws:private-message';
export const WS_PRIVATE_MESSAGE_UPDATE_EVENT = 'ws:private-message-update';

const WS_PATH = '/ws/v1';

/** 服务端主动断开且不应自动重连的 close code（契约级）。 */
const NO_RECONNECT_CODES = new Set([1000, 1008, 4001]);

/** 通知类事件 → ws:notify（payload 携带 notification_id，与 notify/list 同主键）。 */
const NOTIFY_EVENT_TYPES = new Set<ServerEventType>([
  'NewFollower',
  'MutualFollow',
  'MomentCommented',
  'CommentReply',
  'MessageMentioned',
  'ReactionAdded',
  'GroupInviteReceived',
  'GroupInviteRevoked',
  'GroupInviteExpired',
  'GroupInviteRejected',
  'GroupInviteResponse',
  'GroupApplicationReceived',
  'GroupApplicationCancelled',
  'GroupApplicationExpired',
  'GroupApplicationRejected',
  'GroupApplicationResponse',
  'GroupAnnouncementCreated',
  'ReportResolved',
  'ReportDismissed',
  'SystemAnnouncement',
  'NearbyNewMoment',
]);

const SESSION_REVOKED_MESSAGES: Record<string, string> = {
  device_evicted: '账号在其他设备登录',
  password_changed: '密码已修改，请重新登录',
  revoked_by_user: '会话已失效',
};

type Handler<P = unknown> = (payload: P) => void;

function uuid(): string {
  if (typeof crypto !== 'undefined' && typeof crypto.randomUUID === 'function') {
    return crypto.randomUUID();
  }
  return `req-${Date.now()}-${Math.random().toString(36).slice(2, 10)}`;
}

export class WsClient {
  private ws: WebSocket | null = null;
  private readonly options: Required<Pick<WsClientOptions, 'baseBackoffMs' | 'maxBackoffMs'>> &
    WsClientOptions;
  /** 手动断开（登出）标记：禁止自动重连。 */
  private manualClose = false;
  private reconnectAttempts = 0;
  private reconnectTimer: ReturnType<typeof setTimeout> | null = null;
  private heartbeatTimer: ReturnType<typeof setInterval> | null = null;
  private pongTimer: ReturnType<typeof setTimeout> | null = null;
  private resumeToken: string | null = null;
  private serverSeq = 0;
  private pingIntervalMs = 30_000;
  private pingTimeoutMs = 10_000;
  private status: WsStatus = 'idle';
  /** 本地订阅状态为权威（重连后全量重放）。 */
  private subscriptions: Record<'user' | 'group', Set<string>> = {
    user: new Set(),
    group: new Set(),
  };
  private handlers = new Map<ServerEventType, Set<Handler>>();
  private anyHandlers = new Set<Handler<ServerWsEvent>>();
  private statusHandlers = new Set<Handler<WsStatus>>();
  private pendingAcks = new Map<string, { operation: SubscriptionOperation; items: SubscriptionItem[] }>();

  constructor(options: WsClientOptions = {}) {
    this.options = { baseBackoffMs: 1000, maxBackoffMs: 30_000, ...options };
  }

  get isConnected(): boolean {
    return this.status === 'connected';
  }

  get currentStatus(): WsStatus {
    return this.status;
  }

  /** 建立连接（幂等：已连接/连接中则忽略）。URL 基于当前 origin 推导。 */
  connect(): void {
    if (typeof window === 'undefined') return;
    const ready = this.ws?.readyState;
    if (ready === WebSocket.OPEN || ready === WebSocket.CONNECTING) return;

    this.manualClose = false;
    this.setStatus('connecting');
    this.clearTimers();

    const url = this.options.url ?? `${window.location.origin.replace(/^http/, 'ws')}${WS_PATH}`;
    let socket: WebSocket;
    try {
      socket = new WebSocket(url);
    } catch {
      // 构造失败（如 URL 非法）也走退避重连
      this.scheduleReconnect();
      return;
    }
    this.ws = socket;
    socket.onopen = () => this.sendAuth();
    socket.onmessage = (ev) => this.handleMessage(ev.data);
    socket.onclose = (ev) => this.handleClose(ev.code);
    socket.onerror = () => {
      /* 错误统一由 onclose 处理（触发重连） */
    };
  }

  /** 手动断开（登出时调用）：标记 manualClose，禁止自动重连。 */
  disconnect(reason?: string): void {
    if (typeof window === 'undefined') return;
    this.manualClose = true;
    this.clearTimers();
    const socket = this.ws;
    this.ws = null;
    if (socket) {
      socket.onclose = null;
      try {
        socket.close(1000, reason);
      } catch {
        /* 忽略 */
      }
    }
    this.setStatus('disconnected');
  }

  /**
   * 设置订阅（本地状态同步更新 + 发送 SetSubscriptions）。
   * 未连接时仅更新本地状态，重连成功后自动全量重放。
   * @returns client_request_id（未连接时也返回，便于调用方对齐）
   */
  setSubscriptions(operation: SubscriptionOperation, items: SubscriptionItem[]): string | null {
    if (typeof window === 'undefined') return null;

    // 1. 更新本地订阅状态（权威）
    for (const item of items) {
      const set = this.subscriptions[item.type];
      if (operation === 'subscribe') {
        for (const id of item.ids) set.add(id);
      } else if (operation === 'unsubscribe') {
        for (const id of item.ids) set.delete(id);
      } else {
        // set：全量覆盖
        set.clear();
        for (const id of item.ids) set.add(id);
      }
    }

    // 2. 发送（仅当连接可用）
    const clientRequestId = uuid();
    const payload: SetSubscriptionsPayload = {
      client_request_id: clientRequestId,
      operation,
      items,
    };
    if (this.ws?.readyState === WebSocket.OPEN) {
      this.pendingAcks.set(clientRequestId, { operation, items });
      this.send({ type: 'SetSubscriptions', payload });
    }
    return clientRequestId;
  }

  /** 订阅 emitter：按 type 分发 payload（返回解绑函数）。 */
  on<K extends ServerEventType>(type: K, handler: (payload: ServerEventPayloadMap[K]) => void): () => void {
    let set = this.handlers.get(type);
    if (!set) {
      set = new Set();
      this.handlers.set(type, set);
    }
    set.add(handler as Handler);
    return () => this.off(type, handler);
  }

  off<K extends ServerEventType>(type: K, handler: (payload: ServerEventPayloadMap[K]) => void): void {
    this.handlers.get(type)?.delete(handler as Handler);
  }

  /** 监听全部服务端事件（返回解绑函数）。 */
  onAny(handler: (event: ServerWsEvent) => void): () => void {
    this.anyHandlers.add(handler);
    return () => {
      this.anyHandlers.delete(handler);
    };
  }

  /** 连接状态回调（返回解绑函数）。 */
  onStatusChange(cb: (status: WsStatus) => void): () => void {
    this.statusHandlers.add(cb);
    return () => {
      this.statusHandlers.delete(cb);
    };
  }

  // -------------------------------------------------------------------------
  // 内部实现
  // -------------------------------------------------------------------------

  /** 连接打开：契约要求连接后 5 秒内发送 Auth；重连时携带 resume_token。 */
  private sendAuth(): void {
    const payload: AuthPayload = { client_type: 'web' };
    if (this.resumeToken) payload.resume_token = this.resumeToken;
    this.send({ type: 'Auth', payload });
  }

  private handleMessage(data: unknown): void {
    let event: ServerWsEvent;
    try {
      event = JSON.parse(String(data)) as ServerWsEvent;
    } catch {
      console.warn('[ws] 无法解析的服务端消息：', data);
      return;
    }
    if (!event || typeof event.type !== 'string') return;
    this.handleEvent(event);
  }

  private handleEvent(event: ServerWsEvent): void {
    const { type, payload } = event;
    switch (type) {
      case 'ConnectionEstablished': {
        const p = payload as ConnectionEstablishedPayload;
        this.resumeToken = p.resume_token ?? this.resumeToken;
        this.serverSeq = p.server_seq ?? 0;
        this.pingIntervalMs = Math.max(1, p.config?.ping_interval ?? 30) * 1000;
        this.pingTimeoutMs = Math.max(1, p.config?.ping_timeout ?? 10) * 1000;
        this.reconnectAttempts = 0;
        this.setStatus('connected');
        this.startHeartbeat();
        // 重连成功后用 set 全量重放订阅（本地订阅状态为权威；resume 成功时服务端已恢复，set 幂等无害）
        this.replaySubscriptions();
        break;
      }
      case 'Pong':
        this.clearPongTimer();
        break;
      case 'SessionRevoked': {
        const p = payload as ServerEventPayloadMap['SessionRevoked'];
        this.emitAuthEvent(WS_AUTH_FAILED_EVENT, SESSION_REVOKED_MESSAGES[p?.reason] ?? '会话已失效');
        // 服务端随后以 4001 断开；先标记不重连，等待上层登出
        this.manualClose = true;
        break;
      }
      case 'AccountBanned': {
        const p = payload as ServerEventPayloadMap['AccountBanned'];
        this.emitAuthEvent(WS_AUTH_BANNED_EVENT, p?.reason ?? '账号已被封禁');
        this.manualClose = true;
        break;
      }
      case 'PrivateMessageNew':
        this.emitWindowEvent(WS_PRIVATE_MESSAGE_EVENT, payload);
        break;
      case 'PrivateMessageEdited':
        this.emitWindowEvent(WS_PRIVATE_MESSAGE_UPDATE_EVENT, { type: 'edited', payload });
        break;
      case 'PrivateMessageRecalled':
        this.emitWindowEvent(WS_PRIVATE_MESSAGE_UPDATE_EVENT, { type: 'recalled', payload });
        break;
      case 'PrivateMessageRead':
        this.emitWindowEvent(WS_PRIVATE_MESSAGE_UPDATE_EVENT, { type: 'read', payload });
        break;
      case 'SubscribeAck':
        this.handleSubscribeAck(payload as SubscribeAckPayload);
        break;
      default:
        if (NOTIFY_EVENT_TYPES.has(type)) {
          this.emitWindowEvent(WS_NOTIFY_EVENT, { type, payload });
        }
        break;
    }

    // emitter 分发（on/onAny 全部事件可见）
    this.dispatch(type, payload);
  }

  /** SubscribeAck：failed_ids 记录并 console.warn，同时回滚本地状态保持与服务端一致。 */
  private handleSubscribeAck(payload: SubscribeAckPayload): void {
    const failed = payload.failed_ids ?? [];
    if (payload.code !== 0 && failed.length > 0) {
      console.warn(`[ws] 订阅失败（${payload.operation}）：`, failed, payload.message ?? '');
      for (const f of failed) {
        const set = this.subscriptions[f.type];
        if (set) set.delete(f.subscription_id);
      }
    }
    this.pendingAcks.delete(payload.client_request_id);
  }

  private replaySubscriptions(): void {
    const items: SubscriptionItem[] = (['user', 'group'] as const)
      .map((type) => ({ type, ids: [...this.subscriptions[type]] }))
      .filter((i) => i.ids.length > 0);
    if (items.length === 0) return;
    this.setSubscriptions('set', items);
  }

  // --- 心跳 ---

  private startHeartbeat(): void {
    this.clearTimers();
    this.heartbeatTimer = setInterval(() => this.sendPing(), this.pingIntervalMs);
  }

  private sendPing(): void {
    if (this.ws?.readyState !== WebSocket.OPEN) return;
    this.send({ type: 'Ping', payload: {} });
    this.clearPongTimer();
    this.pongTimer = setTimeout(() => {
      // ping_timeout 内未收到 Pong → 视为断线，触发重连
      this.forceReconnect();
    }, this.pingTimeoutMs);
  }

  private clearPongTimer(): void {
    if (this.pongTimer !== null) {
      clearTimeout(this.pongTimer);
      this.pongTimer = null;
    }
  }

  private forceReconnect(): void {
    const socket = this.ws;
    this.ws = null;
    if (socket) {
      socket.onclose = null;
      try {
        socket.close();
      } catch {
        /* 忽略 */
      }
    }
    this.handleClose(1005);
  }

  // --- 连接生命周期 ---

  private handleClose(code: number): void {
    this.clearTimers();
    this.ws = null;
    const shouldReconnect = !this.manualClose && !NO_RECONNECT_CODES.has(code);
    this.setStatus('disconnected');
    if (shouldReconnect) this.scheduleReconnect();
  }

  /** 指数退避 + 随机 jitter（1s/2s/4s…最大 30s）。 */
  private scheduleReconnect(): void {
    if (typeof window === 'undefined' || this.reconnectTimer !== null) return;
    const { baseBackoffMs, maxBackoffMs } = this.options;
    const exponential = baseBackoffMs * 2 ** this.reconnectAttempts;
    const jitter = Math.random() * 1000;
    const delay = Math.min(exponential, maxBackoffMs) + jitter;
    this.reconnectAttempts += 1;
    this.reconnectTimer = setTimeout(() => {
      this.reconnectTimer = null;
      this.connect();
    }, delay);
  }

  private clearTimers(): void {
    this.clearPongTimer();
    if (this.heartbeatTimer !== null) {
      clearInterval(this.heartbeatTimer);
      this.heartbeatTimer = null;
    }
    if (this.reconnectTimer !== null) {
      clearTimeout(this.reconnectTimer);
      this.reconnectTimer = null;
    }
  }

  // --- 发送与分发 ---

  private send(event: WsEvent<ClientEventType, unknown>): void {
    if (this.ws?.readyState !== WebSocket.OPEN) return;
    this.ws.send(JSON.stringify(event));
  }

  private dispatch(type: ServerEventType, payload: unknown): void {
    const set = this.handlers.get(type);
    if (set) {
      for (const h of [...set]) h(payload);
    }
    if (this.anyHandlers.size > 0) {
      const event = { type, payload } as ServerWsEvent;
      for (const h of [...this.anyHandlers]) h(event);
    }
  }

  private setStatus(status: WsStatus): void {
    if (this.status === status) return;
    this.status = status;
    for (const cb of [...this.statusHandlers]) cb(status);
  }

  /** 对齐 src/api/client.ts 的 auth-failed/auth-banned 事件（auth store 监听）。 */
  private emitAuthEvent(type: 'auth-failed' | 'auth-banned', message: string): void {
    if (typeof window === 'undefined') return;
    window.dispatchEvent(new CustomEvent(type, { detail: { type, message } }));
  }

  private emitWindowEvent(name: string, detail: unknown): void {
    if (typeof window === 'undefined') return;
    window.dispatchEvent(new CustomEvent(name, { detail }));
  }
}

/** 模块单例（SSR 下所有操作 no-op，与 src/api/client.ts 一致）。 */
export const wsClient = new WsClient();

export type { ClientWsEvent, ServerWsEvent };
