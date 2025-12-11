<template>
  <UModal>
    <template #body>
      <div class="p-6">
        <!-- 标题 -->
        <div class="flex items-center gap-3 mb-4">
          <UIcon 
            :name="iconName" 
            :class="iconClass"
            class="w-6 h-6" 
          />
          <h3 class="text-lg font-semibold text-neutral-900 dark:text-neutral-100">
            {{ title }}
          </h3>
        </div>

        <!-- 消息内容 -->
        <p class="text-neutral-600 dark:text-neutral-400 mb-6">
          {{ message }}
        </p>

        <!-- 操作按钮 -->
        <div class="flex justify-end gap-3">
          <UButton
            v-if="cancelText"
            variant="outline"
            color="neutral"
            @click="handleCancel"
          >
            {{ cancelText }}
          </UButton>
          <UButton
            :color="confirmColor"
            @click="handleConfirm"
          >
            {{ confirmText }}
          </UButton>
        </div>
      </div>
    </template>
  </UModal>
</template>

<script setup lang="ts">
interface Props {
  title: string
  message: string
  confirmText?: string
  cancelText?: string
  confirmColor?: 'primary' | 'error' | 'warning' | 'success'
  onConfirm?: () => void
  onCancel?: () => void
  onClose?: () => void
}

const props = withDefaults(defineProps<Props>(), {
  confirmText: '确认',
  cancelText: '取消',
  confirmColor: 'primary'
})

// 图标配置
const iconName = computed(() => {
  switch (props.confirmColor) {
    case 'error':
      return 'i-heroicons-exclamation-triangle'
    case 'warning':
      return 'i-heroicons-exclamation-triangle'
    case 'success':
      return 'i-heroicons-check-circle'
    default:
      return 'i-heroicons-question-mark-circle'
  }
})

const iconClass = computed(() => {
  switch (props.confirmColor) {
    case 'error':
      return 'text-red-500'
    case 'warning':
      return 'text-amber-500'
    case 'success':
      return 'text-green-500'
    default:
      return 'text-blue-500'
  }
})

const handleConfirm = () => {
  props.onConfirm?.()
}

const handleCancel = () => {
  props.onCancel?.()
}

// 监听 ESC 键关闭
onMounted(() => {
  const handleKeydown = (event: KeyboardEvent) => {
    if (event.key === 'Escape') {
      handleCancel()
    }
  }
  
  document.addEventListener('keydown', handleKeydown)
  
  onUnmounted(() => {
    document.removeEventListener('keydown', handleKeydown)
  })
})
</script>
