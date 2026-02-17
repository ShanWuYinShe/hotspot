/**
 * WebSocket服务管理器
 * 统一管理WebSocket连接和消息处理
 * 严格遵循 api_spec_websocket.mdc 设计规范
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

export class WebSocketService {
  private static instance: WebSocketService | null = null;
  private webSocketUtil: WebSocketUtil;
  private state: WebSocketState = WebSocketState.DISCONNECTED;
  private reconnectAttempts: number = 0;
  private reconnectConfig: ReconnectConfig = DEFAULT_RECONNECT_CONFIG;
  private heartbeatTimer: number | null = null;
  private currentToken: string = '';
  private subscriptions: SubscriptionItem[] = [];
  private eventHandlers: Map<string, ((event: ServerEvent) => void)[]> = new Map();

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
      if (!wsUrl.endsWith('/ws')) wsUrl = wsUrl.replace(/\/+$/, '') + '/ws';
      return wsUrl;
    } catch {
      return WS_BASE_URL;
    }
  }

  disconnect(): void {
    try {
      this.stopHeartbeat();
      this.webSocketUtil.disconnect();
      this.state = WebSocketState.DISCONNECTED;
      this.reconnectAttempts = 0;
      this.subscriptions = [];
      this.currentToken = '';
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

  // ============================================================================
  // 订阅管理（统一使用 SetSubscriptions + operation）
  // ============================================================================

  subscribe(item: SubscriptionItem): void {
    if (!this.isConnected()) {
      hilog.warn(DOMAIN_NUMBER, TAG, 'WebSocket未连接，无法订阅');
      return;
    }
    this.send(createSetSubscriptionsEvent('subscribe', item));
  }

  unsubscribe(item: SubscriptionItem): void {
    if (!this.isConnected()) {
      hilog.warn(DOMAIN_NUMBER, TAG, 'WebSocket未连接，无法取消订阅');
      return;
    }
    this.send(createSetSubscriptionsEvent('unsubscribe', item));
  }

  setSubscriptions(item: SubscriptionItem, operation: SubscriptionOperation = 'set'): void {
    if (!this.isConnected()) {
      hilog.warn(DOMAIN_NUMBER, TAG, 'WebSocket未连接，无法设置订阅');
      return;
    }
    this.send(createSetSubscriptionsEvent(operation, item));
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
    handlers.push(handler);
    this.eventHandlers.set(eventType, handlers);
  }

  off(eventType: string, handler: (event: ServerEvent) => void): void {
    const handlers = this.eventHandlers.get(eventType) || [];
    const index = handlers.indexOf(handler);
    if (index !== -1) {
      handlers.splice(index, 1);
      this.eventHandlers.set(eventType, handlers);
    }
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

        const pingInterval = connPayload.config?.ping_interval || HEARTBEAT_INTERVAL;
        this.startHeartbeat(pingInterval);

        if (this.subscriptions.length > 0) {
          this.subscriptions.forEach((item) => this.setSubscriptions(item));
        }
        break;

      case 'Pong':
        hilog.debug(DOMAIN_NUMBER, TAG, '收到服务端Pong');
        break;

      case 'SubscribeAck':
        if (event.payload.code === 0) {
          this.upsertSubscription({
            type: event.payload.type,
            ids: event.payload.ids
          });
          hilog.info(DOMAIN_NUMBER, TAG, `订阅成功: ${JSON.stringify(this.subscriptions)}`);
        } else {
          hilog.error(DOMAIN_NUMBER, TAG, `订阅失败: ${event.payload.message}`);
        }
        break;

      case 'SubscriptionsSnapshot':
        this.subscriptions = [];
        if (event.payload.user_ids?.length > 0) {
          this.subscriptions.push({ type: 'user', ids: event.payload.user_ids });
        }
        if (event.payload.group_ids?.length > 0) {
          this.subscriptions.push({ type: 'group', ids: event.payload.group_ids });
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
    const pingInterval = interval || HEARTBEAT_INTERVAL;
    this.heartbeatTimer = setInterval(() => {
      if (this.isConnected()) {
        this.send(createPingEvent());
      }
    }, pingInterval * 1000);
  }

  private stopHeartbeat(): void {
    if (this.heartbeatTimer !== null) {
      clearInterval(this.heartbeatTimer);
      this.heartbeatTimer = null;
    }
  }

  private scheduleReconnect(): void {
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

    setTimeout(() => {
      this.connect(this.currentToken).catch((error: Error) => {
        hilog.error(DOMAIN_NUMBER, TAG, `重连失败: ${error.message}`);
      });
    }, delay);
  }
}
