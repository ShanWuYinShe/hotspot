/**
 * WebSocket工具类
 * 底层WebSocket连接管理，遵循 api_spec.mdc WebSocket 设计规范
 */

import webSocket from '@ohos.net.webSocket';
import { hilog } from '@kit.PerformanceAnalysisKit';

const DOMAIN_NUMBER: number = 0xFF07;
const TAG: string = 'WebSocketUtil';

export class WebSocketUtil {
  private static instance: WebSocketUtil | null = null;
  private ws: webSocket.WebSocket | null = null;
  private url: string = '';
  private token: string = '';
  private rawMessageHandler: ((data: string) => void) | null = null;
  private closeHandler: (() => void) | null = null;
  private errorHandler: ((error: Error) => void) | null = null;

  private constructor() {}

  static getInstance(): WebSocketUtil {
    if (!WebSocketUtil.instance) {
      WebSocketUtil.instance = new WebSocketUtil();
    }
    return WebSocketUtil.instance;
  }

  /**
   * 连接WebSocket
   * WebSocket 鉴权通过 URL Query 参数传递 token
   */
  async connect(baseUrl: string, token: string): Promise<void> {
    this.token = token;
    // 通过 URL Query 传递 token 进行鉴权
    this.url = `${baseUrl}?token=${encodeURIComponent(token)}`;
    
    this.ws = webSocket.createWebSocket();
    
    return new Promise((resolve, reject) => {
      if (!this.ws) {
        reject(new Error('WebSocket创建失败'));
        return;
      }

      this.ws.on('open', () => {
        hilog.info(DOMAIN_NUMBER, TAG, 'WebSocket连接成功');
        resolve();
      });

      this.ws.on('message', (err, value) => {
        if (err) {
          hilog.error(DOMAIN_NUMBER, TAG, `WebSocket消息错误: ${JSON.stringify(err)}`);
          return;
        }
        
        // 直接将原始消息传递给处理器
        if (this.rawMessageHandler && typeof value === 'string') {
          this.rawMessageHandler(value);
        }
      });

      this.ws.on('close', (err, value) => {
        hilog.info(DOMAIN_NUMBER, TAG, `WebSocket连接关闭: ${value.code}`);
        if (this.closeHandler) {
          this.closeHandler();
        }
      });

      this.ws.on('error', (err) => {
        hilog.error(DOMAIN_NUMBER, TAG, `WebSocket错误: ${JSON.stringify(err)}`);
        if (this.errorHandler) {
          this.errorHandler(new Error(JSON.stringify(err)));
        }
        reject(err);
      });

      this.ws.connect(this.url);
    });
  }

  /**
   * 发送原始消息（已序列化的字符串）
   */
  sendRaw(data: string): void {
    if (!this.ws) {
      hilog.error(DOMAIN_NUMBER, TAG, 'WebSocket未连接');
      return;
    }
    this.ws.send(data);
  }

  /**
   * 注册原始消息处理器
   */
  onRawMessage(handler: (data: string) => void): void {
    this.rawMessageHandler = handler;
  }

  /**
   * 注册连接关闭处理器
   */
  onClose(handler: () => void): void {
    this.closeHandler = handler;
  }

  /**
   * 注册错误处理器
   */
  onError(handler: (error: Error) => void): void {
    this.errorHandler = handler;
  }

  /**
   * 断开连接
   */
  disconnect(): void {
    if (this.ws) {
      this.ws.close();
      this.ws = null;
    }
    this.rawMessageHandler = null;
    this.closeHandler = null;
    this.errorHandler = null;
  }

  /**
   * 获取连接状态
   */
  isConnected(): boolean {
    return this.ws !== null;
  }
}
