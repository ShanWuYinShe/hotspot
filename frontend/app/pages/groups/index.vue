<template>
    <div class="min-h-screen bg-gradient-to-br from-slate-50 via-slate-100/50 to-slate-50 dark:from-slate-900 dark:via-slate-800/30 dark:to-slate-900">
        <div class="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-8">
            <!-- 页面标题 -->
            <div class="mb-8">
                <h1 class="text-3xl font-bold text-slate-800 dark:text-slate-100 mb-2">
                    我的群组
                </h1>
                <p class="text-slate-600 dark:text-slate-400">
                    管理您加入和创建的所有群组
                </p>
            </div>

            <!-- 统计卡片 -->
            <div class="grid grid-cols-1 md:grid-cols-3 gap-6 mb-8">
                <UCard class="bg-white/60 dark:bg-slate-800/60 border border-slate-200/50 dark:border-slate-700/50">
                    <div class="flex items-center gap-4">
                        <div class="p-3 bg-blue-100 dark:bg-blue-900/30 rounded-lg">
                            <UIcon name="i-heroicons-user-group" class="w-6 h-6 text-blue-600 dark:text-blue-400" />
                        </div>
                        <div>
                            <p class="text-2xl font-bold text-slate-900 dark:text-white">{{ totalGroups }}</p>
                            <p class="text-sm text-slate-600 dark:text-slate-400">总群组数</p>
                        </div>
                    </div>
                </UCard>
                
                <UCard class="bg-white/60 dark:bg-slate-800/60 border border-slate-200/50 dark:border-slate-700/50">
                    <div class="flex items-center gap-4">
                        <div class="p-3 bg-green-100 dark:bg-green-900/30 rounded-lg">
                            <UIcon name="i-heroicons-crown" class="w-6 h-6 text-green-600 dark:text-green-400" />
                        </div>
                        <div>
                            <p class="text-2xl font-bold text-slate-900 dark:text-white">{{ ownedGroups }}</p>
                            <p class="text-sm text-slate-600 dark:text-slate-400">我创建的</p>
                        </div>
                    </div>
                </UCard>
                
                <UCard class="bg-white/60 dark:bg-slate-800/60 border border-slate-200/50 dark:border-slate-700/50">
                    <div class="flex items-center gap-4">
                        <div class="p-3 bg-orange-100 dark:bg-orange-900/30 rounded-lg">
                            <UIcon name="i-heroicons-bell" class="w-6 h-6 text-orange-600 dark:text-orange-400" />
                        </div>
                        <div>
                            <p class="text-2xl font-bold text-slate-900 dark:text-white">{{ totalUnreadCount }}</p>
                            <p class="text-sm text-slate-600 dark:text-slate-400">未读消息</p>
                        </div>
                    </div>
                </UCard>
            </div>

            <!-- 群组列表 -->
            <UCard class="bg-white/60 dark:bg-slate-800/60 border border-slate-200/50 dark:border-slate-700/50">
                <DataView
                    :data="filteredGroups"
                    :loading="loading"
                    :searchable="true"
                    :search-fields="['name', 'description']"
                    :filters="filterOptions"
                    :show-pagination="true"
                    :page-size="12"
                    :toolbar-actions="toolbarActions"
                    item-component="GroupCard"
                    :get-item-key="(item) => item.group_id"
                    :get-item-props="getGroupCardProps"
                    :get-item-events="getGroupCardEvents"
                    empty-icon="i-heroicons-user-group"
                    empty-title="还没有群组"
                    empty-description="加入或创建您的第一个群组"
                    :empty-action="{ label: '创建群组', action: openCreateGroup }"
                    @item-click="(group) => enterGroup(group)"
                    @search="handleSearch"
                    @filter-change="handleFilterChange"
                />
            </UCard>
        </div>
    </div>
</template>

<script setup lang="ts">
import { getPageSeoConfig, buildSeoMeta } from '~/config/seo';
import type { GroupResponse } from "~/types/api"

// 从数据源获取 SEO 配置
const seoConfig = getPageSeoConfig('groups')

definePageMeta({ 
    title: seoConfig.title, 
    description: seoConfig.description,
    requiresAuth: true, 
    layout: 'dashboard' 
})

// 设置 SEO meta
useSeoMeta(buildSeoMeta('groups'))

// 服务和状态
const groupsStore = useGroupsStore()
const { formatTime } = useFormatters()
const { success, error, info } = useNotifications()
const toast = { success, error, info }

// 数据状态
const loading = ref(true)
const searchQuery = ref('')
const activeFilter = ref('all')
const unreadCounts = reactive<Record<string, number>>({})

// 计算属性
const totalGroups = computed(() => groupsStore.myGroups.length)
const ownedGroups = computed(() => 
    groupsStore.myGroups.filter(g => g.user_role === 'owner').length
)
const totalUnreadCount = computed(() => 
    Object.values(unreadCounts).reduce((sum, count) => sum + count, 0)
)

const filteredGroups = computed(() => {
    let groups = [...groupsStore.myGroups]
    
    // 应用筛选
    if (activeFilter.value !== 'all') {
        switch (activeFilter.value) {
            case 'owned':
                groups = groups.filter(g => g.user_role === 'owner')
                break
            case 'joined':
                groups = groups.filter(g => g.user_role !== 'owner')
                break
            case 'unread':
                groups = groups.filter(g => unreadCounts[g.group_id] > 0)
                break
        }
    }
    
    return groups
})

// 筛选选项
const filterOptions = [
    { label: '全部群组', value: 'all' },
    { label: '我创建的', value: 'owned' },
    { label: '我加入的', value: 'joined' },
    { label: '有未读消息', value: 'unread' }
]

// 工具栏操作
const toolbarActions = computed(() => [
    {
        key: 'refresh',
        label: '刷新',
        icon: 'i-heroicons-arrow-path',
        variant: 'outline' as const,
        action: loadMyGroups
    },
    {
        key: 'create',
        label: '创建群组',
        icon: 'i-heroicons-plus',
        color: 'primary' as const,
        action: openCreateGroup
    },
    {
        key: 'discover',
        label: '发现群组',
        icon: 'i-heroicons-magnifying-glass',
        variant: 'outline' as const,
        action: () => navigateTo('/groups/discover')
    }
])

// GroupCard 配置
const getGroupCardProps = (group: GroupResponse) => {
    const unreadCount = unreadCounts[group.group_id] || 0
    return {
        group: {
            ...group,
            unread_count: unreadCount,
            last_active_display: formatTime(group.last_active_at || group.created_at)
        },
        showRole: true,
        showUnreadCount: unreadCount > 0,
        class: 'cursor-pointer hover:shadow-md transition-shadow'
    }
}

const getGroupCardEvents = (group: GroupResponse) => ({
    click: () => enterGroup(group),
    enter: () => enterGroup(group),
    more: () => showGroupActions(group)
})

// 方法
const loadMyGroups = async () => {
    loading.value = true
    try {
        await groupsStore.fetchMyGroups()
        await loadUnreadCounts()
        toast.success('群组列表已更新')
    } catch (error: any) {
        toast.error(`加载失败: ${error?.message || '未知错误'}`)
    } finally {
        loading.value = false
    }
}

const loadUnreadCounts = async () => {
    try {
        const { getGroupUnreadCount } = useStatus()
        const groups = groupsStore.myGroups.slice(0, 30) // 限制数量

        // 批量获取未读数
        const unreadPromises = groups.map(async (group) => {
            const count = await getGroupUnreadCount(group.group_id)
            return { groupId: group.group_id, count }
        })
        const unreadResults = await Promise.all(unreadPromises)
        
        // 更新未读数映射
        unreadResults.forEach(({ groupId, count }) => {
            unreadCounts[groupId] = count
        })
    } catch (error) {
        console.error('批量获取未读数失败:', error)
    }
}

const enterGroup = async (group: GroupResponse) => {
    try {
        // 标记消息为已读
        // TODO: 实现消息已读标记功能
        
        // 清零本地未读数
        unreadCounts[group.group_id] = 0
        
        // 跳转到群聊页面
        await navigateTo(`/chat/groups/${group.group_id}`)
    } catch (error) {
        console.error('进入群组失败:', error)
        // 即使标记已读失败，也允许进入群组
        await navigateTo(`/chat/groups/${group.group_id}`)
    }
}

const { showCreateGroupModal } = useModal()

const openCreateGroup = async () => {
    await showCreateGroupModal({
        onSuccess: (groupId: string) => {
            // 刷新群组列表
            loadGroups()
            toast.success('群组创建成功')
        }
    })
}

const showGroupActions = (group: GroupResponse) => {
    // 这里可以显示群组操作菜单
    toast.info(`群组操作: ${group.name}`)
}

const handleSearch = (query: string) => {
    searchQuery.value = query
    if (query.trim()) {
        toast.info(`搜索: ${query}`)
    }
}

const handleFilterChange = (filter: string) => {
    activeFilter.value = filter
    toast.info(`筛选: ${filterOptions.find(f => f.value === filter)?.label}`)
}

// 生命周期
onMounted(() => {
    loadMyGroups()
})
</script>