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
                <UInputNumber
                    v-model="radiusMeters"
                    :min="100"
                    :max="50000"
                    :step="100"
                    class="w-44"
                    placeholder="搜索半径(米)"
                    size="lg"
                    :format-options="{
                        style: 'unit',
                        unit: 'meter',
                        unitDisplay: 'short'
                    }"
                />
                <UButton size="md" @click="refresh">应用</UButton>
            </div>

            <!-- 加载状态和表格视图 -->
            <ClientOnly>
                <template v-if="loading">
                    <div class="space-y-4">
                        <UserCardSkeleton v-for="i in 5" :key="i" />
                    </div>
                </template>
                <template v-else>
                    <div class="w-full">
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
                            {{ formatDistance((row as any).distance) }}
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
                                @action="refresh"
                            >
                                <template #illustration>
                                    <EmptyNearby size="lg" />
                                </template>
                            </EmptyState>
                        </div>
                    </template>
                </UTable>
                    </div>
                </template>
            </ClientOnly>

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
import type { UserResponse } from "~/types/api";
import { useUserService } from "~/services/users.service";

const EmptyNearby = defineAsyncComponent(
    () => import("~/components/illustrations/EmptyNearby.vue"),
);

// 公开页面 - 对应 POST /api/v1/public/user/discover
definePageMeta({
    title: "附近的人",
    layout: 'dashboard'
});

const toast = useToast()
const userService = useUserService();
const locationStore = useLocationStore();
const { formatDistance } = useFormatters();

// 本地状态管理
const users = ref<UserResponse[]>([]);
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
    let pos = locationStore.coordinates;
    if (!pos) {
        try {
            await locationStore.getCurrentPosition();
            pos = locationStore.coordinates;
        } catch {
            pos = {
                latitude: defaults.latitude,
                longitude: defaults.longitude,
                accuracy: 0,
                altitude: null,
                altitudeAccuracy: null,
                heading: null,
                speed: null
            };
        }
    }
    return pos;
}

async function refresh() {
    loading.value = true;
    try {
        const pos = await getPositionForSearch();

        const response = await userService.getNearbyUsers({
            latitude: (pos as any).latitude,
            longitude: (pos as any).longitude,
            radius: radiusMeters.value,
            cursor: undefined, // 重新开始
            limit: 20
        });

        users.value = response.items;
        hasMore.value = response.pagination.has_more;
        nextCursor.value = response.pagination.next_cursor;
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

        const response = await userService.getNearbyUsers({
            latitude: (pos as any).latitude,
            longitude: (pos as any).longitude,
            radius: radiusMeters.value,
            cursor: nextCursor.value,
            limit: 20
        });

        users.value.push(...response.items);
        hasMore.value = response.pagination.has_more;
        nextCursor.value = response.pagination.next_cursor;
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

// formatDistance 函数已从统一工具函数导入
</script>
