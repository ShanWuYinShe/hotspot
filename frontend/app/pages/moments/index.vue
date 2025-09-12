<template>
    <!-- Moments: apply glass and depth (HIG) -->
    <div class="mx-auto max-w-7xl px-4 sm:px-6 lg:px-8 py-8 sm:py-10">
        <!-- 页面标题和操作栏 -->
        <div
            class="flex flex-col gap-4 sm:flex-row sm:justify-between sm:items-center mb-6"
        >
            <h1 class="text-2xl font-bold text-neutral-900 dark:text-neutral-100">动态广场</h1>
            <div class="flex flex-col gap-2 sm:flex-row sm:items-center">
                <div class="order-2 sm:order-1 flex gap-2">
                    <UButton @click="scrollToEditor" variant="ghost" size="sm">
                        <UIcon
                            name="i-heroicons-pencil"
                            class="w-4 h-4 mr-1"
                        />快速发布
                    </UButton>
                    <UButton @click="navigateTo('/moments/create')">
                        <UIcon
                            name="i-heroicons-plus"
                            class="w-4 h-4 mr-2"
                        />发布动态
                    </UButton>
                </div>
                <!-- 使用 UTabs 组件替代手动按钮 -->
                <UTabs
                    :items="tabItems"
                    v-model="currentView"
                    class="order-1 sm:order-2"
                />
            </div>
        </div>

        <!-- 搜索和筛选栏 -->
        <div v-if="currentView === 'search'" class="mb-6 space-y-4">
            <div class="flex gap-4">
                <UInput
                    v-model="searchQuery"
                    placeholder="搜索动态内容或标签..."
                    icon="i-heroicons-magnifying-glass"
                    class="flex-1"
                    @keyup.enter="performSearch"
                />
                <UButton @click="performSearch" :loading="searching"
                    >搜索</UButton
                >
            </div>

            <!-- 标签筛选 -->
            <div class="flex flex-wrap gap-2">
                <span
                    class="text-sm text-neutral-600 dark:text-neutral-400 mr-2"
                    >热门标签:</span
                >
                <UButton
                    v-for="tag in popularTags"
                    :key="tag"
                    variant="ghost"
                    size="xs"
                    :class="{
                        'bg-primary/10 text-primary':
                            selectedTags.includes(tag),
                    }"
                    @click="toggleTag(tag)"
                >
                    #{{ tag }}
                </UButton>
            </div>
        </div>

        <UCard class="mb-6" ref="editorCard">
            <div class="flex gap-3">
                <UAvatar :alt="authStore.user?.nickname" size="md">
                    {{ authStore.user?.nickname?.charAt(0) || "U" }}
                </UAvatar>
                <div class="flex-1">
                    <UTextarea
                        v-model="content"
                        :rows="8"
                        placeholder="分享你的精彩时刻..."
                        class="mb-3 w-5/6"
                    />
                    <div class="flex justify-between items-center">
                        <div class="flex gap-2">
                            <UButton variant="ghost" size="sm" disabled>
                                <UIcon
                                    name="i-heroicons-photo"
                                    class="w-4 h-4 mr-1"
                                />图片
                            </UButton>
                            <UButton variant="ghost" size="sm" disabled>
                                <UIcon
                                    name="i-heroicons-map-pin"
                                    class="w-4 h-4 mr-1"
                                />位置
                            </UButton>
                        </div>
                        <UButton
                            :disabled="!content.trim() || publishing"
                            :loading="publishing"
                            @click="onPublish"
                            >发布</UButton
                        >
                    </div>
                </div>
            </div>
        </UCard>

        <!-- 加载中状态 -->
        <div v-if="loading && displayMoments.length === 0">
            <LoadingState 
              :title="getLoadingTitle()"
              :description="getLoadingDescription()"
              icon="i-heroicons-photo"
            />
        </div>

        <!-- 空状态 -->
        <UCard v-else-if="displayMoments.length === 0">
            <!-- 附近动态但没有位置权限时的特殊状态 -->
            <div
                v-if="currentView === 'nearby' && !coordinates"
                class="text-center py-8"
            >
                <div
                    class="w-20 h-20 mx-auto mb-4 rounded-full bg-orange-100 dark:bg-orange-900 flex items-center justify-center"
                >
                    <UIcon
                        name="i-heroicons-map-pin"
                        class="h-10 w-10 text-orange-600 dark:text-orange-400"
                    />
                </div>
                <h3
                    class="text-title-md font-semibold text-neutral-900 dark:text-white mb-2"
                >
                    获取位置权限
                </h3>
                <p
                    class="text-body-md text-neutral-600 dark:text-neutral-400 mb-6 max-w-sm mx-auto"
                >
                    需要获取您的位置信息才能显示附近的动态内容
                </p>
                <div class="space-y-3">
                    <UButton
                        @click="requestLocationPermission"
                        :loading="locationRequesting"
                        size="lg"
                        class="px-6"
                    >
                        <UIcon
                            name="i-heroicons-map-pin"
                            class="h-4 w-4 mr-2"
                        />
                        获取位置
                    </UButton>
                    <UButton
                        @click="currentView = 'mine'"
                        variant="ghost"
                        size="sm"
                    >
                        查看我的动态
                    </UButton>
                </div>
            </div>

            <!-- 常规空状态 -->
            <EmptyState
                v-else
                :icon="getEmptyStateIcon()"
                :title="getEmptyStateTitle()"
                :description="getEmptyStateDescription()"
                :action-text="
                    currentView === 'search'
                        ? '尝试其他关键词'
                        : '发布第一条动态'
                "
                :action-handler="
                    currentView === 'search' ? clearSearch : scrollToEditor
                "
            >
                <template #illustration>
                    <EmptyMoments size="md" />
                </template>
            </EmptyState>
        </UCard>

        <!-- 列表 -->
        <div v-else class="space-y-6">
            <UCard
                v-for="m in displayMoments"
                :key="m.moment_id"
                class="transition-colors duration-200 cursor-pointer hover:bg-neutral-50 dark:hover:bg-neutral-800"
                @click="goToMomentDetail(m.moment_id)"
            >
                <div class="flex items-center gap-3 mb-3">
                    <UAvatar :alt="m.nickname">{{
                        m.nickname?.charAt(0)
                    }}</UAvatar>
                    <div class="flex-1 min-w-0">
                        <div class="font-semibold truncate">
                            {{ m.nickname }}
                        </div>
                        <div
                            class="text-xs text-neutral-600 dark:text-neutral-400"
                        >
                            {{ formatTime(m.created_at) }}
                            <span
                                v-if="m.location_name"
                                class="ml-2 inline-flex items-center gap-1"
                            >
                                <UIcon
                                    name="i-heroicons-map-pin"
                                    class="w-3 h-3"
                                />{{ m.location_name }}
                            </span>
                        </div>
                    </div>
                    <UDropdownMenu :items="getMomentActions(m)">
                        <UButton
                            icon="i-heroicons-ellipsis-horizontal"
                            variant="ghost"
                            size="sm"
                            @click.stop
                        />
                    </UDropdownMenu>
                </div>
                <div class="mb-2 whitespace-pre-wrap">
                    {{ m.description }}
                </div>
                <div
                    class="flex items-center gap-6 pt-2 border-t border-neutral-200 dark:border-neutral-700"
                >
                    <UButton
                        variant="ghost"
                        size="sm"
                        :class="
                            m.liked_by_me
                                ? 'text-red-600'
                                : 'text-neutral-700 dark:text-neutral-300'
                        "
                        @click="toggleLike(m)"
                    >
                        <UIcon
                            :name="
                                m.liked_by_me
                                    ? 'i-heroicons-heart-solid'
                                    : 'i-heroicons-heart'
                            "
                            class="w-4 h-4 mr-1"
                        />{{ m.likes_count }}
                    </UButton>
                    <UButton variant="ghost" size="sm" @click="showComments"
                        ><UIcon
                            name="i-heroicons-chat-bubble-left-right"
                            class="w-4 h-4 mr-1"
                        />{{ m.comments_count }}</UButton
                    >
                    <UButton variant="ghost" size="sm" @click="shareMoment"
                        ><UIcon
                            name="i-heroicons-share"
                            class="w-4 h-4 mr-1"
                        />分享</UButton
                    >
                </div>
            </UCard>
        </div>

        <!-- 加载更多 -->
        <div
            v-if="hasMore && displayMoments.length > 0"
            class="text-center mt-6"
        >
            <UButton variant="outline" :loading="loadingMore" @click="loadMore"
                >加载更多</UButton
            >
        </div>
    </div>
</template>

<script setup lang="ts">
import { DEFAULTS } from "~/config/defaults";
definePageMeta({ title: "动态广场", requiresAuth: true });

import EmptyMoments from "~/components/illustrations/EmptyMoments.vue";

const authStore = useAuthStore();
const momentsStore = useMomentsStore();
import { useUserService } from '~/services/user.service';

const userService = useUserService();
const toast = useSmartToast();
const { coordinates, getCurrentPosition } = useGeolocation();
const { requestGeolocation } = usePermissions();

const content = ref("");
const publishing = ref(false);
const loading = ref(true);
const loadingMore = ref(false);
const editorCard = ref();
const locationRequesting = ref(false);

// 视图状态
const currentView = ref<"nearby" | "search" | "mine">("nearby");

// UTabs 配置
const tabItems = computed(() => [
    {
        key: "nearby",
        label: "附近",
        icon: "i-heroicons-map-pin",
    },
    {
        key: "search",
        label: "搜索",
        icon: "i-heroicons-magnifying-glass",
    },
    {
        key: "mine",
        label: "我的",
        icon: "i-heroicons-user",
    },
]);
const searchQuery = ref("");
const searching = ref(false);
const selectedTags = ref<string[]>([]);

// 热门标签（可以从后端API获取，这里暂时硬编码）
const popularTags = ref([
    "美食",
    "风景",
    "生活",
    "心情",
    "旅行",
    "摄影",
    "健身",
    "音乐",
    "电影",
    "读书",
]);

// 动态计算属性
const moments = computed(() => {
    switch (currentView.value) {
        case "nearby":
            return momentsStore.nearbyMoments;
        case "search":
            return momentsStore.searchResults;
        case "mine":
            return momentsStore.myMoments;
        default:
            return momentsStore.nearbyMoments;
    }
});

const hasMore = computed(() => {
    switch (currentView.value) {
        case "nearby":
            return momentsStore.pagination.nearbyMoments.hasMore;
        case "search":
            return momentsStore.pagination.searchResults.hasMore;
        case "mine":
            return momentsStore.pagination.myMoments.hasMore;
        default:
            return false;
    }
});

const displayMoments = computed(() => moments.value);

// 监听视图切换
watch(currentView, async (newView, oldView) => {
    if (newView !== oldView) {
        await loadCurrentView();
    }
});

onMounted(async () => {
    try {
        await loadCurrentView();
    } catch (error: any) {
        console.error("[Moments] Failed to load initial data:", error);
        toast.add({
            title: "加载失败",
            description: error?.message || "初始数据加载失败，请刷新页面重试",
            color: "error",
        });
    } finally {
        loading.value = false;
    }
});

// 获取加载标题
function getLoadingTitle() {
    switch (currentView.value) {
        case "nearby":
            return "正在加载附近动态...";
        case "mine":
            return "正在加载我的动态...";
        case "search":
            return "正在搜索动态...";
        default:
            return "正在加载动态...";
    }
}

// 获取加载描述
function getLoadingDescription() {
    switch (currentView.value) {
        case "nearby":
            return "为您获取附近用户的精彩内容";
        case "mine":
            return "加载您发布的所有动态";
        case "search":
            return "正在搜索相关内容";
        default:
            return "为您获取最新的精彩内容";
    }
}

// 根据当前视图加载数据
async function loadCurrentView() {
    loading.value = true;
    try {
        switch (currentView.value) {
            case "nearby":
                await refreshNearby();
                break;
            case "mine":
                await momentsStore.fetchMyMoments();
                break;
            case "search":
                // 搜索页面不自动加载，需要用户手动搜索
                break;
        }
    } catch (error: any) {
        console.error("[MomentsIndex] Failed to load current view:", error);
        toast.add({
            title: "加载失败",
            description: error?.message || "生活瞬间加载失败，请刷新页面重试",
            color: "error",
        });
    } finally {
        loading.value = false;
    }
}

async function refreshNearby(userTriggered = false) {
    type Coords = { latitude: number; longitude: number };
    let pos: Coords | null = null;

    // 安全地处理坐标类型转换
    if (
        coordinates.value &&
        typeof coordinates.value === "object" &&
        "latitude" in coordinates.value &&
        "longitude" in coordinates.value
    ) {
        pos = coordinates.value as Coords;
    }

    // 只有在用户主动触发时才尝试获取位置
    if (!pos && userTriggered) {
        try {
            const granted = await requestGeolocation();
            if (granted) {
                const newPos = await getCurrentPosition();
                if (
                    newPos &&
                    typeof newPos === "object" &&
                    "latitude" in newPos &&
                    "longitude" in newPos
                ) {
                    pos = newPos as Coords;
                }
            }
        } catch (error) {
            console.warn("获取位置失败:", error);
        }
    }

    await momentsStore.fetchNearbyMoments({
        latitude: pos?.latitude ?? DEFAULTS.defaultLatitude,
        longitude: pos?.longitude ?? DEFAULTS.defaultLongitude,
    });
}

async function loadMore() {
    loadingMore.value = true;
    try {
        switch (currentView.value) {
            case "nearby":
                type Coords = { latitude: number; longitude: number };
                let pos: Coords | null = null;
                if (
                    coordinates.value &&
                    typeof coordinates.value === "object" &&
                    "latitude" in coordinates.value &&
                    "longitude" in coordinates.value
                ) {
                    pos = coordinates.value as Coords;
                }
                await momentsStore.fetchNearbyMoments({
                    latitude: pos?.latitude ?? 0,
                    longitude: pos?.longitude ?? 0,
                    loadMore: true,
                });
                break;
            case "search":
                // 继续搜索结果
                if (selectedTags.value.length > 0) {
                    let pos2: Coords | null = null;
                    if (
                        coordinates.value &&
                        typeof coordinates.value === "object" &&
                        "latitude" in coordinates.value &&
                        "longitude" in coordinates.value
                    ) {
                        pos2 = coordinates.value as Coords;
                    }
                    await momentsStore.searchMomentsByTags({
                        tags: selectedTags.value,
                        latitude: pos2?.latitude ?? 0,
                        longitude: pos2?.longitude ?? 0,
                        loadMore: true,
                    });
                }
                break;
            case "mine":
                await momentsStore.fetchMyMoments(true);
                break;
        }
    } finally {
        loadingMore.value = false;
    }
}

async function onPublish() {
    if (!content.value.trim() || publishing.value) return;
    publishing.value = true;
    try {
        type Coords = { latitude: number; longitude: number };
        let pos: Coords | null = null;

        // 安全地处理坐标类型转换
        if (
            coordinates.value &&
            typeof coordinates.value === "object" &&
            "latitude" in coordinates.value &&
            "longitude" in coordinates.value
        ) {
            pos = coordinates.value as Coords;
        }

        if (!pos) {
            try {
                const newPos = await getCurrentPosition();
                if (
                    newPos &&
                    typeof newPos === "object" &&
                    "latitude" in newPos &&
                    "longitude" in newPos
                ) {
                    pos = newPos as Coords;
                }
            } catch {}
        }
        const payload = {
            description: content.value.trim(),
            latitude: pos?.latitude ?? 0,
            longitude: pos?.longitude ?? 0,
            location_name: "",
            tags: [] as string[],
        };
        await momentsStore.createMoment(payload);
        content.value = "";
        toast.add({ title: "发布成功", color: "success" });
    } catch (e: any) {
        toast.add({
            title: "发布失败",
            description: e?.message,
            color: "error",
        });
    } finally {
        publishing.value = false;
    }
}

function toggleLike(m: import("~/types").MomentInfo) {
    const action = m.liked_by_me
        ? momentsStore.unlikeMoment
        : momentsStore.likeMoment;
    action(m.moment_id).catch(() => {
        // 回滚由 store 内部完成（此处简单提示即可）
        toast.add({ title: "操作失败", color: "error" });
    });
}

function showComments() {
    toast.add({
        title: "功能暂未开放",
        description: "评论功能即将上线",
        color: "info",
    });
}
function shareMoment() {
    toast.add({
        title: "功能暂未开放",
        description: "分享功能即将上线",
        color: "info",
    });
}
function scrollToEditor() {
    editorCard.value?.$el?.scrollIntoView({
        behavior: "smooth",
        block: "start",
    });
}

function getMomentActions(m: import("~/types").MomentInfo) {
    const isOwner = m.user_id === authStore.user?.user_id;
    const items: any[] = [];
    if (isOwner) {
        items.push([
            {
                label: "删除",
                icon: "i-heroicons-trash",
                click: () => onDelete(m),
            },
        ]);
    } else {
        items.push([
            {
                label: "举报",
                icon: "i-heroicons-flag",
                click: () => reportMoment(),
            },
        ]);
    }
    return items;
}

async function onDelete(m: import("~/types").MomentInfo) {
    try {
        await momentsStore.deleteMoment(m.moment_id);
        toast.add({ title: "已删除", color: "success" });
    } catch (e: any) {
        toast.add({
            title: "操作失败",
            description: e?.message,
            color: "error",
        });
    }
}

function reportMoment() {
    toast.add({ title: "功能暂未开放", color: "info" });
}

function formatTime(s: string) {
    const t = new Date(s);
    const d = Date.now() - t.getTime();
    const m = Math.floor(d / 60000),
        h = Math.floor(d / 3600000),
        day = Math.floor(d / 86400000);
    if (m < 60) return `${m}分钟前`;
    if (h < 24) return `${h}小时前`;
    return `${day}天前`;
}

// 搜索相关功能
async function performSearch() {
    if (!searchQuery.value.trim() && selectedTags.value.length === 0) {
        toast.add({ title: "请输入搜索关键词或选择标签", color: "warning" });
        return;
    }

    searching.value = true;
    try {
        if (selectedTags.value.length > 0) {
            // 按标签搜索
            type Coords = { latitude: number; longitude: number };
            let pos: Coords | null = null;
            if (
                coordinates.value &&
                typeof coordinates.value === "object" &&
                "latitude" in coordinates.value &&
                "longitude" in coordinates.value
            ) {
                pos = coordinates.value as Coords;
            }
            await momentsStore.searchMomentsByTags({
                tags: selectedTags.value,
                latitude: pos?.latitude ?? 0,
                longitude: pos?.longitude ?? 0,
            });
        } else {
            // 这里可以实现按内容搜索的功能
            // 由于后端API目前只支持按标签搜索，这里暂时显示提示
            toast.add({
                title: "功能提示",
                description: "目前只支持标签搜索，请选择下方的热门标签",
                color: "info",
            });
        }
    } catch (e: any) {
        toast.add({
            title: "搜索失败",
            description: e?.message,
            color: "error",
        });
    } finally {
        searching.value = false;
    }
}

function toggleTag(tag: string) {
    const index = selectedTags.value.indexOf(tag);
    if (index > -1) {
        selectedTags.value.splice(index, 1);
    } else {
        selectedTags.value.push(tag);
    }
}

function clearSearch() {
    searchQuery.value = "";
    selectedTags.value = [];
    momentsStore.clearSearchResults();
}

// 导航功能
function goToMomentDetail(momentId: string) {
    navigateTo(`/moments/${momentId}`);
}

// 位置权限处理
async function requestLocationPermission() {
    locationRequesting.value = true;
    try {
        const granted = await requestGeolocation();
        if (granted) {
            await getCurrentPosition();
            toast.success({
                title: "位置获取成功",
                description: "正在加载附近动态..."
            });
            // 重新加载附近动态，用户主动触发
            await refreshNearby(true);
        } else {
            toast.warning({
                title: "位置权限被拒绝",
                description: "无法获取附近动态，可以查看其他内容",
            });
        }
    } catch (error) {
        console.error("获取位置权限失败:", error);
        toast.error({
            title: "位置获取失败",
            description: "请检查浏览器设置并重试"
        });
    } finally {
        locationRequesting.value = false;
    }
}

// 空状态相关
function getEmptyStateIcon() {
    switch (currentView.value) {
        case "nearby":
            return "i-heroicons-map-pin";
        case "search":
            return "i-heroicons-magnifying-glass";
        case "mine":
            return "i-heroicons-user";
        default:
            return "i-heroicons-photo";
    }
}

function getEmptyStateTitle() {
    switch (currentView.value) {
        case "nearby":
            return "附近暂无动态";
        case "search":
            return "没有找到相关动态";
        case "mine":
            return "您还没有发布动态";
        default:
            return "暂无动态";
    }
}

function getEmptyStateDescription() {
    switch (currentView.value) {
        case "nearby":
            return "成为这个区域第一个分享精彩时刻的人吧";
        case "search":
            return "尝试使用其他关键词或标签搜索";
        case "mine":
            return "分享您的精彩时刻，记录生活的美好";
        default:
            return "成为第一个分享精彩时刻的人吧";
    }
}
</script>
