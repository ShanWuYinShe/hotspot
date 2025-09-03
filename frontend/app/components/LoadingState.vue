<template>
  <div class="flex flex-col items-center justify-center py-12 px-4">
    <!-- 加载动画 -->
    <div class="relative mb-4">
      <div class="w-12 h-12 border-4 border-primary/20 border-t-primary rounded-full animate-spin"></div>
      <UIcon v-if="icon" :name="icon" class="absolute inset-0 m-auto w-6 h-6 text-primary/60" />
    </div>
    
    <!-- 加载文本 -->
    <h3 class="text-lg font-medium text-neutral-900 dark:text-neutral-100 mb-2">
      {{ title || '加载中...' }}
    </h3>
    
    <!-- 描述文本 -->
    <p v-if="description" class="text-sm text-neutral-600 dark:text-neutral-400 text-center max-w-sm">
      {{ description }}
    </p>
    
    <!-- 进度条（可选） -->
    <div v-if="showProgress && progress !== undefined" class="w-full max-w-xs mt-4">
      <UProgress :value="progress" :max="100" size="sm" />
      <p class="text-xs text-neutral-500 text-center mt-1">{{ Math.round(progress) }}%</p>
    </div>
    
    <!-- 取消按钮（可选） -->
    <UButton 
      v-if="showCancel" 
      @click="$emit('cancel')" 
      variant="ghost" 
      size="sm" 
      class="mt-4"
    >
      取消
    </UButton>
  </div>
</template>

<script setup lang="ts">
interface Props {
  title?: string
  description?: string
  icon?: string
  showProgress?: boolean
  progress?: number
  showCancel?: boolean
}

defineProps<Props>()
defineEmits<{
  cancel: []
}>()
</script>
