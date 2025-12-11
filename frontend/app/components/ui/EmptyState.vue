<template>
  <div class="empty-state">
    <div class="empty-state-content">
      <!-- 图标/插图 -->
      <div class="empty-state-icon">
        <slot name="icon">
          <div class="default-icon-wrapper">
            <UIcon 
              :name="icon" 
              class="default-icon"
              :class="iconColorClass"
            />
          </div>
        </slot>
      </div>

      <!-- 标题 -->
      <h3 class="empty-state-title">
        <slot name="title">
          {{ title }}
        </slot>
      </h3>

      <!-- 描述 -->
      <p v-if="description || $slots.description" class="empty-state-description">
        <slot name="description">
          {{ description }}
        </slot>
      </p>

      <!-- 操作按钮 -->
      <div v-if="$slots.actions || action || actionText" class="empty-state-actions">
        <slot name="actions">
          <!-- 新 API：使用 action 对象 -->
          <UButton
            v-if="action"
            :to="action.to"
            :href="action.href"
            @click="handleActionClick(action.onClick)"
            :color="action.color || 'primary'"
            :variant="action.variant || 'solid'"
            :size="action.size || 'lg'"
          >
            <UIcon 
              v-if="action.icon" 
              :name="action.icon" 
              class="w-5 h-5 mr-2" 
            />
            {{ action.label }}
          </UButton>

          <!-- 旧 API 兼容：使用 actionText + @action 事件 -->
          <UButton
            v-else-if="actionText"
            @click="emit('action')"
            :color="actionColor || 'primary'"
            :variant="actionVariant || 'solid'"
            :size="actionSize || 'lg'"
          >
            <UIcon 
              v-if="actionIcon" 
              :name="actionIcon" 
              class="w-5 h-5 mr-2" 
            />
            {{ actionText }}
          </UButton>
        </slot>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
interface Action {
  label: string;
  icon?: string;
  to?: string;
  href?: string;
  onClick?: () => void;
  color?: string;
  variant?: string;
  size?: string;
}

interface Props {
  icon?: string;
  iconColor?: 'primary' | 'success' | 'warning' | 'error' | 'neutral';
  title: string;
  description?: string;
  action?: Action;
  // 旧 API 支持（为了兼容现有代码）
  actionText?: string;
  actionIcon?: string;
  actionVariant?: 'solid' | 'outline' | 'soft' | 'ghost';
  actionColor?: string;
  actionSize?: 'xs' | 'sm' | 'md' | 'lg' | 'xl';
}

interface Emits {
  action: [];
}

const props = withDefaults(defineProps<Props>(), {
  icon: 'i-heroicons-inbox',
  iconColor: 'neutral',
  actionVariant: 'solid',
  actionColor: 'primary',
  actionSize: 'lg',
});

const emit = defineEmits<Emits>();

// 处理操作点击（支持 action.onClick）
const handleActionClick = (onClick?: () => void) => {
  if (onClick) {
    onClick();
  }
};

// 图标颜色类
const iconColorClass = computed(() => {
  const colorMap = {
    primary: 'text-primary-500',
    success: 'text-success-500',
    warning: 'text-warning-500',
    error: 'text-error-500',
    neutral: 'text-neutral-400',
  };
  return colorMap[props.iconColor];
});
</script>

<style scoped>
.empty-state {
  display: flex;
  align-items: center;
  justify-content: center;
  min-height: 400px;
  padding: 3rem 1.5rem;
}

.empty-state-content {
  max-width: 28rem;
  text-align: center;
  animation: fade-in-up 0.5s cubic-bezier(0.4, 0, 0.2, 1);
}

.empty-state-icon {
  margin-bottom: 1.5rem;
  display: flex;
  justify-content: center;
}

.default-icon-wrapper {
  width: 5rem;
  height: 5rem;
  border-radius: 1.5rem;
  background: linear-gradient(135deg, rgba(0, 0, 0, 0.03) 0%, rgba(0, 0, 0, 0.06) 100%);
  display: flex;
  align-items: center;
  justify-content: center;
  transition: transform 0.3s cubic-bezier(0.4, 0, 0.2, 1);
}

.dark .default-icon-wrapper {
  background: linear-gradient(135deg, rgba(255, 255, 255, 0.05) 0%, rgba(255, 255, 255, 0.08) 100%);
}

.default-icon-wrapper:hover {
  transform: scale(1.05) rotate(5deg);
}

.default-icon {
  width: 2.5rem;
  height: 2.5rem;
}

.empty-state-title {
  font-size: 1.25rem;
  font-weight: 600;
  color: rgb(var(--color-neutral-900));
  margin-bottom: 0.75rem;
}

.dark .empty-state-title {
  color: rgb(var(--color-neutral-100));
}

.empty-state-description {
  font-size: 0.938rem;
  color: rgb(var(--color-neutral-600));
  margin-bottom: 1.5rem;
  line-height: 1.6;
}

.dark .empty-state-description {
  color: rgb(var(--color-neutral-400));
}

.empty-state-actions {
  display: flex;
  gap: 0.75rem;
  justify-content: center;
  flex-wrap: wrap;
}

@keyframes fade-in-up {
  from {
    opacity: 0;
    transform: translateY(20px);
  }
  to {
    opacity: 1;
    transform: translateY(0);
  }
}

/* 响应式优化 */
@media (max-width: 640px) {
  .empty-state {
    min-height: 300px;
    padding: 2rem 1rem;
  }

  .empty-state-content {
    max-width: 100%;
  }

  .default-icon-wrapper {
    width: 4rem;
    height: 4rem;
  }

  .default-icon {
    width: 2rem;
    height: 2rem;
  }

  .empty-state-title {
    font-size: 1.125rem;
  }

  .empty-state-description {
    font-size: 0.875rem;
  }
}

@media (prefers-reduced-motion: reduce) {
  .empty-state-content,
  .default-icon-wrapper {
    animation: none;
    transition: none;
  }
}
</style>

