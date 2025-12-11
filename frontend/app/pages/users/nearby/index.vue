<template>
    <div class="min-h-screen bg-gradient-purple-blue">
        <!-- 背景装饰 -->
        <div class="absolute inset-0 overflow-hidden pointer-events-none">
            <div class="absolute -top-40 -right-40 w-96 h-96 bg-linear-to-br from-blue-400/10 to-purple-400/10 rounded-full blur-3xl" />
            <div class="absolute -bottom-40 -left-40 w-96 h-96 bg-linear-to-tr from-purple-400/10 to-blue-400/10 rounded-full blur-3xl" />
        </div>

        <div class="relative max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-4 sm:py-6">
            <!-- 页面标题 -->
            <UPageHeader
                title="发现附近的人"
                class="mb-6"
            >
                <template #actions>
                    <UButton variant="outline" icon="i-heroicons-arrow-path" :loading="loading" @click="refresh">
                        刷新
                    </UButton>
                </template>
            </UPageHeader>

            <!-- 搜索和筛选 -->
            <UCard class="mb-8 border-0 shadow-lg bg-glass-white backdrop-blur-sm">
                <div class="flex flex-col md:flex-row gap-4">
                    <div class="flex-1">
                        <UInput
                            v-model="searchQuery"
                            placeholder="搜索用户昵称..."
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
            <div v-if="loading && users.length === 0" class="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4 gap-6">
                <USkeleton v-for="i in 8" :key="i" class="h-48 rounded-2xl" />
            </div>

            <!-- 用户列表 -->
            <div v-else-if="filteredUsers.length > 0" class="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4 gap-6">
                <UserCard
                    v-for="user in filteredUsers"
                    :key="user.user_id"
                    :user="user"
                    @click="handleUserClick(user)"
                    @message="openChat(user.user_id)"
                />
            </div>

            <!-- 空状态 -->
            <UEmpty v-else icon="i-heroicons-user-group" title="附近暂无用户" description="可以尝试扩大搜索半径，或者在侧边栏修改位置">
                <template #actions>
                    <UButton variant="outline" icon="i-heroicons-arrow-path" @click="refresh">重新搜索</UButton>
                </template>
            </UEmpty>

            <!-- 加载更多 -->
            <div v-if="hasMore && filteredUsers.length > 0" class="flex justify-center mt-8">
                <UButton :loading="loadingMore" variant="outline" size="lg" icon="i-heroicons-arrow-down" @click="loadMore">
                    加载更多
                </UButton>
            </div>

            <div v-else-if="users.length > 0 && !hasMore" class="text-center mt-8 text-sm text-secondary-content">
                已显示全部 {{ users.length }} 位附近用户
            </div>
        </div>
    </div>
</template>

<script setup lang="ts">
import { getErrorMessage } from "~/utils/error"
import type { UserSearchItem } from "~/types/api"
import { useUsersService } from "~/services/users.service"

definePageMeta({ title: "附近的人", layout: 'dashboard' })

const toast = useToast()
const userService = useUsersService()
const geolocation = useGeolocation()
const authStore = useAuthStore()
const locationStore = useLocationStore()

// 状态
const users = ref<UserSearchItem[]>([])
const loading = ref(false)
const loadingMore = ref(false)
const hasMore = ref(true)
const nextCursor = ref<string>()
const searchQuery = ref('')
const selectedRadius = ref(5000)

const radiusOptions = [
    { label: '1 公里', value: 1000 },
    { label: '5 公里', value: 5000 },
    { label: '10 公里', value: 10000 },
    { label: '25 公里', value: 25000 },
    { label: '50 公里', value: 50000 }
]

const filteredUsers = computed(() => {
    if (!searchQuery.value.trim()) return users.value
    const query = searchQuery.value.toLowerCase()
    return users.value.filter(user => user.nickname?.toLowerCase().includes(query))
})

async function refresh() {
    loading.value = true
    try {
        // 使用 locationStore 的位置，如果没有则获取并更新 store
        let pos = locationStore.coordinates
        if (!pos) {
            const coords = await geolocation.getCoordinates()
            // 尝试获取位置名称并更新 store
            let locationName = '未知位置'
            try {
                const { geocoding } = await import('~/utils/geocoding')
                locationName = await geocoding.getLocationName(coords.latitude, coords.longitude)
            } catch {
                // 使用默认名称
            }
            locationStore.setLocation(coords.latitude, coords.longitude, locationName)
            pos = coords
        }

        const response = await userService.searchByLocation({
            latitude: pos.latitude,
            longitude: pos.longitude,
            radius: selectedRadius.value,
            limit: 20
        })

        users.value = response.items
        hasMore.value = response.pagination.has_more
        nextCursor.value = response.pagination.next_cursor
    } catch (e: unknown) {
        toast.add({ title: "加载失败", description: getErrorMessage(e, "无法获取附近的人"), color: "error" })
    } finally {
        loading.value = false
    }
}

async function loadMore() {
    const pos = locationStore.coordinates
    if (!hasMore.value || loadingMore.value || !pos) return
    
    loadingMore.value = true
    try {
        const response = await userService.searchByLocation({
            latitude: pos.latitude,
            longitude: pos.longitude,
            radius: selectedRadius.value,
            cursor: nextCursor.value,
            limit: 20
        })

        users.value.push(...response.items)
        hasMore.value = response.pagination.has_more
        nextCursor.value = response.pagination.next_cursor
    } catch (e: unknown) {
        toast.add({ title: "加载失败", description: getErrorMessage(e, "无法加载更多用户"), color: "error" })
    } finally {
        loadingMore.value = false
    }
}

function handleUserClick(user: UserSearchItem) {
    if (user.user_id === authStore.user?.user_id) {
        navigateTo('/profile')
    } else {
        navigateTo(`/users/${user.user_id}`)
    }
}

function openChat(userId: number) {
    if (userId === authStore.user?.user_id) {
        navigateTo('/profile')
    } else {
        navigateTo(`/chat/users/${userId}`)
    }
}

onMounted(() => refresh())
</script>
