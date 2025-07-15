/**
 * WebSocket 通知服务
 * 与后端的 WebSocketNotificationService 完全对接
 * 使用 Composition API 格式，通过 notification store 进行通知过滤
 */

import { useApiService } from './api.factory'
import { useWebSocket } from '~/composables/useWebSocket.client'
import { useAppSettings } from '~/composables/useAppSettings'
import type {
  WebSocketMessage,
  WebSocketClientMessage,
  ServerMessage,
  MessageNotification,
  GroupEventData,
  SystemNotificationData,
  RoomType,
  GroupEventType,
  SystemNotificationType
} from '~/types/websocket'
import { MessageType } from '~/types/websocket'

export interface WebSocketNotificationOptions {
  autoConnect?: boolean
  reconnectInterval?: number
  maxReconnectAttempts?: number
  heartbeatInterval?: number
}

// WebSocket 状态接口
interface WebSocketState {
  wsInstance: any
  isConnected: boolean
  subscriptions: Map<string, Set<Function>>
}

// WebSocket 状态管理（保持全局状态）
const wsState: WebSocketState = {
  wsInstance: null,
  isConnected: false,
  subscriptions: new Map<string, Set<Function>>()
}

/**
 * WebSocket 通知服务 Composition API
 * 统一管理 WebSocket 连接和消息处理
 */
export function useWebSocketNotificationService() {
  const apiService = useApiService()
  const { filterNotification } = useAppSettings()

  // 监听来自 useAppSettings 的 WebSocket 控制事件
  if (typeof window !== 'undefined') {
    window.addEventListener('websocket-enable', () => {
      console.log('收到 WebSocket 启用事件')
      initializeWebSocket()
    })
    
    window.addEventListener('websocket-disable', () => {
      console.log('收到 WebSocket 禁用事件')
      disconnectWebSocket()
    })
  }

  /**
   * 获取WebSocket连接URL
   */
  const getWebSocketUrl = (): string => {
    const config = useRuntimeConfig()

    // 首先尝试使用专门的WebSocket URL配置
    if (config.public.wsBaseUrl) {
      return config.public.wsBaseUrl
    }

    // 如果没有配置专门的WebSocket URL，从API URL推导
    const baseUrl = config.public.apiBaseUrl || 'http://localhost:8080'
    const wsUrl = baseUrl.replace(/^http/, 'ws').replace('/api/v1', '')
    return `${wsUrl}/ws`
  }

  /**
   * WebSocket身份验证
   */
  const authenticateWebSocket = async (): Promise<void> => {
    const authStore = useAuthStore()
    const token = authStore.token

    if (token && wsState.wsInstance) {
      const connectMessage: WebSocketClientMessage = {
        type: 'Connect',
        data: {
          token: token
        }
      }
      wsState.wsInstance.send(JSON.stringify(connectMessage))
    }
  }

  /**
   * 通知订阅者
   */
  const notifySubscribers = (eventType: string, data: any): void => {
    const subscribers = wsState.subscriptions.get(eventType)
    if (subscribers) {
      subscribers.forEach(callback => {
        try {
          callback(data)
        } catch (error) {
          console.error(`订阅者回调执行错误 (${eventType}):`, error)
        }
      })
    }
  }

  /**
   * 处理连接成功消息
   */
  const handleConnected = (message: ServerMessage): void => {
    const data = message.data as { user_id: number; session_id: string }
    console.log(`WebSocket连接成功: 用户ID=${data.user_id}, 会话ID=${data.session_id}`)

    // 触发连接成功的订阅者
    notifySubscribers('connected', data)
  }

  /**
   * 处理新消息
   */
  const handleNewMessage = (messageApi?: any) => {
    return (message: ServerMessage): void => {
      const data = message.data as MessageNotification
      console.log('收到新消息:', data)

      // 使用通知过滤服务检查是否应该显示通知
      const filteredNotification = filterNotification('message', data)
      
      if (filteredNotification?.shouldShow) {
        // 显示通知
        if (messageApi) {
          messageApi.info('收到新消息')
        }

        // 根据设置播放声音
        if (filteredNotification.shouldPlaySound) {
          console.log('播放消息通知声音')
          // 这里可以实际播放声音文件
        }

        // 根据设置触发震动
        if (filteredNotification.shouldVibrate) {
          if ('vibrate' in navigator) {
            navigator.vibrate([200, 100, 200])
          }
        }
      } else {
        console.log('消息通知已被用户设置过滤')
      }

      // 无论是否显示通知，都要触发新消息的订阅者（用于更新UI）
      notifySubscribers('new_message', data)
    }
  }

  /**
   * 处理用户开始输入
   */
  const handleUserStartedTyping = (message: ServerMessage): void => {
    const data = message.data as { user_id: number; user_name: string; room_type: RoomType; room_id: number }
    console.log('用户开始输入:', data)

    // 触发开始输入的订阅者
    notifySubscribers('user_started_typing', data)
  }

  /**
   * 处理用户停止输入
   */
  const handleUserStoppedTyping = (message: ServerMessage): void => {
    const data = message.data as { user_id: number; user_name: string; room_type: RoomType; room_id: number }
    console.log('用户停止输入:', data)

    // 触发停止输入的订阅者
    notifySubscribers('user_stopped_typing', data)
  }

  /**
   * 处理消息已读
   */
  const handleMessageRead = (message: ServerMessage): void => {
    const data = message.data as {
      message_id: number
      reader_id: number
      reader_name: string
      read_at: string
      room_type: RoomType
      room_id: number
    }
    console.log('消息已读:', data)

    // 触发消息已读的订阅者
    notifySubscribers('message_read', data)
  }

  /**
   * 处理群组事件
   */
  const handleGroupEvent = (messageApi?: any) => {
    return (message: ServerMessage): void => {
      const data = message.data as GroupEventData
      console.log('群组事件:', data)

      // 使用通知过滤服务检查是否应该显示通知
      const filteredNotification = filterNotification('group', data)
      
      if (filteredNotification?.shouldShow) {
        // 显示通知
        if (messageApi) {
          messageApi.info('群组事件更新')
        }

        // 根据设置播放声音
        if (filteredNotification.shouldPlaySound) {
          console.log('播放群组事件通知声音')
        }

        // 根据设置触发震动
        if (filteredNotification.shouldVibrate) {
          if ('vibrate' in navigator) {
            navigator.vibrate([100, 50, 100])
          }
        }
      } else {
        console.log('群组通知已被用户设置过滤')
      }

      // 无论是否显示通知，都要触发群组事件的订阅者（用于更新UI）
      notifySubscribers('group_event', data)
    }
  }

  /**
   * 处理系统通知
   */
  const handleSystemNotification = (messageApi?: any) => {
    return (message: ServerMessage): void => {
      const data = message.data as SystemNotificationData
      console.log('系统通知:', data)

      // 使用通知过滤服务检查是否应该显示通知
      const filteredNotification = filterNotification('system', data)
      
      if (filteredNotification?.shouldShow) {
        // 显示通知
        if (messageApi) {
          messageApi.info(data.content)
        }

        // 根据设置播放声音
        if (filteredNotification.shouldPlaySound) {
          console.log('播放系统通知声音')
        }

        // 根据设置触发震动
        if (filteredNotification.shouldVibrate) {
          if ('vibrate' in navigator) {
            navigator.vibrate([300])
          }
        }
      } else {
        console.log('系统通知已被用户设置过滤')
      }

      // 无论是否显示通知，都要触发系统通知的订阅者（用于更新UI）
      notifySubscribers('system_notification', data)
    }
  }

  /**
   * 处理消息删除
   */
  const handleMessageDeleted = (message: ServerMessage): void => {
    const data = message.data as { message_id: number; room_type: RoomType; room_id: number }
    console.log('消息已删除:', data)

    // 触发消息删除的订阅者
    notifySubscribers('message_deleted', data)
  }

  /**
   * 处理错误消息
   */
  const handleError = (messageApi?: any) => {
    return (message: ServerMessage): void => {
      const data = message.data as { error: string; code?: string }
      console.error('WebSocket错误:', data)

      // 使用消息API显示错误通知
      if (messageApi) {
        messageApi.error(`连接错误: ${data.error}`)
      }

      // 触发错误的订阅者
      notifySubscribers('error', data)
    }
  }

  /**
   * 处理WebSocket消息分发
   */
  const createWebSocketMessageHandler = (messageApi?: any) => {
    const handleNewMessageWithApi = handleNewMessage(messageApi)
    const handleGroupEventWithApi = handleGroupEvent(messageApi)
    const handleSystemNotificationWithApi = handleSystemNotification(messageApi)
    const handleErrorWithApi = handleError(messageApi)

    return (message: any): void => {
      try {
        const parsedMessage: ServerMessage = typeof message === 'string' ? JSON.parse(message) : message

        // 根据消息类型分发处理
        switch (parsedMessage.type) {
          case 'Connected':
            handleConnected(parsedMessage)
            break
          case 'NewMessage':
            handleNewMessageWithApi(parsedMessage)
            break
          case 'UserStartedTyping':
            handleUserStartedTyping(parsedMessage)
            break
          case 'UserStoppedTyping':
            handleUserStoppedTyping(parsedMessage)
            break
          case 'MessageRead':
            handleMessageRead(parsedMessage)
            break
          case 'GroupEvent':
            handleGroupEventWithApi(parsedMessage)
            break
          case 'SystemNotification':
            handleSystemNotificationWithApi(parsedMessage)
            break
          case 'MessageDeleted':
            handleMessageDeleted(parsedMessage)
            break
          case 'Error':
            handleErrorWithApi(parsedMessage)
            break
          case 'Pong':
            // 心跳响应，不需要特殊处理
            break
          default:
            console.log('未知的WebSocket消息类型:', parsedMessage.type)
        }
      } catch (error) {
        console.error('处理WebSocket消息时出错:', error)
      }
    }
  }

  /**
   * 初始化WebSocket连接
   */
  const initializeWebSocket = (options: WebSocketNotificationOptions = {}, messageApi?: any): void => {
    const wsUrl = getWebSocketUrl()
    const handleMessage = createWebSocketMessageHandler(messageApi)

    wsState.wsInstance = useWebSocket(wsUrl, {
      autoConnect: false,
      reconnectInterval: options.reconnectInterval || 3000,
      maxReconnectAttempts: options.maxReconnectAttempts || 5,
      heartbeatInterval: options.heartbeatInterval || 30000,
      heartbeatMessage: { type: 'Ping' }
    })

    // 监听WebSocket消息
    wsState.wsInstance?.onMessage('message', handleMessage)

    // 监听连接状态
    wsState.wsInstance?.on('open', () => {
      wsState.isConnected = true
      console.log('✅ WebSocket通知服务已连接')
      // 连接成功后进行身份验证
      authenticateWebSocket()
    })

    wsState.wsInstance?.on('close', () => {
      wsState.isConnected = false
      console.log('❌ WebSocket通知服务已断开')
    })

    wsState.wsInstance?.on('error', (error: any) => {
      console.error('WebSocket通知服务错误:', error)
    })
  }

  /**
   * 连接WebSocket
   */
  const connectWebSocket = (): void => {
    if (wsState.wsInstance) {
      wsState.wsInstance.connect()
    }
  }

  /**
   * 断开WebSocket连接
   */
  const disconnectWebSocket = (): void => {
    if (wsState.wsInstance) {
      wsState.wsInstance.disconnect()
    }
  }

  /**
   * 发送WebSocket消息
   */
  const sendWebSocketMessage = (message: WebSocketClientMessage): boolean => {
    if (wsState.wsInstance && wsState.isConnected) {
      return wsState.wsInstance.send(JSON.stringify(message))
    }
    return false
  }

  /**
   * 订阅WebSocket事件
   */
  const subscribeToWebSocketEvent = (eventType: string, callback: Function): (() => void) => {
    if (!wsState.subscriptions.has(eventType)) {
      wsState.subscriptions.set(eventType, new Set())
    }
    wsState.subscriptions.get(eventType)!.add(callback)

    // 返回取消订阅函数
    return () => {
      const subscribers = wsState.subscriptions.get(eventType)
      if (subscribers) {
        subscribers.delete(callback)
        if (subscribers.size === 0) {
          wsState.subscriptions.delete(eventType)
        }
      }
    }
  }

  /**
   * 加入群组房间
   */
  const joinWebSocketRoom = (roomType: 'Group' | 'Conversation', roomId: number): boolean => {
    if (roomType === 'Group') {
      const joinMessage: WebSocketClientMessage = {
        type: 'JoinGroup',
        data: {
          group_id: roomId.toString()
        }
      }
      console.log(`加入群组: ${roomId}`)
      return sendWebSocketMessage(joinMessage)
    } else {
      const joinMessage: WebSocketClientMessage = {
        type: 'JoinConversation',
        data: {
          conversation_id: roomId.toString()
        }
      }
      console.log(`加入对话: ${roomId}`)
      return sendWebSocketMessage(joinMessage)
    }
  }

  /**
   * 离开房间
   */
  const leaveWebSocketRoom = (roomType: 'Group' | 'Conversation', roomId: number): boolean => {
    if (roomType === 'Group') {
      const leaveMessage: WebSocketClientMessage = {
        type: 'LeaveGroup',
        data: {
          group_id: roomId.toString()
        }
      }
      console.log(`离开群组: ${roomId}`)
      return sendWebSocketMessage(leaveMessage)
    } else {
      const leaveMessage: WebSocketClientMessage = {
        type: 'LeaveConversation',
        data: {
          conversation_id: roomId.toString()
        }
      }
      console.log(`离开对话: ${roomId}`)
      return sendWebSocketMessage(leaveMessage)
    }
  }

  /**
   * 获取连接状态
   */
  const getWebSocketConnectionStatus = (): boolean => {
    return wsState.isConnected
  }

  return {
    getWebSocketUrl,
    authenticateWebSocket,
    notifySubscribers,
    handleConnected,
    handleNewMessage,
    handleUserStartedTyping,
    handleUserStoppedTyping,
    handleMessageRead,
    handleGroupEvent,
    handleSystemNotification,
    handleMessageDeleted,
    handleError,
    createWebSocketMessageHandler,
    initializeWebSocket,
    connectWebSocket,
    disconnectWebSocket,
    sendWebSocketMessage,
    subscribeToWebSocketEvent,
    joinWebSocketRoom,
    leaveWebSocketRoom,
    getWebSocketConnectionStatus
  }
}

