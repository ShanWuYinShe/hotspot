<template>
  <div class="space-y-6 p-4">
    <!-- 多个消息项骨架 -->
    <div 
      v-for="i in count" 
      :key="i" 
      class="flex items-start gap-3"
      :class="{ 'flex-row-reverse': i % 3 === 0 }" 
    >
      <!-- 头像骨架 -->
      <USkeleton class="size-8 rounded-full shrink-0" />
      
      <!-- 消息内容骨架 -->
      <div class="flex-1 max-w-xs space-y-2">
        <!-- 发送者名称骨架 -->
        <USkeleton class="h-3 w-16" :class="{ 'ml-auto': i % 3 === 0 }" />
        
        <!-- 消息气泡骨架 -->
        <div 
          class="p-3 rounded-2xl"
          :class="i % 3 === 0 ? 'bg-primary-50 dark:bg-primary-950/50 rounded-tr-sm' : 'bg-neutral-100 dark:bg-neutral-800 rounded-tl-sm'"
        >
          <div class="space-y-2">
            <USkeleton class="h-4 w-full" />
            <USkeleton 
              v-if="i % 2 === 0" 
              class="h-4"
              :class="i % 4 === 0 ? 'w-3/4' : 'w-1/2'"
            />
          </div>
        </div>
        
        <!-- 时间戳骨架 -->
        <USkeleton class="h-2 w-12" :class="{ 'ml-auto': i % 3 === 0 }" />
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
