/**
 * 现代化 WebSocket 管理 - 基于 Nuxt v4 最佳实践
 * 利用 Nuxt v4 的响应式系统，自动处理 SSR/客户端差异
 */

import type { WebSocketEvent } from '~/types/websocket'

export type WebSocketStatus = 'connecting' | 'connected' | 'disconnected' | 'error'

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
    heartbeatInterval = 25000 // ✅ 修正为25秒，符合API规范
  } = options

  // 利用 Nuxt v4 的响应式系统
  const socket = ref<WebSocket | null>(null)
  const status = ref<WebSocketStatus>('disconnected')
  const lastMessage = ref<WebSocketEvent | null>(null)
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
  let heartbeatTimeoutTimer: NodeJS.Timeout | null = null
  
  // ✅ 心跳超时检测（3次失败断开）
  let missedHeartbeats = 0
  const MAX_MISSED_HEARTBEATS = 3

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
          const message: WebSocketEvent = JSON.parse(event.data)
          lastMessage.value = message
          
          // 处理心跳响应
          if (message.type === 'Pong') { // ✅ 修正为标准事件名
            missedHeartbeats = 0 // ✅ 重置心跳失败计数
            clearHeartbeatTimeout()
            return
          }
          
          // 触发对应类型的处理器
          const handlers = messageHandlers.get(message.type)
          if (handlers) {
            handlers.forEach(handler => {
              try {
                handler(message.payload) // ✅ 使用 payload 而不是 data
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
   * ✅ 使用标准 WebSocketEvent 格式
   */
  const send = <T = any>(type: string, payload: T): boolean => {
    if (!isClient || !isConnected.value) {
      if (import.meta.dev) {
        console.warn('WebSocket 未连接，无法发送消息')
      }
      return false
    }

    try {
      const message: WebSocketEvent<T> = {
        type,
        payload // ✅ 使用 payload 而不是 data
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

  // ============================================================================
  // 订阅管理方法
  // ============================================================================

  /**
   * 增量订阅（追加到现有订阅）
   * ✅ 符合API规范：Subscribe 事件
   */
  const subscribe = (items: import('~/types/websocket').SubscriptionItem[]): boolean => {
    return send('Subscribe', { items })
  }

  /**
   * 取消订阅（从现有订阅中移除）
   * ✅ 符合API规范：Unsubscribe 事件
   */
  const unsubscribe = (items: import('~/types/websocket').SubscriptionItem[]): boolean => {
    return send('Unsubscribe', { items })
  }

  /**
   * 替换式订阅（覆盖现有订阅）
   * ✅ 符合API规范：SetSubscriptions 事件
   */
  const setSubscriptions = (items: import('~/types/websocket').SubscriptionItem[]): boolean => {
    return send('SetSubscriptions', { items })
  }

  /**
   * 便捷方法：订阅群聊消息
   */
  const subscribeToGroupChats = (groupChatIds: string[]): boolean => {
    return subscribe([{ source_type: 'group_chat', source_ids: groupChatIds }])
  }

  /**
   * 便捷方法：取消订阅群聊消息
   */
  const unsubscribeFromGroupChats = (groupChatIds: string[]): boolean => {
    return unsubscribe([{ source_type: 'group_chat', source_ids: groupChatIds }])
  }

  /**
   * 便捷方法：订阅用户状态
   */
  const subscribeToUsers = (userIds: string[]): boolean => {
    return subscribe([{ source_type: 'user', source_ids: userIds }])
  }

  /**
   * 便捷方法：取消订阅用户状态
   */
  const unsubscribeFromUsers = (userIds: string[]): boolean => {
    return unsubscribe([{ source_type: 'user', source_ids: userIds }])
  }

  /**
   * 便捷方法：订阅通知类型
   */
  const subscribeToNotifications = (notificationTypes: string[]): boolean => {
    return subscribe([{ source_type: 'notification', source_ids: notificationTypes }])
  }

  /**
   * 便捷方法：取消订阅通知类型
   */
  const unsubscribeFromNotifications = (notificationTypes: string[]): boolean => {
    return unsubscribe([{ source_type: 'notification', source_ids: notificationTypes }])
  }

  /**
   * 心跳机制
   * ✅ 符合API规范: 25秒间隔，发送标准 Ping 事件
   * ✅ 心跳超时检测: 5秒内必须收到 Pong，3次失败断开
   */
  const startHeartbeat = () => {
    if (!isClient || heartbeatInterval <= 0) return
    
    missedHeartbeats = 0 // 重置计数器
    
    heartbeatTimer = setInterval(() => {
      if (isConnected.value) {
        send('Ping', { timestamp: new Date().toISOString() }) // ✅ 使用标准事件名和ISO时间格式
        
        // ✅ 启动心跳超时检测（5秒内必须收到 Pong）
        heartbeatTimeoutTimer = setTimeout(() => {
          missedHeartbeats++
          if (import.meta.dev) {
            console.warn(`心跳超时 (${missedHeartbeats}/${MAX_MISSED_HEARTBEATS})`)
          }
          
          if (missedHeartbeats >= MAX_MISSED_HEARTBEATS) {
            console.error('心跳超时次数过多，主动断开连接')
            disconnect()
          }
        }, 5000) // 5秒超时
      }
    }, heartbeatInterval)
  }

  const stopHeartbeat = () => {
    if (heartbeatTimer) {
      clearInterval(heartbeatTimer)
      heartbeatTimer = null
    }
    clearHeartbeatTimeout()
  }
  
  const clearHeartbeatTimeout = () => {
    if (heartbeatTimeoutTimer) {
      clearTimeout(heartbeatTimeoutTimer)
      heartbeatTimeoutTimer = null
    }
  }

  /**
   * 重连机制 - 指数退避算法
   * ✅ 符合API规范：1秒 → 2秒 → 4秒 → ... → 最大30秒
   */
  const scheduleReconnect = () => {
    if (!isClient) return
    
    clearReconnectTimer()
    
    // 计算重连延迟（指数退避 + 随机抖动）
    const baseDelay = 1000 // 基础延迟1秒
    const maxDelay = 30000 // 最大延迟30秒
    const jitterFactor = 0.2 // 随机抖动±20%
    
    const exponentialDelay = Math.min(
      baseDelay * Math.pow(2, reconnectCount.value),
      maxDelay
    )
    
    const jitter = exponentialDelay * jitterFactor * (Math.random() * 2 - 1)
    const delay = Math.max(0, Math.floor(exponentialDelay + jitter))
    
    reconnectTimer = setTimeout(() => {
      reconnectCount.value++
      if (import.meta.dev) {
        console.log(`WebSocket 重连尝试 ${reconnectCount.value}/${reconnectAttempts}，延迟 ${delay}ms`)
      }
      connect()
    }, delay)
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
    onStatusChange,
    
    // 订阅管理方法
    subscribe,
    unsubscribe,
    setSubscriptions,
    subscribeToGroupChats,
    unsubscribeFromGroupChats,
    subscribeToUsers,
    unsubscribeFromUsers,
    subscribeToNotifications,
    unsubscribeFromNotifications
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
