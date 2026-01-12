<template>
    <div class="flex flex-col gap-6">
        <!-- Sticky Header & Controls -->
        <div class="sticky top-0 z-20 bg-white/95 dark:bg-neutral-900/95 backdrop-blur-sm border-b border-neutral-200 dark:border-neutral-800 -mx-4 sm:-mx-6 px-4 sm:px-6 py-4 space-y-4">
            <!-- Header -->
            <div>
                <h1 class="text-2xl font-bold text-neutral-900 dark:text-white">{{ $t('groups.title') }}</h1>
                <p class="text-neutral-500 dark:text-neutral-400 mt-1">{{ $t('groups.description') }}</p>
            </div>
            
            <!-- Tabs -->
            <div class="flex gap-2 border-b border-neutral-200 dark:border-neutral-800">
                <NuxtLink
                    v-for="link in headerLinks"
                    :key="link.to"
                    :to="link.to"
                    class="px-4 py-2 text-sm font-medium transition-colors border-b-2 -mb-[2px]"
                    :class="[
                        link.active 
                            ? 'text-primary-600 border-primary-600 dark:text-primary-400 dark:border-primary-400' 
                            : 'text-neutral-500 border-transparent hover:text-neutral-700 dark:text-neutral-400 dark:hover:text-neutral-200'
                    ]"
                >
                    <div class="flex items-center gap-2">
                        <UIcon :name="link.icon" class="size-4" />
                        {{ link.label }}
                    </div>
                </NuxtLink>
            </div>

            <!-- 操作栏 -->
            <div class="flex flex-col sm:flex-row gap-4">
                <UInput
                    v-model="searchQuery"
                    :placeholder="$t('groups.search.placeholder')"
                    icon="i-lucide-search"
                    class="flex-1"
                    :loading="searching"
                    @update:model-value="handleSearch"
                />
                
                <div class="flex gap-2 shrink-0">
                    <UButton
                        to="/groups/nearby"
                        icon="i-lucide-map-pin"
                        variant="soft"
                        color="neutral"
                    >
                        {{ $t('groups.actions.nearby') }}
                    </UButton>
                    <UButton
                        icon="i-lucide-plus"
                        @click="openCreateModal"
                    >
                        {{ $t('groups.actions.create') }}
                    </UButton>
                </div>
            </div>
        </div>

        <div class="space-y-8 pb-12">

            <!-- 推荐群组 -->
            <div v-if="!searchQuery && recommendedGroups.length > 0">
                <div class="flex items-center justify-between mb-4">
                    <h2 class="font-semibold text-lg flex items-center gap-2">
                        <UIcon name="i-lucide-flame" class="text-primary-500" />
                        {{ $t('groups.recommended') }}
                    </h2>
                </div>
                
                <div class="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4 gap-6">
                    <GroupCard
                        v-for="group in recommendedGroups"
                        :key="group.group_chat_id"
                        :group="group"
                        @click="handleGroupClick"
                        @join="handleJoin"
                    />
                </div>
            </div>

            <!-- 搜索结果 -->
            <div v-if="searchQuery">
                <h2 class="font-semibold text-lg mb-4">{{ $t('groups.searchResults') }}</h2>
                <div v-if="searchResults.length > 0" class="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4 gap-6">
                    <GroupCard
                        v-for="group in searchResults"
                        :key="group.group_chat_id"
                        :group="group"
                        @click="handleGroupClick"
                        @join="handleJoin"
                    />
                </div>
                <UEmpty
                    v-else
                    icon="i-lucide-search-x"
                    :title="$t('groups.search.empty.title')"
                    :description="$t('groups.search.empty.description')"
                />
            </div>

            <!-- 我的群组 -->
            <div v-if="!searchQuery">
                <div class="flex items-center justify-between mb-4">
                    <h2 class="font-semibold text-lg">{{ $t('groups.myGroups.title') }}</h2>
                </div>
                
                <div v-if="loadingMyGroups" class="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4 gap-6">
                    <GroupCardSkeleton v-for="i in 4" :key="i" />
                </div>
                
                <div v-else-if="myGroups.length > 0" class="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4 gap-6">
                    <GroupCard
                        v-for="group in myGroups"
                        :key="group.group_chat_id"
                        :group="group"
                        @click="handleGroupClick"
                    />
                </div>
                
                <UEmpty
                    v-else
                    icon="i-lucide-users"
                    :title="$t('groups.myGroups.empty.title')"
                    :description="$t('groups.myGroups.empty.description')"
                >
                    <template #actions>
                        <UButton variant="soft" @click="openCreateModal">{{ $t('groups.actions.createFirst') }}</UButton>
                    </template>
                </UEmpty>
            </div>
        </div>

        <!-- 创建群组模态框 -->
        <CreateGroupModal
            v-model:open="showCreateModal"
            :on-success="handleCreateSuccess"
        />
    </div>
</template>

<script setup lang="ts">
import { useGroupsService } from '~/services/groups.service'
import type { GroupInfo, JoinedGroupItem, GroupItem } from '~/types/api'
import GroupCard from '~/components/cards/GroupCard.vue'
import GroupCardSkeleton from '~/components/skeletons/GroupCardSkeleton.vue'
import CreateGroupModal from '~/components/modals/CreateGroupModal.vue'

const { requireAuth } = useRequireAuth()

definePageMeta({
    layout: 'dashboard',
    requiresAuth: true
})

const { t } = useI18n()
const router = useRouter()
const groupsService = useGroupsService()
const feedback = useFeedbackStore()
const geolocation = useGeolocation()

// 状态
const searchQuery = ref('')
const searching = ref(false)
const showCreateModal = ref(false)
const loadingMyGroups = ref(true)

// 数据
const myGroups = ref<JoinedGroupItem[]>([])
const recommendedGroups = ref<GroupItem[]>([])
const searchResults = ref<GroupItem[]>([])

// 计算属性
const headerLinks = computed(() => [
    { label: t('groups.tabs.myGroups'), icon: 'i-lucide-users', to: '/groups', active: true },
    { label: t('groups.tabs.nearby'), icon: 'i-lucide-map-pin', to: '/groups/nearby' }
])

// 初始化
onMounted(async () => {
    await Promise.all([
        fetchMyGroups(),
        fetchRecommendedGroups()
    ])
})

// 获取我的群组
async function fetchMyGroups() {
    loadingMyGroups.value = true
    try {
        const res = await groupsService.getJoined({ limit: 8 })
        myGroups.value = res.items
    } catch (e) {
        console.error('Failed to fetch my groups', e)
    } finally {
        loadingMyGroups.value = false
    }
}

// 获取推荐群组
async function fetchRecommendedGroups() {
    try {
        const coords = await geolocation.getCoordinates()
        // 使用 searchByLocation 模拟附近推荐
        const res = await groupsService.searchByLocation({
            reference_point: {
                latitude: coords.latitude,
                longitude: coords.longitude,
            },
            radius: 10000,
            limit: 4
        })
        recommendedGroups.value = res.items
    } catch (e) {
        // error
    }
}

// 搜索处理
const handleSearch = useDebounceFn(async (query: string) => {
    if (!query.trim()) {
        searchResults.value = []
        return
    }
    
    searching.value = true
    try {
        const coords = await geolocation.getCoordinates()
        const res = await groupsService.searchByKeyword({
            kw: query,
            reference_point: {
                latitude: coords.latitude,
                longitude: coords.longitude,
            },
        })
        searchResults.value = res.items
    } catch (e) {
        feedback.error('搜索失败')
    } finally {
        searching.value = false
    }
}, 300)

// 交互处理
function handleGroupClick(group: GroupItem) {
    if (group.my_state?.is_joined) {
        router.push(`/chat/groups/${group.group_chat_id}`)
    } else {
        router.push(`/groups/${group.group_chat_id}`)
    }
}

function handleJoin(group: GroupItem) {
    // 符合设计理念：只在执行操作时才要求登录，提供友好的登录引导
    if (!requireAuth('加入群组')) return
    router.push(`/groups/${group.group_chat_id}`)
}

function openCreateModal() {
    showCreateModal.value = true
}

async function handleCreateSuccess(groupId: string) {
    showCreateModal.value = false
    feedback.success('群组创建成功')
    // 刷新我的群组列表
    await fetchMyGroups()
    router.push(`/chat/groups/${groupId}`)
}
</script>
