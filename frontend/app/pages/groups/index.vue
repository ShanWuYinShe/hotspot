<template>
    <div class="w-full">

        <div class="space-y-4">
            <!-- 统计卡片 -->
            <div class="grid grid-cols-1 md:grid-cols-3 gap-3">
                <!-- 统计卡片：信息展示，不需要 hover -->
                <UCard variant="subtle">
                    <div class="flex items-center gap-4">
                        <UBadge color="primary" variant="soft" size="lg" square icon="i-lucide-users-round" />
                        <div class="min-w-0">
                            <p class="text-2xl font-bold">{{ totalGroups }}</p>
                            <UBadge color="neutral" variant="soft" size="xs">总群组数</UBadge>
                        </div>
                    </div>
                </UCard>

                <!-- 统计卡片：信息展示，不需要 hover -->
                <UCard variant="subtle">
                    <div class="flex items-center gap-4">
                        <UBadge color="success" variant="soft" size="lg" square icon="i-lucide-star" />
                        <div class="min-w-0">
                            <p class="text-2xl font-bold">{{ ownedGroups }}</p>
                            <UBadge color="neutral" variant="soft" size="xs">我创建的</UBadge>
                        </div>
                    </div>
                </UCard>

                <!-- 统计卡片：信息展示，不需要 hover -->
                <UCard variant="subtle">
                    <div class="flex items-center gap-4">
                        <UBadge color="warning" variant="soft" size="lg" square icon="i-lucide-bell" />
                        <div class="min-w-0">
                            <p class="text-2xl font-bold">{{ totalUnreadCount }}</p>
                            <UBadge color="neutral" variant="soft" size="xs">未读消息</UBadge>
                        </div>
                    </div>
                </UCard>
            </div>

            <UCard variant="subtle">
                <template #header>
                    <div class="flex flex-col sm:flex-row items-start sm:items-center justify-between gap-3">
                        <div class="flex items-center gap-3 w-full sm:w-auto">
                            <UInput
                                v-model="searchQuery"
                                placeholder="搜索群组..."
                                icon="i-lucide-search"
                                class="w-full sm:w-72"
                            />
                            <USelectMenu
                                v-model="activeFilter"
                                :items="filterOptions"
                                value-key="value"
                                class="w-40"
                            />
                        </div>
                    </div>
                </template>

                <div v-if="loading" class="space-y-3">
                    <USkeleton v-for="i in 3" :key="i" class="h-24 w-full" />
                </div>

                <UEmpty
                    v-else-if="displayGroups.length === 0"
                    icon="i-lucide-users-round"
                    title="还没有群组"
                    description="加入或创建您的第一个群组"
                    :actions="[{ label: '创建群组', icon: 'i-lucide-plus', onClick: openCreateGroup }]"
                />

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
import { useDashboardHeader } from '~/composables/useDashboardHeader'

// 从数据源获取 SEO 配置
const seoConfig = getPageSeoConfig('groups')

definePageMeta({
  title: seoConfig.title,
  description: seoConfig.description,
  layout: 'dashboard',
  requiresAuth: true,
  dashboardHeader: true
})

// 设置 SEO meta
useSeoMeta(buildSeoMeta('groups'))

// 服务和状态
const groupsService = useGroupsService()
const { formatTime } = useFormatters()
const feedback = useFeedbackStore()
const dashboardHeader = useDashboardHeader()

// 数据状态
const loading = ref(true)
const searchQuery = ref('')
const activeFilter = ref('all')
const myGroups = ref<JoinedGroupItem[]>([])

// 计算属性
const totalGroups = computed(() => myGroups.value.length)
const ownedGroups = computed(() => 
    myGroups.value.filter((g: JoinedGroupItem) => g.my_role === 'owner').length
)
const totalUnreadCount = computed(() =>
    myGroups.value.reduce((sum, g) => sum + (g.unread_count || 0), 0)
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
                groups = groups.filter(g => (g.unread_count || 0) > 0)
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
    // 这是“我的群组”页面：列表项天然已加入，必须显式标记，避免 GroupCard 误判展示“加入”
    is_member: true,
    // 已加入群组列表没有 geo 和 has_password，GroupCard 会自动处理
})


// 方法
const loadMyGroups = async () => {
    loading.value = true
    try {
        const response = await groupsService.getJoined()
        myGroups.value = response.items
    } catch (error: unknown) {
        feedback.error('加载失败', getErrorMessage(error))
    } finally {
        loading.value = false
    }
}

watchEffect(() => {
  dashboardHeader.setActions([
    {
      key: 'refresh',
      label: '刷新',
      icon: 'i-lucide-refresh-cw',
      variant: 'outline',
      loading: loading.value,
      action: () => loadMyGroups()
    },
    {
      key: 'create',
      label: '创建群组',
      icon: 'i-lucide-plus',
      variant: 'outline',
      disabled: loading.value,
      action: () => openCreateGroup()
    },
    {
      key: 'nearby',
      label: '附近',
      icon: 'i-lucide-map',
      variant: 'outline',
      disabled: loading.value,
      action: () => navigateTo('/groups/nearby')
    }
  ])
})

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

const modal = useModal()

const openCreateGroup = async () => {
    const groupId = await modal.showCreateGroup()
    if (groupId) {
        loadMyGroups()
        feedback.success('群组创建成功')
    }
}

const showGroupActions = (group: JoinedGroupItem) => {
    // 这里可以显示群组操作菜单
    feedback.info(`群组操作: ${group.name}`)
}


// 生命周期
onMounted(() => {
    loadMyGroups()
})
</script>