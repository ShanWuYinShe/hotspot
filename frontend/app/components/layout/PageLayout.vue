<template>
  <UPage>
    <!-- 页面头部 -->
    <UPageHeader
      v-if="showHeader"
      :title="title"
      :description="description"
      :icon="headerIcon"
      :ui="{
        wrapper: 'relative',
        container: 'max-w-7xl mx-auto px-4 sm:px-6 lg:px-8',
        title: 'text-3xl sm:text-4xl font-bold',
        description: 'text-lg mt-1'
      }"
    >
      <template #leading>
        <UButton
          v-if="showBackButton"
          variant="ghost"
          size="sm"
          icon="i-heroicons-arrow-left"
          @click="handleBack"
        />
      </template>

      <template #trailing>
        <div v-if="headerActions?.length" class="flex gap-3">
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
      </template>
    </UPageHeader>

    <!-- 面包屑导航 -->
    <div v-if="breadcrumbs?.length" class="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 mb-6">
      <UBreadcrumb
        :items="breadcrumbs"
        separator-icon="i-heroicons-chevron-right"
      />
    </div>

    <!-- 统计信息 -->
    <div v-if="stats?.length" class="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 mb-8">
      <StatsGrid :stats="stats" :columns="statsColumns" />
    </div>

    <!-- 主要内容区域 -->
    <UPageBody>
      <slot />
    </UPageBody>

    <!-- 页脚操作面板 -->
    <div v-if="footerActions?.length" class="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 mt-8">
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

export interface BreadcrumbItem {
  label: string
  to?: string
}

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
  breadcrumbs?: BreadcrumbItem[]
  
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

const handleBack = () => {
  emit('back')
  // 默认行为：返回上一页
  if (history.length > 1) {
    history.back()
  } else {
    navigateTo('/')
  }
}
</script>
