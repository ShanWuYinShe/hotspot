<template>
  <div class="flex flex-col items-center justify-center py-8 md:py-12 space-y-4">
    <!-- 图标 -->
    <div class="w-16 h-16 md:w-20 md:h-20 rounded-full bg-primary-100 dark:bg-primary-900/40 flex items-center justify-center">
      <UIcon name="i-heroicons-chat-bubble-left-right" class="w-8 h-8 md:w-10 md:h-10 text-primary-600 dark:text-primary-400" />
    </div>
    
    <!-- 标题和描述 -->
    <div class="text-center space-y-2 max-w-sm mx-auto px-4">
      <h3 class="text-base md:text-lg lg:text-xl font-semibold text-neutral-800 dark:text-neutral-200">
        暂无消息
      </h3>
      <p class="text-sm md:text-base text-neutral-600 dark:text-neutral-400 leading-relaxed">
        还没有收到任何消息，开始与朋友聊天吧！
      </p>
    </div>
    
    <!-- 操作按钮 -->
    <div class="pt-2">
      <UButton 
        variant="soft"
        color="primary"
        size="sm"
        @click="$emit('action')"
        class="transition-colors duration-200"
      >
        <template #leading>
          <UIcon name="i-heroicons-chat-bubble-left-right" class="w-4 h-4" />
        </template>
        开始聊天
      </UButton>
    </div>
  </div>
</template>

<script setup lang="ts">
defineEmits<{
  action: [];
}>();
</script>
