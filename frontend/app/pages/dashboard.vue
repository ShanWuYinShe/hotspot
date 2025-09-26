<template>
    <PageLayout
        title="数据仪表板"
        description="查看您的活动统计和趋势分析"
        :stats="overviewStats"
        :header-actions="headerActions"
    >
        <!-- 图表区域 -->
        <div class="grid grid-cols-1 lg:grid-cols-2 gap-8 mb-8">
            <!-- 活动趋势 -->
            <AnalyticsChart
                title="活动趋势"
                description="过去一周的活动统计"
                chart-type="line"
                :chart-data="activityTrendData"
                :metrics="activityMetrics"
                :time-ranges="timeRanges"
                :loading="loading.activity"
                @refresh="loadActivityData"
                @time-range-change="handleTimeRangeChange"
            />
            
            <!-- 群组分布 -->
            <AnalyticsChart
                title="群组分布"
                description="不同类型群组的分布情况"
                chart-type="pie"
                :chart-data="groupDistributionData"
                :metrics="groupMetrics"
                :loading="loading.groups"
                @refresh="loadGroupData"
            />
        </div>
        
        <!-- 详细统计 -->
        <div class="grid grid-cols-1 lg:grid-cols-2 gap-8 mb-8">
            <!-- 好友统计 -->
            <AnalyticsChart
                title="好友增长"
                description="每日新增好友数量"
                chart-type="bar"
                :chart-data="friendsGrowthData"
                :metrics="friendsMetrics"
                :time-ranges="timeRanges"
                :loading="loading.friends"
                @refresh="loadFriendsData"
            />
            
            <!-- 消息统计 -->
            <AnalyticsChart
                title="消息统计"
                description="消息发送和接收统计"
                chart-type="line"
                :chart-data="messagesData"
                :metrics="messagesMetrics"
                :time-ranges="timeRanges"
                :loading="loading.messages"
                @refresh="loadMessagesData"
            />
        </div>
        
        <!-- 快速操作 -->
        <div class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-6">
            <UCard
                v-for="action in quickActions"
                :key="action.key"
                class="bg-white/60 dark:bg-slate-800/60 border border-slate-200/50 dark:border-slate-700/50 hover:shadow-lg transition-shadow cursor-pointer"
                @click="action.action"
            >
                <div class="flex items-center gap-4">
                    <div 
                        class="p-3 rounded-lg flex-shrink-0"
                        :class="action.bgClass"
                    >
                        <UIcon 
                            :name="action.icon" 
                            class="w-6 h-6"
                            :class="action.iconClass"
                        />
                    </div>
                    <div class="flex-1 min-w-0">
                        <h3 class="font-medium text-slate-900 dark:text-white truncate">
                            {{ action.title }}
                        </h3>
                        <p class="text-sm text-slate-600 dark:text-slate-400 truncate">
                            {{ action.description }}
                        </p>
                    </div>
                </div>
            </UCard>
        </div>
    </PageLayout>
</template>

<script setup lang="ts">
import type { StatItem } from '~/components/StatsGrid.vue'

definePageMeta({
    title: '数据仪表板',
    description: '查看您的活动统计和趋势分析',
    layout: 'dashboard'
})

// 服务和状态
const toast = useSmartToast()
const { executeFeature } = useFeatureStatus()

// 加载状态
const loading = reactive({
    activity: false,
    groups: false,
    friends: false,
    messages: false
})

// 时间范围选项
const timeRanges = [
    { label: '最近7天', value: '7d' },
    { label: '最近30天', value: '30d' },
    { label: '最近90天', value: '90d' }
]

// 概览统计
const overviewStats = computed<StatItem[]>(() => [
    {
        key: 'total-friends',
        label: '好友总数',
        value: 156,
        icon: 'i-heroicons-users',
        color: 'blue',
        description: '您的好友总数',
        change: 12.5,
        trend: 'up',
        onClick: () => navigateTo('/friends')
    },
    {
        key: 'total-groups',
        label: '群组总数',
        value: 23,
        icon: 'i-heroicons-user-group',
        color: 'green',
        description: '加入和创建的群组',
        change: 8.3,
        trend: 'up',
        onClick: () => navigateTo('/groups')
    },
    {
        key: 'total-messages',
        label: '消息总数',
        value: 2847,
        icon: 'i-heroicons-chat-bubble-left-right',
        color: 'purple',
        description: '发送和接收的消息',
        change: -2.1,
        trend: 'down',
        onClick: () => navigateTo('/chat')
    },
    {
        key: 'total-moments',
        label: '动态总数',
        value: 89,
        icon: 'i-heroicons-photo',
        color: 'orange',
        description: '发布的动态数量',
        change: 15.7,
        trend: 'up',
        onClick: () => navigateTo('/moments')
    }
])

// 活动趋势数据
const activityTrendData = ref([
    { label: '周一', value: 45 },
    { label: '周二', value: 52 },
    { label: '周三', value: 38 },
    { label: '周四', value: 67 },
    { label: '周五', value: 89 },
    { label: '周六', value: 76 },
    { label: '周日', value: 54 }
])

const activityMetrics = [
    { key: 'avg', label: '平均活动', value: 60, change: 8.5 },
    { key: 'peak', label: '峰值活动', value: 89, change: 12.3 }
]

// 群组分布数据
const groupDistributionData = ref([
    { label: '我创建的', value: 8 },
    { label: '我加入的', value: 15 },
    { label: '活跃群组', value: 18 },
    { label: '静默群组', value: 5 }
])

const groupMetrics = [
    { key: 'active-rate', label: '活跃率', value: 78, change: 5.2 },
    { key: 'avg-members', label: '平均成员', value: 24, change: -1.8 }
]

// 好友增长数据
const friendsGrowthData = ref([
    { label: '周一', value: 2 },
    { label: '周二', value: 5 },
    { label: '周三', value: 1 },
    { label: '周四', value: 8 },
    { label: '周五', value: 3 },
    { label: '周六', value: 6 },
    { label: '周日', value: 4 }
])

const friendsMetrics = [
    { key: 'weekly-growth', label: '周增长', value: 29, change: 15.3 },
    { key: 'acceptance-rate', label: '接受率', value: 85, change: 2.1 }
]

// 消息统计数据
const messagesData = ref([
    { label: '周一', value: 124 },
    { label: '周二', value: 156 },
    { label: '周三', value: 98 },
    { label: '周四', value: 203 },
    { label: '周五', value: 187 },
    { label: '周六', value: 145 },
    { label: '周日', value: 112 }
])

const messagesMetrics = [
    { key: 'daily-avg', label: '日均消息', value: 146, change: -3.2 },
    { key: 'response-rate', label: '回复率', value: 92, change: 1.8 }
]

// 头部操作
const headerActions = computed(() => [
    {
        key: 'export',
        label: '导出报告',
        icon: 'i-heroicons-document-arrow-down',
        variant: 'outline' as const,
        action: () => executeFeature('reports', () => {
            toast.success('报告导出功能即将上线')
        })
    },
    {
        key: 'settings',
        label: '统计设置',
        icon: 'i-heroicons-cog-6-tooth',
        variant: 'outline' as const,
        action: () => navigateTo('/settings')
    }
])

// 快速操作
const quickActions = [
    {
        key: 'create-group',
        title: '创建群组',
        description: '建立新的群组',
        icon: 'i-heroicons-plus-circle',
        bgClass: 'bg-blue-100 dark:bg-blue-900/30',
        iconClass: 'text-blue-600 dark:text-blue-400',
        action: () => navigateTo('/groups/create')
    },
    {
        key: 'add-friends',
        title: '添加好友',
        description: '寻找新朋友',
        icon: 'i-heroicons-user-plus',
        bgClass: 'bg-green-100 dark:bg-green-900/30',
        iconClass: 'text-green-600 dark:text-green-400',
        action: () => navigateTo('/users/search')
    },
    {
        key: 'post-moment',
        title: '发布动态',
        description: '分享生活瞬间',
        icon: 'i-heroicons-camera',
        bgClass: 'bg-purple-100 dark:bg-purple-900/30',
        iconClass: 'text-purple-600 dark:text-purple-400',
        action: () => navigateTo('/moments')
    },
    {
        key: 'nearby-people',
        title: '附近的人',
        description: '发现周围用户',
        icon: 'i-heroicons-map-pin',
        bgClass: 'bg-orange-100 dark:bg-orange-900/30',
        iconClass: 'text-orange-600 dark:text-orange-400',
        action: () => navigateTo('/users/nearby')
    }
]

// 方法
const loadActivityData = async () => {
    loading.activity = true
    try {
        // 模拟API调用
        await new Promise(resolve => setTimeout(resolve, 1000))
        toast.success('活动数据已更新')
    } catch (error) {
        toast.error('加载活动数据失败')
    } finally {
        loading.activity = false
    }
}

const loadGroupData = async () => {
    loading.groups = true
    try {
        await new Promise(resolve => setTimeout(resolve, 800))
        toast.success('群组数据已更新')
    } finally {
        loading.groups = false
    }
}

const loadFriendsData = async () => {
    loading.friends = true
    try {
        await new Promise(resolve => setTimeout(resolve, 600))
        toast.success('好友数据已更新')
    } finally {
        loading.friends = false
    }
}

const loadMessagesData = async () => {
    loading.messages = true
    try {
        await new Promise(resolve => setTimeout(resolve, 700))
        toast.success('消息数据已更新')
    } finally {
        loading.messages = false
    }
}

const handleTimeRangeChange = (range: string) => {
    toast.info(`时间范围已切换到: ${timeRanges.find(r => r.value === range)?.label}`)
    // 根据时间范围重新加载数据
}

// 页面加载时初始化数据
onMounted(() => {
    // 可以在这里加载真实数据
})
</script>
