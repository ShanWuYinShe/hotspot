<template>
    <div class="flex flex-col gap-6">
        <!-- Header -->
        <div>
            <h1 class="text-2xl font-bold text-neutral-900 dark:text-white">管理控制台</h1>
            <p class="text-neutral-500 dark:text-neutral-400 mt-1">系统管理与监控</p>
        </div>

        <div class="space-y-8">
            <!-- 快捷入口 -->
            <div class="grid grid-cols-1 md:grid-cols-3 gap-6">
                <UCard
                    v-for="link in quickLinks"
                    :key="link.to"
                    variant="outline"
                    @click="navigateTo(link.to)"
                >
                    <div class="flex flex-col items-center text-center space-y-3 py-4">
                        <div class="p-3 rounded-full bg-white dark:bg-neutral-800 shadow-sm">
                            <UIcon :name="link.icon" class="size-8 text-neutral-900 dark:text-white" />
                        </div>
                        <h3 class="font-semibold text-lg text-neutral-900 dark:text-white">{{ link.label }}</h3>
                        <p class="text-sm text-neutral-500 dark:text-neutral-400">{{ link.description }}</p>
                    </div>
                </UCard>
            </div>

            <!-- 系统健康状态 -->
            <UCard variant="outline">
                <template #header>
                    <div class="flex items-center justify-between">
                        <h3 class="font-semibold text-lg flex items-center gap-2">
                            <UIcon name="i-lucide-activity" class="text-primary-500" />
                            系统状态
                        </h3>
                        <UButton variant="ghost" color="neutral" icon="i-lucide-refresh-cw" :loading="checkingHealth" @click="checkHealth" />
                    </div>
                </template>
                
                <div v-if="healthData" class="space-y-4">
                    <div class="flex items-center gap-2">
                        <span class="text-sm font-medium text-neutral-500 dark:text-neutral-400">状态:</span>
                        <UBadge :color="healthColor" variant="soft">{{ healthData.status }}</UBadge>
                    </div>
                    <div class="flex items-center gap-2">
                         <span class="text-sm font-medium text-neutral-500 dark:text-neutral-400">版本:</span>
                         <span class="text-sm font-mono text-neutral-900 dark:text-white">{{ healthData.version }}</span>
                    </div>
                     <div class="flex items-center gap-2">
                         <span class="text-sm font-medium text-neutral-500 dark:text-neutral-400">检查时间:</span>
                         <span class="text-sm text-neutral-900 dark:text-white">{{ formatTime(healthData.timestamp) }}</span>
                    </div>
                </div>
                <div v-else class="text-center py-4 text-neutral-500 dark:text-neutral-400">
                    暂无健康数据
                </div>
            </UCard>
        </div>
    </div>
</template>

<script setup lang="ts">
import { useSystemService } from '~/services/system.service'

definePageMeta({
    layout: 'dashboard',
    requiresAuth: true,
    middleware: ['admin']
})

const systemService = useSystemService()
const { formatTime } = useFormatters()

const quickLinks = [
    { label: '用户管理', description: '管理平台用户账户', icon: 'i-lucide-users', to: '/admin/users' },
    { label: '群组管理', description: '管理平台群组', icon: 'i-lucide-users-2', to: '/admin/groups' },
    { label: '举报审核', description: '处理用户举报内容', icon: 'i-lucide-shield-alert', to: '/admin/reports' },
    { label: '系统公告', description: '发布系统公告通知', icon: 'i-lucide-megaphone', to: '/admin/announcements' },
    { label: '系统配置', description: '管理全局系统参数', icon: 'i-lucide-settings-2', to: '/admin/config' },
    { label: '权限管理', description: 'RBAC 角色与权限', icon: 'i-lucide-key-round', to: '/admin/rbac' },
    { label: '审计日志', description: '查看系统操作记录', icon: 'i-lucide-scroll-text', to: '/admin/audit' }
]

const checkingHealth = ref(false)
const healthData = ref<any>(null)

const healthColor = computed(() => {
    switch (healthData.value?.status) {
        case 'healthy': return 'success'
        case 'degraded': return 'warning'
        case 'unhealthy': return 'error'
        default: return 'neutral'
    }
})

onMounted(() => {
    checkHealth()
})

async function checkHealth() {
    checkingHealth.value = true
    try {
        healthData.value = await systemService.healthCheck()
    } catch (e) {
        // error
    } finally {
        checkingHealth.value = false
    }
}
</script>
