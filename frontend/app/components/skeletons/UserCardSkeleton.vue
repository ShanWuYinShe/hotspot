<template>
  <UCard class="bg-white/60 dark:bg-slate-800/60 border border-slate-200/50 dark:border-slate-700/50">
    <div class="flex items-center gap-4 animate-pulse">
      <!-- 用户头像骨架 -->
      <div class="relative">
        <div class="w-12 h-12 bg-slate-200 dark:bg-slate-700 rounded-full"></div>
        <!-- 在线状态指示器骨架 -->
        <div class="absolute -bottom-1 -right-1 w-4 h-4 bg-slate-200 dark:bg-slate-700 rounded-full"></div>
      </div>

      <!-- 用户信息骨架 -->
      <div class="flex-1 min-w-0 space-y-2">
        <div class="flex items-center gap-2">
          <!-- 用户名骨架 -->
          <div class="h-5 bg-slate-200 dark:bg-slate-700 rounded w-24"></div>
          <!-- 徽章骨架 -->
          <div class="h-4 bg-slate-200 dark:bg-slate-700 rounded w-16"></div>
        </div>
        
        <div class="flex items-center gap-4">
          <!-- 位置信息骨架 -->
          <div class="flex items-center gap-1">
            <div class="w-4 h-4 bg-slate-200 dark:bg-slate-700 rounded"></div>
            <div class="h-3 bg-slate-200 dark:bg-slate-700 rounded w-20"></div>
          </div>
          
          <!-- 距离信息骨架 -->
          <div class="flex items-center gap-1">
            <div class="w-4 h-4 bg-slate-200 dark:bg-slate-700 rounded"></div>
            <div class="h-3 bg-slate-200 dark:bg-slate-700 rounded w-12"></div>
          </div>
        </div>

        <!-- 个人简介骨架 -->
        <div class="space-y-1">
          <div class="h-3 bg-slate-200 dark:bg-slate-700 rounded w-full"></div>
          <div class="h-3 bg-slate-200 dark:bg-slate-700 rounded w-3/4"></div>
        </div>
      </div>

      <!-- 操作按钮骨架 -->
      <div class="flex items-center gap-2">
        <div class="h-8 w-16 bg-slate-200 dark:bg-slate-700 rounded"></div>
        <div class="h-8 w-8 bg-slate-200 dark:bg-slate-700 rounded"></div>
      </div>
    </div>
  </UCard>
</template>

<script setup lang="ts">
// 用户卡片骨架屏组件
</script>
