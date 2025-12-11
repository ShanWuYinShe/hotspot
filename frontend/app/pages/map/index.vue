<template>
    <div class="max-w-7xl mx-auto px-4 py-4 sm:py-6">
        <!-- 页面标题 -->
        <UPageHeader
            title="地图探索"
            description="发现附近有趣的用户、群组和精彩动态，探索你周围的世界"
            icon="i-heroicons-map"
            class="mb-6"
        />

        <!-- 标签页切换 -->
        <UTabs v-model="currentTab" :items="tabs" class="mb-6" />

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
        <div class="h-96 mb-6">
            <MapContainer
                :markers="mapMarkers"
                :center="currentLocation"
                :show-legend="true"
                @marker-click="handleMarkerClick"
                @location-update="handleLocationUpdate"
            />
        </div>

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
            <UEmpty
                v-else-if="currentData.length === 0"
                :icon="tabs.find(t => t.value === currentTab)?.icon || 'i-heroicons-map'"
                :title="`暂无${tabs.find(t => t.value === currentTab)?.label}`"
                :description="`附近没有找到相关的${tabs.find(t => t.value === currentTab)?.label}`"
            />
            <template v-else>
                <!-- 用户列表 -->
                <UserCard
                    v-if="currentTab === 'users'"
                    v-for="user in (currentData as UserSearchItem[])"
                    :key="user.user_id"
                    :user="user"
                    @click="user.user_id === authStore.user?.user_id ? navigateTo('/profile') : navigateTo(`/users/${user.user_id}`)"
                />

                <!-- 群组列表 -->
                <GroupCard
                    v-if="currentTab === 'groups'"
                    v-for="group in (currentData as GroupSearchItem[])"
                    :key="group.id"
                    :group="group"
                    @click="navigateTo(`/groups/${group.id}`)"
                    @join="handleJoinGroup"
                />
                
                <!-- 动态列表 -->
                <MomentCard
                    v-if="currentTab === 'moments'"
                    v-for="moment in (currentData as MomentSearchItem[])"
                    :key="moment.id"
                    :moment="moment"
                    @click="navigateTo(`/moments/${moment.id}`)"
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
import { useUsersService } from '~/services/users.service';
import { useGroupsService } from '~/services/groups.service';
import { useMomentsService } from '~/services/moments.service';
import { getPageSeoConfig, buildSeoMeta } from '~/config/seo';
import type { UserSearchItem, GroupSearchItem, MomentSearchItem } from '~/types/api'
import type { MapMarker } from '~/components/map/MapContainer.vue'

// 从数据源获取 SEO 配置
const seoConfig = getPageSeoConfig('map')

definePageMeta({
    title: seoConfig.title,
    description: seoConfig.description,
    layout: 'dashboard'
})

// 设置 SEO meta
useSeoMeta(buildSeoMeta('map'))

// 基础状态
const currentTab = ref('users')
const isLocationSharing = ref(false)
const refreshing = ref(false)
const currentLoading = ref(false)
const loadingMore = ref(false)
const hasMore = ref(false)
const currentSearchQuery = ref('')
const currentData = ref<(UserSearchItem | GroupSearchItem | MomentSearchItem)[]>([])

// 位置服务
const locationStore = useLocationStore()
const authStore = useAuthStore()

// 地图相关状态 - 从 locationStore 获取
const currentLocation = computed(() => locationStore.coordinates || { latitude: 35.154922, longitude: 112.682524 })
const mapMarkers = ref<any[]>([])

// 类型守卫函数
const isUserItem = (item: UserSearchItem | GroupSearchItem | MomentSearchItem): item is UserSearchItem => {
    return 'user_id' in item && 'nickname' in item
}

const isGroupItem = (item: UserSearchItem | GroupSearchItem | MomentSearchItem): item is GroupSearchItem => {
    return 'members_count' in item && 'name' in item && !('user' in item)
}

const isMomentItem = (item: UserSearchItem | GroupSearchItem | MomentSearchItem): item is MomentSearchItem => {
    return 'id' in item && 'description' in item && 'user' in item
}

// 计算地图标记
const updateMapMarkers = () => {
    mapMarkers.value = currentData.value.map((item, index) => {
        let type: 'user' | 'group' | 'moment' = 'user'
        let title = ''
        let description = ''
        let avatar = ''
        let id = `marker-${index}`
        let latitude = currentLocation.value.latitude
        let longitude = currentLocation.value.longitude
        let distance = 0
        
        if (currentTab.value === 'users' && isUserItem(item)) {
            type = 'user'
            title = item.nickname || '用户'
            description = item.geo.distance ? `距离 ${Math.round(item.geo.distance)}m` : '位置未知'
            avatar = item.avatar_url || ''
            id = String(item.user_id)
            latitude = item.geo.latitude
            longitude = item.geo.longitude
            distance = item.geo.distance
        } else if (currentTab.value === 'groups' && isGroupItem(item)) {
            type = 'group'
            title = item.name || '群组'
            description = `${item.members_count || 0} 名成员`
            avatar = item.avatar_url || ''
            id = item.id
            latitude = item.geo.latitude
            longitude = item.geo.longitude
            distance = item.geo.distance
        } else if (currentTab.value === 'moments' && isMomentItem(item)) {
            type = 'moment'
            title = item.description || '动态'
            description = `来自 ${item.user?.nickname || '用户'}`
            avatar = ''
            id = item.id
            latitude = item.geo.latitude
            longitude = item.geo.longitude
            distance = item.geo.distance
        }
        
        return {
            id,
            type,
            title,
            description,
            latitude,
            longitude,
            avatar,
            distance,
            data: item
        }
    })
}

// 标签页配置
const tabs = [
    { value: 'users', label: '用户', icon: 'i-heroicons-users' },
    { value: 'groups', label: '群组', icon: 'i-heroicons-user-group' },
    { value: 'moments', label: '动态', icon: 'i-heroicons-photo' }
]

// 服务和工具
const userService = useUsersService()
const groupService = useGroupsService()
const toast = useToast()
const geolocation = useGeolocation()

// 数据加载函数
const loadNearbyUsers = async () => {
    try {
        currentLoading.value = true
        
        // 使用 locationStore 的位置
        const pos = currentLocation.value
        
        const result = await userService.searchByLocation({
            latitude: pos.latitude,
            longitude: pos.longitude,
            radius: 5000,
            limit: 20
        })
        
        currentData.value = result.items
        hasMore.value = result.pagination.has_more
    } catch (error) {
        console.error('加载附近用户失败:', error)
        toast.add({
            title: '加载失败',
            description: '无法获取附近用户信息',
            color: 'error'
        })
    } finally {
        currentLoading.value = false
    }
}

const loadNearbyGroups = async () => {
    try {
        currentLoading.value = true
        
        // 使用 locationStore 的位置
        const pos = currentLocation.value
        
        const result = await groupService.searchByLocation({
            latitude: pos.latitude,
            longitude: pos.longitude,
            radius: 5000,
            limit: 20
        })
        
        currentData.value = result.items
        hasMore.value = result.pagination.has_more
    } catch (error) {
        console.error('加载附近群组失败:', error)
        toast.add({
            title: '加载失败',
            description: '无法获取附近群组信息',
            color: 'error'
        })
    } finally {
        currentLoading.value = false
    }
}

const loadNearbyMoments = async () => {
    try {
        currentLoading.value = true
        
        // 使用 locationStore 的位置
        const pos = currentLocation.value
        
        // 加载附近动态
        const momentsService = useMomentsService();
        const response = await momentsService.searchByLocation({
            latitude: pos.latitude,
            longitude: pos.longitude,
            radius: 5000,
            limit: 20
        });
        currentData.value = response.items || [];
        hasMore.value = response.pagination.has_more || false;
        
        toast.add({
            title: '功能开发中',
            description: '动态功能即将上线',
            color: 'info'
        })
    } catch (error) {
        console.error('加载附近动态失败:', error)
        toast.add({
            title: '加载失败',
            description: '无法获取附近动态信息',
            color: 'error'
        })
    } finally {
        currentLoading.value = false
    }
}

// 地图事件处理
const handleMarkerClick = (marker: MapMarker) => {
    const item = marker.data || {}
    if (currentTab.value === 'users') {
        // 如果是当前用户，直接跳转到个人主页
        if (item.user_id === authStore.user?.user_id) {
            navigateTo('/profile')
        } else {
            navigateTo(`/users/${item.user_id}`)
        }
    } else if (currentTab.value === 'groups') {
        navigateTo(`/groups/${item.group_id}`)
    } else if (currentTab.value === 'moments') {
        navigateTo(`/moments/${item.id}`)
    }
}

const handleLocationUpdate = (location: { latitude: number; longitude: number }) => {
    // 更新 locationStore
    locationStore.setLocation(location.latitude, location.longitude)
    // 重新加载附近数据
    fetchNearbyData()
}

// 操作函数
const fetchNearbyData = () => {
    switch (currentTab.value) {
        case 'users':
            loadNearbyUsers()
            break
        case 'groups':
            loadNearbyGroups()
            break
        case 'moments':
            loadNearbyMoments()
            break
    }
}

const handleSearch = async () => {
    if (!currentSearchQuery.value.trim()) {
        fetchNearbyData()
        return
    }
    
    try {
        currentLoading.value = true
        
        // 获取位置（用于搜索结果的距离计算和排序）
        const location = await geolocation.getCoordinates()
        
        switch (currentTab.value) {
            case 'users':
                // 全局搜索用户，位置仅用于计算距离，不限范围
                const userResult = await userService.searchByKeyword({
                    kw: currentSearchQuery.value.trim(),
                    latitude: location.latitude,
                    longitude: location.longitude,
                    limit: 20
                })
                currentData.value = userResult.items
                hasMore.value = userResult.pagination.has_more
                break
            case 'groups':
                // 全局搜索群组，位置仅用于计算距离，不限范围
                const groupResult = await groupService.searchByKeyword({
                    kw: currentSearchQuery.value.trim(),
                    latitude: location.latitude,
                    longitude: location.longitude,
                    limit: 20
                })
                currentData.value = groupResult.items
                hasMore.value = groupResult.pagination.has_more
                break
            case 'moments':
                // 动态搜索
                const momentsService = useMomentsService()
                const momentResult = await momentsService.searchByKeyword({
                    kw: currentSearchQuery.value.trim(),
                    latitude: location.latitude,
                    longitude: location.longitude,
                    limit: 20
                })
                currentData.value = momentResult.items
                hasMore.value = momentResult.pagination.has_more
                break
        }
    } catch (error) {
        console.error('搜索失败:', error)
        toast.add({
            title: '搜索失败',
            description: '请重试',
            color: 'error'
        })
    } finally {
        currentLoading.value = false
    }
}

const loadMore = () => {
    // 加载更多功能暂未实现
    toast.add({
        title: '功能开发中',
        description: '加载更多功能即将上线',
        color: 'info'
    })
}

const refreshLocation = async () => {
    refreshing.value = true
    try {
        await fetchNearbyData()
        toast.add({
            title: '位置已刷新',
            description: '已重新加载附近数据',
            color: 'success'
        })
    } catch (error) {
        toast.add({
            title: '刷新失败',
            description: '请重试',
            color: 'error'
        })
    } finally {
        refreshing.value = false
    }
}

const toggleLocationSharing = () => {
    isLocationSharing.value = !isLocationSharing.value
    toast.add({
        title: isLocationSharing.value ? '开始共享位置' : '停止共享位置',
        description: isLocationSharing.value ? '其他用户可以看到你的位置' : '已停止位置共享',
        color: 'success'
    })
}

const handleJoinGroup = async (group: { id: string; name?: string }) => {
    try {
        await groupService.joinGroup({
            id: group.id
        })
        toast.add({
            title: '加入群组成功',
            description: `已成功加入群组 "${group.name}"`,
            color: 'success'
        })
        // 刷新数据
        fetchNearbyData()
    } catch (error) {
        toast.add({
            title: '加入群组失败',
            description: '请重试',
            color: 'error'
        })
    }
}

const handleLikeMoment = async (moment: MomentSearchItem) => {
    // ✅ 功能级认证检查：游客可以浏览，但不能点赞
    const { requireAuth } = useRequireAuth()
    if (!requireAuth('点赞')) return

    try {
        const momentsService = useMomentsService()
        // 先检查是否已点赞
        const { is_liked } = await momentsService.checkLiked({ id: moment.id })
        
        if (is_liked) {
            await momentsService.unlikeMoment({ id: moment.id })
            moment.likes_count -= 1
            toast.add({ title: '取消点赞', color: 'success' })
        } else {
            await momentsService.likeMoment({ id: moment.id })
            moment.likes_count += 1
            toast.add({ title: '点赞成功', color: 'success' })
        }
    } catch (error) {
        console.error('点赞失败:', error)
        toast.add({
            title: '操作失败',
            description: (error as Error).message,
            color: 'error'
        })
    }
}

// 监听标签页切换
watch(currentTab, () => {
    // 清空搜索和数据
    currentSearchQuery.value = ''
    currentData.value = []
    // 加载新数据
    fetchNearbyData()
})

// 监听数据变化，更新地图标记
watch(currentData, () => {
    updateMapMarkers()
}, { deep: true })

// 页面加载
onMounted(async () => {
    // 如果 locationStore 没有位置，获取并更新
    if (!locationStore.coordinates) {
        try {
            const coords = await geolocation.getCoordinates()
            let locationName = '未知位置'
            try {
                const { geocoding } = await import('~/utils/geocoding')
                locationName = await geocoding.getLocationName(coords.latitude, coords.longitude)
            } catch {
                // 使用默认名称
            }
            locationStore.setLocation(coords.latitude, coords.longitude, locationName)
        } catch {
            // 获取失败，使用默认位置
        }
    }
    fetchNearbyData()
})
</script>