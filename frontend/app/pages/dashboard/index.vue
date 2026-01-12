<template>
    <div class="flex flex-col gap-6">
        <!-- Header -->
        <div class="flex items-center justify-between">
            <div>
                <h1 class="text-2xl font-bold text-neutral-900 dark:text-white">数据仪表板</h1>
                <p class="text-neutral-500 dark:text-neutral-400 mt-1">查看您的活动统计和趋势分析</p>
            </div>
            <div class="flex gap-2">
                <UButton
                    label="刷新"
                    icon="i-lucide-refresh-cw"
                    variant="outline"
                    color="neutral"
                    :loading="loading"
                    @click="loadOverviewStats"
                />
                <UButton
                    label="设置"
                    icon="i-lucide-settings"
                    variant="ghost"
                    color="neutral"
                    to="/settings"
                />
            </div>
        </div>

        <div class="space-y-8">
            <!-- 概览卡片 -->
            <StatsGrid
                :stats="overviewStats"
                :columns="4"
            />

            <!-- 快速操作 -->
            <div>
                <h3 class="font-semibold text-lg mb-4 px-1 text-neutral-900 dark:text-white">常用操作</h3>
                <div class="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-4">
                    <UCard
                        v-for="action in quickActions"
                        :key="action.key"
                        variant="outline"
                        @click="action.action"
                    >
                        <div class="flex items-center gap-3">
                            <UAvatar
                                :icon="action.icon"
                                size="md"
                                :color="action.color"
                                variant="soft"
                            />
                            <div class="flex-1 min-w-0">
                                <div class="font-medium text-neutral-900 dark:text-white truncate">
                                    {{ action.title }}
                                </div>
                                <div class="text-xs text-neutral-500 dark:text-neutral-400 mt-0.5 truncate">
                                    {{ action.description }}
                                </div>
                            </div>
                            <UIcon name="i-lucide-arrow-right" class="size-4 text-neutral-400" />
                        </div>
                    </UCard>
                </div>
            </div>
        </div>

        <!-- 模态框 -->
        <CreateGroupModal
            v-model:open="showCreateGroupModal"
            @success="handleCreateGroupSuccess"
        />
        <CreateMomentModal
            v-model:open="showCreateMomentModal"
            @success="handleCreateMomentSuccess"
        />
    </div>
</template>

<script setup lang="ts">
import type { StatItem } from '~/components/data/StatsGrid.vue'
import StatsGrid from '~/components/data/StatsGrid.vue'
import { useUsersService } from '~/services/users.service'
import type { UserStatsData } from '~/types/api'
import CreateGroupModal from '~/components/modals/CreateGroupModal.vue'
import CreateMomentModal from '~/components/modals/CreateMomentModal.vue'

definePageMeta({
    layout: 'dashboard',
    requiresAuth: true
})

const feedback = useFeedbackStore()
const userService = useUsersService()
const router = useRouter()

const loading = ref(false)
const userStats = ref<UserStatsData>({
    moments_count: 0,
    friends_count: 0,
    followers_count: 0,
    following_count: 0,
    groups_count: 0
})

const showCreateGroupModal = ref(false)
const showCreateMomentModal = ref(false)

const overviewStats = computed<StatItem[]>(() => [
    {
        key: 'friends',
        label: '好友',
        value: userStats.value.friends_count,
        icon: 'i-lucide-users',
        color: 'blue',
        description: '互相关注的好友',
        onClick: () => navigateTo('/friends')
    },
    {
        key: 'groups',
        label: '群组',
        value: userStats.value.groups_count,
        icon: 'i-lucide-users-round',
        color: 'green',
        description: '加入的群组',
        onClick: () => navigateTo('/groups')
    },
    {
        key: 'following',
        label: '关注',
        value: userStats.value.following_count,
        icon: 'i-lucide-user-plus',
        color: 'indigo',
        description: '关注的用户',
        onClick: () => navigateTo('/follows')
    },
    {
        key: 'moments',
        label: '动态',
        value: userStats.value.moments_count,
        icon: 'i-lucide-image',
        color: 'orange',
        description: '发布的动态',
        onClick: () => navigateTo('/moments')
    }
])

const quickActions = [
    {
        key: 'create-group',
        title: '创建群组',
        description: '建立新的群组',
        icon: 'i-lucide-circle-plus',
        color: 'info' as const,
        action: () => showCreateGroupModal.value = true
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
        action: () => showCreateMomentModal.value = true
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

onMounted(loadOverviewStats)

async function loadOverviewStats() {
    loading.value = true
    try {
        userStats.value = await userService.getMyStats()
    } catch (err) {
        feedback.error('数据加载失败')
    } finally {
        loading.value = false
    }
}

function handleCreateGroupSuccess(groupId: string) {
    showCreateGroupModal.value = false
    feedback.success('群组创建成功')
    router.push(`/chat/groups/${groupId}`)
}

function handleCreateMomentSuccess() {
    showCreateMomentModal.value = false
    feedback.success('动态发布成功')
    loadOverviewStats()
}
</script>
