/**
 * WebSocket服务管理器
 * 统一管理WebSocket连接和消息处理
 * 严格遵循 api_spec_websocket.md 设计规范
 */

import { WebSocketUtil } from './WebSocketUtil';
import { HttpUtil } from './HttpUtil';
import { AppRouter, LoginRouteParam } from '../router/AppRouter';
import { API_BASE_URL, PageNames, StorageKeys, WS_BASE_URL } from '../../constants/AppConstants';
import { hilog } from '@kit.PerformanceAnalysisKit';
import { PreferencesUtil } from '../storage/PreferencesUtil';
import { TokenAssetStore } from '../storage/TokenAssetStore';
import { InteractionStateStore } from '../state/InteractionStateStore';
import { DeviceIdUtil } from '../util/DeviceIdUtil';
import {
  ClientEvent,
  ServerEvent,
  ConnectionEstablishedPayload,
  ErrorPayload,
  AccountBannedPayload,
  SessionRevokedPayload,
  FailedSubscriptionId,
  SubscribeAckPayload,
  SubscriptionItem,
  SubscriptionOperation,
  WebSocketState,
  ReconnectConfig,
  createAuthEvent,
  createPingEvent,
  createSetSubscriptionsEvent,
  createGetSubscriptionsEvent,
  createPrivateTypingEvent,
  createGroupTypingEvent,
  parseServerEvent,
  serializeClientEvent
} from '../types/WebSocketTypes';

const DOMAIN_NUMBER: number = 0xFF06;
const TAG: string = 'WebSocketService';

const DEFAULT_RECONNECT_CONFIG: ReconnectConfig = {
  baseDelay: 1000,
  maxDelay: 30000,
  maxAttempts: 10,
  jitterFactor: 0.2
};

// Spec 参考值（仅作 fallback，优先使用 ConnectionEstablished 返回的服务端动态配置）
const SPEC_REFERENCE_PING_INTERVAL = 30;
const SPEC_REFERENCE_PONG_TIMEOUT = 10;

export class WebSocketService {
  private static instance: WebSocketService | null = null;
  private webSocketUtil: WebSocketUtil;
  private state: WebSocketState = WebSocketState.DISCONNECTED;
  private reconnectAttempts: number = 0;
  private reconnectConfig: ReconnectConfig = DEFAULT_RECONNECT_CONFIG;
  private reconnectTimer: number | null = null;
  private heartbeatTimer: number | null = null;
  private pongTimeoutTimer: number | null = null;
  private heartbeatIntervalSec: number = SPEC_REFERENCE_PING_INTERVAL;
  private pongTimeoutSec: number = SPEC_REFERENCE_PONG_TIMEOUT;
  private awaitingPong: boolean = false;
  private currentToken: string = '';
  // resume 增量恢复（spec L155-156）：上次连接的恢复令牌与服务端序号基线。
  // CE 下发时更新，重连 Auth 时透传；仅服务端主动断线的自动重连保留（同一会话
  // 续传），手动 disconnect() 一并清除——resume_token 绑定旧会话，登出/切换账号
  // 后新会话携带它属语义错误（服务端虽会拒绝并回退全量恢复，但不应发送）。
  private resumeToken: string = '';
  // server_seq 跨端为字符串（ArkTS/JS number 无法无损表示 int64，spec L330），
  // 仅保存/透传基线，不做数值运算。
  private serverSeq: string = '';
  private subscriptions: SubscriptionItem[] = [];
  private eventHandlers: Map<string, ((event: ServerEvent) => void)[]> = new Map();
  private lastServerTimeMs: number = 0;
  private lastDisconnectDurationMs: number = 0;
  private interactionStore = InteractionStateStore.getInstance();

  private constructor() {
    this.webSocketUtil = WebSocketUtil.getInstance();
    this.setupInternalHandlers();
  }

  static getInstance(): WebSocketService {
    if (!WebSocketService.instance) {
      WebSocketService.instance = new WebSocketService();
    }
    return WebSocketService.instance;
  }

  async connect(token: string): Promise<void> {
    if (this.state === WebSocketState.CONNECTED || this.state === WebSocketState.CONNECTING) {
      hilog.warn(DOMAIN_NUMBER, TAG, 'WebSocket已连接或正在连接中');
      return;
    }

    this.currentToken = token;
    this.state = WebSocketState.CONNECTING;
    this.clearReconnectTimer();

    try {
      const wsUrl = await this.getWsUrl();
      hilog.info(DOMAIN_NUMBER, TAG, `正在连接WebSocket服务器: ${wsUrl}`);
      await this.webSocketUtil.connect(wsUrl);
      this.webSocketUtil.sendRaw(serializeClientEvent(createAuthEvent('app', token, this.resumeToken, DeviceIdUtil.getDeviceId())));
      hilog.info(DOMAIN_NUMBER, TAG, 'WebSocket握手成功，已发送Auth，等待ConnectionEstablished');
    } catch (error) {
      hilog.error(DOMAIN_NUMBER, TAG, `WebSocket连接失败: ${JSON.stringify(error)}`);
      this.state = WebSocketState.ERROR;
      this.scheduleReconnect();
      throw new Error('WebSocket连接失败');
    }
  }

  private async getWsUrl(): Promise<string> {
    try {
      const baseUrl = await PreferencesUtil.get<string>(StorageKeys.SERVER_BASE_URL, API_BASE_URL);
      const normalized = typeof baseUrl === 'string' && baseUrl.trim().length > 0 ? baseUrl.trim() : API_BASE_URL;
      let wsUrl = normalized.replace(/^http:/, 'ws:').replace(/^https:/, 'wss:');
      if (!wsUrl.endsWith('/ws/v1')) wsUrl = wsUrl.replace(/\/+$/, '') + '/ws/v1';
      return wsUrl;
    } catch {
      return WS_BASE_URL;
    }
  }

  disconnect(): void {
    try {
      this.stopHeartbeat();
      this.clearReconnectTimer();
      this.webSocketUtil.disconnect();
      this.state = WebSocketState.DISCONNECTED;
      this.reconnectAttempts = 0;
      this.subscriptions = [];
      this.resumeToken = '';
      this.currentToken = '';
      this.awaitingPong = false;
      this.lastDisconnectDurationMs = 0;
      this.lastServerTimeMs = 0;
      hilog.info(DOMAIN_NUMBER, TAG, 'WebSocket连接已断开');
    } catch (error) {
      hilog.error(DOMAIN_NUMBER, TAG, `断开WebSocket连接失败: ${JSON.stringify(error)}`);
    }
  }

  // 与 HttpUtil 的 10201 强制登出链路保持一致：清认证数据 + 跳登录页。
  // 用于 WebSocket 通道收到的 fatal Error / AccountBanned（服务端主动断开场景），
  // 确保 WS 侧的封禁/失效与 HTTP 侧处理一致，避免本地 token 残留导致状态错乱。
  private async forceLogout(reason: string, action: string): Promise<void> {
    try {
      await HttpUtil.clearAuthData();
      const param = new LoginRouteParam();
      param.reason = reason;
      param.action = action;
      AppRouter.clear();
      AppRouter.push(PageNames.LOGIN, param);
    } catch (error) {
      hilog.error(DOMAIN_NUMBER, TAG, `强制登出失败: ${JSON.stringify(error)}`);
    }
  }

  getState(): WebSocketState {
    return this.state;
  }

  isConnected(): boolean {
    return this.state === WebSocketState.CONNECTED;
  }

  getLastDisconnectDurationMs(): number {
    return this.lastDisconnectDurationMs;
  }

  // ============================================================================
  // 订阅管理（统一使用 SetSubscriptions + operation）
  // ============================================================================

  private normalizeIds(ids: string[]): string[] {
    const unique: string[] = [];
    ids.forEach((id: string) => {
      if (typeof id !== 'string') {
        return;
      }
      const normalized = id.trim();
      if (normalized.length === 0) {
        return;
      }
      if (unique.indexOf(normalized) === -1) {
        unique.push(normalized);
      }
    });
    return unique;
  }

  private mergeLocalSubscription(item: SubscriptionItem): void {
    const existingIndex = this.subscriptions.findIndex((sub) => sub.type === item.type);
    if (existingIndex < 0) {
      this.subscriptions.push({ type: item.type, ids: item.ids });
      return;
    }
    const merged = this.normalizeIds(this.subscriptions[existingIndex].ids.concat(item.ids));
    this.subscriptions[existingIndex] = { type: item.type, ids: merged };
  }

  private removeLocalSubscription(item: SubscriptionItem): void {
    const existingIndex = this.subscriptions.findIndex((sub) => sub.type === item.type);
    if (existingIndex < 0) {
      return;
    }
    const filtered = this.subscriptions[existingIndex].ids.filter((id: string) => item.ids.indexOf(id) === -1);
    if (filtered.length === 0) {
      this.subscriptions.splice(existingIndex, 1);
      return;
    }
    this.subscriptions[existingIndex] = { type: item.type, ids: filtered };
  }

  subscribe(item: SubscriptionItem): void {
    const normalizedItem: SubscriptionItem = {
      type: item.type,
      ids: this.normalizeIds(item.ids)
    };
    if (normalizedItem.ids.length === 0) {
      return;
    }
    this.mergeLocalSubscription(normalizedItem);
    if (!this.isConnected()) {
      hilog.info(DOMAIN_NUMBER, TAG, `WebSocket未连接，订阅请求已入队: type=${normalizedItem.type}, ids=${JSON.stringify(normalizedItem.ids)}`);
      return;
    }
    this.send(createSetSubscriptionsEvent('subscribe', [normalizedItem]));
  }

  unsubscribe(item: SubscriptionItem): void {
    const normalizedItem: SubscriptionItem = {
      type: item.type,
      ids: this.normalizeIds(item.ids)
    };
    if (normalizedItem.ids.length === 0) {
      return;
    }
    this.removeLocalSubscription(normalizedItem);
    if (!this.isConnected()) {
      hilog.info(DOMAIN_NUMBER, TAG, `WebSocket未连接，取消订阅请求已入队: type=${normalizedItem.type}, ids=${JSON.stringify(normalizedItem.ids)}`);
      return;
    }
    this.send(createSetSubscriptionsEvent('unsubscribe', [normalizedItem]));
  }

  setSubscriptions(item: SubscriptionItem, operation: SubscriptionOperation = 'set'): void {
    const normalizedItem: SubscriptionItem = {
      type: item.type,
      ids: this.normalizeIds(item.ids)
    };
    if ((operation === 'subscribe' || operation === 'unsubscribe') && normalizedItem.ids.length === 0) {
      return;
    }
    if (operation === 'subscribe') {
      this.mergeLocalSubscription(normalizedItem);
    } else if (operation === 'unsubscribe') {
      this.removeLocalSubscription(normalizedItem);
    } else {
      this.upsertSubscription(normalizedItem);
    }
    if (!this.isConnected()) {
      hilog.info(DOMAIN_NUMBER, TAG, `WebSocket未连接，set订阅请求已入队: operation=${operation}, type=${normalizedItem.type}, ids=${JSON.stringify(normalizedItem.ids)}`);
      return;
    }
    this.send(createSetSubscriptionsEvent(operation, [normalizedItem]));
  }

  getSubscriptionsState(): void {
    if (!this.isConnected()) {
      hilog.warn(DOMAIN_NUMBER, TAG, 'WebSocket未连接，无法获取订阅状态');
      return;
    }
    this.send(createGetSubscriptionsEvent());
  }

  private upsertSubscription(item: SubscriptionItem): void {
    const existingIndex = this.subscriptions.findIndex((sub) => sub.type === item.type);
    if (item.ids.length === 0) {
      if (existingIndex >= 0) {
        this.subscriptions.splice(existingIndex, 1);
      }
      return;
    }
    if (existingIndex >= 0) {
      this.subscriptions[existingIndex] = item;
    } else {
      this.subscriptions.push(item);
    }
  }

  private getSubscriptionIdsByType(type: 'group' | 'user'): string[] {
    const item = this.subscriptions.find((sub) => sub.type === type);
    if (!item) return [];
    return this.normalizeIds(item.ids);
  }

  private setSubscriptionIdsByType(type: 'group' | 'user', ids: string[]): void {
    this.upsertSubscription({
      type,
      ids: this.normalizeIds(ids)
    });
  }

  // 订阅回滚：按失败项的 type + subscription_id 修正本地预判状态。
  // - subscribe 失败：本地之前 merge 了失败项，现移除；
  // - unsubscribe 失败：本地之前 remove 了失败项，现加回（仍处于订阅中）；
  // - set：失败项不在 accepted 内，已由 accepted 覆盖时排除，此处无需处理。
  private rollbackFailedIdsByType(
    operation: SubscriptionOperation,
    type: 'group' | 'user',
    failedIds: string[]
  ): void {
    const normalizedFailedIds = this.normalizeIds(failedIds);
    if (normalizedFailedIds.length === 0) return;

    if (operation === 'subscribe') {
      const currentIds = this.getSubscriptionIdsByType(type);
      this.setSubscriptionIdsByType(
        type,
        currentIds.filter((id: string) => normalizedFailedIds.indexOf(id) === -1)
      );
      return;
    }

    if (operation === 'unsubscribe') {
      const currentIds = this.getSubscriptionIdsByType(type);
      this.setSubscriptionIdsByType(type, this.normalizeIds(currentIds.concat(normalizedFailedIds)));
    }
  }

  subscribeGroupChats(groupChatIds: string[]): void {
    this.subscribe({ type: 'group', ids: groupChatIds });
  }

  unsubscribeGroupChats(groupChatIds: string[]): void {
    this.unsubscribe({ type: 'group', ids: groupChatIds });
  }

  subscribeUsers(userIds: string[]): void {
    this.subscribe({ type: 'user', ids: userIds });
  }

  unsubscribeUsers(userIds: string[]): void {
    this.unsubscribe({ type: 'user', ids: userIds });
  }

  // ============================================================================
  // 实时状态同步
  // ============================================================================

  sendPrivateTyping(targetUserId: string, isTyping: boolean): void {
    if (!this.isConnected()) return;
    this.send(createPrivateTypingEvent(targetUserId, isTyping));
  }

  sendGroupTyping(groupChatId: string, isTyping: boolean): void {
    if (!this.isConnected()) return;
    this.send(createGroupTypingEvent(groupChatId, isTyping));
  }

  // ============================================================================
  // 事件监听
  // ============================================================================

  on(eventType: string, handler: (event: ServerEvent) => void): void {
    const handlers = this.eventHandlers.get(eventType) || [];
    if (handlers.indexOf(handler) !== -1) {
      return;
    }
    handlers.push(handler);
    this.eventHandlers.set(eventType, handlers);
  }

  off(eventType: string, handler: (event: ServerEvent) => void): void {
    const handlers = this.eventHandlers.get(eventType) || [];
    const index = handlers.indexOf(handler);
    if (index !== -1) {
      handlers.splice(index, 1);
      if (handlers.length === 0) {
        this.eventHandlers.delete(eventType);
      } else {
        this.eventHandlers.set(eventType, handlers);
      }
    }
  }

  /**
   * 移除某事件类型下所有监听器，组件在 aboutToDisappear 时调用可避免内存泄漏
   */
  offAll(eventType: string): void {
    this.eventHandlers.delete(eventType);
  }

  // ============================================================================
  // 内部方法
  // ============================================================================

  private send(event: ClientEvent): void {
    if (!this.isConnected()) {
      hilog.warn(DOMAIN_NUMBER, TAG, 'WebSocket未连接，无法发送消息');
      return;
    }

    try {
      const data = serializeClientEvent(event);
      this.webSocketUtil.sendRaw(data);
      hilog.debug(DOMAIN_NUMBER, TAG, `发送WebSocket消息: ${event.type}`);
    } catch (error) {
      hilog.error(DOMAIN_NUMBER, TAG, `发送WebSocket消息失败: ${JSON.stringify(error)}`);
    }
  }

  private setupInternalHandlers(): void {
    this.webSocketUtil.onRawMessage((data: string) => {
      const event = parseServerEvent(data);
      if (event) {
        this.handleServerEvent(event as ServerEvent);
      }
    });

    this.webSocketUtil.onClose((code: number) => {
      hilog.info(DOMAIN_NUMBER, TAG, `WebSocket连接关闭: code=${code}`);
      this.state = WebSocketState.DISCONNECTED;
      this.stopHeartbeat();
      // Close code 4001：认证失败/Token 失效/会话被驱逐（SessionRevoked）。
      // 该场景已由 SessionRevoked/Error 事件处理（清本地态 + 提示），
      // 重连由用户重新登录触发，此处不自动重连（见 spec「Close Code 规范」）。
      if (code === 4001) {
        hilog.info(DOMAIN_NUMBER, TAG, '收到 close code 4001（认证/会话失效），停止自动重连');
        this.clearReconnectTimer();
        return;
      }
      this.scheduleReconnect();
    });

    this.webSocketUtil.onError((error: Error) => {
      hilog.error(DOMAIN_NUMBER, TAG, `WebSocket错误: ${error.message}`);
      this.state = WebSocketState.ERROR;
      this.stopHeartbeat();
      this.scheduleReconnect();
    });
  }

  private handleServerEvent(event: ServerEvent): void {
    hilog.debug(DOMAIN_NUMBER, TAG, `收到服务端事件: ${event.type}`);

    switch (event.type) {
      case 'ConnectionEstablished':
        const connPayload = event.payload as ConnectionEstablishedPayload;
        hilog.info(DOMAIN_NUMBER, TAG, `连接已建立: connection_id=${connPayload.connection_id}, user_id=${connPayload.user_id}`);
        hilog.debug(DOMAIN_NUMBER, TAG, `服务端时间: ${connPayload.server_time}`);
        hilog.debug(DOMAIN_NUMBER, TAG, `配置: ping_interval=${connPayload.config.ping_interval}s, ping_timeout=${connPayload.config.ping_timeout}s`);

        // resume（spec L155-156）：记录本次连接令牌与序号基线，重连时透传
        // resume_token 尝试增量续传；服务端未下发（Redis 不可用降级）时保留旧值。
        this.resumeToken = connPayload.resume_token || this.resumeToken;
        this.serverSeq = connPayload.server_seq || this.serverSeq;
        hilog.debug(DOMAIN_NUMBER, TAG, `resume: token=${this.resumeToken ? 'set' : 'none'}, server_seq=${this.serverSeq}`);

        this.state = WebSocketState.CONNECTED;
        this.reconnectAttempts = 0;
        this.clearReconnectTimer();
        this.awaitingPong = false;

        this.heartbeatIntervalSec = connPayload.config?.ping_interval || SPEC_REFERENCE_PING_INTERVAL;
        this.pongTimeoutSec = connPayload.config?.ping_timeout || SPEC_REFERENCE_PONG_TIMEOUT;
        const currentServerTimeMs = this.parseServerTimeMs(connPayload.server_time);
        if (currentServerTimeMs > 0 && this.lastServerTimeMs > 0 && currentServerTimeMs > this.lastServerTimeMs) {
          this.lastDisconnectDurationMs = currentServerTimeMs - this.lastServerTimeMs;
        } else {
          this.lastDisconnectDurationMs = 0;
        }
        if (currentServerTimeMs > 0) {
          this.lastServerTimeMs = currentServerTimeMs;
        }
        hilog.info(
          DOMAIN_NUMBER,
          TAG,
          `重连断开时长: ${this.lastDisconnectDurationMs}ms`
        );
        this.startHeartbeat(this.heartbeatIntervalSec);

        if (this.subscriptions.length > 0) {
          this.subscriptions.forEach((item) => this.setSubscriptions(item));
        }
        break;

      case 'Pong':
        hilog.debug(DOMAIN_NUMBER, TAG, '收到服务端Pong');
        this.awaitingPong = false;
        this.clearPongTimeout();
        break;

      case 'SubscribeAck':
        const ackPayload = event.payload as SubscribeAckPayload;
        const failedItems: FailedSubscriptionId[] = ackPayload.failed_ids ?? [];

        // set 操作：accepted 是该批 type 的最终权威列表，按 type 覆盖本地状态。
        // subscribe/unsubscribe 为增量：本地已预先合并/移除，无需再用 accepted 刷新。
        if (ackPayload.operation === 'set') {
          ackPayload.accepted.forEach((acceptedItem: SubscriptionItem) => {
            this.setSubscriptionIdsByType(acceptedItem.type, acceptedItem.ids);
          });
        }

        // failed_ids 回滚本地预判（每项含 type + subscription_id，按 type 分组处理）
        if (failedItems.length > 0) {
          const failedByType = new Map<'group' | 'user', string[]>();
          failedItems.forEach((f: FailedSubscriptionId) => {
            const list = failedByType.get(f.type) ?? [];
            list.push(f.subscription_id);
            failedByType.set(f.type, list);
          });
          failedByType.forEach((ids: string[], type: 'group' | 'user') => {
            this.rollbackFailedIdsByType(ackPayload.operation, type, ids);
          });
        }

        if (ackPayload.code === 0) {
          hilog.info(DOMAIN_NUMBER, TAG, `订阅成功: operation=${ackPayload.operation}, accepted=${JSON.stringify(ackPayload.accepted)}`);
        } else if (failedItems.length > 0) {
          hilog.warn(DOMAIN_NUMBER, TAG, `订阅部分失败: operation=${ackPayload.operation}, failed=${JSON.stringify(failedItems)}`);
        } else {
          hilog.error(DOMAIN_NUMBER, TAG, `订阅失败: operation=${ackPayload.operation}, message=${ackPayload.message ?? ''}`);
        }
        break;

      case 'SubscriptionsSnapshot':
        this.subscriptions = [];
        if (event.payload.user_ids?.length > 0) {
          this.subscriptions.push({ type: 'user', ids: this.normalizeIds(event.payload.user_ids) });
        }
        if (event.payload.group_ids?.length > 0) {
          this.subscriptions.push({ type: 'group', ids: this.normalizeIds(event.payload.group_ids) });
        }
        hilog.info(DOMAIN_NUMBER, TAG, `订阅快照更新: ${JSON.stringify(this.subscriptions)}`);
        break;

      case 'Error':
        const errPayload = event.payload as ErrorPayload;
        hilog.error(DOMAIN_NUMBER, TAG, `服务端错误: [${errPayload.code}] ${errPayload.message} severity=${errPayload.severity}`);
        if (errPayload.severity === 'fatal') {
          hilog.error(DOMAIN_NUMBER, TAG, '收到致命错误，断开连接并强制登出');
          this.disconnect();
          this.forceLogout('banned', errPayload.message || '账号已被封禁');
        }
        break;

      case 'AccountBanned':
        hilog.error(DOMAIN_NUMBER, TAG, `账号被封禁: ${JSON.stringify(event.payload)}`);
        this.disconnect();
        const bannedPayload = event.payload as AccountBannedPayload;
        this.forceLogout('banned', `账号已被封禁：${bannedPayload.reason}`);
        break;

      case 'SessionRevoked': {
        // 会话被驱逐/撤销：按 reason 分支提示，清本地态并主动断开、禁止自动重连。
        // 服务端推送此事件后以 close code 4001 断开（onClose 中对 4001 不自动重连）。
        const revokedPayload = event.payload as SessionRevokedPayload;
        let revokeMessage = '会话已失效';
        if (revokedPayload.reason === 'device_evicted') {
          revokeMessage = '账号在其他设备登录';
        } else if (revokedPayload.reason === 'password_changed') {
          revokeMessage = '密码已修改';
        } else {
          revokeMessage = '会话已失效';
        }
        hilog.warn(DOMAIN_NUMBER, TAG, `会话被撤销: reason=${revokedPayload.reason}, revoked_at=${revokedPayload.revoked_at}`);
        this.disconnect();
        this.forceLogout('session_revoked', revokeMessage);
        break;
      }
    }

    // 触发事件处理器
    const handlers = this.eventHandlers.get(event.type) || [];
    handlers.forEach(handler => {
      try {
        handler(event as ServerEvent);
      } catch (error) {
        hilog.error(DOMAIN_NUMBER, TAG, `事件处理器错误: ${JSON.stringify(error)}`);
      }
    });

    // 触发通用处理器
    const allHandlers = this.eventHandlers.get('*') || [];
    allHandlers.forEach(handler => {
      try {
        handler(event as ServerEvent);
      } catch (error) {
        hilog.error(DOMAIN_NUMBER, TAG, `通用事件处理器错误: ${JSON.stringify(error)}`);
      }
    });
  }

  private startHeartbeat(interval?: number): void {
    this.stopHeartbeat();
    const pingInterval = interval || this.heartbeatIntervalSec || SPEC_REFERENCE_PING_INTERVAL;
    this.heartbeatTimer = setInterval(() => {
      if (this.isConnected()) {
        this.send(createPingEvent());
        this.awaitingPong = true;
        this.startPongTimeout();
      }
    }, pingInterval * 1000);
  }

  private stopHeartbeat(): void {
    if (this.heartbeatTimer !== null) {
      clearInterval(this.heartbeatTimer);
      this.heartbeatTimer = null;
    }
    this.awaitingPong = false;
    this.clearPongTimeout();
  }

  private startPongTimeout(): void {
    this.clearPongTimeout();
    const timeoutSec = this.pongTimeoutSec > 0 ? this.pongTimeoutSec : SPEC_REFERENCE_PONG_TIMEOUT;
    this.pongTimeoutTimer = setTimeout(() => {
      if (!this.awaitingPong) {
        return;
      }
      this.awaitingPong = false;
      hilog.warn(DOMAIN_NUMBER, TAG, `Pong超时(${timeoutSec}s)，触发重连`);
      this.state = WebSocketState.ERROR;
      this.stopHeartbeat();
      this.webSocketUtil.disconnect();
      this.scheduleReconnect();
    }, timeoutSec * 1000);
  }

  private clearPongTimeout(): void {
    if (this.pongTimeoutTimer !== null) {
      clearTimeout(this.pongTimeoutTimer);
      this.pongTimeoutTimer = null;
    }
  }

  private clearReconnectTimer(): void {
    if (this.reconnectTimer !== null) {
      clearTimeout(this.reconnectTimer);
      this.reconnectTimer = null;
    }
  }

  private parseServerTimeMs(serverTime: string): number {
    const parsed = Date.parse(serverTime);
    if (Number.isNaN(parsed)) {
      return 0;
    }
    return parsed;
  }

  private scheduleReconnect(): void {
    if (this.reconnectTimer !== null) {
      hilog.debug(DOMAIN_NUMBER, TAG, '重连任务已存在，跳过重复调度');
      return;
    }

    if (this.reconnectAttempts >= this.reconnectConfig.maxAttempts) {
      hilog.error(DOMAIN_NUMBER, TAG, '达到最大重连次数，停止重连');
      return;
    }

    if (!this.currentToken) {
      hilog.warn(DOMAIN_NUMBER, TAG, '没有有效的Token，无法重连');
      return;
    }

    this.reconnectAttempts++;

    const baseDelay = this.reconnectConfig.baseDelay;
    const maxDelay = this.reconnectConfig.maxDelay;
    const jitterFactor = this.reconnectConfig.jitterFactor;

    let delay = Math.min(baseDelay * Math.pow(2, this.reconnectAttempts - 1), maxDelay);
    const jitter = delay * jitterFactor * (Math.random() * 2 - 1);
    delay = Math.round(delay + jitter);

    hilog.info(DOMAIN_NUMBER, TAG, `${delay}ms后进行第${this.reconnectAttempts}次重连`);

    this.reconnectTimer = setTimeout(async () => {
      this.reconnectTimer = null;
      // 重连前重新读取最新 token（可能已被 HttpUtil 的 refresh 逻辑更新）
      const freshToken = (await TokenAssetStore.get(TokenAssetStore.ALIAS_CURRENT_ACCESS)) ?? '';
      const tokenToUse = freshToken || this.currentToken;

      this.connect(tokenToUse).catch((error: Error) => {
        hilog.error(DOMAIN_NUMBER, TAG, `重连失败: ${error.message}`);
      });
    }, delay);
  }
}
