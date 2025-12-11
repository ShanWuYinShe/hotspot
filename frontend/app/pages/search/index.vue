<template>
    <div class="mx-auto max-w-7xl space-y-6">
        <!-- 页面标题 -->
        <UPageHeader
            :title="$t('users.search.title')"
            :description="$t('users.search.subtitle')"
        />

        <!-- 搜索栏 -->
        <UCard class="mb-6">
            <div class="space-y-4">
                <div class="flex gap-4">
                    <UInput
                        v-model="searchQuery"
                        :placeholder="$t('users.search.placeholder')"
                        icon="i-heroicons-magnifying-glass"
                        class="flex-1"
                        size="md"
                        :loading="searching"
                        @keyup.enter="performSearch"
                    />
                    <UButton
                        @click="performSearch"
                        :loading="searching"
                        :disabled="!searchQuery.trim()"
                        color="primary"
                        size="md"
                        class="px-6 py-3 shadow-sm transition-colors duration-200 font-medium"
                    >
                        <template #leading>
                            <UIcon name="i-heroicons-magnifying-glass" class="icon-sm" />
                        </template>
                        {{ $t("app.actions.search") }}
                    </UButton>
                </div>

                <!-- 搜索类型选择 -->
                <div class="flex gap-2">
                    <span
                        class="text-sm text-secondary-content mr-2"
                        >{{ $t("users.search.searchType") }}</span
                    >
                    <UButton
                        variant="ghost"
                        size="xs"
                        :class="{
                            'bg-primary/10 text-primary': searchType === 'name',
                        }"
                        @click="searchType = 'name'"
                    >
                        {{ $t("users.search.type.name") }}
                    </UButton>
                    <UButton
                        variant="ghost"
                        size="xs"
                        :class="{
                            'bg-primary/10 text-primary':
                                searchType === 'phone',
                        }"
                        @click="searchType = 'phone'"
                    >
                        {{ $t("users.search.type.phone") }}
                    </UButton>
                    <UButton
                        variant="ghost"
                        size="xs"
                        :class="{
                            'bg-primary/10 text-primary': searchType === 'all',
                        }"
                        @click="searchType = 'all'"
                    >
                        {{ $t("users.search.type.all") }}
                    </UButton>
                </div>
            </div>
        </UCard>

        <!-- 搜索结果 -->
        <div v-if="hasSearched">
            <!-- 加载中 -->
            <LoadingState 
              v-if="searching"
              title="搜索用户中..."
              description="正在查找匹配的用户"
            />

            <!-- 搜索结果表格 -->
            <div v-else>
                <UTable 
                    :data="pagedResults" 
                    :columns="columns"
                    :ui="{
                        tr: 'interactive-table-row cursor-pointer'
                    }"
                >
                    <template #user-cell="{ row }">
                        <UUser
                            :name="row.original.nickname"
                            :description="getUserSearchDescription(row.original)"
                            :avatar="{
                                src: row.original.avatar_url,
                                alt: row.original.nickname,
                                size: 'lg',
                                chip: isUserOnline(row.original.user_id) ? { color: 'success', position: 'bottom-right' } : undefined
                            }"
                            :chip="getFriendshipStatus(row.original) !== 'none' ? { 
                                text: getFriendshipStatusText(row.original), 
                                color: getFriendshipStatusChipColor(row.original) 
                            } : undefined"
                            size="lg"
                            class="min-w-[220px]"
                        >
                            <template #description>
                                <div class="space-y-1 text-sm text-secondary-content">
                                    <div v-if="row.original.bio" class="truncate">
                                        {{ row.original.bio }}
                                    </div>
                                    <div class="flex items-center gap-4">
                                        <span v-if="row.original.geo.location_name" class="flex items-center gap-1">
                                            <UIcon name="i-heroicons-map-pin" class="icon-xs" />
                                            {{ row.original.geo.location_name }}
                                        </span>
                                        <span v-if="row.original.geo.distance !== undefined" class="flex items-center gap-1">
                                            <UIcon name="i-heroicons-signal" class="icon-xs" />
                                            {{ formatDistance(row.original.geo.distance) }}
                                        </span>
                                    </div>
                                </div>
                            </template>
                        </UUser>
                    </template>
                    <template #actions-cell="{ row }">
                        <div class="flex gap-2 justify-end">
                            <UButton
                                v-if="getFriendshipStatus(row.original) === 'none'"
                                size="sm"
                                @click.stop="sendFriendRequest(row.original)"
                                :loading="row.original.requesting"
                            >
                                <UIcon name="i-heroicons-user-plus" class="w-4 h-4 mr-1" />
                                {{ $t("users.detail.actions.addFriend") }}
                            </UButton>
                            <UButton
                                v-else-if="getFriendshipStatus(row.original) === 'pending_sent'"
                                size="sm"
                                variant="ghost"
                                disabled
                            >
                                <UIcon name="i-heroicons-clock" class="w-4 h-4 mr-1" />
                                {{ $t("users.search.actions.sent") }}
                            </UButton>
                            <UButton
                                v-else-if="getFriendshipStatus(row.original) === 'pending_received'"
                                size="sm"
                                variant="outline"
                                @click.stop="handleFriendRequest(row.original)"
                            >
                                <UIcon name="i-heroicons-check" class="w-4 h-4 mr-1" />
                                {{ $t("friends.requests.actions.accept") }}
                            </UButton>
                            <UButton
                                v-else-if="getFriendshipStatus(row.original) === 'accepted'"
                                size="sm"
                                variant="ghost"
                                @click.stop="row.original.user_id === authStore.user?.user_id ? navigateTo('/profile') : navigateTo(`/chat/users/${row.original.user_id}`)"
                            >
                                <UIcon name="i-heroicons-chat-bubble-left-right" class="w-4 h-4 mr-1" />
                                {{ $t("users.search.actions.chat") }}
                            </UButton>
                            <UButton
                                size="sm"
                                variant="ghost"
                                @click.stop="viewUserProfile(row.original)"
                            >
                                <UIcon name="i-heroicons-user" class="w-4 h-4 mr-1" />
                                {{ $t("users.search.actions.view") }}
                            </UButton>
                        </div>
                    </template>
                    <template #empty>
                        <UEmpty
                            icon="i-heroicons-user-minus"
                            :title="$t('users.search.empty.title')"
                            :description="$t('users.search.empty.description')"
                            :actions="[{ label: $t('users.search.empty.actionText'), onClick: clearSearch }]"
                        />
                    </template>
                </UTable>
                <div v-if="pageCount > 1" class="mt-4 flex justify-center">
                    <UPagination
                        v-model:page="page"
                        :items-per-page="pageSize"
                        :total="searchResults.length"
                    />
                </div>
            </div>
        </div>

        <!-- 推荐用户 -->
        <div v-else>
            <UCard>
                <template #header>
                    <div class="flex items-center justify-between">
                        <h3 class="text-lg font-semibold">
                            {{ $t("users.search.recommended.title") }}
                        </h3>
                        <UButton
                            size="sm"
                            variant="ghost"
                            @click="loadRecommendations"
                            :loading="loadingRecommended"
                        >
                            <UIcon
                                name="i-heroicons-arrow-path"
                                class="icon-sm loading-spin"
                            />
                        </UButton>
                    </div>
                </template>
                <!-- 推荐用户加载中 -->
                <LoadingState 
                  v-if="loadingRecommended"
                  title="加载推荐用户..."
                  description="正在为您寻找可能认识的人"
                />

                <!-- 推荐用户列表 -->
                <div v-else-if="recommendedUsers.length > 0" class="space-y-2">
                    <div
                        v-for="user in recommendedUsers"
                        :key="user.user_id"
                        class="interactive-list-item flex items-center gap-3 cursor-pointer rounded-lg p-3"
                        @click="viewUserProfile(user)"
                    >
                        <UAvatar :alt="user.nickname">{{
                            user.nickname?.charAt(0)
                        }}</UAvatar>
                        <div class="flex-1 min-w-0">
                            <div class="font-medium truncate">
                                {{ user.nickname }}
                            </div>
                            <div
                                class="text-sm text-secondary-content flex items-center gap-2"
                            >
                                <span v-if="user.geo.location_name">{{
                                    user.geo.location_name
                                }}</span>
                                <span v-if="user.geo.distance !== undefined"
                                    >· {{ formatDistance(user.geo.distance) }}</span
                                >
                            </div>
                        </div>
                        <UButton
                            size="sm"
                            variant="outline"
                            @click.stop="sendFriendRequest(user)"
                            :loading="user.requesting"
                        >
                            <UIcon name="i-heroicons-user-plus" class="w-4 h-4 mr-1" />
                            {{ $t("users.detail.actions.addFriend") }}
                        </UButton>
                    </div>
                </div>
                <UEmpty
                  v-else
                  icon="i-heroicons-users"
                  :title="$t('users.search.recommended.empty.title')"
                  :description="$t('users.search.recommended.empty.description')"
                  :actions="[{ label: $t('users.search.recommended.empty.actionText'), onClick: loadRecommendations }]"
                />
            </UCard>
        </div>
    </div>
</template>

<script setup lang="ts">
import { useUsersService } from '~/services/users.service'
import { getErrorMessage } from '~/utils/error'
import type { UserSearchItem } from '~/types/api'
import type { TableColumn } from '@nuxt/ui'
// import { buildFriendUrl, FRIEND_ENDPOINTS } from '~/config/api' // 好友功能暂未实现

definePageMeta({
    title: "users.search.title",
    layout: 'dashboard'
})

// 搜索结果用户类型（扩展 UserSearchItem，添加本地状态）
interface SearchUserItem extends UserSearchItem {
    requesting?: boolean
    // 关系状态（通过 API 获取或本地更新）
    is_following?: boolean
    is_followed_by?: boolean
    is_mutual_follow?: boolean
}

// 表格行类型
type UserRow = SearchUserItem

const userService = useUsersService()
const toast = useToast()
const router = useRouter()
// 使用水合安全的 store 访问
const authStore = useAuthStore()

const { t: tt } = useI18n()
const { formatDistance } = useFormatters()

// 搜索状态
const searchQuery = ref('')
const searchType = ref<'name' | 'phone' | 'all'>('name')
const searching = ref(false)
const hasSearched = ref(false)
const searchResults = ref<SearchUserItem[]>([])

// 表格列
const columns = computed(() => {
  const cols: TableColumn<UserRow>[] = [
    {
        id: 'user',
        accessorKey: 'nickname',
        header: tt('common.user')
    },
    {
        id: 'actions',
        accessorKey: 'user_id',
        header: tt('common.actions')
    },
  ]
  return cols
})

// 客户端分页
const page = ref(1);
const pageSize = ref(10);
const pageCount = computed(() =>
    Math.max(1, Math.ceil(searchResults.value.length / pageSize.value)),
);
const pagedResults = computed(() => {
    const start = (page.value - 1) * pageSize.value;
    return searchResults.value.slice(start, start + pageSize.value);
});

// 推荐用户状态
const recommendedUsers = ref<SearchUserItem[]>([])
const loadingRecommended = ref(false)

// 搜索功能（全局搜索，不限范围）
async function performSearch() {
    if (!searchQuery.value.trim() || searching.value) return
    searching.value = true
    hasSearched.value = true
    page.value = 1
    try {
        // 获取位置（仅用于计算距离，不过滤范围）
        const geolocation = useGeolocation()
        const location = await geolocation.getCoordinates()
        
        const response = await userService.searchByKeyword({ 
            kw: searchQuery.value,
            latitude: location.latitude,
            longitude: location.longitude
        })
        searchResults.value = (response.items || []).map((u: UserSearchItem): SearchUserItem => ({
            ...u,
            requesting: false,
        }))
    } catch (e: unknown) {
        toast.add({
            title: tt('users.search.toasts.searchFailed'),
            description: getErrorMessage(e),
            color: 'error',
        })
        searchResults.value = []
    } finally {
        searching.value = false
    }
}

// 清空搜索
function clearSearch() {
    searchQuery.value = "";
    hasSearched.value = false;
    searchResults.value = [];
    page.value = 1;
}

// 推荐用户
async function loadRecommendations() {
    loadingRecommended.value = true
    try {
        const geolocation = useGeolocation()
        const coords = await geolocation.getCoordinates()
        const response = await userService.searchByLocation({
            latitude: coords.latitude,
            longitude: coords.longitude,
            radius: 1000,
            limit: 5,
        })
        recommendedUsers.value = response.items
            .filter((user: UserSearchItem) => user.user_id !== authStore.user?.user_id)
            .map((u: UserSearchItem): SearchUserItem => ({ ...u, requesting: false }))
    } catch (e: unknown) {
        toast.add({
            title: tt('users.search.toasts.recommendLoadFailed'),
            description: getErrorMessage(e),
            color: 'error',
        })
    } finally {
        loadingRecommended.value = false
    }
}

// 关注系统功能（好友 = 互相关注）
async function followUser(user: SearchUserItem) {
    if (user.requesting) return
    user.requesting = true
    try {
        await userService.followUser({ id: user.user_id })
        // 更新本地状态
        user.is_following = true
        toast.add({
            title: tt("users.detail.toasts.followed"),
            color: "success",
        })
    } catch (e: unknown) {
        toast.add({
            title: tt("users.detail.toasts.followFailed"),
            description: getErrorMessage(e, tt("users.search.toasts.unableToFollow")),
            color: "error",
        })
    } finally {
        user.requesting = false
    }
}

async function unfollowUser(user: SearchUserItem) {
    if (user.requesting) return
    user.requesting = true
    try {
        await userService.unfollowUser({ id: user.user_id })
        // 更新本地状态
        user.is_following = false
        toast.add({
            title: tt("users.detail.toasts.unfollowed"),
            color: "success",
        })
    } catch (e: unknown) {
        toast.add({
            title: tt("common.actionFailed"),
            description: getErrorMessage(e),
            color: "error",
        })
    } finally {
        user.requesting = false
    }
}

// 关系状态管理
function getRelationshipStatus(user: SearchUserItem): 'none' | 'following' | 'followed_by' | 'mutual' {
    if (user.is_mutual_follow) return 'mutual'
    if (user.is_following) return 'following'
    if (user.is_followed_by) return 'followed_by'
    return 'none'
}

function getRelationshipStatusText(user: SearchUserItem): string {
    switch (getRelationshipStatus(user)) {
        case "following":
            return tt("users.search.status.following");
        case "followed_by":
            return tt("users.search.status.followsYou");
        case "mutual":
            return tt("profile.stats.friends");
        default:
            return "";
    }
}

function getRelationshipStatusColor(user: SearchUserItem): string {
    switch (getRelationshipStatus(user)) {
        case 'following':
            return 'info'
        case 'followed_by':
            return 'warning'
        case 'mutual':
            return 'success'
        default:
            return 'neutral'
    }
}

// 向后兼容函数（保留以避免模板编译错误）
function getFriendshipStatus(user: SearchUserItem): 'none' | 'pending_sent' | 'pending_received' | 'accepted' {
    const status = getRelationshipStatus(user)
    if (status === 'mutual') return 'accepted'
    if (status === 'following') return 'pending_sent'
    if (status === 'followed_by') return 'pending_received'
    return 'none'
}

function getFriendshipStatusText(user: SearchUserItem): string {
    return getRelationshipStatusText(user)
}

function sendFriendRequest(user: SearchUserItem) {
    return followUser(user)
}

function handleFriendRequest(user: SearchUserItem) {
    return followUser(user)
}

// 用户导航
function viewUserProfile(user: SearchUserItem) {
    // 如果是当前用户，直接跳转到个人主页
    if (user.user_id === authStore.user?.user_id) {
        navigateTo('/profile')
    } else {
        navigateTo(`/users/${user.user_id}`)
    }
}

// formatDistance 函数已从统一工具函数导入

// 页面加载时获取推荐用户
onMounted(async () => {
    try {
        await loadRecommendations()
    } catch (error: unknown) {
        console.error('[UserSearch] Failed to load recommendations:', error)
        toast.add({
            title: tt('users.search.toasts.recommendLoadFailed'),
            description: getErrorMessage(error, tt('users.search.toasts.recommendUnable')),
            color: 'error',
        })
    }
})

// UUser 组件相关辅助函数
const getUserSearchDescription = (user: SearchUserItem): string => {
    const parts: string[] = []
    const locationName = user.geo.location_name
    const distance = user.geo.distance
    if (locationName && distance !== undefined) {
        parts.push(`${locationName} · ${formatDistance(distance)}`)
    } else if (locationName) {
        parts.push(locationName)
    } else if (distance !== undefined) {
        parts.push(formatDistance(distance))
    }
    return parts.join(' · ')
}

// 检查用户在线状态（暂时禁用，需要WebSocket支持）
const isUserOnline = (_userId: number): boolean => false

// 获取关系状态的芯片颜色
const getFriendshipStatusChipColor = (user: SearchUserItem): string => {
    return getRelationshipStatusColor(user)
}
</script>
