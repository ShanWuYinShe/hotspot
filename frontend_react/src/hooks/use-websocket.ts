'use client';

import * as React from 'react';
import { useAuthStore } from '@/stores/auth';
import { getAccessToken } from '@/lib/api/client';

const WS_URL = process.env.NEXT_PUBLIC_WS_URL || 'ws://localhost:8080/ws';

export type WebSocketStatus = 'connecting' | 'connected' | 'disconnected' | 'error';

interface WebSocketMessage {
  type: string;
  payload: unknown;
}

interface UseWebSocketOptions {
  onMessage?: (message: WebSocketMessage) => void;
  onConnect?: () => void;
  onDisconnect?: () => void;
  onError?: (error: Event) => void;
  autoReconnect?: boolean;
  reconnectInterval?: number;
  maxReconnectAttempts?: number;
}

export function useWebSocket(options: UseWebSocketOptions = {}) {
  const {
    onMessage,
    onConnect,
    onDisconnect,
    onError,
    autoReconnect = true,
    reconnectInterval = 3000,
    maxReconnectAttempts = 5,
  } = options;

  const { isAuthenticated } = useAuthStore();
  const wsRef = React.useRef<WebSocket | null>(null);
  const reconnectAttempts = React.useRef(0);
  const reconnectTimer = React.useRef<ReturnType<typeof setTimeout> | undefined>(undefined);
  const heartbeatTimer = React.useRef<ReturnType<typeof setInterval> | undefined>(undefined);

  const [status, setStatus] = React.useState<WebSocketStatus>('disconnected');

  const connect = React.useCallback(() => {
    if (!isAuthenticated) return;
    
    const token = getAccessToken();
    if (!token) return;

    if (wsRef.current?.readyState === WebSocket.OPEN) return;

    setStatus('connecting');

    try {
      const ws = new WebSocket(`${WS_URL}?token=${encodeURIComponent(token)}`);
      wsRef.current = ws;

      ws.onopen = () => {
        setStatus('connected');
        reconnectAttempts.current = 0;
        onConnect?.();

        heartbeatTimer.current = setInterval(() => {
          if (ws.readyState === WebSocket.OPEN) {
            ws.send(JSON.stringify({ type: 'ping' }));
          }
        }, 30000);
      };

      ws.onmessage = (event) => {
        try {
          const message = JSON.parse(event.data) as WebSocketMessage;
          if (message.type === 'pong') return;
          onMessage?.(message);
        } catch {
          // Ignore parse errors
        }
      };

      ws.onclose = () => {
        setStatus('disconnected');
        clearInterval(heartbeatTimer.current);
        onDisconnect?.();

        if (autoReconnect && reconnectAttempts.current < maxReconnectAttempts) {
          reconnectAttempts.current += 1;
          reconnectTimer.current = setTimeout(() => {
            connect();
          }, reconnectInterval * reconnectAttempts.current);
        }
      };

      ws.onerror = (error) => {
        setStatus('error');
        onError?.(error);
      };
    } catch {
      setStatus('error');
    }
  }, [
    isAuthenticated,
    onConnect,
    onDisconnect,
    onError,
    onMessage,
    autoReconnect,
    reconnectInterval,
    maxReconnectAttempts,
  ]);

  const disconnect = React.useCallback(() => {
    clearTimeout(reconnectTimer.current);
    clearInterval(heartbeatTimer.current);
    if (wsRef.current) {
      wsRef.current.close();
      wsRef.current = null;
    }
    setStatus('disconnected');
  }, []);

  const send = React.useCallback((message: WebSocketMessage) => {
    if (wsRef.current?.readyState === WebSocket.OPEN) {
      wsRef.current.send(JSON.stringify(message));
      return true;
    }
    return false;
  }, []);

  React.useEffect(() => {
    if (isAuthenticated) {
      connect();
    } else {
      disconnect();
    }

    return () => {
      disconnect();
    };
  }, [isAuthenticated, connect, disconnect]);

  return {
    status,
    connect,
    disconnect,
    send,
  };
}
