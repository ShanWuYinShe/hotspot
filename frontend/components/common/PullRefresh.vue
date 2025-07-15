<template>
  <div class="pull-refresh-container" @touchstart="handleTouchStart" @touchmove="handleTouchMove"
    @touchend="handleTouchEnd">
    <!-- 下拉刷新指示器 -->
    <div class="pull-refresh-indicator" :class="indicatorClass" :style="indicatorStyle">
      <div class="indicator-content">
        <NIcon v-if="status === 'pulling' || status === 'can-refresh'" class="indicator-icon"
          :class="{ 'rotate-180': status === 'can-refresh' }">
          <ArrowDown />
        </NIcon>

        <NIcon v-else-if="status === 'refreshing'" class="indicator-icon loading">
          <Loading />
        </NIcon>

        <NIcon v-else-if="status === 'success'" class="indicator-icon success">
          <Check />
        </NIcon>

        <span class="indicator-text">{{ indicatorText }}</span>
      </div>
    </div>

    <!-- 内容区域 -->
    <div class="pull-refresh-content" :style="contentStyle">
      <slot />
    </div>
  </div>
</template>

<script setup lang="ts">
import { NIcon } from 'naive-ui'
import { ref, computed } from 'vue'
import { ArrowDown, Reload as Loading, Checkmark as Check } from '@vicons/ionicons5'

interface PullRefreshProps {
  /** 是否禁用下拉刷新 */
  disabled?: boolean
  /** 触发刷新的下拉距离 */
  threshold?: number
  /** 最大下拉距离 */
  maxDistance?: number
  /** 刷新动画持续时间 */
  duration?: number
  /** 成功状态显示时间 */
  successDuration?: number
  /** 自定义提示文字 */
  pullingText?: string
  canRefreshText?: string
  refreshingText?: string
  successText?: string
}

const props = withDefaults(defineProps<PullRefreshProps>(), {
  disabled: false,
  threshold: 60,
  maxDistance: 80,
  duration: 300,
  successDuration: 1500,
  pullingText: '下拉刷新',
  canRefreshText: '释放刷新',
  refreshingText: '刷新中...',
  successText: '刷新成功'
})

const emit = defineEmits<{
  refresh: []
}>()

// 状态管理
type RefreshStatus = 'normal' | 'pulling' | 'can-refresh' | 'refreshing' | 'success'
const status = ref<RefreshStatus>('normal')
const distance = ref(0)
const startY = ref(0)
const currentY = ref(0)
const scrollTop = ref(0)

// 计算属性
const indicatorClass = computed(() => ({
  'pull-refresh-indicator--visible': distance.value > 0,
  'pull-refresh-indicator--can-refresh': status.value === 'can-refresh',
  'pull-refresh-indicator--refreshing': status.value === 'refreshing',
  'pull-refresh-indicator--success': status.value === 'success'
}))

const indicatorStyle = computed(() => ({
  height: `${Math.min(distance.value, props.maxDistance)}px`,
  transform: `translateY(${Math.min(distance.value, props.maxDistance) - props.maxDistance}px)`
}))

const contentStyle = computed(() => ({
  transform: `translateY(${Math.min(distance.value, props.maxDistance)}px)`,
  transition: status.value === 'refreshing' || status.value === 'success'
    ? `transform ${props.duration}ms ease`
    : 'none'
}))

const indicatorText = computed(() => {
  switch (status.value) {
    case 'pulling':
      return props.pullingText
    case 'can-refresh':
      return props.canRefreshText
    case 'refreshing':
      return props.refreshingText
    case 'success':
      return props.successText
    default:
      return ''
  }
})

// 触摸事件处理
const handleTouchStart = (event: TouchEvent) => {
  if (props.disabled) return

  startY.value = event.touches[0].clientY
  scrollTop.value = document.documentElement.scrollTop || document.body.scrollTop
}

const handleTouchMove = (event: TouchEvent) => {
  if (props.disabled) return

  currentY.value = event.touches[0].clientY
  const deltaY = currentY.value - startY.value

  // 只有在页面顶部且向下拉动时才触发
  if (scrollTop.value <= 0 && deltaY > 0) {
    event.preventDefault()

    // 计算下拉距离，添加阻尼效果
    distance.value = Math.min(deltaY * 0.6, props.maxDistance)

    // 更新状态
    if (distance.value >= props.threshold) {
      status.value = 'can-refresh'
    } else if (distance.value > 0) {
      status.value = 'pulling'
    } else {
      status.value = 'normal'
    }

    // 触觉反馈
    if (status.value === 'can-refresh' && distance.value >= props.threshold) {
      if ('vibrate' in navigator) {
        navigator.vibrate(10)
      }
    }
  }
}

const handleTouchEnd = () => {
  if (props.disabled) return

  if (status.value === 'can-refresh') {
    // 触发刷新
    status.value = 'refreshing'
    distance.value = props.threshold

    emit('refresh')

    // 触觉反馈
    if ('vibrate' in navigator) {
      navigator.vibrate(30)
    }
  } else {
    // 回弹到初始位置
    status.value = 'normal'
    distance.value = 0
  }
}

// 刷新完成
const finishRefresh = (success = true) => {
  if (success) {
    status.value = 'success'

    // 显示成功状态一段时间后隐藏
    setTimeout(() => {
      status.value = 'normal'
      distance.value = 0
    }, props.successDuration)
  } else {
    status.value = 'normal'
    distance.value = 0
  }
}

// 暴露方法给父组件
defineExpose({
  finishRefresh
})
</script>

<style scoped lang="scss">
.pull-refresh-container {
  position: relative;
  overflow: hidden;
}

.pull-refresh-indicator {
  position: absolute;
  top: 0;
  left: 0;
  right: 0;
  display: flex;
  align-items: flex-end;
  justify-content: center;
  background: var(--color-bg-primary);
  overflow: hidden;
  transition: opacity 0.2s ease;

  &--visible {
    opacity: 1;
  }

  &:not(&--visible) {
    opacity: 0;
  }
}

.indicator-content {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 0.5rem;
  padding: 1rem;
  color: var(--color-text-primary);
}

.indicator-icon {
  font-size: 1.5rem;
  transition: all 0.3s ease;

  &.rotate-180 {
    transform: rotate(180deg);
  }

  &.loading {
    animation: spin 1s linear infinite;
    color: var(--color-primary-500);
  }

  &.success {
    color: var(--color-success-500);
  }
}

.indicator-text {
  font-size: 0.875rem;
  font-weight: 500;
}

.pull-refresh-content {
  background: var(--color-bg-elevated);
  min-height: 100vh;
}

@keyframes spin {
  from {
    transform: rotate(0deg);
  }

  to {
    transform: rotate(360deg);
  }
}

// 移动端优化
@media (max-width: 768px) {
  .indicator-content {
    padding: 0.75rem;
  }

  .indicator-icon {
    font-size: 1.25rem;
  }

  .indicator-text {
    font-size: 0.8125rem;
  }
}
</style>
