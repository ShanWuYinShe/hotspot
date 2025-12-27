<template>
    <div class="w-full">

        <!-- 加载中 -->
        <div v-if="loading">
            <div class="grid grid-cols-1 lg:grid-cols-3 gap-6">
                <div class="lg:col-span-1">
                    <USkeleton class="h-80" />
                </div>
                <div class="lg:col-span-2 space-y-6">
                    <USkeleton class="h-48" />
                    <USkeleton class="h-64" />
                </div>
            </div>
        </div>

        <!-- 用户不存在 -->
        <div
            v-else-if="!user"
            class="max-w-md mx-auto py-16 text-center"
        >
            <UEmpty
                icon="i-lucide-user-x"
                title="用户不存在"
                description="该用户可能已被删除或不存在"
            />
        </div>

        <!-- 用户详情 -->
        <div
            v-else
            class="grid grid-cols-1 lg:grid-cols-3 gap-6"
        >
            <!-- 左侧：用户信息卡片 -->
            <div class="lg:col-span-1 space-y-6">
                <!-- 主要信息卡片 -->
                <UCard variant="subtle" class="overflow-hidden">
                    <div class="flex items-center justify-end gap-1 pt-2">
                        <UButton
                            variant="ghost"
                            color="neutral"
                            icon="i-lucide-share"
                            :square="true"
                            size="sm"
                            @click="shareUser"
                        />
                        <UDropdownMenu :items="userActions">
                            <UButton
                                variant="ghost"
                                color="neutral"
                                icon="i-lucide-ellipsis"
                                :square="true"
                                size="sm"
                            />
                        </UDropdownMenu>
                    </div>

                    <div class="pb-2">
                        <div class="relative -mt-10 mb-3">
                            <UAvatar
                                :src="user.avatar_url"
                                :alt="user.nickname"
                                size="3xl"
                                :chip="userStatus.is_online ? { color: 'success', position: 'bottom-right', size: '2xs' } : false"
                            >
                                {{ user.nickname?.charAt(0) || 'U' }}
                            </UAvatar>
                        </div>

                        <div class="space-y-2">
                            <UUser
                                :name="user.nickname || '未设置昵称'"
                                :description="user.bio || undefined"
                                :avatar="{
                                  src: user.avatar_url,
                                  alt: user.nickname,
                                  size: '3xl',
                                  chip: userStatus.is_online ? { color: 'success', position: 'bottom-right', size: '2xs' } : false
                                }"
                            />

                            <div class="flex flex-wrap gap-2">
                                <UBadge color="neutral" variant="soft" size="xs">
                                  ID: {{ user.user_id }}
                                </UBadge>

                                <UBadge
                                  v-if="user.geo?.location_name"
                                  color="neutral"
                                  variant="soft"
                                  size="xs"
                                  icon="i-lucide-map-pin"
                                  class="max-w-full truncate"
                                >
                                  {{ user.geo.location_name }}
                                </UBadge>

                                <UBadge
                                  v-if="user.geo?.distance"
                                  color="neutral"
                                  variant="soft"
                                  size="xs"
                                  icon="i-lucide-plane"
                                >
                                  {{ formatDistance(user.geo.distance) }}
                                </UBadge>
                            </div>

                            <div class="flex gap-2 pt-2">
                                <UButton
                                    class="flex-1"
                                    icon="i-lucide-messages-square"
                                    @click="() => { if (user && requireAuth('发送消息')) navigateTo(`/chat/users/${user.user_id}`) }"
                                >
                                    私聊
                                </UButton>

                                <UButton
                                    v-if="isMutualFollow"
                                    variant="soft"
                                    color="success"
                                    icon="i-lucide-user-round-check"
                                    :loading="operating"
                                    @click="handleUnfollow"
                                >
                                    好友
                                </UButton>
                                <UButton
                                    v-else-if="isFollowing"
                                    variant="outline"
                                    icon="i-lucide-heart"
                                    :loading="operating"
                                    @click="handleUnfollow"
                                >
                                    已关注
                                </UButton>
                                <UButton
                                    v-else
                                    variant="outline"
                                    icon="i-lucide-heart"
                                    :loading="operating"
                                    @click="handleFollow"
                                >
                                    关注
                                </UButton>
                            </div>
                        </div>
                    </div>
                </UCard>

                <!-- 统计卡片 -->
                <UCard variant="subtle">
                    <template #header>
                        <div class="flex items-center justify-between">
                            <div class="text-lg font-semibold">数据统计</div>
                        </div>
                    </template>
                    <div class="grid grid-cols-3 gap-3">
                        <UCard variant="soft">
                          <div class="flex flex-col items-center justify-center gap-2">
                            <p class="text-xl font-bold">
                                <USkeleton v-if="!userStats" class="h-8 w-12 mx-auto" />
                                <span v-else>{{ userStats.moments_count }}</span>
                            </p>
                            <UBadge color="neutral" variant="soft" size="xs">动态</UBadge>
                          </div>
                        </UCard>
                        <UCard variant="soft">
                          <div class="flex flex-col items-center justify-center gap-2">
                            <p class="text-xl font-bold">
                                <USkeleton v-if="!userStats" class="h-8 w-12 mx-auto" />
                                <span v-else>{{ userStats.friends_count }}</span>
                            </p>
                            <UBadge color="neutral" variant="soft" size="xs">好友</UBadge>
                          </div>
                        </UCard>
                        <UCard variant="soft">
                          <div class="flex flex-col items-center justify-center gap-2">
                            <p class="text-xl font-bold">
                                <USkeleton v-if="!userStats" class="h-8 w-12 mx-auto" />
                                <span v-else>{{ userStats.groups_count }}</span>
                            </p>
                            <UBadge color="neutral" variant="soft" size="xs">群组</UBadge>
                          </div>
                        </UCard>
                    </div>
                </UCard>

                <!-- 在线状态卡片 -->
                <UCard variant="subtle">
                    <template #header>
                        <div class="text-lg font-semibold">在线状态</div>
                    </template>
                    <div class="flex items-center gap-3">
                        <UBadge
                          :color="userStatus.is_online ? 'success' : 'neutral'"
                          variant="soft"
                          size="sm"
                        >
                          {{ userStatus.is_online ? '在线' : '离线' }}
                        </UBadge>
                        <UBadge
                          v-if="!userStatus.is_online && userStatus.last_online"
                          color="neutral"
                          variant="soft"
                          size="xs"
                        >
                          最后在线：{{ formatTime(userStatus.last_online) }}
                        </UBadge>
                    </div>
                </UCard>
            </div>

            <!-- 右侧：动态和更多内容 -->
            <div class="lg:col-span-2 space-y-6">
                <!-- 动态列表 -->
                <UCard variant="subtle">
                    <template #header>
                        <div class="flex items-center gap-3">
                            <UBadge color="primary" variant="soft" size="lg" square icon="i-lucide-image" />
                            <div>
                                <h3 class="text-lg font-semibold">TA的动态</h3>
                                <UBadge color="neutral" variant="soft" size="xs" class="mt-1">
                                  <USkeleton v-if="!userStats" class="h-4 w-12" />
                                  <span v-else>{{ userStats.moments_count }} 条动态</span>
                                </UBadge>
                            </div>
                        </div>
                    </template>

                    <!-- 加载中 -->
                    <div
                        v-if="loadingMoments"
                        class="space-y-3"
                    >
                        <USkeleton
                            class="h-24"
                            v-for="i in 3"
                            :key="i"
                        />
                    </div>

                    <!-- 动态列表 -->
                    <div v-else-if="userMoments.length > 0">
                        <UPageList divide>
                            <UPageCard
                                v-for="moment in userMoments"
                                :key="moment.id"
                                :title="moment.description || '动态'"
                                :description="moment.geo?.location_name ? `${moment.geo.location_name} · ${formatTime(moment.created_at)}` : formatTime(moment.created_at)"
                                icon="i-lucide-image"
                                variant="ghost"
                                :to="`/moments/${moment.id}`"
                            >
                                <template #footer>
                                    <div class="flex items-center gap-2 flex-wrap">
                                        <UBadge color="neutral" variant="soft" size="xs" icon="i-lucide-heart">
                                          {{ moment.likes_count }}
                                        </UBadge>
                                        <UBadge color="neutral" variant="soft" size="xs" icon="i-lucide-message-circle">
                                          {{ moment.comments_count }}
                                        </UBadge>
                                        <UBadge v-if="moment.tags?.length" color="neutral" variant="soft" size="xs" class="max-w-full truncate">
                                          #{{ moment.tags.slice(0, 3).join(' #') }}
                                        </UBadge>
                                    </div>
                                </template>
                            </UPageCard>
                        </UPageList>

                        <!-- 加载更多 -->
                        <div
                            v-if="hasMoreMoments"
                            class="pt-3 text-center"
                        >
                            <UButton
                                variant="ghost"
                                color="neutral"
                                :loading="loadingMoreMoments"
                                @click="loadMoreMoments"
                            >
                                加载更多
                            </UButton>
                        </div>
                    </div>

                    <!-- 空状态 -->
                    <div
                        v-else
                        class="py-8"
                    >
                        <UEmpty
                            icon="i-lucide-image"
                            title="暂无动态"
                            description="TA还没有发布任何动态"
                        />
                    </div>
                </UCard>

                <!-- 好友列表/群组列表：未实现前不展示，避免“假功能”占位 -->
            </div>
        </div>
    </div>
</template>

<script setup lang="ts">
import { useUsersService } from '~/services/users.service';
import { useMomentsService } from '~/services/moments.service';
import type { MomentSearchItem, UserSearchItem, UserStatsData } from '~/types/api';

definePageMeta({
    title: '用户资料',
    description: '查看用户信息与动态',
    layout: 'dashboard',
    dashboardHeader: true
});

const route = useRoute();
const router = useRouter();
// 返回：dashboard layout 内已经有统一的“返回”逻辑（避免每页重复注册 header actions）
const userService = useUsersService();
const usersStore = useUsersStore();
const momentsService = useMomentsService();
const geolocation = useGeolocation();
const authStore = useAuthStore();
const hydratedMe = useHydratedStore(authStore, 'user', null) as Ref<{ user_id?: number } | null>
const feedback = useFeedbackStore()
const { formatTime, formatDistance } = useFormatters();
const modal = useModal();
const { withErrorHandling } = useErrorHandler();
const { share, isSupported: isShareSupported } = useShare()
const { copy, isSupported: isClipboardSupported } = useClipboard()
const { requireAuth } = useRequireAuth()

const userId = Number(route.params.id);

// 如果是自己的用户ID，跳转到个人资料页面
onBeforeMount(() => {
    if (import.meta.server) return
    // 用水合安全的 user，避免首屏 authStore.user 为空导致判断失效
    if (authStore.isAuthenticated && hydratedMe.value?.user_id === userId) {
        navigateTo('/profile', { replace: true });
    }
});

// 状态变量
const loading = ref(true);
const operating = ref(false);
const user = ref<UserSearchItem | null>(null);
const userMoments = ref<MomentSearchItem[]>([]);
const loadingMoments = ref(false);
const loadingMoreMoments = ref(false);
const hasMoreMoments = ref(false);

// 在线状态
const userStatus = ref({
    is_online: false,
    last_online: ''
});

// 关系状态
const isFollowing = ref(false);
const isMutualFollow = ref(false);

// 用户统计
const userStats = ref<UserStatsData | null>(null);

// 用户操作菜单
const userActions = computed(() => {
    if (!user.value) return [];

    return [
        [{
            label: isFollowing.value ? '取消关注' : '关注用户',
            icon: isFollowing.value ? 'i-lucide-heart-off' : 'i-lucide-heart',
            onSelect: () => isFollowing.value ? handleUnfollow() : handleFollow()
        }],
        [{
            label: '举报用户',
            icon: 'i-lucide-shield-alert',
            onSelect: () => reportUser()
        },
        {
            label: '拉黑用户',
            icon: 'i-lucide-ban',
            onSelect: () => blockUser()
        }]
    ];
});

// 加载用户资料
onMounted(async () => {
    // 1. 尝试从 store 获取缓存数据
    const cachedUser = usersStore.getUser(userId);
    if (cachedUser) {
        user.value = cachedUser;
        userStatus.value = {
            is_online: cachedUser.is_online ?? false,
            last_online: cachedUser.last_online ?? ''
        };
        if (cachedUser.stats) {
            userStats.value = cachedUser.stats;
        }
        loading.value = false;
    }

    // 2. 获取最新数据
    await loadUserProfile();
});

// 监听认证状态变化获取关系
watch(
    () => [authStore.isAuthenticated, user.value] as const,
    async ([isAuthenticated, userData]) => {
        if (isAuthenticated && userData) {
            await withErrorHandling(async () => {
                const relationship = await userService.getRelationship({ id: userId });
                isMutualFollow.value = relationship.is_mutual_follow ?? false;
                isFollowing.value = relationship.is_following ?? false;
            }, { showToast: false })
        }
    },
    { immediate: true }
);

async function loadUserProfile() {
    await withErrorHandling(async () => {
        const coords = await geolocation.getCoordinates();

        const response = await userService.openTheDoor({
            id: userId,
            latitude: coords.latitude,
            longitude: coords.longitude
        });
        user.value = response;
        usersStore.setUser(response);

        // 在线状态已包含在 user 中，无需单独获取
        userStatus.value = {
            is_online: response.is_online ?? false,
            last_online: response.last_online ?? ''
        };

        await Promise.all([
            loadUserStats(),
            loadUserMoments()
        ]);

    }, {
        title: '加载失败',
        // 如果已经有缓存数据显示，则不显示全屏 loading
        loading: user.value ? undefined : loading
    })
}

async function loadUserStats() {
    await withErrorHandling(async () => {
        const stats = await userService.getStats({ id: userId });
        userStats.value = stats;
        usersStore.setUserStats(userId, stats);
    }, { showToast: false })
}

async function loadUserMoments() {
    await withErrorHandling(async () => {
        const coords = await geolocation.getCoordinates();
        const resp = await momentsService.getUserMoments({
            id: userId,
            latitude: coords.latitude,
            longitude: coords.longitude,
            limit: 20
        });

        userMoments.value = resp?.items || [];
        hasMoreMoments.value = resp?.pagination.has_more || false;
    }, {
        loading: loadingMoments,
        showToast: false
    })
}

async function loadMoreMoments() {
    if (loadingMoreMoments.value || !hasMoreMoments.value) return;

    await withErrorHandling(async () => {
        const lastMoment = userMoments.value[userMoments.value.length - 1];
        const coords = await geolocation.getCoordinates();

        const resp = await momentsService.getUserMoments({
            id: userId,
            latitude: coords.latitude,
            longitude: coords.longitude,
            cursor: lastMoment?.id,
            limit: 20
        });

        userMoments.value.push(...(resp?.items || []));
        hasMoreMoments.value = resp?.pagination.has_more || false;
    }, {
        title: '加载失败',
        loading: loadingMoreMoments
    })
}

async function handleFollow() {
    if (operating.value) return;
    if (!requireAuth('关注用户')) return

    await withErrorHandling(async () => {
        await userService.follow({ id: userId });
        isFollowing.value = true;
        feedback.success('关注成功', `已关注 ${user.value?.nickname}`)
        await loadUserProfile();
    }, {
        title: '操作失败',
        loading: operating
    })
}

async function handleUnfollow() {
    if (operating.value) return;
    if (!requireAuth('取消关注')) return

    const message = isMutualFollow.value
        ? `取消关注后将解除与"${user.value?.nickname || '该用户'}"的好友关系，确定吗？`
        : `确定要取消关注"${user.value?.nickname || '该用户'}"吗？`;

    const confirmed = await modal.showConfirm({
        title: '确认取消关注',
        message,
        confirmText: '确认',
        cancelText: '取消',
        confirmColor: 'error'
    });

    if (confirmed) {
        await withErrorHandling(async () => {
            await userService.unfollow({ id: userId });
            isFollowing.value = false;
            isMutualFollow.value = false;
            feedback.success('已取消关注')
        }, {
            title: '操作失败',
            loading: operating
        })
    }
}

async function shareUser() {
    if (!user.value) return;

    const shareData = {
        title: `${user.value.nickname} 的个人资料`,
        text: `来看看 ${user.value.nickname} 在 HotSpot 上的精彩内容！`,
        url: useRequestURL().href
    }

    try {
        if (isShareSupported.value) {
            await share(shareData)
        } else if (isClipboardSupported.value) {
            await copy(shareData.url)
            feedback.success('链接已复制', '用户链接已复制到剪贴板')
        }
    } catch {
        // 忽略
    }
}

async function reportUser() {
    if (!user.value) return;
    if (!requireAuth('提交举报')) return

    await modal.showReport('user', String(user.value.user_id));
    feedback.success('举报已提交', '我们会尽快处理您的举报')
}

async function blockUser() {
    if (!user.value) return;
    if (!requireAuth('拉黑用户')) return

    const confirmed = await modal.showConfirm({
        title: '拉黑用户',
        message: `确定要拉黑 ${user.value.nickname} 吗？拉黑后您将无法看到对方的任何内容，对方也无法与您互动。`,
        confirmText: '确认拉黑',
        cancelText: '取消',
        confirmColor: 'error'
    });

    if (!confirmed) return;

    await withErrorHandling(async () => {
        await userService.block({ id: user.value!.user_id });
        feedback.success('拉黑成功', `已拉黑 ${user.value!.nickname}`)
        router.back();
    }, {
        title: '拉黑失败'
    })
}
</script>
