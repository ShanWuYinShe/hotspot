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
                        @click="openCreateMomentModal"
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
                                v-model="dataLoaders.search.query.value"
                                placeholder="搜索动态内容或标签..."
                                icon="i-heroicons-magnifying-glass"
                                size="lg"
                                class="flex-1"
                            />
                            <UButton
                                @click="dataLoaders.search.performSearch"
                                :loading="dataLoaders.search.searching.value"
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
            <div class="space-y-6">
                <!-- 加载状态 -->
                <div v-if="currentLoading" class="space-y-6">
                    <div v-for="i in 3" :key="i" class="bg-white/60 dark:bg-slate-800/60 rounded-xl p-6 border border-slate-200/50 dark:border-slate-700/50 animate-pulse">
                        <div class="flex items-center gap-4 mb-4">
                            <div class="w-12 h-12 bg-slate-300 dark:bg-slate-600 rounded-full"></div>
                            <div class="space-y-2 flex-1">
                                <div class="h-4 bg-slate-300 dark:bg-slate-600 rounded w-1/4"></div>
                                <div class="h-3 bg-slate-200 dark:bg-slate-700 rounded w-1/3"></div>
                            </div>
                        </div>
                        <div class="space-y-3">
                            <div class="h-4 bg-slate-200 dark:bg-slate-700 rounded"></div>
                            <div class="h-4 bg-slate-200 dark:bg-slate-700 rounded w-3/4"></div>
                        </div>
                    </div>
                </div>

                <!-- 空状态 -->
                <div v-else-if="currentData.length === 0" class="text-center py-16">
                    <UCard class="bg-white/60 dark:bg-slate-800/60 border border-slate-200/50 dark:border-slate-700/50 max-w-md mx-auto">
                        <div class="space-y-6">
                            <div class="w-24 h-24 bg-slate-100 dark:bg-slate-700 rounded-2xl flex items-center justify-center mx-auto">
                                <UIcon name="i-heroicons-photo" class="w-12 h-12 text-slate-500" />
                            </div>
                            <div class="space-y-2">
                                <h3 class="text-xl font-bold text-slate-900 dark:text-white">暂无动态</h3>
                                <p class="text-slate-600 dark:text-slate-400">成为第一个分享生活瞬间的人</p>
                            </div>
                        </div>
                    </UCard>
                </div>

                <!-- 动态列表 -->
                <div v-else class="space-y-6">
                    <UCard
                        v-for="moment in currentData"
                        :key="moment.moment_id"
                        class="group bg-white/60 dark:bg-slate-800/60 border border-slate-200/50 dark:border-slate-700/50 cursor-pointer hover:bg-white/80 dark:hover:bg-slate-800/80 transition-colors"
                        @click="navigateTo(`/moments/${moment.moment_id}`)"
                    >
                        <div class="space-y-4">
                            <!-- 用户信息 -->
                            <div class="flex items-center gap-4">
                                <div class="relative">
                                    <UAvatar :alt="moment.nickname" size="lg">
                                        {{ moment.nickname?.charAt(0) }}
                                    </UAvatar>
                                    <div class="absolute -bottom-1 -right-1 w-4 h-4 bg-green-500 rounded-full border-2 border-white dark:border-slate-800"></div>
                                </div>
                                <div class="flex-1 min-w-0">
                                    <div class="flex items-center gap-2">
                                        <h3 class="font-semibold text-slate-900 dark:text-white truncate">{{ moment.nickname }}</h3>
                                        <UBadge v-if="moment.user_id === authStore.user?.user_id" variant="soft" size="xs" color="primary">
                                            我的
                                        </UBadge>
                                    </div>
                                    <div class="flex items-center gap-2 text-sm text-slate-600 dark:text-slate-400">
                                        <UIcon name="i-heroicons-clock" class="w-4 h-4" />
                                        <span>{{ formatTime(moment.created_at) }}</span>
                                        <span v-if="moment.location_name" class="flex items-center gap-1">
                                            <UIcon name="i-heroicons-map-pin" class="w-4 h-4" />
                                            <span class="truncate">{{ moment.location_name }}</span>
                                        </span>
                                    </div>
                                </div>
                                <UButton
                                    variant="ghost"
                                    size="sm"
                                    icon="i-heroicons-ellipsis-horizontal"
                                    class="opacity-0 group-hover:opacity-100 transition-opacity"
                                    @click.stop
                                />
                            </div>

                            <!-- 动态内容 -->
                            <div class="space-y-4">
                                <p class="text-slate-900 dark:text-slate-100 leading-relaxed text-lg">
                                    {{ moment.description }}
                                </p>

                                <!-- 标签 -->
                                <div v-if="moment.tags?.length" class="flex flex-wrap gap-2">
                                    <UBadge
                                        v-for="tag in moment.tags"
                                        :key="tag"
                                        variant="soft"
                                        size="sm"
                                        color="neutral"
                                        class="cursor-pointer hover:color-primary"
                                        @click.stop="toggleTag(tag)"
                                    >
                                        #{{ tag }}
                                    </UBadge>
                                </div>
                            </div>

                            <!-- 互动信息 -->
                            <div class="flex items-center justify-between pt-4 border-t border-slate-200/50 dark:border-slate-700/50">
                                <div class="flex items-center gap-6">
                                    <UButton
                                        variant="ghost"
                                        size="sm"
                                        @click.stop="toggleLike(moment)"
                                        :color="moment.liked_by_me ? 'error' : 'neutral'"
                                    >
                                        <UIcon
                                            :name="moment.liked_by_me ? 'i-heroicons-heart-solid' : 'i-heroicons-heart'"
                                            class="w-5 h-5 mr-2"
                                        />
                                        {{ moment.likes_count }}
                                    </UButton>

                                    <UButton variant="ghost" size="sm" color="neutral">
                                        <UIcon name="i-heroicons-chat-bubble-left" class="w-5 h-5 mr-2" />
                                        {{ moment.comments_count }}
                                    </UButton>

                                    <UButton variant="ghost" size="sm" color="neutral">
                                        <UIcon name="i-heroicons-share" class="w-5 h-5 mr-2" />
                                        分享
                                    </UButton>
                                </div>

                                <div v-if="moment.distance" class="flex items-center gap-1 text-sm text-slate-500">
                                    <UIcon name="i-heroicons-map-pin" class="w-4 h-4" />
                                    <span>{{ formatDistance(moment.distance || 0) }}</span>
                                </div>
                            </div>
                        </div>
                    </UCard>

                    <!-- 加载更多 -->
                    <div v-if="hasMore" class="text-center py-8">
                        <UButton
                            @click="loadMore"
                            :loading="loadingMore"
                            variant="outline"
                            size="lg"
                        >
                            <UIcon name="i-heroicons-arrow-down" class="w-5 h-5 mr-2" />
                            加载更多
                        </UButton>
                    </div>
                </div>
            </div>
        </div>
    </div>
</template>

<script setup lang="ts">
import type { Moment } from '~/types'

definePageMeta({
    title: '动态广场',
    description: '发现和分享生活中的精彩瞬间'
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

// 创建动态Modal
const CreateMomentModal = defineAsyncComponent(() => import('~/components/CreateMomentModal.vue'))
const createMomentModal = overlay.create(CreateMomentModal)

// 打开创建动态Modal
const openCreateMomentModal = () => {
    createMomentModal.open()
}

// 刷新当前视图数据
const refreshCurrentView = () => {
    if (currentView.value === 'nearby') {
        dataLoaders.nearby.refresh()
    } else if (currentView.value === 'following') {
        dataLoaders.following.refresh()
    }
}

// 处理创建成功
createMomentModal.patch({
    onSuccess: () => {
        // 刷新当前视图的数据
        refreshCurrentView()
        toast.success('动态发布成功')
    }
})

// 数据管理器
const dataLoaders = {
    nearby: useDataLoader(async (cursor?: string) => {
        // 获取附近动态
        const result = await momentService.searchMoments({
            type: 'location',
            latitude: 39.9042, // 默认位置，实际应从位置服务获取
            longitude: 116.4074,
            radius: 5000
        }, cursor, 20)
        return {
            items: result.items,
            hasMore: result.has_more,
            nextCursor: result.next_cursor
        }
    }, { autoLoad: false }),
    following: useDataLoader(async (cursor?: string) => {
        // 关注的用户动态 - 暂时返回空数据，因为好友功能未实现
        return { items: [], hasMore: false, nextCursor: undefined }
    }, { autoLoad: false }),
    search: useSearch<Moment>(async (query: string, filters?: any) => {
        if (!query.trim()) return []
        // 按标签搜索动态
        const result = await momentService.searchMoments({
            type: 'tags',
            tags: [query.trim()]
        })
        return result.items
    })
}


// 计算属性
const currentData = computed(() => {
    if (currentView.value === 'search') {
        return dataLoaders.search.results.value
    }
    const loader = dataLoaders[currentView.value as 'nearby' | 'following'] as any
    return loader?.data?.value || []
})

const currentLoading = computed(() => {
    if (currentView.value === 'search') {
        return dataLoaders.search.searching.value
    }
    const loader = dataLoaders[currentView.value as 'nearby' | 'following'] as any
    return loader?.loading?.value || false
})

const hasMore = computed(() => {
    if (currentView.value === 'search') return false
    const loader = dataLoaders[currentView.value as 'nearby' | 'following'] as any
    return loader?.hasMore?.value || false
})

const loadingMore = computed(() => {
    if (currentView.value === 'search') return false
    const loader = dataLoaders[currentView.value as 'nearby' | 'following'] as any
    return loader?.loadingMore?.value || false
})

// 操作函数
const toggleTag = (tag: string) => {
    const index = selectedTags.value.indexOf(tag)
    if (index > -1) {
        selectedTags.value.splice(index, 1)
    } else {
        selectedTags.value.push(tag)
    }

    // 更新搜索筛选条件
    dataLoaders.search.filters.value = { tags: selectedTags.value }
}

const toggleLike = async (moment: Moment) => {
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
    if (currentView.value !== 'search') {
        const loader = dataLoaders[currentView.value as 'nearby' | 'following'] as any
        loader?.loadMore?.()
    }
}

// 监听视图切换
watch(currentView, () => {
    if (currentView.value === 'search') {
        dataLoaders.search.clear()
    } else {
        const loader = dataLoaders[currentView.value as 'nearby' | 'following'] as any
        loader?.refresh?.()
    }
})

// 页面加载
onMounted(() => {
    if (currentView.value !== 'search') {
        const loader = dataLoaders[currentView.value as 'nearby' | 'following'] as any
        loader?.refresh?.()
    }
})
</script>
