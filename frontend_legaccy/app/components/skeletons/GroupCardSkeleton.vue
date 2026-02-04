<template>
  <UCard class="h-full" variant="outline">
    <div class="space-y-4">
      <!-- 群组头部信息骨架 -->
      <div class="flex items-start gap-4">
        <!-- 群组头像骨架 -->
        <USkeleton class="size-10 rounded-xl shrink-0" />

        <!-- 群组信息骨架 -->
        <div class="flex-1 min-w-0 space-y-2">
          <div class="flex items-center gap-2">
            <!-- 群组名称骨架 -->
            <USkeleton class="h-5 w-32" />
            <!-- 状态徽章骨架 -->
            <USkeleton class="h-4 w-12" />
            <USkeleton class="h-4 w-16" />
          </div>
          
          <div class="flex items-center gap-4">
            <!-- 成员数量骨架 -->
            <div class="flex items-center gap-1">
              <USkeleton class="size-4 rounded-full" />
              <USkeleton class="h-3 w-12" />
            </div>
            
            <!-- 位置信息骨架 -->
            <div class="flex items-center gap-1">
              <USkeleton class="size-4 rounded-full" />
              <USkeleton class="h-3 w-20" />
            </div>
            
            <!-- 距离信息骨架 -->
            <div class="flex items-center gap-1">
              <USkeleton class="size-4 rounded-full" />
              <USkeleton class="h-3 w-12" />
            </div>
          </div>

          <!-- 群组描述骨架 -->
          <div class="space-y-1">
            <USkeleton class="h-3 w-full" />
            <USkeleton class="h-3 w-5/6" />
          </div>
        </div>

        <!-- 操作按钮骨架 -->
        <div class="flex items-center gap-2">
          <USkeleton class="h-8 w-16 rounded-lg" />
          <USkeleton class="size-8 rounded-lg" />
        </div>
      </div>

      <!-- 群组活动信息骨架 -->
      <div class="pt-3">
        <USeparator class="mb-3" />
        <div class="flex items-center justify-between">
          <div class="flex items-center gap-1">
            <USkeleton class="size-4 rounded-full" />
            <USkeleton class="h-3 w-24" />
          </div>

          <div class="flex items-center gap-1">
            <USkeleton class="size-4 rounded-full" />
            <USkeleton class="h-3 w-20" />
          </div>
        </div>
      </div>
    </div>
  </UCard>
</template>

<script setup lang="ts">
// 群组卡片骨架屏组件 - 使用 Nuxt UI v4 的 USkeleton
</script>
