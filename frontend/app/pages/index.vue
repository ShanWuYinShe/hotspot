<template>
    <div class="space-y-6" :style="{ viewTransitionName: 'main-content' }">
        <!-- 用户欢迎区域 -->
        <div
            v-if="isLoggedIn && currentUser"
            class="bg-gradient-blue-card rounded-xl p-6 border border-primary-subtle"
        >
            <div class="flex items-center gap-4">
                <div class="relative shrink-0">
                    <UChip color="success" size="lg" position="bottom-right">
                        <UAvatar
                            :alt="currentUser.nickname"
                            size="xl"
                            :src="currentUser.avatar_url"
                        >
                            {{
                                currentUser.nickname?.charAt(0) || "U"
                            }}
                        </UAvatar>
                    </UChip>
                </div>

                <div class="flex-1">
                    <h1
                        class="text-2xl font-bold text-primary-content mb-2"
                    >
                        {{ getGreetingTitle() }}
                    </h1>
                    <p class="text-secondary-content mb-3">
                        {{ getGreetingMessage() }}
                    </p>

                    <div class="flex items-center gap-3">
                        <UBadge
                            v-if="currentUser.user_type && isTempUser(currentUser.user_type)"
                            color="warning"
                            variant="soft"
                            size="sm"
                        >
                            <UIcon
                                name="i-heroicons-clock"
                                class="w-3 h-3 mr-1"
                            />
                            临时用户
                        </UBadge>
                        <UBadge color="success" variant="soft" size="sm">
                            <UIcon
                                name="i-heroicons-signal"
                                class="w-3 h-3 mr-1"
                            />
                            在线
                        </UBadge>
                        <div
                            class="flex items-center gap-1 text-sm text-secondary-content"
                        >
                            <UIcon
                                name="i-heroicons-chat-bubble-left-right"
                                class="icon-sm"
                            />
                            <span>{{
                                unreadCount > 0
                                    ? `${unreadCount} 条未读消息`
                                    : "暂无未读消息"
                            }}</span>
                        </div>
                    </div>
                </div>
            </div>
        </div>

        <!-- 访客欢迎区域 -->
        <div v-else-if="!isLoggedIn" class="text-center py-8">
            <div
                class="w-16 h-16 bg-linear-to-br from-primary-500 to-blue-600 rounded-2xl flex items-center justify-center mx-auto mb-4 shadow-lg"
            >
                <UIcon name="i-heroicons-fire" class="w-8 h-8 text-white" />
            </div>
            <h1
                class="text-3xl font-bold text-primary-content mb-2"
            >
                欢迎来到 HotSpot
            </h1>
            <p class="text-lg text-secondary-content mb-6">
                基于位置的社交平台，发现身边有趣的人和事
            </p>
            <div class="flex gap-3 justify-center">
                <UButton to="/auth/register" color="primary" size="lg">
                    <UIcon name="i-heroicons-user-plus" class="w-5 h-5 mr-2" />
                    立即注册
                </UButton>
                <UButton to="/auth/login" variant="outline" size="lg">
                    <UIcon
                        name="i-heroicons-arrow-right-on-rectangle"
                        class="w-5 h-5 mr-2"
                    />
                    登录
                </UButton>
            </div>
        </div>

        <!-- 快速操作区域 - 仅对未登录用户显示 -->
        <div v-if="!isLoggedIn">
            <div class="flex items-center gap-3 mb-4">
                <UIcon name="i-heroicons-bolt" class="w-5 h-5 text-blue" />
                <h2
                    class="text-xl font-semibold text-primary-content"
                >
                    快速操作
                </h2>
            </div>
            <p class="text-secondary-content mb-6">
                选择您想要进行的操作，快速开始使用 HotSpot
            </p>

            <UPageGrid>
                <UPageCard
                    title="附近用户"
                    description="发现周围的朋友"
                    icon="i-heroicons-users"
                    to="/users/nearby"
                    class="interactive-card cursor-pointer"
                />

                <UPageCard
                    title="我的群组"
                    description="管理群组聊天"
                    icon="i-heroicons-user-group"
                    to="/groups"
                    class="interactive-card cursor-pointer"
                />

                <UPageCard
                    title="动态分享"
                    description="分享精彩瞬间"
                    icon="i-heroicons-photo"
                    to="/moments"
                    class="interactive-card cursor-pointer"
                />

                <UPageCard
                    title="地图探索"
                    description="查看周边热点"
                    icon="i-heroicons-map"
                    to="/map"
                    class="interactive-card cursor-pointer"
                />
            </UPageGrid>
        </div>
        
        <!-- 已登录用户的统计信息 -->
        <div v-else-if="isLoggedIn">
            <div class="flex items-center gap-3 mb-4">
                <UIcon name="i-heroicons-chart-bar" class="w-5 h-5 text-blue" />
                <h2
                    class="text-xl font-semibold text-primary-content"
                >
                    概览
                </h2>
            </div>
            <p class="text-secondary-content mb-6">
                您的社交动态一览
            </p>

            <div class="grid grid-cols-2 md:grid-cols-4 gap-4">
                <div class="bg-card-elevated rounded-xl p-4 border border-default">
                    <div class="flex items-center gap-3 mb-2">
                        <div class="w-10 h-10 rounded-lg bg-primary-base flex items-center justify-center">
                            <UIcon name="i-heroicons-chat-bubble-left-right" class="w-5 h-5 text-accent" />
                        </div>
                    </div>
                    <div class="text-2xl font-bold text-primary-content">{{ unreadCount }}</div>
                    <div class="text-sm text-secondary-content">未读消息</div>
                </div>
                
                <div class="bg-card-elevated rounded-xl p-4 border border-default">
                    <div class="flex items-center gap-3 mb-2">
                        <div class="w-10 h-10 rounded-lg bg-success-subtle flex items-center justify-center">
                            <UIcon name="i-heroicons-user-group" class="w-5 h-5 text-success" />
                        </div>
                    </div>
                    <div class="text-2xl font-bold text-primary-content">--</div>
                    <div class="text-sm text-secondary-content">我的群组</div>
                </div>
                
                <div class="bg-card-elevated rounded-xl p-4 border border-default">
                    <div class="flex items-center gap-3 mb-2">
                        <div class="w-10 h-10 rounded-lg bg-warning-subtle flex items-center justify-center">
                            <UIcon name="i-heroicons-photo" class="w-5 h-5 text-warning" />
                        </div>
                    </div>
                    <div class="text-2xl font-bold text-primary-content">--</div>
                    <div class="text-sm text-secondary-content">我的动态</div>
                </div>
                
                <div class="bg-card-elevated rounded-xl p-4 border border-default">
                    <div class="flex items-center gap-3 mb-2">
                        <div class="w-10 h-10 rounded-lg bg-info-subtle flex items-center justify-center">
                            <UIcon name="i-heroicons-users" class="w-5 h-5 text-info" />
                        </div>
                    </div>
                    <div class="text-2xl font-bold text-primary-content">--</div>
                    <div class="text-sm text-secondary-content">好友数量</div>
                </div>
            </div>
        </div>

        <!-- 最近活动 -->
        <div>
            <div class="flex items-center justify-between w-full mb-4">
                <div class="flex items-center gap-3">
                    <UIcon
                        name="i-heroicons-clock"
                        class="w-5 h-5 text-blue"
                    />
                    <h2
                        class="text-xl font-semibold text-primary-content"
                    >
                        最近活动
                    </h2>
                </div>
                <UButton
                    v-if="isLoggedIn"
                    variant="outline"
                    size="sm"
                    @click="refreshActivities"
                    :loading="refreshing"
                >
                    <UIcon
                        name="i-heroicons-arrow-path"
                        class="w-4 h-4 mr-2"
                        :class="{ 'animate-spin': refreshing }"
                    />
                    刷新
                </UButton>
            </div>

            <p class="text-secondary-content mb-6">
                查看最新的社区动态和好友活动
            </p>

            <!-- 加载状态 -->
            <div v-if="activitiesLoading" class="space-y-4">
                <div
                    v-for="i in 3"
                    :key="i"
                    class="flex items-center gap-3 p-4 bg-surface rounded-lg"
                >
                    <USkeleton class="w-10 h-10 rounded-full" />
                    <div class="flex-1 space-y-2">
                        <USkeleton class="h-4 w-3/4" />
                        <USkeleton class="h-3 w-1/2" />
                    </div>
                    <USkeleton class="h-3 w-16" />
                </div>
            </div>

            <!-- 错误状态 -->
            <UEmpty
                v-else-if="activitiesError"
                icon="i-heroicons-exclamation-triangle"
                :title="activitiesError"
                description="请稍后重试"
            >
                <template #actions>
                    <UButton @click="fetchActivities" variant="outline" size="sm">
                        <UIcon name="i-heroicons-arrow-path" class="w-4 h-4 mr-2" />
                        重试
                    </UButton>
                </template>
            </UEmpty>

            <!-- 活动时间线 -->
            <div v-else-if="activities.length > 0">
                <UTimeline :items="timelineItems" size="sm" />
            </div>

            <!-- 空状态 -->
            <UEmpty
                v-else
                icon="i-heroicons-sparkles"
                title="暂无活动"
                description="开始探索，创建你的第一个活动吧！"
                :actions="[{ label: '发布动态', icon: 'i-heroicons-plus', to: '/moments' }]"
            />
        </div>

        <!-- 功能特色 -->
        <div v-if="!isLoggedIn">
            <div class="flex items-center gap-3 mb-4">
                <UIcon name="i-heroicons-star" class="w-5 h-5 text-blue" />
                <h2
                    class="text-xl font-semibold text-primary-content"
                >
                    为什么选择 HotSpot
                </h2>
            </div>

            <p class="text-secondary-content mb-6">
                基于位置的社交平台，让您轻松发现身边的精彩
            </p>

            <UPageColumns>
                <UPageCard
                    title="基于位置的发现"
                    description="发现附近用户与群组，快速开始交流"
                    icon="i-heroicons-map-pin"
                    class="lg:col-span-2"
                />

                <UPageCard
                    title="实时消息"
                    description="与朋友和群组实时沟通，支持多种消息类型"
                    icon="i-heroicons-chat-bubble-left-right"
                />

                <UPageCard
                    title="动态与互动"
                    description="发布动态，点赞互动，记录精彩"
                    icon="i-heroicons-photo"
                />

                <UPageCard
                    title="隐私安全"
                    description="端到端加密，保护您的隐私和数据安全"
                    icon="i-heroicons-shield-check"
                    class="lg:col-span-2"
                />
            </UPageColumns>
        </div>

        <!-- CTA 区域 -->
        <div
            v-if="!isLoggedIn"
            class="bg-gradient-blue-deep rounded-xl p-8 text-center border border-primary-subtle"
        >
            <div class="max-w-2xl mx-auto">
                <h2 class="text-3xl font-bold mb-4 text-primary-content">准备开始了吗？</h2>
                <p class="text-xl text-secondary-strong mb-8">
                    立即加入 HotSpot，开始您的社交之旅
                </p>

                <div class="flex flex-col sm:flex-row gap-4 justify-center">
                    <UButton
                        to="/auth/register"
                        color="primary"
                        size="lg"
                    >
                        <UIcon
                            name="i-heroicons-user-plus"
                            class="w-5 h-5 mr-2"
                        />
                        立即注册
                    </UButton>
                    <UButton
                        to="/auth/login"
                        variant="outline"
                        color="primary"
                        size="lg"
                    >
                        <UIcon
                            name="i-heroicons-arrow-right-on-rectangle"
                            class="w-5 h-5 mr-2"
                        />
                        登录账户
                    </UButton>
                </div>
            </div>
        </div>
    </div>
</template>

<script setup lang="ts">
import { getPageSeoConfig, buildSeoMeta } from "~/config/seo";
import { isTempUser } from "~/utils/userTypes";

// 从数据源获取 SEO 配置
const seoConfig = getPageSeoConfig("home");

// 使用 Nuxt v4 的 definePageMeta
definePageMeta({
    title: seoConfig.title,
    description: seoConfig.description,
    layout: "dashboard",
});

// 直接使用 Nuxt v4 的 useSeoMeta
useSeoMeta(buildSeoMeta("home"));

const modal = useModal();

// 使用水合安全的 store 访问
const authStore = useAuthStore();
const messagesStore = useMessagesStore();
const settingsStore = useSettingsStore();
const unreadCount = computed(() => messagesStore.totalUnreadCount);

// 水合安全：SSR 时默认未登录，onMounted 后同步 store 值
const isLoggedIn = ref(false);
const currentUser = ref<typeof authStore.user>(null);

// 问候消息（使用水合安全的 currentUser）
const getGreetingTitle = () => {
    if (currentUser.value) {
        return `你好，${currentUser.value.nickname}！`;
    }
    return "发现身边的精彩";
};

const getGreetingMessage = () => {
    const hour = new Date().getHours();
    if (currentUser.value) {
        if (hour < 6) return "夜深了，注意休息哦";
        if (hour < 12) return "早上好！新的一天开始了";
        if (hour < 18) return "下午好！今天过得怎么样？";
        if (hour < 22) return "晚上好！放松一下吧";
        return "夜深了，早点休息哦";
    } else {
        return "基于位置的社交平台，与朋友分享生活的精彩时刻";
    }
};

// 活动数据
const refreshing = ref(false);
const activities = ref<any[]>([]);
const activitiesLoading = ref(false);
const activitiesError = ref<string | null>(null);

// 时间线数据转换 - 符合 UTimeline 组件格式
const timelineItems = computed(() => {
    return activities.value.map((activity) => ({
        title: activity.content,
        description: activity.time,
        icon: activity.icon || 'i-heroicons-sparkles',
        avatar: activity.user ? {
            alt: activity.user,
            fallback: activity.user?.charAt(0) || "U",
        } : undefined,
    }));
});

// 获取活动数据
const fetchActivities = async () => {
    try {
        activitiesLoading.value = true;
        activitiesError.value = null;

        // 注意：活动功能需要后端 API 支持，当前使用模拟数据
        // const response = await $fetch('/api/activities/recent');
        // activities.value = response.data;

        activities.value = [];
    } catch (error) {
        console.error("获取活动数据失败:", error);
        activitiesError.value = "获取活动数据失败";
        activities.value = [];
    } finally {
        activitiesLoading.value = false;
    }
};

// 刷新活动
const refreshActivities = async () => {
    refreshing.value = true;
    await fetchActivities();
    refreshing.value = false;
};

// 生命周期
onMounted(async () => {
    // 水合安全：同步用户状态（onMounted 只在客户端执行，此时 store 已从 localStorage 恢复）
    isLoggedIn.value = !!authStore.user;
    currentUser.value = authStore.user;
    
    // 监听用户状态变化
    watch(() => authStore.user, (user) => {
        isLoggedIn.value = !!user;
        currentUser.value = user;
    });
    
    await fetchActivities();
    
    // 用户引导逻辑
    const currentTime = Date.now();

    if (!settingsStore.hasVisited && !isLoggedIn.value) {
        setTimeout(() => modal.showWelcome(), 1000);
        settingsStore.markVisited();
    } else if (!isLoggedIn.value) {
        const shouldShowPrompt =
            !settingsStore.neverShowLoginPrompt &&
            (settingsStore.lastLoginPromptTime === 0 ||
                currentTime - settingsStore.lastLoginPromptTime > 24 * 60 * 60 * 1000);

        if (shouldShowPrompt) {
            setTimeout(() => {
                modal.showLoginPrompt();
                settingsStore.updateLastLoginPromptTime();
            }, 3000);
        }
    }

    if (settingsStore.lastVisitTime > 0) {
        settingsStore.updateLastVisitTime();
    }
});

// 快捷键管理 - 使用 Nuxt UI v4 原生支持
defineShortcuts({
    h: {
        usingInput: false,
        handler: () => navigateTo("/"),
    },
    m: {
        usingInput: false,
        handler: () => navigateTo("/map"),
    },
    g: {
        usingInput: false,
        handler: () => navigateTo("/groups"),
    },
    c: {
        usingInput: false,
        handler: () => navigateTo("/chat"),
    },
    u: {
        usingInput: false,
        handler: () => navigateTo("/users/nearby"),
    },
    p: {
        usingInput: false,
        handler: () => navigateTo("/moments"),
    },
    r: {
        usingInput: false,
        handler: refreshActivities,
    },
});
</script>
