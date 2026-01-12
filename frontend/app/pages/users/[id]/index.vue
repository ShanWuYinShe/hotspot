<template>
    <div class="flex flex-col gap-6 pb-20">
        <!-- Header -->
        <div class="flex items-center gap-4 shrink-0">
            <UButton
                icon="i-lucide-arrow-left"
                variant="ghost"
                color="neutral"
                to="/users/nearby"
            />
            <h1 class="text-2xl font-bold text-neutral-900 dark:text-white">
                用户资料
            </h1>
        </div>

        <div class="max-w-4xl mx-auto w-full space-y-8">
            <!-- 加载中 -->
            <div v-if="loading" class="space-y-6">
                <!-- 用户信息卡片骨架 -->
                <UCard variant="solid" class="h-auto">
                    <div class="flex flex-col md:flex-row gap-8 items-center md:items-start">
                        <USkeleton class="size-24 rounded-full shrink-0" />
                        <div class="flex-1 w-full space-y-4">
                            <div class="space-y-2">
                                <USkeleton class="h-8 w-48 mx-auto md:mx-0" />
                                <USkeleton class="h-4 w-64 mx-auto md:mx-0" />
                            </div>
                            <div class="flex gap-2 justify-center md:justify-start">
                                <USkeleton class="h-6 w-20 rounded-full" />
                                <USkeleton class="h-6 w-24 rounded-full" />
                            </div>
                            <div class="flex gap-8 justify-center md:justify-start pt-2">
                                <div v-for="i in 3" :key="i" class="text-center space-y-1">
                                    <USkeleton class="h-6 w-8 mx-auto" />
                                    <USkeleton class="h-3 w-8 mx-auto" />
                                </div>
                            </div>
                        </div>
                    </div>
                </UCard>

                <div class="grid grid-cols-1 md:grid-cols-3 gap-6">
                    <!-- 左侧骨架 -->
                    <UCard variant="subtle" class="h-64">
                        <div class="space-y-4">
                            <USkeleton class="h-6 w-16" />
                            <div class="flex flex-wrap gap-2">
                                <USkeleton v-for="i in 4" :key="i" class="h-6 w-16 rounded-full" />
                            </div>
                        </div>
                    </UCard>
                    
                    <!-- 右侧动态列表骨架 -->
                    <div class="md:col-span-2 space-y-4">
                        <div class="flex justify-between">
                            <USkeleton class="h-6 w-24" />
                        </div>
                        <USkeleton v-for="i in 2" :key="i" class="h-48 w-full rounded-2xl" />
                    </div>
                </div>
            </div>

            <template v-else-if="user">
                <!-- 用户信息卡片 -->
                <UCard variant="solid">
                    <div
                        class="flex flex-col md:flex-row gap-8 items-center md:items-start"
                    >
                        <!-- 头像 -->
                        <div class="relative group shrink-0">
                            <UAvatar
                                :src="user.avatar_url"
                                :alt="user.nickname"
                                size="3xl"
                            />
                        </div>

                        <div class="flex-1 text-center md:text-left space-y-4">
                            <div>
                                <div
                                    class="flex flex-col md:flex-row items-center md:items-baseline gap-2 justify-center md:justify-start"
                                >
                                    <h1
                                        class="text-2xl font-bold text-neutral-900 dark:text-white"
                                    >
                                        {{ user.nickname }}
                                    </h1>
                                    <div class="flex gap-1">
                                        <UBadge
                                            v-if="user.user_type !== UserType.Normal"
                                            :color="
                                                user.user_type === UserType.Admin
                                                    ? 'error'
                                                    : 'primary'
                                            "
                                            variant="soft"
                                            size="xs"
                                        >
                                            {{
                                                user.user_type === UserType.Admin
                                                    ? "管理员"
                                                    : "官方"
                                            }}
                                        </UBadge>
                                        <UBadge
                                            v-if="
                                                relationship?.is_mutual_follow
                                            "
                                            color="primary"
                                            variant="soft"
                                            size="xs"
                                            >好友</UBadge
                                        >
                                        <UBadge
                                            v-else-if="
                                                relationship?.is_following
                                            "
                                            color="neutral"
                                            variant="soft"
                                            size="xs"
                                            >已关注</UBadge
                                        >
                                        <UBadge
                                            v-else-if="
                                                relationship?.is_followed_by
                                            "
                                            color="neutral"
                                            variant="soft"
                                            size="xs"
                                            >关注了你</UBadge
                                        >
                                    </div>
                                </div>
                                <p
                                    class="text-neutral-500 dark:text-neutral-400 mt-2 max-w-lg mx-auto md:mx-0"
                                >
                                    {{ user.bio || "暂无简介" }}
                                </p>
                            </div>

                            <div
                                class="flex flex-wrap justify-center md:justify-start gap-2"
                            >
                                <UBadge
                                    v-if="user.location?.location_name"
                                    color="neutral"
                                    variant="soft"
                                    icon="i-lucide-map-pin"
                                >
                                    {{ user.location.location_name }}
                                    <template v-if="user.location.distance">
                                        ·
                                        {{
                                            formatDistance(user.location.distance)
                                        }}</template
                                    >
                                </UBadge>
                                <UBadge
                                    v-if="user.last_online"
                                    color="neutral"
                                    variant="soft"
                                    icon="i-lucide-clock"
                                >
                                    最后在线: {{ formatTime(user.last_online) }}
                                </UBadge>
                            </div>

                            <div
                                v-if="stats"
                                class="flex gap-6 justify-center md:justify-start pt-2"
                            >
                                <div class="text-center">
                                    <div
                                        class="font-bold text-xl text-neutral-900 dark:text-white"
                                    >
                                        {{ stats.moments_count }}
                                    </div>
                                    <div
                                        class="text-xs text-neutral-500 dark:text-neutral-400"
                                    >
                                        动态
                                    </div>
                                </div>
                                <div class="text-center">
                                    <div
                                        class="font-bold text-xl text-neutral-900 dark:text-white"
                                    >
                                        {{ stats.friends_count }}
                                    </div>
                                    <div
                                        class="text-xs text-neutral-500 dark:text-neutral-400"
                                    >
                                        好友
                                    </div>
                                </div>
                                <div class="text-center">
                                    <div
                                        class="font-bold text-xl text-neutral-900 dark:text-white"
                                    >
                                        {{ stats.followers_count }}
                                    </div>
                                    <div
                                        class="text-xs text-neutral-500 dark:text-neutral-400"
                                    >
                                        粉丝
                                    </div>
                                </div>
                            </div>
                        </div>

                        <!-- 操作按钮 (桌面端) -->
                        <div class="hidden md:flex flex-col gap-2 shrink-0">
                            <UButton
                                v-if="!relationship?.is_mutual_follow"
                                :variant="isFollowing ? 'outline' : 'solid'"
                                :color="isFollowing ? 'neutral' : 'primary'"
                                :icon="isFollowing ? 'i-lucide-user-check' : 'i-lucide-user-plus'"
                                :loading="followLoading"
                                @click="toggleFollow"
                            >
                                {{ isFollowing ? "取消关注" : "关注" }}
                            </UButton>
                            <UButton
                                variant="soft"
                                color="primary"
                                icon="i-lucide-message-circle"
                                @click="enterChat"
                            >
                                发消息
                            </UButton>
                            <UDropdownMenu :items="moreActions">
                                <UButton
                                    variant="ghost"
                                    color="neutral"
                                    icon="i-lucide-more-horizontal"
                                />
                            </UDropdownMenu>
                        </div>
                    </div>

                    <!-- 操作按钮 (移动端) -->
                    <template #footer>
                        <div class="flex md:hidden gap-3">
                            <UButton
                                class="flex-1"
                                :variant="isFollowing ? 'outline' : 'solid'"
                                :color="isFollowing ? 'neutral' : 'primary'"
                                :icon="isFollowing ? 'i-lucide-user-check' : 'i-lucide-user-plus'"
                                :loading="followLoading"
                                @click="toggleFollow"
                            >
                                {{ isFollowing ? "已关注" : "关注" }}
                            </UButton>
                            <UButton
                                class="flex-1"
                                variant="soft"
                                color="primary"
                                icon="i-lucide-message-circle"
                                @click="enterChat"
                            >
                                发消息
                            </UButton>
                            <UDropdownMenu :items="moreActions">
                                <UButton
                                    variant="ghost"
                                    color="neutral"
                                    icon="i-lucide-more-horizontal"
                                />
                            </UDropdownMenu>
                        </div>
                    </template>
                </UCard>

                <!-- 下方内容区 -->
                <div class="grid grid-cols-1 md:grid-cols-3 gap-6">
                    <!-- 左侧：标签与群组 -->
                    <div class="space-y-6">
                        <UCard variant="subtle">
                            <template #header
                                ><h3
                                    class="text-lg font-semibold text-neutral-900 dark:text-white"
                                >
                                    标签
                                </h3></template
                            >
                            <div class="flex flex-wrap gap-2">
                                <UBadge
                                    v-for="tag in user.tags"
                                    :key="tag"
                                    variant="outline"
                                    color="neutral"
                                >
                                    {{ tag }}
                                </UBadge>
                                <span
                                    v-if="!user.tags?.length"
                                    class="text-sm text-neutral-500 dark:text-neutral-400"
                                    >暂无标签</span
                                >
                            </div>
                        </UCard>
                    </div>

                    <!-- 右侧：动态列表 -->
                    <div class="md:col-span-2 space-y-4">
                        <div class="flex items-center justify-between">
                            <h3
                                class="text-lg font-semibold text-neutral-900 dark:text-white"
                            >
                                最近动态
                            </h3>
                        </div>

                        <div v-if="loadingMoments" class="space-y-4">
                            <USkeleton
                                v-for="i in 2"
                                :key="i"
                                class="h-48 w-full rounded-2xl"
                            />
                        </div>

                        <div v-else-if="moments.length > 0" class="space-y-4">
                            <MomentCard
                                v-for="moment in moments"
                                :key="moment.moment_id"
                                :moment="moment"
                                @click="navigateTo(`/moments/${moment.moment_id}`)"
                                @like="toggleLikeMoment(moment)"
                            />
                        </div>

                        <UEmpty
                            v-else
                            icon="i-lucide-camera-off"
                            title="暂无动态"
                            description="该用户最近没有发布动态"
                            variant="naked"
                            class="py-12"
                        />
                    </div>
                </div>
            </template>

            <UEmpty
                v-else-if="errorSnapshot && (isUserNotFoundCode(errorSnapshot.code) || isNotFoundCode(errorSnapshot.code) || errorSnapshot.status === 404)"
                icon="i-lucide-user-x"
                title="用户不存在"
                description="该用户可能已被注销"
                class="py-20"
            >
                <template #actions>
                    <UButton to="/users/nearby">返回列表</UButton>
                </template>
            </UEmpty>

            <UEmpty
                v-else-if="errorSnapshot"
                icon="i-lucide-server-crash"
                title="加载失败"
                :description="guessUserMessageFromSnapshot(errorSnapshot)"
                class="py-20"
            >
                <template #actions>
                    <UButton @click="fetchData" color="primary" variant="soft">重试</UButton>
                    <UButton to="/users/nearby" color="neutral" variant="ghost" class="ml-2">返回列表</UButton>
                </template>
            </UEmpty>

            <UEmpty
                v-else
                icon="i-lucide-user-x"
                title="用户不存在"
                description="该用户可能已被注销"
                class="py-20"
            >
                <template #actions>
                    <UButton to="/users/nearby">返回列表</UButton>
                </template>
            </UEmpty>
        </div>
    </div>
</template>

<script setup lang="ts">
import { useUsersService } from "~/services/users.service";
import { useMomentsService } from "~/services/moments.service";
import type {
    UserItem,
    UserRelation,
    UserStatsData,
    MomentItem,
} from "~/types/api";
import MomentCard from "~/components/cards/MomentCard.vue";
import { useRequireAuth } from "~/composables/useRequireAuth";
import { UserType } from "~/config/constants";
import { extractApiErrorSnapshot, guessUserMessageFromSnapshot, isUserNotFoundCode, isNotFoundCode } from "~/utils/apiError";

definePageMeta({
    layout: "dashboard",
    requiresAuth: false  // 用户详情页允许未登录用户浏览
});

const route = useRoute();
const router = useRouter();
const usersService = useUsersService();
const momentsService = useMomentsService();
const geolocation = useGeolocation();
const feedback = useFeedbackStore();
const authStore = useAuthStore();
const { formatDistance, formatTime } = useFormatters();
const { requireAuth } = useRequireAuth();

const userId = parseInt(route.params.id as string);
const loading = ref(true);
const user = ref<UserItem | null>(null);
const relationship = ref<UserRelation | null>(null);
const stats = ref<UserStatsData | null>(null);
const errorSnapshot = ref<ReturnType<typeof extractApiErrorSnapshot> | null>(null);

const loadingMoments = ref(true);
const moments = ref<MomentItem[]>([]);

const isFollowing = computed({
    get: () => relationship.value?.is_following ?? false,
    set: (val) => {
        if (relationship.value) relationship.value.is_following = val;
    },
});

const followLoading = ref(false);
async function toggleFollow() {
    if (!requireAuth(isFollowing.value ? "取消关注" : "关注用户")) return;

    followLoading.value = true;
    try {
        if (isFollowing.value) {
            await usersService.unfollow({ user_id: userId });
            feedback.success("已取消关注");
        } else {
            await usersService.follow({ user_id: userId });
            feedback.success("关注成功");
        }
        // 刷新关系状态
        const rel = await usersService.getRelationship({ user_id: userId });
        relationship.value = rel;
    } catch (e) {
        feedback.error("操作失败");
    } finally {
        followLoading.value = false;
    }
}

const moreActions = computed(() => [
    [
        {
            label: relationship.value?.is_blocked ? "解除拉黑" : "拉黑用户",
            icon: "i-lucide-ban",
            color: relationship.value?.is_blocked
                ? "neutral"
                : ("error" as const),
            click: toggleBlock,
        },
        {
            label: "举报用户",
            icon: "i-lucide-flag",
            click: () => {
                /* open report modal */
            },
        },
    ],
]);

onMounted(async () => {
    if (authStore.user?.user_id === userId) {
        return router.replace("/profile");
    }
    await fetchData();
});

async function fetchData() {
    loading.value = true;
    errorSnapshot.value = null;
    try {
        const coords = await geolocation.getCoordinates();
        const [userData, relData, statsData] = await Promise.all([
            usersService.openTheDoor({
                user_id: userId,
                reference_point: {
                    latitude: coords.latitude,
                    longitude: coords.longitude,
                },
            }),
            usersService.getRelationship({ user_id: userId }),
            usersService.getStats({ user_id: userId }),
        ]);

        user.value = userData;
        relationship.value = relData;
        stats.value = statsData;

        // 加载动态
        loadUserMoments();
    } catch (e) {
        errorSnapshot.value = extractApiErrorSnapshot(e);
    } finally {
        loading.value = false;
    }
}

async function loadUserMoments() {
    loadingMoments.value = true;
    try {
        const coords = await geolocation.getCoordinates();
        const res = await momentsService.getUserMoments({
            user_id: userId,
            reference_point: {
              latitude: coords.latitude,
              longitude: coords.longitude,
            },
            limit: 5,
        });
        moments.value = res.items;
    } catch (e) {
        // 可能是权限不足（不可见）
    } finally {
        loadingMoments.value = false;
    }
}

async function toggleBlock() {
    if (!requireAuth('屏蔽用户')) return
    if (
        !confirm(
            relationship.value?.is_blocked
                ? "确定要解除拉黑吗？"
                : "确定要拉黑该用户吗？",
        )
    )
        return;
    try {
        if (relationship.value?.is_blocked) {
            await usersService.unblock({ user_id: userId });
            feedback.success("已解除拉黑");
        } else {
            await usersService.block({ user_id: userId });
            feedback.success("已拉黑");
        }
        const rel = await usersService.getRelationship({ user_id: userId });
        relationship.value = rel;
    } catch (e) {
        feedback.error("操作失败");
    }
}

function enterChat() {
    if (!requireAuth('发送消息')) return
    router.push(`/chat/users/${userId}`);
}

async function toggleLikeMoment(moment: MomentItem) {
    // 未登录时 is_liked 不存在（undefined），需要先登录
    if (moment.is_liked === undefined) {
        if (!requireAuth('点赞')) return
        return
    }
    
    // 复用 MomentCard 的逻辑，或者简单的 API 调用
    try {
        if (moment.is_liked) {
            await momentsService.unlikeMoment({ moment_id: moment.moment_id });
            moment.is_liked = false;
            moment.likes_count--;
        } else {
            await momentsService.likeMoment({ moment_id: moment.moment_id });
            moment.is_liked = true;
            moment.likes_count++;
        }
    } catch (e) {
        // error
    }
}
</script>
