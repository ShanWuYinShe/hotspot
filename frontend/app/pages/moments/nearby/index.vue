<template>
    <div class="w-full">
        <div class="flex items-center gap-3 mb-6">
            <!-- 搜索输入 -->
            <UInput
                v-model="searchQuery"
                placeholder="搜索动态内容..."
                icon="i-lucide-search"
                class="flex-1"
                :interactive="true"
            >
                <template #trailing>
                    <UButton
                        v-if="searchQuery !== ''"
                        color="neutral"
                        variant="link"
                        icon="i-lucide-x"
                        :padded="false"
                        @click="searchQuery = ''"
                    />
                </template>
            </UInput>
            
            <!-- 半径筛选 (Popover) -->
            <UPopover width="md">
                <UButton
                    color="neutral"
                    variant="soft"
                    icon="i-lucide-sliders-horizontal"
                    :label="formatDistance(selectedRadius)"
                />
                <template #content>
                    <div class="space-y-4">
                        <div class="flex justify-between items-center">
                            <span class="text-sm font-medium">搜索半径</span>
                            <UBadge variant="soft" :label="formatDistance(selectedRadius)" />
                        </div>
                        <USlider
                            v-model="selectedRadius"
                            :min="1000"
                            :max="50000"
                            :step="1000"
                            @update:model-value="debouncedRefresh"
                        />
                        <div class="flex justify-between text-xs">
                            <span>1km</span>
                            <span>25km</span>
                            <span>50km</span>
                        </div>
                    </div>
                </template>
            </UPopover>
        </div>

        <!-- 内容区域 -->
        <div class="relative min-h-[400px]">
            <!-- 加载状态 -->
            <div v-if="loading && moments.length === 0" class="flex flex-col gap-6">
                <USkeleton v-for="i in 3" :key="i" class="h-56" />
            </div>

            <div v-else-if="filteredMoments.length > 0" class="flex flex-col gap-6">
                <MomentCard
                    v-for="moment in filteredMoments"
                    :key="moment.id"
                    :moment="moment"
                    :show-actions="true"
                    class="w-full"
                    @click="navigateTo(`/moments/${moment.id}`)"
                    @like="toggleLike(moment)"
                    @comment="navigateTo(`/moments/${moment.id}#comments`)"
                    @share="handleShare(moment)"
                    @tag-click="handleTagClick"
                    @deleted="refresh"
                    @updated="refresh"
                />
            </div>

            <!-- 空状态 -->
            <UEmpty
                v-else
                icon="i-lucide-image"
                title="暂无附近动态"
                description="这里还有点冷清，成为第一个分享生活瞬间的人吧！"
                class="py-16"
            >
                <template #actions>
                    <UButton variant="soft" icon="i-lucide-refresh-cw" @click="refresh">重新搜索</UButton>
                    <UButton icon="i-lucide-camera" @click="openCreateModal">发布动态</UButton>
                </template>
            </UEmpty>
            
            <!-- 加载更多按钮 -->
            <div v-if="hasMore && filteredMoments.length > 0" class="flex justify-center mt-12 mb-10">
                <!-- 触底自动加载：IntersectionObserver + 保留按钮作为兜底 -->
                <div ref="loadMoreTrigger" class="h-1 w-1" />
                <USkeleton v-if="loadingMore" class="h-10 w-32" />
                <UButton
                  v-else-if="!canAutoLoadMore"
                  variant="ghost"
                  size="lg"
                  color="neutral"
                  leading-icon="i-lucide-arrow-down"
                  class="px-8"
                  @click="loadMore"
                >
                  加载更多
                </UButton>
            </div>
            
            <!-- 已显示全部提示 -->
            <div v-else-if="moments.length > 0 && !hasMore" class="text-center mt-12 mb-10">
                <UBadge color="neutral" variant="soft" size="xs">已显示全部内容</UBadge>
            </div>
        </div>
    </div>
</template>

<script setup lang="ts">
import { useMomentsService } from '~/services/moments.service';
import type { MomentSearchItem } from '~/types/api'
import type { Moment } from '~/stores/moments.store'
import { useDashboardHeader } from '~/composables/useDashboardHeader'

definePageMeta({
    title: '附近动态',
    description: '发现身边的新鲜事',
    layout: 'dashboard',
    dashboardHeader: true,
    contentWidth: 'full'
})

// 基础状态
const { formatDistance } = useFormatters()
const locationStore = useLocationStore()
const geolocation = useGeolocation()
const { share, isSupported: isShareSupported } = useShare()
const { copy, isSupported: isClipboardSupported } = useClipboard()
const { withErrorHandling } = useErrorHandler()
const dashboardHeader = useDashboardHeader()
const momentService = useMomentsService()
const momentsStore = useMomentsStore()
const modal = useModal()
const feedback = useFeedbackStore()

// 状态
const moments = ref<Moment[]>([])
const loading = ref(false)
const loadingMore = ref(false)
const hasMore = ref(true)
const nextCursor = ref<string>()
const searchQuery = ref('')
const selectedRadius = ref(5000)

const debouncedRefresh = useDebounceFn(() => {
    refresh()
}, 500)

const filteredMoments = computed(() => {
    if (!searchQuery.value.trim()) return moments.value
    const query = searchQuery.value.toLowerCase()
    return moments.value.filter(m => 
        m.description?.toLowerCase().includes(query) || 
        m.tags?.some(t => t.toLowerCase().includes(query))
    )
})

// 操作函数
const openCreateModal = async () => {
  const momentId = await modal.showCreateMoment()
  if (momentId) {
    refresh()
  }
}

const refresh = async () => {
    loading.value = true
    try {
        // 使用 locationStore 的位置，如果没有则获取并更新 store
        let pos = locationStore.coordinates;
        if (!pos) {
            const coords = await geolocation.getCoordinates();
            // 尝试获取位置名称并更新 store
            let locationName = '未知位置';
            try {
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
            limit: 20
        });
        
        moments.value = result.items as Moment[];
        // 同步到 store，以便详情页使用缓存
        momentsStore.setNearbyMoments(result.items as Moment[], result.pagination.has_more, result.pagination.next_cursor);
        
        hasMore.value = result.pagination.has_more;
        nextCursor.value = result.pagination.next_cursor;
    } catch (error) {
        console.error('加载附近动态失败:', error);
        feedback.error('加载附近动态失败')
    } finally {
        loading.value = false;
    }
};

// Dashboard header actions（替代每页包 PageLayout 的 header-actions）
watchEffect(() => {
  dashboardHeader.setActions([
    {
      key: 'refresh',
      label: '刷新',
      icon: 'i-lucide-refresh-cw',
      variant: 'outline',
      loading: loading.value,
      action: () => refresh()
    },
    {
      key: 'create',
      label: '发布动态',
      icon: 'i-lucide-camera',
      variant: 'outline',
      disabled: loading.value,
      action: () => openCreateModal()
    }
  ])
})

const loadMore = async () => {
    if (loadingMore.value || !hasMore.value) return;
    
    loadingMore.value = true;
    try {
        const pos = locationStore.coordinates;
        if (!pos) return;

        const result = await momentService.searchByLocation({
            latitude: pos.latitude,
            longitude: pos.longitude,
            radius: selectedRadius.value,
            limit: 20,
            cursor: nextCursor.value
        });
        
        moments.value.push(...result.items as Moment[]);
        hasMore.value = result.pagination.has_more;
        nextCursor.value = result.pagination.next_cursor;
        
        // result.items.length === 0 时：不提示，避免滚动触底时“频繁打扰”
    } catch (error) {
        console.error('加载更多失败:', error);
        feedback.error('加载更多失败')
    } finally {
        loadingMore.value = false;
    }
};

const isBusy = computed(() => loading.value || loadingMore.value)
const { trigger: loadMoreTrigger, canAutoLoadMore } = useInfiniteLoadMore(
  loadMore,
  isBusy,
  computed(() => hasMore.value),
  { rootMargin: '600px 0px' }
)

const handleTagClick = (tag: string) => {
    searchQuery.value = tag
}

const toggleLike = async (moment: Moment) => {
    try {
        // 乐观更新
        const wasLiked = moment.liked_by_me
        
        if (wasLiked) {
            await momentService.unlikeMoment({ id: moment.id })
            moment.liked_by_me = false
            moment.likes_count--
        } else {
            await momentService.likeMoment({ id: moment.id })
            moment.liked_by_me = true
            moment.likes_count++
        }
    } catch (error) {
        feedback.error('操作失败')
        // 回滚状态
        if (moment.liked_by_me !== undefined) {
             moment.liked_by_me = !moment.liked_by_me
             moment.likes_count += moment.liked_by_me ? 1 : -1
        }
    }
}

const handleShare = async (moment: MomentSearchItem) => {
    const url = `${useRequestURL().origin}/moments/${moment.id}`
    const shareData = {
        title: `${moment.user.nickname}的动态`,
        text: moment.description,
        url: url
    }

    try {
        if (isShareSupported.value) {
            await share(shareData)
        } else if (isClipboardSupported.value) {
            await copy(url)
            feedback.success('链接已复制到剪贴板')
        } else {
             feedback.error('无法分享', '您的浏览器不支持分享或复制功能')
        }
    } catch (error) {
        console.error('分享失败:', error)
    }
}

onMounted(() => {
    refresh();
});
</script>


