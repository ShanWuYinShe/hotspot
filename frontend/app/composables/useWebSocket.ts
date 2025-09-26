/**
 * WebSocket连接管理composable
 * 处理实时消息通信 - 基于新的 websocket.ts 类型定义
 */

import { OTHER_ENDPOINTS } from "~/config/api";
import type {
  ServerMessage,
  ClientMessage,
  RoomType,
  ConnectedData,
  NewMessageData,
  MessageReadData,
  UserOnlineData,
  UserOfflineData,
  ErrorData,
  GroupMemberJoinedData,
  SystemNotificationData,
  TypingIndicatorData,
  ClientMessageType,
  ServerMessageType,
  WebSocketConnectionState,
  WebSocketEventHandlers,
  AuthenticateData,
  JoinRoomData,
  LeaveRoomData,
  SendMessageData,
  TypingData,
  MarkReadData,
} from "~/types/websocket";

type EventMap = {
  connected: ConnectedData | undefined;
  authenticated: any | undefined;
  newMessage: NewMessageData;
  messageRead: MessageReadData;
  userOnline: UserOnlineData;
  userOffline: UserOfflineData;
  groupMemberJoined: GroupMemberJoinedData | undefined;
  systemNotification: SystemNotificationData | undefined;
  typingIndicator: TypingIndicatorData | undefined;
  error: ErrorData | undefined;
};

export const useWebSocket = () => {
  const config = useRuntimeConfig();
  const authStore = useAuthStore();
  const toast = useToast();

  // 状态管理
  const socket = ref<WebSocket | null>(null);
  const connectionState = ref<WebSocketConnectionState>('disconnected');
  const isConnected = computed(() => connectionState.value === 'connected');
  const isAuthenticated = computed(() => connectionState.value === 'authenticated');
  const reconnectAttempts = ref(0);
  const maxReconnectAttempts = 5;
  const reconnectDelay = ref(1000);
  const heartbeatInterval = ref<NodeJS.Timeout | null>(null);
  const reconnectTimeout = ref<NodeJS.Timeout | null>(null);

  // 事件监听器
  const eventListeners = ref<Map<keyof EventMap, Set<Function>>>(new Map());

  // 消息队列（连接断开时缓存消息）
  const messageQueue = ref<ClientMessage[]>([]);

  // 订阅的房间
  const subscribedRooms = ref<Set<string>>(new Set());

  // 获取WebSocket URL
  const getWebSocketUrl = (): string => {
    const wsProtocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
    const apiBase = config.public.apiBase as string;
    const baseUrl = apiBase || window.location.host;
    return `${wsProtocol}//${baseUrl.replace(/^https?:\/\//, '')}/ws/`;
  };

  // 发送消息
  const sendMessage = (message: ClientMessage): void => {
    if (socket.value && socket.value.readyState === WebSocket.OPEN) {
      socket.value.send(JSON.stringify(message));
    } else {
      // 如果连接断开，将消息加入队列
      messageQueue.value.push(message);
      console.warn('[WebSocket] 连接断开，消息已加入队列:', message);
    }
  };

  // 处理队列中的消息
  const processMessageQueue = (): void => {
    while (messageQueue.value.length > 0) {
      const message = messageQueue.value.shift();
      if (message) {
        sendMessage(message);
      }
    }
  };

  // 事件监听
  const on = <K extends keyof EventMap>(
    event: K,
    callback: (data: EventMap[K]) => void
  ): void => {
    if (!eventListeners.value.has(event)) {
      eventListeners.value.set(event, new Set());
    }
    eventListeners.value.get(event)?.add(callback);
  };

  // 移除事件监听
  const off = <K extends keyof EventMap>(
    event: K,
    callback: (data: EventMap[K]) => void
  ): void => {
    eventListeners.value.get(event)?.delete(callback);
  };

  // 触发事件
  const emit = <K extends keyof EventMap>(event: K, data: EventMap[K]): void => {
    const listeners = eventListeners.value.get(event);
    if (listeners) {
      listeners.forEach((callback) => {
        try {
          callback(data);
        } catch (error) {
          console.error(`[WebSocket] 事件处理器错误 (${event}):`, error);
        }
      });
    }
  };

  // 处理连接打开
  const handleOpen = (): void => {
    console.log('[WebSocket] 连接已建立');
    connectionState.value = 'connected';
    reconnectAttempts.value = 0;
    reconnectDelay.value = 1000;

    // 发送认证消息
    if (authStore.token) {
      authenticate();
    }

    // 开始心跳
    startHeartbeat();
  };

  // 处理消息接收
  const handleMessage = (event: MessageEvent): void => {
    try {
      const message: ServerMessage = JSON.parse(event.data);
      
      switch (message.type) {
        case 'connected':
          emit('connected', message.data as ConnectedData);
          break;
          
        case 'authenticated':
          connectionState.value = 'authenticated';
          emit('authenticated', message.data);
          // 认证成功后处理队列中的消息
          processMessageQueue();
          // 重新加入之前订阅的房间
          resubscribeRooms();
          break;
          
        case 'NewMessage':
          emit('newMessage', message.data as NewMessageData);
          break;
          
        case 'MessageRead':
          emit('messageRead', message.data as MessageReadData);
          break;
          
        case 'TypingIndicator':
          emit('typingIndicator', message.data as TypingIndicatorData);
          break;
          
        case 'UserOnline':
          emit('userOnline', message.data as UserOnlineData);
          break;
          
        case 'UserOffline':
          emit('userOffline', message.data as UserOfflineData);
          break;
          
        case 'GroupMemberJoined':
          emit('groupMemberJoined', message.data as GroupMemberJoinedData);
          break;
          
        case 'SystemNotification':
          emit('systemNotification', message.data as SystemNotificationData);
          break;
          
        case 'error':
          const errorData = message.data as ErrorData;
          emit('error', errorData);
          console.error('[WebSocket] 服务器错误:', errorData);
          toast.add({
            title: '连接错误',
            description: errorData.message,
            color: 'error'
          });
          break;
          
        case 'pong':
          // 心跳响应，更新最后心跳时间
          break;
          
        default:
          console.warn('[WebSocket] 未知消息类型:', message);
      }
    } catch (error) {
      console.error('[WebSocket] 消息解析错误:', error);
    }
  };

  // 处理连接关闭
  const handleClose = (event: CloseEvent): void => {
    console.log('[WebSocket] 连接已关闭:', event.code, event.reason);
    connectionState.value = 'disconnected';
    
    // 停止心跳
    stopHeartbeat();
    
    // 如果不是正常关闭，尝试重连
    if (event.code !== 1000 && reconnectAttempts.value < maxReconnectAttempts) {
      scheduleReconnect();
    }
  };

  // 处理连接错误
  const handleError = (event: Event): void => {
    console.error('[WebSocket] 连接错误:', event);
    connectionState.value = 'error';
    
    const errorData: ErrorData = {
      code: 1001, // CONNECTION_ERROR
      message: '连接失败',
      details: event
    };
    
    emit('error', errorData);
  };

  // 开始心跳
  const startHeartbeat = (): void => {
    stopHeartbeat();
    heartbeatInterval.value = setInterval(() => {
      if (socket.value && socket.value.readyState === WebSocket.OPEN) {
        sendMessage({
          type: 'ping',
          timestamp: Date.now(),
          data: {}
        });
      }
    }, 30000) as any; // 30秒心跳间隔
  };

  // 停止心跳
  const stopHeartbeat = (): void => {
    if (heartbeatInterval.value) {
      clearInterval(heartbeatInterval.value);
      heartbeatInterval.value = null;
    }
  };

  // 安排重连
  const scheduleReconnect = (): void => {
    if (reconnectTimeout.value) {
      clearTimeout(reconnectTimeout.value);
    }
    
    reconnectAttempts.value++;
    const delay = Math.min(reconnectDelay.value * Math.pow(2, reconnectAttempts.value - 1), 30000);
    
    console.log(`[WebSocket] 将在 ${delay}ms 后尝试重连 (${reconnectAttempts.value}/${maxReconnectAttempts})`);
    
    reconnectTimeout.value = setTimeout(() => {
      connect();
    }, delay);
  };

  // 重新订阅房间
  const resubscribeRooms = (): void => {
    subscribedRooms.value.forEach(roomId => {
      // 这里需要根据房间ID的格式判断房间类型
      // 简化处理，假设都是群组房间
      joinRoom('group', roomId);
    });
  };

  // 连接WebSocket
  const connect = (): void => {
    if (socket.value && socket.value.readyState === WebSocket.OPEN) {
      return;
    }

    try {
      connectionState.value = 'connecting';
      socket.value = new WebSocket(getWebSocketUrl());
      
      socket.value.onopen = handleOpen;
      socket.value.onmessage = handleMessage;
      socket.value.onclose = handleClose;
      socket.value.onerror = handleError;
      
    } catch (error) {
      console.error('[WebSocket] 连接创建失败:', error);
      connectionState.value = 'error';
    }
  };

  // 断开连接
  const disconnect = (): void => {
    if (reconnectTimeout.value) {
      clearTimeout(reconnectTimeout.value);
      reconnectTimeout.value = null;
    }
    
    stopHeartbeat();
    
    if (socket.value) {
      socket.value.close(1000, '用户主动断开');
      socket.value = null;
    }
    
    connectionState.value = 'disconnected';
    reconnectAttempts.value = 0;
  };

  // 认证
  const authenticate = (): void => {
    if (!authStore.token) {
      console.error('[WebSocket] 没有认证令牌');
      return;
    }

    const authMessage: ClientMessage = {
      type: 'authenticate',
      timestamp: Date.now(),
      data: {
        token: authStore.token
      } as AuthenticateData
    };

    sendMessage(authMessage);
  };

  // 加入房间
  const joinRoom = (roomType: RoomType, roomId: string): void => {
    const joinMessage: ClientMessage = {
      type: 'join_room',
      timestamp: Date.now(),
      data: {
        room_type: roomType,
        room_id: roomId
      } as JoinRoomData
    };

    sendMessage(joinMessage);
    subscribedRooms.value.add(roomId);
  };

  // 离开房间
  const leaveRoom = (roomType: RoomType, roomId: string): void => {
    const leaveMessage: ClientMessage = {
      type: 'leave_room',
      timestamp: Date.now(),
      data: {
        room_type: roomType,
        room_id: roomId
      } as LeaveRoomData
    };

    sendMessage(leaveMessage);
    subscribedRooms.value.delete(roomId);
  };

  // 发送聊天消息
  const sendChatMessage = (groupId: string, content: string, messageType: string = 'text'): void => {
    const chatMessage: ClientMessage = {
      type: 'send_message',
      timestamp: Date.now(),
      data: {
        group_id: groupId,
        content,
        message_type: messageType
      } as SendMessageData
    };

    sendMessage(chatMessage);
  };

  // 发送输入状态
  const sendTyping = (groupId: string, isTyping: boolean): void => {
    const typingMessage: ClientMessage = {
      type: isTyping ? 'typing_start' : 'typing_stop',
      timestamp: Date.now(),
      data: {
        group_id: groupId
      } as TypingData
    };

    sendMessage(typingMessage);
  };

  // 标记消息已读
  const markAsRead = (groupId: string, messageId?: string): void => {
    const markReadMessage: ClientMessage = {
      type: 'mark_read',
      timestamp: Date.now(),
      data: {
        group_id: groupId,
        message_id: messageId
      } as MarkReadData
    };

    sendMessage(markReadMessage);
  };

  // 组件卸载时清理
  onUnmounted(() => {
    disconnect();
    eventListeners.value.clear();
  });

  return {
    // 状态
    connectionState: readonly(connectionState),
    isConnected,
    isAuthenticated,
    
    // 连接管理
    connect,
    disconnect,
    
    // 房间管理
    joinRoom,
    leaveRoom,
    
    // 消息发送
    sendChatMessage,
    sendTyping,
    markAsRead,
    
    // 事件监听
    on,
    off,
    
    // 原始发送方法（高级用法）
    sendMessage,
  };
};