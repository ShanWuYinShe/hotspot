<template>
  <div :class="gridClass">
    <UCard
      v-for="stat in stats"
      :key="stat.key"
      class="interactive-card border cursor-pointer"
      @click="stat.onClick?.()"
    >
      <div class="flex items-center gap-4">
        <!-- 图标区域 -->
        <div 
          class="p-3 rounded-lg shrink-0"
          :class="getIconBgClass(stat.color)"
        >
          <UIcon 
            :name="stat.icon" 
            class="icon-lg"
            :class="getIconClass(stat.color)"
          />
        </div>
        
        <!-- 数据区域 -->
        <div class="flex-1 min-w-0">
          <div class="flex items-baseline gap-2">
            <p class="text-2xl font-bold text-primary-content truncate">
              {{ transformValue(stat.value) }}
            </p>
            <UBadge
              v-if="stat.change !== undefined"
              :color="getChangeColor(stat.change)"
              variant="soft"
              size="xs"
            >
              {{ transformChange(stat.change) }}
            </UBadge>
          </div>
          <p class="text-sm text-secondary-content truncate">
            {{ stat.label }}
          </p>
          <p v-if="stat.description" class="text-xs text-muted truncate mt-1">
            {{ stat.description }}
          </p>
        </div>
        
        <!-- 操作按钮 -->
        <div v-if="stat.actions?.length" class="shrink-0">
          <UDropdownMenu :items="stat.actions">
            <UButton
              variant="ghost"
              size="sm"
              icon="i-heroicons-ellipsis-horizontal"
              @click.stop
            />
          </UDropdownMenu>
        </div>
      </div>
      
      <!-- 趋势图或额外内容 -->
      <div v-if="stat.trend || stat.extraContent" class="mt-4 pt-4 border-t ">
        <!-- 简单趋势指示器 -->
        <div v-if="stat.trend" class="flex items-center justify-between text-xs">
          <span class="text-muted">趋势</span>
          <div class="flex items-center gap-1">
            <UIcon 
              :name="getTrendIcon(stat.trend)" 
              class="icon-xs"
              :class="getTrendColor(stat.trend)"
            />
            <span :class="getTrendColor(stat.trend)">
              {{ getTrendText(stat.trend) }}
            </span>
          </div>
        </div>
        
        <!-- 额外内容插槽 -->
        <div v-if="stat.extraContent">
          <component :is="stat.extraContent" v-bind="stat.extraProps" />
        </div>
      </div>
    </UCard>
  </div>
</template>

<script setup lang="ts">
import type { Component } from 'vue'

export interface StatAction {
  label: string
  icon?: string
  click: () => void
}

export interface StatItem {
  key: string
  label: string
  value: number | string
  icon: string
  color?: 'blue' | 'green' | 'orange' | 'purple' | 'red' | 'yellow' | 'indigo' | 'pink'
  description?: string
  change?: number // 变化百分比
  trend?: 'up' | 'down' | 'stable'
  onClick?: () => void
  actions?: StatAction[]
  extraContent?: Component // 组件
  extraProps?: Record<string, unknown>
}

interface Props {
  stats: StatItem[]
  columns?: 1 | 2 | 3 | 4 | 5 | 6
  responsive?: boolean
}

const props = withDefaults(defineProps<Props>(), {
  columns: 4,
  responsive: true
})
const { formatCount } = useFormatters()

// 转换数值显示
const transformValue = (value: number | string) => {
  if (typeof value === 'string') return value
  return formatCount(value)
}

// 转换变化百分比
const transformChange = (change: number) => {
  const sign = change > 0 ? '+' : ''
  return `${sign}${change.toFixed(1)}%`
}

// 计算网格类名
const gridClass = computed(() => {
  const baseClass = 'grid gap-6'
  
  if (!props.responsive) {
    return `${baseClass} grid-cols-${props.columns}`
  }
  
  // 响应式网格
  const responsiveClasses = {
    1: 'grid-cols-1',
    2: 'grid-cols-1 md:grid-cols-2',
    3: 'grid-cols-1 md:grid-cols-2 lg:grid-cols-3',
    4: 'grid-cols-1 md:grid-cols-2 lg:grid-cols-4',
    5: 'grid-cols-1 md:grid-cols-2 lg:grid-cols-3 xl:grid-cols-5',
    6: 'grid-cols-1 md:grid-cols-2 lg:grid-cols-3 xl:grid-cols-6'
  }
  
  return `${baseClass} ${responsiveClasses[props.columns]}`
})

// 颜色映射
const colorMaps = {
  blue: {
    bg: 'bg-info-base',
    icon: 'text-info'
  },
  green: {
    bg: 'bg-success-base',
    icon: 'text-success'
  },
  orange: {
    bg: 'bg-warning-base',
    icon: 'text-warning'
  },
  purple: {
    bg: 'bg-purple-subtle',
    icon: 'text-purple'
  },
  red: {
    bg: 'bg-red-subtle',
    icon: 'text-error'
  },
  yellow: {
    bg: 'bg-yellow-subtle',
    icon: 'text-yellow'
  },
  indigo: {
    bg: 'bg-indigo-subtle',
    icon: 'text-indigo'
  },
  pink: {
    bg: 'bg-pink-subtle',
    icon: 'text-pink'
  }
}

const getIconBgClass = (color: string = 'blue') => {
  return colorMaps[color as keyof typeof colorMaps]?.bg || colorMaps.blue.bg
}

const getIconClass = (color: string = 'blue') => {
  return colorMaps[color as keyof typeof colorMaps]?.icon || colorMaps.blue.icon
}

const getChangeColor = (change: number) => {
  if (change > 0) return 'success'
  if (change < 0) return 'error'
  return 'neutral'
}

const getTrendIcon = (trend: string) => {
  switch (trend) {
    case 'up': return 'i-heroicons-arrow-trending-up'
    case 'down': return 'i-heroicons-arrow-trending-down'
    default: return 'i-heroicons-minus'
  }
}

const getTrendColor = (trend: string) => {
  switch (trend) {
    case 'up': return 'text-success'
    case 'down': return 'text-error'
    default: return 'text-slate'
  }
}

const getTrendText = (trend: string) => {
  switch (trend) {
    case 'up': return '上升'
    case 'down': return '下降'
    default: return '稳定'
  }
}

</script>
