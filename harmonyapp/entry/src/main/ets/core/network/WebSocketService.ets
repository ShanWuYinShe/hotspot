/**
 * WebSocket服务管理器
 * 统一管理WebSocket连接和消息处理
 * 严格遵循 api_spec_websocket.md 设计规范
 */

import { WebSocketUtil } from './WebSocketUtil';
import { API_BASE_URL, StorageKeys, WS_BASE_URL } from '../../constants/AppConstants';
import { hilog } from '@kit.PerformanceAnalysisKit';
import { PreferencesUtil } from '../storage/PreferencesUtil';
import {
  ClientEvent,
  ServerEvent,
  ConnectionEstablishedPayload,
  ErrorPayload,
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

const HEARTBEAT_INTERVAL = 45;
const DEFAULT_PONG_TIMEOUT = 10;

export class WebSocketService {
  private static instance: WebSocketService | null = null;
  private webSocketUtil: WebSocketUtil;
  private state: WebSocketState = WebSocketState.DISCONNECTED;
  private reconnectAttempts: number = 0;
  private reconnectConfig: ReconnectConfig = DEFAULT_RECONNECT_CONFIG;
  private reconnectTimer: number | null = null;
  private heartbeatTimer: number | null = null;
  private pongTimeoutTimer: number | null = null;
  private heartbeatIntervalSec: number = HEARTBEAT_INTERVAL;
  private pongTimeoutSec: number = DEFAULT_PONG_TIMEOUT;
  private awaitingPong: boolean = false;
  private currentToken: string = '';
  private subscriptions: SubscriptionItem[] = [];
  private eventHandlers: Map<string, ((event: ServerEvent) => void)[]> = new Map();
  private lastServerTimeMs: number = 0;
  private lastDisconnectDurationMs: number = 0;

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
      this.webSocketUtil.sendRaw(serializeClientEvent(createAuthEvent('app', token)));
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
      this.currentToken = '';
      this.awaitingPong = false;
      this.lastDisconnectDurationMs = 0;
      this.lastServerTimeMs = 0;
      hilog.info(DOMAIN_NUMBER, TAG, 'WebSocket连接已断开');
    } catch (error) {
      hilog.error(DOMAIN_NUMBER, TAG, `断开WebSocket连接失败: ${JSON.stringify(error)}`);
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
    this.send(createSetSubscriptionsEvent('subscribe', normalizedItem));
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
    this.send(createSetSubscriptionsEvent('unsubscribe', normalizedItem));
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
    this.send(createSetSubscriptionsEvent(operation, normalizedItem));
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

  private rollbackSubscriptionByFailedIds(
    operation: SubscriptionOperation,
    type: 'group' | 'user',
    failedIds: string[],
    serverIds: string[]
  ): void {
    const normalizedFailedIds = this.normalizeIds(failedIds);
    if (normalizedFailedIds.length === 0) {
      this.setSubscriptionIdsByType(type, serverIds);
      return;
    }

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
      return;
    }

    const successfulIds = this
      .normalizeIds(serverIds)
      .filter((id: string) => normalizedFailedIds.indexOf(id) === -1);
    this.setSubscriptionIdsByType(type, successfulIds);
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

    this.webSocketUtil.onClose(() => {
      hilog.info(DOMAIN_NUMBER, TAG, 'WebSocket连接关闭');
      this.state = WebSocketState.DISCONNECTED;
      this.stopHeartbeat();
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

        this.state = WebSocketState.CONNECTED;
        this.reconnectAttempts = 0;
        this.clearReconnectTimer();
        this.awaitingPong = false;

        this.heartbeatIntervalSec = connPayload.config?.ping_interval || HEARTBEAT_INTERVAL;
        this.pongTimeoutSec = connPayload.config?.ping_timeout || DEFAULT_PONG_TIMEOUT;
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
        const failedIds: string[] = failedItems.map((item: FailedSubscriptionId): string => item.subscription_id);
        if (failedIds.length > 0) {
          this.rollbackSubscriptionByFailedIds(
            ackPayload.operation,
            ackPayload.type,
            failedIds,
            ackPayload.ids
          );
        } else {
          this.setSubscriptionIdsByType(ackPayload.type, ackPayload.ids);
        }
        const effectiveIds = this.getSubscriptionIdsByType(ackPayload.type);
        if (ackPayload.code === 0) {
          hilog.info(DOMAIN_NUMBER, TAG, `订阅成功: operation=${ackPayload.operation}, type=${ackPayload.type}, ids=${JSON.stringify(effectiveIds)}`);
        } else if (ackPayload.failed_ids && ackPayload.failed_ids.length > 0) {
          hilog.warn(
            DOMAIN_NUMBER,
            TAG,
            `订阅部分失败: operation=${ackPayload.operation}, type=${ackPayload.type}, failed=${JSON.stringify(ackPayload.failed_ids)}, current=${JSON.stringify(effectiveIds)}`
          );
        } else {
          const message = typeof ackPayload.message === 'string' ? ackPayload.message : '';
          hilog.error(DOMAIN_NUMBER, TAG, `订阅失败: operation=${ackPayload.operation}, type=${ackPayload.type}, message=${message}`);
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
          hilog.error(DOMAIN_NUMBER, TAG, '收到致命错误，断开连接');
          this.disconnect();
        }
        break;

      case 'AccountBanned':
        hilog.error(DOMAIN_NUMBER, TAG, `账号被封禁: ${JSON.stringify(event.payload)}`);
        this.disconnect();
        break;

      // 多端同步事件（⏳ 延期实现，由 eventHandlers 分发处理）
      case 'SelfReadSync':
      case 'SelfChatSettingsSync':
      case 'SelfSettingsUpdated':
      case 'SelfRelationSync':
        hilog.debug(DOMAIN_NUMBER, TAG, `多端同步事件: ${event.type}`);
        break;
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
    const pingInterval = interval || this.heartbeatIntervalSec || HEARTBEAT_INTERVAL;
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
    const timeoutSec = this.pongTimeoutSec > 0 ? this.pongTimeoutSec : DEFAULT_PONG_TIMEOUT;
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
      const freshToken = await PreferencesUtil.get<string>(StorageKeys.ACCESS_TOKEN, '');
      const tokenToUse = freshToken || this.currentToken;

      this.connect(tokenToUse).catch((error: Error) => {
        hilog.error(DOMAIN_NUMBER, TAG, `重连失败: ${error.message}`);
      });
    }, delay);
  }
}
