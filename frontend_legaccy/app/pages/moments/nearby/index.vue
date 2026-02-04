<template>
    <div class="flex flex-col gap-6">
        <!-- Sticky Header & Filter -->
        <div class="sticky top-0 z-20 bg-white/95 dark:bg-neutral-900/95 backdrop-blur-sm border-b border-neutral-200 dark:border-neutral-800 -mx-4 sm:-mx-6 px-4 sm:px-6 py-4 space-y-4">
            <!-- Header -->
            <div class="flex items-center gap-4">
                <UButton
                    variant="ghost"
                    color="neutral"
                    icon="i-lucide-arrow-left"
                    @click="$router.back()"
                />
                <div>
                    <h1 class="text-2xl font-bold text-neutral-900 dark:text-white">附近动态</h1>
                    <p class="text-neutral-500 dark:text-neutral-400 mt-1">发现身边有趣的人和事</p>
                </div>
            </div>

            <!-- 筛选工具栏 -->
            <div class="flex items-center gap-3">
                <UInput
                    v-model="searchQuery"
                    placeholder="搜索动态..."
                    icon="i-lucide-search"
                    class="flex-1"
                    size="sm"
                >
                    <template v-if="searchQuery" #trailing>
                        <UButton
                            color="neutral"
                            variant="link"
                            icon="i-lucide-x"
                            size="xs"
                            @click="searchQuery = ''"
                        />
                    </template>
                </UInput>
                
                <UPopover :popper="{ placement: 'bottom-end' }">
                    <UButton
                        color="neutral"
                        variant="soft"
                        icon="i-lucide-sliders-horizontal"
                        :label="formatDistance(selectedRadius)"
                        size="sm"
                    />
                    <template #content>
                        <div class="p-4 w-64 space-y-4">
                            <div class="flex justify-between items-center">
                                <span class="text-sm font-semibold text-neutral-900 dark:text-white">搜索半径</span>
                                <span class="font-bold text-primary-500">{{ formatDistance(selectedRadius) }}</span>
                            </div>
                            <USlider
                                v-model="selectedRadius"
                                :min="1000"
                                :max="50000"
                                :step="1000"
                                size="sm"
                                color="primary"
                                @update:modelValue="debouncedRefresh"
                            />
                            <div class="flex justify-between text-xs text-neutral-400 dark:text-neutral-500">
                                <span>1km</span>
                                <span>50km</span>
                            </div>
                        </div>
                    </template>
                </UPopover>
            </div>
        </div>

        <div class="space-y-6 pb-12">
            <!-- 列表内容 -->
            <div class="min-h-[60vh]">
                <!-- 加载中 -->
                <div v-if="loading && moments.length === 0" class="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4 gap-6">
                    <div v-for="i in 8" :key="i" class="space-y-3">
                        <div class="flex items-center gap-3">
                            <USkeleton shape="circle" class="size-10" />
                            <div class="space-y-1">
                                <USkeleton class="h-4 w-24" />
                                <USkeleton class="h-3 w-16" />
                            </div>
                        </div>
                        <USkeleton shape="rectangle" class="h-48 w-full rounded-2xl" />
                    </div>
                </div>

                <!-- 结果列表 -->
                <div v-else-if="filteredMoments.length > 0" class="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4 gap-6">
                    <MomentCard
                        v-for="moment in filteredMoments"
                        :key="moment.moment_id"
                        :moment="moment"
                        class="w-full"
                        @click="navigateTo(`/moments/${moment.moment_id}`)"
                        @like="toggleLike(moment)"
                        @comment="navigateTo(`/moments/${moment.moment_id}#comments`)"
                        @share="handleShare(moment)"
                        @tag-click="searchQuery = $event"
                        @deleted="refresh"
                        @updated="refresh"
                    />
                </div>

                <!-- 空状态 -->
                <UEmpty
                    v-else
                    icon="i-lucide-camera-off"
                    title="暂无动态"
                    description="附近还没有人发布动态，来发一条吧！"
                    class="py-20 max-w-md mx-auto"
                >
                    <template #actions>
                        <UButton
                            icon="i-lucide-refresh-cw"
                            variant="soft"
                            @click="refresh"
                        >刷新试试</UButton>
                        <UButton icon="i-lucide-plus" @click="openCreateModal">发布动态</UButton>
                    </template>
                </UEmpty>

                <!-- 加载更多 -->
                <div v-if="hasMore && filteredMoments.length > 0" class="py-8 flex justify-center">
                     <div ref="loadMoreTrigger" class="h-1 w-1" />
                     <UButton
                        :loading="loadingMore"
                        variant="ghost"
                        color="neutral"
                        @click="loadMore"
                    >
                        {{ loadingMore ? '加载中...' : '加载更多' }}
                    </UButton>
                </div>
            </div>
        </div>
    </div>
</template>

<script setup lang="ts">
import { useMomentsService } from "~/services/moments.service";
import type { MomentItem } from "~/types/api";
import type { Moment } from "~/stores/moments.store";
import MomentCard from "~/components/cards/MomentCard.vue";

// 附近的动态是公开页面
definePageMeta({
    title: "附近动态",
    layout: "dashboard",
    requiresAuth: false
});

const { formatDistance } = useFormatters();
const locationStore = useLocationStore();
const geolocation = useGeolocation();
const { requireAuth } = useRequireAuth();
const { share, isSupported: isShareSupported } = useShare();
const { copy, isSupported: isClipboardSupported } = useClipboard();
const momentService = useMomentsService();
const momentsStore = useMomentsStore();
const modal = useModal();
const feedback = useFeedbackStore();

const moments = ref<Moment[]>([]);
const loading = ref(false);
const loadingMore = ref(false);
const hasMore = ref(true);
const nextCursor = ref<string>();
const searchQuery = ref("");
const selectedRadius = ref(5000);

const debouncedRefresh = useDebounceFn(() => refresh(), 500);

const filteredMoments = computed(() => {
    if (!searchQuery.value.trim()) return moments.value;
    const q = searchQuery.value.toLowerCase();
    return moments.value.filter(
        (m) =>
            m.description?.toLowerCase().includes(q) ||
            m.tags?.some((t) => t.toLowerCase().includes(q)),
    );
});

async function openCreateModal() {
    if (!requireAuth("发布动态")) return;
    if (await modal.showCreateMoment()) refresh();
}

async function refresh() {
    loading.value = true;
    try {
        let pos = locationStore.coordinates;
        if (!pos) {
            const coords = await geolocation.getCoordinates();
            locationStore.setLocation(
                coords.latitude,
                coords.longitude,
                "当前位置",
            );
            pos = coords;
        }
        const res = await momentService.searchByLocation({
            reference_point: {
                latitude: pos.latitude,
                longitude: pos.longitude,
            },
            radius: selectedRadius.value,
            limit: 20,
        });
        moments.value = (res.items || []) as Moment[];
        momentsStore.setNearbyMoments(
            (res.items || []) as Moment[],
            res.pagination.has_more,
            res.pagination.next_cursor,
        );
        hasMore.value = res.pagination.has_more;
        nextCursor.value = res.pagination.next_cursor;
    } catch (e) {
        feedback.error("加载失败");
    } finally {
        loading.value = false;
    }
}

async function loadMore() {
    if (loadingMore.value || !hasMore.value) return;
    loadingMore.value = true;
    try {
        const pos = locationStore.coordinates;
        if (!pos) return;
        const res = await momentService.searchByLocation({
            reference_point: {
                latitude: pos.latitude,
                longitude: pos.longitude,
            },
            radius: selectedRadius.value,
            limit: 20,
            cursor: nextCursor.value,
        });
        moments.value.push(...((res.items || []) as Moment[]));
        hasMore.value = res.pagination.has_more;
        nextCursor.value = res.pagination.next_cursor;
    } finally {
        loadingMore.value = false;
    }
}

const { trigger: loadMoreTrigger } = useInfiniteLoadMore(
    loadMore,
    computed(() => loading.value || loadingMore.value),
    computed(() => hasMore.value),
);

async function toggleLike(moment: Moment) {
    if (!requireAuth("点赞")) return;
    try {
        const wasLiked = moment.liked_by_me;
        if (wasLiked) {
            await momentService.unlikeMoment({ moment_id: moment.moment_id });
            moment.liked_by_me = false;
            moment.likes_count--;
        } else {
            await momentService.likeMoment({ moment_id: moment.moment_id });
            moment.liked_by_me = true;
            moment.likes_count++;
        }
    } catch {
        feedback.error("操作失败");
    }
}

async function handleShare(moment: MomentItem) {
    const url = `${location.origin}/moments/${moment.moment_id}`;
    const data = { title: "分享动态", text: moment.description, url };
    if (isShareSupported.value) await share(data);
    else {
        await copy(url);
        feedback.success("链接已复制");
    }
}

onMounted(refresh);
</script>