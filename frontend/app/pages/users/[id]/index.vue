<template>
    <div class="mx-auto max-w-7xl px-4 sm:px-6 lg:px-8 py-8 sm:py-10">
        <!-- 面包屑导航 -->
        <div class="mb-6">
            <UBreadcrumb :links="breadcrumbLinks" />
        </div>

        <!-- 加载中 -->
        <div v-if="loading" class="space-y-6">
            <USkeleton class="h-32" />
            <USkeleton class="h-24" />
            <USkeleton class="h-48" />
        </div>

        <!-- 用户不存在 -->
        <UCard v-else-if="!user">
            <EmptyState
                icon="i-heroicons-exclamation-circle"
                title="用户不存在"
                description="该用户可能已被删除或不存在"
                action-text="返回搜索"
                :action-handler="() => $router.push('/users/search')"
            />
        </UCard>

        <!-- 用户详情 -->
        <div v-else class="space-y-6">
            <!-- 用户基本信息 -->
            <UCard>
                <div class="flex flex-col sm:flex-row gap-6">
                    <!-- 头像和基本信息 -->
                    <div class="flex flex-col items-center sm:items-start">
                        <UAvatar :alt="user.nickname" size="2xl" class="mb-4">
                            {{ user.nickname?.charAt(0) || 'U' }}
                        </UAvatar>
                        
                        <!-- 操作按钮 -->
                        <div class="flex gap-2 w-full sm:w-auto">
                            <!-- 私聊按钮 -->
                            <UTooltip :text="`与 ${user.nickname} 开始私聊`" :kbds="['enter']">
                                <UButton
                                    icon="i-heroicons-chat-bubble-left-right"
                                    @click="$router.push(`/chat/users/${user.user_id}`)"
                                    class="flex-1 sm:flex-none"
                                >
                                    私聊
                                </UButton>
                            </UTooltip>
                            
                            <!-- 好友操作按钮 -->
                            <UTooltip
                                v-if="friendshipStatus === 'none'"
                                :text="`向 ${user.nickname} 发送好友请求`"
                            >
                                <UButton
                                    variant="outline"
                                    icon="i-heroicons-user-plus"
                                    :loading="operating"
                                    @click="sendFriendRequest"
                                    class="flex-1 sm:flex-none"
                                >
                                    加好友
                                </UButton>
                            </UTooltip>
                            
                            <UTooltip
                                v-else-if="friendshipStatus === 'pending'"
                                text="好友请求已发送，等待对方确认"
                            >
                                <UButton
                                    variant="outline"
                                    icon="i-heroicons-clock"
                                    disabled
                                    class="flex-1 sm:flex-none"
                                >
                                    已发送
                                </UButton>
                            </UTooltip>
                            
                            <UTooltip
                                v-else-if="friendshipStatus === 'friends'"
                                :text="`从好友列表中移除 ${user.nickname}`"
                            >
                                <UButton
                                    variant="outline"
                                    icon="i-heroicons-user-minus"
                                    :loading="operating"
                                    @click="removeFriend"
                                    class="flex-1 sm:flex-none"
                                >
                                    取消好友
                                </UButton>
                            </UTooltip>
                        </div>
                    </div>
                    
                    <!-- 用户信息 -->
                    <div class="flex-1 space-y-4">
                        <div>
                            <h1 class="text-2xl font-bold text-neutral-900 dark:text-white mb-2">
                                {{ user.nickname || '未设置昵称' }}
                            </h1>
                            <div class="flex flex-wrap gap-4 text-sm text-neutral-600 dark:text-neutral-300">
                                <div class="flex items-center gap-1">
                                    <UIcon name="i-heroicons-identification" />
                                    <span>ID: {{ user.user_id }}</span>
                                </div>
                                <div v-if="user.location_name" class="flex items-center gap-1">
                                    <UIcon name="i-heroicons-map-pin" />
                                    <span>{{ user.location_name }}</span>
                                </div>
                                <div class="flex items-center gap-1">
                                    <UIcon name="i-heroicons-calendar" />
                                    <span>{{ formatJoinTime(user.created_at || new Date().toISOString()) }}</span>
                                </div>
                            </div>
                        </div>
                        
                        <!-- 用户统计 -->
                        <div class="grid grid-cols-3 gap-4 p-4 bg-neutral-50 dark:bg-neutral-800 rounded-lg">
                            <div class="text-center">
                                <div class="text-xl font-semibold text-neutral-900 dark:text-white">
                                    {{ userStats.momentsCount }}
                                </div>
                                <div class="text-sm text-neutral-600 dark:text-neutral-300">
                                    动态
                                </div>
                            </div>
                            <div class="text-center">
                                <div class="text-xl font-semibold text-neutral-900 dark:text-white">
                                    {{ userStats.friendsCount }}
                                </div>
                                <div class="text-sm text-neutral-600 dark:text-neutral-300">
                                    好友
                                </div>
                            </div>
                            <div class="text-center">
                                <div class="text-xl font-semibold text-neutral-900 dark:text-white">
                                    {{ userStats.groupsCount }}
                                </div>
                                <div class="text-sm text-neutral-600 dark:text-neutral-300">
                                    群组
                                </div>
                            </div>
                        </div>
                    </div>
                </div>
            </UCard>

            <!-- 详细信息折叠面板 -->
            <UAccordion :items="userInfoAccordionItems" class="space-y-3">
                <template #moments="{ item, open }">
                    <div class="flex items-center gap-2">
                        <UIcon name="i-heroicons-photo" class="w-5 h-5" />
                        <span class="font-medium">TA的动态</span>
                        <UBadge v-if="userStats.momentsCount > 0" size="xs" variant="soft">
                            {{ userStats.momentsCount }}
                        </UBadge>
                    </div>
                </template>
                
                <template #moments-content>
                    <div v-if="loadingMoments" class="space-y-4">
                        <USkeleton class="h-24" v-for="i in 3" :key="i" />
                    </div>
                    
                    <div v-else-if="userMoments.length > 0" class="space-y-4">
                        <div 
                            v-for="moment in userMoments" 
                            :key="moment.moment_id"
                            class="p-4 border border-neutral-200 dark:border-neutral-700 rounded-lg hover:bg-neutral-50 dark:hover:bg-neutral-800 cursor-pointer transition-colors"
                            @click="$router.push(`/moments/${moment.moment_id}`)"
                        >
                            <div class="flex gap-3">
                                <UAvatar :alt="moment.nickname" size="sm">
                                    {{ moment.nickname?.charAt(0) }}
                                </UAvatar>
                                <div class="flex-1 min-w-0">
                                    <div class="flex items-center gap-2 mb-1">
                                        <span class="font-medium text-sm">{{ moment.nickname }}</span>
                                        <span class="text-xs text-neutral-500">{{ formatTime(moment.created_at) }}</span>
                                    </div>
                                    <p class="text-sm text-neutral-700 dark:text-neutral-300 line-clamp-2">
                                        {{ moment.description }}
                                    </p>
                                    <div class="flex items-center gap-4 mt-2 text-xs text-neutral-500">
                                        <span class="flex items-center gap-1">
                                            <UIcon name="i-heroicons-heart" />
                                            {{ moment.likes_count }}
                                        </span>
                                        <span v-if="moment.location_name" class="flex items-center gap-1">
                                            <UIcon name="i-heroicons-map-pin" />
                                            {{ moment.location_name }}
                                        </span>
                                    </div>
                                </div>
                            </div>
                        </div>
                        
                        <!-- 加载更多 -->
                        <div v-if="hasMoreMoments" class="text-center">
                            <UButton 
                                variant="ghost" 
                                :loading="loadingMoreMoments"
                                @click="loadMoreMoments"
                            >
                                加载更多
                            </UButton>
                        </div>
                    </div>
                    
                    <div v-else class="py-8 text-center text-neutral-500">
                        TA还没有发布任何动态
                    </div>
                </template>
                
                <template #friends="{ item, open }">
                    <div class="flex items-center gap-2">
                        <UIcon name="i-heroicons-users" class="w-5 h-5" />
                        <span class="font-medium">好友列表</span>
                        <UBadge v-if="userStats.friendsCount > 0" size="xs" variant="soft">
                            {{ userStats.friendsCount }}
                        </UBadge>
                    </div>
                </template>
                
                <template #friends-content>
                    <div class="py-8 text-center text-neutral-500">
                        好友列表功能开发中...
                    </div>
                </template>
                
                <template #groups="{ item, open }">
                    <div class="flex items-center gap-2">
                        <UIcon name="i-heroicons-user-group" class="w-5 h-5" />
                        <span class="font-medium">加入的群组</span>
                        <UBadge v-if="userStats.groupsCount > 0" size="xs" variant="soft">
                            {{ userStats.groupsCount }}
                        </UBadge>
                    </div>
                </template>
                
                <template #groups-content>
                    <div class="py-8 text-center text-neutral-500">
                        群组列表功能开发中...
                    </div>
                </template>
            </UAccordion>
            <!-- 删除好友确认现在通过 useOverlay 管理 -->
        </div>
    </div>
</template>

<script setup lang="ts">
import { buildUserUrl, USER_ENDPOINTS } from '~/config/api';
import type { UserInfo, MomentInfo } from '~/types';

definePageMeta({ 
    title: '用户详情', 
    requiresAuth: true 
});

const route = useRoute();
const userService = useUserService();
const friendService = useFriendService();
const momentsStore = useMomentsStore();
const toast = useSmartToast();
const authStore = useAuthStore();
const api = useApi();
const { openConfirmModal } = useModals();

const userId = route.params.id as string;

// 状态变量
const loading = ref(true);
const operating = ref(false);
const user = ref<UserInfo | null>(null);
const userMoments = ref<MomentInfo[]>([]);
const loadingMoments = ref(false);
const loadingMoreMoments = ref(false);
const hasMoreMoments = ref(false);

// 好友状态
const friendshipStatus = ref<'none' | 'pending' | 'friends'>('none');

// 用户统计
const userStats = ref({
    momentsCount: 0,
    friendsCount: 0,
    groupsCount: 0
});

// 面包屑导航
const breadcrumbLinks = computed(() => [
    {
        label: '首页',
        icon: 'i-heroicons-home',
        to: '/'
    },
    {
        label: '用户搜索',
        icon: 'i-heroicons-magnifying-glass',
        to: '/users/search'
    },
    {
        label: user.value ? user.value.nickname || `用户${userId}` : '用户详情',
        icon: 'i-heroicons-user'
    }
]);

// UAccordion 配置
const userInfoAccordionItems = computed(() => [
    {
        label: 'TA的动态',
        icon: 'i-heroicons-photo',
        slot: 'moments',
        defaultOpen: true,
        content: 'moments-content'
    },
    {
        label: '好友列表',
        icon: 'i-heroicons-users', 
        slot: 'friends',
        defaultOpen: false,
        content: 'friends-content'
    },
    {
        label: '加入的群组',
        icon: 'i-heroicons-user-group',
        slot: 'groups', 
        defaultOpen: false,
        content: 'groups-content'
    }
]);

// 生命周期
onMounted(async () => {
    await loadUserProfile();
});

// 加载用户资料
async function loadUserProfile() {
    loading.value = true;
    try {
        // 暂无 getById，沿用搜索以最小改动取数
        try {
            const response = await userService.searchUsers(userId);
            const foundUser = (response as any)?.users?.find((u: any) => u.user_id.toString() === userId);
            if (foundUser) {
                user.value = foundUser;
                friendshipStatus.value = foundUser.friendship_status || 'none';
            }
        } catch (error) {
            if (import.meta.dev) {
                console.warn('Failed to find user in store:', error);
            }
        }
        
        await loadUserStats();
        await loadUserMoments();
        
    } catch (e: any) {
        toast.add({
            title: '加载失败',
            description: e?.message,
            color: 'error'
        });
    } finally {
        loading.value = false;
    }
}

// 加载用户统计信息（TODO: 需要后端实现统计API或通过现有API计算）
async function loadUserStats() {
    try {
        userStats.value = { momentsCount: 0, friendsCount: 0, groupsCount: 0 };
        
        // 可能的实现方式：
        // 1. 通过 moments/history API 获取用户动态数量
        // 2. 通过 friends API 获取好友数量（当好友功能实现时）
        // 3. 通过 groups API 获取用户参与的群组数量
        
        if (import.meta.dev) {
            console.log('用户统计功能暂未实现，使用默认值');
        }
    } catch (e: any) {
        console.error('加载用户统计失败:', e);
        toast.add({ title: '加载失败', description: e?.message || '无法加载用户统计信息', color: 'error' });
    }
}

// 加载用户动态（使用生活瞬间历史API - 符合 hotspot.md 7.5.1）
async function loadUserMoments() {
    loadingMoments.value = true;
    try {
        // 使用生活瞬间服务获取用户历史
        const momentService = useMomentService();
        const resp = await momentService.getUserMomentsHistory(userId, '0');
        
        userMoments.value = resp?.items || [];
        hasMoreMoments.value = resp?.has_more || false;
    } catch (e: any) {
        console.error('加载用户动态失败:', e);
        toast.add({ title: '加载失败', description: e?.message || '无法加载用户动态', color: 'error' });
    } finally {
        loadingMoments.value = false;
    }
}

// 加载更多动态
async function loadMoreMoments() {
    if (loadingMoreMoments.value || !hasMoreMoments.value) return;
    
    loadingMoreMoments.value = true;
    try {
        const lastMoment = userMoments.value[userMoments.value.length - 1];
        const cursor = lastMoment?.moment_id || 0;
        
        // 使用生活瞬间服务获取更多历史
        const momentService = useMomentService();
        const resp = await momentService.getUserMomentsHistory(userId, cursor.toString());
        
        const newMoments = resp?.items || [];
        userMoments.value.push(...newMoments);
        hasMoreMoments.value = resp?.has_more || false;
    } catch (e: any) {
        toast.add({ title: '加载失败', description: e?.message || '无法加载更多动态', color: 'error' });
    } finally {
        loadingMoreMoments.value = false;
    }
}

// 好友操作
async function sendFriendRequest() {
    if (operating.value) return;
    
    operating.value = true;
    try {
        await friendService.sendFriendRequest(userId);
        friendshipStatus.value = 'pending';
        toast.add({
            title: '好友请求已发送',
            color: 'success'
        });
    } catch (e: any) {
        toast.add({
            title: '操作失败',
            description: e?.message || '发送好友请求失败',
            color: 'error'
        });
    } finally {
        operating.value = false;
    }
}

async function removeFriend() {
    if (operating.value) return;
    
    const confirmed = await openConfirmModal({
        title: '确认删除好友',
        message: `确定要删除好友"${user.value?.nickname || '该用户'}"吗？`,
        confirmText: '确认删除',
        cancelText: '取消'
    });
    
    if (!confirmed) return;
    
    operating.value = true;
    try {
        await friendService.removeFriend(userId);
        friendshipStatus.value = 'none';
        toast.add({
            title: '已删除好友',
            color: 'success'
        });
    } catch (e: any) {
        toast.add({
            title: '操作失败',
            description: e?.message || '删除好友失败',
            color: 'error'
        });
    } finally {
        operating.value = false;
    }
}

// 工具函数
function formatJoinTime(dateString: string) {
    const date = new Date(dateString);
    return `${date.getFullYear()}年${date.getMonth() + 1}月加入`;
}

function formatTime(dateString: string) {
    const now = new Date();
    const date = new Date(dateString);
    const diff = now.getTime() - date.getTime();
    
    const minutes = Math.floor(diff / 60000);
    const hours = Math.floor(diff / 3600000);
    const days = Math.floor(diff / 86400000);
    
    if (minutes < 60) return `${minutes}分钟前`;
    if (hours < 24) return `${hours}小时前`;
    if (days < 30) return `${days}天前`;
    
    return date.toLocaleDateString();
}
</script>
