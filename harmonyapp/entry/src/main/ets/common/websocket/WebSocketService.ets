/**
 * WebSocket服务管理器
 * 统一管理WebSocket连接和消息处理
 */

import { WebSocketUtil } from './WebSocketUtil';
import { WS_BASE_URL } from '../../constants/AppConstants';
import { hilog } from '@kit.PerformanceAnalysisKit';

const DOMAIN_NUMBER: number = 0xFF06;
const TAG: string = 'WebSocketService';

export class WebSocketService {
  private static instance: WebSocketService | null = null;
  private webSocketUtil: WebSocketUtil;
  private isConnected: boolean = false;
  private reconnectAttempts: number = 0;
  private maxReconnectAttempts: number = 5;
  private reconnectInterval: number = 3000; // 3秒

  private constructor() {
    this.webSocketUtil = WebSocketUtil.getInstance();
    this.setupMessageHandlers();
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
    try {
      hilog.info(DOMAIN_NUMBER, TAG, `正在连接WebSocket服务器: ${WS_BASE_URL}`);

      await this.webSocketUtil.connect(WS_BASE_URL, token);
      this.isConnected = true;
      this.reconnectAttempts = 0;

      hilog.info(DOMAIN_NUMBER, TAG, 'WebSocket连接成功');
    } catch (error) {
      hilog.error(DOMAIN_NUMBER, TAG, `WebSocket连接失败: ${JSON.stringify(error)}`);
      this.isConnected = false;

      // 自动重连
      this.scheduleReconnect(token);
      throw new Error('WebSocket连接失败');
    }
  }

  /**
   * 断开WebSocket连接
   */
  disconnect(): void {
    try {
      this.webSocketUtil.disconnect();
      this.isConnected = false;
      this.reconnectAttempts = 0;

      hilog.info(DOMAIN_NUMBER, TAG, 'WebSocket连接已断开');
    } catch (error) {
      hilog.error(DOMAIN_NUMBER, TAG, `断开WebSocket连接失败: ${JSON.stringify(error)}`);
    }
  }

  /**
   * 发送消息
   */
  sendMessage(type: string, data: Record<string, Object>): void {
    if (!this.isConnected) {
      hilog.warn(DOMAIN_NUMBER, TAG, 'WebSocket未连接，无法发送消息');
      return;
    }

    try {
      this.webSocketUtil.send(type, data);
      hilog.debug(DOMAIN_NUMBER, TAG, `发送WebSocket消息: ${type}`);
    } catch (error) {
      hilog.error(DOMAIN_NUMBER, TAG, `发送WebSocket消息失败: ${JSON.stringify(error)}`);
    }
  }

  /**
   * 获取连接状态
   */
  getConnectionStatus(): boolean {
    return this.isConnected;
  }

  /**
   * 设置消息处理器
   */
  private setupMessageHandlers(): void {
    // 认证确认
    this.webSocketUtil.onMessage('auth_success', (data: Record<string, Object>) => {
      hilog.info(DOMAIN_NUMBER, TAG, 'WebSocket认证成功');
    });

    // 认证失败
    this.webSocketUtil.onMessage('auth_failed', (data: Record<string, Object>) => {
      hilog.error(DOMAIN_NUMBER, TAG, 'WebSocket认证失败');
      this.isConnected = false;
    });

    // 根据后端文档，所有WebSocket消息都通过统一的'message'事件传递
    // 具体的消息类型通过消息体中的type字段区分
    // 这里只设置基础的连接状态处理，具体的消息处理由各个页面组件自行监听
  }

  /**
   * 计划重连
   */
  private scheduleReconnect(token: string): void {
    if (this.reconnectAttempts >= this.maxReconnectAttempts) {
      hilog.error(DOMAIN_NUMBER, TAG, '达到最大重连次数，停止重连');
      return;
    }

    this.reconnectAttempts++;
    hilog.info(DOMAIN_NUMBER, TAG, `${this.reconnectInterval}ms后进行第${this.reconnectAttempts}次重连`);

    setTimeout(() => {
      this.connect(token).catch((error: Error) => {
        hilog.error(DOMAIN_NUMBER, TAG, `重连失败: ${JSON.stringify(error)}`);
      });
    }, this.reconnectInterval);
  }

  /**
   * 注册自定义消息处理器
   */
  onMessage(type: string, handler: (data: Record<string, Object>) => void): void {
    this.webSocketUtil.onMessage(type, handler);
  }
}








