<template>
    <div class="flex flex-col gap-6">
        <!-- Sticky Header & Filter -->
        <div
            class="sticky top-0 z-20 bg-white/95 dark:bg-neutral-900/95 backdrop-blur-sm border-b border-neutral-200 dark:border-neutral-800 px-4 sm:px-6 py-4 space-y-4">
            <!-- Header -->
            <div class="flex items-center gap-4">
                <UButton
                    variant="ghost"
                    color="neutral"
                    icon="i-lucide-arrow-left"
                    @click="$router.back()"
                />
                <div>
                    <h1 class="text-2xl font-bold text-neutral-900 dark:text-white">附近的人</h1>
                    <p class="text-neutral-500 dark:text-neutral-400 mt-1">发现周围的朋友</p>
                </div>
            </div>

            <!-- 筛选工具栏 -->
            <div class="flex items-center gap-3">
                <!-- 搜索输入 -->
                <UInput
                    v-model="searchQuery"
                    placeholder="筛选用户昵称..."
                    icon="i-lucide-search"
                    class="flex-1"
                    size="sm"
                >
                    <template
                        v-if="searchQuery"
                        #trailing
                    >
                        <UButton
                            color="neutral"
                            variant="link"
                            icon="i-lucide-x"
                            size="xs"
                            @click="searchQuery = ''"
                        />
                    </template>
                </UInput>

                <!-- 半径筛选 (Popover) -->
                <UPopover :popper="{ placement: 'bottom-end' }">
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
                                <span class="text-sm font-semibold text-neutral-900 dark:text-white">搜索半径</span>
                                <span class="font-bold text-primary-500">{{ formatDistance(selectedRadius) }}</span>
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
                            <div class="flex justify-between text-xs text-neutral-400 dark:text-neutral-500">
                                <span>1km</span>
                                <span>50km</span>
                            </div>
                        </div>
                    </template>
                </UPopover>
            </div>
        </div>

        <div class="space-y-6 pb-12 px-4 sm:px-6">
            <!-- Content Grid -->
            <div class="min-h-[60vh]">
                <div
                    v-if="loading && (!users || users.length === 0)"
                    class="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4 gap-6"
                >
                    <UserCardSkeleton
                        v-for="i in 8"
                        :key="i"
                    />
                </div>

                <div
                    v-else-if="filteredUsers.length > 0"
                    class="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4 gap-6"
                >
                    <UserCard
                        v-for="user in filteredUsers"
                        :key="user.user_id"
                        :user="user"
                        @click="handleUserClick(user)"
                        @message="openChat(user.user_id)"
                    />
                </div>

                <!-- Empty State -->
                <UEmpty
                    v-else
                    icon="i-lucide-users"
                    title="暂无附近用户"
                    description="尝试扩大搜索半径，或者移动到人更多的地方看看"
                    class="py-20 max-w-md mx-auto"
                >
                    <template #actions>
                        <UButton
                            variant="soft"
                            icon="i-lucide-refresh-cw"
                            @click="refresh"
                        >重新搜索</UButton>
                    </template>
                </UEmpty>

                <!-- Load More -->
                <div
                    v-if="hasMore && filteredUsers.length > 0"
                    class="flex justify-center mt-12 mb-8"
                >
                    <div
                        ref="loadMoreTrigger"
                        class="h-1 w-1"
                    />
                    <UButton
                        v-if="loadingMore"
                        loading
                        variant="ghost"
                        color="neutral"
                    >加载中...</UButton>
                    <UButton
                        v-else-if="!canAutoLoadMore"
                        variant="ghost"
                        color="neutral"
                        size="sm"
                        label="加载更多"
                        class="px-8"
                        @click="loadMore"
                    />
                </div>
            </div>
        </div>
    </div>
</template>

<script setup lang="ts">
import { getErrorMessage } from "~/utils/error"
import type { UserInfo, UserItem } from "~/types/api"
import { useUsersService } from "~/services/users.service"
import UserCard from '~/components/cards/UserCard.vue'
import UserCardSkeleton from '~/components/skeletons/UserCardSkeleton.vue'

import { geocoding } from "~/utils/geocoding"

definePageMeta({
    title: "附近的人",
    description: '探索你身边的有趣灵魂',
    layout: 'dashboard',
    requiresAuth: false,
})

const { withErrorHandling } = useErrorHandler()
const { formatDistance } = useFormatters()
const { requireAuth } = useRequireAuth()
const userService = useUsersService()
const usersStore = useUsersStore()
const geolocation = useGeolocation()
const authStore = useAuthStore()
const locationStore = useLocationStore()
const router = useRouter()

const selectedRadius = ref(5000)

// 水合安全：避免 authStore.user 在首屏为 null 导致“过滤自己”不生效
const hydratedMe = useHydratedStore(authStore, 'user', null) as Ref<UserInfo | null>

const users = ref<UserItem[]>([])
const loading = ref(false)
const loadingMore = ref(false)
const hasMore = ref(true)
const nextCursor = ref<string>()
const searchQuery = ref('')

const isBusy = computed(() => loading.value || loadingMore.value)
const { trigger: loadMoreTrigger, canAutoLoadMore } = useInfiniteLoadMore(
    loadMore,
    isBusy,
    computed(() => hasMore.value),
    { rootMargin: '600px 0px' }
)

const debouncedRefresh = useDebounceFn(() => {
    refresh()
}, 500)

const filteredUsers = computed(() => {
    if (!users.value || !Array.isArray(users.value)) return []
    const myId = hydratedMe.value?.user_id
    const base = myId ? users.value.filter(u => u.user_id !== myId) : users.value
    if (!searchQuery.value.trim()) return base
    const query = searchQuery.value.toLowerCase()
    return base.filter(user => user.nickname?.toLowerCase().includes(query))
})

async function refresh() {
    await withErrorHandling(async () => {
        loading.value = true
        // 每次刷新重置分页
        nextCursor.value = undefined
        hasMore.value = true

        let pos = locationStore.coordinates
        if (!pos) {
            const coords = await geolocation.getCoordinates()
            let locationName = '未知位置'
            try {
                locationName = await geocoding.getLocationName(coords.latitude, coords.longitude)
            } catch { }
            locationStore.setLocation(coords.latitude, coords.longitude, locationName)
            pos = coords
        }

        const response = await userService.searchByLocation({
            reference_point: {
                latitude: pos.latitude,
                longitude: pos.longitude,
            },
            radius: selectedRadius.value,
            limit: 20
        })

        users.value = response.items || []
        usersStore.setUsers(response.items || [])
        hasMore.value = response.pagination?.has_more ?? false
        nextCursor.value = response.pagination?.next_cursor

    }, {
        title: "加载失败",
        fallbackMessage: "无法获取附近的人",
    }).finally(() => {
        loading.value = false
    })
}

async function loadMore() {
    const pos = locationStore.coordinates
    if (!hasMore.value || loadingMore.value || !pos) return

    await withErrorHandling(async () => {
        loadingMore.value = true
        const response = await userService.searchByLocation({
            reference_point: {
                latitude: pos.latitude,
                longitude: pos.longitude,
            },
            radius: selectedRadius.value,
            cursor: nextCursor.value,
            limit: 20
        })

        users.value.push(...(response.items || []))
        usersStore.setUsers(response.items || [])
        hasMore.value = response.pagination?.has_more ?? false
        nextCursor.value = response.pagination?.next_cursor
    }, {
        title: "加载失败",
        fallbackMessage: "无法加载更多用户",
    }).finally(() => {
        loadingMore.value = false
    })
}

function handleUserClick(user: UserItem) {
    if (authStore.user && user.user_id === authStore.user?.user_id) {
        navigateTo('/profile')
    } else {
        navigateTo(`/users/${user.user_id}`)
    }
}

function openChat(userId: number) {
    if (!requireAuth('发送私信')) return
    if (authStore.user && userId === authStore.user?.user_id) {
        navigateTo('/profile')
    } else {
        navigateTo(`/chat/users/${userId}`)
    }
}

onMounted(() => refresh())
</script>
