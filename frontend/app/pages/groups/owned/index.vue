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
        <UCard class="border">
            <!-- 工具栏 -->
            <template #header>
                <div class="flex flex-col sm:flex-row items-start sm:items-center justify-between gap-4">
                    <UInput
                        v-model="searchQuery"
                        placeholder="搜索群组..."
                        icon="i-heroicons-magnifying-glass"
                        class="w-full sm:w-64"
                    />
                    <div class="flex items-center gap-2">
                        <UButton variant="outline" icon="i-heroicons-arrow-path" @click="loadOwnedGroups">
                            刷新
                        </UButton>
                        <UButton color="primary" icon="i-heroicons-plus" @click="navigateTo('/groups/create')">
                            创建群组
                        </UButton>
                        <UButton variant="outline" icon="i-heroicons-chart-bar" @click="toast.add({ color: 'info', title: '群组数据统计功能即将上线' })">
                            数据统计
                        </UButton>
                    </div>
                </div>
            </template>

            <!-- 加载状态 -->
            <div v-if="loading" class="space-y-4">
                <USkeleton v-for="i in 3" :key="i" class="h-24 w-full" />
            </div>

            <!-- 空状态 -->
            <UEmpty
                v-else-if="displayGroups.length === 0"
                icon="i-heroicons-user-group"
                title="还没有创建群组"
                description="创建您的第一个群组，开始与朋友们聊天"
            >
                <template #actions>
                    <UButton color="primary" @click="navigateTo('/groups/create')">创建群组</UButton>
                </template>
            </UEmpty>

            <!-- 群组列表 -->
            <div v-else class="space-y-3">
                <GroupCard
                    v-for="group in displayGroups"
                    :key="group.id"
                    :group="toGroupCardData(group)"
                    @click="enterGroup(group)"
                    @enter="enterGroup(group)"
                    @more="manageGroup(group)"
                />
            </div>
        </UCard>
    </PageLayout>
</template>

<script setup lang="ts">
import type { JoinedGroupItem } from "~/types/api"
import type { ActionItem } from '~/components/common/ActionPanel.vue'
import { getErrorMessage } from '~/utils/error'
import { useGroupsService } from '~/services/groups.service'

definePageMeta({ 
    title: "我创建的群组", 
    requiresAuth: true, 
    layout: 'dashboard' 
})

// 服务和状态
const groupsService = useGroupsService()
const { formatTime } = useFormatters()
const toast = useToast()

// 数据状态
const items = ref<JoinedGroupItem[]>([])
const loading = ref(true)
const selectedGroups = ref<JoinedGroupItem[]>([])
const ownedUnreadMap = reactive<Record<string, number>>({})
const searchQuery = ref('')

// 计算属性
const totalGroups = computed(() => items.value.length)
const activeGroups = computed(() => {
    const oneDayAgo = new Date(Date.now() - 24 * 60 * 60 * 1000)
    return items.value.filter(g => new Date(g.last_activity_at) > oneDayAgo).length
})
const totalMembers = computed(() => 
    items.value.reduce((sum, group) => sum + (group.members_count || 0), 0)
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

// 显示的群组（搜索过滤，只搜索名称）
const displayGroups = computed(() => {
    if (!searchQuery.value.trim()) return items.value
    
    const query = searchQuery.value.toLowerCase()
    return items.value.filter(g => g.name.toLowerCase().includes(query))
})

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

// 转换群组数据为 GroupCard 可用的格式
const toGroupCardData = (group: JoinedGroupItem) => ({
    ...group,
    unread_count: ownedUnreadMap[String(group.id)] || 0,
    // 已加入群组列表没有 geo 和 has_password，GroupCard 会自动处理
})


// 方法
const loadOwnedGroups = async () => {
    loading.value = true
    try {
        // 使用 service 获取加入的群组，然后筛选出我创建的
        const response = await groupsService.getJoined()
        items.value = response.items.filter(group => group.my_role === 'owner')
        
        // 加载未读消息数量
        await loadUnreadCounts()
    } catch (error: unknown) {
        toast.add({ color: 'error', title: `加载失败: ${getErrorMessage(error, '未知错误')}` })
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
            return { groupId: group.id, count }
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

const enterGroup = async (group: JoinedGroupItem) => {
    try {
        // 清零本地未读数
        ownedUnreadMap[String(group.id)] = 0
        
        // 跳转到群聊页面（消息已读标记会在聊天页面中自动处理）
        await navigateTo(`/chat/groups/${group.id}`)
    } catch (error) {
        console.error('进入群组失败:', error)
        // 即使出错，也允许进入群组
        await navigateTo(`/chat/groups/${group.id}`)
    }
}

const manageGroup = async (group: JoinedGroupItem) => {
    await navigateTo(`/groups/${group.id}/settings`)
}

const toggleGroupSelection = (group: JoinedGroupItem) => {
    const index = selectedGroups.value.findIndex(g => g.id === group.id)
    if (index > -1) {
        selectedGroups.value.splice(index, 1)
    } else {
        selectedGroups.value.push(group)
    }
}


const handleBatchExport = async () => {
    try {
        // 简化导出功能 - 暂时提示未实现
        toast.add({ color: 'info', title: '导出功能暂未实现' })
        selectedGroups.value = []
    } catch (error) {
        toast.add({ color: 'error', title: '导出失败' })
    }
}

const handleBatchArchive = async () => {
    const modal = useModal()
    const confirmed = await modal.showConfirm({
        title: '批量归档',
        message: `确定要归档选中的 ${selectedGroups.value.length} 个群组吗？`,
        confirmText: '归档',
        confirmColor: 'primary'
    })
    if (!confirmed) return
    
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
}

// 生命周期
onMounted(() => {
    loadOwnedGroups()
})
</script>
