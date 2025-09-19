<template>
  <div class="space-y-3">
    <!-- 多个会话项骨架 -->
    <div 
      v-for="i in count" 
      :key="i" 
      class="flex items-center gap-4 p-4 animate-pulse"
    >
      <!-- 头像骨架 -->
      <div class="relative">
        <div class="w-12 h-12 bg-slate-200 dark:bg-slate-700 rounded-full"></div>
        <!-- 在线状态指示器骨架 -->
        <div class="absolute -bottom-1 -right-1 w-4 h-4 bg-slate-200 dark:bg-slate-700 rounded-full"></div>
      </div>
      
      <!-- 会话信息骨架 -->
      <div class="flex-1 min-w-0 space-y-2">
        <div class="flex items-center justify-between">
          <!-- 用户名骨架 -->
          <div class="h-4 bg-slate-200 dark:bg-slate-700 rounded w-24"></div>
          <!-- 时间骨架 -->
          <div class="h-3 bg-slate-200 dark:bg-slate-700 rounded w-12"></div>
        </div>
        
        <div class="flex items-center justify-between">
          <!-- 最后消息预览骨架 -->
          <div class="h-3 bg-slate-200 dark:bg-slate-700 rounded w-40"></div>
          <!-- 未读消息数骨架 -->
          <div 
            v-if="i % 3 === 0" 
            class="w-5 h-5 bg-slate-200 dark:bg-slate-700 rounded-full"
          ></div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
interface Props {
  count?: number
}

withDefaults(defineProps<Props>(), {
  count: 6
})
</script>
