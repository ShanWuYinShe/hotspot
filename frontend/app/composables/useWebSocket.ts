/**
 * WebSocket连接管理composable
 * 处理实时消息通信
 */

import { OTHER_ENDPOINTS } from "~/config/api";
import type {
  ServerMessage,
  ClientMessage,
  RoomType,
  WsConnectedData,
  WsNewMessagePayload,
  WsMessageStatusUpdatePayload,
  WsUserStatusUpdatePayload,
  WsErrorPayload,
  WsGroupEventPayload,
  WsSystemNotificationPayload,
  WsMessageDeletedPayload,
  WsMessageReadPayload,
  WsUserTypingPayload,
} from "~/types/realtime";
import { useErrorHandler } from "~/composables/useErrorHandler";

type EventMap = {
  connected: WsConnectedData | undefined;
  newMessage: WsNewMessagePayload;
  messageStatusUpdate: WsMessageStatusUpdatePayload;
  userStatusUpdate: WsUserStatusUpdatePayload;
  groupEvent: WsGroupEventPayload | undefined;
  systemNotification: WsSystemNotificationPayload | undefined;
  messageDeleted: WsMessageDeletedPayload | undefined;
  messageRead: WsMessageReadPayload | undefined;
  userTyping: (WsUserTypingPayload & { typing: boolean }) | undefined;
};

export const useWebSocket = () => {
  const config = useRuntimeConfig();
  const authStore = useAuthStore();
  const toast = useSmartToast();
  const { handleError } = useErrorHandler();

  // WebSocket连接状态
  const isConnected = ref(false);
  const isConnecting = ref(false);
  const connState = ref<
    "idle" | "connecting" | "authenticating" | "connected" | "closing"
  >("idle");
  const socket = ref<WebSocket | null>(null);
  // 去重订阅集：记录已订阅的房间，避免重复 Join
  const currentSubscriptions = new Set<string>();
  const reconnectAttempts = ref(0);
  const maxReconnectAttempts = 3;
  const reconnectInterval = ref<NodeJS.Timeout | null>(null);
  // 心跳/看门狗
  const HEARTBEAT_MS = 30000;
  let heartbeatTimer: NodeJS.Timeout | null = null;
  let watchdogTimer: NodeJS.Timeout | null = null;
  let lastPongAt = Date.now();
  const outboundQueue: ClientMessage[] = [];

  // 消息事件监听器 - 使用 WeakMap 避免内存泄漏
  const messageListeners = ref<
    Map<keyof EventMap | string, Array<(data: unknown) => void>>
  >(new Map());
  
  // 清理监听器的方法
  const clearAllListeners = () => {
    messageListeners.value.clear();
  };

  // WebSocket URL
  const getWebSocketUrl = () => {
    const pub: any = config.public as any;
    const wsUrl = pub?.wsBaseUrl;
    if (wsUrl) return wsUrl;

    const apiUrl = pub?.apiBaseUrl;
    if (apiUrl) {
      const u = new URL(apiUrl);
      const proto = u.protocol === "https:" ? "wss" : "ws";
      return `${proto}://${u.host}${OTHER_ENDPOINTS.WS}`;
    }

    return "ws://localhost:8080/realtime/ws";
  };

  // 添加消息监听器
  function addEventListener<T extends keyof EventMap>(
    type: T,
    listener: (data: EventMap[T]) => void,
  ): void;
  function addEventListener(
    type: string,
    listener: (data: unknown) => void,
  ): void;
  function addEventListener(type: string, listener: (data: unknown) => void) {
    if (!messageListeners.value.has(type)) {
      messageListeners.value.set(type, []);
    }
    messageListeners.value.get(type)!.push(listener as any);
  }

  // 移除消息监听器
  function removeEventListener<T extends keyof EventMap>(
    type: T,
    listener: (data: EventMap[T]) => void,
  ): void;
  function removeEventListener(
    type: string,
    listener: (data: unknown) => void,
  ): void;
  function removeEventListener(
    type: string,
    listener: (data: unknown) => void,
  ) {
    const listeners = messageListeners.value.get(type);
    if (listeners) {
      const index = listeners.indexOf(listener as any);
      if (index > -1) {
        listeners.splice(index, 1);
      }
    }
  }

  // 触发消息事件
  const triggerEvent = <T extends keyof EventMap>(
    type: T,
    data: EventMap[T],
  ) => {
    const listeners = messageListeners.value.get(type as string);
    if (listeners) {
      listeners.forEach((listener) =>
        (listener as (d: EventMap[T]) => void)(data),
      );
    }
  };

  // 发送消息
  const sendRaw = (message: ClientMessage) => {
    if (!socket.value || socket.value.readyState !== WebSocket.OPEN)
      return false;
    try {
      socket.value.send(JSON.stringify(message));
      if (import.meta.dev) console.debug("发送WebSocket消息:", message);
      return true;
    } catch (error) {
      console.error("发送WebSocket消息失败:", error);
      return false;
    }
  };

  const sendMessage = (message: ClientMessage) => {
    // 仅在已通过认证后直接发送，其余阶段入队
    if (connState.value === "connected" && sendRaw(message)) return;
    outboundQueue.push(message);
    if (import.meta.dev) console.debug("消息入队等待连接:", message);
  };

  const flushQueue = () => {
    if (connState.value !== "connected") return;
    while (outboundQueue.length) {
      const msg = outboundQueue.shift()!;
      if (!sendRaw(msg)) {
        // 发送失败则放回队列尾部并中止，待下次机会
        outboundQueue.unshift(msg);
        break;
      }
    }
  };

  // 连接WebSocket
  const connect = () => {
    if (isConnecting.value || isConnected.value) {
      return;
    }

    if (!authStore.token) {
      console.warn("未找到认证token，无法连接WebSocket");
      return;
    }

    isConnecting.value = true;

    try {
      const wsUrl = getWebSocketUrl();
      if (import.meta.dev) console.debug("连接WebSocket:", wsUrl);

      connState.value = "connecting";
      socket.value = new WebSocket(wsUrl);

      socket.value.onopen = () => {
        if (import.meta.dev) console.debug("WebSocket连接已建立");
        isConnecting.value = false;
        reconnectAttempts.value = 0;
        connState.value = "authenticating";
        // 启动心跳/看门狗
        startHeartbeat();
        startWatchdog();

        // 发送认证消息
        sendRaw({
          type: "Connect",
          data: { token: authStore.token as string },
        });
        // 注意：isConnected 在收到服务端 Connected 消息后才置true
      };

      socket.value.onmessage = (event) => {
        try {
          const message: ServerMessage = JSON.parse(event.data);
          if (import.meta.dev) console.debug("收到WebSocket消息:", message);

          handleServerMessage(message);
        } catch (error) {
          console.error("解析WebSocket消息失败:", error);
        }
      };

      socket.value.onclose = (event) => {
        if (import.meta.dev)
          console.debug("WebSocket连接已关闭:", event.code, event.reason);
        
        // 只在非正常关闭且开发模式下记录错误
        // 1000: 正常关闭, 1001: 端点离开, 1005: 无状态码
        const isNormalClose = [1000, 1001, 1005].includes(event.code);
        if (import.meta.dev && !isNormalClose && event.code !== 1006) {
          handleError({
            status: event.code,
            message: `WebSocket已关闭: ${event.code} ${event.reason || ""}`.trim(),
          }, "WebSocket连接");
        }
        
        isConnected.value = false;
        isConnecting.value = false;
        connState.value = "idle";
        currentSubscriptions.clear();
        socket.value = null;
        stopHeartbeat();
        stopWatchdog();

        // 只有在异常关闭且有token时才尝试重连
        if (!isNormalClose && event.code !== 1006 && authStore.token) {
          scheduleReconnect();
        }
      };

      socket.value.onerror = (error) => {
        if (import.meta.dev) console.error("WebSocket连接错误:", error);
        // 只在开发模式下或重连次数较多时显示错误
        if (import.meta.dev || reconnectAttempts.value >= 2) {
          handleError({
            name: "NetworkError",
            message: "WebSocket连接错误",
          }, "WebSocket连接");
        }
        isConnecting.value = false;
        connState.value = "idle";
        stopHeartbeat();
        stopWatchdog();

        if (authStore.token) {
          scheduleReconnect();
        }
      };
    } catch (error) {
      console.error("创建WebSocket连接失败:", error);
      handleError({
        name: "NetworkError",
        message: "创建WebSocket连接失败",
      }, "WebSocket连接");
      isConnecting.value = false;
      scheduleReconnect();
    }
  };

  // 处理服务器消息
  const handleServerMessage = (message: ServerMessage) => {
    switch (message.type) {
      case "Connected":
        isConnected.value = true;
        connState.value = "connected";
        if (import.meta.dev) console.debug("WebSocket认证成功:", message.data);
        triggerEvent("connected", message.data);
        // 认证成功后默认订阅用户私有房间，接收系统/私有通知
        if (!currentSubscriptions.has("user:self")) {
          sendRaw({ type: "JoinUserRoom" });
          currentSubscriptions.add("user:self");
        }
        // 刷新发送队列
        flushQueue();
        break;

      case "Pong":
        if (import.meta.dev) console.debug("收到心跳响应");
        lastPongAt = Date.now();
        break;

      case "NewMessage":
        if (import.meta.dev) console.debug("收到新消息:", message.data);
        triggerEvent("newMessage", message.data);
        break;

      case "MessageStatusUpdate":
        if (import.meta.dev) console.debug("消息状态更新:", message.data);
        triggerEvent("messageStatusUpdate", message.data);
        break;

      case "UserStatusUpdate":
        if (import.meta.dev) console.debug("用户状态更新:", message.data);
        triggerEvent("userStatusUpdate", message.data);
        break;

      case "Error": {
        const anyData = message.data as any;
        const code = anyData?.code ?? anyData?.error ?? "";
        const msg = anyData?.message ?? anyData?.msg ?? "连接出现错误";
        console.error("WebSocket错误:", code, msg);
        handleError({
          status: code,
          message: String(msg || "WebSocket错误"),
        }, "WebSocket服务器");
        // 令牌失效时，优先尝试静默刷新并重连
        if (
          String(code).toUpperCase() === "INVALID_TOKEN" &&
          authStore.refreshToken
        ) {
          (async () => {
            try {
              await authStore.refreshAuth();
              // 使用新token重连
              disconnect();
              connect();
            } catch (e) {
              // 刷新失败，清理认证并提示
              authStore.clearAuth();
              toast.warning({
                title: "登录已过期",
                description: "请重新登录"
              });
            }
          })();
        } else {
          toast.error({
            title: "WebSocket错误",
            description: msg
          });
        }
        break;
      }

      case "GroupEvent":
        if (import.meta.dev) console.debug("群组事件:", message.data);
        triggerEvent("groupEvent", message.data);
        break;

      case "SystemNotification":
        if (import.meta.dev) console.debug("系统通知:", message.data);
        triggerEvent("systemNotification", message.data);
        break;

      case "MessageDeleted":
        if (import.meta.dev) console.debug("消息删除:", message.data);
        triggerEvent("messageDeleted", message.data);
        break;

      case "MessageRead":
        if (import.meta.dev) console.debug("消息已读:", message.data);
        triggerEvent("messageRead", message.data);
        break;

      case "UserStartedTyping": {
        if (import.meta.dev) console.debug("用户输入中:", message.data);
        const payload =
          message.data as import("~/types/realtime").WsUserTypingPayload;
        triggerEvent("userTyping", { ...payload, typing: true });
        break;
      }
      case "UserStoppedTyping": {
        if (import.meta.dev) console.debug("用户停止输入:", message.data);
        const payload =
          message.data as import("~/types/realtime").WsUserTypingPayload;
        triggerEvent("userTyping", { ...payload, typing: false });
        break;
      }

      // no default: 让 TypeScript 保持穷尽检查
    }
  };

  // 安排重连
  const scheduleReconnect = () => {
    if (reconnectAttempts.value >= maxReconnectAttempts) {
      if (import.meta.dev) console.error("WebSocket重连次数达到上限，停止重连");
      handleError({
        name: "NetworkError",
        message: "WebSocket重连次数达到上限，停止重连",
      }, "WebSocket重连");
      toast.error({
        title: "连接失败",
        description: "无法连接到服务器，请检查网络连接"
      });
      return;
    }

    if (reconnectInterval.value) {
      clearTimeout(reconnectInterval.value);
    }

    const base = Math.min(1000 * Math.pow(2, reconnectAttempts.value), 30000);
    const jitter = base * (0.8 + Math.random() * 0.4);
    const delay = Math.round(jitter);
    if (import.meta.dev) console.log(`${delay}ms后尝试重连WebSocket`);

    reconnectInterval.value = setTimeout(() => {
      reconnectAttempts.value++;
      connect();
    }, delay);
  };

  // 心跳/看门狗
  const startHeartbeat = () => {
    stopHeartbeat();
    lastPongAt = Date.now();
    heartbeatTimer = setInterval(() => {
      if (isConnected.value) sendHeartbeat();
    }, HEARTBEAT_MS);
  };
  const stopHeartbeat = () => {
    if (heartbeatTimer) {
      clearInterval(heartbeatTimer);
      heartbeatTimer = null;
    }
  };
  const startWatchdog = () => {
    stopWatchdog();
    watchdogTimer = setInterval(
      () => {
        const timeoutMs = HEARTBEAT_MS * 2.5;
        if (isConnected.value && Date.now() - lastPongAt > timeoutMs) {
          if (import.meta.dev) console.warn("心跳超时，准备重连");
          handleError({
            name: "NetworkError",
            message: "WebSocket心跳超时",
          }, "WebSocket心跳");
          try {
            socket.value?.close();
          } catch {}
          isConnected.value = false;
          isConnecting.value = false;
          socket.value = null;
          stopHeartbeat();
          scheduleReconnect();
        }
      },
      Math.max(HEARTBEAT_MS, 10000),
    );
  };
  const stopWatchdog = () => {
    if (watchdogTimer) {
      clearInterval(watchdogTimer);
      watchdogTimer = null;
    }
  };

  // 断开连接
  const disconnect = () => {
    if (reconnectInterval.value) {
      clearTimeout(reconnectInterval.value);
      reconnectInterval.value = null;
    }

    if (socket.value) {
      socket.value.close(1000, "主动断开连接");
      socket.value = null;
    }

    isConnected.value = false;
    isConnecting.value = false;
    reconnectAttempts.value = 0;
    stopHeartbeat();
    stopWatchdog();
    
    // 清理所有监听器避免内存泄漏
    clearAllListeners();
    currentSubscriptions.clear();
    outboundQueue.length = 0;
  };

  // 加入房间
  const joinRoom = (roomId: string, roomType: "group" | "conversation") => {
    const key = `${roomType}:${roomId}`;
    if (socket.value && socket.value.readyState === WebSocket.OPEN) {
      if (currentSubscriptions.has(key)) return;
      currentSubscriptions.add(key);
    }
    if (roomType === "group") {
      sendMessage({
        type: "JoinGroup",
        data: { group_id: roomId },
      });
    } else {
      sendMessage({
        type: "JoinConversation",
        data: { conversation_id: roomId },
      });
    }
  };

  // 离开房间
  const leaveRoom = (roomId: string, roomType: "group" | "conversation") => {
    currentSubscriptions.delete(`${roomType}:${roomId}`);
    if (roomType === "group") {
      sendMessage({
        type: "LeaveGroup",
        data: { group_id: roomId },
      });
    } else {
      sendMessage({
        type: "LeaveConversation",
        data: { conversation_id: roomId },
      });
    }
  };

  // 发送心跳
  const sendHeartbeat = () => {
    sendMessage({ type: "Ping" });
  };

  // 启动心跳定时器（外部可调用但实际内部自管）
  const startHeartbeatLegacy = () => {
    startHeartbeat();
  };

  // 监听认证状态变化
  watch(
    () => authStore.token,
    (newToken, oldToken) => {
      if (newToken && !isConnected.value && !isConnecting.value) {
        // 有token且未连接时，自动连接
        connect();
      } else if (!newToken && oldToken && (isConnected.value || isConnecting.value)) {
        // 从有token变为无token时，优雅断开连接（避免错误提示）
        if (socket.value && socket.value.readyState === WebSocket.OPEN) {
          socket.value.close(1000, "用户注销");
        } else {
          disconnect();
        }
      } else if (newToken && oldToken && newToken !== oldToken) {
        // token变化（用户切换），先断开再重连
        if (socket.value) {
          socket.value.close(1000, "用户切换");
        }
        // 短暂延迟后重连，给旧连接时间完全关闭
        setTimeout(() => {
          if (newToken === authStore.token) { // 确保token仍然有效
            connect();
          }
        }, 100);
      }
    },
    { immediate: true },
  );

  // 页面可见性变化处理（仅在客户端）
  if (typeof document !== "undefined") {
    document.addEventListener("visibilitychange", () => {
      if (
        document.visibilityState === "visible" &&
        authStore.token &&
        !isConnected.value &&
        !isConnecting.value
      ) {
        connect();
      }
    });
  }

  // 组件卸载时清理（仅在组件上下文且客户端执行）
  if (typeof window !== "undefined" && getCurrentInstance()) {
    onUnmounted(() => {
      disconnect();
    });
  }

  return {
    // 状态
    isConnected: readonly(isConnected),
    isConnecting: readonly(isConnecting),

    // 方法
    connect,
    disconnect,
    sendMessage,
    joinRoom,
    leaveRoom,
    addEventListener,
    removeEventListener,
    startHeartbeat: startHeartbeatLegacy,
  };
};
