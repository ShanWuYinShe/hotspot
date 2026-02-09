/**
 * WebSocket服务管理器
 * 统一管理WebSocket连接和消息处理
 * 严格遵循 api_spec.mdc WebSocket 设计规范
 */

import { WebSocketUtil } from './WebSocketUtil';
import { API_BASE_URL, StorageKeys, WS_BASE_URL } from '../../constants/AppConstants';
import { hilog } from '@kit.PerformanceAnalysisKit';
import { PreferencesUtil } from '../storage/PreferencesUtil';
import {
  ClientEvent,
  ServerEvent,
  BatchEventsEvent,
  ConnectionEstablishedPayload,
  SubscriptionItem,
  WebSocketState,
  ReconnectConfig,
  createAuthEvent,
  createPingEvent,
  createSubscribeEvent,
  createUnsubscribeEvent,
  createSetSubscriptionsEvent,
  createGetSubscriptionsEvent,
  createPrivateTypingEvent,
  createGroupTypingEvent,
  parseServerEvent,
  serializeClientEvent
} from '../types/WebSocketTypes';

const DOMAIN_NUMBER: number = 0xFF06;
const TAG: string = 'WebSocketService';

// 默认重连配置（遵循 API 文档的指数退避算法）
const DEFAULT_RECONNECT_CONFIG: ReconnectConfig = {
  baseDelay: 1000,      // 1秒
  maxDelay: 30000,      // 30秒
  maxAttempts: 10,
  jitterFactor: 0.2     // ±20% 抖动
};

// 心跳间隔（秒） - 遵循 API 文档定义的 45秒
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

  /**
   * 连接WebSocket服务器
   */
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
      // 握手成功后立即发送首帧 Auth（鉴权通过后服务端才会发 ConnectionEstablished）
      this.webSocketUtil.sendRaw(serializeClientEvent(createAuthEvent(token)));
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

  /**
   * 断开WebSocket连接
   */
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

  /**
   * 获取连接状态
   */
  getState(): WebSocketState {
    return this.state;
  }

  /**
   * 是否已连接
   */
  isConnected(): boolean {
    return this.state === WebSocketState.CONNECTED;
  }

  // ============================================================================
  // 订阅管理
  // ============================================================================

  /**
   * 增量订阅
   */
  subscribe(item: SubscriptionItem): void {
    if (!this.isConnected()) {
      hilog.warn(DOMAIN_NUMBER, TAG, 'WebSocket未连接，无法订阅');
      return;
    }
    this.send(createSubscribeEvent(item) as ClientEvent);
  }

  /**
   * 取消订阅
   */
  unsubscribe(item: SubscriptionItem): void {
    if (!this.isConnected()) {
      hilog.warn(DOMAIN_NUMBER, TAG, 'WebSocket未连接，无法取消订阅');
      return;
    }
    this.send(createUnsubscribeEvent(item) as ClientEvent);
  }

  /**
   * 替换式订阅（覆盖现有订阅）
   */
  setSubscriptions(item: SubscriptionItem): void {
    if (!this.isConnected()) {
      hilog.warn(DOMAIN_NUMBER, TAG, 'WebSocket未连接，无法设置订阅');
      return;
    }
    this.send(createSetSubscriptionsEvent(item));
  }

  getSubscriptions(): void {
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

  /**
   * 订阅群聊消息
   */
  subscribeGroupChats(groupChatIds: string[]): void {
    this.subscribe({
      type: 'group',
      ids: groupChatIds
    });
  }

  /**
   * 取消订阅群聊消息
   */
  unsubscribeGroupChats(groupChatIds: string[]): void {
    this.unsubscribe({
      type: 'group',
      ids: groupChatIds
    });
  }

  /**
   * 订阅用户状态
   */
  subscribeUsers(userIds: number[]): void {
    this.subscribe({
      type: 'user',
      ids: userIds.map(id => id.toString())
    });
  }

  /**
   * 取消订阅用户状态
   */
  unsubscribeUsers(userIds: number[]): void {
    this.unsubscribe({
      type: 'user',
      ids: userIds.map(id => id.toString())
    });
  }

  // ============================================================================
  // 实时状态同步
  // ============================================================================

  /**
   * 发送正在输入状态
   */
  sendPrivateTyping(targetUserId: number, isTyping: boolean): void {
    if (!this.isConnected()) {
      return;
    }
    this.send(createPrivateTypingEvent(targetUserId, isTyping));
  }

  sendGroupTyping(groupChatId: string, isTyping: boolean): void {
    if (!this.isConnected()) {
      return;
    }
    this.send(createGroupTypingEvent(groupChatId, isTyping));
  }

  // ============================================================================
  // 事件监听
  // ============================================================================

  /**
   * 监听服务端事件
   */
  on(eventType: string, handler: (event: ServerEvent) => void): void {
    const handlers = this.eventHandlers.get(eventType) || [];
    handlers.push(handler);
    this.eventHandlers.set(eventType, handlers);
  }

  /**
   * 移除事件监听
   */
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

  /**
   * 发送客户端事件
   */
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

  /**
   * 设置内部消息处理器
   */
  private setupInternalHandlers(): void {
    // 监听原始消息
    this.webSocketUtil.onRawMessage((data: string) => {
      const event = parseServerEvent(data);
      if (event) {
        this.handleServerEvent(event as ServerEvent | BatchEventsEvent);
      }
    });

    // 监听连接关闭
    this.webSocketUtil.onClose(() => {
      hilog.info(DOMAIN_NUMBER, TAG, 'WebSocket连接关闭');
      this.state = WebSocketState.DISCONNECTED;
      this.stopHeartbeat();
      this.scheduleReconnect();
    });

    // 监听连接错误
    this.webSocketUtil.onError((error: Error) => {
      hilog.error(DOMAIN_NUMBER, TAG, `WebSocket错误: ${error.message}`);
      this.state = WebSocketState.ERROR;
    });
  }

  /**
   * 处理服务端事件
   */
  private handleServerEvent(event: ServerEvent | BatchEventsEvent): void {
    // 处理批量事件
    if (event.type === 'BatchEvents') {
      const batchEvent = event as BatchEventsEvent;
      hilog.debug(DOMAIN_NUMBER, TAG, `收到批量事件，批次ID: ${batchEvent.payload.batch_id}，事件数: ${batchEvent.payload.events.length}`);
      
      // 按序处理每个事件
      batchEvent.payload.events.forEach((subEvent) => {
        // 每个事件仍需检查 event_id 去重（在 parseServerEvent 中已处理）
        // 这里直接处理事件内容
        const eventId = subEvent.event_id;
        const eventRecord = (subEvent as Object) as Record<string, Object>;
        const eventWithoutId: Record<string, Object> = {};
        const keys = Object.keys(eventRecord);
        for (let i = 0; i < keys.length; i++) {
          const key = keys[i];
          if (key !== 'event_id') {
            eventWithoutId[key] = eventRecord[key];
          }
        }
        this.handleServerEvent((eventWithoutId as Object) as ServerEvent);
      });
      return;
    }

    hilog.debug(DOMAIN_NUMBER, TAG, `收到服务端事件: ${event.type}`);

    // 处理特殊事件
    switch (event.type) {
      case 'ConnectionEstablished':
        const connPayload = event.payload as ConnectionEstablishedPayload;
        hilog.info(DOMAIN_NUMBER, TAG, `连接已建立: connection_id=${connPayload.connection_id}, user_id=${connPayload.user_id}`);
        hilog.debug(DOMAIN_NUMBER, TAG, `服务端时间: ${connPayload.server_time}`);
        hilog.debug(DOMAIN_NUMBER, TAG, `配置: ping_interval=${connPayload.config.ping_interval}s, ping_timeout=${connPayload.config.ping_timeout}s`);
        hilog.debug(DOMAIN_NUMBER, TAG, `未读数: messages=${connPayload.unread_counts.messages}, group_messages=${connPayload.unread_counts.group_messages}, notifications=${connPayload.unread_counts.notifications}`);
        
        this.state = WebSocketState.CONNECTED;
        this.reconnectAttempts = 0;
        
        // 使用服务端配置的心跳间隔（如果可用）
        const pingInterval = connPayload.config?.ping_interval || HEARTBEAT_INTERVAL;
        this.startHeartbeat(pingInterval);
        
        // 重新订阅之前的频道
        if (this.subscriptions.length > 0) {
          this.subscriptions.forEach((item) => this.setSubscriptions(item));
        }
        break;

      case 'Pong':
        // 服务端心跳响应 (Pong)，连接正常
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
        hilog.error(DOMAIN_NUMBER, TAG, `服务端错误: [${event.payload.code}] ${event.payload.message}`);
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

  /**
   * 启动心跳
   * @param interval 心跳间隔（秒），默认使用常量值
   */
  private startHeartbeat(interval?: number): void {
    this.stopHeartbeat();
    const pingInterval = interval || HEARTBEAT_INTERVAL;
    this.heartbeatTimer = setInterval(() => {
      if (this.isConnected()) {
        this.send(createPingEvent());
      }
    }, pingInterval * 1000);
  }

  /**
   * 停止心跳
   */
  private stopHeartbeat(): void {
    if (this.heartbeatTimer !== null) {
      clearInterval(this.heartbeatTimer);
      this.heartbeatTimer = null;
    }
  }

  /**
   * 计划重连（指数退避算法）
   */
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

    // 计算延迟：min(baseDelay * 2^attempt, maxDelay) * (1 ± jitterFactor)
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
