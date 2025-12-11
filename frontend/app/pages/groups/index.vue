<template>
    <div class="min-h-screen bg-gradient-slate-page">
        <div class="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-4 sm:py-6">
            <!-- 页面标题 -->
            <UPageHeader
                title="我的群组"
                description="管理您加入和创建的所有群组"
                class="mb-6"
            />

            <!-- 统计卡片 -->
            <div class="grid grid-cols-1 md:grid-cols-3 gap-6 mb-8">
                <UCard class=" border ">
                    <div class="flex items-center gap-4">
                        <div class="p-3 bg-primary-base rounded-lg">
                            <UIcon name="i-heroicons-user-group" class="icon-lg text-secondary-strong" />
                        </div>
                        <div>
                            <p class="text-2xl font-bold text-primary-content">{{ totalGroups }}</p>
                            <p class="text-sm text-secondary-content">总群组数</p>
                        </div>
                    </div>
                </UCard>
                
                <UCard class=" border ">
                    <div class="flex items-center gap-4">
                        <div class="p-3 bg-success-base rounded-lg">
                            <UIcon name="i-heroicons-star" class="icon-lg text-success" />
                        </div>
                        <div>
                            <p class="text-2xl font-bold text-primary-content">{{ ownedGroups }}</p>
                            <p class="text-sm text-secondary-content">我创建的</p>
                        </div>
                    </div>
                </UCard>
                
                <UCard class=" border ">
                    <div class="flex items-center gap-4">
                        <div class="p-3 bg-warning-base rounded-lg">
                            <UIcon name="i-heroicons-bell" class="icon-lg text-warning" />
                        </div>
                        <div>
                            <p class="text-2xl font-bold text-primary-content">{{ totalUnreadCount }}</p>
                            <p class="text-sm text-secondary-content">未读消息</p>
                        </div>
                    </div>
                </UCard>
            </div>

            <!-- 群组列表 -->
            <UCard class="border">
                <!-- 工具栏 -->
                <template #header>
                    <div class="flex flex-col sm:flex-row items-start sm:items-center justify-between gap-4">
                        <div class="flex items-center gap-4 w-full sm:w-auto">
                            <UInput
                                v-model="searchQuery"
                                placeholder="搜索群组..."
                                icon="i-heroicons-magnifying-glass"
                                class="w-full sm:w-64"
                            />
                            <USelectMenu
                                v-model="activeFilter"
                                :items="filterOptions"
                                value-key="value"
                                class="w-40"
                            />
                        </div>
                        <div class="flex items-center gap-2">
                            <UButton variant="outline" icon="i-heroicons-arrow-path" @click="loadMyGroups">
                                刷新
                            </UButton>
                            <UButton color="primary" icon="i-heroicons-plus" @click="openCreateGroup">
                                创建群组
                            </UButton>
                            <UButton variant="outline" icon="i-heroicons-magnifying-glass" @click="navigateTo('/groups/discover')">
                                发现群组
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
                    title="还没有群组"
                    description="加入或创建您的第一个群组"
                >
                    <template #actions>
                        <UButton color="primary" @click="openCreateGroup">创建群组</UButton>
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
                        @more="showGroupActions(group)"
                    />
                </div>
            </UCard>
        </div>
    </div>
</template>

<script setup lang="ts">
import { getPageSeoConfig, buildSeoMeta } from '~/config/seo'
import type { JoinedGroupItem } from "~/types/api"
import { useGroupsService } from '~/services/groups.service'
import { getErrorMessage } from '~/utils/error'

// 从数据源获取 SEO 配置
const seoConfig = getPageSeoConfig('groups')

definePageMeta({ 
    title: seoConfig.title, 
    description: seoConfig.description,
    layout: 'dashboard',
    requiresAuth: true  // 使用全局中间件处理认证
})

// 设置 SEO meta
useSeoMeta(buildSeoMeta('groups'))

// 服务和状态
const groupsService = useGroupsService()
const { formatTime } = useFormatters()
const toast = useToast()

// 数据状态
const loading = ref(true)
const searchQuery = ref('')
const activeFilter = ref('all')
const unreadCounts = reactive<Record<string, number>>({})
const myGroups = ref<JoinedGroupItem[]>([])

// 计算属性
const totalGroups = computed(() => myGroups.value.length)
const ownedGroups = computed(() => 
    myGroups.value.filter((g: JoinedGroupItem) => g.my_role === 'owner').length
)
const totalUnreadCount = computed(() => 
    Object.values(unreadCounts).reduce((sum, count) => sum + count, 0)
)

// 显示的群组（搜索 + 筛选）
const displayGroups = computed(() => {
    let groups = [...myGroups.value]
    
    // 搜索过滤（只搜索名称，JoinedGroupItem 没有 description 字段）
    if (searchQuery.value.trim()) {
        const query = searchQuery.value.toLowerCase()
        groups = groups.filter(g => g.name.toLowerCase().includes(query))
    }
    
    // 应用筛选
    if (activeFilter.value !== 'all') {
        switch (activeFilter.value) {
            case 'owned':
                groups = groups.filter(g => g.my_role === 'owner')
                break
            case 'joined':
                groups = groups.filter(g => g.my_role !== 'owner')
                break
            case 'unread':
                groups = groups.filter(g => unreadCounts[g.id] > 0)
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

// 转换群组数据为 GroupCard 可用的格式
const toGroupCardData = (group: JoinedGroupItem) => ({
    ...group,
    unread_count: unreadCounts[group.id] || 0,
    // 已加入群组列表没有 geo 和 has_password，GroupCard 会自动处理
})


// 方法
const loadMyGroups = async () => {
    loading.value = true
    try {
        const response = await groupsService.getJoined()
        myGroups.value = response.items
        await loadUnreadCounts()
    } catch (error: unknown) {
        toast.add({ color: 'error', title: '加载失败', description: getErrorMessage(error) })
    } finally {
        loading.value = false
    }
}

const loadUnreadCounts = async () => {
    try {
        // 暂时禁用未读数获取，需要WebSocket支持
        const groups = myGroups.value.slice(0, 30) // 限制数量

        // 批量获取未读数
        const unreadPromises = groups.map(async (group) => {
            const count = 0 // 暂时返回0
            return { groupId: group.id, count }
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

const enterGroup = async (group: JoinedGroupItem) => {
    try {
        // 清零本地未读数
        unreadCounts[group.id] = 0
        
        // 跳转到群聊页面（消息已读标记会在聊天页面中自动处理）
        await navigateTo(`/chat/groups/${group.id}`)
    } catch (error) {
        console.error('进入群组失败:', error)
        // 即使出错，也允许进入群组
        await navigateTo(`/chat/groups/${group.id}`)
    }
}

const modal = useModal()

const openCreateGroup = async () => {
    const groupId = await modal.showCreateGroup()
    if (groupId) {
        loadMyGroups()
        toast.add({ color: 'success', title: '群组创建成功' })
    }
}

const showGroupActions = (group: JoinedGroupItem) => {
    // 这里可以显示群组操作菜单
    toast.add({ color: 'info', title: `群组操作: ${group.name}` })
}


// 生命周期
onMounted(() => {
    loadMyGroups()
})
</script>