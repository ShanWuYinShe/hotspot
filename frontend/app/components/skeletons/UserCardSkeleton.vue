<template>
  <UCard>
    <div class="flex items-center gap-4">
      <!-- 用户头像骨架 -->
      <div class="relative">
        <USkeleton class="h-12 w-12 rounded-full" />
        <!-- 在线状态指示器骨架 -->
        <USkeleton class="absolute -bottom-1 -right-1 h-4 w-4 rounded-full" />
      </div>

      <!-- 用户信息骨架 -->
      <div class="flex-1 min-w-0 space-y-2">
        <div class="flex items-center gap-2">
          <!-- 用户名骨架 -->
          <USkeleton class="h-5 w-24" />
          <!-- 徽章骨架 -->
          <USkeleton class="h-4 w-16" />
        </div>
        
        <div class="flex items-center gap-4">
          <!-- 位置信息骨架 -->
          <div class="flex items-center gap-1">
            <USkeleton class="h-4 w-4" />
            <USkeleton class="h-3 w-20" />
          </div>
          
          <!-- 距离信息骨架 -->
          <div class="flex items-center gap-1">
            <USkeleton class="h-4 w-4" />
            <USkeleton class="h-3 w-12" />
          </div>
        </div>

        <!-- 个人简介骨架 -->
        <div class="space-y-1">
          <USkeleton class="h-3 w-full" />
          <USkeleton class="h-3 w-3/4" />
        </div>
      </div>

      <!-- 操作按钮骨架 -->
      <div class="flex items-center gap-2">
        <USkeleton class="h-8 w-16" />
        <USkeleton class="h-8 w-8" />
      </div>
    </div>
  </UCard>
</template>

<script setup lang="ts">
// 用户卡片骨架屏组件 - 使用 Nuxt UI v4 的 USkeleton
</script>
