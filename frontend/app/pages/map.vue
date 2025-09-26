<template>
    <div class="max-w-7xl mx-auto px-4 py-8">
        <!-- 页面标题 -->
        <div class="mb-8">
            <div class="flex items-center gap-3 mb-3">
                <div class="w-10 h-10 bg-gradient-to-br from-emerald-500 to-teal-600 rounded-xl flex items-center justify-center shadow-lg">
                    <UIcon name="i-heroicons-map" class="w-6 h-6 text-white" />
                </div>
                <div>
                    <h1 class="text-3xl font-bold bg-gradient-to-r from-emerald-600 to-teal-600 bg-clip-text text-transparent">地图探索</h1>
                    <div class="w-16 h-0.5 bg-gradient-to-r from-emerald-500 to-teal-500 rounded-full mt-1"></div>
                </div>
            </div>
            <p class="text-secondary text-lg leading-relaxed">发现附近有趣的用户、群组和精彩动态，探索你周围的世界</p>
        </div>

        <!-- 标签页切换 -->
        <UTabs v-model="currentTab" :items="tabs" :default-index="0" class="mb-6" />

        <!-- 搜索栏 -->
        <div class="mb-6 flex gap-4">
            <UInput
                v-model="currentSearchQuery"
                placeholder="搜索用户、群组或动态..."
                icon="i-heroicons-magnifying-glass"
                class="flex-1"
                @keyup.enter="handleSearch"
            />
            <UButton @click="handleSearch" :loading="currentLoading">
                搜索
            </UButton>
        </div>

        <!-- 地图容器 -->
        <MapContainer 
            :amap-enabled="amapEnabled"
            ref="mapContainerRef"
            class="mb-6"
        />

        <!-- 操作按钮 -->
        <div class="flex gap-2 mb-6">
                            <UButton
                :loading="refreshing"
                @click="refreshLocation"
                icon="i-heroicons-arrow-path"
            >
                刷新位置
                            </UButton>
                                <UButton
                :variant="isLocationSharing ? 'solid' : 'outline'"
                @click="toggleLocationSharing"
                icon="i-heroicons-map-pin"
            >
                {{ isLocationSharing ? '停止共享' : '共享位置' }}
                            </UButton>
                        </div>

        <!-- 数据列表 -->
        <div class="space-y-4">
            <USkeleton v-if="currentLoading" class="h-32" />
            <EmptyState
                v-else-if="currentData.length === 0"
                :icon="tabs.find(t => t.value === currentTab)?.icon || 'i-heroicons-map'"
                :title="`暂无${tabs.find(t => t.value === currentTab)?.label}`"
                :description="`附近没有找到相关的${tabs.find(t => t.value === currentTab)?.label}`"
            />
            <template v-else>
                <!-- 用户列表 -->
                <UserCard
                    v-if="currentTab === 'users'"
                    v-for="user in (currentData as unknown as User[])"
                    :key="user.user_id"
                    :user="user"
                    @click="navigateTo(`/users/${user.user_id}`)"
                />

            <!-- 群组列表 -->
                <GroupCard
                    v-if="currentTab === 'groups'"
                    v-for="group in (currentData as unknown as Group[])"
                    :key="group.group_id"
                    :group="group"
                    @click="navigateTo(`/groups/${group.group_id}`)"
                    @join="handleJoinGroup"
                />
                
                <!-- 动态列表 -->
                <MomentCard
                    v-if="currentTab === 'moments'"
                    v-for="moment in (currentData as unknown as Moment[])"
                    :key="moment.moment_id"
                    :moment="moment"
                    @click="navigateTo(`/moments/${moment.moment_id}`)"
                    @like="handleLikeMoment"
                />
                </template>
            </div>

        <!-- 加载更多 -->
        <div v-if="hasMore" class="mt-6 text-center">
            <UButton @click="loadMore" :loading="loadingMore" variant="outline">
                                加载更多
                            </UButton>
                        </div>
    </div>
</template>

<script setup lang="ts">
import type { User, Group, Moment } from '~/types'

definePageMeta({
    title: '地图',
    description: '发现附近的用户、群组和动态',
    layout: 'dashboard'
})

// 基础状态
const currentTab = ref('users')
const isLocationSharing = ref(false)
const amapEnabled = ref(true)
const refreshing = ref(false)

// 标签页配置
const tabs = [
    { value: 'users', label: '用户', icon: 'i-heroicons-users' },
    { value: 'groups', label: '群组', icon: 'i-heroicons-user-group' },
    { value: 'moments', label: '动态', icon: 'i-heroicons-photo' }
]

// 服务和工具
const userService = useUserService()
const groupService = useGroupService()
const momentService = useMomentService()
const toast = useSmartToast()
const { currentLocation } = useLocationManager()

// 用户数据加载器
const createUserLoader = () => {
    return useDataLoader<User>(async (cursor?: string) => {
        if (!currentLocation.value) {
            throw new Error('无法获取位置信息')
        }
        
        const { latitude, longitude } = currentLocation.value
        const result = await userService.findNearbyUsers({
            latitude,
            longitude,
            radius: 5000, // 5km半径
            cursor,
            limit: 20
        })
        return {
            items: result.items,
            hasMore: result.has_more,
            nextCursor: result.next_cursor
        }
    }, { autoLoad: false })
}

// 群组数据加载器
const createGroupLoader = () => {
    return useDataLoader<Group>(async (cursor?: string) => {
        if (!currentLocation.value) {
            throw new Error('无法获取位置信息')
        }
        
        const { latitude, longitude } = currentLocation.value
        const result = await groupService.searchNearbyGroups(
            latitude,
            longitude,
            5000 // 5km半径
        )
        return {
            items: result.items,
            hasMore: result.has_more,
            nextCursor: result.next_cursor
        }
    }, { autoLoad: false })
}

// 动态数据加载器
const createMomentLoader = () => {
    return useDataLoader<Moment>(async (cursor?: string) => {
        if (!currentLocation.value) {
            throw new Error('无法获取位置信息')
        }
        
        const { latitude, longitude } = currentLocation.value
        const result = await momentService.getNearbyMoments(
            latitude,
            longitude,
            5000 // 5km半径
        )
        return {
            items: result.items,
            hasMore: result.has_more,
            nextCursor: result.next_cursor
        }
    }, { autoLoad: false })
}

// 用户搜索器
const createUserSearch = () => {
    return useSearch<User>(async (query: string) => {
        if (!query.trim()) return []
        const result = await userService.searchUsers(query.trim())
        return result.items
    })
}

// 群组搜索器
const createGroupSearch = () => {
    return useSearch<Group>(async (query: string) => {
        if (!query.trim()) return []
        const result = await groupService.searchGroupsByName(query.trim())
        return result.items
    })
}

// 动态搜索器
const createMomentSearch = () => {
    return useSearch<Moment>(async (query: string) => {
        if (!query.trim()) return []
        const result = await momentService.searchMoments({
            type: 'tags',
            tags: [query.trim()]
        })
        return result.items
    })
}

// 为每个标签页创建数据管理器
const dataManagers = {
    users: { loader: createUserLoader(), search: createUserSearch() },
    groups: { loader: createGroupLoader(), search: createGroupSearch() },
    moments: { loader: createMomentLoader(), search: createMomentSearch() }
}

// 当前活动的数据管理器
const currentManager = computed(() => dataManagers[currentTab.value as keyof typeof dataManagers])

// 统一的数据和状态访问
const currentData = computed(() => {
    if (!currentManager.value) return []
    const { loader, search } = currentManager.value
    return search.query.value ? search.results.value : loader.data.value
})

const currentLoading = computed(() => {
    if (!currentManager.value) return false
    const { loader, search } = currentManager.value
    return search.query.value ? search.searching.value : loader.loading.value
})

const hasMore = computed(() => currentManager.value?.loader?.hasMore?.value ?? false)
const loadingMore = computed(() => currentManager.value?.loader?.loadingMore?.value ?? false)

const currentSearchQuery = computed({
    get: () => currentManager.value?.search?.query?.value ?? '',
    set: (value) => { 
        if (currentManager.value?.search?.query) {
            currentManager.value.search.query.value = value
        }
    }
})

// 操作函数
const fetchNearbyData = () => {
    if (currentManager.value?.loader?.refresh) {
        currentManager.value.loader.refresh()
    }
}
const handleSearch = () => {
    if (currentManager.value?.search?.performSearch) {
        currentManager.value.search.performSearch()
    }
}
const loadMore = () => {
    if (currentManager.value?.loader?.loadMore) {
        currentManager.value.loader.loadMore()
    }
}

const refreshLocation = async () => {
    refreshing.value = true
    try {
        // 刷新位置逻辑
        await fetchNearbyData()
        toast.success('位置已刷新')
    } catch (error) {
        toast.error('刷新失败')
    } finally {
        refreshing.value = false
    }
}

const toggleLocationSharing = () => {
    isLocationSharing.value = !isLocationSharing.value
    toast.success(isLocationSharing.value ? '开始共享位置' : '停止共享位置')
}

const handleJoinGroup = async (group: Group) => {
    try {
        // await groupService.joinGroup(group.group_id)
        toast.success('加入群组成功')
    } catch (error) {
        toast.error('加入群组失败')
    }
}

const handleLikeMoment = async (moment: Moment) => {
    try {
        // await momentService.likeMoment(moment.moment_id)
        toast.success('点赞成功')
    } catch (error) {
        toast.error('点赞失败')
    }
}

// 监听标签页切换
watch(currentTab, () => {
    // 清空搜索
    if (currentManager.value?.search?.clear) {
        currentManager.value.search.clear()
    }
    // 加载数据
    fetchNearbyData()
})

// 页面加载
onMounted(() => {
    fetchNearbyData()
})
</script>