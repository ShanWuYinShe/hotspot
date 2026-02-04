<template>
    <div class="flex flex-col gap-6">
        <!-- Header -->
        <div class="space-y-4">
            <div>
                <h1 class="text-2xl font-bold text-neutral-900 dark:text-white">
                    关注列表
                </h1>
                <p class="text-neutral-500 dark:text-neutral-400 mt-1">
                    我关注的用户
                </p>
            </div>

            <!-- Tabs -->
            <div
                class="flex gap-2 border-b border-neutral-200 dark:border-neutral-800"
            >
                <NuxtLink
                    v-for="link in headerLinks"
                    :key="link.to"
                    :to="link.to"
                    class="px-4 py-2 text-sm font-medium transition-colors border-b-2 -mb-[2px]"
                    :class="[
                        link.active
                            ? 'text-primary-600 border-primary-600 dark:text-primary-400 dark:border-primary-400'
                            : 'text-neutral-500 border-transparent hover:text-neutral-700 dark:text-neutral-400 dark:hover:text-neutral-200',
                    ]"
                >
                    <div class="flex items-center gap-2">
                        <UIcon :name="link.icon" class="size-4" />
                        {{ link.label }}
                    </div>
                </NuxtLink>
            </div>
        </div>

        <div class="space-y-6 pb-20">
            <!-- 搜索栏 -->
            <UCard variant="subtle">
                <UInput
                    v-model="searchQuery"
                    placeholder="搜索关注..."
                    icon="i-lucide-search"
                    :loading="loading"
                />
            </UCard>

            <!-- 列表 -->
            <div class="min-h-[50vh]">
                <!-- 加载中 -->
                <div
                    v-if="loading && following.length === 0"
                    class="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4 gap-6"
                >
                    <UserCardSkeleton v-for="i in 8" :key="i" />
                </div>

                <!-- 关注列表 -->
                <div
                    v-else-if="filteredFollowing.length > 0"
                    class="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4 gap-6"
                >
                    <UserCard
                        v-for="item in filteredFollowing"
                        :key="item.user.user_id"
                        :user="{
                            ...item.user,
                            relationship: {
                                is_following: true,
                                is_mutual_follow: item.is_mutual,
                            },
                        }"
                        @click="handleUserClick(item.user)"
                        @message="openChat(item.user.user_id)"
                    >
                        <template #footer>
                            <div
                                class="flex items-center justify-between mt-4 pt-4 border-t border-neutral-100 dark:border-neutral-800"
                            >
                                <div class="text-xs text-neutral-500">
                                    {{ formatTime(item.followed_at) }} 关注
                                </div>
                                <UDropdownMenu :items="getMenuItems(item)">
                                    <UButton
                                        variant="ghost"
                                        color="neutral"
                                        size="xs"
                                        icon="i-lucide-more-horizontal"
                                        @click.stop
                                    />
                                </UDropdownMenu>
                            </div>
                        </template>
                    </UserCard>
                </div>

                <!-- 空状态 -->
                <UEmpty
                    v-else
                    icon="i-lucide-user-check"
                    title="暂无关注"
                    description="去关注一些感兴趣的人吧"
                    class="py-20"
                >
                    <template #actions>
                        <UButton to="/users/nearby" icon="i-lucide-map-pin"
                            >探索附近</UButton
                        >
                    </template>
                </UEmpty>

                <!-- 加载更多 -->
                <div
                    v-if="hasMore && filteredFollowing.length > 0"
                    class="py-8 flex justify-center"
                >
                    <div ref="loadMoreTrigger" class="h-1 w-1" />
                    <UButton
                        :loading="loadingMore"
                        variant="ghost"
                        color="neutral"
                        @click="loadMore"
                    >
                        {{ loadingMore ? "加载中..." : "加载更多" }}
                    </UButton>
                </div>
            </div>
        </div>
    </div>
</template>

<script setup lang="ts">
import { useUsersService } from "~/services/users.service";
import type { FollowingItem, UserInfo } from "~/types/api";
import UserCard from "~/components/cards/UserCard.vue";
import UserCardSkeleton from "~/components/skeletons/UserCardSkeleton.vue";

definePageMeta({
    layout: "dashboard",
    requiresAuth: true,
});

const router = useRouter();
const usersService = useUsersService();
const feedback = useFeedbackStore();
const { formatTime } = useFormatters();

// 状态
const loading = ref(true);
const loadingMore = ref(false);
const following = ref<FollowingItem[]>([]);
const searchQuery = ref("");
const cursor = ref<string | undefined>(undefined);
const hasMore = ref(true);
const loadMoreTrigger = ref<HTMLElement | null>(null);

// 计算属性
const headerLinks = computed(() => [
    { label: "好友列表", icon: "i-lucide-users", to: "/friends" },
    {
        label: "关注列表",
        icon: "i-lucide-user-check",
        to: "/follows",
        active: true,
    },
    { label: "粉丝列表", icon: "i-lucide-heart", to: "/followers" },
]);

const filteredFollowing = computed(() => {
    if (!searchQuery.value) return following.value;
    const query = searchQuery.value.toLowerCase();
    return following.value.filter((f) =>
        f.user.nickname.toLowerCase().includes(query),
    );
});

// 初始化
onMounted(() => {
    refresh();
    useIntersectionObserver(loadMoreTrigger, ([entry]) => {
        if (
            entry?.isIntersecting &&
            hasMore.value &&
            !loading.value &&
            !loadingMore.value
        ) {
            loadMore();
        }
    });
});

async function refresh() {
    loading.value = true;
    cursor.value = undefined;
    hasMore.value = true;
    try {
        const res = await usersService.getFollowing({ limit: 20 });
        following.value = res.items;
        cursor.value = res.pagination.next_cursor;
        hasMore.value = res.pagination.has_more;
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
        const res = await usersService.getFollowing({
            limit: 20,
            cursor: cursor.value,
        });
        following.value.push(...res.items);
        cursor.value = res.pagination.next_cursor;
        hasMore.value = res.pagination.has_more;
    } catch (e) {
        // error
    } finally {
        loadingMore.value = false;
    }
}

function handleUserClick(user: UserInfo) {
    router.push(`/users/${user.user_id}`);
}

function openChat(userId: number) {
    router.push(`/chat/users/${userId}`);
}

function getMenuItems(item: FollowingItem) {
    return [
        [
            {
                label: "发消息",
                icon: "i-lucide-message-circle",
                click: () => openChat(item.user.user_id),
            },
            {
                label: "查看资料",
                icon: "i-lucide-user",
                click: () => handleUserClick(item.user),
            },
        ],
        [
            {
                label: "取消关注",
                icon: "i-lucide-user-x",
                color: "error" as const,
                click: () => handleUnfollow(item.user),
            },
        ],
    ];
}

async function handleUnfollow(user: UserInfo) {
    if (!confirm(`确定要取消关注 ${user.nickname} 吗？`)) return;
    try {
        await usersService.unfollow({ user_id: user.user_id });
        following.value = following.value.filter(
            (f) => f.user.user_id !== user.user_id,
        );
        feedback.success("已取消关注");
    } catch (e) {
        feedback.error("操作失败");
    }
}
</script>
