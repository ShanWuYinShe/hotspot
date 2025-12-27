<template>
  <div class="space-y-4">
    <!-- 多个消息项骨架 -->
    <div 
      v-for="i in count" 
      :key="i" 
      class="flex items-start gap-3"
      :class="{ 'flex-row-reverse': i % 3 === 0 }" 
    >
      <!-- 头像骨架 -->
      <UAvatar
        size="xs"
        icon="i-lucide-user"
        color="neutral"
      />
      
      <!-- 消息内容骨架 -->
      <div class="flex-1 max-w-xs space-y-2">
        <!-- 发送者名称骨架 -->
        <USkeleton class="h-3 w-16" />
        
        <!-- 消息气泡骨架 -->
        <UCard
          :variant="i % 3 === 0 ? 'subtle' : 'soft'"
        >
          <div class="space-y-1">
          <USkeleton class="h-4 w-full" />
          <USkeleton 
            v-if="i % 2 === 0" 
            class="h-4"
            :class="i % 4 === 0 ? 'w-3/4' : 'w-1/2'"
          />
          </div>
        </UCard>
        
        <!-- 时间戳骨架 -->
        <USkeleton class="h-2 w-12" />
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
