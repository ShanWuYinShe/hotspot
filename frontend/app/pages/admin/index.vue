<template>
  <div class="w-full">
    <div v-if="canRenderAdmin">
      <div class="grid grid-cols-1 lg:grid-cols-2 gap-6 mb-8">
        <UCard>
          <template #header>
            <div class="flex items-center gap-2">
              <UIcon name="i-lucide-heart" :size="20" />
              <h3 class="font-semibold">系统健康状态</h3>
            </div>
          </template>
          
          <div class="space-y-4">
            <div class="flex items-center justify-between">
              <UBadge color="neutral" variant="soft" size="xs">系统状态</UBadge>
              <UBadge
                :color="getHealthColor(healthData.status)"
                :label="getHealthLabel(healthData.status)"
              />
            </div>
            
            <div class="flex items-center justify-between">
              <UBadge color="neutral" variant="soft" size="xs">版本</UBadge>
              <span class="text-sm font-medium tabular-nums">{{ healthData.version }}</span>
            </div>
            
            <div class="flex items-center justify-between">
              <UBadge color="neutral" variant="soft" size="xs">服务器时间</UBadge>
              <span class="text-sm font-medium tabular-nums">
                {{ new Date(healthData.timestamp).toLocaleString() }}
              </span>
            </div>
          </div>
        </UCard>

        <UCard>
          <template #header>
            <div class="flex items-center gap-2">
              <UIcon name="i-lucide-bar-chart-3" :size="20" />
              <h3 class="font-semibold">管理入口</h3>
            </div>
          </template>
          
          <div class="grid grid-cols-2 gap-3">
            <UButton variant="outline" color="neutral" block @click="navigateTo('/admin/reports')">
              举报处理
            </UButton>
            <UButton variant="outline" color="neutral" block @click="navigateTo('/admin/config')">
              系统配置
            </UButton>
            <UButton
              v-if="canManageRbac"
              variant="outline"
              color="neutral"
              block
              @click="navigateTo('/admin/rbac')"
            >
              权限管理
            </UButton>
          </div>
        </UCard>
      </div>
    </div>

    <div v-else class="w-full space-y-4">
      <USkeleton class="h-32 w-full" />
      <USkeleton class="h-32 w-full" />
    </div>
  </div>
</template>

<script setup lang="ts">
import { useSystemService } from '~/services/system.service'
import { useAdminStore } from '~/stores/admin.store'
import { useDashboardHeader } from '~/composables/useDashboardHeader'
import { useHydratedStore } from '~/composables/useHydratedStore'

definePageMeta({
  title: '系统管理面板',
  description: '监控系统运行状态和管理平台数据',
  requiresAuth: true,
  layout: 'dashboard',
  middleware: 'admin', // 需要管理员权限
  dashboardHeader: true
})

const systemService = useSystemService()
const { withErrorHandling } = useErrorHandler()
const authStore = useAuthStore()
const adminStore = useAdminStore()
const dashboardHeader = useDashboardHeader()

const isAuthenticated = useHydratedStore(authStore, 'isAuthenticated', false)
const canRenderAdmin = computed(() => isAuthenticated.value && adminStore.hasAccess)

const canManageRbac = computed(() => adminStore.hasPermission('admin.rbac.manage'))

// 状态管理
const loading = ref(false)
const healthData = ref({
  status: 'healthy',
  timestamp: new Date().toISOString(),
  version: '0.0.0'
})

watchEffect(() => {
  dashboardHeader.setActions([
    {
      key: 'refresh',
      label: '刷新数据',
      icon: 'i-lucide-refresh-cw',
      variant: 'outline',
      loading: loading.value,
      action: () => loadHealthData(true)
    }
  ])
})

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

// 方法
const loadHealthData = async (manual = false) => {
  await withErrorHandling(async () => {
    // 调用系统健康检查 API
    const response = await systemService.healthCheck()
    // 更新基本信息（来自 API）
    healthData.value.status = response.status
    healthData.value.version = response.version
    healthData.value.timestamp = response.timestamp
  }, {
    title: '加载系统状态失败',
    loading: loading,
    showToast: manual // 自动刷新失败不弹 Toast，手动刷新时弹
  })
}

// 页面加载
onMounted(() => {
  loadHealthData(true)
})

// 定时刷新数据（30秒刷新一次）
useIntervalFn(() => {
  loadHealthData(false)
}, 30000)
</script>

