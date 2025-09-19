<template>
  <div class="space-y-4">
    <!-- 多个消息项骨架 -->
    <div 
      v-for="i in count" 
      :key="i" 
      class="flex items-start gap-3 animate-pulse"
      :class="{ 'flex-row-reverse': i % 3 === 0 }" 
    >
      <!-- 头像骨架 -->
      <div class="w-8 h-8 bg-slate-200 dark:bg-slate-700 rounded-full flex-shrink-0"></div>
      
      <!-- 消息内容骨架 -->
      <div class="flex-1 max-w-xs space-y-2">
        <!-- 发送者名称骨架 -->
        <div class="h-3 bg-slate-200 dark:bg-slate-700 rounded w-16"></div>
        
        <!-- 消息气泡骨架 -->
        <div 
          class="p-3 rounded-2xl space-y-1"
          :class="i % 3 === 0 ? 'bg-primary-100 dark:bg-primary-900/30' : 'bg-slate-100 dark:bg-slate-800'"
        >
          <div class="h-4 bg-slate-200 dark:bg-slate-700 rounded w-full"></div>
          <div 
            v-if="i % 2 === 0" 
            class="h-4 bg-slate-200 dark:bg-slate-700 rounded"
            :class="i % 4 === 0 ? 'w-3/4' : 'w-1/2'"
          ></div>
        </div>
        
        <!-- 时间戳骨架 -->
        <div class="h-2 bg-slate-200 dark:bg-slate-700 rounded w-12"></div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
interface Props {
  count?: number
}

withDefaults(defineProps<Props>(), {
  count: 5
})
</script>
