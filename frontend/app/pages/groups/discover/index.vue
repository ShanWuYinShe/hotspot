<template>
    <div class="min-h-screen bg-gradient-green-emerald">
        <!-- 背景装饰 -->
        <div class="absolute inset-0 overflow-hidden pointer-events-none">
            <div class="absolute -top-40 -right-40 w-96 h-96 bg-linear-to-br from-green-400/10 to-emerald-400/10 rounded-full blur-3xl" />
            <div class="absolute -bottom-40 -left-40 w-96 h-96 bg-linear-to-tr from-emerald-400/10 to-green-400/10 rounded-full blur-3xl" />
        </div>

        <div class="relative max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-4 sm:py-6">
            <!-- 页面标题 -->
            <UPageHeader
                title="发现附近群组"
                class="mb-6"
            >
                <template #actions>
                    <UButton variant="outline" icon="i-heroicons-arrow-path" :loading="loading" @click="refresh">
                        刷新
                    </UButton>
                    <UButton icon="i-heroicons-plus" @click="createGroup">
                        创建群组
                    </UButton>
                </template>
            </UPageHeader>

            <!-- 搜索和筛选 -->
            <UCard class="mb-8 border-0 shadow-lg bg-glass-white backdrop-blur-sm">
                <div class="flex flex-col md:flex-row gap-4">
                    <div class="flex-1">
                        <UInput
                            v-model="searchQuery"
                            placeholder="搜索群组名称..."
                            icon="i-heroicons-magnifying-glass"
                            size="lg"
                        />
                    </div>
                    <div class="flex items-center gap-3">
                        <span class="text-sm text-secondary-content whitespace-nowrap">搜索半径</span>
                        <USelect
                            v-model="selectedRadius"
                            :items="radiusOptions"
                            size="lg"
                            class="w-32"
                            @change="refresh"
                        />
                    </div>
                </div>
            </UCard>

            <!-- 加载状态 -->
            <div v-if="loading && groups.length === 0" class="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-6">
                <USkeleton v-for="i in 6" :key="i" class="h-56 rounded-2xl" />
            </div>

            <!-- 群组列表 -->
            <div v-else-if="filteredGroups.length > 0" class="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-6">
                <div
                    v-for="group in filteredGroups"
                    :key="group.id"
                    class="interactive-card group bg-card-elevated rounded-2xl overflow-hidden cursor-pointer"
                    @click="handleGroupClick(group)"
                >
                    <div class="p-6">
                        <div class="flex items-center gap-4 mb-4">
                            <UAvatar :alt="group.name" :src="group.avatar_url" size="xl" class="ring-2 ring-default shadow-lg">
                                {{ group.name?.charAt(0) || 'G' }}
                            </UAvatar>
                            <div class="min-w-0 flex-1">
                                <div class="flex items-center gap-2 mb-1">
                                    <h3 class="font-semibold text-lg text-primary-content truncate">{{ group.name }}</h3>
                                    <UIcon v-if="group.has_password" name="i-heroicons-lock-closed" class="icon-sm text-amber" />
                                </div>
                                <p class="text-sm text-secondary-content flex items-center gap-1">
                                    <UIcon name="i-heroicons-users" class="icon-sm" />
                                    {{ group.members_count }} 成员
                                </p>
                            </div>
                        </div>
                        
                        <div class="flex items-center justify-between text-sm text-secondary-content mb-4">
                            <div class="flex items-center gap-2 truncate">
                                <UIcon name="i-heroicons-map-pin" class="icon-sm text-green shrink-0" />
                                <span class="truncate">{{ group.geo.location_name || '未设置位置' }}</span>
                            </div>
                            <span class="flex items-center gap-1 shrink-0">
                                <UIcon name="i-heroicons-signal" class="icon-sm" />
                                {{ formatDistance(group.geo.distance) }}
                            </span>
                        </div>
                        
                        <div class="flex gap-2">
                            <UButton
                                size="sm"
                                :variant="joinedGroupIds.has(group.id) ? 'soft' : 'solid'"
                                :color="joinedGroupIds.has(group.id) ? 'neutral' : 'primary'"
                                :icon="joinedGroupIds.has(group.id) ? 'i-heroicons-check' : 'i-heroicons-user-plus'"
                                :disabled="joinedGroupIds.has(group.id)"
                                class="flex-1"
                                @click.stop="joinGroup(group)"
                            >
                                {{ joinedGroupIds.has(group.id) ? '已加入' : '加入群组' }}
                            </UButton>
                            <UButton size="sm" variant="ghost" icon="i-heroicons-chat-bubble-left" @click.stop="openGroupChat(group.id)" />
                        </div>
                    </div>
                </div>
            </div>

            <!-- 空状态 -->
            <UEmpty v-else icon="i-heroicons-user-group" title="附近暂无群组" description="可以尝试扩大搜索半径，或者创建一个新群组">
                <template #actions>
                    <UButton variant="outline" icon="i-heroicons-arrow-path" @click="refresh">重新搜索</UButton>
                    <UButton icon="i-heroicons-plus" @click="createGroup">创建群组</UButton>
                </template>
            </UEmpty>

            <!-- 加载更多 -->
            <div v-if="hasMore && filteredGroups.length > 0" class="flex justify-center mt-8">
                <UButton :loading="loadingMore" variant="outline" size="lg" icon="i-heroicons-arrow-down" @click="loadMore">
                    加载更多
                </UButton>
            </div>

            <div v-else-if="groups.length > 0 && !hasMore" class="text-center mt-8 text-sm text-secondary-content">
                已显示全部 {{ groups.length }} 个附近群组
            </div>
        </div>
    </div>
</template>

<script setup lang="ts">
import { DEFAULTS as defaults } from "~/config/defaults"
import type { GroupSearchItem } from "~/types/api"
import { useGroupsService } from '~/services/groups.service'
import { geocoding } from '~/utils/geocoding'
import { getErrorMessage } from '~/utils/error'

definePageMeta({ title: "发现群组", layout: 'dashboard' })

const toast = useToast()
const groupsService = useGroupsService()
const groupsStore = useGroupsStore()
const locationStore = useLocationStore()
const geolocation = useGeolocation()
const modal = useModal()
const { formatDistance } = useFormatters()
const authStore = useAuthStore()

// 状态
const groups = ref<GroupSearchItem[]>([])
const loading = ref(false)
const loadingMore = ref(false)
const hasMore = ref(true)
const nextCursor = ref<string>()
const searchQuery = ref('')
const selectedRadius = ref(5000)

// 加入群组状态
const joiningGroup = ref(false)

// 本地管理已加入的群组
import type { JoinedGroupItem } from '~/types/api'
const myGroups = ref<JoinedGroupItem[]>([])

const radiusOptions = [
    { label: '1 公里', value: 1000 },
    { label: '5 公里', value: 5000 },
    { label: '10 公里', value: 10000 },
    { label: '25 公里', value: 25000 },
    { label: '50 公里', value: 50000 }
]

const joinedGroupIds = computed(() => new Set(myGroups.value.map(g => g.id)))

const filteredGroups = computed(() => {
    if (!searchQuery.value.trim()) return groups.value
    const query = searchQuery.value.toLowerCase()
    return groups.value.filter(group => group.name?.toLowerCase().includes(query))
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
        hasMore.value = response.pagination.has_more
        nextCursor.value = response.pagination.next_cursor
    } catch (e: unknown) {
        toast.add({ title: "加载失败", description: getErrorMessage(e, "无法获取附近群组"), color: "error" })
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
        hasMore.value = response.pagination.has_more
        nextCursor.value = response.pagination.next_cursor
    } catch (e: unknown) {
        toast.add({ title: "加载失败", description: getErrorMessage(e, "无法加载更多群组"), color: "error" })
    } finally {
        loadingMore.value = false
    }
}

async function joinGroup(group: GroupSearchItem) {
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
            password_hash: password ? await hashPassword(password) : undefined
        })
        
        const response = await groupsService.getJoined()
        myGroups.value = response.items
        
        toast.add({ title: "加入成功", description: `已加入群组 ${group.name}`, color: "success" })
        navigateTo(`/chat/groups/${group.id}`)
    } catch (e: unknown) {
        toast.add({ title: "加入失败", description: getErrorMessage(e), color: "error" })
    } finally {
        joiningGroup.value = false
    }
}

async function hashPassword(password: string): Promise<string> {
    const encoder = new TextEncoder()
    const data = encoder.encode(password)
    const hashBuffer = await crypto.subtle.digest('SHA-256', data)
    const hashArray = Array.from(new Uint8Array(hashBuffer))
    return hashArray.map(b => b.toString(16).padStart(2, '0')).join('')
}

function openGroupChat(groupId: string) {
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
    const groupId = await modal.showCreateGroup()
    if (groupId) {
        toast.add({ title: "创建成功", description: "群组已创建", color: "success" })
        navigateTo(`/chat/groups/${groupId}`)
    }
}

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
