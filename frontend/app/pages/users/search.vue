<template>
    <div class="mx-auto max-w-7xl px-4 sm:px-6 lg:px-8 py-8 sm:py-10">
        <!-- 页面标题 -->
        <div class="mb-8">
            <div class="flex items-center gap-3 mb-2">
                <UButton
                    variant="ghost"
                    icon="i-heroicons-arrow-left"
                    @click="$router.back()"
                />
                <h1 class="text-3xl font-bold text-neutral-900 dark:text-white">
                    {{ $t("users.search.title") }}
                </h1>
            </div>
            <p class="text-neutral-600 dark:text-neutral-300">
                {{ $t("users.search.subtitle") }}
            </p>
        </div>

        <!-- 搜索栏 -->
        <UCard class="mb-6">
            <div class="space-y-4">
                <div class="flex gap-4">
                    <UInput
                        v-model="searchQuery"
                        :placeholder="$t('users.search.placeholder')"
                        icon="i-heroicons-magnifying-glass"
                        class="flex-1"
                        @keyup.enter="performSearch"
                        :loading="searching"
                    />
                    <UButton
                        @click="performSearch"
                        :loading="searching"
                        :disabled="!searchQuery.trim()"
                    >
                        {{ $t("app.actions.search") }}
                    </UButton>
                </div>

                <!-- 搜索类型选择 -->
                <div class="flex gap-2">
                    <span
                        class="text-sm text-neutral-600 dark:text-neutral-400 mr-2"
                        >{{ $t("users.search.searchType") }}</span
                    >
                    <UButton
                        variant="ghost"
                        size="xs"
                        :class="{
                            'bg-primary/10 text-primary': searchType === 'name',
                        }"
                        @click="searchType = 'name'"
                    >
                        {{ $t("users.search.type.name") }}
                    </UButton>
                    <UButton
                        variant="ghost"
                        size="xs"
                        :class="{
                            'bg-primary/10 text-primary':
                                searchType === 'phone',
                        }"
                        @click="searchType = 'phone'"
                    >
                        {{ $t("users.search.type.phone") }}
                    </UButton>
                    <UButton
                        variant="ghost"
                        size="xs"
                        :class="{
                            'bg-primary/10 text-primary': searchType === 'all',
                        }"
                        @click="searchType = 'all'"
                    >
                        {{ $t("users.search.type.all") }}
                    </UButton>
                </div>
            </div>
        </UCard>

        <!-- 搜索结果 -->
        <div v-if="hasSearched">
            <!-- 加载中 -->
            <div v-if="searching" class="space-y-4">
                <USkeleton class="h-20" />
                <USkeleton class="h-20" />
                <USkeleton class="h-20" />
            </div>

            <!-- 搜索结果表格 -->
            <div v-else>
                <UTable :rows="pagedResults" :columns="columns as any">
                    <template #avatar_name-cell="{ row }">
                        <div class="flex items-center gap-3 min-w-[220px]">
                            <UAvatar
                                :alt="(row as any).nickname"
                                size="lg"
                                :src="(row as any).avatar_url"
                            >
                                {{ ((row as any).nickname || "").charAt(0) }}
                            </UAvatar>
                            <div class="min-w-0">
                                <div class="flex items-center gap-2 mb-1">
                                    <h3 class="font-semibold truncate">
                                        {{ (row as any).nickname }}
                                    </h3>
                                    <UBadge
                                        v-if="
                                            getFriendshipStatus(row as any) !==
                                            'none'
                                        "
                                        :color="
                                            getFriendshipStatusColor(row as any)
                                        "
                                        size="sm"
                                    >
                                        {{
                                            getFriendshipStatusText(row as any)
                                        }}
                                    </UBadge>
                                </div>
                                <div
                                    class="space-y-1 text-sm text-neutral-600 dark:text-neutral-400"
                                >
                                    <div
                                        v-if="(row as any).bio"
                                        class="truncate"
                                    >
                                        {{ (row as any).bio }}
                                    </div>
                                    <div class="flex items-center gap-4">
                                        <span
                                            v-if="(row as any).location_name"
                                            class="flex items-center gap-1"
                                        >
                                            <UIcon
                                                name="i-heroicons-map-pin"
                                                class="w-3 h-3"
                                            />
                                            {{ (row as any).location_name }}
                                        </span>
                                        <span
                                            v-if="
                                                (row as any).distance !==
                                                undefined
                                            "
                                            class="flex items-center gap-1"
                                        >
                                            <UIcon
                                                name="i-heroicons-signal"
                                                class="w-3 h-3"
                                            />
                                            {{
                                                formatDistance(
                                                    (row as any).distance,
                                                )
                                            }}
                                        </span>
                                    </div>
                                </div>
                            </div>
                        </div>
                    </template>
                    <template #actions-cell="{ row }">
                        <div class="flex gap-2 justify-end">
                            <UButton
                                v-if="
                                    getFriendshipStatus(row as any) === 'none'
                                "
                                size="sm"
                                @click.stop="sendFriendRequest(row as any)"
                                :loading="(row as any).requesting"
                            >
                                <UIcon
                                    name="i-heroicons-user-plus"
                                    class="w-4 h-4 mr-1"
                                />
                                {{ $t("users.detail.actions.addFriend") }}
                            </UButton>
                            <UButton
                                v-else-if="
                                    getFriendshipStatus(row as any) ===
                                    'pending_sent'
                                "
                                size="sm"
                                variant="ghost"
                                disabled
                            >
                                <UIcon
                                    name="i-heroicons-clock"
                                    class="w-4 h-4 mr-1"
                                />
                                {{ $t("users.search.actions.sent") }}
                            </UButton>
                            <UButton
                                v-else-if="
                                    getFriendshipStatus(row as any) ===
                                    'pending_received'
                                "
                                size="sm"
                                variant="outline"
                                @click.stop="handleFriendRequest(row as any)"
                            >
                                <UIcon
                                    name="i-heroicons-check"
                                    class="w-4 h-4 mr-1"
                                />
                                {{ $t("friends.requests.actions.accept") }}
                            </UButton>
                            <UButton
                                v-else-if="
                                    getFriendshipStatus(row as any) ===
                                    'accepted'
                                "
                                size="sm"
                                variant="ghost"
                                @click.stop="
                                    $router.push(
                                        `/chat/user/${(row as any).user_id}`,
                                    )
                                "
                            >
                                <UIcon
                                    name="i-heroicons-chat-bubble-left-right"
                                    class="w-4 h-4 mr-1"
                                />
                                {{ $t("users.search.actions.chat") }}
                            </UButton>
                            <UButton
                                size="sm"
                                variant="ghost"
                                @click.stop="viewUserProfile(row as any)"
                            >
                                <UIcon
                                    name="i-heroicons-user"
                                    class="w-4 h-4 mr-1"
                                />
                                {{ $t("users.search.actions.view") }}
                            </UButton>
                        </div>
                    </template>
                    <template #empty>
                        <EmptyState
                            icon="i-heroicons-user-minus"
                            :title="$t('users.search.empty.title')"
                            :description="$t('users.search.empty.description')"
                            :action-text="$t('users.search.empty.actionText')"
                            :action-handler="clearSearch"
                        />
                    </template>
                </UTable>
                <div v-if="pageCount > 1" class="mt-4 flex justify-center">
                    <UPagination
                        v-model:page="page"
                        :items-per-page="pageSize"
                        :total="searchResults.length"
                    />
                </div>
            </div>
        </div>

        <!-- 推荐用户 -->
        <div v-else>
            <UCard>
                <template #header>
                    <div class="flex items-center justify-between">
                        <h3 class="text-lg font-semibold">
                            {{ $t("users.search.recommended.title") }}
                        </h3>
                        <UButton
                            size="sm"
                            variant="ghost"
                            @click="loadRecommendations"
                            :loading="loadingRecommended"
                        >
                            <UIcon
                                name="i-heroicons-arrow-path"
                                class="w-4 h-4"
                            />
                        </UButton>
                    </div>
                </template>

                <!-- 推荐用户加载中 -->
                <div v-if="loadingRecommended" class="space-y-4">
                    <USkeleton class="h-16" />
                    <USkeleton class="h-16" />
                    <USkeleton class="h-16" />
                </div>

                <!-- 推荐用户列表 -->
                <div v-else-if="recommendedUsers.length > 0" class="divide-y">
                    <div
                        v-for="user in recommendedUsers"
                        :key="user.user_id"
                        class="py-4 first:pt-0 last:pb-0 flex items-center gap-3 cursor-pointer hover:bg-neutral-50 dark:hover:bg-neutral-900 rounded-lg px-2"
                        @click="viewUserProfile(user)"
                    >
                        <UAvatar :alt="user.nickname">{{
                            user.nickname?.charAt(0)
                        }}</UAvatar>
                        <div class="flex-1 min-w-0">
                            <div class="font-medium truncate">
                                {{ user.nickname }}
                            </div>
                            <div
                                class="text-sm text-neutral-500 flex items-center gap-2"
                            >
                                <span v-if="user.location_name">{{
                                    user.location_name
                                }}</span>
                                <span v-if="user.distance !== undefined"
                                    >· {{ formatDistance(user.distance) }}</span
                                >
                            </div>
                        </div>
                        <UButton
                            size="sm"
                            variant="outline"
                            @click.stop="sendFriendRequest(user)"
                            :loading="user.requesting"
                        >
                            {{ $t("app.actions.apply") }}
                        </UButton>
                    </div>
                </div>

                <!-- 推荐用户为空 -->
                <div v-else class="py-8 text-center text-neutral-500">
                    {{ $t("users.search.recommended.empty") }}
                </div>
            </UCard>
        </div>
    </div>
</template>

<script setup lang="ts">
import type { TableColumn } from "@nuxt/ui";
import { useUserService } from "~/services/user.service";
import type { UserInfo } from "~/types";
import { useApi } from "~/composables/useApi";
import { buildFriendUrl, FRIEND_ENDPOINTS } from "~/config/api";

definePageMeta({
    title: "users.search.title",
    requiresAuth: true,
});

const api = useApi();
const userService = useUserService();
const toast = useSmartToast();
const authStore = useAuthStore();
const { t: tt } = useI18n();

// 搜索状态
const searchQuery = ref("");
const searchType = ref<"name" | "phone" | "all">("name");
const searching = ref(false);
const hasSearched = ref(false);
const searchResults = ref<UserInfo[]>([]);

// 表格列
const columns = computed(() => [
    {
        id: "avatar_name",
        key: "avatar_name",
        label: tt("common.user") as string,
    },
    {
        id: "actions",
        key: "actions",
        label: tt("common.actions") as string,
        class: "w-48 text-right",
    },
]) as any;

// 客户端分页
const page = ref(1);
const pageSize = ref(10);
const pageCount = computed(() =>
    Math.max(1, Math.ceil(searchResults.value.length / pageSize.value)),
);
const pagedResults = computed(() => {
    const start = (page.value - 1) * pageSize.value;
    return searchResults.value.slice(start, start + pageSize.value);
});

// 推荐用户状态
const recommendedUsers = ref<UserInfo[]>([]);
const loadingRecommended = ref(false);

// 搜索功能
async function performSearch() {
    if (!searchQuery.value.trim() || searching.value) return;
    searching.value = true;
    hasSearched.value = true;
    page.value = 1;
    try {
        const response = await userService.searchUsers(searchQuery.value);
        searchResults.value = (response.users || []).map((u: any) => ({
            ...u,
            requesting: false,
        }));
    } catch (e: any) {
        toast.add({
            title: tt("users.search.toasts.searchFailed"),
            description: e?.message,
            color: "error",
        });
        searchResults.value = [];
    } finally {
        searching.value = false;
    }
}

// 清空搜索
function clearSearch() {
    searchQuery.value = "";
    hasSearched.value = false;
    searchResults.value = [];
    page.value = 1;
}

// 推荐用户
async function loadRecommendations() {
    loadingRecommended.value = true;
    try {
        const { coordinates, getCurrentPosition } = useGeolocation();
        const pos = coordinates.value || (await getCurrentPosition());
        const response = await userService.findNearbyUsers({
            latitude: pos.latitude,
            longitude: pos.longitude,
            radius: 1000,
            limit: 5,
        });
        recommendedUsers.value = response.items
            .filter((user: any) => user.user_id !== authStore.user?.user_id)
            .map((u: any) => ({ ...u, requesting: false }));
    } catch (e: any) {
        toast.add({
            title: tt("users.search.toasts.recommendLoadFailed"),
            description: e?.message,
            color: "error",
        });
    } finally {
        loadingRecommended.value = false;
    }
}

// 好友系统功能
async function sendFriendRequest(user: UserInfo) {
    if ((user as any).requesting) return;
    (user as any).requesting = true;
    try {
        await api.post(
            buildFriendUrl(FRIEND_ENDPOINTS.REQUESTS),
            {
                user_id: user.user_id,
                message: tt("users.search.defaultRequestMessage"),
            },
            { auth: true },
        );
        (user as any).friendship_status = "pending_sent";
        toast.add({
            title: tt("users.detail.toasts.requestSent"),
            color: "success",
        });
    } catch (e: any) {
        toast.add({
            title: tt("users.detail.toasts.sendFailed"),
            description: e?.message || tt("users.search.toasts.unableToSend"),
            color: "error",
        });
    } finally {
        (user as any).requesting = false;
    }
}

async function handleFriendRequest(user: UserInfo) {
    try {
        await api.post(
            buildFriendUrl(`${FRIEND_ENDPOINTS.ACCEPT}/${user.user_id}`),
            {},
            { auth: true },
        );
        (user as any).friendship_status = "accepted";
        toast.add({ title: tt("users.detail.toasts.added"), color: "success" });
    } catch (e: any) {
        toast.add({
            title: tt("common.actionFailed"),
            description: e?.message,
            color: "error",
        });
    }
}

// 好友状态管理
function getFriendshipStatus(
    user: UserInfo,
): "none" | "pending_sent" | "pending_received" | "accepted" {
    return (user as any).friendship_status || "none";
}

function getFriendshipStatusText(user: UserInfo): string {
    switch (getFriendshipStatus(user)) {
        case "pending_sent":
            return tt("users.search.status.sent");
        case "pending_received":
            return tt("users.search.status.pending");
        case "accepted":
            return tt("profile.stats.friends");
        default:
            return "";
    }
}

function getFriendshipStatusColor(user: UserInfo): any {
    switch (getFriendshipStatus(user)) {
        case "pending_sent":
            return "info";
        case "pending_received":
            return "warning";
        case "accepted":
            return "success";
        default:
            return "neutral";
    }
}

// 用户导航
function viewUserProfile(user: UserInfo) {
    navigateTo(`/users/${user.user_id}`);
}

// 距离格式化
function formatDistance(distance?: number): string {
    if (typeof distance !== "number") return "";
    if (distance < 1000)
        return tt("distance.meters", { n: Math.round(distance) });
    return tt("distance.kilometers", { n: (distance / 1000).toFixed(1) });
}

// 页面加载时获取推荐用户
onMounted(async () => {
    try {
        await loadRecommendations();
    } catch (error: any) {
        console.error("[UserSearch] Failed to load recommendations:", error);
        toast.add({
            title: tt("users.search.toasts.recommendLoadFailed"),
            description:
                error?.message || tt("users.search.toasts.recommendUnable"),
            color: "error",
        });
    }
});
</script>

<style scoped>
/* 为避免Tailwind问题，使用内联样式或基础CSS */
.line-clamp-2 {
    display: -webkit-box;
    line-clamp: 2;
    -webkit-line-clamp: 2;
    -webkit-box-orient: vertical;
    overflow: hidden;
}
</style>
