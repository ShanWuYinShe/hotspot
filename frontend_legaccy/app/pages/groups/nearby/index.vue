<template>
    <div class="flex flex-col gap-6">
        <!-- Sticky Header & Filter -->
        <div class="sticky top-0 z-20 bg-white/95 dark:bg-neutral-900/95 backdrop-blur-sm border-b border-neutral-200 dark:border-neutral-800 -mx-4 sm:-mx-6 px-4 sm:px-6 py-4 space-y-4">
            <!-- Header -->
            <div class="flex items-center gap-4">
                <UButton
                    variant="ghost"
                    color="neutral"
                    icon="i-lucide-arrow-left"
                    @click="$router.back()"
                />
                <div>
                    <h1 class="text-2xl font-bold text-neutral-900 dark:text-white">附近群组</h1>
                    <p class="text-neutral-500 dark:text-neutral-400 mt-1">发现周边的兴趣小组</p>
                </div>
            </div>

            <!-- 筛选工具栏 -->
            <div class="flex items-center gap-3">
                <UInput
                    v-model="searchQuery"
                    placeholder="筛选群组名称..."
                    icon="i-lucide-search"
                    class="flex-1"
                    size="sm"
                >
                    <template v-if="searchQuery" #trailing>
                        <UButton
                            color="neutral"
                            variant="link"
                            icon="i-lucide-x"
                            size="xs"
                            @click="searchQuery = ''"
                        />
                    </template>
                </UInput>
                
                <UPopover>
                    <UButton
                        color="neutral"
                        variant="soft"
                        icon="i-lucide-sliders-horizontal"
                        :label="formatDistance(selectedRadius)"
                        size="sm"
                    />
                    <template #content>
                        <div class="p-4 w-64 space-y-4">
                            <div class="flex justify-between items-center">
                                <span class="text-sm font-medium">搜索半径</span>
                                <span class="text-xs text-primary-500">{{ formatDistance(selectedRadius) }}</span>
                            </div>
                            <USlider
                                v-model="selectedRadius"
                                :min="1000"
                                :max="50000"
                                :step="1000"
                                size="sm"
                                color="primary"
                                @update:modelValue="debouncedRefresh"
                            />
                            <div class="flex justify-between text-xs text-neutral-500">
                                <span>1km</span>
                                <span>50km</span>
                            </div>
                        </div>
                    </template>
                </UPopover>
            </div>
        </div>

        <div class="space-y-6 pb-12">
            <!-- 列表内容 -->
            <div class="min-h-[60vh]">
                <!-- 加载中 -->
                <div v-if="loading && groups.length === 0" class="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4 gap-6">
                    <GroupCardSkeleton v-for="i in 8" :key="i" />
                </div>

                <!-- 结果列表 -->
                <div v-else-if="filteredGroups.length > 0" class="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4 gap-6">
                    <GroupCard
                        v-for="group in filteredGroups"
                        :key="group.group_chat_id"
                        :group="group"
                        @click="handleGroupClick"
                        @join="handleJoin"
                    />
                </div>

                <!-- 空状态 -->
                <UEmpty
                    v-else
                    icon="i-lucide-users"
                    title="附近暂无群组"
                    description="尝试扩大搜索范围，或者创建一个新群组"
                    class="py-20 max-w-md mx-auto"
                >
                    <template #actions>
                        <UButton variant="soft" icon="i-lucide-refresh-cw" @click="refresh">刷新</UButton>
                        <UButton icon="i-lucide-plus" @click="openCreateModal">创建群组</UButton>
                    </template>
                </UEmpty>

                <!-- 加载更多 -->
                <div v-if="hasMore && filteredGroups.length > 0" class="py-8 flex justify-center">
                     <div ref="loadMoreTrigger" class="h-1 w-1" />
                     <UButton
                        :loading="loadingMore"
                        variant="ghost"
                        color="neutral"
                        @click="loadMore"
                    >
                        {{ loadingMore ? '加载中...' : '加载更多' }}
                    </UButton>
                </div>
            </div>
        </div>
        
        <!-- 创建群组模态框 -->
        <CreateGroupModal
            v-model:open="showCreateModal"
            @success="handleCreateSuccess"
        />
    </div>
</template>

<script setup lang="ts">
import { useGroupsService } from '~/services/groups.service'
import type { GroupItem } from '~/types/api'
import GroupCard from '~/components/cards/GroupCard.vue'
import GroupCardSkeleton from '~/components/skeletons/GroupCardSkeleton.vue'
import CreateGroupModal from '~/components/modals/CreateGroupModal.vue'

definePageMeta({
    layout: 'dashboard',
    requiresAuth: false
})

const { t } = useI18n()
const { formatDistance } = useFormatters()
const { requireAuth } = useRequireAuth()
const router = useRouter()
const groupsService = useGroupsService()
const geolocation = useGeolocation()
const feedback = useFeedbackStore()

// 状态
const loading = ref(true)
const loadingMore = ref(false)
const groups = ref<GroupItem[]>([])
const searchQuery = ref('')
const selectedRadius = ref(5000)
const cursor = ref<string | undefined>(undefined)
const hasMore = ref(true)
const showCreateModal = ref(false)
const loadMoreTrigger = ref<HTMLElement | null>(null)

const filteredGroups = computed(() => {
    if (!searchQuery.value) return groups.value
    const query = searchQuery.value.toLowerCase()
    return groups.value.filter(g => g.name.toLowerCase().includes(query))
})

// 初始化
onMounted(() => {
    refresh()
    useIntersectionObserver(loadMoreTrigger, ([entry]) => {
        if (entry?.isIntersecting && hasMore.value && !loading.value && !loadingMore.value) {
            loadMore()
        }
    })
})

const debouncedRefresh = useDebounceFn(refresh, 500)

async function refresh() {
    loading.value = true
    cursor.value = undefined
    hasMore.value = true
    try {
        const coords = await geolocation.getCoordinates()
        const res = await groupsService.searchByLocation({
            reference_point: {
                latitude: coords.latitude,
                longitude: coords.longitude,
            },
            radius: selectedRadius.value,
            limit: 20
        })
        groups.value = res.items || []
        cursor.value = res.pagination.next_cursor
        hasMore.value = res.pagination.has_more
    } catch (e) {
        feedback.error('加载失败')
    } finally {
        loading.value = false
    }
}

async function loadMore() {
    if (loadingMore.value || !hasMore.value) return
    loadingMore.value = true
    try {
        const coords = await geolocation.getCoordinates()
        const res = await groupsService.searchByLocation({
            reference_point: {
                latitude: coords.latitude,
                longitude: coords.longitude,
            },
            radius: selectedRadius.value,
            limit: 20,
            cursor: cursor.value
        })
        groups.value.push(...(res.items || []))
        cursor.value = res.pagination.next_cursor
        hasMore.value = res.pagination.has_more
    } catch (e) {
        // error
    } finally {
        loadingMore.value = false
    }
}

function handleGroupClick(group: { group_chat_id: string }) {
    router.push(`/groups/${group.group_chat_id}`)
}

function handleJoin(group: { group_chat_id: string }) {
    // 符合设计理念：只在执行操作时才要求登录，提供友好的登录引导
    if (!requireAuth('加入群组')) return
    router.push(`/groups/${group.group_chat_id}`)
}

function openCreateModal() {
    if (!requireAuth('创建群组')) return
    showCreateModal.value = true
}

function handleCreateSuccess(groupId: string) {
    showCreateModal.value = false
    feedback.success('群组创建成功')
    router.push(`/chat/groups/${groupId}`)
}
</script>
