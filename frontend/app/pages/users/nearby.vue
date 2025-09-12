<template>
    <div class="mx-auto max-w-7xl px-4 sm:px-6 lg:px-8 py-8 sm:py-10">
        <UCard class="mb-6">
            <template #header>
                <div class="flex items-center justify-between">
                    <h1 class="text-2xl font-bold">附近的人</h1>
                    <div class="flex gap-2">
                        <UButton size="sm" variant="outline" @click="refresh"
                            >刷新</UButton
                        >
                    </div>
                </div>
            </template>

            <div class="mb-4 flex gap-3 items-center">
                <UInput
                    v-model.number="radiusMeters"
                    type="number"
                    min="100"
                    step="100"
                    class="w-44"
                    placeholder="搜索半径(米)"
                    size="lg"
                />
                <UButton size="md" @click="refresh">应用</UButton>
            </div>

            <!-- 加载状态 -->
            <div v-if="loading" class="space-y-4">
                <SkeletonLoader type="table-row" :count="5" />
            </div>

            <!-- 表格视图 -->
            <div v-else class="w-full">
                <UTable :rows="pagedUsers" :columns="columns">
                    <template #avatar_name-cell="{ row }">
                        <div class="flex items-center gap-3 min-w-0">
                            <UAvatar :alt="(row as any).nickname">
                                {{ ((row as any).nickname || "友").charAt(0) }}
                            </UAvatar>
                            <div class="min-w-0">
                                <div class="font-semibold truncate">
                                    {{ (row as any).nickname }}
                                </div>
                                <div class="text-xs text-neutral-500 truncate">
                                    {{
                                        (row as any).location_name ||
                                        defaults.locationName
                                    }}
                                </div>
                            </div>
                        </div>
                    </template>
                    <template #distance-cell="{ row }">
                        <span
                            class="text-sm text-neutral-600 dark:text-neutral-400 flex items-center gap-1"
                        >
                            <UIcon name="i-heroicons-signal" class="w-4 h-4" />
                            {{ formatDistanceWrapper((row as any).distance) }}
                        </span>
                    </template>
                    <template #location-cell="{ row }">
                        <span
                            class="text-sm text-neutral-600 dark:text-neutral-400 flex items-center gap-1"
                        >
                            <UIcon name="i-heroicons-map-pin" class="w-4 h-4" />
                            {{
                                (row as any).location_name ||
                                defaults.locationName
                            }}
                        </span>
                    </template>
                    <template #actions-cell="{ row }">
                        <div class="flex items-center justify-end gap-2">
                            <UButton
                                size="sm"
                                variant="solid"
                                @click="openChat(String((row as any).user_id))"
                                >发消息</UButton
                            >
                        </div>
                    </template>
                    <template #empty>
                        <div class="py-10">
                            <EmptyState
                                icon="i-heroicons-user-group"
                                title="附近暂无用户"
                                description="可以尝试扩大搜索半径或稍后再试"
                                action-text="刷新"
                                :action-handler="refresh"
                            >
                                <template #illustration>
                                    <EmptyNearby size="lg" />
                                </template>
                            </EmptyState>
                        </div>
                    </template>
                </UTable>
            </div>

            <!-- 加载更多 -->
            <div v-if="hasMore" class="mt-6 flex justify-center">
                <UButton 
                    :loading="loadingMore"
                    :disabled="loadingMore"
                    variant="outline"
                    size="lg"
                    @click="loadMore"
                >
                    {{ loadingMore ? '加载中...' : '加载更多' }}
                </UButton>
            </div>
            
            <!-- 已加载完毕提示 -->
            <div v-else-if="users.length > 0" class="mt-6 text-center text-sm text-neutral-500">
                已显示全部 {{ users.length }} 位附近用户
            </div>
        </UCard>
    </div>
</template>

<script setup lang="ts">
// TableColumn 类型已在组件内部定义
import { defineAsyncComponent } from "vue";
import { DEFAULTS as defaults } from "~/config/defaults";
import type { UserInfo } from "~/types";
import { useUserService } from '~/services/user.service';
import { formatDistance } from '~/utils/formatters';

const EmptyNearby = defineAsyncComponent(
    () => import("~/components/illustrations/EmptyNearby.vue"),
);

definePageMeta({
    title: "附近的人",
    requiresAuth: true,
});

const toast = useSmartToast();
const userService = useUserService();
const { coordinates, getCurrentPosition } = useGeolocation();

// 本地状态管理
const users = ref<UserInfo[]>([]);
const loading = ref(false);
const loadingMore = ref(false);
const hasMore = ref(true);
const nextCursor = ref<string | undefined>(undefined);
const radiusMeters = ref<number>(defaults.nearbyRadiusMeters);

// 表格列定义
interface TableColumn {
    id: string;
    key: string;
    label: string;
    class?: string;
}

// 表格列
const columns: TableColumn[] = [
    { id: "avatar_name", key: "avatar_name", label: "用户" },
    {
        id: "distance",
        key: "distance",
        label: "距离",
        class: "hidden sm:table-cell w-24",
    },
    {
        id: "location",
        key: "location",
        label: "位置",
        class: "hidden md:table-cell w-56",
    },
    { id: "actions", key: "actions", label: "操作", class: "w-24 text-right" },
] as any;

// 游标分页 - 显示所有已加载的用户
const pagedUsers = computed(() => users.value);

onMounted(async () => {
    try {
        await refresh();
    } catch (error: any) {
        console.error("[NearbyUsers] Failed to load initial data:", error);
        toast.add({
            title: "加载失败",
            description: error?.message || "附近用户加载失败，请刷新页面重试",
            color: "error",
        });
    }
});

// 获取当前位置的通用函数
async function getPositionForSearch() {
    let pos = coordinates.value;
    if (!pos) {
        try {
            pos = await getCurrentPosition();
        } catch {
            pos = {
                latitude: defaults.latitude,
                longitude: defaults.longitude,
            } as any;
        }
    }
    return pos;
}

async function refresh() {
    loading.value = true;
    try {
        const pos = await getPositionForSearch();

        const response = await userService.findNearbyUsers({
            latitude: (pos as any).latitude,
            longitude: (pos as any).longitude,
            radius: radiusMeters.value,
            cursor: undefined, // 重新开始
            limit: 20
        });

        users.value = response.items;
        hasMore.value = response.has_more;
        nextCursor.value = response.next_cursor;
    } catch (e: any) {
        const msg = e?.message || "无法获取附近的人";
        const detail = e?.details?.message || "";
        toast.add({
            title: "加载失败",
            description: detail || msg,
            color: "error",
        });
    } finally {
        loading.value = false;
    }
}

async function loadMore() {
    if (!hasMore.value || loadingMore.value) return;
    
    loadingMore.value = true;
    try {
        const pos = await getPositionForSearch();

        const response = await userService.findNearbyUsers({
            latitude: (pos as any).latitude,
            longitude: (pos as any).longitude,
            radius: radiusMeters.value,
            cursor: nextCursor.value,
            limit: 20
        });

        users.value.push(...response.items);
        hasMore.value = response.has_more;
        nextCursor.value = response.next_cursor;
    } catch (e: any) {
        const msg = e?.message || "无法加载更多用户";
        toast.add({
            title: "加载失败",
            description: msg,
            color: "error",
        });
    } finally {
        loadingMore.value = false;
    }
}

function openChat(targetUserId: string) {
    navigateTo(`/chat/users/${targetUserId}`);
}

// 距离格式化已从统一工具函数导入
function formatDistanceWrapper(distance?: number): string {
    if (typeof distance !== "number") return "";
    return formatDistance(distance);
}
</script>
