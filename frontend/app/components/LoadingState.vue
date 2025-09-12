<template>
  <div class="flex flex-col items-center justify-center py-8 md:py-12 space-y-4">
    <!-- 加载动画 -->
    <div class="relative">
      <div class="w-12 h-12 md:w-16 md:h-16 border-4 border-primary-200 dark:border-primary-800 rounded-full animate-spin">
        <div class="absolute top-0 left-0 w-full h-full border-4 border-transparent border-t-primary-500 rounded-full animate-spin"></div>
      </div>
    </div>
    
    <!-- 加载文本 -->
    <div class="text-center space-y-2">
      <p class="text-sm md:text-base font-medium text-neutral-700 dark:text-neutral-300">
        {{ title || '加载中...' }}
      </p>
      <p v-if="description" class="text-xs md:text-sm text-neutral-500 dark:text-neutral-400">
        {{ description }}
      </p>
    </div>
    
    <!-- 进度条（可选） -->
    <div v-if="showProgress && progress !== undefined" class="w-full max-w-xs">
      <UProgress 
        :value="progress" 
        :max="100"
        class="h-2 [&>div]:bg-primary-500 [&>div]:dark:bg-primary-400 [&>div]:transition-all [&>div]:duration-300 [&>div]:ease-out"
      />
      <p class="text-xs text-center mt-2 text-neutral-500 dark:text-neutral-400">
        {{ Math.round(progress) }}%
      </p>
    </div>
  </div>
</template>

<script setup lang="ts">
interface Props {
  title?: string;
  description?: string;
  showProgress?: boolean;
  progress?: number;
}

defineProps<Props>();
</script>