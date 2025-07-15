// WebSocket实时通信Hook
import type { 
  WebSocketMessage, 
  WebSocketClientMessage, 
  ServerMessage,
  WebSocketConnectionState 
} from '~/types/websocket'

// WebSocket事件类型
type WebSocketEvent = 'message' | 'open' | 'close' | 'error' | 'reconnect'

type MessageHandler = (data: ServerMessage) => void
type EventCallback = (data?: any) => void

// WebSocket状态
type SocketStatus = 'connecting' | 'open' | 'closing' | 'closed' | 'error'

export function useWebSocket(url: string, options: {
  autoConnect?: boolean
  reconnectInterval?: number
  maxReconnectAttempts?: number
  heartbeatInterval?: number
  heartbeatMessage?: WebSocketClientMessage
  parseMessage?: boolean
} = {}) {
  // 配置参数
  const {
    autoConnect = true,
    reconnectInterval = 3000,
    maxReconnectAttempts = 5,
    heartbeatInterval = 30000,
    heartbeatMessage = { type: 'Ping' },
    parseMessage = true
  } = options
  
  // 网络状态检测
  const isOnline = ref<boolean>(navigator.onLine)
  
  // 监听网络状态变化
  const handleOnlineChange = () => {
    isOnline.value = navigator.onLine
  }
  
  window.addEventListener('online', handleOnlineChange)
  window.addEventListener('offline', handleOnlineChange)
  
  // WebSocket实例
  const socket = ref<WebSocket | null>(null)
  const status = ref<SocketStatus>('closed')
  const reconnectCount = ref<number>(0)
  const lastMessageTime = ref<number>(Date.now())
  
  // 事件监听器
  const listeners = ref<Map<WebSocketEvent, Set<EventCallback>>>(new Map())
  
  // 消息处理器
  const messageHandlers = ref<Map<string, Set<MessageHandler>>>(new Map())
  
  // 心跳定时器
  let heartbeatTimer: any = null
  
  // 重连定时器
  let reconnectTimer: any = null
  
  // 计算属性
  const isConnected = computed(() => status.value === 'open')
  const isConnecting = computed(() => status.value === 'connecting')
  const canReconnect = computed(() => reconnectCount.value < maxReconnectAttempts)
  
  // 创建WebSocket连接
  const connect = () => {
    if (socket.value || status.value === 'connecting') return
    
    try {
      status.value = 'connecting'
      socket.value = new WebSocket(url)
      
      // 设置事件处理器
      socket.value.onopen = handleOpen
      socket.value.onclose = handleClose
      socket.value.onerror = handleError
      socket.value.onmessage = handleMessage
    } catch (error) {
      console.error('创建WebSocket连接失败:', error)
      status.value = 'error'
      handleReconnect()
    }
  }
  
  // 关闭连接
  const disconnect = () => {
    if (!socket.value) return
    
    try {
      stopHeartbeat()
      status.value = 'closing'
      socket.value.close()
    } catch (error) {
      console.error('关闭WebSocket连接失败:', error)
      status.value = 'error'
    }
  }
  
  // 发送消息
  const send = (data: WebSocketClientMessage | string): boolean => {
    if (!isConnected.value || !socket.value) return false
    
    try {
      const message = typeof data === 'string' ? data : JSON.stringify(data)
      socket.value.send(message)
      return true
    } catch (error) {
      console.error('发送WebSocket消息失败:', error)
      return false
    }
  }
  
  // 注册事件监听器
  const on = (event: WebSocketEvent, callback: EventCallback) => {
    if (!listeners.value.has(event)) {
      listeners.value.set(event, new Set())
    }
    listeners.value.get(event)?.add(callback)
    
    // 返回取消监听函数
    return () => off(event, callback)
  }
  
  // 取消事件监听
  const off = (event: WebSocketEvent, callback: EventCallback) => {
    if (listeners.value.has(event)) {
      listeners.value.get(event)?.delete(callback)
    }
  }
  
  // 注册消息类型处理器
  const onMessage = (type: string, handler: MessageHandler) => {
    if (!messageHandlers.value.has(type)) {
      messageHandlers.value.set(type, new Set())
    }
    messageHandlers.value.get(type)?.add(handler)
    
    // 返回取消监听函数
    return () => offMessage(type, handler)
  }
  
  // 取消消息类型监听
  const offMessage = (type: string, handler: MessageHandler) => {
    if (messageHandlers.value.has(type)) {
      messageHandlers.value.get(type)?.delete(handler)
    }
  }
  
  // 触发事件
  const emit = (event: WebSocketEvent, data?: any) => {
    if (listeners.value.has(event)) {
      listeners.value.get(event)?.forEach(callback => {
        try {
          callback(data)
        } catch (error) {
          console.error(`WebSocket ${event} 事件处理器错误:`, error)
        }
      })
    }
  }
  
  // 事件处理函数
  const handleOpen = (event: Event) => {
    status.value = 'open'
    reconnectCount.value = 0
    lastMessageTime.value = Date.now()
    
    // 启动心跳
    startHeartbeat()
    
    // 触发事件
    emit('open', event)
  }
  
  const handleClose = (event: CloseEvent) => {
    status.value = 'closed'
    socket.value = null
    stopHeartbeat()
    
    // 触发事件
    emit('close', event)
    
    // 尝试重新连接
    handleReconnect()
  }
  
  const handleError = (event: Event) => {
    status.value = 'error'
    
    // 触发事件
    emit('error', event)
    
    // 尝试重新连接
    handleReconnect()
  }
  
  const handleMessage = (event: MessageEvent) => {
    lastMessageTime.value = Date.now()
    
    try {
      // 解析消息
      let data: ServerMessage | string = event.data
      
      if (parseMessage && typeof data === 'string') {
        try {
          data = JSON.parse(data) as ServerMessage
        } catch (e) {
          // 不是JSON格式，保持原始字符串
        }
      }
      
      // 触发消息事件
      emit('message', data)
      
      // 处理特定类型的消息
      if (data && typeof data === 'object' && 'type' in data) {
        const msgType = data.type
        if (messageHandlers.value.has(msgType)) {
          messageHandlers.value.get(msgType)?.forEach(handler => {
            try {
              handler(data as ServerMessage)
            } catch (error) {
              console.error(`处理 ${msgType} 类型消息错误:`, error)
            }
          })
        }
      }
    } catch (error) {
      console.error('处理WebSocket消息失败:', error)
    }
  }
  
  // 重连处理
  const handleReconnect = () => {
    if (reconnectTimer) {
      clearTimeout(reconnectTimer)
      reconnectTimer = null
    }
    
    if (!isOnline.value || reconnectCount.value >= maxReconnectAttempts) {
      return
    }
    
    reconnectCount.value++
    
    // 触发重连事件
    emit('reconnect', { attempt: reconnectCount.value, max: maxReconnectAttempts })
    
    // 设置重连定时器
    reconnectTimer = setTimeout(() => {
      console.log(`尝试重新连接 WebSocket (${reconnectCount.value}/${maxReconnectAttempts})...`)
      connect()
    }, reconnectInterval)
  }
  
  // 心跳机制
  const startHeartbeat = () => {
    // 确保只在客户端执行
    if (typeof window === 'undefined') return
    
    stopHeartbeat()
    
    heartbeatTimer = setInterval(() => {
      if (isConnected.value) {
        send(heartbeatMessage)
      } else {
        stopHeartbeat()
      }
    }, heartbeatInterval)
  }
  
  const stopHeartbeat = () => {
    if (heartbeatTimer) {
      clearInterval(heartbeatTimer)
      heartbeatTimer = null
    }
  }
  
  // 重置连接
  const reset = () => {
    disconnect()
    
    // 重置状态
    reconnectCount.value = 0
    
    // 延迟后重新连接
    setTimeout(() => {
      if (autoConnect) {
        connect()
      }
    }, 500)
  }
  
  // 监听网络状态变化
  watch(isOnline, (newVal) => {
    if (newVal && !isConnected.value && !isConnecting.value) {
      // 网络恢复，尝试重连
      reset()
    }
  })
  
  // 生命周期管理
  onUnmounted(() => {
    disconnect()
    listeners.value.clear()
    messageHandlers.value.clear()
    
    // 清理定时器
    if (reconnectTimer) {
      clearTimeout(reconnectTimer)
      reconnectTimer = null
    }
    
    // 清理网络状态监听器
    window.removeEventListener('online', handleOnlineChange)
    window.removeEventListener('offline', handleOnlineChange)
  })
  
  // 初始连接
  if (autoConnect) {
    connect()
  }
  
  return {
    socket,
    status,
    isConnected,
    isConnecting,
    canReconnect,
    reconnectCount,
    lastMessageTime,
    connect,
    disconnect,
    send,
    on,
    off,
    onMessage,
    offMessage,
    reset
  }
}
