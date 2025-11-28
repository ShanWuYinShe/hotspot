<template>
  <div :class="gridClass">
    <UCard
      v-for="stat in stats"
      :key="stat.key"
      class="bg-white/60 dark:bg-slate-800/60 border border-slate-200/50 dark:border-slate-700/50 hover:shadow-lg transition-shadow cursor-pointer"
      @click="stat.onClick?.()"
    >
      <div class="flex items-center gap-4">
        <!-- 图标区域 -->
        <div 
          class="p-3 rounded-lg flex-shrink-0"
          :class="getIconBgClass(stat.color)"
        >
          <UIcon 
            :name="stat.icon" 
            class="w-6 h-6"
            :class="getIconClass(stat.color)"
          />
        </div>
        
        <!-- 数据区域 -->
        <div class="flex-1 min-w-0">
          <div class="flex items-baseline gap-2">
            <p class="text-2xl font-bold text-neutral-900 dark:text-white truncate">
              {{ formatValue(stat.value) }}
            </p>
            <UBadge
              v-if="stat.change !== undefined"
              :color="getChangeColor(stat.change)"
              variant="soft"
              size="xs"
            >
              {{ formatChange(stat.change) }}
            </UBadge>
          </div>
          <p class="text-sm text-neutral-600 dark:text-neutral-400 truncate">
            {{ stat.label }}
          </p>
          <p v-if="stat.description" class="text-xs text-neutral-500 dark:text-neutral-400 dark:text-neutral-500 dark:text-neutral-400 truncate mt-1">
            {{ stat.description }}
          </p>
        </div>
        
        <!-- 操作按钮 -->
        <div v-if="stat.actions?.length" class="flex-shrink-0">
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
      <div v-if="stat.trend || stat.extraContent" class="mt-4 pt-4 border-t border-slate-200/50 dark:border-slate-700/50">
        <!-- 简单趋势指示器 -->
        <div v-if="stat.trend" class="flex items-center justify-between text-xs">
          <span class="text-neutral-500 dark:text-neutral-400">趋势</span>
          <div class="flex items-center gap-1">
            <UIcon 
              :name="getTrendIcon(stat.trend)" 
              class="w-3 h-3"
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
  extraContent?: any // 组件
  extraProps?: Record<string, any>
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

// 格式化数值显示
const formatValue = (value: number | string) => {
  if (typeof value === 'string') return value
  return formatCount(value)
}

// 格式化变化百分比
const formatChange = (change: number) => {
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
    bg: 'bg-primary-100 dark:bg-primary-900/30',
    icon: 'text-neutral-700 dark:text-neutral-300'
  },
  green: {
    bg: 'bg-green-100 dark:bg-green-900/30',
    icon: 'text-green-600 dark:text-green-400'
  },
  orange: {
    bg: 'bg-orange-100 dark:bg-orange-900/30',
    icon: 'text-orange-600 dark:text-orange-400'
  },
  purple: {
    bg: 'bg-purple-100 dark:bg-purple-900/30',
    icon: 'text-purple-600 dark:text-purple-400'
  },
  red: {
    bg: 'bg-red-100 dark:bg-red-900/30',
    icon: 'text-red-600 dark:text-red-400'
  },
  yellow: {
    bg: 'bg-yellow-100 dark:bg-yellow-900/30',
    icon: 'text-yellow-600 dark:text-yellow-400'
  },
  indigo: {
    bg: 'bg-indigo-100 dark:bg-indigo-900/30',
    icon: 'text-indigo-600 dark:text-indigo-400'
  },
  pink: {
    bg: 'bg-pink-100 dark:bg-pink-900/30',
    icon: 'text-pink-600 dark:text-pink-400'
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
    case 'up': return 'text-green-600 dark:text-green-400'
    case 'down': return 'text-red-600 dark:text-red-400'
    default: return 'text-neutral-500 dark:text-neutral-400'
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
