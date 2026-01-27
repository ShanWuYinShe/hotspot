import { AnyWebSocketEvent } from '@/types/websocket';

type Listener = (event: AnyWebSocketEvent) => void;

const HEARTBEAT_INTERVAL = 30_000;
const PONG_TIMEOUT = 10_000;
const RECONNECT_BACKOFF = [1000, 2000, 4000, 8000, 16000, 30000];
const MAX_RECONNECT_ATTEMPTS = 5;

const getWebSocketUrl = () => {
  if (process.env.NEXT_PUBLIC_WS_URL) {
    return process.env.NEXT_PUBLIC_WS_URL;
  }
  if (typeof window === 'undefined') {
    return '';
  }
  const protocol = window.location.protocol === 'https:' ? 'wss' : 'ws';
  return `${protocol}://${window.location.host}/ws`;
};

const createClientMessageId = () => {
  if (typeof crypto?.randomUUID === 'function') {
    return crypto.randomUUID();
  }
  return `${Date.now()}-${Math.random().toString(36).slice(2, 10)}`;
};

class WebSocketClient {
  private ws: WebSocket | null = null;
  private listeners = new Set<Listener>();
  private heartbeatTimer: number | null = null;
  private pongTimeoutTimer: number | null = null;
  private reconnectAttempts = 0;
  private manualClose = false;
  private isAuthenticated = false;
  private lastAccessToken: string | null = null;
  private userSubscriptions = new Set<number>();
  private groupSubscriptions = new Set<string>();

  connect(accessToken: string) {
    if (!accessToken) {
      return;
    }
    if (this.ws && this.ws.readyState === WebSocket.OPEN) {
      if (this.lastAccessToken !== accessToken) {
        this.sendAuth(accessToken);
        this.lastAccessToken = accessToken;
      }
      return;
    }

    this.manualClose = false;
    this.lastAccessToken = accessToken;
    const url = getWebSocketUrl();
    if (!url) {
      return;
    }
    this.ws = new WebSocket(url);

    this.ws.onopen = () => {
      this.reconnectAttempts = 0;
      this.sendAuth(accessToken);
    };

    this.ws.onmessage = (event) => {
      this.handleMessage(event.data);
    };

    this.ws.onclose = () => {
      this.cleanup();
      if (!this.manualClose) {
        this.scheduleReconnect();
      }
    };

    this.ws.onerror = () => {
      if (this.ws?.readyState === WebSocket.OPEN) {
        this.ws?.close();
      }
    };
  }

  disconnect() {
    this.manualClose = true;
    this.cleanup();
    this.ws?.close();
  }

  on(listener: Listener) {
    this.listeners.add(listener);
    return () => this.listeners.delete(listener);
  }

  subscribeUsers(userIds: number[]) {
    userIds.forEach((id) => this.userSubscriptions.add(id));
    this.send({
      type: 'Subscribe',
      payload: {
        client_msg_id: createClientMessageId(),
        type: 'user',
        ids: userIds,
      },
    });
  }

  subscribeGroups(groupIds: string[]) {
    groupIds.forEach((id) => this.groupSubscriptions.add(id));
    this.send({
      type: 'Subscribe',
      payload: {
        client_msg_id: createClientMessageId(),
        type: 'group',
        ids: groupIds,
      },
    });
  }

  setSubscriptions() {
    if (this.userSubscriptions.size > 0) {
      this.send({
        type: 'SetSubscriptions',
        payload: {
          client_msg_id: createClientMessageId(),
          type: 'user',
          ids: Array.from(this.userSubscriptions),
        },
      });
    }
    if (this.groupSubscriptions.size > 0) {
      this.send({
        type: 'SetSubscriptions',
        payload: {
          client_msg_id: createClientMessageId(),
          type: 'group',
          ids: Array.from(this.groupSubscriptions),
        },
      });
    }
  }

  send(event: AnyWebSocketEvent) {
    if (!this.ws || this.ws.readyState !== WebSocket.OPEN) {
      return;
    }
    this.ws.send(JSON.stringify(event));
  }

  sendPrivateTyping(targetUserId: number, isTyping: boolean) {
    this.send({
      type: 'PrivateTyping',
      payload: { target_user_id: targetUserId, is_typing: isTyping },
    });
  }

  sendGroupTyping(groupChatId: string, isTyping: boolean) {
    this.send({
      type: 'GroupTyping',
      payload: { group_chat_id: groupChatId, is_typing: isTyping },
    });
  }

  private sendAuth(accessToken: string) {
    this.send({
      type: 'Auth',
      payload: { access_token: accessToken },
    });
  }

  private handleMessage(raw: string) {
    let event: AnyWebSocketEvent | null = null;
    try {
      event = JSON.parse(raw) as AnyWebSocketEvent;
    } catch {
      return;
    }

    if (!event?.type) {
      return;
    }

    if (event.type === 'ConnectionEstablished') {
      this.isAuthenticated = true;
      this.startHeartbeat();
      this.setSubscriptions();
    }

    if (event.type === 'Pong') {
      this.clearPongTimeout();
    }

    this.listeners.forEach((listener) => listener(event));
  }

  private startHeartbeat() {
    this.stopHeartbeat();
    this.heartbeatTimer = window.setInterval(() => {
      if (!this.isAuthenticated) {
        return;
      }
      this.send({
        type: 'Ping',
        payload: { timestamp: new Date().toISOString() },
      });
      this.schedulePongTimeout();
    }, HEARTBEAT_INTERVAL);
  }

  private schedulePongTimeout() {
    this.clearPongTimeout();
    this.pongTimeoutTimer = window.setTimeout(() => {
      this.ws?.close();
    }, PONG_TIMEOUT);
  }

  private clearPongTimeout() {
    if (this.pongTimeoutTimer) {
      window.clearTimeout(this.pongTimeoutTimer);
      this.pongTimeoutTimer = null;
    }
  }

  private stopHeartbeat() {
    if (this.heartbeatTimer) {
      window.clearInterval(this.heartbeatTimer);
      this.heartbeatTimer = null;
    }
    this.clearPongTimeout();
  }

  private cleanup() {
    this.isAuthenticated = false;
    this.stopHeartbeat();
  }

  private scheduleReconnect() {
    if (this.reconnectAttempts >= MAX_RECONNECT_ATTEMPTS) {
      return;
    }
    const delay = RECONNECT_BACKOFF[Math.min(this.reconnectAttempts, RECONNECT_BACKOFF.length - 1)];
    this.reconnectAttempts += 1;
    window.setTimeout(() => {
      if (!this.lastAccessToken) {
        return;
      }
      this.connect(this.lastAccessToken);
    }, delay);
  }
}

export const websocketClient = new WebSocketClient();
