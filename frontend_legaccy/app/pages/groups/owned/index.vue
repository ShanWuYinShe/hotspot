<template>
    <div class="flex flex-col gap-6">
        <!-- Header -->
        <div class="flex items-center justify-between">
            <div>
                <h1 class="text-2xl font-bold text-neutral-900 dark:text-white">我创建的群组</h1>
                <p class="text-neutral-500 dark:text-neutral-400 mt-1">管理您创建的所有群组</p>
            </div>
            <UButton
                variant="ghost"
                color="neutral"
                icon="i-lucide-arrow-left"
                to="/groups"
            >
                返回列表
            </UButton>
        </div>

        <div class="space-y-6">
            <!-- 统计卡片 -->
            <StatsGrid :stats="groupStats" :columns="4" />

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
                        <UButton icon="i-lucide-plus" @click="openCreateModal">创建群组</UButton>
                    </div>
                </template>

                <!-- 加载状态 -->
                <div v-if="loading" class="space-y-4">
                    <USkeleton v-for="i in 3" :key="i" shape="rectangle" class="h-24 w-full" />
                </div>

                <!-- 空状态 -->
                <UEmpty
                    v-else-if="displayGroups.length === 0"
                    icon="i-lucide-users-round"
                    title="还没有创建群组"
                    description="创建您的第一个群组，开始与朋友们聊天"
                    class="py-12"
                >
                    <template #actions>
                        <UButton color="primary" @click="openCreateModal">创建群组</UButton>
                    </template>
                </UEmpty>

                <!-- 群组列表 -->
                <div v-else class="space-y-3">
                    <GroupCard
                        v-for="group in displayGroups"
                        :key="group.group_chat_id"
                        :group="toGroupCardData(group)"
                        @click="enterGroup(group)"
                        @join="enterGroup(group)"
                    />
                </div>
            </UCard>
        </div>

        <!-- 创建群组模态框 -->
        <CreateGroupModal
            v-model:open="showCreateModal"
            @success="handleCreateSuccess"
        />
    </div>
</template>

<script setup lang="ts">
import type { JoinedGroupItem } from "~/types/api"
import { useGroupsService } from '~/services/groups.service'
import StatsGrid from '~/components/data/StatsGrid.vue'
import GroupCard from '~/components/cards/GroupCard.vue'
import CreateGroupModal from '~/components/modals/CreateGroupModal.vue'

definePageMeta({ 
    layout: 'dashboard',
    requiresAuth: true
})

const groupsService = useGroupsService()
const { formatTime } = useFormatters()
const feedback = useFeedbackStore()
const router = useRouter()

// 状态
const items = ref<JoinedGroupItem[]>([])
const loading = ref(true)
const searchQuery = ref('')
const showCreateModal = ref(false)

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

const groupStats = computed(() => [
    {
        key: 'total',
        label: '创建的群组',
        value: totalGroups.value,
        icon: 'i-lucide-users-round',
        color: 'blue' as const,
        description: '您创建的群组总数'
    },
    {
        key: 'active',
        label: '活跃群组',
        value: activeGroups.value,
        icon: 'i-lucide-flame',
        color: 'green' as const,
        description: '24小时内有活动的群组'
    },
    {
        key: 'members',
        label: '总成员数',
        value: totalMembers.value,
        icon: 'i-lucide-users',
        color: 'purple' as const,
        description: '所有群组的成员总数'
    },
    {
        key: 'unread',
        label: '未读消息',
        value: totalUnread.value,
        icon: 'i-lucide-bell',
        color: 'orange' as const,
        description: '所有群组的未读消息'
    }
])

const displayGroups = computed(() => {
    if (!searchQuery.value.trim()) return items.value
    const query = searchQuery.value.toLowerCase()
    return items.value.filter(g => g.name.toLowerCase().includes(query))
})

const toGroupCardData = (group: JoinedGroupItem) => ({
    ...group,
    is_member: true
})

onMounted(() => {
    loadOwnedGroups()
})

const loadOwnedGroups = async () => {
    loading.value = true
    try {
        const response = await groupsService.getJoined()
        items.value = response.items.filter(group => group.my_role === 'owner')
    } catch (error: unknown) {
        feedback.error('加载失败')
    } finally {
        loading.value = false
    }
}

const enterGroup = async (group: JoinedGroupItem) => {
    router.push(`/chat/groups/${group.group_chat_id}`)
}

const openCreateModal = () => {
    showCreateModal.value = true
}

const handleCreateSuccess = (group: JoinedGroupItem) => {
    showCreateModal.value = false
    items.value.unshift(group)
    feedback.success('群组创建成功')
    router.push(`/chat/groups/${group.group_chat_id}`)
}
</script>
