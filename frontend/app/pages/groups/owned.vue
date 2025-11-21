<template>
    <PageLayout
        title="我创建的群组"
        description="管理您创建的所有群组"
        :stats="groupStats"
        :header-actions="headerActions"
        :footer-title="selectedGroups.length > 0 ? '群组管理' : undefined"
        :footer-description="selectedGroups.length > 0 ? `已选择 ${selectedGroups.length} 个群组` : undefined"
        :footer-icon="selectedGroups.length > 0 ? 'i-heroicons-cog-6-tooth' : undefined"
        :footer-actions="selectedGroups.length > 0 ? batchActions : undefined"
    >
        <!-- 群组列表 -->
        <UCard class="bg-white/60 dark:bg-slate-800/60 border border-slate-200/50 dark:border-slate-700/50">
            <DataView
                :data="items"
                :loading="loading"
                :searchable="true"
                :search-fields="['name', 'description']"
                :show-pagination="true"
                :page-size="10"
                :toolbar-actions="toolbarActions"
                item-component="GroupCard"
                :get-item-key="(item) => item.group_id"
                :get-item-props="getGroupCardProps"
                :get-item-events="getGroupCardEvents"
                empty-icon="i-heroicons-user-group"
                empty-title="还没有创建群组"
                empty-description="创建您的第一个群组，开始与朋友们聊天"
                :empty-action="{ label: '创建群组', action: () => navigateTo('/groups/create') }"
                @item-click="(group) => enterGroup(group)"
                @search="handleSearch"
            />
        </UCard>
    </PageLayout>
</template>

<script setup lang="ts">
import type { GroupResponse } from "~/types/api"
import type { ActionItem } from '~/components/common/ActionPanel.vue'

definePageMeta({ 
    title: "我创建的群组", 
    requiresAuth: true, 
    layout: 'dashboard' 
})

// 服务和状态
const groupsStore = useGroupsStore()
const { formatTime } = useFormatters()
const toast = useToast()

// 数据状态
const items = ref<GroupResponse[]>([])
const loading = ref(true)
const selectedGroups = ref<GroupResponse[]>([])
const ownedUnreadMap = reactive<Record<string, number>>({})

// 计算属性
const totalGroups = computed(() => items.value.length)
const activeGroups = computed(() => {
    const oneDayAgo = new Date(Date.now() - 24 * 60 * 60 * 1000)
    return items.value.filter(g => new Date(g.last_active_at || g.created_at) > oneDayAgo).length
})
const totalMembers = computed(() => 
    items.value.reduce((sum, group) => sum + (group.member_count || 0), 0)
)
const totalUnread = computed(() => 
    Object.values(ownedUnreadMap).reduce((sum, count) => sum + count, 0)
)

// 统计数据
const groupStats = computed(() => [
    {
        key: 'total',
        label: '创建的群组',
        value: totalGroups.value,
        icon: 'i-heroicons-user-group',
        color: 'blue' as const,
        description: '您创建的群组总数',
        onClick: () => toast.add({ color: 'info', title: '显示全部群组' })
    },
    {
        key: 'active',
        label: '活跃群组',
        value: activeGroups.value,
        icon: 'i-heroicons-fire',
        color: 'green' as const,
        description: '24小时内有活动的群组',
        trend: (activeGroups.value > totalGroups.value * 0.5 ? 'up' : 'down') as 'up' | 'down',
        onClick: () => toast.add({ color: 'info', title: '显示活跃群组' })
    },
    {
        key: 'members',
        label: '总成员数',
        value: totalMembers.value,
        icon: 'i-heroicons-users',
        color: 'purple' as const,
        description: '所有群组的成员总数',
        onClick: () => toast.add({ color: 'info', title: '查看成员统计' })
    },
    {
        key: 'unread',
        label: '未读消息',
        value: totalUnread.value,
        icon: 'i-heroicons-bell',
        color: 'orange' as const,
        description: '所有群组的未读消息',
        onClick: () => toast.add({ color: 'info', title: '查看未读消息' })
    }
])

// 头部操作
const headerActions = computed(() => [
    {
        key: 'create',
        label: '创建群组',
        icon: 'i-heroicons-plus',
        color: 'primary' as const,
        action: () => navigateTo('/groups/create')
    },
    {
        key: 'manage-all',
        label: '批量管理',
        icon: 'i-heroicons-cog-6-tooth',
        variant: 'outline' as const,
        action: () => toast.add({ color: 'info', title: '批量管理功能' })
    }
])

// 工具栏操作
const toolbarActions = computed(() => [
    {
        key: 'refresh',
        label: '刷新',
        icon: 'i-heroicons-arrow-path',
        variant: 'outline' as const,
        action: loadOwnedGroups
    },
    {
        key: 'create',
        label: '创建群组',
        icon: 'i-heroicons-plus',
        color: 'primary' as const,
        action: () => navigateTo('/groups/create')
    },
    {
        key: 'analytics',
        label: '数据统计',
        icon: 'i-heroicons-chart-bar',
        variant: 'outline' as const,
        action: () => executeFeature('reports', () => {
            toast.add({ color: 'info', title: '群组数据统计功能即将上线' })
        })
    }
])

// 批量操作
const batchActions = computed<ActionItem[]>(() => [
    {
        key: 'cancel',
        label: '取消选择',
        variant: 'outline',
        action: () => { selectedGroups.value = [] }
    },
    {
        key: 'export',
        label: '导出数据',
        color: 'primary',
        action: handleBatchExport
    },
    {
        key: 'archive',
        label: '批量归档',
        color: 'warning',
        action: handleBatchArchive
    }
])

// GroupCard 配置
const getGroupCardProps = (group: GroupResponse) => {
    const unreadCount = ownedUnreadMap[String(group.group_id)] || 0
    return {
        group: {
            ...group,
            unread_count: unreadCount,
            role: 'owner', // 标记为群主
            last_active_display: formatTime(group.last_active_at || group.created_at)
        },
        showOwnerBadge: true,
        showUnreadCount: unreadCount > 0,
        class: 'cursor-pointer hover:shadow-md transition-shadow'
    }
}

const getGroupCardEvents = (group: GroupResponse) => ({
    click: () => enterGroup(group),
    enter: () => enterGroup(group),
    more: () => manageGroup(group),
    select: () => toggleGroupSelection(group)
})

// 方法
const loadOwnedGroups = async () => {
    loading.value = true
    try {
        await groupsStore.getOwnedGroups()
        items.value = groupsStore.myGroups || []
        
        // 加载未读消息数量
        await loadUnreadCounts()
        
        toast.add({ color: 'success', title: '群组列表已更新' })
    } catch (error: any) {
        toast.add({ color: 'error', title: `加载失败: ${error?.message || '未知错误'}` })
    } finally {
        loading.value = false
    }
}

const loadUnreadCounts = async () => {
    try {
        // 暂时禁用未读数获取，需要WebSocket支持
        const groupsToCheck = items.value.slice(0, 30) // 限制数量避免性能问题

        // 批量获取未读数
        const unreadPromises = groupsToCheck.map(async (group) => {
            const count = 0 // 暂时返回0
            return { groupId: group.group_id, count }
        })
        const unreadResults = await Promise.all(unreadPromises)
        
        // 更新未读数映射
        unreadResults.forEach(({ groupId, count }) => {
            ownedUnreadMap[groupId] = count
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
        ownedUnreadMap[String(group.group_id)] = 0
        
        // 跳转到群聊页面
        await navigateTo(`/chat/groups/${group.group_id}`)
    } catch (error) {
        console.error('进入群组失败:', error)
        // 即使标记已读失败，也允许进入群组
        await navigateTo(`/chat/groups/${group.group_id}`)
    }
}

const manageGroup = async (group: GroupResponse) => {
    await executeFeature('groupManage', async () => {
        await navigateTo(`/groups/${group.group_id}/settings`)
    })
}

const toggleGroupSelection = (group: GroupResponse) => {
    const index = selectedGroups.value.findIndex(g => g.group_id === group.group_id)
    if (index > -1) {
        selectedGroups.value.splice(index, 1)
    } else {
        selectedGroups.value.push(group)
    }
}

const handleSearch = (query: string) => {
    // DataView 组件会自动处理搜索
    if (query.trim()) {
        toast.add({ color: 'info', title: `搜索: ${query}` })
    }
}

const handleBatchExport = async () => {
    await executeFeature('reports', async () => {
        const { exportGroupList } = useDataExport()
        
        try {
            exportGroupList(selectedGroups.value, {
                filename: 'owned-groups',
                format: 'csv'
            })
            
            selectedGroups.value = []
        } catch (error) {
            toast.add({ color: 'error', title: '导出失败' })
        }
    })
}

const handleBatchArchive = async () => {
    const confirmed = confirm(`确定要归档选中的 ${selectedGroups.value.length} 个群组吗？`)
    if (!confirmed) return
    
    await executeFeature('groupManage', async () => {
        try {
            // 这里应该调用实际的归档 API
            await new Promise(resolve => setTimeout(resolve, 1000)) // 模拟 API 调用
            
            toast.add({ color: 'success', title: `已归档 ${selectedGroups.value.length} 个群组` })
            selectedGroups.value = []
            
            // 刷新列表
            await loadOwnedGroups()
        } catch (error) {
            toast.add({ color: 'error', title: '归档失败' })
        }
    })
}

// 生命周期
onMounted(() => {
    loadOwnedGroups()
})
</script>
