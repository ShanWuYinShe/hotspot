/**
 * WebSocket服务管理器
 * 统一管理WebSocket连接和消息处理
 * 严格遵循 api_spec.mdc WebSocket 设计规范
 */

import { WebSocketUtil } from './WebSocketUtil';
import { WS_BASE_URL } from '../../constants/AppConstants';
import { hilog } from '@kit.PerformanceAnalysisKit';
import {
  ClientEvent,
  ServerEvent,
  SubscriptionItem,
  WebSocketState,
  ReconnectConfig,
  createPingEvent,
  createSubscribeEvent,
  createUnsubscribeEvent,
  createSetSubscriptionsEvent,
  createMarkAsReadEvent,
  createTypingEvent,
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

// 心跳间隔（秒）
const HEARTBEAT_INTERVAL = 25;

export class WebSocketService {
  private static instance: WebSocketService | null = null;
  private webSocketUtil: WebSocketUtil;
  private state: WebSocketState = 'disconnected';
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
    if (this.state === 'connected' || this.state === 'connecting') {
      hilog.warn(DOMAIN_NUMBER, TAG, 'WebSocket已连接或正在连接中');
      return;
    }

    this.currentToken = token;
    this.state = 'connecting';

    try {
      hilog.info(DOMAIN_NUMBER, TAG, `正在连接WebSocket服务器: ${WS_BASE_URL}`);
      await this.webSocketUtil.connect(WS_BASE_URL, token);
      this.state = 'connected';
      this.reconnectAttempts = 0;
      this.startHeartbeat();
      hilog.info(DOMAIN_NUMBER, TAG, 'WebSocket连接成功');
    } catch (error) {
      hilog.error(DOMAIN_NUMBER, TAG, `WebSocket连接失败: ${JSON.stringify(error)}`);
      this.state = 'error';
      this.scheduleReconnect();
      throw new Error('WebSocket连接失败');
    }
  }

  /**
   * 断开WebSocket连接
   */
  disconnect(): void {
    try {
      this.stopHeartbeat();
      this.webSocketUtil.disconnect();
      this.state = 'disconnected';
      this.reconnectAttempts = 0;
      this.subscriptions = [];
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
    return this.state === 'connected';
  }

  // ============================================================================
  // 订阅管理
  // ============================================================================

  /**
   * 增量订阅
   */
  subscribe(items: SubscriptionItem[]): void {
    if (!this.isConnected()) {
      hilog.warn(DOMAIN_NUMBER, TAG, 'WebSocket未连接，无法订阅');
      return;
    }
    this.send(createSubscribeEvent(items));
  }

  /**
   * 取消订阅
   */
  unsubscribe(items: SubscriptionItem[]): void {
    if (!this.isConnected()) {
      hilog.warn(DOMAIN_NUMBER, TAG, 'WebSocket未连接，无法取消订阅');
      return;
    }
    this.send(createUnsubscribeEvent(items));
  }

  /**
   * 替换式订阅（覆盖现有订阅）
   */
  setSubscriptions(items: SubscriptionItem[]): void {
    if (!this.isConnected()) {
      hilog.warn(DOMAIN_NUMBER, TAG, 'WebSocket未连接，无法设置订阅');
      return;
    }
    this.send(createSetSubscriptionsEvent(items));
  }

  /**
   * 订阅群聊消息
   */
  subscribeGroupChats(groupChatIds: string[]): void {
    this.subscribe([{
      source_type: 'group_chat',
      source_ids: groupChatIds
    }]);
  }

  /**
   * 取消订阅群聊消息
   */
  unsubscribeGroupChats(groupChatIds: string[]): void {
    this.unsubscribe([{
      source_type: 'group_chat',
      source_ids: groupChatIds
    }]);
  }

  /**
   * 订阅用户状态
   */
  subscribeUsers(userIds: string[]): void {
    this.subscribe([{
      source_type: 'user',
      source_ids: userIds
    }]);
  }

  /**
   * 取消订阅用户状态
   */
  unsubscribeUsers(userIds: string[]): void {
    this.unsubscribe([{
      source_type: 'user',
      source_ids: userIds
    }]);
  }

  // ============================================================================
  // 实时状态同步
  // ============================================================================

  /**
   * 标记消息已读
   */
  markAsRead(chatType: 'private_chat' | 'group_chat', chatId: string, upTo: string): void {
    if (!this.isConnected()) {
      hilog.warn(DOMAIN_NUMBER, TAG, 'WebSocket未连接，无法标记已读');
      return;
    }
    this.send(createMarkAsReadEvent(chatType, chatId, upTo));
  }

  /**
   * 发送正在输入状态
   */
  sendTyping(chatType: 'private_chat' | 'group_chat', chatId: string, isTyping: boolean): void {
    if (!this.isConnected()) {
      return;
    }
    this.send(createTypingEvent(chatType, chatId, isTyping));
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
        this.handleServerEvent(event);
      }
    });

    // 监听连接关闭
    this.webSocketUtil.onClose(() => {
      hilog.info(DOMAIN_NUMBER, TAG, 'WebSocket连接关闭');
      this.state = 'disconnected';
      this.stopHeartbeat();
      this.scheduleReconnect();
    });

    // 监听连接错误
    this.webSocketUtil.onError((error: Error) => {
      hilog.error(DOMAIN_NUMBER, TAG, `WebSocket错误: ${error.message}`);
      this.state = 'error';
    });
  }

  /**
   * 处理服务端事件
   */
  private handleServerEvent(event: ServerEvent): void {
    hilog.debug(DOMAIN_NUMBER, TAG, `收到服务端事件: ${event.type}`);

    // 处理特殊事件
    switch (event.type) {
      case 'ConnectionEstablished':
        hilog.info(DOMAIN_NUMBER, TAG, `连接已建立: ${JSON.stringify(event.payload)}`);
        // 重新订阅之前的频道
        if (this.subscriptions.length > 0) {
          this.setSubscriptions(this.subscriptions);
        }
        break;

      case 'Heartbeat':
        // 服务端心跳响应，更新连接状态
        break;

      case 'SubscribeAck':
        if (event.payload.code === 0) {
          this.subscriptions = event.payload.items;
          hilog.info(DOMAIN_NUMBER, TAG, `订阅成功: ${JSON.stringify(this.subscriptions)}`);
        } else {
          hilog.error(DOMAIN_NUMBER, TAG, `订阅失败: ${event.payload.message}`);
        }
        break;

      case 'Error':
        hilog.error(DOMAIN_NUMBER, TAG, `服务端错误: [${event.payload.code}] ${event.payload.message}`);
        break;
    }

    // 触发事件处理器
    const handlers = this.eventHandlers.get(event.type) || [];
    handlers.forEach(handler => {
      try {
        handler(event);
      } catch (error) {
        hilog.error(DOMAIN_NUMBER, TAG, `事件处理器错误: ${JSON.stringify(error)}`);
      }
    });

    // 触发通用处理器
    const allHandlers = this.eventHandlers.get('*') || [];
    allHandlers.forEach(handler => {
      try {
        handler(event);
      } catch (error) {
        hilog.error(DOMAIN_NUMBER, TAG, `通用事件处理器错误: ${JSON.stringify(error)}`);
      }
    });
  }

  /**
   * 启动心跳
   */
  private startHeartbeat(): void {
    this.stopHeartbeat();
    this.heartbeatTimer = setInterval(() => {
      if (this.isConnected()) {
        this.send(createPingEvent());
      }
    }, HEARTBEAT_INTERVAL * 1000);
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
