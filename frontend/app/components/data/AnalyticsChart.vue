<template>
  <UCard variant="outline">
    <template #header>
      <div class="flex items-center justify-between">
        <div>
          <h3 class="text-lg font-semibold">{{ title }}</h3>
          <p
            v-if="description"
            class="font-medium mt-0.5"
          >{{ description }}</p>
        </div>
        <div class="flex items-center gap-2">
          <USelectMenu
            v-if="timeRanges?.length"
            v-model="selectedTimeRange"
            :options="timeRanges"
            color="neutral"
            variant="soft"
            class="min-w-[100px]"
          />
          <UButton
            variant="ghost"
            color="neutral"
            icon="i-lucide-refresh-cw"
            @click="refresh"
          />
        </div>
      </div>
    </template>

    <div class="space-y-6">
      <!-- 主要指标 -->
      <div
        v-if="metrics?.length"
        class="grid grid-cols-2 md:grid-cols-4 gap-4"
      >
        <UCard
          v-for="metric in metrics"
          :key="metric.key"
          variant="subtle"
        >
          <div class="text-center">
            <div class="text-2xl font-bold mt-1">
              {{ transformValue(metric.value) }}
            </div>
            <div class="text-xs font-medium mt-1 uppercase tracking-wide">
              {{ metric.label }}
            </div>
            <div
              v-if="metric.change !== undefined"
              class="flex items-center justify-center gap-1 mt-2"
            >
              <UBadge
                :color="metric.change >= 0 ? 'success' : 'error'"
                variant="soft"
                size="xs"
              >
                <UIcon
                  :name="metric.change >= 0 ? 'i-lucide-trending-up' : 'i-lucide-trending-down'"
                  :size="14"
                />
                <span>
                  {{ Math.abs(metric.change).toFixed(1) }}%
                </span>
              </UBadge>
            </div>
          </div>
        </UCard>
      </div>

      <!-- 图表区域 -->
      <div class="relative">
        <div
          v-if="loading"
          class="flex items-center justify-center h-64"
        >
          <USkeleton class="h-64 w-full" />
        </div>

        <div
          v-if="chartType === 'line'"
          class="h-64"
        >
          <!-- 简单的折线图模拟 -->
          <UCard
            variant="subtle"
            class="h-full"
          >
            <div class="relative h-full p-4">
              <svg
                class="w-full h-full"
                viewBox="0 0 400 200"
                style="color: var(--ui-primary);"
              >
                <!-- 网格线 -->
                <defs>
                  <linearGradient
                    id="lineGradient"
                    x1="0"
                    y1="0"
                    x2="0"
                    y2="1"
                  >
                    <stop
                      offset="0%"
                      stop-color="currentColor"
                      stop-opacity="0.2"
                    />
                    <stop
                      offset="100%"
                      stop-color="currentColor"
                      stop-opacity="0"
                    />
                  </linearGradient>
                  <pattern
                    id="grid"
                    width="40"
                    height="40"
                    patternUnits="userSpaceOnUse"
                  >
                    <path
                      d="M 40 0 L 0 0 0 40"
                      fill="none"
                      stroke="currentColor"
                      stroke-width="0.5"
                      opacity="0.05"
                    />
                  </pattern>
                </defs>
                <rect
                  width="100%"
                  height="100%"
                  fill="url(#grid)"
                />

                <!-- 填充区域 -->
                <polyline
                  v-if="chartData?.length"
                  :points="generateAreaPoints(chartData || [])"
                  fill="url(#lineGradient)"
                  stroke="none"
                />

                <!-- 数据线 -->
                <polyline
                  v-if="chartData?.length"
                  :points="generateLinePoints(chartData || [])"
                  fill="none"
                  stroke="currentColor"
                  stroke-width="3"
                  stroke-linecap="round"
                  stroke-linejoin="round"
                  class="hs-chart-line"
                />

                <!-- 数据点 -->
                <circle
                  v-for="(point, index) in generatePoints(chartData || [])"
                  :key="index"
                  :cx="point.x"
                  :cy="point.y"
                  r="4"
                  fill="currentColor"
                  stroke="currentColor"
                  stroke-width="2"
                  class="cursor-pointer transition-transform duration-200 hover:scale-110"
                />
              </svg>
            </div>
          </UCard>
        </div>

        <div
          v-else-if="chartType === 'bar'"
          class="h-64"
        >
          <!-- 简单的柱状图模拟 -->
          <UCard
            variant="subtle"
            class="h-full"
          >
            <div class="flex items-end justify-between h-full p-4 gap-3">
              <div
                v-for="(item, index) in chartData"
                :key="index"
                class="flex-1 relative group cursor-pointer transition-all duration-300 hs-chart-bar"
                :style="{ height: `${Math.max((item.value / maxValue) * 100, 4)}%` }"
              >
                <div class="absolute inset-0 opacity-0 group-hover:opacity-100 transition-opacity"></div>

                <!-- 悬浮提示 -->
                <UBadge
                  class="absolute -top-10 left-1/2 -translate-x-1/2 opacity-0 group-hover:opacity-100 transition-opacity whitespace-nowrap"
                  color="neutral"
                  variant="solid"
                  size="xs"
                >
                  {{ item.label }}: {{ transformValue(item.value) }}
                </UBadge>
              </div>
            </div>
          </UCard>

          <!-- X轴标签 -->
          <div class="flex justify-between mt-3 px-4">
            <span
              v-for="(item, index) in chartData"
              :key="index"
              class="text-[10px] font-medium uppercase tracking-wider flex-1 text-center"
            >
              {{ item.label }}
            </span>
          </div>
        </div>

        <div
          v-else-if="chartType === 'pie'"
          class="h-64 flex items-center justify-center"
        >
          <!-- 简单的饼图模拟 -->
          <div class="relative">
            <svg
              width="200"
              height="200"
              viewBox="0 0 200 200"
            >
              <circle
                v-for="(segment, index) in pieSegments"
                :key="index"
                cx="100"
                cy="100"
                r="80"
                fill="none"
                :stroke="segment.color"
                stroke-width="24"
                :stroke-dasharray="`${segment.length} ${500 - segment.length}`"
                :stroke-dashoffset="segment.offset"
                class="cursor-pointer transition-opacity duration-200 hover:opacity-100 opacity-90"
                transform="rotate(-90 100 100)"
              />
            </svg>

            <!-- 中心文字 -->
            <div class="absolute inset-0 flex items-center justify-center flex-col pointer-events-none">
              <UBadge
                color="neutral"
                variant="soft"
                size="xs"
              >Total</UBadge>
              <UBadge
                color="neutral"
                variant="solid"
                size="lg"
              >100%</UBadge>
            </div>

            <!-- 图例 -->
            <div class="absolute -right-32 top-1/2 -translate-y-1/2 space-y-3">
              <div
                v-for="(item, index) in chartData"
                :key="index"
                class="flex items-center gap-2 text-sm group cursor-pointer"
              >
                <div
                  class="size-3 shrink-0 transition-transform group-hover:scale-110 hs-chart-dot"
                  :style="{ backgroundColor: pieColors[index % pieColors.length] }"
                ></div>
                <UBadge
                  color="neutral"
                  variant="soft"
                  size="xs"
                >{{ item.label }}</UBadge>
              </div>
            </div>
          </div>
        </div>

        <!-- 空状态 -->
        <UCard
          v-else-if="!chartData?.length"
          variant="subtle"
          class="h-64 flex items-center justify-center"
        >
          <UEmpty
            icon="i-lucide-bar-chart-3"
            title="暂无数据"
            description="当前时间段内没有数据可显示"
          />
        </UCard>
      </div>
    </div>
  </UCard>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue'

interface ChartDataItem {
  label: string
  value: number
}

interface MetricItem {
  key: string
  label: string
  value: number
  change?: number
}

interface TimeRangeItem {
  label: string
  value: string
}

const props = defineProps<{
  title?: string
  description?: string
  chartType?: 'line' | 'bar' | 'pie'
  chartData?: ChartDataItem[]
  metrics?: MetricItem[]
  loading?: boolean
  timeRanges?: TimeRangeItem[]
}>()

const emit = defineEmits<{
  (e: 'refresh'): void
  (e: 'time-range-change', value: string): void
}>()

const selectedTimeRange = ref(props.timeRanges?.[0] || null)

watch(selectedTimeRange, (newVal) => {
  if (newVal) {
    emit('time-range-change', newVal.value)
  }
})

const refresh = () => {
  emit('refresh')
}

const transformValue = (val: number) => {
  if (val >= 10000) return (val / 10000).toFixed(1) + 'w'
  if (val >= 1000) return (val / 1000).toFixed(1) + 'k'
  return val
}

const maxValue = computed(() => {
  if (!props.chartData?.length) return 0
  return Math.max(...props.chartData.map(item => item.value))
})

// Pie Chart Logic
const pieColors = [
  '#007aff',   // Apple Blue
  '#34c759',   // Apple Green
  '#ff9500',   // Apple Orange
  '#af52de',   // Apple Purple
  '#ff3b30',   // Apple Red
  '#5856d6',   // Apple Indigo
  '#ff2d55',   // Apple Pink
  '#5ac8fa'    // Apple Teal
]

const pieSegments = computed(() => {
  if (!props.chartData?.length) return []
  const total = props.chartData.reduce((acc, item) => acc + item.value, 0)
  let currentOffset = 0

  return props.chartData.map((item, index) => {
    const length = (item.value / total) * 500 // 500 is roughly 2 * PI * 80 (circumference)
    const segment = {
      length,
      offset: -currentOffset,
      color: pieColors[index % pieColors.length]
    }
    currentOffset += length
    return segment
  })
})

// Line Chart Logic
const generatePoints = (data: ChartDataItem[]) => {
  if (!data?.length) return []
  const max = maxValue.value || 1
  const width = 400
  const height = 200
  const step = width / (data.length - 1)

  return data.map((item, index) => ({
    x: index * step,
    y: height - (item.value / max) * height * 0.8 - 20 // Padding
  }))
}

const generateLinePoints = (data: ChartDataItem[]) => {
  const points = generatePoints(data)
  return points.map(p => `${p.x},${p.y}`).join(' ')
}

const generateAreaPoints = (data: ChartDataItem[]) => {
  if (!data?.length) return ''
  const linePoints = generateLinePoints(data)
  return `0,200 ${linePoints} 400,200`
}
</script>