'use client';

import { useEffect, useRef, useCallback, useState } from 'react';
import { useAuthStore } from '@/stores/auth';
import type {
  WebSocketState,
  ClientEvent,
  ServerEvent,
  SubscriptionItem,
  ServerEventPayloadMap,
} from '@/types/websocket';

const resolveWebSocketUrl = () => {
  if (typeof location === 'undefined') {
    return null;
  }
  const protocol = location.protocol === 'https:' ? 'wss' : 'ws';
  return `${protocol}://${location.host}/ws`;
};
const HEARTBEAT_INTERVAL = 30000; // 30 秒
const PONG_TIMEOUT = 10000; // 10 秒
const RECONNECT_BASE_DELAY = 1000; // 1 秒
const RECONNECT_MAX_DELAY = 30000; // 30 秒
const AUTH_TIMEOUT = 5000; // 5 秒

type EventHandler<K extends keyof ServerEventPayloadMap> = (
  payload: ServerEventPayloadMap[K]
) => void;

interface UseWebSocketOptions {
  autoConnect?: boolean;
  onConnect?: () => void;
  onDisconnect?: () => void;
  onError?: (error: string) => void;
}

export function useWebSocket(options: UseWebSocketOptions = {}) {
  const { autoConnect = true, onConnect, onDisconnect, onError } = options;
  
  const [state, setState] = useState<WebSocketState>('disconnected');
  const [connectionId, setConnectionId] = useState<string | null>(null);
  
  const wsRef = useRef<WebSocket | null>(null);
  const heartbeatTimerRef = useRef<NodeJS.Timeout | null>(null);
  const pongTimeoutRef = useRef<NodeJS.Timeout | null>(null);
  const reconnectTimerRef = useRef<NodeJS.Timeout | null>(null);
  const reconnectAttemptsRef = useRef(0);
  const isAuthenticatedRef = useRef(false);
  const eventHandlersRef = useRef<Map<string, Set<EventHandler<keyof ServerEventPayloadMap>>>>(new Map());
  const subscriptionsRef = useRef<SubscriptionItem[]>([]);
  const connectRef = useRef<() => void>(() => {});
  
  const getAccessToken = useAuthStore((state) => state.getAccessToken);
  const isAuthenticated = useAuthStore((state) => state.isAuthenticated);

  // 清理定时器
  const clearTimers = useCallback(() => {
    if (heartbeatTimerRef.current) {
      clearInterval(heartbeatTimerRef.current);
      heartbeatTimerRef.current = null;
    }
    if (pongTimeoutRef.current) {
      clearTimeout(pongTimeoutRef.current);
      pongTimeoutRef.current = null;
    }
    if (reconnectTimerRef.current) {
      clearTimeout(reconnectTimerRef.current);
      reconnectTimerRef.current = null;
    }
  }, []);

  // 发送消息
  const send = useCallback((event: ClientEvent) => {
    if (wsRef.current?.readyState === WebSocket.OPEN) {
      wsRef.current.send(JSON.stringify(event));
    }
  }, []);

  // 开始心跳
  const startHeartbeat = useCallback(() => {
    heartbeatTimerRef.current = setInterval(() => {
      send({
        type: 'Ping',
        payload: { timestamp: new Date().toISOString() },
      });

      // 设置 Pong 超时
      pongTimeoutRef.current = setTimeout(() => {
        // Pong 超时，断开连接并重连
        wsRef.current?.close();
      }, PONG_TIMEOUT);
    }, HEARTBEAT_INTERVAL);
  }, [send]);

  // 处理服务端消息
  const handleMessage = useCallback((event: MessageEvent) => {
    try {
      const data = JSON.parse(event.data) as ServerEvent;
      
      // 处理特殊事件
      switch (data.type) {
        case 'ConnectionEstablished':
          isAuthenticatedRef.current = true;
          setConnectionId(data.payload.connection_id);
          setState('connected');
          reconnectAttemptsRef.current = 0;
          startHeartbeat();
          onConnect?.();
          
          // 恢复订阅
          if (subscriptionsRef.current.length > 0) {
            send({
              type: 'SetSubscriptions',
              payload: { items: subscriptionsRef.current },
            });
          }
          break;

        case 'Pong':
          // 清除 Pong 超时
          if (pongTimeoutRef.current) {
            clearTimeout(pongTimeoutRef.current);
            pongTimeoutRef.current = null;
          }
          break;

        case 'Error':
          onError?.(data.payload.message);
          break;
      }

      // 调用注册的事件处理器
      const handlers = eventHandlersRef.current.get(data.type);
      if (handlers) {
        handlers.forEach((handler) => {
          (handler as EventHandler<typeof data.type>)(data.payload as ServerEventPayloadMap[typeof data.type]);
        });
      }
    } catch {
      console.error('Failed to parse WebSocket message');
    }
  }, [onConnect, onError, send, startHeartbeat]);

  // 重连
  const reconnect = useCallback(() => {
    if (reconnectTimerRef.current) return;

    const delay = Math.min(
      RECONNECT_BASE_DELAY * Math.pow(2, reconnectAttemptsRef.current),
      RECONNECT_MAX_DELAY
    );

    reconnectTimerRef.current = setTimeout(() => {
      reconnectTimerRef.current = null;
      reconnectAttemptsRef.current++;
      connectRef.current();
    }, delay);
  }, []);

  // 连接
  const connect = useCallback(() => {
    if (!isAuthenticated) return;
    
    const token = getAccessToken();
    if (!token) return;

    // 关闭现有连接
    if (wsRef.current) {
      wsRef.current.close();
    }

    clearTimers();
    setState('connecting');
    isAuthenticatedRef.current = false;

    const wsUrl = resolveWebSocketUrl();
    if (!wsUrl) return;
    const ws = new WebSocket(wsUrl);
    wsRef.current = ws;

    ws.onopen = () => {
      // 发送认证消息
      send({
        type: 'Auth',
        payload: { access_token: token },
      });

      // 设置认证超时
      setTimeout(() => {
        if (!isAuthenticatedRef.current) {
          ws.close();
          onError?.('Authentication timeout');
        }
      }, AUTH_TIMEOUT);
    };

    ws.onmessage = handleMessage;

    ws.onclose = () => {
      clearTimers();
      setState('disconnected');
      setConnectionId(null);
      onDisconnect?.();
      
      // 自动重连
      if (isAuthenticated) {
        reconnect();
      }
    };

    ws.onerror = () => {
      setState('error');
    };
  }, [isAuthenticated, getAccessToken, clearTimers, send, handleMessage, onDisconnect, onError, reconnect]);

  useEffect(() => {
    connectRef.current = connect;
  }, [connect]);

  // 断开连接
  const disconnect = useCallback(() => {
    clearTimers();
    reconnectAttemptsRef.current = 0;
    
    if (wsRef.current) {
      wsRef.current.close();
      wsRef.current = null;
    }
  }, [clearTimers]);

  // 订阅
  const subscribe = useCallback((items: SubscriptionItem[]) => {
    // 更新本地订阅列表
    items.forEach((item) => {
      const existingIndex = subscriptionsRef.current.findIndex(
        (s) => s.type === item.type
      );
      if (existingIndex >= 0) {
        // 合并 ids
        const existing = subscriptionsRef.current[existingIndex];
        if (existing) {
          const ids = new Set<string | number>([
            ...(existing.ids as (string | number)[]),
            ...(item.ids as (string | number)[]),
          ]);
          (existing as { ids: (string | number)[] }).ids = Array.from(ids);
        }
      } else {
        subscriptionsRef.current.push(item);
      }
    });

    if (state === 'connected') {
      send({
        type: 'Subscribe',
        payload: { items },
      });
    }
  }, [state, send]);

  // 取消订阅
  const unsubscribe = useCallback((items: SubscriptionItem[]) => {
    // 更新本地订阅列表
    items.forEach((item) => {
      const existingIndex = subscriptionsRef.current.findIndex(
        (s) => s.type === item.type
      );
      if (existingIndex >= 0) {
        const existing = subscriptionsRef.current[existingIndex];
        if (existing) {
          const idsToRemove = new Set<string | number>(item.ids as (string | number)[]);
          (existing as { ids: (string | number)[] }).ids = (
            existing.ids as (string | number)[]
          ).filter((id) => !idsToRemove.has(id));
          if (existing.ids.length === 0) {
            subscriptionsRef.current.splice(existingIndex, 1);
          }
        }
      }
    });

    if (state === 'connected') {
      send({
        type: 'Unsubscribe',
        payload: { items },
      });
    }
  }, [state, send]);

  // 注册事件处理器
  const on = useCallback(<K extends keyof ServerEventPayloadMap>(
    eventType: K,
    handler: EventHandler<K>
  ) => {
    if (!eventHandlersRef.current.has(eventType)) {
      eventHandlersRef.current.set(eventType, new Set());
    }
    eventHandlersRef.current.get(eventType)?.add(handler as EventHandler<keyof ServerEventPayloadMap>);

    // 返回取消注册函数
    return () => {
      eventHandlersRef.current.get(eventType)?.delete(handler as EventHandler<keyof ServerEventPayloadMap>);
    };
  }, []);

  // 发送打字状态
  const sendTyping = useCallback((targetUserId: number, isTyping: boolean) => {
    send({
      type: 'PrivateTyping',
      payload: { target_user_id: targetUserId, is_typing: isTyping },
    });
  }, [send]);

  const sendGroupTyping = useCallback((groupChatId: string, isTyping: boolean) => {
    send({
      type: 'GroupTyping',
      payload: { group_chat_id: groupChatId, is_typing: isTyping },
    });
  }, [send]);

  // 自动连接
  useEffect(() => {
    if (autoConnect && isAuthenticated) {
      connect();
    }

    return () => {
      disconnect();
    };
  }, [autoConnect, isAuthenticated, connect, disconnect]);

  return {
    state,
    connectionId,
    isConnected: state === 'connected',
    connect,
    disconnect,
    subscribe,
    unsubscribe,
    on,
    sendTyping,
    sendGroupTyping,
  };
}
