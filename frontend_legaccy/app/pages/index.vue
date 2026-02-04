<template>
    <div class="flex flex-col gap-6">
        <!-- Header -->
        <div class="flex items-center justify-between">
            <div>
                <h1 class="text-2xl font-bold text-neutral-900 dark:text-white">
                    HotSpot
                </h1>
                <p class="text-neutral-500 dark:text-neutral-400 mt-1">
                    {{ headerDescription }}
                </p>
            </div>
            
            <div class="flex gap-2">
                <template v-if="!isHydrated">
                    <USkeleton class="h-8 w-20 rounded-full" />
                    <USkeleton class="h-8 w-20 rounded-full" />
                </template>
                <template v-else>
                    <div v-for="link in headerLinks" :key="link.label">
                        <UButton
                            :label="link.label"
                            :icon="link.icon"
                            :color="link.color"
                            :variant="link.variant"
                            @click="link.onClick"
                        />
                    </div>
                </template>
            </div>
        </div>

        <div class="space-y-8">
            <!-- 探索卡片 -->
            <div class="space-y-4">
                <div class="flex items-center gap-2 px-1">
                    <UIcon
                        name="i-lucide-compass"
                        class="size-5 text-primary-500"
                    />
                    <h2
                        class="font-semibold text-lg text-neutral-900 dark:text-white"
                    >
                        探索周边
                    </h2>
                </div>
                
                <div class="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-6">
                    <UCard
                        v-for="item in exploreItems"
                        :key="item.to"
                        variant="outline"
                        class="card-feature"
                        @click="navigateTo(item.to)"
                    >
                        <div class="flex items-start gap-4">
                            <div class="lsyb-icon-container-sm card-feature-icon">
                                <UIcon :name="item.icon" class="size-6" />
                            </div>
                            <div class="flex-1 min-w-0">
                                <h3 class="font-medium text-neutral-900 dark:text-white card-feature-title">
                                    {{ item.title }}
                                </h3>
                                <p class="text-sm text-neutral-500 dark:text-neutral-400 mt-1">
                                    {{ item.description }}
                                </p>
                            </div>
                            <UIcon name="i-lucide-chevron-right" class="size-5 text-neutral-300 dark:text-neutral-600 card-feature-arrow" />
                        </div>
                    </UCard>
                </div>
            </div>
            
            <!-- 个人中心/统计 -->
            <div class="space-y-4">
                <div class="flex items-center justify-between px-1">
                    <div class="flex items-center gap-2">
                        <UIcon
                            name="i-lucide-user-circle"
                            class="size-5 text-primary-500"
                        />
                        <h2
                            class="font-semibold text-lg text-neutral-900 dark:text-white"
                        >
                            我的
                        </h2>
                    </div>
                    <UBadge
                        v-if="isHydrated && isLoggedIn && unreadCount > 0"
                        color="error"
                        variant="soft"
                        size="sm"
                    >
                        {{ unreadCount > 99 ? "99+" : unreadCount }} 条未读消息
                    </UBadge>
                </div>

                <div
                    v-if="!isHydrated"
                    class="grid grid-cols-2 md:grid-cols-4 gap-4"
                >
                    <USkeleton
                        v-for="i in 4"
                        :key="i"
                        class="h-28 rounded-2xl"
                    />
                </div>

                <StatsGrid
                    v-else-if="isLoggedIn"
                    :stats="statsItems"
                    :columns="4"
                />

                <!-- 未登录引导 -->
                <div v-else class="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-6">
                    <!-- 快速体验 - 可点击 -->
                    <UCard variant="outline" class="card-feature" @click="createTempUser">
                        <div class="flex flex-col items-center text-center py-4 space-y-3">
                            <div class="lsyb-icon-container-md card-feature-icon">
                                <UIcon name="i-lucide-zap" class="size-8" />
                            </div>
                            <h3 class="font-medium text-lg card-feature-title">快速体验</h3>
                            <p class="text-sm text-neutral-500 dark:text-neutral-400">
                                一键创建访客账号，无需注册
                            </p>
                        </div>
                    </UCard>
                    <!-- 隐私设置 - 禁用状态 -->
                    <UCard variant="outline" class="opacity-50">
                        <div class="flex flex-col items-center text-center py-4 space-y-3">
                            <div class="lsyb-icon-container-neutral">
                                <UIcon name="i-lucide-shield-check" class="size-8" />
                            </div>
                            <h3 class="font-medium text-lg text-neutral-900 dark:text-white">隐私设置</h3>
                            <p class="text-sm text-neutral-500 dark:text-neutral-400">
                                登录后管理可见性与隐私
                            </p>
                        </div>
                    </UCard>
                    <!-- 通知中心 - 禁用状态 -->
                    <UCard variant="outline" class="opacity-50">
                        <div class="flex flex-col items-center text-center py-4 space-y-3">
                            <div class="lsyb-icon-container-neutral">
                                <UIcon name="i-lucide-bell" class="size-8" />
                            </div>
                            <h3 class="font-medium text-lg text-neutral-900 dark:text-white">通知中心</h3>
                            <p class="text-sm text-neutral-500 dark:text-neutral-400">
                                登录后查看实时互动消息
                            </p>
                        </div>
                    </UCard>
                </div>
            </div>
        </div>
    </div>
</template>

<script setup lang="ts">
import { getPageSeoConfig, buildSeoMeta } from "~/config/seo";
import { useAuthService } from "~/services/auth.service";
import { useUsersService } from "~/services/users.service";
import type { UserStatsData } from "~/types/api";
import type { AuthState } from "~/stores/auth.store";
import StatsGrid from "~/components/data/StatsGrid.vue";

const seoConfig = getPageSeoConfig("home");
definePageMeta({
    title: seoConfig.title,
    description: seoConfig.description,
    layout: "dashboard",
    requiresAuth: false  // 首页允许未登录用户访问
});
useSeoMeta(buildSeoMeta("home"));

const modal = useModal();
const { requireAuth } = useRequireAuth();
const { withErrorHandling } = useErrorHandler();
const authService = useAuthService();
const userService = useUsersService();
const feedback = useFeedbackStore();
const isHydrated = ref(false);

onMounted(() => {
    isHydrated.value = true;
});

const authStore = useAuthStore();
const messagesStore = useMessagesStore();
const locationStore = useLocationStore();

const DEFAULT_HYDRATED_AUTH_STATE: AuthState = {
    currentUserId: null,
    accounts: {},
    accountOrder: [],
};
const hydratedAuthState = useHydratedStore(
    authStore,
    "authState",
    DEFAULT_HYDRATED_AUTH_STATE,
);
const hydratedIsAuthed = useHydratedStore(authStore, "isAuthenticated", false);

const unreadCount = computed(() => messagesStore.totalUnreadCount);
const currentUser = computed(() => {
    const state = hydratedAuthState.value;
    if (!state.currentUserId) return null;
    const session = state.accounts[state.currentUserId];
    return session?.user ?? null;
});
const isLoggedIn = computed(() => hydratedIsAuthed.value);
const currentLocationName = computed(
    () => locationStore.locationName || "未设置位置",
);

// 确保服务端和客户端使用相同的默认值，避免 hydration mismatch
const headerDescription = computed(() => {
    // 在服务端和客户端首次渲染时，isLoggedIn 和 currentUser 可能不同
    // 使用 useHydratedStore 确保一致性
    if (!isLoggedIn.value) return "发现身边有趣的人和事";
    if (!currentUser.value) return "欢迎回来";
    return `你好，${currentUser.value.nickname} · 当前位置：${currentLocationName.value}`;
});

const openCreateMoment = async () => {
    const id = await modal.showCreateMoment();
    if (id) await navigateTo(`/moments/${id}`);
};

const openCreateGroup = async () => {
    const id = await modal.showCreateGroup();
    if (id) await navigateTo(`/groups/${id}`);
};

// 使用 useHydratedStore 确保服务端和客户端一致性
// 注意：headerLinks 在服务端应该返回空数组，避免渲染不同的内容
const headerLinks = computed(() => {
    // 在服务端和客户端首次渲染时，isLoggedIn 可能不同
    // 使用 useHydratedStore 确保一致性，但这里需要检查 isHydrated 避免在服务端渲染按钮
    if (!isHydrated.value || !isLoggedIn.value) return [];
    return [
        {
            label: "发瞬间",
            onClick: openCreateMoment,
            color: "primary",
            variant: "solid",
            icon: "i-lucide-square-pen",
        },
        {
            label: "建群",
            onClick: openCreateGroup,
            color: "neutral",
            variant: "outline",
            icon: "i-lucide-users-round",
        },
    ];
});

const createTempUser = async () => {
    await withErrorHandling(
        async () => {
        const authData = await authService.guest({
            nickname: `访客_${Date.now().toString(36).slice(-4)}`,
                client_type: "app",
            });
        authStore.setAccountSession(
            { user_id: authData.user_id, nickname: authData.nickname },
            authData.access_token,
            authData.refresh_token ?? null,
            new Date(authData.access_token_expires_at).getTime(),
                new Date(authData.refresh_token_expires_at).getTime(),
            );
            feedback.success("欢迎体验 HotSpot", "临时账户已创建");
        },
        {
            title: "创建失败",
            fallbackMessage: "请稍后重试",
        },
    );
};

const loadingStats = ref(false);
const myStats = ref<UserStatsData | null>(null);

const loadMyStats = async () => {
    if (!isLoggedIn.value) return;
    if (loadingStats.value) return;
    loadingStats.value = true;
    try {
        myStats.value = await userService.getMyStats();
    } catch {
        myStats.value = null;
    } finally {
        loadingStats.value = false;
    }
};

watch(
    isLoggedIn,
    (v) => {
        if (v) void loadMyStats();
        else myStats.value = null;
    },
    { immediate: true },
);

const statsItems = computed(() => {
    const s = myStats.value;
    return [
        {
            key: "moments",
            label: "瞬间",
            value: s?.moments_count ?? 0,
            icon: "i-lucide-image",
            color: "primary" as const,
            onClick: () => navigateTo("/moments"),
        },
        {
            key: "friends",
            label: "好友",
            value: s?.friends_count ?? 0,
            icon: "i-lucide-users-round",
            color: "primary" as const,
            onClick: () => navigateTo("/friends"),
        },
        {
            key: "followers",
            label: "粉丝",
            value: s?.followers_count ?? 0,
            icon: "i-lucide-users",
            color: "primary" as const,
            onClick: () => navigateTo("/followers"),
        },
        {
            key: "groups",
            label: "群组",
            value: s?.groups_count ?? 0,
            icon: "i-lucide-users-round",
            color: "primary" as const,
            onClick: () => navigateTo("/groups"),
        },
    ];
});

// 探索卡片数据
const exploreItems = [
    {
        to: "/users/nearby",
        icon: "i-lucide-users",
        title: "附近用户",
        description: "发现周围的朋友",
    },
    {
        to: "/groups/nearby",
        icon: "i-lucide-users-round",
        title: "附近群组",
        description: "发现可加入的群组",
    },
    {
        to: "/moments/nearby",
        icon: "i-lucide-image",
        title: "附近瞬间",
        description: "浏览周边动态",
    },
];

defineShortcuts({
    h: { usingInput: false, handler: () => navigateTo("/") },
    g: { usingInput: false, handler: () => navigateTo("/groups") },
    c: { usingInput: false, handler: () => navigateTo("/chat") },
    u: { usingInput: false, handler: () => navigateTo("/users/nearby") },
    p: { usingInput: false, handler: () => navigateTo("/moments") },
});
</script>
