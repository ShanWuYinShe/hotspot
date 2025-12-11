<template>
    <PageLayout
        title="我的好友"
        description="互相关注的好友列表"
        :stats="friendsStats"
        :header-actions="headerActions"
    >
        <!-- 好友列表 -->
        <UCard class="border">
            <!-- 工具栏 -->
            <template #header>
                <div class="flex flex-col sm:flex-row items-start sm:items-center justify-between gap-4">
                    <div class="flex items-center gap-4 w-full sm:w-auto">
                        <UInput
                            v-model="searchQuery"
                            placeholder="搜索好友..."
                            icon="i-heroicons-magnifying-glass"
                            class="w-full sm:w-64"
                        />
                        <USelectMenu
                            v-model="activeFilter"
                            :items="filterOptions"
                            value-key="value"
                            class="w-36"
                        />
                    </div>
                    <div class="flex items-center gap-2">
                        <UButton variant="outline" icon="i-heroicons-arrow-path" @click="loadFriends">
                            刷新
                        </UButton>
                        <UButton color="primary" icon="i-heroicons-user-plus" @click="navigateTo('/search')">
                            发现用户
                        </UButton>
                        <UButton variant="outline" icon="i-heroicons-map-pin" @click="navigateTo('/users/nearby')">
                            附近的人
                        </UButton>
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
                icon="i-heroicons-users"
                title="还没有好友"
                description="添加您的第一个好友，开始社交之旅"
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
                    @more="handleUserActions(friend)"
                />
            </div>
        </UCard>
    </PageLayout>
</template>

<script setup lang="ts">
import type { FriendListItem } from "~/types/api"

definePageMeta({ 
    title: "我的好友", 
    requiresAuth: true, 
    layout: 'dashboard' 
})

// 服务和状态
import { useUsersService } from '~/services/users.service'
const toast = useToast()
const userService = useUsersService()
const authStore = useAuthStore()

// 数据状态
const loading = ref(true)
const friends = ref<FriendListItem[]>([])
const searchQuery = ref('')
const activeFilter = ref('all')

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
        icon: 'i-heroicons-users',
        color: 'blue' as const,
        description: '互相关注的好友总数',
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
    }
])

// 头部操作
const headerActions = computed(() => [
    {
        key: 'add',
        label: '发现用户',
        icon: 'i-heroicons-user-plus',
        color: 'primary' as const,
        action: () => navigateTo('/search')
    },
    {
        key: 'following',
        label: '我的关注',
        icon: 'i-heroicons-user-group',
        variant: 'outline' as const,
        action: () => navigateTo('/following')
    }
])

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
    switch (activeFilter.value) {
        case 'online':
            result = result.filter(f => f.is_online === true)
            break
    }
    
    return result
})

// 筛选选项
const filterOptions = [
    { label: '全部好友', value: 'all' },
    { label: '在线好友', value: 'online' }
]

// 转换好友数据为 UserCard 可用的格式
const toUserCardData = (friend: FriendListItem) => ({
    ...friend.user,
    nickname: friend.remark || friend.user.nickname, // 优先显示备注
    is_online: friend.is_online,
    // 好友列表没有 geo 数据，UserCard 会自动处理
})


// 导入错误处理工具
import { getErrorMessage } from '~/utils/error'

// 方法
const loadFriends = async () => {
    loading.value = true
    try {
        const result = await userService.getFriends({})
        friends.value = result.items
        toast.add({ color: 'success', title: '好友列表已更新' })
    } catch (error: unknown) {
        toast.add({ color: 'error', title: '加载失败', description: getErrorMessage(error) })
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

const handleUserActions = (friend: FriendListItem) => {
    // 显示用户操作菜单
    toast.add({ color: 'info', title: `用户操作: ${friend.user.nickname}` })
}


// 生命周期
onMounted(async () => {
    await loadFriends()
})
</script>