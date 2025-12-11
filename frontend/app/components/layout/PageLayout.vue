<template>
  <UPage>
    <!-- 页面头部 -->
    <div v-if="showHeader" class="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 pb-6">
      <!-- 返回按钮 -->
      <div v-if="showBackButton" class="mb-4">
        <UButton
          variant="ghost"
          color="neutral"
          size="md"
          icon="i-heroicons-arrow-left"
          @click="handleBack"
        >
          返回
        </UButton>
      </div>
      
      <!-- 标题区域 -->
      <div class="flex items-start justify-between gap-4">
        <div class="flex-1 min-w-0">
          <div class="flex items-center gap-3">
            <div v-if="headerIcon" class="p-2.5 bg-primary-subtle rounded-xl">
              <UIcon :name="headerIcon" class="w-6 h-6 text-primary" />
            </div>
            <div>
              <h1 class="text-2xl sm:text-3xl font-bold text-primary-content">
                {{ title }}
              </h1>
              <p v-if="description" class="mt-1 text-base text-secondary-content">
                {{ description }}
              </p>
            </div>
          </div>
        </div>
        
        <!-- 头部操作按钮 -->
        <div v-if="headerActions?.length" class="flex gap-3 shrink-0">
          <UButton
            v-for="action in headerActions"
            :key="action.key"
            :color="action.color || 'primary'"
            :variant="action.variant || 'solid'"
            :size="action.size || 'md'"
            :loading="action.loading"
            :disabled="action.disabled"
            :class="action.class"
            @click="action.action"
          >
            <UIcon v-if="action.icon" :name="action.icon" class="w-4 h-4 mr-2" />
            {{ action.label }}
          </UButton>
        </div>
      </div>
    </div>

    <!-- 统计信息 -->
    <div v-if="stats?.length" class="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 mb-6">
      <StatsGrid :stats="stats" :columns="statsColumns" />
    </div>

    <!-- 主要内容区域 -->
    <UPageBody>
      <slot />
    </UPageBody>

    <!-- 页脚操作面板 -->
    <div v-if="footerActions?.length" class="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 mt-6">
      <ActionPanel
        :title="footerTitle || '操作'"
        :description="footerDescription"
        :icon="footerIcon || 'i-heroicons-cog-6-tooth'"
        :actions="footerActions"
      />
    </div>
  </UPage>
</template>

<script setup lang="ts">
import type { StatItem } from '../data/StatsGrid.vue'
import type { ActionItem } from '../common/ActionPanel.vue'

export interface HeaderAction {
  key: string
  label: string
  icon?: string
  color?: 'primary' | 'secondary' | 'success' | 'info' | 'warning' | 'error' | 'neutral'
  variant?: 'solid' | 'outline' | 'soft' | 'ghost' | 'link'
  size?: 'xs' | 'sm' | 'md' | 'lg' | 'xl'
  loading?: boolean
  disabled?: boolean
  class?: string
  action: () => void
}

interface Props {
  // 页面基本信息
  title: string
  description?: string
  headerIcon?: string
  
  // 导航相关
  showBackButton?: boolean
  
  // 布局控制
  showHeader?: boolean
  
  // 统计信息
  stats?: StatItem[]
  statsColumns?: 1 | 2 | 3 | 4 | 5 | 6
  
  // 头部操作
  headerActions?: HeaderAction[]
  
  // 页脚操作面板
  footerTitle?: string
  footerDescription?: string
  footerIcon?: string
  footerActions?: ActionItem[]
}

const props = withDefaults(defineProps<Props>(), {
  showHeader: true,
  showBackButton: false,
  statsColumns: 4
})

const emit = defineEmits<{
  back: []
}>()

const router = useRouter()

const handleBack = () => {
  emit('back')
  router.back()
}
</script>
