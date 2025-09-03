<template>
  <!-- 通用骨架屏组件 - 使用 Nuxt UI v3 USkeleton -->
  <div class="animate-pulse">
    <!-- 用户卡片骨架屏 -->
    <div v-if="type === 'user-card'" class="space-y-4">
      <UCard>
        <div class="flex items-center space-x-4">
          <USkeleton class="h-12 w-12 rounded-full" />
          <div class="space-y-2 flex-1">
            <USkeleton class="h-4 w-32" />
            <USkeleton class="h-3 w-24" />
          </div>
          <USkeleton class="h-8 w-20" />
        </div>
      </UCard>
    </div>

    <!-- 群组卡片骨架屏 -->
    <div v-else-if="type === 'group-card'" class="space-y-4">
      <UCard>
        <div class="flex items-center space-x-4">
          <USkeleton class="h-12 w-12 rounded-lg" />
          <div class="space-y-2 flex-1">
            <USkeleton class="h-4 w-40" />
            <USkeleton class="h-3 w-32" />
            <USkeleton class="h-3 w-28" />
          </div>
          <div class="space-y-2">
            <USkeleton class="h-6 w-16" />
            <USkeleton class="h-6 w-12" />
          </div>
        </div>
      </UCard>
    </div>

    <!-- 动态卡片骨架屏 -->
    <div v-else-if="type === 'moment-card'" class="space-y-4">
      <UCard>
        <!-- 用户信息 -->
        <div class="flex items-center space-x-3 mb-4">
          <USkeleton class="h-10 w-10 rounded-full" />
          <div class="space-y-2 flex-1">
            <USkeleton class="h-4 w-24" />
            <USkeleton class="h-3 w-32" />
          </div>
        </div>
        
        <!-- 内容 -->
        <div class="space-y-2 mb-4">
          <USkeleton class="h-4 w-full" />
          <USkeleton class="h-4 w-3/4" />
          <USkeleton class="h-4 w-1/2" />
        </div>
        
        <!-- 图片占位 -->
        <div v-if="showImage" class="grid grid-cols-3 gap-2 mb-4">
          <USkeleton class="aspect-square rounded-lg" />
          <USkeleton class="aspect-square rounded-lg" />
          <USkeleton class="aspect-square rounded-lg" />
        </div>
        
        <!-- 操作按钮 -->
        <div class="flex items-center space-x-4 pt-3 border-t border-neutral-200 dark:border-neutral-700">
          <USkeleton class="h-6 w-12" />
          <USkeleton class="h-6 w-12" />
        </div>
      </UCard>
    </div>

    <!-- 对话列表骨架屏 -->
    <div v-else-if="type === 'conversation-item'" class="space-y-4">
      <UCard>
        <div class="flex items-center space-x-4">
          <USkeleton class="h-12 w-12 rounded-full" />
          <div class="space-y-2 flex-1">
            <div class="flex justify-between items-center">
              <USkeleton class="h-4 w-32" />
              <USkeleton class="h-3 w-16" />
            </div>
            <USkeleton class="h-3 w-48" />
          </div>
        </div>
      </UCard>
    </div>

    <!-- 表格行骨架屏 -->
    <div v-else-if="type === 'table-row'" class="space-y-2">
      <div class="flex items-center space-x-4 py-3 border-b border-neutral-200 dark:border-neutral-700">
        <USkeleton class="h-8 w-8 rounded-full" />
        <USkeleton class="h-4 w-32 flex-1" />
        <USkeleton class="h-4 w-24" />
        <USkeleton class="h-4 w-16" />
        <USkeleton class="h-6 w-20" />
      </div>
    </div>

    <!-- 页面标题骨架屏 -->
    <div v-else-if="type === 'page-header'" class="space-y-4">
      <div class="flex items-center justify-between">
        <div class="space-y-2">
          <USkeleton class="h-8 w-48" />
          <USkeleton class="h-4 w-64" />
        </div>
        <USkeleton class="h-10 w-24" />
      </div>
    </div>

    <!-- 列表骨架屏 -->
    <div v-else-if="type === 'list'" class="space-y-3">
      <div v-for="i in count" :key="i">
        <component :is="getListItemComponent()" />
      </div>
    </div>

    <!-- 基础文本骨架屏 -->
    <div v-else class="space-y-2">
      <USkeleton v-for="i in lines" :key="i" :class="getLineClass(i)" />
    </div>
  </div>
</template>

<script setup lang="ts">
interface Props {
  type?: 'user-card' | 'group-card' | 'moment-card' | 'conversation-item' | 'table-row' | 'page-header' | 'list' | 'text'
  lines?: number
  count?: number
  showImage?: boolean
}

const props = withDefaults(defineProps<Props>(), {
  type: 'text',
  lines: 3,
  count: 5,
  showImage: false
})

// 获取列表项组件
const getListItemComponent = () => {
  switch (props.type) {
    case 'list':
      return resolveComponent('SkeletonLoader')
    default:
      return 'div'
  }
}

// 获取文本行样式
const getLineClass = (index: number) => {
  const baseClass = 'h-4'
  
  if (props.lines === 1) return `${baseClass} w-3/4`
  
  if (index === 1) return `${baseClass} w-full`
  if (index === props.lines) return `${baseClass} w-1/2`
  
  return `${baseClass} w-5/6`
}
</script>
