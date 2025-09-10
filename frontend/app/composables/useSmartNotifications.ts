/**
 * 智能通知管理器
 * 根据用户设置和系统状态，智能决定通知方式
 * 避免重复提示，提供最佳用户体验
 */

export interface NotificationOptions {
  title: string;
  description?: string;
  type: 'success' | 'error' | 'warning' | 'info';
  priority: 'low' | 'medium' | 'high' | 'critical';
  category: 'network' | 'auth' | 'message' | 'system' | 'business';
  dedupeKey?: string; // 去重标识
  duration?: number; // 显示时长(ms)
  actions?: Array<{
    label: string;
    onClick: () => void;
  }>;
}

interface NotificationHistory {
  key: string;
  timestamp: number;
  count: number;
}

export const useSmartNotifications = () => {
  const toast = useSmartToast();
  const settingsStore = useSettingsStore();
  const systemStatus = useSystemStatus();

  // 通知历史记录（用于去重）
  const notificationHistory = ref<Map<string, NotificationHistory>>(new Map());

  // 去重时间窗口（毫秒）
  const DEDUPE_WINDOW = {
    network: 30000, // 30秒
    auth: 60000,    // 1分钟
    message: 5000,  // 5秒
    system: 15000,  // 15秒
    business: 10000 // 10秒
  };

  // 检查是否应该显示通知
  const shouldShowNotification = (options: NotificationOptions): boolean => {
    const settings = settingsStore.settings;
    if (!settings) return true; // 如果没有设置，默认显示

    // 检查用户设置
    switch (options.category) {
      case 'auth':
        // 认证错误始终显示
        return true;

      case 'message':
        if (options.type === 'info') {
          // 新消息通知
          return options.priority === 'high' ?
            settings.notifications.message_notifications ?? true :
            settings.notifications.group_notifications ?? true;
        }
        return settings.notifications.message_notifications ?? true;

      case 'network':
        // 网络错误通过状态指示器显示，不用Toast
        return false;

      case 'system':
        return settings.notifications.system_notification_toast ?? true;

      case 'business':
        return options.type === 'error' ?
          (settings.notifications.error_notifications ?? true) :
          (settings.notifications.success_notifications ?? true);

      default:
        return true;
    }
  };

  // 检查是否需要去重
  const shouldDedupe = (options: NotificationOptions): boolean => {
    if (!options.dedupeKey) return false;

    const history = notificationHistory.value.get(options.dedupeKey);
    if (!history) return false;

    const windowTime = DEDUPE_WINDOW[options.category] || 10000;
    const now = Date.now();

    return (now - history.timestamp) < windowTime;
  };

  // 更新通知历史
  const updateHistory = (options: NotificationOptions) => {
    if (!options.dedupeKey) return;

    const existing = notificationHistory.value.get(options.dedupeKey);
    const now = Date.now();

    if (existing) {
      existing.count++;
      existing.timestamp = now;
    } else {
      notificationHistory.value.set(options.dedupeKey, {
        key: options.dedupeKey,
        timestamp: now,
        count: 1
      });
    }
  };

  // 清理过期的历史记录
  const cleanupHistory = () => {
    const now = Date.now();
    const maxAge = Math.max(...Object.values(DEDUPE_WINDOW));

    for (const [key, history] of notificationHistory.value.entries()) {
      if (now - history.timestamp > maxAge) {
        notificationHistory.value.delete(key);
      }
    }
  };

  // 主要的通知方法
  const notify = (options: NotificationOptions) => {
    // 清理过期历史
    cleanupHistory();

    // 检查是否应该显示
    if (!shouldShowNotification(options)) {
      return;
    }

    // 检查去重
    if (shouldDedupe(options)) {
      // 如果是重复通知，更新计数但不显示
      updateHistory(options);
      return;
    }

    // 准备Toast选项
    const toastOptions: any = {
      title: options.title,
      description: options.description,
      color: getToastColor(options.type),
      timeout: options.duration || getDefaultDuration(options.priority),
    };

    // 添加重复计数信息
    const history = notificationHistory.value.get(options.dedupeKey || '');
    if (history && history.count > 1) {
      toastOptions.description = `${toastOptions.description} (${history.count}次)`;
    }

    // 添加操作按钮
    if (options.actions && options.actions.length > 0) {
      toastOptions.actions = options.actions;
    }

    // 显示通知
    toast.add(toastOptions);

    // 更新历史
    updateHistory(options);
  };

  // 获取Toast颜色
  const getToastColor = (type: NotificationOptions['type']) => {
    switch (type) {
      case 'success': return 'success';
      case 'error': return 'error';
      case 'warning': return 'warning';
      case 'info': return 'info';
      default: return 'info';
    }
  };

  // 获取默认显示时长
  const getDefaultDuration = (priority: NotificationOptions['priority']) => {
    switch (priority) {
      case 'critical': return 0; // 不自动消失
      case 'high': return 8000;
      case 'medium': return 5000;
      case 'low': return 3000;
      default: return 5000;
    }
  };

  // 快捷方法
  const success = (title: string, description?: string, options?: Partial<NotificationOptions>) => {
    notify({
      title,
      description,
      type: 'success',
      priority: 'medium',
      category: 'business',
      ...options
    });
  };

  const error = (title: string, description?: string, options?: Partial<NotificationOptions>) => {
    notify({
      title,
      description,
      type: 'error',
      priority: 'high',
      category: 'system',
      ...options
    });
  };

  const warning = (title: string, description?: string, options?: Partial<NotificationOptions>) => {
    notify({
      title,
      description,
      type: 'warning',
      priority: 'medium',
      category: 'business',
      ...options
    });
  };

  const info = (title: string, description?: string, options?: Partial<NotificationOptions>) => {
    notify({
      title,
      description,
      type: 'info',
      priority: 'low',
      category: 'system',
      ...options
    });
  };

  // 特殊的消息通知
  const messageReceived = (senderName: string, content: string, isGroup: boolean = false) => {
    const settings = settingsStore.settings;
    if (!settings) return;

    const shouldShow = isGroup ?
      (settings.notifications.group_notifications ?? true) :
      (settings.notifications.message_notifications ?? true);

    if (shouldShow) {
      notify({
        title: senderName,
        description: content.slice(0, 50) + (content.length > 50 ? '...' : ''),
        type: 'info',
        priority: isGroup ? 'medium' : 'high',
        category: 'message',
        dedupeKey: `message-${senderName}-${isGroup ? 'group' : 'direct'}`,
        actions: [
          {
            label: '查看',
            onClick: () => {
              // 这里应该导航到对应的聊天页面
              navigateTo(`/chat/${isGroup ? 'group' : 'user'}/${senderName}`);
            }
          }
        ]
      });
    }
  };

  // 网络错误（不显示Toast，但可以记录）
  const networkError = (message: string) => {
    systemStatus.updateApiStatus('error');
    // 网络错误通过状态指示器显示，不用Toast
    console.warn('Network error:', message);
  };

  // 认证错误（强制显示）
  const authError = (message: string = '需要重新登录') => {
    notify({
      title: '认证失败',
      description: message,
      type: 'error',
      priority: 'critical',
      category: 'auth',
      dedupeKey: 'auth-failed',
      actions: [
        {
          label: '去登录',
          onClick: () => navigateTo('/auth')
        }
      ]
    });
  };

  // 业务错误
  const businessError = (title: string, description?: string, dedupeKey?: string) => {
    notify({
      title,
      description,
      type: 'error',
      priority: 'medium',
      category: 'business',
      dedupeKey
    });
  };

  return {
    // 核心方法
    notify,

    // 快捷方法
    success,
    error,
    warning,
    info,

    // 专用方法
    messageReceived,
    networkError,
    authError,
    businessError,

    // 工具方法
    cleanupHistory,

    // 状态
    historySize: computed(() => notificationHistory.value.size)
  };
};
