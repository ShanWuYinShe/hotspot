<template>
    <div class="min-h-screen bg-gradient-slate-page">
        <!-- 背景装饰 -->
        <div class="absolute inset-0 overflow-hidden pointer-events-none">
            <div class="absolute -top-40 -right-40 w-80 h-80 bg-linear-to-br from-slate-300/10 to-slate-400/10 rounded-full blur-3xl"></div>
            <div class="absolute -bottom-40 -left-40 w-80 h-80 bg-linear-to-tr from-slate-300/10 to-slate-400/10 rounded-full blur-3xl"></div>
        </div>

        <div class="relative max-w-6xl mx-auto px-4 sm:px-6 lg:px-8 py-4 sm:py-6">
            <!-- 页面标题和操作栏 -->
            <UPageHeader
                title="动态广场"
                class="mb-6"
            >
                <template #actions>
                    <UButton
                        @click="openCreateModal"
                        icon="i-heroicons-plus"
                        color="primary"
                    >
                        <span class="hidden sm:inline">发布动态</span>
                        <span class="sm:hidden">发布</span>
                    </UButton>
                </template>
            </UPageHeader>

            <!-- 视图切换 -->
            <div class="mb-6">
                <UTabs
                    :items="tabItems"
                    v-model="currentView"
                    :default-index="0"
                />
            </div>

            <!-- 附近动态筛选栏 -->
            <UCard v-if="currentView === 'nearby'" class="mb-6 border-0 shadow-lg bg-glass-white backdrop-blur-sm">
                <div class="flex flex-col sm:flex-row gap-4 items-center">
                    <div class="flex items-center gap-3 w-full sm:w-auto">
                        <span class="text-sm text-secondary-content whitespace-nowrap">搜索半径</span>
                        <USelect
                            v-model="selectedRadius"
                            :items="radiusOptions"
                            class="w-32"
                            @change="refreshCurrentView"
                        />
                    </div>
                    <div class="flex-1" />
                    <UButton variant="outline" icon="i-heroicons-arrow-path" :loading="loading.nearby" @click="refreshCurrentView">
                        刷新
                    </UButton>
                </div>
            </UCard>

            <!-- 搜索和筛选栏 -->
            <div v-if="currentView === 'search'" class="mb-6">
                <UCard class=" border ">
                    <div class="space-y-6">
                        <div class="flex flex-col sm:flex-row gap-4">
                            <UInput
                                v-model="searchQuery"
                                placeholder="搜索动态内容或标签..."
                                icon="i-heroicons-magnifying-glass"
                                size="lg"
                                class="flex-1"
                            />
                            <UButton
                                @click="async () => { await searchMoments(searchQuery); }"
                                :loading="loading.search"
                                size="lg"
                                color="primary"
                            >
                                <UIcon name="i-heroicons-magnifying-glass" class="w-5 h-5 mr-2" />
                                搜索
                            </UButton>
                        </div>

                        <!-- 热门标签 -->
                        <div class="space-y-3">
                            <h3 class="text-sm font-semibold text-secondary-strong flex items-center">
                                <UIcon name="i-heroicons-hashtag" class="w-4 h-4 mr-2" />
                                热门标签
                            </h3>
                            <div class="flex flex-wrap gap-2">
                                <UButton
                                    v-for="tag in popularTags"
                                    :key="tag"
                                    variant="soft"
                                    size="sm"
                                    :color="selectedTags.includes(tag) ? 'primary' : 'neutral'"
                                    @click="toggleTag(tag)"
                                >
                                    #{{ tag }}
                                </UButton>
                            </div>
                        </div>
                    </div>
                </UCard>
            </div>


            <!-- 关注视图刷新按钮 -->
            <div v-if="currentView === 'following'" class="flex justify-end mb-4">
                <UButton variant="outline" icon="i-heroicons-arrow-path" :loading="loading.following" @click="refreshCurrentView">
                    刷新
                </UButton>
            </div>

            <!-- 加载状态 -->
            <div v-if="currentLoading && currentData.length === 0" class="grid gap-4">
                <USkeleton v-for="i in 3" :key="i" class="h-48 rounded-2xl" />
            </div>

            <!-- 动态列表 -->
            <div v-else-if="currentData.length > 0" class="grid gap-4">
                <MomentCard
                    v-for="moment in currentData"
                    :key="moment.id"
                    :moment="moment"
                    :show-actions="true"
                    @click="navigateTo(`/moments/${moment.id}`)"
                    @like="toggleLike(moment as unknown as Moment)"
                    @comment="navigateTo(`/moments/${moment.id}#comments`)"
                    @share="handleShare(moment)"
                    @tag-click="toggleTag"
                    @deleted="refreshCurrentView"
                />
            </div>

            <!-- 空状态 -->
            <UEmpty v-else icon="i-heroicons-photo" title="暂无动态" description="成为第一个分享生活瞬间的人">
                <template #actions>
                    <UButton icon="i-heroicons-plus" @click="openCreateModal">发布动态</UButton>
                </template>
            </UEmpty>
            
            <!-- 加载更多按钮 -->
            <div v-if="hasMoreData && currentData.length > 0" class="flex justify-center mt-8">
                <UButton
                    @click="loadMore"
                    :loading="loadingMore"
                    variant="outline"
                    size="lg"
                    class="px-8"
                >
                    <UIcon name="i-heroicons-arrow-down" class="w-4 h-4 mr-2" />
                    加载更多
                </UButton>
            </div>
            
            <!-- 已显示全部提示 -->
            <div v-else-if="currentData.length > 0 && !hasMoreData" class="text-center mt-8 text-sm text-secondary-content">
                已显示全部 {{ currentData.length }} 条动态
            </div>
        </div>
    </div>
</template>

<script setup lang="ts">
import { useMomentsService } from '~/services/moments.service';
import { useFeedService } from '~/services/feed.service';
import type { MomentSearchItem } from '~/types/api'
import type { Moment } from '~/stores/moments.store'

definePageMeta({
    title: '动态广场',
    description: '发现和分享生活中的精彩瞬间',
    layout: 'dashboard'
})

// 基础状态
const currentView = ref('nearby')
const selectedTags = ref<string[]>([])
const { formatTime, formatDistance } = useFormatters()

// 位置服务
const locationStore = useLocationStore()

// 搜索范围
const selectedRadius = ref(5000)

const radiusOptions = [
    { label: '1 公里', value: 1000 },
    { label: '5 公里', value: 5000 },
    { label: '10 公里', value: 10000 },
    { label: '25 公里', value: 25000 },
    { label: '50 公里', value: 50000 }
]

// 配置
const tabItems = [
    { value: 'nearby', label: '附近', icon: 'i-heroicons-map-pin' },
    { value: 'following', label: '关注', icon: 'i-heroicons-users' },
    { value: 'search', label: '搜索', icon: 'i-heroicons-magnifying-glass' }
]

const popularTags = ref(['美食', '旅行', '生活', '摄影', '音乐', '运动'])

// 服务
const momentService = useMomentsService()
const feedService = useFeedService()
const geolocation = useGeolocation()
const toast = useToast()

// 创建动态功能
const modal = useModal()

const openCreateModal = async () => {
  const momentId = await modal.showCreateMoment()
  if (momentId) {
    refreshCurrentView()
    toast.add({ color: 'success', title: '动态发布成功' })
  }
}

// 刷新当前视图数据
const refreshCurrentView = () => {
    if (currentView.value === 'nearby') {
        loadNearbyMoments();
    } else if (currentView.value === 'following') {
        loadFollowingMoments();
    }
};

// 数据状态
const nearbyMoments = ref<MomentSearchItem[]>([]);
const followingMoments = ref<MomentSearchItem[]>([]);
const searchResults = ref<MomentSearchItem[]>([]);
const loading = ref({
    nearby: false,
    following: false,
    search: false
});

const searchQuery = ref('');
const hasMore = ref({
    nearby: false,
    following: false
});

// 数据加载函数
const loadNearbyMoments = async (cursor?: string) => {
    loading.value.nearby = true;
    try {
        // 使用 locationStore 的位置，如果没有则获取并更新 store
        let pos = locationStore.coordinates;
        if (!pos) {
            const coords = await geolocation.getCoordinates();
            // 尝试获取位置名称并更新 store
            let locationName = '未知位置';
            try {
                const { geocoding } = await import('~/utils/geocoding');
                locationName = await geocoding.getLocationName(coords.latitude, coords.longitude);
            } catch {
                // 使用默认名称
            }
            locationStore.setLocation(coords.latitude, coords.longitude, locationName);
            pos = coords;
        }

        const result = await momentService.searchByLocation({
            latitude: pos.latitude,
            longitude: pos.longitude,
            radius: selectedRadius.value,
            limit: 20,
            cursor
        });
        
        if (cursor) {
            nearbyMoments.value.push(...result.items);
        } else {
            nearbyMoments.value = result.items;
        }
        
        hasMore.value.nearby = result.pagination.has_more;
        return result;
    } catch (error) {
        console.error('加载附近动态失败:', error);
        toast.add({ color: 'error', title: '加载附近动态失败' });
        return { items: [], hasMore: false, nextCursor: undefined };
    } finally {
        loading.value.nearby = false;
    }
};

const loadFollowingMoments = async (cursor?: string) => {
    loading.value.following = true;
    try {
        // 使用Feed API获取关注用户的动态
        const feedService = useFeedService()
        const result = await feedService.getMoments({
            source: 'following',
            limit: 20,
            cursor
        });
        
        // FeedMomentItem 继承自 MomentSearchItem，可以直接使用
        const moments = result.items as MomentSearchItem[];
        
        if (cursor) {
            followingMoments.value.push(...moments);
        } else {
            followingMoments.value = moments;
        }
        
        hasMore.value.following = result.pagination.has_more;
        return result;
    } catch (error) {
        console.error('加载关注动态失败:', error);
        toast.add({ color: 'error', title: '加载关注动态失败' });
        return { items: [], pagination: { has_more: false } };
    } finally {
        loading.value.following = false;
    }
};

const searchMoments = async (query: string, filters?: Record<string, unknown>) => {
    if (!query.trim()) {
        searchResults.value = [];
        return [];
    }
    
    loading.value.search = true;
    try {
        const coords = await geolocation.getCoordinates();
        
        // 使用关键字搜索
        const result = await momentService.searchByKeyword({
            kw: query.trim(),
            latitude: coords.latitude,
            longitude: coords.longitude,
            limit: 20
        });
        searchResults.value = result.items;
        
        if (result.items.length === 0) {
            toast.add({ color: 'info', title: '未找到相关动态' });
        }
        
        return result.items;
    } catch (error) {
        console.error('搜索动态失败:', error);
        toast.add({ color: 'error', title: '搜索失败，请重试' });
        return [];
    } finally {
        loading.value.search = false;
    }
};


// 计算属性
const currentData = computed(() => {
    if (currentView.value === 'search') {
        return searchResults.value;
    }
    if (currentView.value === 'nearby') {
        return nearbyMoments.value;
    }
    if (currentView.value === 'following') {
        return followingMoments.value;
    }
    return [];
});

const currentLoading = computed(() => {
    return loading.value[currentView.value as keyof typeof loading.value] || false;
});

const hasMoreData = computed(() => {
    if (currentView.value === 'search') return false;
    return hasMore.value[currentView.value as keyof typeof hasMore.value] || false;
});

const loadingMore = ref(false);


const handleShare = async (moment: MomentSearchItem) => {
    try {
        if (navigator.share) {
            await navigator.share({
                title: `${moment.user.nickname}的动态`,
                text: moment.description,
                url: `${window.location.origin}/moments/${moment.id}`
            })
        } else {
            // 复制链接到剪贴板
            await navigator.clipboard.writeText(`${window.location.origin}/moments/${moment.id}`)
            toast.add({ color: 'success', title: '链接已复制到剪贴板' })
        }
    } catch (error) {
        console.error('分享失败:', error)
    }
}

// 操作函数
const toggleTag = (tag: string) => {
    const index = selectedTags.value.indexOf(tag)
    if (index > -1) {
        selectedTags.value.splice(index, 1)
    } else {
        selectedTags.value.push(tag)
    }

    // 标签变化时重新搜索
    if (searchQuery.value.trim()) {
        searchMoments(searchQuery.value);
    }
}

const toggleLike = async (moment: Moment) => {
    try {
        // 乐观更新
        const wasLiked = moment.liked_by_me
        moment.liked_by_me = !moment.liked_by_me
        moment.likes_count += moment.liked_by_me ? 1 : -1

        // 根据当前状态调用对应 API
        if (wasLiked) {
            await momentService.unlikeMoment({ id: moment.id })
        } else {
            await momentService.likeMoment({ id: moment.id })
        }
    } catch (error) {
        // 回滚状态
        moment.liked_by_me = !moment.liked_by_me
        moment.likes_count += moment.liked_by_me ? 1 : -1
        toast.add({ color: 'error', title: '操作失败' })
    }
}

const loadMore = async () => {
    if (loadingMore.value || !hasMoreData.value) return;
    
    loadingMore.value = true
    try {
        let result
        const lastItem = currentData.value[currentData.value.length - 1]
        const cursor = lastItem?.id
        
        if (currentView.value === 'nearby') {
            result = await loadNearbyMoments(cursor);
        } else if (currentView.value === 'following') {
            result = await loadFollowingMoments(cursor);
        }
        
        if (result && result.items.length === 0) {
            toast.add({ color: 'info', title: '没有更多动态了' });
        }
    } catch (error) {
        console.error('加载更多失败:', error);
        toast.add({ color: 'error', title: '加载更多失败' });
    } finally {
        loadingMore.value = false;
    }
};

// 监听视图切换
watch(currentView, () => {
    if (currentView.value === 'search') {
        searchResults.value = [];
        searchQuery.value = '';
    } else {
        refreshCurrentView();
    }
});

// 页面加载
onMounted(() => {
    if (currentView.value !== 'search') {
        refreshCurrentView();
    }
});
</script>
