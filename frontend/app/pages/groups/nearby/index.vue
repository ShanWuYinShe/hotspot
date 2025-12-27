<template>
    <div class="w-full">
        <div class="flex items-center gap-3 mb-6">
            <!-- 搜索输入 -->
            <UInput
                v-model="searchQuery"
                placeholder="搜索群组名称..."
                icon="i-lucide-search"
                class="flex-1"
                :interactive="true"
            >
                <template #trailing>
                    <UButton
                        v-if="searchQuery !== ''"
                        color="neutral"
                        variant="link"
                        icon="i-lucide-x"
                        :padded="false"
                        @click="searchQuery = ''"
                    />
                </template>
            </UInput>
            
            <!-- 半径筛选 (Popover) -->
            <UPopover width="md">
                <UButton
                    color="neutral"
                    variant="soft"
                    icon="i-lucide-sliders-horizontal"
                    :label="formatDistance(selectedRadius)"
                />
                <template #content>
                    <div class="space-y-4">
                        <div class="flex justify-between items-center">
                            <span class="text-sm font-medium">搜索半径</span>
                            <UBadge variant="soft" :label="formatDistance(selectedRadius)" />
                        </div>
                        <USlider
                            v-model="selectedRadius"
                            :min="1000"
                            :max="50000"
                            :step="1000"
                            @update:model-value="debouncedRefresh"
                        />
                        <div class="flex justify-between text-xs">
                            <span>1km</span>
                            <span>25km</span>
                            <span>50km</span>
                        </div>
                    </div>
                </template>
            </UPopover>
        </div>

        <!-- 加载状态 -->
        <div v-if="loading && groups.length === 0" class="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-6">
            <USkeleton v-for="i in 6" :key="i" class="h-48" />
        </div>

        <!-- 群组列表 -->
        <div v-else-if="filteredGroups.length > 0" class="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-6">
            <GroupCard
                v-for="group in filteredGroups"
                :key="group.id"
                :group="toGroupCardData(group)"
                @click="handleGroupClick(group)"
                @join="joinGroup(group)"
                @enter="openGroupChat(group.id)"
            />
        </div>

        <!-- 空状态 -->
        <UEmpty 
            v-else 
            icon="i-lucide-users-round" 
            title="附近暂无群组" 
            description="可以尝试扩大搜索半径，或者创建一个新群组"
            class="py-16"
        >
            <template #actions>
                <UButton variant="soft" icon="i-lucide-refresh-cw" @click="refresh">重新搜索</UButton>
                <UButton icon="i-lucide-plus" @click="createGroup">创建群组</UButton>
            </template>
        </UEmpty>

        <!-- 加载更多 -->
        <div v-if="hasMore && filteredGroups.length > 0" class="flex justify-center mt-12 mb-8">
            <!-- 触底自动加载：IntersectionObserver + 保留按钮作为兜底 -->
            <div ref="loadMoreTrigger" class="h-1 w-1" />
            <USkeleton v-if="loadingMore" class="h-10 w-32" />
            <UButton
              v-else-if="!canAutoLoadMore"
              variant="ghost"
              size="md"
              icon="i-lucide-arrow-down"
              class="px-8"
              @click="loadMore"
            >
              加载更多
            </UButton>
        </div>

        <div v-else-if="groups.length > 0 && !hasMore" class="text-center mt-12 mb-8">
            <UBadge color="neutral" variant="soft" size="xs">
              已显示全部 {{ groups.length }} 个附近群组
            </UBadge>
        </div>
    </div>
</template>

<script setup lang="ts">
import { DEFAULTS as defaults } from "~/config/defaults"
import type { GroupSearchItem } from "~/types/api"
import { useGroupsService } from '~/services/groups.service'
import { useDashboardHeader } from '~/composables/useDashboardHeader'

definePageMeta({
  title: "附近群组",
  description: "探索您身边的有趣社群",
  layout: 'dashboard',
  dashboardHeader: true,
  contentWidth: 'full'
})

const groupsService = useGroupsService()
const groupsStore = useGroupsStore()
const locationStore = useLocationStore()
const geolocation = useGeolocation()
const modal = useModal()
const { formatDistance } = useFormatters()
const authStore = useAuthStore()
const feedback = useFeedbackStore()
const dashboardHeader = useDashboardHeader()

// 状态
const groups = ref<GroupSearchItem[]>([])
const loading = ref(false)
const loadingMore = ref(false)
const hasMore = ref(true)
const nextCursor = ref<string>()
const searchQuery = ref('')
const selectedRadius = ref(5000)

const isBusy = computed(() => loading.value || loadingMore.value)
const { trigger: loadMoreTrigger, canAutoLoadMore } = useInfiniteLoadMore(
  loadMore,
  isBusy,
  computed(() => hasMore.value),
  { rootMargin: '600px 0px' }
)

// 加入群组状态
const joiningGroup = ref(false)

// 本地管理已加入的群组
import type { JoinedGroupItem } from '~/types/api'
const myGroups = ref<JoinedGroupItem[]>([])

// 使用 VueUse 的防抖函数
const debouncedRefresh = useDebounceFn(() => {
    refresh()
}, 500)

const joinedGroupIds = computed(() => new Set(myGroups.value.map(g => g.id)))

const filteredGroups = computed(() => {
    if (!searchQuery.value.trim()) return groups.value
    const query = searchQuery.value.toLowerCase()
    return groups.value.filter(group => group.name?.toLowerCase().includes(query))
})

const toGroupCardData = (group: GroupSearchItem) => ({
    ...group,
    is_member: joinedGroupIds.value.has(group.id),
    // 搜索结果中 my_role 未知，默认为 member 如果已加入，否则 undefined
    my_role: (joinedGroupIds.value.has(group.id) ? 'member' : undefined) as 'member' | undefined
})

async function refresh() {
    loading.value = true
    try {
        let pos = locationStore.coordinates
        if (!pos) {
            const coords = await geolocation.getCoordinates()
            // 尝试获取位置名称
            let locationName = defaults.locationName
            try {
                locationName = await geocoding.getLocationName(coords.latitude, coords.longitude)
            } catch {
                // 使用默认位置名称
            }
            locationStore.setLocation(coords.latitude, coords.longitude, locationName)
            pos = coords
        }

        const response = await groupsService.searchByLocation({
            latitude: pos.latitude,
            longitude: pos.longitude,
            radius: selectedRadius.value,
            limit: 20
        })

        groups.value = response.items
        // 同步到 store
        groupsStore.setNearbyGroups(response.items)
        
        hasMore.value = response.pagination.has_more
        nextCursor.value = response.pagination.next_cursor
    } catch (e: unknown) {
        feedback.error('加载失败', getErrorMessage(e, "无法获取附近群组"))
    } finally {
        loading.value = false
    }
}

async function loadMore() {
    const coords = locationStore.coordinates
    if (!hasMore.value || loadingMore.value || !coords) return
    
    loadingMore.value = true
    try {
        const response = await groupsService.searchByLocation({
            latitude: coords.latitude,
            longitude: coords.longitude,
            radius: selectedRadius.value,
            cursor: nextCursor.value,
            limit: 20
        })

        groups.value.push(...response.items)
        // 同步到 store (loadMore)
        groupsStore.setNearbyGroups(response.items, true)
        
        hasMore.value = response.pagination.has_more
        nextCursor.value = response.pagination.next_cursor
    } catch (e: unknown) {
        feedback.error('加载失败', getErrorMessage(e, "无法加载更多群组"))
    } finally {
        loadingMore.value = false
    }
}

async function joinGroup(group: GroupSearchItem) {
    const { requireAuth } = useRequireAuth()
    if (!requireAuth('加入群组')) return

    if (joinedGroupIds.value.has(group.id)) return
    
    let password: string | undefined
    
    if (group.has_password) {
        const inputPassword = await modal.showGroupPassword(group.name)
        if (!inputPassword) return // 用户取消
        password = inputPassword
    }
    
    joiningGroup.value = true
    try {
        await groupsService.joinGroup({
            id: group.id,
            password: password || undefined
        })
        
        const response = await groupsService.getJoined()
        myGroups.value = response.items
        
        navigateTo(`/chat/groups/${group.id}`)
    } catch (e: unknown) {
        feedback.error('加入失败', getErrorMessage(e))
    } finally {
        joiningGroup.value = false
    }
}

function openGroupChat(groupId: string) {
    const { requireAuth } = useRequireAuth()
    if (!requireAuth('进入群聊')) return

    navigateTo(`/chat/groups/${groupId}`)
}

// 点击群组卡片 - 已加入的群组进入聊天，未加入的群组进入详情
function handleGroupClick(group: GroupSearchItem) {
    if (joinedGroupIds.value.has(group.id)) {
        navigateTo(`/chat/groups/${group.id}`)
    } else {
        navigateTo(`/groups/${group.id}`)
    }
}

async function createGroup() {
    const { requireAuth } = useRequireAuth()
    if (!requireAuth('创建群组')) return

    const groupId = await modal.showCreateGroup()
    if (groupId) {
        feedback.success('创建成功', '群组已创建')
        navigateTo(`/chat/groups/${groupId}`)
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
      action: () => refresh()
    },
    {
      key: 'create',
      label: '创建群组',
      icon: 'i-lucide-plus',
      variant: 'outline',
      disabled: loading.value,
      action: () => createGroup()
    }
  ])
})

// 加载附近群组
onMounted(async () => {
    await refresh()
})

// 获取我的群组（响应认证状态变化）
watch(
    () => authStore.isAuthenticated,
    async (isAuthenticated) => {
        if (isAuthenticated) {
            try {
                const joinedGroups = await groupsService.getJoined()
                myGroups.value = joinedGroups.items
            } catch (e) {
                console.warn('获取我的群组失败:', e)
            }
        }
    },
    { immediate: true }
)
</script>
