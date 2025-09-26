<template>
    <div class="min-h-screen bg-gradient-to-br from-slate-50 via-slate-100/50 to-slate-50 dark:from-slate-900 dark:via-slate-800/30 dark:to-slate-900">
        <!-- 背景装饰 -->
        <div class="absolute inset-0 overflow-hidden pointer-events-none">
            <div class="absolute -top-40 -right-40 w-80 h-80 bg-gradient-to-br from-slate-300/10 to-slate-400/10 rounded-full blur-3xl"></div>
            <div class="absolute -bottom-40 -left-40 w-80 h-80 bg-gradient-to-tr from-slate-300/10 to-slate-400/10 rounded-full blur-3xl"></div>
        </div>

        <div class="relative max-w-6xl mx-auto px-4 sm:px-6 lg:px-8 py-8">
            <!-- 页面标题和操作栏 -->
            <div class="flex flex-col sm:flex-row justify-between items-start sm:items-center gap-4 mb-8">
                <div class="space-y-2">
                    <h1 class="text-3xl sm:text-4xl font-bold text-slate-800 dark:text-slate-100">
                        动态广场
                    </h1>
                    <p class="text-slate-600 dark:text-slate-400 text-lg">发现和分享生活中的精彩瞬间</p>
                </div>
                <div class="flex gap-3 w-full sm:w-auto">
                    <UButton
                        @click="openCreateModal"
                        icon="i-heroicons-plus"
                        color="primary"
                        class="flex-1 sm:flex-none"
                    >
                        <span class="hidden sm:inline">发布动态</span>
                        <span class="sm:hidden">发布</span>
                    </UButton>
                </div>
            </div>

            <!-- 视图切换 -->
            <div class="mb-8">
                <UTabs
                    :items="tabItems"
                    v-model="currentView"
                    :default-index="0"
                />
            </div>

            <!-- 搜索和筛选栏 -->
            <div v-if="currentView === 'search'" class="mb-8">
                <UCard class="bg-white/60 dark:bg-slate-800/60 border border-slate-200/50 dark:border-slate-700/50">
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
                            <h3 class="text-sm font-semibold text-slate-700 dark:text-slate-300 flex items-center">
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


            <!-- 动态列表 -->
            <DataView
                :data="currentData"
                :loading="currentLoading"
                :searchable="currentView === 'search'"
                :search-fields="['description', 'tags', 'nickname']"
                :show-pagination="false"
                :toolbar-actions="dataViewActions"
                item-component="MomentCard"
                :get-item-key="(item) => item.moment_id"
                :get-item-props="getMomentCardProps"
                :get-item-events="getMomentCardEvents"
                empty-icon="i-heroicons-photo"
                empty-title="暂无动态"
                empty-description="成为第一个分享生活瞬间的人"
                :empty-action="{ label: '发布动态', action: openCreateModal }"
                @item-click="(moment) => navigateTo(`/moments/${moment.moment_id}`)"
                @search="handleSearch"
            />
        </div>
        
        <!-- 创建动态弹窗 -->
        <CreateMomentModal
            v-model:open="showCreateModal"
            @success="handleMomentCreated"
        />
    </div>
</template>

<script setup lang="ts">
import type { MomentResponse } from '~/types/api'
import { formatTime, formatDistance } from '~/utils/formatters'

definePageMeta({
    title: '动态广场',
    description: '发现和分享生活中的精彩瞬间',
    layout: 'dashboard'
})

// 基础状态
const currentView = ref('nearby')
const selectedTags = ref<string[]>([])

// 配置
const tabItems = [
    { value: 'nearby', label: '附近', icon: 'i-heroicons-map-pin' },
    { value: 'following', label: '关注', icon: 'i-heroicons-users' },
    { value: 'search', label: '搜索', icon: 'i-heroicons-magnifying-glass' }
]

const popularTags = ref(['美食', '旅行', '生活', '摄影', '音乐', '运动'])

// 服务
const momentService = useMomentService()
const authStore = useAuthStore()
const toast = useSmartToast()
const overlay = useOverlay()
const { executeFeature } = useFeatureStatus()

// 创建动态功能
const showCreateModal = ref(false)

const openCreateModal = () => {
  showCreateModal.value = true
}

const handleMomentCreated = (moment: any) => {
  // 刷新当前视图数据
  refreshCurrentView()
  toast.success('动态发布成功')
}

// 刷新当前视图数据
const refreshCurrentView = () => {
    if (currentView.value === 'nearby') {
        loadNearbyMoments();
    } else if (currentView.value === 'following') {
        loadFollowingMoments();
    }
};

// 创建成功处理暂时移除

// 数据状态
const nearbyMoments = ref<MomentResponse[]>([]);
const followingMoments = ref<MomentResponse[]>([]);
const searchResults = ref<MomentResponse[]>([]);
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
        const result = await momentService.searchMoments({
            type: 'location',
            latitude: 39.9042, // 默认位置，实际应从位置服务获取
            longitude: 116.4074,
            radius: 5000
        }, cursor, 20);
        if (cursor) {
            nearbyMoments.value.push(...result.items);
        } else {
            nearbyMoments.value = result.items;
        }
        return result;
    } finally {
        loading.value.nearby = false;
    }
};

const loadFollowingMoments = async (cursor?: string) => {
    loading.value.following = true;
    try {
        // 关注的用户动态 - 暂时返回空数据，因为好友功能未实现
        followingMoments.value = [];
        return { items: [], hasMore: false, nextCursor: undefined };
    } catch (error) {
        console.error('加载关注动态失败:', error);
        return { items: [], hasMore: false, nextCursor: undefined };
    } finally {
        loading.value.following = false;
    }
};

const searchMoments = async (query: string, filters?: any) => {
    if (!query.trim()) {
        searchResults.value = [];
        return [];
    }
    
    await executeFeature('momentSearch', async () => {
        loading.value.search = true;
        try {
            // 按标签搜索动态
            const result = await momentService.searchMoments({
                type: 'tags',
                tags: [query.trim()]
            });
            searchResults.value = result.items;
        } finally {
            loading.value.search = false;
        }
    });
    
    return searchResults.value;
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

const loadingMore = computed(() => {
    return false; // 暂时不支持加载更多
});

// DataView 配置
const dataViewActions = computed(() => [
    {
        key: 'refresh',
        label: '刷新',
        icon: 'i-heroicons-arrow-path',
        variant: 'outline' as const,
        action: () => { refreshCurrentView(); }
    },
    {
        key: 'create',
        label: '发布动态',
        icon: 'i-heroicons-plus',
        color: 'primary' as const,
        action: openCreateModal
    }
])

const getMomentCardProps = (moment: MomentResponse) => ({
    moment,
    showActions: true,
    class: 'group bg-white/60 dark:bg-slate-800/60 border border-slate-200/50 dark:border-slate-700/50 cursor-pointer hover:bg-white/80 dark:hover:bg-slate-800/80 transition-colors'
})

const getMomentCardEvents = (moment: MomentResponse) => ({
    like: () => toggleLike(moment),
    comment: () => navigateTo(`/moments/${moment.moment_id}#comments`),
    share: () => handleShare(moment),
    'tag-click': (tag: string) => toggleTag(tag),
    deleted: () => refreshCurrentView()
})

const handleSearch = (query: string) => {
    searchQuery.value = query
    if (query.trim()) {
        searchMoments(query)
    } else {
        searchResults.value = []
    }
}

const handleShare = async (moment: MomentResponse) => {
    try {
        if (navigator.share) {
            await navigator.share({
                title: `${moment.nickname}的动态`,
                text: moment.description,
                url: `${window.location.origin}/moments/${moment.moment_id}`
            })
        } else {
            // 复制链接到剪贴板
            await navigator.clipboard.writeText(`${window.location.origin}/moments/${moment.moment_id}`)
            toast.success('链接已复制到剪贴板')
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

const toggleLike = async (moment: MomentResponse) => {
    try {
        // 乐观更新
        const wasLiked = moment.liked_by_me
        moment.liked_by_me = !moment.liked_by_me
        moment.likes_count += moment.liked_by_me ? 1 : -1

        // 调用真实API
        if (moment.liked_by_me) {
            await momentService.likeMoment(moment.moment_id)
        } else {
            await momentService.unlikeMoment(moment.moment_id)
        }
    } catch (error) {
        // 回滚状态
        moment.liked_by_me = !moment.liked_by_me
        moment.likes_count += moment.liked_by_me ? 1 : -1
        toast.error('操作失败')
    }
}

const loadMore = () => {
    // 暂时不支持加载更多
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
