<template>
    <div class="max-w-7xl mx-auto px-4 py-8">
        <!-- 页面标题 -->
        <div class="mb-8">
            <div class="flex items-center gap-3 mb-3">
                <div class="w-10 h-10 bg-gradient-to-br from-emerald-500 to-teal-600 rounded-xl flex items-center justify-center shadow-lg">
                    <UIcon name="i-heroicons-map" class="w-6 h-6 text-white" />
                </div>
                <div>
                    <h1 class="text-3xl font-bold text-black dark:text-white">地图探索</h1>
                    <div class="w-16 h-0.5 bg-gradient-to-r from-emerald-500 to-teal-500 rounded-full mt-1"></div>
                </div>
            </div>
            <p class="text-secondary text-lg leading-relaxed">发现附近有趣的用户、群组和精彩动态，探索你周围的世界</p>
        </div>

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
            <div
                v-else-if="currentData.length === 0"
                class="text-center py-12"
            >
                <UIcon 
                    :name="tabs.find(t => t.value === currentTab)?.icon || 'i-heroicons-map'" 
                    class="w-16 h-16 text-neutral-400 mx-auto mb-4" 
                />
                <h3 class="text-lg font-semibold text-neutral-600 dark:text-neutral-400 mb-2">
                    暂无{{ tabs.find(t => t.value === currentTab)?.label }}
                </h3>
                <p class="text-neutral-500">
                    附近没有找到相关的{{ tabs.find(t => t.value === currentTab)?.label }}
                </p>
            </div>
            <template v-else>
                <!-- 用户列表 -->
                <UserCard
                    v-if="currentTab === 'users'"
                    v-for="user in (currentData as UserResponse[])"
                    :key="user.user_id"
                    :user="user"
                    @click="navigateTo(`/users/${user.user_id}`)"
                />

                <!-- 群组列表 -->
                <GroupCard
                    v-if="currentTab === 'groups'"
                    v-for="group in (currentData as GroupResponse[])"
                    :key="group.group_id"
                    :group="group"
                    @click="navigateTo(`/groups/${group.group_id}`)"
                    @join="handleJoinGroup"
                />
                
                <!-- 动态列表 -->
                <MomentCard
                    v-if="currentTab === 'moments'"
                    v-for="moment in (currentData as MomentResponse[])"
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
import { useUsersService } from '~/services/users.service';
import { useGroupsService } from '~/services/groups.service';
import { useMomentsService } from '~/services/moments.service';
import { getPageSeoConfig, buildSeoMeta } from '~/config/seo';
import type { UserResponse, GroupResponse, MomentResponse } from '~/types/api'

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
const currentData = ref<(UserResponse | GroupResponse | MomentResponse)[]>([])

// 地图相关状态
const currentLocation = ref({ latitude: 0, longitude: 0 }) // 将通过位置服务获取
const mapMarkers = ref<any[]>([])

// 计算地图标记
const updateMapMarkers = () => {
    mapMarkers.value = currentData.value.map((item, index) => {
        let type: 'user' | 'group' | 'moment' = 'user'
        let title = ''
        let description = ''
        let avatar = ''
        
        if (currentTab.value === 'users') {
            const user = item as UserResponse
            type = 'user'
            title = user.nickname || '用户'
            description = user.distance ? `距离 ${Math.round(user.distance)}m` : '位置未知'
            avatar = user.avatar_url || ''
        } else if (currentTab.value === 'groups') {
            const group = item as GroupResponse
            type = 'group'
            title = group.name || '群组'
            description = `${group.member_count || 0} 名成员`
            avatar = group.avatar_url || ''
        } else if (currentTab.value === 'moments') {
            const moment = item as MomentResponse
            type = 'moment'
            title = moment.description || '动态'
            description = `来自 ${moment.nickname || '用户'}`
            avatar = ''
        }
        
        return {
            id: (item as any).user_id || (item as any).group_id || (item as any).moment_id || `marker-${index}`,
            type,
            title,
            description,
            latitude: item.latitude || currentLocation.value.latitude,
            longitude: item.longitude || currentLocation.value.longitude,
            avatar,
            distance: item.distance || 0,
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
const authStore = useAuthStore()
const locationComposable = useLocationStore()

// 数据加载函数
const loadNearbyUsers = async () => {
    try {
        currentLoading.value = true
        
        // 获取当前位置（失败时使用默认位置）
        let latitude: number;
        let longitude: number;
        
        if (locationComposable.coordinates) {
            latitude = locationComposable.coordinates.latitude;
            longitude = locationComposable.coordinates.longitude;
        } else {
            // 使用系统默认位置（南天门）
            latitude = 35.154922;
            longitude = 112.682524;
        }
        
        currentLocation.value = { latitude, longitude };
        
        const result = await userService.getNearbyUsers({
            latitude,
            longitude,
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
        
        // 获取当前位置（失败时使用默认位置）
        let latitude: number;
        let longitude: number;
        
        if (locationComposable.coordinates) {
            latitude = locationComposable.coordinates.latitude;
            longitude = locationComposable.coordinates.longitude;
        } else {
            // 使用系统默认位置（南天门）
            latitude = 35.154922;
            longitude = 112.682524;
        }
        
        currentLocation.value = { latitude, longitude };
        
        const result = await groupService.getNearbyGroups({
            latitude,
            longitude,
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
        
        // 获取当前位置（失败时使用默认位置）
        let latitude: number;
        let longitude: number;
        
        if (locationComposable.coordinates) {
            latitude = locationComposable.coordinates.latitude;
            longitude = locationComposable.coordinates.longitude;
        } else {
            // 使用系统默认位置（南天门）
            latitude = 35.154922;
            longitude = 112.682524;
        }
        
        currentLocation.value = { latitude, longitude };
        
        // 加载附近动态
        const momentsService = useMomentsService();
        const response = await momentsService.getNearbyMoments({
            latitude,
            longitude,
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
const handleMarkerClick = (marker: any) => {
    const item = marker.data
    if (currentTab.value === 'users') {
        navigateTo(`/users/${item.user_id}`)
    } else if (currentTab.value === 'groups') {
        navigateTo(`/groups/${item.group_id}`)
    } else if (currentTab.value === 'moments') {
        navigateTo(`/moments/${item.moment_id}`)
    }
}

const handleLocationUpdate = (location: { latitude: number; longitude: number }) => {
    currentLocation.value = location
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
        
        switch (currentTab.value) {
            case 'users':
                const userResult = await userService.searchUsers({
                    keyword: currentSearchQuery.value.trim(),
                    limit: 20
                })
                currentData.value = userResult.items
                hasMore.value = userResult.pagination.has_more
                break
            case 'groups':
                const groupResult = await groupService.searchGroups({
                    keyword: currentSearchQuery.value.trim(),
                    limit: 20
                })
                currentData.value = groupResult.items
                hasMore.value = groupResult.pagination.has_more
                break
            case 'moments':
                // 动态搜索功能暂未实现
                currentData.value = []
                toast.add({
                    title: '功能开发中',
                    description: '动态搜索功能即将上线',
                    color: 'info'
                })
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

const handleJoinGroup = async (group: GroupResponse) => {
    try {
        await groupService.joinGroup({
            group_chat_id: group.group_id
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

const handleLikeMoment = async (moment: MomentResponse) => {
    // ✅ 功能级认证检查：游客可以浏览，但不能点赞
    const { requireAuth } = useRequireAuth()
    if (!requireAuth('点赞')) return

    try {
        const momentsService = useMomentsService()
        await momentsService.toggleLike({
            id: moment.moment_id,
            action: moment.liked_by_me ? 'unlike' : 'like'
        })
        
        // 更新本地状态
        moment.liked_by_me = !moment.liked_by_me
        moment.likes_count += moment.liked_by_me ? 1 : -1
        
        toast.add({
            title: moment.liked_by_me ? '点赞成功' : '取消点赞',
            color: 'success'
        })
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
onMounted(() => {
    fetchNearbyData()
})
</script>