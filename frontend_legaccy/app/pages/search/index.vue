<template>
    <div class="flex flex-col gap-6">
        <!-- Header -->
        <div class="flex flex-col gap-1 shrink-0">
            <h1 class="text-2xl font-bold text-neutral-900 dark:text-white">
                {{ $t("users.search.title") || "搜索用户" }}
            </h1>
            <p class="text-neutral-500 dark:text-neutral-400">
                {{ $t("users.search.description") || "按昵称或手机号查找用户" }}
            </p>
        </div>

        <div class="min-w-0">
            <!-- 搜索栏 -->
            <UCard variant="subtle" class="mb-6">
                <div class="space-y-4">
                    <div class="flex flex-col sm:flex-row gap-3">
                        <UInput
                            v-model="searchQuery"
                            :placeholder="$t('users.search.placeholder')"
                            icon="i-lucide-search"
                            class="flex-1"
                            size="md"
                            :loading="searching"
                            @keyup.enter="performSearch"
                        />
                        <UButton
                            @click="performSearch"
                            :loading="searching"
                            :disabled="!searchQuery.trim()"
                            color="primary"
                            size="md"
                            leading-icon="i-lucide-search"
                            class="w-full sm:w-auto"
                        >
                            {{ $t("app.actions.search") }}
                        </UButton>
                    </div>

                    <!-- 搜索类型选择 -->
                    <div class="flex flex-wrap gap-2 items-center">
                        <UBadge
                            color="neutral"
                            variant="soft"
                            size="xs"
                        >
                            {{ $t("users.search.searchType") }}
                        </UBadge>
                        <UButton
                            v-for="type in searchTypeOptions"
                            :key="type"
                            :variant="searchType === type ? 'soft' : 'ghost'"
                            size="xs"
                            :color="searchType === type ? 'primary' : 'neutral'"
                            @click="searchType = type"
                        >
                            {{ $t(`users.search.type.${type}`) }}
                        </UButton>
                    </div>
                </div>
            </UCard>

            <!-- 搜索结果 -->
            <div v-if="hasSearched">
                <!-- 加载中 -->
                <LoadingState
                    v-if="searching"
                    title="搜索用户中..."
                    description="正在查找匹配的用户"
                />

                <!-- 搜索结果列表 -->
                <div v-else>
                    <UCard variant="subtle" class="overflow-hidden p-0">
                        <div
                            v-if="pagedResults.length > 0"
                            class="divide-y divide-neutral-100 dark:divide-neutral-800"
                        >
                            <div
                                v-for="user in pagedResults"
                                :key="user.user_id"
                                class="flex items-center gap-4 p-4 hover:bg-neutral-50 dark:hover:bg-neutral-800/50 transition-colors cursor-pointer"
                                @click="() => viewUserProfile(user)"
                            >
                                <div
                                    class="flex items-center gap-3 flex-1 min-w-0"
                                >
                                    <UAvatar
                                        :src="user.avatar_url"
                                        :alt="user.nickname"
                                        size="lg"
                                    />
                                    <div class="flex flex-col min-w-0">
                                        <span
                                            class="text-sm font-medium truncate text-neutral-900 dark:text-white"
                                            >{{ user.nickname }}</span
                                        >
                                        <span
                                            v-if="
                                                getUserSearchDescription(user)
                                            "
                                            class="text-xs text-neutral-500 dark:text-neutral-400 truncate"
                                            >{{
                                                getUserSearchDescription(user)
                                            }}</span
                                        >
                                    </div>
                                </div>

                                <div class="flex items-center gap-2 shrink-0">
                                    <!-- 关系状态徽章 -->
                                    <UBadge
                                        v-if="
                                            getRelationshipStatus(user) !==
                                            'none'
                                        "
                                        :label="getRelationshipStatusText(user)"
                                        :color="
                                            getRelationshipStatusColor(user)
                                        "
                                        variant="soft"
                                        size="sm"
                                        class="mr-2"
                                    />

                                    <!-- 操作按钮 -->
                                    <UButton
                                        v-if="
                                            getRelationshipStatus(user) ===
                                            'none'
                                        "
                                        size="sm"
                                        color="primary"
                                        variant="soft"
                                        leading-icon="i-lucide-user-plus"
                                        @click.stop="followUser(user)"
                                        :loading="user.requesting"
                                    >
                                        {{
                                            $t("users.detail.actions.addFriend")
                                        }}
                                    </UButton>

                                    <UButton
                                        v-else-if="
                                            getRelationshipStatus(user) ===
                                            'mutual'
                                        "
                                        size="sm"
                                        variant="ghost"
                                        color="neutral"
                                        @click.stop="
                                            authStore.user &&
                                            user.user_id ===
                                                authStore.user?.user_id
                                                ? navigateTo('/profile')
                                                : navigateTo(
                                                      `/chat/users/${user.user_id}`,
                                                  )
                                        "
                                        icon="i-lucide-messages-square"
                                        :square="true"
                                    >
                                    </UButton>

                                    <UButton
                                        size="sm"
                                        variant="ghost"
                                        color="neutral"
                                        icon="i-lucide-chevron-right"
                                        @click.stop="viewUserProfile(user)"
                                    />
                                </div>
                            </div>
                        </div>

                        <UEmpty
                            v-else
                            icon="i-lucide-user-minus"
                            :title="$t('users.search.empty.title')"
                            :description="$t('users.search.empty.description')"
                            class="py-12"
                        >
                            <template #actions>
                                <UButton variant="soft" @click="clearSearch">{{
                                    $t("users.search.empty.actionText")
                                }}</UButton>
                            </template>
                        </UEmpty>
                    </UCard>

                    <div
                        v-if="pageCount > 1"
                        class="mt-6 flex justify-center"
                    >
                        <UPagination
                            v-model:page="page"
                            :items-per-page="pageSize"
                            :total="searchResults.length"
                            align="center"
                        />
                    </div>
                </div>
            </div>

            <!-- 推荐用户 -->
            <div v-else>
                <UCard>
                    <template #header>
                        <div class="flex items-center justify-between">
                            <span
                                class="text-lg font-semibold text-neutral-900 dark:text-white"
                            >
                                {{ $t("users.search.recommended.title") }}
                            </span>
                            <UButton
                                size="sm"
                                variant="ghost"
                                color="neutral"
                                @click="loadRecommendations"
                                :loading="loadingRecommended"
                                icon="i-lucide-refresh-cw"
                            />
                        </div>
                    </template>

                    <!-- 推荐用户加载中 -->
                    <div v-if="loadingRecommended" class="space-y-4 py-4">
                        <div
                            v-for="i in 3"
                            :key="i"
                            class="flex items-center gap-4"
                        >
                            <UAvatar
                                size="md"
                                icon="i-lucide-user"
                                color="neutral"
                            />
                            <div class="flex-1 space-y-2">
                                <USkeleton class="h-4 w-1/3" />
                                <USkeleton class="h-3 w-1/2" />
                            </div>
                        </div>
                    </div>

                    <!-- 推荐用户列表 -->
                    <div
                        v-else-if="recommendedUsers.length > 0"
                        class="grid grid-cols-1 md:grid-cols-2 gap-4"
                    >
                        <div
                            v-for="user in recommendedUsers"
                            :key="user.user_id"
                            class="flex items-center gap-4 p-4 border border-neutral-200 dark:border-neutral-800 rounded-lg hover:bg-neutral-50 dark:hover:bg-neutral-800/50 transition-colors cursor-pointer"
                            @click="() => viewUserProfile(user)"
                        >
                            <div class="flex items-center gap-3 flex-1 min-w-0">
                                <UAvatar
                                    :src="user.avatar_url"
                                    :alt="user.nickname"
                                    size="md"
                                />
                                <div class="flex flex-col min-w-0">
                                    <span
                                        class="text-sm font-medium truncate text-neutral-900 dark:text-white"
                                        >{{ user.nickname }}</span
                                    >
                                    <span
                                        v-if="user.location?.location_name"
                                        class="text-xs text-neutral-500 dark:text-neutral-400 truncate"
                                    >
                                        {{ user.location.location_name
                                        }}{{
                                            user.location.distance
                                                ? ` · ${formatDistance(user.location.distance)}`
                                                : ""
                                        }}
                                    </span>
                                </div>
                            </div>

                            <UButton
                                size="xs"
                                variant="soft"
                                color="primary"
                                @click.stop="followUser(user)"
                                :loading="user.requesting"
                                icon="i-lucide-user-plus"
                            />
                        </div>
                    </div>

                    <UEmpty
                        v-else
                        icon="i-lucide-users"
                        :title="$t('users.search.recommended.empty.title')"
                        :description="
                            $t('users.search.recommended.empty.description')
                        "
                        class="py-8"
                    >
                        <template #actions>
                            <UButton variant="soft" @click="loadRecommendations"
                                >{{
                                    $t(
                                        "users.search.recommended.empty.actionText",
                                    )
                                }}</UButton
                            >
                        </template>
                    </UEmpty>
                </UCard>
            </div>
        </div>
    </div>
</template>

<script setup lang="ts">
import { useUsersService } from "~/services/users.service";
import type { UserSearchItem } from "~/types/api";
import LoadingState from "~/components/common/LoadingState.vue";

definePageMeta({
    title: "搜索用户",
    description: "按昵称或手机号查找用户",
    layout: "dashboard",
    requiresAuth: false  // 搜索页面允许未登录用户使用
});

// 搜索结果用户类型（扩展 UserSearchItem，添加本地状态）
interface SearchUserItem extends UserSearchItem {
    requesting?: boolean;
}

const userService = useUsersService();
const feedback = useFeedbackStore();
const router = useRouter();
// 使用水合安全的 store 访问
const authStore = useAuthStore();
const { requireAuth } = useRequireAuth();

const { t: tt } = useI18n();
const { formatDistance } = useFormatters();

// 搜索状态
const searchQuery = ref("");
const searchTypeOptions = ["name", "phone", "all"] as const;
const searchType = ref<(typeof searchTypeOptions)[number]>("name");
const searching = ref(false);
const hasSearched = ref(false);
const searchResults = ref<SearchUserItem[]>([]);

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
const recommendedUsers = ref<SearchUserItem[]>([]);
const loadingRecommended = ref(false);

// 搜索功能（全局搜索，不限范围）
async function performSearch() {
    if (!searchQuery.value.trim() || searching.value) return;
    searching.value = true;
    hasSearched.value = true;
    page.value = 1;
    try {
        // 获取位置（仅用于计算距离，不过滤范围）
        const geolocation = useGeolocation();
        const location = await geolocation.getCoordinates();

        const response = await userService.searchByKeyword({
            kw: searchQuery.value,
            reference_point: {
                latitude: location.latitude,
                longitude: location.longitude,
            },
        });
        const me = authStore.user?.user_id;
        searchResults.value = (response.items || [])
            .filter((u: UserSearchItem) => !me || u.user_id !== me)
            .map(
                (u: UserSearchItem): SearchUserItem => ({
                    ...u,
                    requesting: false,
                }),
            );
    } catch (e: unknown) {
        feedback.error(
            tt("users.search.toasts.searchFailed"),
            getErrorMessage(e),
        );
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
        const geolocation = useGeolocation();
        const coords = await geolocation.getCoordinates();
        const response = await userService.searchByLocation({
            reference_point: {
                latitude: coords.latitude,
                longitude: coords.longitude,
            },
            radius: 1000,
            limit: 6,
        });
        recommendedUsers.value = response.items
            .filter(
                (user: UserSearchItem) =>
                    !authStore.user || user.user_id !== authStore.user?.user_id,
            )
            .map(
                (u: UserSearchItem): SearchUserItem => ({
                    ...u,
                    requesting: false,
                }),
            );
    } catch (e: unknown) {
        feedback.error(
            tt("users.search.toasts.recommendLoadFailed"),
            getErrorMessage(e),
        );
    } finally {
        loadingRecommended.value = false;
    }
}

// 关注系统功能（好友 = 互相关注）
async function followUser(user: SearchUserItem) {
    if (!requireAuth("关注用户")) return;
    if (user.requesting) return;
    user.requesting = true;
    try {
        await userService.follow({ user_id: user.user_id });
        // 更新本地状态
        if (!user.relation) {
            user.relation = {
                is_following: true,
                is_mutual_follow: false,
                is_followed_by: false,
                is_blocked: false,
                is_blocking: false,
                has_pending_follow_request: false,
            };
        } else {
            user.relation.is_following = true;
        }

        feedback.success(tt("users.detail.toasts.followed"));
    } catch (e: unknown) {
        feedback.error(
            tt("users.detail.toasts.followFailed"),
            getErrorMessage(e, tt("users.search.toasts.unableToFollow")),
        );
    } finally {
        user.requesting = false;
    }
}

async function unfollowUser(user: SearchUserItem) {
    if (!requireAuth("取消关注")) return;
    if (user.requesting) return;
    user.requesting = true;
    try {
        await userService.unfollow({ user_id: user.user_id });
        // 更新本地状态
        if (user.relation) {
            user.relation.is_following = false;
        }
        feedback.success(tt("users.detail.toasts.unfollowed"));
    } catch (e: unknown) {
        feedback.error(tt("common.actionFailed"), getErrorMessage(e));
    } finally {
        user.requesting = false;
    }
}

// 关系状态管理
function getRelationshipStatus(
    user: SearchUserItem,
): "none" | "following" | "followed_by" | "mutual" {
    if (user.relation?.is_mutual_follow) return "mutual";
    if (user.relation?.is_following) return "following";
    if (user.relation?.is_followed_by) return "followed_by";
    return "none";
}

function getRelationshipStatusText(user: SearchUserItem): string {
    switch (getRelationshipStatus(user)) {
        case "following":
            return tt("users.search.status.following");
        case "followed_by":
            return tt("users.search.status.followsYou");
        case "mutual":
            return tt("profile.stats.friends");
        default:
            return "";
    }
}

type RelationshipBadgeColor = "info" | "warning" | "success" | "neutral";

function getRelationshipStatusColor(
    user: SearchUserItem,
): RelationshipBadgeColor {
    switch (getRelationshipStatus(user)) {
        case "following":
            return "info";
        case "followed_by":
            return "warning";
        case "mutual":
            return "success";
        default:
            return "neutral";
    }
}

// 用户导航
function viewUserProfile(user: SearchUserItem) {
    // 如果是当前用户，直接跳转到个人主页
    if (authStore.user && user.user_id === authStore.user?.user_id) {
        navigateTo("/profile");
    } else {
        navigateTo(`/users/${user.user_id}`);
    }
}

// 页面加载时获取推荐用户
onMounted(async () => {
    try {
        await loadRecommendations();
    } catch (error: unknown) {
        console.error("[UserSearch] Failed to load recommendations:", error);
    }
});

// 辅助函数
const getUserSearchDescription = (user: SearchUserItem): string => {
    const parts: string[] = [];
    if (user.location?.location_name) parts.push(user.location.location_name);
    // if (user.location?.distance) parts.push(formatDistance(user.location.distance))
    return parts.join(" · ");
};
</script>
