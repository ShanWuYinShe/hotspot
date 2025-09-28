<template>
  <div class="flex flex-col items-center justify-center py-8 md:py-12 lg:py-16 space-y-4 md:space-y-6">
    <!-- 错误图标 -->
    <div class="relative">
      <div class="w-16 h-16 md:w-20 md:h-20 lg:w-24 lg:h-24 rounded-full bg-red-50 dark:bg-red-900/20 flex items-center justify-center">
        <UIcon 
          :name="icon" 
          class="w-8 h-8 md:w-10 md:h-10 lg:w-12 lg:h-12 text-red-500 dark:text-red-400" 
        />
      </div>
      <!-- 错误指示器 -->
      <div class="absolute -top-1 -right-1 w-6 h-6 md:w-7 md:h-7 bg-red-500 rounded-full flex items-center justify-center">
        <UIcon name="i-heroicons-exclamation-mark" class="w-3 h-3 md:w-4 md:h-4 text-white" />
      </div>
    </div>
    
    <!-- 错误信息 -->
    <div class="text-center space-y-2 max-w-md mx-auto px-4">
      <h3 class="text-base md:text-lg lg:text-xl font-semibold text-red-800 dark:text-red-200">
        {{ title }}
      </h3>
      <p class="text-sm md:text-base text-red-600 dark:text-red-300 leading-relaxed">
        {{ description }}
      </p>
      
      <!-- 错误详情（开发模式） -->
      <details v-if="error && isDev" class="mt-4 text-left">
        <summary class="text-xs text-neutral-500 dark:text-neutral-400 cursor-pointer hover:text-neutral-700 dark:hover:text-neutral-300">
          查看错误详情
        </summary>
        <pre class="mt-2 p-3 bg-neutral-100 dark:bg-neutral-800 rounded-lg text-xs text-neutral-700 dark:text-neutral-300 overflow-auto max-h-32">{{ error }}</pre>
      </details>
    </div>
    
    <!-- 操作按钮 -->
    <div class="flex flex-col sm:flex-row gap-3 pt-2">
      <UButton 
        variant="soft"
        color="error"
        size="sm"
        @click="$emit('retry')"
        class="transition-colors duration-200"
      >
        <template #leading>
          <UIcon name="i-heroicons-arrow-path" class="w-4 h-4" />
        </template>
        重试
      </UButton>
      
      <UButton 
        v-if="showGoBack"
        variant="outline"
        color="neutral"
        size="sm"
        @click="$emit('goBack')"
        class="transition-colors duration-200"
      >
        <template #leading>
          <UIcon name="i-heroicons-arrow-left" class="w-4 h-4" />
        </template>
        返回
      </UButton>
    </div>
    
    <!-- 自定义操作插槽 -->
    <div v-if="$slots.actions" class="pt-2">
      <slot name="actions" />
    </div>
  </div>
</template>

<script setup lang="ts">
interface Props {
  icon?: string;
  title: string;
  description: string;
  error?: any;
  showGoBack?: boolean;
}

withDefaults(defineProps<Props>(), {
  icon: 'i-heroicons-exclamation-triangle',
  showGoBack: false
});

defineEmits<{
  retry: [];
  goBack: [];
}>();

// 检查是否为开发模式
const isDev = import.meta.dev;
</script>