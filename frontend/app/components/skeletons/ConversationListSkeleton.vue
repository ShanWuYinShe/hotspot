<template>
  <div class="space-y-3">
    <!-- 多个会话项骨架 -->
    <div 
      v-for="i in count" 
      :key="i" 
      class="flex items-center gap-4 p-4"
    >
      <!-- 头像骨架 -->
      <div class="relative">
        <UAvatar
          size="md"
          icon="i-lucide-user"
          color="neutral"
        />
      </div>
      
      <!-- 会话信息骨架 -->
      <div class="flex-1 min-w-0 space-y-2">
        <div class="flex items-center justify-between">
          <!-- 用户名骨架 -->
          <USkeleton class="h-4 w-24" />
          <!-- 时间骨架 -->
          <USkeleton class="h-3 w-12" />
        </div>
        
        <div class="flex items-center justify-between">
          <!-- 最后消息预览骨架 -->
          <USkeleton class="h-3 w-40" />
          <!-- 未读消息数骨架 -->
          <USkeleton 
            v-if="i % 3 === 0" 
            class="w-5 h-5 rounded-full"
          />
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
