<template>
  <div class="action-bar" :class="actionBarClass">
    <div class="action-bar-header" v-if="statusLabel || statusValue">
      <div class="status-info">
        <span class="status-label">{{ statusLabel }}</span>
        <span class="status-value">{{ statusValue }}</span>
      </div>
    </div>
    <div class="action-bar-content">
      <slot />
    </div>
  </div>
</template>

<script setup lang="ts">
interface Props {
  /** 状态标签 */
  statusLabel?: string
  /** 状态值 */
  statusValue?: string | number
  /** 操作栏类型 */
  variant?: 'primary' | 'secondary' | 'success' | 'warning' | 'danger'
  /** 是否紧凑模式 */
  compact?: boolean
}

const props = withDefaults(defineProps<Props>(), {
  variant: 'primary',
  compact: false
})

const actionBarClass = computed(() => [
  `action-bar--${props.variant}`,
  {
    'action-bar--compact': props.compact
  }
])
</script>

<style scoped lang="scss">
.action-bar {
  background: var(--color-bg-secondary);
  border: 1px solid var(--color-border-primary);
  border-radius: var(--border-radius-lg);
  padding: 1rem;
  margin: 1rem 0;
  box-shadow: var(--shadow-sm);
  transition: all 0.3s ease;

  &:hover {
    border-color: var(--color-border-hover);
    box-shadow: var(--shadow-md);
  }

  &.action-bar--compact {
    padding: 0.5rem;
    margin: 0.5rem 0;
  }

  // 变体样式
  &.action-bar--primary {
    border-color: var(--color-primary-200);
    background: var(--color-primary-50);
  }

  &.action-bar--secondary {
    border-color: var(--color-gray-200);
    background: var(--color-gray-50);
  }

  &.action-bar--success {
    border-color: var(--color-success-200);
    background: var(--color-success-50);
  }

  &.action-bar--warning {
    border-color: var(--color-warning-200);
    background: var(--color-warning-50);
  }

  &.action-bar--danger {
    border-color: var(--color-danger-200);
    background: var(--color-danger-50);
  }
}

.action-bar-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 0.75rem;
  padding-bottom: 0.5rem;
  border-bottom: 1px solid var(--color-border-secondary);
}

.status-info {
  display: flex;
  align-items: center;
  gap: 0.5rem;
}

.status-label {
  color: var(--color-text-secondary);
  font-size: 0.875rem;
  font-weight: 500;
}

.status-value {
  color: var(--color-text-primary);
  font-size: 0.875rem;
  font-weight: 600;
  padding: 0.25rem 0.5rem;
  background: var(--color-bg-primary);
  border-radius: var(--border-radius-sm);
  border: 1px solid var(--color-border-secondary);
}

.action-bar-content {
  display: flex;
  flex-wrap: wrap;
  gap: 0.5rem;
  align-items: center;
  justify-content: center;
}

// 响应式设计
@media (max-width: 768px) {
  .action-bar {
    padding: 0.75rem;
    margin: 0.75rem 0;
  }

  .action-bar-header {
    margin-bottom: 0.5rem;
  }

  .action-bar-content {
    gap: 0.375rem;
  }
}
</style> 