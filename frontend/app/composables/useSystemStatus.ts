/**
 * 系统状态指示器 Composable
 * 统一管理 API、WebSocket、网络等状态，为导航栏状态指示器提供数据
 */

export interface StatusItem {
  id: string;
  icon: string;
  label: string;
  status: "good" | "warning" | "error" | "loading" | "disabled";
  tooltip: string;
  count?: number;
  lastUpdate?: Date;
  details?: string;
}

export const useSystemStatus = () => {
  const webSocket = useWebSocket();
  const authStore = useAuthStore();
  const messagesStore = useMessagesStore();
  const settingsStore = useSettingsStore();

  // 简化的状态管理
  const lastApiCheck = ref<Date | null>(null);
  const apiStatus = ref<"good" | "warning" | "error">("good");

  // API服务器状态
  const getApiStatus = (): StatusItem => {
    return {
      id: "api",
      icon:
        apiStatus.value === "good"
          ? "i-heroicons-check-circle"
          : apiStatus.value === "warning"
            ? "i-heroicons-exclamation-triangle"
            : "i-heroicons-x-circle",
      label: "API",
      status: apiStatus.value,
      tooltip:
        apiStatus.value === "good"
          ? "服务器连接正常"
          : apiStatus.value === "warning"
            ? "服务器响应较慢"
            : "服务器连接异常",
    };
  };

  // WebSocket状态
  const getWebSocketStatus = (): StatusItem => {
    const connected = webSocket.isConnected.value;
    const connecting = webSocket.isConnecting.value;

    let status: StatusItem["status"];
    let tooltip: string;
    let icon: string;

    if (connected) {
      status = "good";
      tooltip = "实时连接正常";
      icon = "i-heroicons-wifi";
    } else if (connecting) {
      status = "loading";
      tooltip = "正在连接...";
      icon = "i-heroicons-arrow-path";
    } else if (authStore.isAuthenticated) {
      status = "error";
      tooltip = "实时连接断开";
      icon = "i-heroicons-wifi-slash";
    } else {
      status = "disabled";
      tooltip = "未登录";
      icon = "i-heroicons-minus-circle";
    }

    return {
      id: "websocket",
      icon,
      label: "实时",
      status,
      tooltip,
    };
  };

  // 网络状态
  const getNetworkStatus = (): StatusItem => {
    const isOnline = typeof navigator !== "undefined" ? navigator.onLine : true;

    return {
      id: "network",
      icon: isOnline ? "i-heroicons-signal" : "i-heroicons-signal-slash",
      label: "网络",
      status: isOnline ? "good" : "error",
      tooltip: isOnline ? "网络连接正常" : "网络断开",
    };
  };

  // 未读消息状态
  const getUnreadStatus = (): StatusItem => {
    const count = messagesStore.totalUnreadCount;

    return {
      id: "unread",
      icon: "i-heroicons-envelope",
      label: "消息",
      status: count > 0 ? "warning" : "good",
      tooltip: count > 0 ? `${count} 条未读消息` : "暂无未读消息",
      count: count > 0 ? count : undefined,
    };
  };

  // 权限状态
  const getPermissionStatus = (): StatusItem => {
    const hasNotificationPerm =
      typeof Notification !== "undefined" &&
      Notification.permission === "granted";

    const status = hasNotificationPerm ? "good" : "warning";

    return {
      id: "permissions",
      icon:
        status === "good"
          ? "i-heroicons-shield-check"
          : "i-heroicons-shield-exclamation",
      label: "权限",
      status,
      tooltip: status === "good" ? "通知权限已授予" : "需要通知权限",
    };
  };

  // 获取所有状态（只返回关键状态）
  const getAllStatus = computed((): StatusItem[] => {
    const statuses = [];

    // API状态
    const api = getApiStatus();
    if (api.status !== "good") statuses.push(api);

    // WebSocket状态
    const ws = getWebSocketStatus();
    if (ws.status !== "good" && ws.status !== "disabled") statuses.push(ws);

    // 网络状态
    const network = getNetworkStatus();
    if (network.status !== "good") statuses.push(network);

    // 未读消息状态
    const unread = getUnreadStatus();
    if (unread.status === "warning") statuses.push(unread);

    // 权限状态
    const permissions = getPermissionStatus();
    if (permissions.status !== "good") statuses.push(permissions);

    return statuses;
  });

  // 获取需要注意的状态（有问题的状态）
  const getAlertStatus = computed((): StatusItem[] => {
    return getAllStatus.value.filter(
      (item) => item.status === "error" || item.status === "warning",
    );
  });

  // 更新API状态
  const updateApiStatus = (status: "good" | "warning" | "error") => {
    apiStatus.value = status;
    lastApiCheck.value = new Date();
  };

  // 监听网络状态变化（仅注册一次，避免重复注册导致的内存增长）
  // 使用模块级注册标记
  // eslint-disable-next-line @typescript-eslint/no-unused-vars
  const _ensureNetworkListeners = (() => {
    let registered = (globalThis as any).__hs_network_listeners__ || false;
    if (typeof window !== "undefined" && !registered) {
      const onlineHandler = () => updateApiStatus("good");
      const offlineHandler = () => updateApiStatus("error");
      window.addEventListener("online", onlineHandler);
      window.addEventListener("offline", offlineHandler);
      (globalThis as any).__hs_network_listeners__ = true;
    }
  })();

  return {
    // 状态获取
    getAllStatus,
    getAlertStatus,

    // 状态计算
    getApiStatus: () => getApiStatus(),
    getWebSocketStatus: () => getWebSocketStatus(),
    getUnreadStatus: () => getUnreadStatus(),
    getPermissionStatus: () => getPermissionStatus(),

    // 状态更新
    updateApiStatus,

    // 工具方法
    hasErrors: computed(() =>
      getAlertStatus.value.some((s) => s.status === "error"),
    ),
    hasWarnings: computed(() =>
      getAlertStatus.value.some((s) => s.status === "warning"),
    ),
    totalAlerts: computed(() => getAlertStatus.value.length),
  };
};
