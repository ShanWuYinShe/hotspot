<template>
  <NButton
    :class="buttonClass"
    :disabled="disabled"
    :type="buttonType"
    :size="size"
    @click="handleClick"
  >
    <NIcon v-if="icon" :size="iconSize">
      <component :is="iconComponent" />
    </NIcon>
    <span v-if="title" class="button-text">{{ title }}</span>
    <span v-if="hotkey" class="button-hotkey">{{ hotkey }}</span>
  </NButton>
</template>

<script setup lang="ts">
import { NButton, NIcon } from 'naive-ui'
import { Add as Plus, Create as Edit, Trash as Delete, ShareOutline as Share, Settings as Setting, Person as User, ChatbubbleOutline as Message, LocationOutline as Location, Camera, Search } from '@vicons/ionicons5'

interface Props {
  /** 图标名称 */
  icon?: string
  /** 标题 */
  title?: string
  /** 快捷键 */
  hotkey?: string
  /** 是否禁用 */
  disabled?: boolean
  /** 是否激活 */
  active?: boolean
  /** 尺寸 */
  size?: 'large' | 'medium' | 'small'
  /** 按钮类型 */
  variant?: 'primary' | 'success' | 'warning' | 'danger' | 'info' | 'text'
}

const props = withDefaults(defineProps<Props>(), {
  title: '',
  size: 'medium',
  variant: 'primary',
  disabled: false,
  active: false
})

const emit = defineEmits<{
  click: []
}>()

// 图标组件映射
const iconComponent = computed(() => {
  const iconMap: Record<string, any> = {
    plus: Plus,
    edit: Edit,
    delete: Delete,
    share: Share,
    setting: Setting,
    user: User,
    message: Message,
    location: Location,
    camera: Camera,
    search: Search
  }
  
  return iconMap[props.icon || ''] || Plus
})

// 按钮类型映射
const buttonType = computed(() => {
  if (props.active) return 'primary'
  return props.variant as any
})

// 按钮样式类
const buttonClass = computed(() => ({
  'action-button': true,
  'action-button--active': props.active,
  'action-button--with-hotkey': !!props.hotkey
}))

// 图标尺寸
const iconSize = computed(() => {
  switch (props.size) {
    case 'small': return 14
    case 'large': return 20
    default: return 16
  }
})

// 方法
const handleClick = () => {
  emit('click')
}
</script>

<style scoped lang="scss">
.action-button {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  transition: all 0.2s ease;
  
  &.action-button--active {
    box-shadow: var(--shadow-md);
  }
  
  &.action-button--with-hotkey {
    position: relative;
  }
}

.button-text {
  flex: 1;
  min-width: 0;
  text-align: left;
}

.button-hotkey {
  margin-left: auto;
  padding: 0.125rem 0.25rem;
  background: var(--color-bg-tertiary);
  border-radius: var(--border-radius-xs);
  font-size: 0.75rem;
  color: var(--color-text-tertiary);
  font-weight: 500;
  opacity: 0.8;
}
</style> 