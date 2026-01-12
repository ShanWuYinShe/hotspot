<template>
    <div class="flex flex-col gap-6">
        <!-- Header -->
        <div class="space-y-4">
            <div>
                <h1 class="text-2xl font-bold text-neutral-900 dark:text-white">
                    我的好友
                </h1>
                <p class="text-neutral-500 dark:text-neutral-400 mt-1">
                    互相关注的用户将成为好友
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
                    placeholder="搜索好友昵称..."
                    icon="i-lucide-search"
                    :loading="loading"
                />
            </UCard>

            <!-- 列表 -->
            <div class="min-h-[50vh]">
                <!-- 加载中 -->
                <div
                    v-if="loading && friends.length === 0"
                    class="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4 gap-6"
                >
                    <UserCardSkeleton v-for="i in 8" :key="i" />
                </div>

                <!-- 好友列表 -->
                <div
                    v-else-if="filteredFriends.length > 0"
                    class="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4 gap-6"
                >
                    <UserCard
                        v-for="friend in filteredFriends"
                        :key="friend.user.user_id"
                        :user="{
                            ...friend.user,
                            is_online: friend.is_online,
                        }"
                        @click="handleUserClick(friend.user)"
                        @message="openChat(friend.user.user_id)"
                    >
                        <template #footer>
                            <div
                                class="flex items-center justify-between mt-4 pt-4 border-t border-neutral-100 dark:border-neutral-800"
                            >
                                <div class="text-xs text-neutral-500">
                                    {{ formatTime(friend.followed_back_at) }}
                                    成为好友
                                </div>
                                <UDropdownMenu :items="getMenuItems(friend)">
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
                    icon="i-lucide-users"
                    title="暂无好友"
                    description="去认识一些新朋友吧"
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
                    v-if="hasMore && filteredFriends.length > 0"
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
import type { FriendListItem, UserInfo } from "~/types/api";
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
const friends = ref<FriendListItem[]>([]);
const searchQuery = ref("");
const cursor = ref<string | undefined>(undefined);
const hasMore = ref(true);
const loadMoreTrigger = ref<HTMLElement | null>(null);

// 计算属性
const headerLinks = computed(() => [
    { label: "好友列表", icon: "i-lucide-users", to: "/friends", active: true },
    { label: "关注列表", icon: "i-lucide-user-check", to: "/follows" },
    { label: "粉丝列表", icon: "i-lucide-heart", to: "/followers" },
]);

const filteredFriends = computed(() => {
    if (!searchQuery.value) return friends.value;
    const query = searchQuery.value.toLowerCase();
    return friends.value.filter((f) =>
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
        const res = await usersService.getFriends({ limit: 20 });
        friends.value = res.items;
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
        const res = await usersService.getFriends({
            limit: 20,
            cursor: cursor.value,
        });
        friends.value.push(...res.items);
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

function getMenuItems(friend: FriendListItem) {
    return [
        [
            {
                label: "发消息",
                icon: "i-lucide-message-circle",
                click: () => openChat(friend.user.user_id),
            },
            {
                label: "查看资料",
                icon: "i-lucide-user",
                click: () => handleUserClick(friend.user),
            },
        ],
        [
            {
                label: "修改备注",
                icon: "i-lucide-edit-2",
                click: () => {
                    /* TODO: Implement remark modal */
                },
            },
        ],
        [
            {
                label: "解除好友",
                icon: "i-lucide-user-x",
                color: "error" as const,
                click: () => handleUnfollow(friend.user),
            },
        ],
    ];
}

async function handleUnfollow(user: UserInfo) {
    if (!confirm(`确定要解除与 ${user.nickname} 的好友关系吗？`)) return;
    try {
        await usersService.unfollow({ user_id: user.user_id });
        friends.value = friends.value.filter(
            (f) => f.user.user_id !== user.user_id,
        );
        feedback.success("已解除好友关系");
    } catch (e) {
        feedback.error("操作失败");
    }
}
</script>
