<template>
    <div class="w-full">
        <div class="mb-4">
            <StatsGrid :stats="groupStats" :columns="4" />
        </div>

        <!-- 群组列表 -->
        <UCard variant="subtle">
            <!-- 工具栏 -->
            <template #header>
                <div class="flex flex-col sm:flex-row items-start sm:items-center justify-between gap-4">
                    <UInput
                        v-model="searchQuery"
                        placeholder="搜索群组..."
                        icon="i-lucide-search"
                        class="w-full sm:w-64"
                    />
                </div>
            </template>

            <!-- 加载状态 -->
            <div v-if="loading" class="space-y-4">
                <USkeleton v-for="i in 3" :key="i" class="h-24 w-full" />
            </div>

            <!-- 空状态 -->
            <UEmpty
                v-else-if="displayGroups.length === 0"
                icon="i-lucide-users-round"
                title="还没有创建群组"
                description="创建您的第一个群组，开始与朋友们聊天"
            >
                <template #actions>
                    <UButton color="primary" @click="createGroup">创建群组</UButton>
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
    </div>
</template>

<script setup lang="ts">
import type { JoinedGroupItem } from "~/types/api"
import { useGroupsService } from '~/services/groups.service'
import StatsGrid from '~/components/data/StatsGrid.vue'
import { useDashboardHeader } from '~/composables/useDashboardHeader'

definePageMeta({ 
    title: "我创建的群组", 
    description: "管理您创建的所有群组",
    requiresAuth: true, 
    layout: 'dashboard',
    dashboardHeader: true
})

// 服务和状态
const groupsService = useGroupsService()
const { formatTime } = useFormatters()
const feedback = useFeedbackStore()
const modal = useModal()
const dashboardHeader = useDashboardHeader()

// 数据状态
const items = ref<JoinedGroupItem[]>([])
const loading = ref(true)
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
    items.value.reduce((sum, g) => sum + (g.unread_count || 0), 0)
)

// 统计数据
const groupStats = computed(() => [
    {
        key: 'total',
        label: '创建的群组',
        value: totalGroups.value,
        icon: 'i-lucide-users-round',
        color: 'blue' as const,
        description: '您创建的群组总数',
        onClick: () => feedback.info('显示全部群组')
    },
    {
        key: 'active',
        label: '活跃群组',
        value: activeGroups.value,
        icon: 'i-lucide-flame',
        color: 'green' as const,
        description: '24小时内有活动的群组',
        trend: (activeGroups.value > totalGroups.value * 0.5 ? 'up' : 'down') as 'up' | 'down',
        onClick: () => feedback.info('显示活跃群组')
    },
    {
        key: 'members',
        label: '总成员数',
        value: totalMembers.value,
        icon: 'i-lucide-users',
        color: 'purple' as const,
        description: '所有群组的成员总数',
        onClick: () => feedback.info('查看成员统计')
    },
    {
        key: 'unread',
        label: '未读消息',
        value: totalUnread.value,
        icon: 'i-lucide-bell',
        color: 'orange' as const,
        description: '所有群组的未读消息',
        onClick: () => feedback.info('查看未读消息')
    }
])

// 显示的群组（搜索过滤，只搜索名称）
const displayGroups = computed(() => {
    if (!searchQuery.value.trim()) return items.value
    
    const query = searchQuery.value.toLowerCase()
    return items.value.filter(g => g.name.toLowerCase().includes(query))
})

watchEffect(() => {
    dashboardHeader.setActions([
        {
            key: 'refresh',
            label: '刷新',
            icon: 'i-lucide-refresh-cw',
            variant: 'outline',
            loading: loading.value,
            action: () => loadOwnedGroups()
        },
        {
            key: 'create',
            label: '创建群组',
            icon: 'i-lucide-plus',
            variant: 'outline',
            action: () => createGroup()
        }
    ])
})

// 转换群组数据为 GroupCard 可用的格式
const toGroupCardData = (group: JoinedGroupItem) => ({
    ...group,
    // 这是“我创建的群组”页面：列表项天然已加入，必须显式标记，避免 GroupCard 误判展示“加入”
    is_member: true,
    // JoinedGroupItem 已包含 unread_count/last_activity_at 等信息
})


// 方法
const loadOwnedGroups = async () => {
    loading.value = true
    try {
        // 使用 service 获取加入的群组，然后筛选出我创建的
        const response = await groupsService.getJoined()
        items.value = response.items.filter(group => group.my_role === 'owner')
    } catch (error: unknown) {
        feedback.error('加载失败', getErrorMessage(error, '未知错误'))
    } finally {
        loading.value = false
    }
}

const enterGroup = async (group: JoinedGroupItem) => {
    try {
        // 跳转到群聊页面（消息已读标记会在聊天页面中自动处理）
        await navigateTo(`/chat/groups/${group.id}`)
    } catch (error) {
        console.error('进入群组失败:', error)
        // 即使出错，也允许进入群组
        await navigateTo(`/chat/groups/${group.id}`)
    }
}

const manageGroup = async (group: JoinedGroupItem) => {
    await modal.showGroupSettings(String(group.id))
}

const createGroup = async () => {
    const id = await modal.showCreateGroup()
    if (id) await navigateTo(`/groups/${id}`)
}

// 生命周期
onMounted(() => {
    loadOwnedGroups()
})
</script>
