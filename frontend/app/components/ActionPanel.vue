<template>
  <div class="bg-white dark:bg-gray-900 border-t border-gray-200 dark:border-gray-700 p-4">
    <div class="flex items-center justify-between max-w-7xl mx-auto">
      <!-- 左侧信息 -->
      <div class="flex items-center gap-3">
        <UIcon :name="icon" class="w-5 h-5 text-primary" />
        <div>
          <h3 class="font-medium text-gray-900 dark:text-white">{{ title }}</h3>
          <p v-if="description" class="text-sm text-gray-500 dark:text-gray-400">{{ description }}</p>
        </div>
      </div>
      
      <!-- 右侧操作按钮 -->
      <div class="flex items-center gap-2">
        <UButton
          v-for="action in actions"
          :key="action.key"
          :color="action.color || 'primary'"
          :variant="action.variant || 'solid'"
          :size="action.size || 'sm'"
          :loading="action.loading"
          :disabled="action.disabled"
          @click="handleAction(action)"
        >
          <UIcon v-if="action.icon" :name="action.icon" class="w-4 h-4 mr-1" />
          {{ action.label }}
        </UButton>
      </div>
    </div>
  </div>
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

const { executeFeature } = useFeatureStatus()

const handleAction = async (action: ActionItem) => {
  try {
    await action.action()
  } catch (error) {
    console.error(`操作 ${action.label} 失败:`, error)
  }
}
</script>
