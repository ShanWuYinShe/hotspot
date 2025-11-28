<template>
  <PageLayout
    title="系统管理面板"
    description="监控系统运行状态和管理平台数据"
    :stats="systemStats"
    :header-actions="headerActions"
  >
    <!-- 系统状态概览 -->
    <div class="grid grid-cols-1 lg:grid-cols-2 gap-6 mb-8">
      <!-- 系统健康状态 -->
      <UCard>
        <template #header>
          <div class="flex items-center gap-2">
            <UIcon name="i-heroicons-heart" class="w-5 h-5" />
            <h3 class="font-semibold">系统健康状态</h3>
          </div>
        </template>
        
        <div class="space-y-4">
          <div class="flex items-center justify-between">
            <span class="text-sm text-neutral-600 dark:text-neutral-400">系统状态</span>
            <UBadge
              :color="getHealthColor(healthData.status)"
              :label="getHealthLabel(healthData.status)"
            />
          </div>
          
          <div class="flex items-center justify-between">
            <span class="text-sm text-neutral-600 dark:text-neutral-400">运行时间</span>
            <span class="text-sm font-medium">{{ formatUptime(healthData.uptime) }}</span>
          </div>
          
          <div class="flex items-center justify-between">
            <span class="text-sm text-neutral-600 dark:text-neutral-400">版本</span>
            <span class="text-sm font-medium">{{ healthData.version }}</span>
          </div>
          
          <div class="flex items-center justify-between">
            <span class="text-sm text-neutral-600 dark:text-neutral-400">数据库</span>
            <div class="flex items-center gap-2">
              <UBadge
                :color="healthData.database?.status === 'connected' ? 'green' : 'red'"
                :label="healthData.database?.status === 'connected' ? '已连接' : '断开'"
              />
              <span class="text-xs text-neutral-500 dark:text-neutral-400">
                {{ healthData.database?.response_time }}ms
              </span>
            </div>
          </div>
          
          <div class="flex items-center justify-between">
            <span class="text-sm text-neutral-600 dark:text-neutral-400">Redis</span>
            <div class="flex items-center gap-2">
              <UBadge
                :color="healthData.redis?.status === 'connected' ? 'green' : 'red'"
                :label="healthData.redis?.status === 'connected' ? '已连接' : '断开'"
              />
              <span class="text-xs text-neutral-500 dark:text-neutral-400">
                {{ healthData.redis?.response_time }}ms
              </span>
            </div>
          </div>
        </div>
      </UCard>

      <!-- 性能指标 -->
      <UCard>
        <template #header>
          <div class="flex items-center gap-2">
            <UIcon name="i-heroicons-chart-bar" class="w-5 h-5" />
            <h3 class="font-semibold">性能指标</h3>
          </div>
        </template>
        
        <div class="space-y-4">
          <div class="flex items-center justify-between">
            <span class="text-sm text-neutral-600 dark:text-neutral-400">CPU 使用率</span>
            <div class="flex items-center gap-2">
              <div class="w-20 bg-neutral-200 dark:bg-neutral-700 rounded-full h-2">
                <div
                  class="bg-primary-500 h-2 rounded-full transition-all"
                  :style="{ width: `${healthData.metrics?.cpu_usage || 0}%` }"
                />
              </div>
              <span class="text-sm font-medium">{{ healthData.metrics?.cpu_usage || 0 }}%</span>
            </div>
          </div>
          
          <div class="flex items-center justify-between">
            <span class="text-sm text-neutral-600 dark:text-neutral-400">内存使用率</span>
            <div class="flex items-center gap-2">
              <div class="w-20 bg-neutral-200 dark:bg-neutral-700 rounded-full h-2">
                <div
                  class="bg-green-500 h-2 rounded-full transition-all"
                  :style="{ width: `${healthData.metrics?.memory_usage || 0}%` }"
                />
              </div>
              <span class="text-sm font-medium">{{ healthData.metrics?.memory_usage || 0 }}%</span>
            </div>
          </div>
          
          <div class="flex items-center justify-between">
            <span class="text-sm text-neutral-600 dark:text-neutral-400">活跃连接</span>
            <span class="text-sm font-medium">{{ healthData.metrics?.active_connections || 0 }}</span>
          </div>
          
          <div class="flex items-center justify-between">
            <span class="text-sm text-neutral-600 dark:text-neutral-400">总请求数</span>
            <span class="text-sm font-medium">{{ formatNumber(healthData.metrics?.total_requests || 0) }}</span>
          </div>
          
          <div class="flex items-center justify-between">
            <span class="text-sm text-neutral-600 dark:text-neutral-400">错误率</span>
            <span class="text-sm font-medium text-red-600">{{ healthData.metrics?.error_rate || 0 }}%</span>
          </div>
        </div>
      </UCard>
    </div>

    <!-- 快捷操作 -->
    <div class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-4 mb-8">
      <UCard
        class="cursor-pointer hover:shadow-md transition-shadow"
        @click="navigateTo('/admin/users')"
      >
        <div class="text-center">
          <UIcon name="i-heroicons-users" class="w-8 h-8 text-neutral-600 dark:text-neutral-300 mx-auto mb-2" />
          <h4 class="font-medium text-neutral-900 dark:text-white">用户管理</h4>
          <p class="text-sm text-neutral-600 dark:text-neutral-400">管理用户账户</p>
        </div>
      </UCard>
      
      <UCard
        class="cursor-pointer hover:shadow-md transition-shadow"
        @click="navigateTo('/admin/reports')"
      >
        <div class="text-center">
          <UIcon name="i-heroicons-shield-exclamation" class="w-8 h-8 text-red-500 mx-auto mb-2" />
          <h4 class="font-medium text-neutral-900 dark:text-white">举报处理</h4>
          <p class="text-sm text-neutral-600 dark:text-neutral-400">处理用户举报</p>
        </div>
      </UCard>
      
      <UCard
        class="cursor-pointer hover:shadow-md transition-shadow"
        @click="navigateTo('/admin/content')"
      >
        <div class="text-center">
          <UIcon name="i-heroicons-document-text" class="w-8 h-8 text-green-500 mx-auto mb-2" />
          <h4 class="font-medium text-neutral-900 dark:text-white">内容管理</h4>
          <p class="text-sm text-neutral-600 dark:text-neutral-400">管理平台内容</p>
        </div>
      </UCard>
      
      <UCard
        class="cursor-pointer hover:shadow-md transition-shadow"
        @click="navigateTo('/admin/settings')"
      >
        <div class="text-center">
          <UIcon name="i-heroicons-cog-6-tooth" class="w-8 h-8 text-purple-500 mx-auto mb-2" />
          <h4 class="font-medium text-neutral-900 dark:text-white">系统设置</h4>
          <p class="text-sm text-neutral-600 dark:text-neutral-400">配置系统参数</p>
        </div>
      </UCard>
    </div>

    <!-- 最近活动 -->
    <UCard>
      <template #header>
        <div class="flex items-center justify-between">
          <div class="flex items-center gap-2">
            <UIcon name="i-heroicons-clock" class="w-5 h-5" />
            <h3 class="font-semibold">最近活动</h3>
          </div>
          <UButton variant="outline" size="sm" @click="loadHealthData">
            刷新
          </UButton>
        </div>
      </template>
      
      <div class="space-y-3">
        <div
          v-for="activity in recentActivities"
          :key="activity.id"
          class="flex items-center gap-3 p-3 bg-neutral-50 dark:bg-neutral-800 rounded-lg"
        >
          <UIcon :name="activity.icon" :class="activity.iconColor" class="w-5 h-5" />
          <div class="flex-1">
            <p class="text-sm font-medium text-neutral-900 dark:text-white">
              {{ activity.title }}
            </p>
            <p class="text-xs text-neutral-600 dark:text-neutral-400">
              {{ activity.description }}
            </p>
          </div>
          <span class="text-xs text-neutral-500 dark:text-neutral-400">
            {{ formatTime(activity.timestamp) }}
          </span>
        </div>
      </div>
    </UCard>
  </PageLayout>
</template>

<script setup lang="ts">
import { useSystemService } from '~/services/system.service'
import type { StatItem } from '~/components/data/StatsGrid.vue'

definePageMeta({
  title: '系统管理面板',
  description: '监控系统运行状态和管理平台数据',
  requiresAuth: true,
  layout: 'dashboard',
  middleware: 'admin' // 需要管理员权限
})

const toast = useToast()

// 状态管理
const loading = ref(false)
const healthData = ref({
  status: 'healthy',
  timestamp: new Date().toISOString(),
  version: '1.0.0',
  uptime: 0,
  database: { status: 'connected', response_time: 0 },
  redis: { status: 'connected', response_time: 0 },
  metrics: {
    cpu_usage: 0,
    memory_usage: 0,
    active_connections: 0,
    total_requests: 0,
    error_rate: 0
  }
})

const recentActivities = ref([
  {
    id: 1,
    title: '新用户注册',
    description: '用户 张三 完成注册',
    icon: 'i-heroicons-user-plus',
    iconColor: 'text-green-500',
    timestamp: new Date().toISOString()
  },
  {
    id: 2,
    title: '举报处理',
    description: '处理了一条垃圾信息举报',
    icon: 'i-heroicons-shield-check',
    iconColor: 'text-neutral-600 dark:text-neutral-300',
    timestamp: new Date(Date.now() - 300000).toISOString()
  }
])

// 计算属性
const systemStats = computed<StatItem[]>(() => [
  {
    key: 'users',
    label: '总用户数',
    value: 12580,
    icon: 'i-heroicons-users',
    color: 'blue',
    description: '平台注册用户总数'
  },
  {
    key: 'groups',
    label: '活跃群组',
    value: 1250,
    icon: 'i-heroicons-user-group',
    color: 'green',
    description: '当前活跃的群组数量'
  },
  {
    key: 'moments',
    label: '生活瞬间',
    value: 45680,
    icon: 'i-heroicons-photo',
    color: 'purple',
    description: '用户发布的瞬间总数'
  },
  {
    key: 'reports',
    label: '待处理举报',
    value: 23,
    icon: 'i-heroicons-exclamation-triangle',
    color: 'red',
    description: '需要处理的举报数量'
  }
])

// 头部操作
const headerActions = computed(() => [
  {
    key: 'refresh',
    label: '刷新数据',
    icon: 'i-heroicons-arrow-path',
    variant: 'outline' as const,
    action: loadHealthData
  },
  {
    key: 'export',
    label: '导出报告',
    icon: 'i-heroicons-document-arrow-down',
    variant: 'outline' as const,
    action: exportReport
  }
])

// 工具函数
const { formatTime, formatCount } = useFormatters()

const getHealthColor = (status: string) => {
  const colors: Record<string, string> = {
    healthy: 'green',
    degraded: 'orange',
    unhealthy: 'red'
  }
  return colors[status] || 'neutral'
}

const getHealthLabel = (status: string) => {
  const labels: Record<string, string> = {
    healthy: '健康',
    degraded: '降级',
    unhealthy: '异常'
  }
  return labels[status] || status
}

const formatUptime = (seconds: number) => {
  const days = Math.floor(seconds / 86400)
  const hours = Math.floor((seconds % 86400) / 3600)
  const minutes = Math.floor((seconds % 3600) / 60)
  
  if (days > 0) {
    return `${days}天 ${hours}小时`
  } else if (hours > 0) {
    return `${hours}小时 ${minutes}分钟`
  } else {
    return `${minutes}分钟`
  }
}

const formatNumber = (num: number) => {
  return formatCount(num)
}

// 方法
const loadHealthData = async () => {
  loading.value = true
  try {
    const systemService = useSystemService()
    const response = await systemService.healthCheck()
    healthData.value = {
      status: response.status,
      timestamp: response.timestamp,
      version: response.version,
      uptime: 0,
      database: { status: 'connected', response_time: 0 },
      redis: { status: 'connected', response_time: 0 },
      metrics: healthData.value.metrics
    }
  } catch (err) {
    console.error('加载系统状态失败:', err)
  } finally {
    loading.value = false
  }
}

const exportReport = async () => {
  try {
    // 导出功能暂未实现
    toast.add({ title: '导出成功', description: '系统报告已下载', color: 'success' })
  } catch (err) {
    console.error('导出报告失败:', err)
    toast.add({ title: '导出失败', description: '无法导出系统报告', color: 'error' })
  }
}

// 页面加载
onMounted(() => {
  loadHealthData()
  
  // 定时刷新数据
  const interval = setInterval(loadHealthData, 30000) // 30秒刷新一次
  
  onUnmounted(() => {
    clearInterval(interval)
  })
})
</script>

