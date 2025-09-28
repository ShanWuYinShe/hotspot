/**
 * 现代化 WebSocket 管理 - 基于 Nuxt v4 最佳实践
 * 利用 Nuxt v4 的响应式系统，自动处理 SSR/客户端差异
 */

export type WebSocketStatus = 'connecting' | 'connected' | 'disconnected' | 'error'

export interface WebSocketMessage<T = any> {
  type: string
  data: T
  timestamp: number
}

export interface WebSocketOptions {
  autoConnect?: boolean
  reconnectAttempts?: number
  reconnectInterval?: number
  heartbeatInterval?: number
}

/**
 * 现代化 WebSocket 管理 - 自动处理 SSR
 */
export function useWebSocket(url?: string, options: WebSocketOptions = {}) {
  const config = useRuntimeConfig()
  const authStore = useAuthStore()
  
  // 使用配置中的 WebSocket URL 或传入的 URL
  const wsUrl = url || `${config.public.wsBaseUrl || 'ws://localhost:8080/ws'}/chat`
  
  const {
    autoConnect = false, // 默认不自动连接，由应用控制
    reconnectAttempts = 5,
    reconnectInterval = 3000,
    heartbeatInterval = 30000
  } = options

  // 利用 Nuxt v4 的响应式系统
  const socket = ref<WebSocket | null>(null)
  const status = ref<WebSocketStatus>('disconnected')
  const lastMessage = ref<WebSocketMessage | null>(null)
  const reconnectCount = ref(0)

  // 计算属性 - 利用 Vue 3 的响应式
  const isConnected = computed(() => status.value === 'connected')
  const isConnecting = computed(() => status.value === 'connecting')

  // 事件系统 - 使用 Map 提高性能
  const messageHandlers = new Map<string, Set<(data: any) => void>>()
  const statusHandlers = new Set<(status: WebSocketStatus) => void>()

  // 定时器引用
  let heartbeatTimer: NodeJS.Timeout | null = null
  let reconnectTimer: NodeJS.Timeout | null = null

  /**
   * 检查是否在客户端环境
   */
  const isClient = typeof window !== 'undefined'

  /**
   * 连接 WebSocket - 自动处理 SSR
   */
  const connect = async () => {
    // SSR 环境下直接返回
    if (!isClient) {
      return
    }

    // 防止重复连接
    if (socket.value?.readyState === WebSocket.OPEN) {
      return
    }

    // 检查认证状态
    if (!authStore.isAuthenticated) {
      console.warn('WebSocket: 用户未认证，跳过连接')
      return
    }

    status.value = 'connecting'
    
    try {
      // 构建带认证的 WebSocket URL
      const token = authStore.token
      const authenticatedUrl = token ? `${wsUrl}?token=${token}` : wsUrl
      
      socket.value = new WebSocket(authenticatedUrl)
      
      socket.value.onopen = () => {
        status.value = 'connected'
        reconnectCount.value = 0
        startHeartbeat()
        notifyStatusChange('connected')
        
        if (import.meta.dev) {
          console.log('WebSocket 连接成功')
        }
      }

      socket.value.onmessage = (event) => {
        try {
          const message: WebSocketMessage = JSON.parse(event.data)
          lastMessage.value = message
          
          // 处理心跳响应
          if (message.type === 'pong') {
            return
          }
          
          // 触发对应类型的处理器
          const handlers = messageHandlers.get(message.type)
          if (handlers) {
            handlers.forEach(handler => {
              try {
                handler(message.data)
              } catch (error) {
                console.error(`WebSocket 消息处理器错误 (${message.type}):`, error)
              }
            })
          }
        } catch (error) {
          console.error('WebSocket 消息解析失败:', error)
        }
      }

      socket.value.onclose = (event) => {
        status.value = 'disconnected'
        stopHeartbeat()
        notifyStatusChange('disconnected')
        
        // 只有在非手动关闭时才重连
        if (event.code !== 1000 && reconnectCount.value < reconnectAttempts) {
          scheduleReconnect()
        }
      }

      socket.value.onerror = (error) => {
        status.value = 'error'
        notifyStatusChange('error')
        console.error('WebSocket 错误:', error)
      }
    } catch (error) {
      status.value = 'error'
      console.error('WebSocket 连接失败:', error)
    }
  }

  /**
   * 断开连接
   */
  const disconnect = () => {
    stopHeartbeat()
    clearReconnectTimer()
    
    if (socket.value) {
      socket.value.close(1000, 'Client disconnect') // 正常关闭
      socket.value = null
    }
    
    status.value = 'disconnected'
  }

  /**
   * 发送消息 - 增强类型安全
   */
  const send = <T = any>(type: string, data: T): boolean => {
    if (!isClient || !isConnected.value) {
      if (import.meta.dev) {
        console.warn('WebSocket 未连接，无法发送消息')
      }
      return false
    }

    try {
      const message: WebSocketMessage<T> = {
        type,
        data,
        timestamp: Date.now()
      }
      
      socket.value!.send(JSON.stringify(message))
      return true
    } catch (error) {
      console.error('WebSocket 发送消息失败:', error)
      return false
    }
  }

  /**
   * 监听消息 - 返回清理函数
   */
  const on = <T = any>(type: string, handler: (data: T) => void) => {
    if (!messageHandlers.has(type)) {
      messageHandlers.set(type, new Set())
    }
    messageHandlers.get(type)!.add(handler)

    // 返回清理函数
    return () => {
      const handlers = messageHandlers.get(type)
      if (handlers) {
        handlers.delete(handler)
        if (handlers.size === 0) {
          messageHandlers.delete(type)
        }
      }
    }
  }

  /**
   * 监听状态变化
   */
  const onStatusChange = (handler: (status: WebSocketStatus) => void) => {
    statusHandlers.add(handler)
    
    return () => {
      statusHandlers.delete(handler)
    }
  }

  /**
   * 心跳机制
   */
  const startHeartbeat = () => {
    if (!isClient || heartbeatInterval <= 0) return
    
    heartbeatTimer = setInterval(() => {
      if (isConnected.value) {
        send('ping', { timestamp: Date.now() })
      }
    }, heartbeatInterval)
  }

  const stopHeartbeat = () => {
    if (heartbeatTimer) {
      clearInterval(heartbeatTimer)
      heartbeatTimer = null
    }
  }

  /**
   * 重连机制
   */
  const scheduleReconnect = () => {
    if (!isClient) return
    
    clearReconnectTimer()
    
    reconnectTimer = setTimeout(() => {
      reconnectCount.value++
      if (import.meta.dev) {
        console.log(`WebSocket 重连尝试 ${reconnectCount.value}/${reconnectAttempts}`)
      }
      connect()
    }, reconnectInterval)
  }

  const clearReconnectTimer = () => {
    if (reconnectTimer) {
      clearTimeout(reconnectTimer)
      reconnectTimer = null
    }
  }

  /**
   * 通知状态变化
   */
  const notifyStatusChange = (newStatus: WebSocketStatus) => {
    statusHandlers.forEach(handler => {
      try {
        handler(newStatus)
      } catch (error) {
        console.error('WebSocket 状态处理器错误:', error)
      }
    })
  }

  // 监听认证状态变化
  watch(() => authStore.isAuthenticated, (isAuth) => {
    if (isAuth && autoConnect && isClient) {
      connect()
    } else if (!isAuth) {
      disconnect()
    }
  })

  // 自动连接 - 只在客户端
  if (autoConnect && authStore.isAuthenticated && isClient) {
    nextTick(() => connect())
  }

  // 自动清理 - 利用 Nuxt v4 的生命周期
  onBeforeUnmount(() => {
    disconnect()
    messageHandlers.clear()
    statusHandlers.clear()
  })

  return {
    // 只读状态
    status: readonly(status),
    isConnected: readonly(isConnected),
    isConnecting: readonly(isConnecting),
    lastMessage: readonly(lastMessage),
    
    // 方法
    connect,
    disconnect,
    send,
    on,
    onStatusChange
  }
}

// 全局 WebSocket 实例 - 利用 Nuxt v4 的单例模式
let globalWebSocket: ReturnType<typeof useWebSocket> | null = null

/**
 * 获取全局 WebSocket 实例
 */
export function useGlobalWebSocket() {
  if (!globalWebSocket) {
    globalWebSocket = useWebSocket(undefined, {
      autoConnect: true,
      reconnectAttempts: 5,
      reconnectInterval: 3000,
      heartbeatInterval: 30000
    })
  }
  
  return globalWebSocket
}
