<template>
    <div class="w-full">
        <!-- 快速操作 -->
        <div class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-4">
            <UCard
                v-for="action in quickActions"
                :key="action.key"
                variant="interactive"
                @click="action.action"
            >
                <div class="flex items-center gap-4">
                    <UAvatar
                        :icon="action.icon"
                        size="lg"
                        :color="action.color"
                        variant="soft"
                    />
                    <div class="flex-1 min-w-0">
                        <h3 class="hs-card-title">
                            {{ action.title }}
                        </h3>
                        <p class="hs-card-description">
                          {{ action.description }}
                        </p>
                    </div>
                </div>
            </UCard>
        </div>
    </div>
</template>

<script setup lang="ts">
import { promiseTimeout } from '@vueuse/core'
import type { StatItem } from '~/components/data/StatsGrid.vue'
import StatsGrid from '~/components/data/StatsGrid.vue'
import { useUsersService } from '~/services/users.service'
import type { UserStatsData } from '~/types/api'
import { useDashboardHeader } from '~/composables/useDashboardHeader'

// 需要登录 - 数据仪表板需要认证
definePageMeta({
    title: '数据仪表板',
    description: '查看您的活动统计和趋势分析',
    requiresAuth: true,
    layout: 'dashboard',
    dashboardHeader: true
})

// 服务和状态
const feedback = useFeedbackStore()
const userService = useUsersService()
const authStore = useAuthStore()
const modal = useModal()
const dashboardHeader = useDashboardHeader()

const hydratedUser = useHydratedStore(authStore, 'user', null as typeof authStore.user)
const currentUserId = computed(() => hydratedUser.value?.user_id)

// 提示辅助函数
const success = (message: string) => feedback.success(message)
const error = (message: string) => feedback.error(message)
const info = (message: string) => feedback.info(message)

// 加载状态
const loading = reactive({
    overview: false
})

// 用户统计数据
const userStats = ref<UserStatsData>({
    moments_count: 0,
    friends_count: 0,
    followers_count: 0,
    following_count: 0,
    groups_count: 0
})

// 概览统计 (使用真实数据)
const overviewStats = computed<StatItem[]>(() => [
    {
        key: 'total-friends',
        label: '好友总数',
        value: userStats.value.friends_count,
        icon: 'i-lucide-users',
        color: 'blue' as 'blue',
        description: '您的好友总数',
        // change: 0, // 暂无历史数据
        // trend: 'stable',
        onClick: () => navigateTo('/friends')
    },
    {
        key: 'total-groups',
        label: '群组总数',
        value: userStats.value.groups_count,
        icon: 'i-lucide-users-round',
        color: 'green' as 'green',
        description: '加入的群组',
        // change: 0,
        // trend: 'stable',
        onClick: () => navigateTo('/groups')
    },
    {
        key: 'total-following',
        label: '关注中',
        value: userStats.value.following_count,
        icon: 'i-lucide-user-plus',
        color: 'indigo' as 'indigo',
        description: '您关注的用户',
        // change: 0,
        // trend: 'stable',
        onClick: () => navigateTo('/following')
    },
    {
        key: 'total-moments',
        label: '动态总数',
        value: userStats.value.moments_count,
        icon: 'i-lucide-image',
        color: 'orange' as 'orange',
        description: '发布的动态数量',
        // change: 0,
        // trend: 'stable',
        onClick: () => navigateTo('/moments')
    }
])

watchEffect(() => {
    dashboardHeader.setActions([
        {
            key: 'refresh',
            label: '刷新数据',
            icon: 'i-lucide-refresh-cw',
            variant: 'outline',
            loading: loading.overview,
            action: () => loadOverviewStats()
        },
        {
            key: 'settings',
            label: '统计设置',
            icon: 'i-lucide-settings',
            color: 'neutral',
            variant: 'outline',
            action: () => navigateTo('/settings')
        }
    ])
    dashboardHeader.setStats(overviewStats.value)
})

// 快速操作
const quickActions = [
    {
        key: 'create-group',
        title: '创建群组',
        description: '建立新的群组',
        icon: 'i-lucide-circle-plus',
        color: 'info' as const,
        action: async () => {
            const id = await modal.showCreateGroup()
            if (id) await navigateTo(`/groups/${id}`)
        }
    },
    {
        key: 'add-friends',
        title: '添加好友',
        description: '寻找新朋友',
        icon: 'i-lucide-user-plus',
        color: 'success' as const,
        action: () => navigateTo('/search')
    },
    {
        key: 'post-moment',
        title: '发布动态',
        description: '分享生活瞬间',
        icon: 'i-lucide-camera',
        color: 'primary' as const,
        action: async () => {
            const id = await modal.showCreateMoment()
            if (id) await navigateTo(`/moments/${id}`)
        }
    },
    {
        key: 'nearby-people',
        title: '附近的人',
        description: '发现周围用户',
        icon: 'i-lucide-map-pin',
        color: 'warning' as const,
        action: () => navigateTo('/users/nearby')
    }
]

// 加载概览数据 (真实 API)
const loadOverviewStats = async () => {
    loading.overview = true
    try {
        const stats = await userService.getMyStats()
        userStats.value = stats
    } catch (err) {
        error('加载统计数据失败')
        console.error(err)
    } finally {
        loading.overview = false
    }
}

// 页面加载时初始化数据
onMounted(() => {
    loadOverviewStats()
})
</script>