<template>
  <UCard class="bg-white/60 dark:bg-slate-800/60 border border-slate-200/50 dark:border-slate-700/50">
    <template #header>
      <div class="flex items-center justify-between">
        <div>
          <h3 class="text-lg font-semibold text-neutral-900 dark:text-white">{{ title }}</h3>
          <p v-if="description" class="text-sm text-neutral-600 dark:text-neutral-400 mt-1">{{ description }}</p>
        </div>
        <div class="flex items-center gap-2">
          <USelectMenu
            v-if="timeRanges?.length"
            v-model="selectedTimeRange"
            :options="timeRanges"
            size="sm"
          />
          <UButton
            variant="ghost"
            size="sm"
            icon="i-heroicons-arrow-path"
            @click="refresh"
          />
        </div>
      </div>
    </template>

    <div class="space-y-6">
      <!-- 主要指标 -->
      <div v-if="metrics?.length" class="grid grid-cols-2 md:grid-cols-4 gap-4">
        <div
          v-for="metric in metrics"
          :key="metric.key"
          class="text-center p-4 bg-slate-50 dark:bg-slate-700/50 rounded-lg"
        >
          <div class="text-2xl font-bold text-neutral-900 dark:text-white">
            {{ formatValue(metric.value) }}
          </div>
          <div class="text-sm text-neutral-600 dark:text-neutral-400 mt-1">
            {{ metric.label }}
          </div>
          <div v-if="metric.change !== undefined" class="flex items-center justify-center gap-1 mt-2">
            <UIcon
              :name="metric.change >= 0 ? 'i-heroicons-arrow-trending-up' : 'i-heroicons-arrow-trending-down'"
              :class="metric.change >= 0 ? 'text-green-600' : 'text-red-600'"
              class="w-3 h-3"
            />
            <span
              :class="metric.change >= 0 ? 'text-green-600' : 'text-red-600'"
              class="text-xs font-medium"
            >
              {{ Math.abs(metric.change).toFixed(1) }}%
            </span>
          </div>
        </div>
      </div>

      <!-- 图表区域 -->
      <div class="relative">
        <div v-if="loading" class="flex items-center justify-center h-64">
          <USkeleton class="h-64 w-full" />
        </div>
        
        <div v-else-if="chartType === 'line'" class="h-64">
          <!-- 简单的折线图模拟 -->
          <div class="relative h-full bg-slate-50 dark:bg-slate-700/30 rounded-lg p-4">
            <svg class="w-full h-full" viewBox="0 0 400 200">
              <!-- 网格线 -->
              <defs>
                <pattern id="grid" width="40" height="20" patternUnits="userSpaceOnUse">
                  <path d="M 40 0 L 0 0 0 20" fill="none" stroke="currentColor" stroke-width="0.5" opacity="0.2"/>
                </pattern>
              </defs>
              <rect width="100%" height="100%" fill="url(#grid)" />
              
              <!-- 数据线 -->
              <polyline
                v-if="chartData?.length"
                :points="generateLinePoints(chartData || [])"
                fill="none"
                stroke="rgb(59 130 246)"
                stroke-width="2"
                class="drop-shadow-sm"
              />
              
              <!-- 数据点 -->
              <circle
                v-for="(point, index) in generatePoints(chartData || [])"
                :key="index"
                :cx="point.x"
                :cy="point.y"
                r="3"
                fill="rgb(59 130 246)"
                class="drop-shadow-sm"
              />
            </svg>
          </div>
        </div>
        
        <div v-else-if="chartType === 'bar'" class="h-64">
          <!-- 简单的柱状图模拟 -->
          <div class="flex items-end justify-between h-full bg-slate-50 dark:bg-slate-700/30 rounded-lg p-4 gap-2">
            <div
              v-for="(item, index) in chartData"
              :key="index"
              class="flex-1 bg-gradient-to-t from-primary-600 to-primary-400 rounded-t-sm relative group cursor-pointer"
              :style="{ height: `${(item.value / maxValue) * 100}%` }"
            >
              <!-- 悬浮提示 -->
              <div class="absolute bottom-full left-1/2 transform -translate-x-1/2 mb-2 px-2 py-1 bg-slate-900 text-white text-xs rounded opacity-0 group-hover:opacity-100 transition-opacity whitespace-nowrap">
                {{ item.label }}: {{ formatValue(item.value) }}
              </div>
            </div>
          </div>
          
          <!-- X轴标签 -->
          <div class="flex justify-between mt-2 px-4">
            <span
              v-for="(item, index) in chartData"
              :key="index"
              class="text-xs text-neutral-600 dark:text-neutral-400 flex-1 text-center"
            >
              {{ item.label }}
            </span>
          </div>
        </div>
        
        <div v-else-if="chartType === 'pie'" class="h-64 flex items-center justify-center">
          <!-- 简单的饼图模拟 -->
          <div class="relative">
            <svg width="200" height="200" viewBox="0 0 200 200">
              <circle
                v-for="(segment, index) in pieSegments"
                :key="index"
                cx="100"
                cy="100"
                r="80"
                fill="none"
                :stroke="segment.color"
                stroke-width="20"
                :stroke-dasharray="`${segment.length} ${500 - segment.length}`"
                :stroke-dashoffset="segment.offset"
                class="cursor-pointer hover:stroke-width-[25] transition-all"
                transform="rotate(-90 100 100)"
              />
            </svg>
            
            <!-- 图例 -->
            <div class="absolute -right-24 top-0 space-y-2">
              <div
                v-for="(item, index) in chartData"
                :key="index"
                class="flex items-center gap-2 text-sm"
              >
                <div
                  class="w-3 h-3 rounded-full"
                  :style="{ backgroundColor: pieColors[index % pieColors.length] }"
                ></div>
                <span class="text-neutral-700 dark:text-neutral-300">{{ item.label }}</span>
              </div>
            </div>
          </div>
        </div>
        
        <!-- 空状态 -->
        <div v-else-if="!chartData?.length" class="flex items-center justify-center h-64">
          <EmptyState
            icon="i-heroicons-chart-bar"
            title="暂无数据"
            description="当前时间段内没有数据可显示"
          />
        </div>
      </div>
    </div>
  </UCard>
</template>

<script setup lang="ts">
export interface ChartDataItem {
  label: string
  value: number
}

export interface MetricItem {
  key: string
  label: string
  value: number
  change?: number
}

export interface TimeRange {
  label: string
  value: string
}

interface Props {
  title: string
  description?: string
  chartType: 'line' | 'bar' | 'pie'
  chartData?: ChartDataItem[]
  metrics?: MetricItem[]
  timeRanges?: TimeRange[]
  loading?: boolean
}

const props = withDefaults(defineProps<Props>(), {
  loading: false
})
const { formatCount } = useFormatters()

// 格式化数值显示
const formatValue = (value: number) => formatCount(value)

const emit = defineEmits<{
  refresh: []
  timeRangeChange: [value: string]
}>()

const selectedTimeRange = ref(props.timeRanges?.[0]?.value || '7d')

// 计算属性
const maxValue = computed(() => {
  if (!props.chartData?.length) return 0
  return Math.max(...props.chartData.map(item => item.value))
})

const pieSegments = computed(() => {
  if (!props.chartData?.length) return []
  
  const total = props.chartData.reduce((sum, item) => sum + item.value, 0)
  let currentOffset = 0
  
  return props.chartData.map((item, index) => {
    const percentage = item.value / total
    const length = percentage * 500 // 圆周长度
    const segment = {
      length,
      offset: -currentOffset,
      color: pieColors[index % pieColors.length]
    }
    currentOffset += length
    return segment
  })
})

// 颜色配置
const pieColors = [
  'rgb(59 130 246)',   // blue
  'rgb(16 185 129)',   // green
  'rgb(245 158 11)',   // orange
  'rgb(139 92 246)',   // purple
  'rgb(239 68 68)',    // red
  'rgb(6 182 212)',    // cyan
  'rgb(236 72 153)',   // pink
  'rgb(34 197 94)'     // emerald
]

// 方法

const generateLinePoints = (data: ChartDataItem[]) => {
  if (!data?.length) return ''
  
  const width = 400
  const height = 200
  const padding = 20
  
  return data.map((item, index) => {
    const x = padding + (index / (data.length - 1)) * (width - 2 * padding)
    const y = height - padding - ((item.value / maxValue.value) * (height - 2 * padding))
    return `${x},${y}`
  }).join(' ')
}

const generatePoints = (data: ChartDataItem[]) => {
  if (!data?.length) return []
  
  const width = 400
  const height = 200
  const padding = 20
  
  return data.map((item, index) => ({
    x: padding + (index / (data.length - 1)) * (width - 2 * padding),
    y: height - padding - ((item.value / maxValue.value) * (height - 2 * padding))
  }))
}

const refresh = () => {
  emit('refresh')
}

// 监听时间范围变化
watch(selectedTimeRange, (newValue) => {
  emit('timeRangeChange', newValue)
})
</script>
