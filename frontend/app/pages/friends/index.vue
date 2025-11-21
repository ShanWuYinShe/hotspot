<template>
    <PageLayout
        title="我的好友"
        description="管理您的好友关系和社交网络"
        :stats="friendsStats"
        :header-actions="headerActions"
    >
        <!-- 好友列表 -->
        <UCard class="bg-white/60 dark:bg-slate-800/60 border border-slate-200/50 dark:border-slate-700/50">
            <DataView
                :data="filteredFriends"
                :loading="loading"
                :searchable="true"
                :search-fields="['nickname', 'display_name', 'email']"
                :filters="filterOptions"
                :show-pagination="true"
                :page-size="12"
                :toolbar-actions="toolbarActions"
                item-component="UserCard"
                :get-item-key="(item) => item.user_id"
                :get-item-props="getUserCardProps"
                :get-item-events="getUserCardEvents"
                empty-icon="i-heroicons-users"
                empty-title="还没有好友"
                empty-description="添加您的第一个好友，开始社交之旅"
                :empty-action="{ label: '添加好友', action: () => navigateTo('/search') }"
                @item-click="(user) => handleUserClick(user)"
                @search="handleSearch"
                @filter-change="handleFilterChange"
            />
        </UCard>
    </PageLayout>
</template>

<script setup lang="ts">
import type { UserResponse } from "~/types/api"

definePageMeta({ 
    title: "我的好友", 
    requiresAuth: true, 
    layout: 'dashboard' 
})

// 服务和状态
const toast = useToast()
const userService = useUserService()
const { formatTime } = useFormatters()
const { showFriendRequestsModal } = useModal()

// 数据状态
const loading = ref(true)
const friends = ref<UserResponse[]>([])
const pendingRequests = ref<UserResponse[]>([])
const searchQuery = ref('')
const activeFilter = ref('all')

// 计算属性
const totalFriends = computed(() => friends.value.length)
const onlineFriends = computed(() => 
    friends.value.filter(f => (f as any).is_online).length
)
const pendingRequestsCount = computed(() => pendingRequests.value.length)
const nearbyFriends = computed(() => 
    friends.value.filter(f => (f as any).distance && (f as any).distance < 5000).length
)

// 统计数据
const friendsStats = computed(() => [
    {
        key: 'total',
        label: '总好友数',
        value: totalFriends.value,
        icon: 'i-heroicons-users',
        color: 'blue' as const,
        description: '您的好友总数',
        onClick: () => {
            activeFilter.value = 'all'
            toast.add({ color: 'info', title: '显示全部好友' })
        }
    },
    {
        key: 'online',
        label: '在线好友',
        value: onlineFriends.value,
        icon: 'i-heroicons-signal',
        color: 'green' as const,
        description: '当前在线的好友',
        trend: (onlineFriends.value > totalFriends.value * 0.3 ? 'up' : 'down') as 'up' | 'down',
        onClick: () => {
            activeFilter.value = 'online'
            toast.add({ color: 'info', title: '显示在线好友' })
        }
    },
    {
        key: 'requests',
        label: '待处理请求',
        value: pendingRequestsCount.value,
        icon: 'i-heroicons-bell',
        color: 'orange' as const,
        description: '等待您处理的好友请求',
        onClick: () => navigateTo('/friends/requests')
    },
    {
        key: 'nearby',
        label: '附近好友',
        value: nearbyFriends.value,
        icon: 'i-heroicons-map-pin',
        color: 'purple' as const,
        description: '5公里内的好友',
        onClick: () => {
            activeFilter.value = 'nearby'
            toast.add({ color: 'info', title: '显示附近好友' })
        }
    }
])

// 头部操作
const headerActions = computed(() => [
    {
        key: 'add',
        label: '添加好友',
        icon: 'i-heroicons-user-plus',
        color: 'primary' as const,
        action: () => navigateTo('/search')
    },
    {
        key: 'requests',
        label: `好友请求${pendingRequestsCount.value > 0 ? ` (${pendingRequestsCount.value})` : ''}`,
        icon: 'i-heroicons-bell',
        variant: 'outline' as const,
        action: () => showFriendRequestsModal()
    }
])

const filteredFriends = computed(() => {
    let result = [...friends.value]
    
    // 应用筛选
    switch (activeFilter.value) {
        case 'online':
            result = result.filter(f => (f as any).is_online)
            break
        case 'nearby':
            result = result.filter(f => (f as any).distance && (f as any).distance < 5000)
            break
        case 'recent':
            result = result.filter(f => {
                const lastActive = new Date((f as any).last_active_at || 0)
                const dayAgo = new Date(Date.now() - 24 * 60 * 60 * 1000)
                return lastActive > dayAgo
            })
            break
    }
    
    return result
})

// 筛选选项
const filterOptions = [
    { label: '全部好友', value: 'all' },
    { label: '在线好友', value: 'online' },
    { label: '附近好友', value: 'nearby' },
    { label: '最近活跃', value: 'recent' }
]

// 工具栏操作
const toolbarActions = computed(() => [
    {
        key: 'refresh',
        label: '刷新',
        icon: 'i-heroicons-arrow-path',
        variant: 'outline' as const,
        action: loadFriends
    },
    {
        key: 'add',
        label: '添加好友',
        icon: 'i-heroicons-user-plus',
        color: 'primary' as const,
        action: () => navigateTo('/search')
    },
    {
        key: 'requests',
        label: '好友请求',
        icon: 'i-heroicons-bell',
        variant: 'outline' as const,
        action: () => navigateTo('/friends/requests')
    },
    {
        key: 'nearby',
        label: '附近的人',
        icon: 'i-heroicons-map-pin',
        variant: 'outline' as const,
        action: () => navigateTo('/users/nearby')
    }
])

// UserCard 配置
const getUserCardProps = (user: UserResponse) => ({
    user: {
        ...user,
        display_name: (user as any).display_name || user.nickname,
        is_online: (user as any).is_online || false,
        last_active_display: (user as any).last_active_at ? 
            formatTime((user as any).last_active_at) : '未知'
    },
    showOnlineStatus: true,
    showLastActive: true,
    class: 'cursor-pointer hover:shadow-md transition-shadow'
})

const getUserCardEvents = (user: UserResponse) => ({
    click: () => handleUserClick(user),
    message: () => handleSendMessage(user),
    more: () => handleUserActions(user)
})

// 方法
const loadFriends = async () => {
    loading.value = true
    try {
        const result = await userService.getFriends({ status: 'accepted' })
        friends.value = result.items.map(item => item.user)
        toast.add({ color: 'success', title: '好友列表已更新' })
    } catch (error: any) {
        toast.add({ color: 'error', title: '加载失败', description: error?.message || '未知错误' })
    } finally {
        loading.value = false
    }
}

const loadPendingRequests = async () => {
    try {
        const result = await userService.getFriendRequests({ status: 'received' })
        pendingRequests.value = result.items.map(item => item.from_user)
    } catch (error) {
        console.error('加载好友请求失败:', error)
    }
}

const handleUserClick = (user: UserResponse) => {
    navigateTo(`/users/${user.user_id}`)
}

const handleSendMessage = (user: UserResponse) => {
    executeFeature('privateChat', () => {
        navigateTo(`/chat/users/${user.user_id}`)
    })
}

const handleUserActions = (user: UserResponse) => {
    // 显示用户操作菜单
    toast.add({ color: 'info', title: `用户操作: ${user.nickname}` })
}

const handleSearch = (query: string) => {
    searchQuery.value = query
    if (query.trim()) {
        toast.add({ color: 'info', title: `搜索: ${query}` })
    }
}

const handleFilterChange = (filter: string) => {
    activeFilter.value = filter
    toast.add({ color: 'info', title: `筛选: ${filterOptions.find(f => f.value === filter)?.label}` })
}

// 生命周期
onMounted(async () => {
    await Promise.all([
        loadFriends(),
        loadPendingRequests()
    ])
})
</script>