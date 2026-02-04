import type {
  ClientEvent,
  ServerEvent,
  ServerEventType,
  ServerPayload,
  WebSocketState,
  SubscriptionItem,
  SubscribeAckEvent,
  ClientEventType
} from '~/types/websocket'
import { WS_CONSTANTS } from '~/config/defaults'
import { isTokenInvalidOrExpiredCode } from '~/utils/apiError'

// 类型定义
type MessageHandler<K extends ServerEventType> = (payload: ServerPayload<K>) => void
type StatusHandler = (status: WebSocketState) => void
type TokenProvider = () => string | null | Promise<string | null>
type ErrorHandler = (error: { code: number; message: string }) => void

/**
 * WebSocket 服务实现 (单例模式)
 * 
 * 职责:
 * - 连接生命周期管理 (连接/断开/重连)
 * - 认证 (握手后首帧 Auth)
 * - 心跳保活 (Ping/Pong)
 * - 消息分发 (Strict Typed)
 * - 订阅状态维护 (自动重发订阅)
 */
class WebSocketService {
  private static instance: WebSocketService
  
  // 配置
  private url: string = ''
  private tokenProvider: TokenProvider | null = null
  private errorHandler: ErrorHandler | null = null
  
  // 状态
  private ws: WebSocket | null = null
  private status: WebSocketState = 'disconnected'
  private reconnectCount = 0
  private missedHeartbeats = 0
  private isManuallyDisconnected = false
  private isConnecting = false
  private pendingAuthToken: string | null = null
  
  // 定时器
  private heartbeatTimer: ReturnType<typeof setInterval> | null = null
  private heartbeatTimeoutTimer: ReturnType<typeof setTimeout> | null = null
  private reconnectTimer: ReturnType<typeof setTimeout> | null = null
  
  // 处理器
  private messageHandlers = new Map<ServerEventType, Set<(payload: unknown) => void>>()
  private statusHandlers = new Set<StatusHandler>()
  
  // 订阅状态 (key: "type:id") -> Item
  private activeSubscriptions = new Map<string, SubscriptionItem>() 
  
  private constructor() {}

  public static getInstance(): WebSocketService {
    if (!WebSocketService.instance) {
      WebSocketService.instance = new WebSocketService()
    }
    return WebSocketService.instance
  }

  // ==========================================================================
  // 公共 API
  // ==========================================================================

  /**
   * 获取当前连接状态
   */
  public getStatus(): WebSocketState {
    return this.status
  }

  /**
   * 设置 Token 获取器 (用于重连时获取最新 Token)
   */
  public setTokenProvider(provider: TokenProvider) {
    this.tokenProvider = provider
  }

  /**
   * 设置全局错误处理器
   */
  public setErrorHandler(handler: ErrorHandler) {
    this.errorHandler = handler
  }

  /**
   * 连接 WebSocket
   */
  public async connect(url: string) {
    this.url = url
    this.isManuallyDisconnected = false
    
    // 如果已经连接或正在连接，且 URL 未变，则忽略
    if ((this.status === 'connected' || this.isConnecting) && this.ws?.url === url) {
      return
    }

    // 如果 URL 变了，或者需要强制重连，先断开
    if (this.ws) {
      this.disconnect()
    }

    this.reconnectCount = 0
    await this._connect()
  }

  /**
   * 主动断开连接
   * (通常用于用户登出)
   */
  public disconnect() {
    this.isManuallyDisconnected = true
    this._cleanup()
    
    // 主动断开时清空订阅，防止切换用户后订阅串台
    this.activeSubscriptions.clear()
    
    if (this.ws) {
      this.ws.close(1000, 'Client Disconnect')
      this.ws = null
    }
    this._setStatus('disconnected')
  }

  /**
   * 发送客户端事件
   */
  public send<T extends ClientEvent>(type: T['type'], payload: T['payload']): boolean {
    if (this.status !== 'connected' || !this.ws) {
      console.warn('[WS] Cannot send message: not connected', type)
      return false
    }
    try {
      this.ws.send(JSON.stringify({ type, payload }))
      return true
    } catch (e) {
      console.error('[WS] Send failed', e)
      return false
    }
  }

  /**
   * 订阅数据源
   */
  public subscribe(items: SubscriptionItem[]) {
    if (items.length === 0) return

    // 1. 更新本地订阅状态
    items.forEach(item => {
      item.source_ids.forEach(id => {
        const key = `${item.source_type}:${id}`
        this.activeSubscriptions.set(key, { 
          source_type: item.source_type, 
          source_ids: [id] as typeof item.source_ids
        })
      })
    })

    // 2. 如果已连接，立即发送订阅请求
    if (this.status === 'connected') {
      const payloadItems = items.map(item => ({
        ...item,
        source_ids: item.source_ids.map(String)
      }))
      this.send('Subscribe', { items: payloadItems })
    }
  }

  /**
   * 取消订阅
   */
  public unsubscribe(items: SubscriptionItem[]) {
    if (items.length === 0) return

    // 1. 更新本地订阅状态
    items.forEach(item => {
      item.source_ids.forEach(id => {
        this.activeSubscriptions.delete(`${item.source_type}:${id}`)
      })
    })

    // 2. 如果已连接，发送取消订阅请求
    if (this.status === 'connected') {
      const payloadItems = items.map(item => ({
        ...item,
        source_ids: item.source_ids.map(String)
      }))
      this.send('Unsubscribe', { items: payloadItems })
    }
  }

  /**
   * 替换所有订阅 (全量更新)
   */
  public setSubscriptions(items: SubscriptionItem[]) {
    // 1. 清空并重建本地状态
    this.activeSubscriptions.clear()
    items.forEach(item => {
      item.source_ids.forEach(id => {
        const key = `${item.source_type}:${id}`
        this.activeSubscriptions.set(key, { 
          source_type: item.source_type, 
          source_ids: [id] as typeof item.source_ids
        })
      })
    })

    // 2. 如果已连接，发送设置请求
    if (this.status === 'connected') {
      const payloadItems = items.map(item => ({
        ...item,
        source_ids: item.source_ids.map(String)
      }))
      this.send('SetSubscriptions', { items: payloadItems })
    }
  }

  /**
   * 注册消息监听器
   * 返回取消监听函数
   */
  public on<K extends ServerEventType>(type: K, handler: MessageHandler<K>): () => void {
    if (!this.messageHandlers.has(type)) {
      this.messageHandlers.set(type, new Set())
    }
    this.messageHandlers.get(type)!.add(handler as unknown as (payload: unknown) => void)
    
    return () => {
      const handlers = this.messageHandlers.get(type)
      if (handlers) {
        handlers.delete(handler as unknown as (payload: unknown) => void)
        if (handlers.size === 0) {
          this.messageHandlers.delete(type)
        }
      }
    }
  }

  /**
   * 注册状态变更监听器
   */
  public onStatusChange(handler: StatusHandler): () => void {
    this.statusHandlers.add(handler)
    // 立即回调当前状态
    handler(this.status)
    return () => {
      this.statusHandlers.delete(handler)
    }
  }

  // ==========================================================================
  // 私有实现
  // ==========================================================================

  private async _connect() {
    if (!this.url || this.isManuallyDisconnected) return
    if (this.isConnecting) return

    this.isConnecting = true
    this._setStatus('connecting')

    // 获取 Token
    let token: string | null = null
    if (this.tokenProvider) {
      try {
        token = await this.tokenProvider()
      } catch (e) {
        console.error('[WS] Failed to get token', e)
      }
    }

    if (!token) {
      console.warn('[WS] No token available, scheduling reconnect')
      this.isConnecting = false
      this._scheduleReconnect()
      return
    }
    this.pendingAuthToken = token
    
    try {
      this.ws = new WebSocket(this.url)
      this.ws.onopen = this._handleOpen.bind(this)
      this.ws.onmessage = this._handleMessage.bind(this)
      this.ws.onclose = this._handleClose.bind(this)
      this.ws.onerror = this._handleError.bind(this)
    } catch (e) {
      console.error('[WS] Connection creation failed', e)
      this.isConnecting = false
      this._handleClose({ code: 1006 } as CloseEvent)
    }
  }

  private _handleOpen() {
    if (import.meta.dev) {
      console.log('[WS] Socket opened, sending Auth...')
    }
    this.isConnecting = false
    this._sendAuth()
  }

  private _handleMessage(event: MessageEvent) {
    try {
      const message: ServerEvent = JSON.parse(event.data)

      // 鉴权成功：服务端会先发 ConnectionEstablished
      if (message.type === 'ConnectionEstablished') {
        this.pendingAuthToken = null
        this._setStatus('connected')
        this.reconnectCount = 0
        this.missedHeartbeats = 0
        this._startHeartbeat()
        this._resyncSubscriptions()
      }
      
      // 心跳响应处理：接收服务端的 Pong
      if (message.type === 'Pong') {
        this.missedHeartbeats = 0
        if (this.heartbeatTimeoutTimer) {
          clearTimeout(this.heartbeatTimeoutTimer)
          this.heartbeatTimeoutTimer = null
        }
        return
      }

      // 系统错误处理
      if (message.type === 'Error') {
         if (this.errorHandler) {
           this.errorHandler(message.payload)
         }
         // 40101 = Token 无效/过期 -> 可能需要刷新 Token 重连
         if (isTokenInvalidOrExpiredCode(message.payload.code)) {
           console.warn('[WS] Auth error from server, closing connection to trigger reconnect logic')
           // 关闭连接，触发 reconnect 逻辑 (reconnect 会重新调用 tokenProvider 获取新 token)
           this.ws?.close(4001, 'Token Expired') 
         }
      }

      // 订阅确认处理 (清理失败的订阅)
      if (message.type === 'SubscribeAck') {
        this._handleSubscribeAck(message.payload as SubscribeAckEvent['payload'])
      }

      // 分发消息
      const handlers = this.messageHandlers.get(message.type)
      if (handlers) {
        handlers.forEach(handler => {
          try {
            handler(message.payload as unknown)
          } catch (e) {
            console.error('[WS] Handler error', e)
          }
        })
      }
      
    } catch (e) {
      console.error('[WS] Message parse error', e)
    }
  }

  private _sendAuth() {
    if (!this.ws || this.ws.readyState !== WebSocket.OPEN) return
    if (!this.pendingAuthToken) return

    try {
      this.ws.send(JSON.stringify({
        type: 'Auth',
        payload: { access_token: this.pendingAuthToken }
      }))
    } catch (e) {
      console.error('[WS] Auth send failed', e)
    }
  }

  private _handleSubscribeAck(payload: SubscribeAckEvent['payload']) {
    if (payload.code !== 0 && payload.failed_items) {
      console.warn('[WS] Partial subscription failure', payload.failed_items)
      // 移除失败的订阅项，避免无限重试
      payload.failed_items.forEach(item => {
        item.source_ids.forEach(id => {
          this.activeSubscriptions.delete(`${item.source_type}:${id}`)
        })
      })
    }
  }

  private _handleClose(event: CloseEvent) {
    this.isConnecting = false
    
    if (this.isManuallyDisconnected) {
      if (import.meta.dev) {
        console.log(`[WS] Closed manually (Code: ${event.code})`)
      }
      return
    }

    if (import.meta.dev) {
      console.log(`[WS] Closed (Code: ${event.code}, Reason: ${event.reason})`)
    }
    this._cleanup()
    this._setStatus('disconnected')

    // 自动重连逻辑
    // 1000: 正常关闭 (通常由服务端发起，如果非手动断开，也应该重连)
    // 1001: 离开页面 (浏览器行为)
    // 1006: 异常关闭 (网络断开等)
    // 4001: Token 过期 (自定义)
    
    this._scheduleReconnect()
  }

  private _handleError(event: Event) {
    // console.error('[WS] Error', event)
    this._setStatus('error')
    this.isConnecting = false
    // Error 之后通常会触发 Close，重连逻辑在 Close 中处理
  }

  private _setStatus(status: WebSocketState) {
    if (this.status !== status) {
      this.status = status
      this.statusHandlers.forEach(handler => handler(status))
    }
  }

  private _cleanup() {
    if (this.heartbeatTimer) clearInterval(this.heartbeatTimer)
    if (this.heartbeatTimeoutTimer) clearTimeout(this.heartbeatTimeoutTimer)
    if (this.reconnectTimer) clearTimeout(this.reconnectTimer)
    
    this.heartbeatTimer = null
    this.heartbeatTimeoutTimer = null
    this.reconnectTimer = null

    if (this.ws) {
      this.ws.onopen = null
      this.ws.onmessage = null
      this.ws.onclose = null
      this.ws.onerror = null
    }
  }

  private _scheduleReconnect() {
    if (this.isManuallyDisconnected) return

    const { MAX_RECONNECT_ATTEMPTS, RECONNECT_INTERVAL } = WS_CONSTANTS
    const maxDelay = 30000 // 最大延迟 30s

    if (this.reconnectCount >= MAX_RECONNECT_ATTEMPTS) {
      console.error('[WS] Max reconnect attempts reached')
      this._setStatus('error')
      return
    }

    // 指数退避算法: base * 2^n
    const delay = Math.min(RECONNECT_INTERVAL * Math.pow(2, this.reconnectCount), maxDelay)
    // 增加随机抖动 (+/- 20%)
    const jitter = delay * 0.2 * (Math.random() * 2 - 1)
    const finalDelay = Math.max(0, Math.floor(delay + jitter))

    if (import.meta.dev) {
      console.log(`[WS] Reconnecting in ${finalDelay}ms... (Attempt ${this.reconnectCount + 1}/${MAX_RECONNECT_ATTEMPTS})`)
    }
    
    this.reconnectTimer = setTimeout(() => {
      this.reconnectCount++
      this._connect()
    }, finalDelay)
  }

  private _startHeartbeat() {
    const { HEARTBEAT_INTERVAL, HEARTBEAT_TIMEOUT, MAX_MISSED_HEARTBEATS } = WS_CONSTANTS
    
    // 清理旧定时器
    if (this.heartbeatTimer) clearInterval(this.heartbeatTimer)

    this.heartbeatTimer = setInterval(() => {
      if (this.status === 'connected') {
        this.send('Ping', { timestamp: new Date().toISOString() })
        
        // 等待 Pong
        this.heartbeatTimeoutTimer = setTimeout(() => {
          this.missedHeartbeats++
          console.warn(`[WS] Heartbeat timeout (${this.missedHeartbeats}/${MAX_MISSED_HEARTBEATS})`)
          
          if (this.missedHeartbeats >= MAX_MISSED_HEARTBEATS) {
            console.error('[WS] Heartbeat failed, reconnecting')
            if (this.ws) this.ws.close() // 触发 Close -> Reconnect
          }
        }, HEARTBEAT_TIMEOUT)
      }
    }, HEARTBEAT_INTERVAL)
  }

  /**
   * 重连后重新发送订阅请求
   */
  private _resyncSubscriptions() {
    if (this.activeSubscriptions.size === 0) return

    // 将 Map 转换为 SubscriptionItem 数组
    const groups: string[] = []
    const users: number[] = []
    const notifs: string[] = []

    this.activeSubscriptions.forEach(item => {
      if (item.source_type === 'group_chat') groups.push(...item.source_ids as string[])
      else if (item.source_type === 'user') users.push(...item.source_ids as number[])
      else if (item.source_type === 'notification') notifs.push(...item.source_ids as string[])
    })

    const items: SubscriptionItem[] = []
    if (groups.length) items.push({ source_type: 'group_chat', source_ids: [...new Set(groups)] })
    if (users.length) items.push({ source_type: 'user', source_ids: [...new Set(users)].map(String) })
    if (notifs.length) items.push({ source_type: 'notification', source_ids: [...new Set(notifs)] })

    if (items.length > 0) {
      if (import.meta.dev) {
        console.log('[WS] Resyncing subscriptions...', items)
      }
      this.send('SetSubscriptions', { items })
    }
  }
}

// 导出单例
export const webSocketService = WebSocketService.getInstance()
