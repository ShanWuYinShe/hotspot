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
            <div v-if="isLoading" class="space-y-4">
                <SkeletonLoader type="table-row" :count="5" />
            </div>

            <!-- 表格视图 -->
            <div v-else class="w-full">
                <UTable :rows="pagedUsers" :columns="columns as any">
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

            <!-- 分页 -->
            <div v-if="pageCount > 1" class="mt-4 flex justify-center">
                <UPagination
                    v-model:page="page"
                    :items-per-page="pageSize"
                    :total="users.length"
                />
            </div>
        </UCard>
    </div>
</template>

<script setup lang="ts">
import type { TableColumn } from "@nuxt/ui";
import { defineAsyncComponent } from "vue";
import { DEFAULTS as defaults } from "~/config/defaults";
import type { UserInfo } from "~/types";
const EmptyNearby = defineAsyncComponent(
    () => import("~/components/illustrations/EmptyNearby.vue"),
);

definePageMeta({
    title: "附近的人",
    requiresAuth: true,
});

const toast = useSmartToast();
const { useUserService } = await import("~/services/user.service");
const userService = useUserService();
const { coordinates, getCurrentPosition } = useGeolocation();

// 本地状态管理
const users = ref<UserInfo[]>([]);
const loading = ref(false);
const radiusMeters = ref<number>(defaults.nearbyRadiusMeters);

// 表格列
const columns: TableColumn<any>[] = [
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

// 客户端分页
const page = ref(1);
const pageSize = ref(10);
const pageCount = computed(() =>
    Math.max(1, Math.ceil(users.value.length / pageSize.value)),
);
const pagedUsers = computed(() => {
    const start = (page.value - 1) * pageSize.value;
    return users.value.slice(start, start + pageSize.value);
});

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

async function refresh() {
    loading.value = true;
    try {
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

        const response = await userService.findNearbyUsers({
            latitude: (pos as any).latitude,
            longitude: (pos as any).longitude,
            radius_meters: radiusMeters.value,
        });

        users.value = response.items;
        page.value = 1;
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

function openChat(targetUserId: string) {
    navigateTo(`/chat/user/${targetUserId}`);
}

function formatDistance(distance?: number): string {
    if (typeof distance !== "number") return "";
    if (distance < 1000) return `${Math.round(distance)}米`;
    return `${(distance / 1000).toFixed(1)}公里`;
}
</script>
