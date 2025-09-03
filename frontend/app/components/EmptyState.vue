<template>
  <div class="flex flex-col items-center justify-center py-16 px-4">
    <!-- 图标或插图 -->
    <div class="mb-6">
      <div v-if="!$slots.illustration" 
           class="w-20 h-20 rounded-full bg-neutral-100 dark:bg-neutral-800 flex items-center justify-center">
        <UIcon :name="icon || 'i-heroicons-inbox'" class="w-10 h-10 text-neutral-400" />
      </div>
      <slot name="illustration" />
    </div>
    
    <!-- 标题 -->
    <h3 class="text-xl font-semibold text-neutral-900 dark:text-neutral-100 mb-2 text-center">
      {{ title || '暂无内容' }}
    </h3>
    
    <!-- 描述 -->
    <p class="text-neutral-600 dark:text-neutral-400 text-center max-w-md mb-6">
      {{ description || '这里还没有任何内容，快来添加第一个吧！' }}
    </p>
    
    <!-- 操作按钮 -->
    <div v-if="$slots.actions || actionText" class="flex flex-col sm:flex-row gap-3">
      <slot name="actions">
        <UButton 
          v-if="actionText"
          @click="$emit('action')"
          :icon="actionIcon"
          size="lg"
        >
          {{ actionText }}
        </UButton>
      </slot>
    </div>
    
    <!-- 辅助操作 -->
    <div v-if="$slots.secondary" class="mt-4">
      <slot name="secondary" />
    </div>
  </div>
</template>

<script setup lang="ts">
interface Props {
  title?: string
  description?: string
  icon?: string
  actionText?: string
  actionIcon?: string
}

defineProps<Props>()
defineEmits<{
  action: []
}>()
</script>