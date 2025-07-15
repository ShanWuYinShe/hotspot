<template>
  <Teleport to="body">
    <div class="fab-container" :class="containerClass">
      <!-- 主按钮 -->
      <NButton
        :type="type"
        :size="size"
        :loading="loading"
        class="fab-main"
        :class="mainButtonClass"
        @click="handleMainClick"
        @touchstart="handleTouchStart"
        @touchend="handleTouchEnd"
        circle
        :style="mainButtonStyle"
      >
        <template #icon>
          <NIcon v-if="!loading" :size="iconSize">
            <component :is="currentIcon" />
          </NIcon>
        </template>
        <span v-if="text" class="fab-text">{{ text }}</span>
      </NButton>
      
      <!-- 子按钮组 -->
      <Transition name="fab-actions">
        <div v-if="expanded && actions.length > 0" class="fab-actions">
          <div
            v-for="(action, index) in actions"
            :key="action.key || index"
            class="fab-action"
            :style="getActionStyle(index)"
          >
            <NButton
              :type="action.type || 'default'"
              :size="actionSize"
              :disabled="action.disabled"
              class="fab-action-btn"
              @click="handleActionClick(action, index)"
              circle
            >
              <template #icon>
                <NIcon :size="actionIconSize">
                  <component :is="action.icon" />
                </NIcon>
              </template>
            </NButton>
            
            <!-- 标签 -->
            <div v-if="action.label" class="fab-action-label">
              {{ action.label }}
            </div>
          </div>
        </div>
      </Transition>
      
      <!-- 遮罩层 -->
      <Transition name="fab-overlay">
        <div 
          v-if="expanded && overlay"
          class="fab-overlay"
          @click="collapse"
        ></div>
      </Transition>
    </div>
  </Teleport>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, onBeforeUnmount, watch } from 'vue'
import { NButton, NIcon } from 'naive-ui'
import { Add as Plus, Close } from '@vicons/ionicons5'

interface FabAction {
  key?: string | number
  icon: any
  label?: string
  type?: 'primary' | 'success' | 'warning' | 'error' | 'info' | 'default'
  disabled?: boolean
  onClick?: () => void
}

interface FloatingActionButtonProps {
  /** 按钮类型 */
  type?: 'primary' | 'success' | 'warning' | 'error' | 'info' | 'default'
  /** 按钮尺寸 */
  size?: 'large' | 'medium' | 'small'
  /** 主图标 */
  icon?: any
  /** 展开时的图标 */
  expandIcon?: any
  /** 按钮文字 */
  text?: string
  /** 是否加载中 */
  loading?: boolean
  /** 是否展开 */
  modelValue?: boolean
  /** 位置 */
  position?: 'bottom-right' | 'bottom-left' | 'top-right' | 'top-left' | 'bottom-center'
  /** 距离边缘的距离 */
  offset?: number
  /** 子按钮列表 */
  actions?: FabAction[]
  /** 是否显示遮罩 */
  overlay?: boolean
  /** 方向 */
  direction?: 'up' | 'down' | 'left' | 'right' | 'circular'
  /** 是否可拖拽 */
  draggable?: boolean
  /** 自动收起时间(ms) */
  autoCollapse?: number
}

const props = withDefaults(defineProps<FloatingActionButtonProps>(), {
  type: 'primary',
  size: 'medium',
  icon: Plus,
  expandIcon: Close,
  modelValue: false,
  position: 'bottom-right',
  offset: 20,
  actions: () => [],
  overlay: true,
  direction: 'up',
  draggable: false,
  autoCollapse: 0
})

const emit = defineEmits<{
  'update:modelValue': [value: boolean]
  'click': [event: MouseEvent]
  'action-click': [action: FabAction, index: number]
}>()

// 状态管理
const expanded = ref(props.modelValue)
const isDragging = ref(false)
const position = ref({ x: 0, y: 0 })
const dragStart = ref({ x: 0, y: 0 })

// 计算属性
const currentIcon = computed(() => {
  return expanded.value ? props.expandIcon : props.icon
})

const containerClass = computed(() => ({
  'fab-container--dragging': isDragging.value,
  [`fab-container--${props.position}`]: !props.draggable,
  'fab-container--draggable': props.draggable
}))

const mainButtonClass = computed(() => ({
  'fab-main--expanded': expanded.value,
  'fab-main--draggable': props.draggable
}))

const mainButtonStyle = computed(() => {
  const sizeMap = {
    large: { width: '64px', height: '64px' },
    medium: { width: '56px', height: '56px' },
    small: { width: '48px', height: '48px' }
  }
  return sizeMap[props.size]
})

const actionSize = computed(() => {
  return props.size === 'large' ? 'medium' : 'small'
})

const iconSize = computed(() => {
  const sizeMap = {
    large: 24,
    medium: 20,
    small: 16
  }
  return sizeMap[props.size]
})

const actionIconSize = computed(() => {
  const sizeMap = {
    large: 20,
    medium: 16,
    small: 14
  }
  return sizeMap[props.size]
})

// 同步展开状态
watch(() => props.modelValue, (val) => {
  expanded.value = val
})

watch(expanded, (val) => {
  emit('update:modelValue', val)
})

// 主按钮点击
const handleMainClick = (event: MouseEvent) => {
  if (props.actions.length > 0) {
    toggle()
  } else {
    emit('click', event)
  }
  
  // 触觉反馈
  if ('vibrate' in navigator) {
    navigator.vibrate(10)
  }
}

// 子按钮点击
const handleActionClick = (action: FabAction, index: number) => {
  if (action.disabled) return
  
  action.onClick?.()
  emit('action-click', action, index)
  
  // 点击后自动收起
  collapse()
  
  // 触觉反馈
  if ('vibrate' in navigator) {
    navigator.vibrate(15)
  }
}

// 展开/收起控制
const toggle = () => {
  expanded.value = !expanded.value
}

const expand = () => {
  expanded.value = true
}

const collapse = () => {
  expanded.value = false
}

// 获取子按钮样式
const getActionStyle = (index: number) => {
  const totalActions = props.actions.length
  const spacing = props.size === 'large' ? 70 : props.size === 'medium' ? 60 : 50
  
  let transform = ''
  
  switch (props.direction) {
    case 'up':
      transform = `translateY(-${(index + 1) * spacing}px)`
      break
    case 'down':
      transform = `translateY(${(index + 1) * spacing}px)`
      break
    case 'left':
      transform = `translateX(-${(index + 1) * spacing}px)`
      break
    case 'right':
      transform = `translateX(${(index + 1) * spacing}px)`
      break
    case 'circular':
      const angle = (index * 60) - 90 // 从顶部开始，每个按钮间隔60度
      const radius = props.size === 'large' ? 90 : props.size === 'medium' ? 80 : 70
      const x = Math.cos(angle * Math.PI / 180) * radius
      const y = Math.sin(angle * Math.PI / 180) * radius
      transform = `translate(${x}px, ${y}px)`
      break
  }
  
  return {
    transform,
    transitionDelay: `${index * 50}ms`
  }
}

// 拖拽功能
const handleTouchStart = (event: TouchEvent) => {
  if (!props.draggable) return
  
  isDragging.value = true
  const touch = event.touches[0]
  dragStart.value = {
    x: touch.clientX - position.value.x,
    y: touch.clientY - position.value.y
  }
}

const handleTouchMove = (event: TouchEvent) => {
  if (!props.draggable || !isDragging.value) return
  
  event.preventDefault()
  const touch = event.touches[0]
  position.value = {
    x: touch.clientX - dragStart.value.x,
    y: touch.clientY - dragStart.value.y
  }
}

const handleTouchEnd = () => {
  if (!props.draggable) return
  
  isDragging.value = false
  
  // 边缘吸附
  const { innerWidth, innerHeight } = window
  const threshold = 50
  const buttonSize = props.size === 'large' ? 64 : props.size === 'medium' ? 56 : 48
  
  if (position.value.x < threshold) {
    position.value.x = props.offset
  } else if (position.value.x > innerWidth - threshold) {
    position.value.x = innerWidth - props.offset - buttonSize
  }
  
  if (position.value.y < threshold) {
    position.value.y = props.offset
  } else if (position.value.y > innerHeight - threshold) {
    position.value.y = innerHeight - props.offset - buttonSize
  }
}

// 自动收起
let autoCollapseTimer: ReturnType<typeof setTimeout> | null = null

watch(expanded, (val) => {
  if (props.autoCollapse > 0) {
    if (val) {
      autoCollapseTimer = setTimeout(() => {
        collapse()
      }, props.autoCollapse)
    } else if (autoCollapseTimer) {
      clearTimeout(autoCollapseTimer)
      autoCollapseTimer = null
    }
  }
})

// 拖拽事件监听
onMounted(() => {
  if (props.draggable) {
    document.addEventListener('touchmove', handleTouchMove, { passive: false })
    document.addEventListener('touchend', handleTouchEnd)
  }
})

onBeforeUnmount(() => {
  if (props.draggable) {
    document.removeEventListener('touchmove', handleTouchMove)
    document.removeEventListener('touchend', handleTouchEnd)
  }
  
  if (autoCollapseTimer) {
    clearTimeout(autoCollapseTimer)
  }
})

// 暴露方法
defineExpose({
  expand,
  collapse,
  toggle
})
</script>

<style scoped lang="scss">
.fab-container {
  position: fixed;
  z-index: var(--z-index-toast, 2000);
  
  &--bottom-right {
    bottom: v-bind('props.offset + "px"');
    right: v-bind('props.offset + "px"');
  }
  
  &--bottom-left {
    bottom: v-bind('props.offset + "px"');
    left: v-bind('props.offset + "px"');
  }
  
  &--top-right {
    top: v-bind('props.offset + "px"');
    right: v-bind('props.offset + "px"');
  }
  
  &--top-left {
    top: v-bind('props.offset + "px"');
    left: v-bind('props.offset + "px"');
  }
  
  &--bottom-center {
    bottom: v-bind('props.offset + "px"');
    left: 50%;
    transform: translateX(-50%);
  }
  
  &--draggable {
    transform: v-bind('`translate(${position.x}px, ${position.y}px)`');
    transition: transform 0.3s ease;
  }
  
  &--dragging {
    transition: none;
  }
}

.fab-main {
  box-shadow: var(--shadow-lg, 0 4px 12px rgba(0, 0, 0, 0.15), 0 2px 6px rgba(0, 0, 0, 0.1));
  transition: all 0.3s cubic-bezier(0.4, 0, 0.2, 1);
  position: relative;
  overflow: hidden;
  
  &:hover {
    box-shadow: var(--shadow-xl, 0 6px 16px rgba(0, 0, 0, 0.2), 0 3px 8px rgba(0, 0, 0, 0.15));
    transform: translateY(-2px);
  }
  
  &:active {
    transform: scale(0.95);
  }
  
  &--expanded {
    transform: rotate(45deg);
    
    &:hover {
      transform: rotate(45deg) translateY(-2px);
    }
  }
  
  &--draggable {
    cursor: move;
    user-select: none;
    -webkit-user-select: none;
    -moz-user-select: none;
    -ms-user-select: none;
  }
}

.fab-text {
  margin-left: 0.5rem;
  font-size: 0.875rem;
  font-weight: 500;
  color: inherit;
}

.fab-actions {
  position: absolute;
  bottom: 0;
  right: 0;
}

.fab-action {
  position: absolute;
  display: flex;
  align-items: center;
  gap: 0.75rem;
  transition: all 0.3s cubic-bezier(0.4, 0, 0.2, 1);
}

.fab-action-btn {
  box-shadow: var(--shadow-md, 0 2px 8px rgba(0, 0, 0, 0.15));
  
  &:hover {
    transform: scale(1.1);
    box-shadow: var(--shadow-lg, 0 4px 12px rgba(0, 0, 0, 0.2));
  }
}

.fab-action-label {
  background: var(--color-bg-inverse, rgba(0, 0, 0, 0.8));
  color: var(--color-text-inverse, white);
  padding: 0.5rem 0.75rem;
  border-radius: var(--radius-base, 4px);
  font-size: 0.8125rem;
  white-space: nowrap;
  box-shadow: var(--shadow-md, 0 2px 8px rgba(0, 0, 0, 0.15));
}

.fab-overlay {
  position: fixed;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  background: rgba(0, 0, 0, 0.3);
  z-index: -1;
}

// 动画
.fab-actions-enter-active,
.fab-actions-leave-active {
  transition: all 0.3s ease;
}

.fab-actions-enter-from,
.fab-actions-leave-to {
  opacity: 0;
  transform: scale(0.8);
}

.fab-overlay-enter-active,
.fab-overlay-leave-active {
  transition: opacity 0.3s ease;
}

.fab-overlay-enter-from,
.fab-overlay-leave-to {
  opacity: 0;
}

// 移动端适配
@media (max-width: 768px) {
  .fab-container {
    &--bottom-right,
    &--bottom-left {
      bottom: calc(v-bind('props.offset + "px"') + env(safe-area-inset-bottom));
    }
  }
  
  .fab-action-label {
    font-size: 0.75rem;
    padding: 0.375rem 0.5rem;
  }
}

// 深色模式适配
@media (prefers-color-scheme: dark) {
  .fab-action-label {
    background: var(--color-bg-elevated, rgba(255, 255, 255, 0.9));
    color: var(--color-text-primary, #1f2937);
  }
}
</style>
