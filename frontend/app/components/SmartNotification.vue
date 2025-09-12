<template>
  <div class="fixed top-4 right-4 z-50 space-y-3 max-w-sm">
    <TransitionGroup
      name="notification"
      tag="div"
      class="space-y-3"
    >
      <UCard
        v-for="notification in notifications"
        :key="notification.id"
        class="shadow-lg border-l-4 backdrop-blur-sm bg-white/95 dark:bg-neutral-800/95"
        :class="getNotificationClass(notification.type)"
      >
        <div class="flex items-start gap-3">
          <!-- 图标 -->
          <div class="flex-shrink-0 mt-0.5">
            <UIcon 
              :name="getNotificationIcon(notification.type)" 
              :class="getIconClass(notification.type)"
              class="w-5 h-5"
            />
          </div>
          
          <!-- 内容 -->
          <div class="flex-1 min-w-0">
            <h4 v-if="notification.title" class="font-medium text-sm text-neutral-900 dark:text-neutral-100 mb-1">
              {{ notification.title }}
            </h4>
            <p class="text-sm text-neutral-600 dark:text-neutral-400 leading-relaxed">
              {{ notification.message }}
            </p>
            
            <!-- 操作按钮 -->
            <div v-if="notification.actions && notification.actions.length > 0" class="flex gap-2 mt-3">
              <UButton
                v-for="action in notification.actions"
                :key="action.label"
                :variant="action.variant || 'soft'"
                :color="action.color || 'primary'"
                size="xs"
                @click="handleAction(notification, action)"
              >
                {{ action.label }}
              </UButton>
            </div>
          </div>
          
          <!-- 关闭按钮 -->
          <UButton
            variant="ghost"
            size="xs"
            color="neutral"
            @click="removeNotification(notification.id)"
            class="flex-shrink-0"
          >
            <UIcon name="i-heroicons-x-mark" class="w-4 h-4" />
          </UButton>
        </div>
        
        <!-- 进度条（自动消失） -->
        <div 
          v-if="notification.duration && notification.duration > 0"
          class="absolute bottom-0 left-0 h-1 bg-current opacity-30 transition-all duration-100 ease-linear"
          :style="{ width: getProgressWidth(notification) + '%' }"
        ></div>
      </UCard>
    </TransitionGroup>
  </div>
</template>

<script setup lang="ts">
interface NotificationAction {
  label: string;
  handler: () => void | Promise<void>;
  variant?: 'solid' | 'outline' | 'soft' | 'ghost';
  color?: 'primary' | 'secondary' | 'success' | 'info' | 'warning' | 'error' | 'neutral';
}

interface SmartNotification {
  id: string;
  type: 'success' | 'error' | 'warning' | 'info';
  title?: string;
  message: string;
  duration?: number; // 0 表示不自动消失
  actions?: NotificationAction[];
  persistent?: boolean;
  createdAt: number;
  startTime?: number;
}

// 通知列表
const notifications = ref<SmartNotification[]>([]);

// 通知样式映射
const getNotificationClass = (type: SmartNotification['type']) => {
  const classes = {
    success: 'border-l-green-500',
    error: 'border-l-red-500',
    warning: 'border-l-yellow-500',
    info: 'border-l-blue-500'
  };
  return classes[type];
};

const getNotificationIcon = (type: SmartNotification['type']) => {
  const icons = {
    success: 'i-heroicons-check-circle',
    error: 'i-heroicons-x-circle',
    warning: 'i-heroicons-exclamation-triangle',
    info: 'i-heroicons-information-circle'
  };
  return icons[type];
};

const getIconClass = (type: SmartNotification['type']) => {
  const classes = {
    success: 'text-green-500',
    error: 'text-red-500',
    warning: 'text-yellow-500',
    info: 'text-blue-500'
  };
  return classes[type];
};

// 添加通知
const addNotification = (notification: Omit<SmartNotification, 'id' | 'createdAt'>) => {
  const id = Date.now().toString();
  const newNotification: SmartNotification = {
    ...notification,
    id,
    createdAt: Date.now(),
    startTime: Date.now()
  };
  
  notifications.value.push(newNotification);
  
  // 自动移除（如果设置了持续时间）
  if (notification.duration && notification.duration > 0) {
    setTimeout(() => {
      removeNotification(id);
    }, notification.duration);
  }
  
  return id;
};

// 移除通知
const removeNotification = (id: string) => {
  const index = notifications.value.findIndex(n => n.id === id);
  if (index > -1) {
    notifications.value.splice(index, 1);
  }
};

// 清除所有通知
const clearAll = () => {
  notifications.value = [];
};

// 处理操作按钮点击
const handleAction = async (notification: SmartNotification, action: NotificationAction) => {
  try {
    await action.handler();
    // 执行操作后移除通知（除非是持久通知）
    if (!notification.persistent) {
      removeNotification(notification.id);
    }
  } catch (error) {
    console.error('Notification action failed:', error);
  }
};

// 计算进度条宽度
const getProgressWidth = (notification: SmartNotification) => {
  if (!notification.duration || !notification.startTime) return 100;
  
  const elapsed = Date.now() - notification.startTime;
  const progress = Math.max(0, 100 - (elapsed / notification.duration) * 100);
  return progress;
};

// 快捷方法
const success = (message: string, options?: Partial<SmartNotification>) => {
  return addNotification({
    type: 'success',
    message,
    duration: 3000,
    ...options
  });
};

const error = (message: string, options?: Partial<SmartNotification>) => {
  return addNotification({
    type: 'error',
    message,
    duration: 5000,
    ...options
  });
};

const warning = (message: string, options?: Partial<SmartNotification>) => {
  return addNotification({
    type: 'warning',
    message,
    duration: 4000,
    ...options
  });
};

const info = (message: string, options?: Partial<SmartNotification>) => {
  return addNotification({
    type: 'info',
    message,
    duration: 3000,
    ...options
  });
};

// 暴露方法
defineExpose({
  addNotification,
  removeNotification,
  clearAll,
  success,
  error,
  warning,
  info
});

// 定时更新进度条
let progressInterval: NodeJS.Timeout;

onMounted(() => {
  progressInterval = setInterval(() => {
    // 触发响应式更新
    notifications.value = [...notifications.value];
  }, 100);
});

onUnmounted(() => {
  if (progressInterval) {
    clearInterval(progressInterval);
  }
});
</script>

<style scoped>
/* 通知动画 */
.notification-enter-active,
.notification-leave-active {
  transition: all 0.3s ease;
}

.notification-enter-from {
  opacity: 0;
  transform: translateX(100%);
}

.notification-leave-to {
  opacity: 0;
  transform: translateX(100%);
}

.notification-move {
  transition: transform 0.3s ease;
}
</style>
</template>
</invoke>
