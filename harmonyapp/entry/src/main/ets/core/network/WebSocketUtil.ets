/**
 * WebSocket工具类
 * 底层WebSocket连接管理，遵循 api_spec.md WebSocket 设计规范
 */

import webSocket from '@ohos.net.webSocket';
import { hilog } from '@kit.PerformanceAnalysisKit';

const DOMAIN_NUMBER: number = 0xFF07;
const TAG: string = 'WebSocketUtil';

export class WebSocketUtil {
  private static instance: WebSocketUtil | null = null;
  private ws: webSocket.WebSocket | null = null;
  private url: string = '';
  private rawMessageHandler: ((data: string) => void) | null = null;
  private closeHandler: ((code: number) => void) | null = null;
  private errorHandler: ((error: Error) => void) | null = null;

  private constructor() {}

  static getInstance(): WebSocketUtil {
    if (!WebSocketUtil.instance) {
      WebSocketUtil.instance = new WebSocketUtil();
    }
    return WebSocketUtil.instance;
  }

  /** 由 ws(s) 地址推导同源 Origin（http(s)://host[:port]），去掉 /ws 路径 */
  private static buildOriginHeader(wsUrl: string): string {
    let url = wsUrl;
    const wsPathIndex = url.indexOf('/ws');
    if (wsPathIndex >= 0) {
      url = url.substring(0, wsPathIndex);
    }
    return url.replace(/^wss:\/\//, 'https://').replace(/^ws:\/\//, 'http://');
  }

  /**
   * 连接WebSocket
   */
  async connect(baseUrl: string): Promise<void> {
    this.url = baseUrl;
    
    this.ws = webSocket.createWebSocket();
    
    return new Promise((resolve, reject) => {
      if (!this.ws) {
        reject(new Error('WebSocket创建失败'));
        return;
      }

      this.ws.on('open', (err, value) => {
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
        hilog.info(DOMAIN_NUMBER, TAG, `WebSocket连接关闭: ${value.code}, 原因: ${value.reason}`);
        if (this.closeHandler) {
          this.closeHandler(value.code);
        }
      });

      this.ws.on('error', (err) => {
        hilog.error(DOMAIN_NUMBER, TAG, `WebSocket错误: ${JSON.stringify(err)}`);
        if (this.errorHandler) {
          this.errorHandler(new Error(JSON.stringify(err)));
        }
        reject(err);
      });

      // 显式携带 Origin：鸿蒙原生 WebSocket 不保证发送浏览器式 Origin，
      // 实测会以 `Origin: null` 发出；后端 CSWSH 防护把 Origin≠Host 视为跨站并返回 403。
      // 这里按连接地址推导同源 Origin（ws:// → http://，wss:// → https://，去掉 /ws 路径），
      // 与 nginx 转发的 Host（$http_host）一致，既通过同源校验也不放宽服务端防护。
      const options: webSocket.WebSocketRequestOptions = {
        header: { 'Origin': WebSocketUtil.buildOriginHeader(this.url) }
      };

      this.ws.connect(this.url, options);
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
    try {
      this.ws.send(data);
    } catch (error) {
      hilog.error(DOMAIN_NUMBER, TAG, `WebSocket发送消息失败: ${JSON.stringify(error)}`);
    }
  }

  /**
   * 注册原始消息处理器
   */
  onRawMessage(handler: (data: string) => void): void {
    this.rawMessageHandler = handler;
  }

  /**
   * 注册连接关闭处理器（透传 close code，供上层判断是否自动重连）
   */
  onClose(handler: (code: number) => void): void {
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
   * @param clearHandlers 是否同时清空事件处理器（默认保留，保证重连后消息路由不丢失）
   */
  disconnect(clearHandlers: boolean = false): void {
    if (this.ws) {
      try {
        this.ws.close();
      } catch (error) {
        hilog.error(DOMAIN_NUMBER, TAG, `WebSocket关闭失败: ${JSON.stringify(error)}`);
      }
      this.ws = null;
    }
    if (clearHandlers) {
      this.rawMessageHandler = null;
      this.closeHandler = null;
      this.errorHandler = null;
    }
  }

  /**
   * 获取连接状态
   */
  isConnected(): boolean {
    return this.ws !== null;
  }
}
