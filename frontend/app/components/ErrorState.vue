<template>
  <div class="flex flex-col items-center justify-center py-16 px-4">
    <!-- 错误图标 -->
    <div class="mb-6">
      <div class="w-20 h-20 rounded-full bg-error-50 dark:bg-error-950 flex items-center justify-center">
        <UIcon :name="icon || 'i-heroicons-exclamation-triangle'" class="w-10 h-10 text-error-500" />
      </div>
    </div>
    
    <!-- 错误标题 -->
    <h3 class="text-xl font-semibold text-neutral-900 dark:text-neutral-100 mb-2 text-center">
      {{ title || '出现错误' }}
    </h3>
    
    <!-- 错误描述 -->
    <p class="text-neutral-600 dark:text-neutral-400 text-center max-w-md mb-6">
      {{ description || '抱歉，加载内容时出现了问题。请稍后重试。' }}
    </p>
    
    <!-- 错误详情（开发模式） -->
    <UAccordion v-if="showDetails && errorDetails" :items="[{ label: '错误详情', slot: 'details' }]" class="w-full max-w-md mb-6">
      <template #details>
        <pre class="text-xs text-neutral-600 dark:text-neutral-400 bg-neutral-50 dark:bg-neutral-800 p-3 rounded overflow-auto max-h-32">{{ errorDetails }}</pre>
      </template>
    </UAccordion>
    
    <!-- 操作按钮 -->
    <div class="flex flex-col sm:flex-row gap-3">
      <UButton 
        @click="$emit('retry')"
        icon="i-heroicons-arrow-path"
        size="lg"
        :loading="retrying"
      >
        重试
      </UButton>
      
      <UButton 
        v-if="showGoBack"
        @click="$emit('goBack')"
        variant="outline"
        icon="i-heroicons-arrow-left"
        size="lg"
      >
        返回
      </UButton>
    </div>
    
    <!-- 辅助信息 -->
    <div v-if="showHelp" class="mt-6 text-center">
      <p class="text-sm text-neutral-500 mb-2">问题持续存在？</p>
      <UButton 
        @click="$emit('contact')"
        variant="ghost" 
        size="sm"
        icon="i-heroicons-chat-bubble-left-right"
      >
        联系客服
      </UButton>
    </div>
  </div>
</template>

<script setup lang="ts">
interface Props {
  title?: string
  description?: string
  icon?: string
  errorDetails?: string
  showDetails?: boolean
  showGoBack?: boolean
  showHelp?: boolean
  retrying?: boolean
}

defineProps<Props>()
defineEmits<{
  retry: []
  goBack: []
  contact: []
}>()
</script>
