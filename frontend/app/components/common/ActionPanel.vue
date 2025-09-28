<template>
  <UBanner
    :title="title"
    :description="description"
    :icon="icon"
    :actions="bannerActions"
    class="border-t"
  />
</template>

<script setup lang="ts">
export interface ActionItem {
  key: string
  label: string
  icon?: string
  color?: 'primary' | 'secondary' | 'success' | 'info' | 'warning' | 'error' | 'neutral'
  variant?: 'solid' | 'outline' | 'soft' | 'ghost' | 'link'
  size?: 'xs' | 'sm' | 'md' | 'lg' | 'xl'
  loading?: boolean
  disabled?: boolean
  action: () => void | Promise<void>
}

interface Props {
  title: string
  description?: string
  icon: string
  actions: ActionItem[]
}

const props = defineProps<Props>()

// 转换为 UBanner 需要的格式
const bannerActions = computed(() => 
  props.actions.map(action => ({
    label: action.label,
    icon: action.icon,
    color: action.color || 'primary',
    variant: action.variant || 'solid',
    size: action.size || 'sm',
    loading: action.loading,
    disabled: action.disabled,
    click: action.action
  }))
)
</script>
