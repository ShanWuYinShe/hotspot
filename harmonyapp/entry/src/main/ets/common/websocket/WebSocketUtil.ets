/**
 * 极简WebSocket工具类
 * 直接使用HarmonyOS WebSocket API
 */

import webSocket from '@ohos.net.webSocket';
import { hilog } from '@kit.PerformanceAnalysisKit';

const DOMAIN_NUMBER: number = 0xFF07;
const TAG: string = 'WebSocketUtil';

export interface WebSocketMessage {
  type: string;
  data: Record<string, Object>;
}

export class WebSocketUtil {
  private static instance: WebSocketUtil | null = null;
  private ws: webSocket.WebSocket | null = null;
  private url: string = '';
  private token: string = '';
  private messageHandlers: Map<string, (data: Record<string, Object>) => void> = new Map();

  private constructor() {}

  static getInstance(): WebSocketUtil {
    if (!WebSocketUtil.instance) {
      WebSocketUtil.instance = new WebSocketUtil();
    }
    return WebSocketUtil.instance;
  }

  /**
   * 连接WebSocket
   */
  async connect(url: string, token: string): Promise<void> {
    this.url = url;
    this.token = token;
    
    this.ws = webSocket.createWebSocket();
    
    return new Promise((resolve, reject) => {
      if (!this.ws) {
        reject(new Error('WebSocket创建失败'));
        return;
      }

      this.ws.on('open', () => {
        hilog.info(DOMAIN_NUMBER, TAG, 'WebSocket连接成功');
        // 发送认证消息
        this.send('auth', { "token": this.token });
        resolve();
      });

      this.ws.on('message', (err, value) => {
        if (err) {
          hilog.error(DOMAIN_NUMBER, TAG, `WebSocket消息错误: ${JSON.stringify(err)}`);
          return;
        }
        
        try {
          const message = JSON.parse(value as string) as WebSocketMessage;
          const handler = this.messageHandlers.get(message.type);
          if (handler) {
            handler(message.data);
          }
        } catch (error) {
          hilog.error(DOMAIN_NUMBER, TAG, `WebSocket消息解析失败: ${JSON.stringify(error)}`);
        }
      });

      this.ws.on('close', (err, value) => {
        hilog.info(DOMAIN_NUMBER, TAG, `WebSocket连接关闭: ${value.code}`);
      });

      this.ws.on('error', (err) => {
        hilog.error(DOMAIN_NUMBER, TAG, `WebSocket错误: ${JSON.stringify(err)}`);
        reject(err);
      });

      this.ws.connect(url);
    });
  }

  /**
   * 发送消息
   */
  send(type: string, data: Record<string, Object>): void {
    if (!this.ws) {
      hilog.error(DOMAIN_NUMBER, TAG, 'WebSocket未连接');
      return;
    }

    const message: WebSocketMessage = { type, data };
    this.ws.send(JSON.stringify(message));
  }

  /**
   * 注册消息处理器
   */
  onMessage(type: string, handler: (data: Record<string, Object>) => void): void {
    this.messageHandlers.set(type, handler);
  }

  /**
   * 断开连接
   */
  disconnect(): void {
    if (this.ws) {
      this.ws.close();
      this.ws = null;
    }
    this.messageHandlers.clear();
  }
}