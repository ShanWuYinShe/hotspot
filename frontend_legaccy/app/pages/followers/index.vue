<template>
    <div class="flex flex-col gap-6">
        <!-- Header -->
        <div class="space-y-4">
            <div>
                <h1 class="text-2xl font-bold text-neutral-900 dark:text-white">
                    我的粉丝
                </h1>
                <p class="text-neutral-500 dark:text-neutral-400 mt-1">
                    关注我的用户
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
                    placeholder="搜索粉丝..."
                    icon="i-lucide-search"
                    :loading="loading"
                />
            </UCard>

            <!-- 列表 -->
            <div class="min-h-[50vh]">
                <!-- 加载中 -->
                <div
                    v-if="loading && followers.length === 0"
                    class="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4 gap-6"
                >
                    <UserCardSkeleton v-for="i in 8" :key="i" />
                </div>

                <!-- 粉丝列表 -->
                <div
                    v-else-if="filteredFollowers.length > 0"
                    class="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4 gap-6"
                >
                    <UserCard
                        v-for="item in filteredFollowers"
                        :key="item.user.user_id"
                        :user="{
                            ...item.user,
                            relationship: {
                                is_following: item.is_following_back,
                                is_followed_by: true,
                                is_mutual_follow: item.is_following_back,
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
                                    {{ formatTime(item.followed_at) }} 关注了你
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
                    icon="i-lucide-heart-off"
                    title="暂无粉丝"
                    description="发布更多精彩内容来吸引粉丝吧"
                    class="py-20"
                >
                    <template #actions>
                        <UButton to="/moments" icon="i-lucide-camera"
                            >发布动态</UButton
                        >
                    </template>
                </UEmpty>

                <!-- 加载更多 -->
                <div
                    v-if="hasMore && filteredFollowers.length > 0"
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
import type { FollowerItem, UserInfo } from "~/types/api";
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
const followers = ref<FollowerItem[]>([]);
const searchQuery = ref("");
const cursor = ref<string | undefined>(undefined);
const hasMore = ref(true);
const loadMoreTrigger = ref<HTMLElement | null>(null);

// 计算属性
const headerLinks = computed(() => [
    { label: "好友列表", icon: "i-lucide-users", to: "/friends" },
    { label: "关注列表", icon: "i-lucide-user-check", to: "/follows" },
    {
        label: "粉丝列表",
        icon: "i-lucide-heart",
        to: "/followers",
        active: true,
    },
]);

const filteredFollowers = computed(() => {
    if (!searchQuery.value) return followers.value;
    const query = searchQuery.value.toLowerCase();
    return followers.value.filter((f) =>
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
        const res = await usersService.getFollowers({ limit: 20 });
        followers.value = res.items;
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
        const res = await usersService.getFollowers({
            limit: 20,
            cursor: cursor.value,
        });
        followers.value.push(...res.items);
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

interface MenuItem {
    label: string;
    icon?: string;
    click?: () => void;
    color?: string;
}

function getMenuItems(item: FollowerItem) {
    const items: MenuItem[][] = [
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
    ];

    // 如果未回关，显示回关选项
    if (!item.is_following_back) {
        items[0].push({
            label: "关注",
            icon: "i-lucide-user-plus",
            click: () => handleFollowBack(item),
        });
    } else {
        // 如果已互关，显示取消关注
        items.push([
            {
                label: "取消关注",
                icon: "i-lucide-user-x",
                color: "error" as const,
                click: () => handleUnfollow(item.user),
            },
        ]);
    }

    return items;
}

async function handleFollowBack(item: FollowerItem) {
    try {
        await usersService.follow({ user_id: item.user.user_id });
        item.is_following_back = true;
        feedback.success("已关注");
    } catch (e) {
        feedback.error("操作失败");
    }
}

async function handleUnfollow(user: UserInfo) {
    if (!confirm(`确定要取消关注 ${user.nickname} 吗？`)) return;
    try {
        await usersService.unfollow({ user_id: user.user_id });
        const item = followers.value.find(
            (f) => f.user.user_id === user.user_id,
        );
        if (item) item.is_following_back = false;
        feedback.success("已取消关注");
    } catch (e) {
        feedback.error("操作失败");
    }
}
</script>
