<template>
  <div class="min-h-screen bg-gradient-to-br from-slate-50 via-slate-100/50 to-slate-50 dark:from-slate-900 dark:via-slate-800/30 dark:to-slate-900">
    <!-- 背景装饰 -->
    <div v-if="showDecorations" class="absolute inset-0 overflow-hidden pointer-events-none">
      <div class="absolute -top-40 -right-40 w-80 h-80 bg-gradient-to-br from-slate-300/10 to-slate-400/10 rounded-full blur-3xl"></div>
      <div class="absolute -bottom-40 -left-40 w-80 h-80 bg-gradient-to-tr from-slate-300/10 to-slate-400/10 rounded-full blur-3xl"></div>
    </div>

    <div class="relative max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-8">
      <!-- 页面头部 -->
      <header v-if="showHeader" class="mb-8">
        <div class="flex flex-col sm:flex-row justify-between items-start sm:items-center gap-4">
          <!-- 标题区域 -->
          <div class="space-y-2">
            <div class="flex items-center gap-3">
              <UButton
                v-if="showBackButton"
                variant="ghost"
                size="sm"
                icon="i-heroicons-arrow-left"
                @click="handleBack"
              />
              <div>
                <h1 class="text-3xl sm:text-4xl font-bold text-slate-800 dark:text-slate-100">
                  {{ title }}
                </h1>
                <p v-if="description" class="text-slate-600 dark:text-slate-400 text-lg mt-1">
                  {{ description }}
                </p>
              </div>
            </div>
            
            <!-- 面包屑导航 -->
            <UBreadcrumb
              v-if="breadcrumbs?.length"
              :items="breadcrumbs"
              separator-icon="i-heroicons-chevron-right"
            />
          </div>

          <!-- 操作区域 -->
          <div v-if="headerActions?.length" class="flex gap-3 w-full sm:w-auto">
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
      </header>

      <!-- 统计信息 -->
      <section v-if="stats?.length" class="mb-8">
        <StatsGrid :stats="stats" :columns="statsColumns" />
      </section>

      <!-- 主要内容区域 -->
      <main>
        <slot />
      </main>

      <!-- 页脚操作面板 -->
      <footer v-if="footerActions?.length">
        <ActionPanel
          :title="footerTitle || '操作'"
          :description="footerDescription"
          :icon="footerIcon || 'i-heroicons-cog-6-tooth'"
          :actions="footerActions"
        />
      </footer>
    </div>
  </div>
</template>

<script setup lang="ts">
import type { StatItem } from './StatsGrid.vue'
import type { ActionItem } from './ActionPanel.vue'

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
  
  // 导航相关
  showBackButton?: boolean
  breadcrumbs?: BreadcrumbItem[]
  
  // 布局控制
  showHeader?: boolean
  showDecorations?: boolean
  
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
  showDecorations: true,
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
