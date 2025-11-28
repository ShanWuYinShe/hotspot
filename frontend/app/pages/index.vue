<template>
    <div class="space-y-6" :style="{ viewTransitionName: 'main-content' }">
        <!-- 已登录用户仪表板 -->
        <template v-if="displayedUserResponse">
            <!-- 用户状态卡片 -->
            <div class="flex items-center gap-4 p-4 bg-white dark:bg-neutral-900 rounded-xl border border-neutral-200 dark:border-neutral-800">
                <UChip color="success" size="md" position="bottom-right">
                    <UAvatar
                        :alt="displayedUserResponse.nickname"
                        size="lg"
                        :src="displayedUserResponse.avatar_url"
                    >
                        {{ displayedUserResponse.nickname?.charAt(0) || "U" }}
                    </UAvatar>
                </UChip>

                <div class="flex-1 min-w-0">
                    <h1 class="text-lg font-semibold text-neutral-900 dark:text-white truncate">
                        {{ displayedUserResponse.nickname }}
                    </h1>
                    <p class="text-sm text-neutral-500 dark:text-neutral-400">
                        {{ getGreetingMessage() }}
                    </p>
                </div>

                <div class="flex items-center gap-2">
                    <UBadge
                        v-if="isTempUser(displayedUserResponse.user_id)"
                        color="warning"
                        variant="soft"
                        size="xs"
                    >
                        临时用户
                    </UBadge>
                    <UBadge
                        v-if="unreadCount > 0"
                        color="error"
                        variant="solid"
                        size="xs"
                    >
                        {{ unreadCount > 99 ? '99+' : unreadCount }} 条消息
                    </UBadge>
                </div>
            </div>

            <!-- 快速操作网格 - 更紧凑 -->
            <div class="grid grid-cols-2 sm:grid-cols-3 lg:grid-cols-6 gap-3">
                <NuxtLink 
                    v-for="action in quickActions" 
                    :key="action.to"
                    :to="action.to" 
                    class="group flex flex-col items-center gap-2 p-4 bg-white dark:bg-neutral-900 rounded-xl border border-neutral-200 dark:border-neutral-800 hover:border-primary-300 dark:hover:border-primary-700 hover:shadow-lg transition-all"
                >
                    <div :class="['w-10 h-10 rounded-lg flex items-center justify-center transition-transform group-hover:scale-110', action.bgClass]">
                        <UIcon :name="action.icon" class="w-5 h-5 text-white" />
                    </div>
                    <span class="text-sm font-medium text-neutral-700 dark:text-neutral-300">{{ action.label }}</span>
                    <UBadge
                        v-if="action.badge"
                        :color="action.badgeColor || 'error'"
                        size="xs"
                        class="absolute -top-1 -right-1"
                    >
                        {{ action.badge }}
                    </UBadge>
                </NuxtLink>
            </div>

            <!-- 最近活动 -->
            <div class="bg-white dark:bg-neutral-900 rounded-xl border border-neutral-200 dark:border-neutral-800 overflow-hidden">
                <div class="flex items-center justify-between p-4 border-b border-neutral-200 dark:border-neutral-800">
                    <h2 class="font-semibold text-neutral-900 dark:text-white flex items-center gap-2">
                        <UIcon name="i-heroicons-clock" class="w-4 h-4" />
                        最近活动
                    </h2>
                    <UButton
                        variant="ghost"
                        size="xs"
                        :loading="refreshing"
                        @click="refreshActivities"
                    >
                        <UIcon name="i-heroicons-arrow-path" class="w-4 h-4" />
                    </UButton>
                </div>

                <div class="p-4">
                    <!-- 加载状态 -->
                    <div v-if="activitiesLoading" class="space-y-3">
                        <div v-for="i in 3" :key="i" class="flex items-center gap-3">
                            <USkeleton class="w-8 h-8 rounded-full" />
                            <div class="flex-1 space-y-1">
                                <USkeleton class="h-3 w-3/4" />
                                <USkeleton class="h-2 w-1/2" />
                            </div>
                        </div>
                    </div>

                    <!-- 空状态 -->
                    <div v-else-if="activities.length === 0" class="text-center py-8">
                        <UIcon name="i-heroicons-inbox" class="w-10 h-10 text-neutral-300 dark:text-neutral-600 mx-auto mb-3" />
                        <p class="text-sm text-neutral-500 dark:text-neutral-400 mb-3">暂无最近活动</p>
                        <UButton to="/moments" size="xs" variant="soft">
                            发布第一条动态
                        </UButton>
                    </div>

                    <!-- 活动列表 -->
                    <div v-else class="space-y-3">
                        <UTimeline :items="timelineItems" />
                    </div>
                </div>
            </div>
        </template>

        <!-- 未登录用户 - 简洁登录提示 -->
        <template v-else>
            <div class="text-center py-16">
                <div class="w-16 h-16 bg-gradient-to-br from-primary-500 to-blue-600 rounded-2xl flex items-center justify-center mx-auto mb-6 shadow-lg">
                    <UIcon name="i-heroicons-fire" class="w-8 h-8 text-white" />
                </div>
                <h1 class="text-2xl font-bold text-neutral-900 dark:text-white mb-2">
                    欢迎使用 HotSpot
                </h1>
                <p class="text-neutral-600 dark:text-neutral-400 mb-8 max-w-md mx-auto">
                    基于位置的社交平台，发现身边有趣的人和事
                </p>
                <div class="flex gap-3 justify-center">
                    <UButton to="/auth/register" color="primary" size="lg">
                        立即注册
                    </UButton>
                    <UButton to="/auth/login" variant="outline" size="lg">
                        登录
                    </UButton>
                </div>
            </div>

            <!-- 功能预览卡片 -->
            <div class="grid grid-cols-2 lg:grid-cols-4 gap-4">
                <div v-for="feature in features" :key="feature.title" 
                    class="p-4 bg-white dark:bg-neutral-900 rounded-xl border border-neutral-200 dark:border-neutral-800">
                    <div :class="['w-8 h-8 rounded-lg flex items-center justify-center mb-3', feature.bgClass]">
                        <UIcon :name="feature.icon" class="w-4 h-4 text-white" />
                    </div>
                    <h3 class="font-medium text-neutral-900 dark:text-white text-sm">{{ feature.title }}</h3>
                    <p class="text-xs text-neutral-500 dark:text-neutral-400 mt-1">{{ feature.desc }}</p>
                </div>
            </div>
        </template>
    </div>
</template>

<script setup lang="ts">
import { getPageSeoConfig, buildSeoMeta } from "~/config/seo";
import { isTempUser } from "~/utils/userTypes";

// 从数据源获取 SEO 配置
const seoConfig = getPageSeoConfig("home");

definePageMeta({
    title: seoConfig.title,
    description: seoConfig.description,
    layout: "dashboard",
});

useSeoMeta(buildSeoMeta("home"));

const authStore = useAuthStore();
const modal = useModal();

// 认证状态
import type { UserResponse } from "~/types/api";
const displayedUserResponse = computed<UserResponse | null>(() => authStore.user);

// 未读消息数量
const messagesStore = useMessagesStore();
const unreadCount = computed(() => 
  authStore.isReady ? messagesStore.totalUnreadCount : 0
);

// 快速操作配置
const quickActions = computed(() => [
    { 
        to: '/users/nearby', 
        icon: 'i-heroicons-users', 
        label: '附近', 
        bgClass: 'bg-blue-500' 
    },
    { 
        to: '/groups', 
        icon: 'i-heroicons-user-group', 
        label: '群组', 
        bgClass: 'bg-violet-500' 
    },
    { 
        to: '/moments', 
        icon: 'i-heroicons-photo', 
        label: '动态', 
        bgClass: 'bg-rose-500' 
    },
    { 
        to: '/map', 
        icon: 'i-heroicons-map', 
        label: '地图', 
        bgClass: 'bg-emerald-500' 
    },
    { 
        to: '/chat', 
        icon: 'i-heroicons-chat-bubble-left-right', 
        label: '消息', 
        bgClass: 'bg-amber-500',
        badge: unreadCount.value > 0 ? (unreadCount.value > 99 ? '99+' : unreadCount.value.toString()) : undefined,
        badgeColor: 'error'
    },
    { 
        to: '/settings', 
        icon: 'i-heroicons-cog-6-tooth', 
        label: '设置', 
        bgClass: 'bg-neutral-500' 
    },
]);

// 功能特性（未登录用户看到）
const features = [
    { icon: 'i-heroicons-map-pin', title: '位置发现', desc: '发现附近的人', bgClass: 'bg-blue-500' },
    { icon: 'i-heroicons-chat-bubble-left-right', title: '实时聊天', desc: '即时沟通', bgClass: 'bg-amber-500' },
    { icon: 'i-heroicons-photo', title: '分享动态', desc: '记录生活', bgClass: 'bg-rose-500' },
    { icon: 'i-heroicons-shield-check', title: '安全可靠', desc: '保护隐私', bgClass: 'bg-emerald-500' },
];

// 问候消息
const getGreetingMessage = () => {
    const hour = new Date().getHours();
    if (hour < 6) return "夜深了，注意休息";
    if (hour < 12) return "早上好！";
    if (hour < 18) return "下午好！";
    if (hour < 22) return "晚上好！";
    return "夜深了，早点休息";
};

// 活动数据
const refreshing = ref(false);
const activities = ref<any[]>([]);
const activitiesLoading = ref(false);

// 时间线数据转换
const timelineItems = computed(() => {
    return activities.value.map((activity) => ({
        title: activity.content,
        description: activity.time,
        icon: activity.icon,
        avatar: {
            alt: activity.user,
            fallback: activity.user?.charAt(0) || "U",
        },
        color: "primary",
    }));
});

// 获取活动数据
const fetchActivities = async () => {
    try {
        activitiesLoading.value = true;
        // TODO: 接入后端 API
        activities.value = [];
    } catch (error) {
        console.error("获取活动数据失败:", error);
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
    const hasVisited = localStorage.getItem("has-visited-hotspot");

    if (!hasVisited) {
        // 首次访问：显示欢迎弹窗
        setTimeout(() => {
            modal.showWelcomeModal();
        }, 500);
        localStorage.setItem("has-visited-hotspot", "true");
    }

    await fetchActivities();
});

// 快捷键
defineShortcuts({
    h: { usingInput: false, handler: () => navigateTo("/") },
    m: { usingInput: false, handler: () => navigateTo("/map") },
    g: { usingInput: false, handler: () => navigateTo("/groups") },
    c: { usingInput: false, handler: () => navigateTo("/chat") },
    u: { usingInput: false, handler: () => navigateTo("/users/nearby") },
    p: { usingInput: false, handler: () => navigateTo("/moments") },
    r: { usingInput: false, handler: refreshActivities },
});
</script>
