<template>
    <div class="w-full">
        <div class="mb-4">
            <StatsGrid :stats="friendsStats" :columns="4" />
        </div>

        <!-- 好友列表 -->
        <UCard variant="subtle">
            <!-- 工具栏 -->
            <template #header>
                <div class="flex flex-col sm:flex-row items-start sm:items-center justify-between gap-4">
                    <div class="flex items-center gap-4 w-full sm:w-auto">
                        <UInput
                            v-model="searchQuery"
                            placeholder="搜索好友..."
                            icon="i-lucide-search"
                            class="w-full sm:w-64"
                        >
                            <template #trailing>
                                <UButton
                                    v-show="searchQuery !== ''"
                                    color="neutral"
                                    variant="link"
                                    icon="i-lucide-x"
                                    :padded="false"
                                    @click="searchQuery = ''"
                                />
                            </template>
                        </UInput>
                        <USelectMenu
                            v-model="activeFilter"
                            :items="filterOptions"
                            option-attribute="label"
                            class="w-36"
                            variant="soft"
                        />
                    </div>
                </div>
            </template>

            <!-- 加载状态 -->
            <div v-if="loading" class="space-y-4">
                <USkeleton v-for="i in 3" :key="i" class="h-20 w-full" />
            </div>

            <!-- 空状态 -->
            <UEmpty
                v-else-if="displayFriends.length === 0"
                icon="i-lucide-users"
                title="还没有好友"
                description="添加您的第一个好友，开始社交之旅"
                class="py-12"
            >
                <template #actions>
                    <UButton color="primary" @click="navigateTo('/search')">添加好友</UButton>
                </template>
            </UEmpty>

            <!-- 好友列表 -->
            <div v-else class="space-y-3">
            <UserCard
                v-for="friend in displayFriends"
                :key="friend.user.user_id"
                :user="toUserCardData(friend)"
                @click="handleUserClick(friend)"
                @message="handleSendMessage(friend)"
            />
            </div>
        </UCard>
    </div>
</template>

<script setup lang="ts">
import type { UserInfo, FriendListItem } from "~/types/api"
import { useUsersService } from '~/services/users.service'
import StatsGrid from '~/components/data/StatsGrid.vue'
import { useDashboardHeader } from '~/composables/useDashboardHeader'

definePageMeta({ 
    title: "我的好友", 
    description: "互相关注的好友列表",
    requiresAuth: true, 
    layout: 'dashboard',
    dashboardHeader: true
})

// 服务和状态
const feedback = useFeedbackStore()
const userService = useUsersService()
const authStore = useAuthStore()
const dashboardHeader = useDashboardHeader()

// 数据状态
const loading = ref(true)
const friends = ref<FriendListItem[]>([])
const searchQuery = ref('')
const filterOptions = [
    { label: '全部好友', value: 'all' },
    { label: '在线好友', value: 'online' }
]

const activeFilter = ref(filterOptions[0])

// 计算属性
const totalFriends = computed(() => friends.value.length)
const onlineFriends = computed(() => 
    friends.value.filter(f => f.is_online === true).length
)

// 统计数据
const friendsStats = computed(() => [
    {
        key: 'total',
        label: '总好友数',
        value: totalFriends.value,
        icon: 'i-lucide-users',
        color: 'blue' as const,
        description: '互相关注的好友总数',
        onClick: () => {
            activeFilter.value = filterOptions[0]
            feedback.info('显示全部好友')
        }
    },
    {
        key: 'online',
        label: '在线好友',
        value: onlineFriends.value,
        icon: 'i-lucide-signal',
        color: 'green' as const,
        description: '当前在线的好友',
        trend: (onlineFriends.value > totalFriends.value * 0.3 ? 'up' : 'down') as 'up' | 'down',
        onClick: () => {
            activeFilter.value = filterOptions[1]
            feedback.info('显示在线好友')
        }
    }
])

watchEffect(() => {
    dashboardHeader.setActions([
        {
            key: 'refresh',
            label: '刷新',
            icon: 'i-lucide-refresh-cw',
            variant: 'outline',
            loading: loading.value,
            action: () => loadFriends()
        },
        {
            key: 'add',
            label: '发现用户',
            icon: 'i-lucide-user-plus',
            variant: 'outline',
            action: () => navigateTo('/search')
        },
        {
            key: 'nearby',
            label: '附近的人',
            icon: 'i-lucide-map-pin',
            color: 'neutral',
            variant: 'outline',
            action: () => navigateTo('/users/nearby')
        }
    ])
})

// 显示的好友（搜索 + 筛选）
const displayFriends = computed(() => {
    let result = [...friends.value]
    
    // 搜索过滤
    if (searchQuery.value.trim()) {
        const query = searchQuery.value.toLowerCase()
        result = result.filter(f => 
            f.user.nickname.toLowerCase().includes(query) ||
            (f.remark?.toLowerCase().includes(query))
        )
    }
    
    // 应用筛选
    switch (activeFilter.value?.value) {
        case 'online':
            result = result.filter(f => f.is_online === true)
            break
    }
    
    return result
})

// 转换好友数据为 UserCard 可用的格式
const toUserCardData = (friend: FriendListItem) => ({
    ...friend.user,
    nickname: friend.remark || friend.user.nickname, // 优先显示备注
    is_online: friend.is_online,
    // 好友列表没有 geo 数据，UserCard 会自动处理
})


// 方法
const loadFriends = async () => {
    loading.value = true
    try {
        const result = await userService.getFriends({})
        friends.value = result.items
        feedback.success('好友列表已更新')
    } catch (error: unknown) {
        feedback.error('加载失败', getErrorMessage(error))
    } finally {
        loading.value = false
    }
}

const handleUserClick = (friend: FriendListItem) => {
    if (friend.user.user_id === authStore.user?.user_id) {
        navigateTo('/profile')
    } else {
        navigateTo(`/users/${friend.user.user_id}`)
    }
}

const handleSendMessage = (friend: FriendListItem) => {
    navigateTo(`/chat/users/${friend.user.user_id}`)
}


// 生命周期
onMounted(async () => {
    await loadFriends()
})
</script>