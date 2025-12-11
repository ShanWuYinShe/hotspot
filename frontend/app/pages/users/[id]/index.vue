<template>
    <div>
        <!-- 页面标题和操作 -->
        <UPageHeader
            v-if="user"
            title="用户资料"
            class="mb-6"
        >
            <template #actions>
                <UButton
                    variant="ghost"
                    color="neutral"
                    icon="i-heroicons-share"
                    @click="shareUser"
                />
                <UDropdownMenu :items="userActions">
                    <UButton
                        variant="ghost"
                        color="neutral"
                        icon="i-heroicons-ellipsis-horizontal"
                    />
                </UDropdownMenu>
            </template>
        </UPageHeader>

        <!-- 加载中 -->
        <div v-if="loading">
            <div class="grid grid-cols-1 lg:grid-cols-3 gap-8">
                <div class="lg:col-span-1">
                    <USkeleton class="h-80 rounded-2xl" />
                </div>
                <div class="lg:col-span-2 space-y-6">
                    <USkeleton class="h-48 rounded-2xl" />
                    <USkeleton class="h-64 rounded-2xl" />
                </div>
            </div>
        </div>

        <!-- 用户不存在 -->
        <div v-else-if="!user" class="max-w-md mx-auto py-16 text-center">
            <div class="p-4 bg-interactive rounded-full w-20 h-20 mx-auto mb-6 flex items-center justify-center">
                <UIcon name="i-heroicons-user-slash" class="w-10 h-10 text-muted" />
            </div>
            <h2 class="text-xl font-bold text-primary-content mb-2">用户不存在</h2>
            <p class="text-secondary-content mb-6">该用户可能已被删除或不存在</p>
        </div>

        <!-- 用户详情 -->
        <div v-else class="grid grid-cols-1 lg:grid-cols-3 gap-8">
                <!-- 左侧：用户信息卡片 -->
                <div class="lg:col-span-1 space-y-6">
                    <!-- 主要信息卡片 -->
                    <div class="bg-card rounded-2xl shadow-sm border border-default overflow-hidden">
                        <!-- 封面背景 -->
                        <div class="h-24 bg-linear-to-br from-primary-400 to-primary-600" />
                        
                        <!-- 头像和基本信息 -->
                        <div class="px-6 pb-6">
                            <div class="relative -mt-12 mb-4">
                                <UAvatar 
                                    :src="user.avatar_url" 
                                    :alt="user.nickname"
                                    size="3xl"
                                    class="ring-4 ring-default shadow-lg"
                                >
                                    {{ user.nickname?.charAt(0) || 'U' }}
                                </UAvatar>
                                
                                <!-- 在线状态指示器 -->
                                <span 
                                    v-if="userStatus.is_online"
                                    class="absolute bottom-1 right-1 w-5 h-5 bg-icon-success border-3 border-white-dark rounded-full"
                                />
                            </div>
                            
                            <h1 class="text-xl font-bold text-primary-content">
                                {{ user.nickname || '未设置昵称' }}
                            </h1>
                            <p class="text-sm text-secondary-content mt-1">ID: {{ user.user_id }}</p>
                            
                            <!-- 个人简介 -->
                            <p v-if="user.bio" class="text-secondary-strong mt-3 text-sm">
                                {{ user.bio }}
                            </p>
                            
                            <!-- 位置和距离 -->
                            <div v-if="user.geo" class="flex items-center gap-2 mt-3 text-sm text-secondary-content">
                                <UIcon name="i-heroicons-map-pin" class="icon-sm" />
                                <span>{{ user.geo.location_name }}</span>
                                <span v-if="user.geo.distance" class="text-primary font-medium">
                                    · {{ formatDistance(user.geo.distance) }}
                                </span>
                            </div>
                            
                            <!-- 操作按钮组 -->
                            <div class="flex gap-2 mt-6">
                                <UButton
                                    class="flex-1"
                                    icon="i-heroicons-chat-bubble-left-right"
                                    @click="navigateTo(`/chat/users/${user.user_id}`)"
                                >
                                    私聊
                                </UButton>
                                
                                <UButton
                                    v-if="isMutualFollow"
                                    variant="soft"
                                    color="success"
                                    icon="i-heroicons-user-check"
                                    :loading="operating"
                                    @click="handleUnfollow"
                                >
                                    好友
                                </UButton>
                                <UButton
                                    v-else-if="isFollowing"
                                    variant="outline"
                                    icon="i-heroicons-heart-solid"
                                    :loading="operating"
                                    @click="handleUnfollow"
                                >
                                    已关注
                                </UButton>
                                <UButton
                                    v-else
                                    variant="outline"
                                    icon="i-heroicons-heart"
                                    :loading="operating"
                                    @click="handleFollow"
                                >
                                    关注
                                </UButton>
                            </div>
                        </div>
                    </div>
                    
                    <!-- 统计卡片 -->
                    <div class="bg-card rounded-2xl shadow-sm border border-default p-6">
                        <h3 class="font-semibold text-primary-content mb-4">数据统计</h3>
                        <div class="grid grid-cols-3 gap-4">
                            <div class="text-center p-3 bg-muted rounded-xl">
                                <p class="text-2xl font-bold text-primary-content">
                                    {{ userStats.momentsCount }}
                                </p>
                                <p class="text-xs text-secondary-content mt-1">动态</p>
                            </div>
                            <div class="text-center p-3 bg-muted rounded-xl">
                                <p class="text-2xl font-bold text-primary-content">
                                    {{ userStats.friendsCount }}
                                </p>
                                <p class="text-xs text-secondary-content mt-1">好友</p>
                            </div>
                            <div class="text-center p-3 bg-muted rounded-xl">
                                <p class="text-2xl font-bold text-primary-content">
                                    {{ userStats.groupsCount }}
                                </p>
                                <p class="text-xs text-secondary-content mt-1">群组</p>
                            </div>
                        </div>
                    </div>
                    
                    <!-- 在线状态卡片 -->
                    <div class="bg-card rounded-2xl shadow-sm border border-default p-6">
                        <h3 class="font-semibold text-primary-content mb-4">在线状态</h3>
                        <div class="flex items-center gap-3">
                            <div :class="[
                                'w-3 h-3 rounded-full',
                                userStatus.is_online ? 'bg-icon-success' : 'bg-neutral-soft'
                            ]" />
                            <div>
                                <p class="text-sm font-medium text-primary-content">
                                    {{ userStatus.is_online ? '在线' : '离线' }}
                                </p>
                                <p v-if="!userStatus.is_online && userStatus.last_online" class="text-xs text-secondary-content">
                                    最后在线：{{ formatTime(userStatus.last_online) }}
                                </p>
                            </div>
                        </div>
                    </div>
                </div>

                <!-- 右侧：动态和更多内容 -->
                <div class="lg:col-span-2 space-y-6">
                    <!-- 动态列表 -->
                    <div class="bg-card rounded-2xl shadow-sm border border-default overflow-hidden">
                        <div class="flex items-center justify-between px-6 py-4 border-b border-default">
                            <div class="flex items-center gap-3">
                                <div class="p-2 bg-primary-subtle rounded-lg">
                                    <UIcon name="i-heroicons-photo" class="w-5 h-5 text-primary" />
                                </div>
                                <div>
                                    <h3 class="font-semibold text-primary-content">TA的动态</h3>
                                    <p class="text-sm text-secondary-content">{{ userStats.momentsCount }} 条动态</p>
                                </div>
                            </div>
                        </div>
                        
                        <!-- 加载中 -->
                        <div v-if="loadingMoments" class="p-6 space-y-4">
                            <USkeleton class="h-24" v-for="i in 3" :key="i" />
                        </div>
                        
                        <!-- 动态列表 -->
                        <div v-else-if="userMoments.length > 0">
                            <div 
                                v-for="moment in userMoments" 
                                :key="moment.id"
                                class="p-6 border-b border-subtle/50 last:border-b-0 hover-bg-muted cursor-pointer transition-colors"
                                @click="$router.push(`/moments/${moment.id}`)"
                            >
                                <div class="flex gap-4">
                                    <!-- 图片预览（如果有） -->
                                    <div
                                        v-if="moment.images?.length"
                                        class="w-20 h-20 rounded-xl overflow-hidden shrink-0 bg-interactive"
                                    >
                                        <NuxtImg
                                            :src="moment.images[0]"
                                            :alt="moment.description"
                                            class="w-full h-full object-cover"
                                        />
                                    </div>
                                    
                                    <div class="flex-1 min-w-0">
                                        <p class="text-primary-content line-clamp-2">
                                            {{ moment.description }}
                                        </p>
                                        
                                        <!-- 标签 -->
                                        <div v-if="moment.tags?.length" class="flex flex-wrap gap-1 mt-2">
                                            <UBadge 
                                                v-for="tag in moment.tags.slice(0, 3)" 
                                                :key="tag"
                                                variant="soft"
                                                color="neutral"
                                                size="xs"
                                            >
                                                #{{ tag }}
                                            </UBadge>
                                        </div>
                                        
                                        <!-- 元信息 -->
                                        <div class="flex items-center gap-4 mt-3 text-xs text-secondary-content">
                                            <span class="flex items-center gap-1">
                                                <UIcon name="i-heroicons-heart" class="w-3.5 h-3.5" />
                                                {{ moment.likes_count }}
                                            </span>
                                            <span class="flex items-center gap-1">
                                                <UIcon name="i-heroicons-chat-bubble-left" class="w-3.5 h-3.5" />
                                                {{ moment.comments_count }}
                                            </span>
                                            <span v-if="moment.geo?.location_name" class="flex items-center gap-1">
                                                <UIcon name="i-heroicons-map-pin" class="w-3.5 h-3.5" />
                                                {{ moment.geo.location_name }}
                                            </span>
                                            <span>{{ formatTime(moment.created_at) }}</span>
                                        </div>
                                    </div>
                                </div>
                            </div>
                            
                            <!-- 加载更多 -->
                            <div v-if="hasMoreMoments" class="p-4 text-center">
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
                        <div v-else class="py-16 text-center">
                            <div class="p-4 bg-interactive rounded-full w-16 h-16 mx-auto mb-4 flex items-center justify-center">
                                <UIcon name="i-heroicons-photo" class="w-8 h-8 text-muted" />
                            </div>
                            <p class="text-secondary-content">TA还没有发布任何动态</p>
                        </div>
                    </div>
                    
                    <!-- 好友列表卡片 -->
                    <div class="bg-card rounded-2xl shadow-sm border border-default overflow-hidden">
                        <div class="flex items-center justify-between px-6 py-4 border-b border-default">
                            <div class="flex items-center gap-3">
                                <div class="p-2 bg-blue-subtle rounded-lg">
                                    <UIcon name="i-heroicons-users" class="w-5 h-5 text-blue" />
                                </div>
                                <div>
                                    <h3 class="font-semibold text-primary-content">好友列表</h3>
                                    <p class="text-sm text-secondary-content">{{ userStats.friendsCount }} 位好友</p>
                                </div>
                            </div>
                        </div>
                        
                        <div class="py-12 text-center">
                            <p class="text-secondary-content">好友列表功能开发中...</p>
                        </div>
                    </div>
                    
                    <!-- 群组列表卡片 -->
                    <div class="bg-card rounded-2xl shadow-sm border border-default overflow-hidden">
                        <div class="flex items-center justify-between px-6 py-4 border-b border-default">
                            <div class="flex items-center gap-3">
                                <div class="p-2 bg-purple-subtle rounded-lg">
                                    <UIcon name="i-heroicons-user-group" class="w-5 h-5 text-purple" />
                                </div>
                                <div>
                                    <h3 class="font-semibold text-primary-content">加入的群组</h3>
                                    <p class="text-sm text-secondary-content">{{ userStats.groupsCount }} 个群组</p>
                                </div>
                            </div>
                        </div>
                        
                        <div class="py-12 text-center">
                            <p class="text-secondary-content">群组列表功能开发中...</p>
                        </div>
                    </div>
                </div>
            </div>
        </div>
</template>

<script setup lang="ts">
import { useUsersService } from '~/services/users.service';
import { useMomentsService } from '~/services/moments.service';
import { getErrorMessage } from '~/utils/error';
import type { MomentSearchItem, UserSearchItem } from '~/types/api';

definePageMeta({ 
    title: '用户详情',
    layout: 'dashboard'
});

const route = useRoute();
const router = useRouter();
const userService = useUsersService();
const momentsService = useMomentsService();
const geolocation = useGeolocation();
const authStore = useAuthStore();
const toast = useToast()
const { formatTime } = useFormatters();
const modal = useModal();

const userId = Number(route.params.id);

// 如果是自己的用户ID，跳转到个人资料页面
onBeforeMount(() => {
    if (authStore.isAuthenticated && authStore.user?.user_id === userId) {
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
const userStats = ref({
    momentsCount: 0,
    friendsCount: 0,
    groupsCount: 0
});

// 格式化距离
const formatDistance = (meters: number) => {
    if (meters < 1000) return `${Math.round(meters)}米`
    return `${(meters / 1000).toFixed(1)}公里`
}

// 用户操作菜单
const userActions = computed(() => {
    if (!user.value) return [];
    
    return [
        [{
            label: isFollowing.value ? '取消关注' : '关注用户',
            icon: isFollowing.value ? 'i-heroicons-heart-slash' : 'i-heroicons-heart',
            click: () => isFollowing.value ? handleUnfollow() : handleFollow()
        }],
        [{
            label: '举报用户',
            icon: 'i-heroicons-shield-exclamation',
            click: () => reportUser()
        },
        {
            label: '拉黑用户',
            icon: 'i-heroicons-no-symbol',
            click: () => blockUser()
        }]
    ];
});

// 加载用户资料
onMounted(async () => {
    await loadUserProfile();
});

// 监听认证状态变化获取关系
watch(
    () => [authStore.isAuthenticated, user.value] as const,
    async ([isAuthenticated, userData]) => {
        if (isAuthenticated && userData) {
            try {
                const relationship = await userService.getRelationship({ id: userId });
                isMutualFollow.value = relationship.is_mutual_follow ?? false;
                isFollowing.value = relationship.is_following ?? false;
            } catch {
                // 忽略
            }
        }
    },
    { immediate: true }
);

async function loadUserProfile() {
    loading.value = true;
    try {
        const coords = await geolocation.getCoordinates();
        
        const response = await userService.openTheDoor({ 
            id: userId,
            latitude: coords.latitude,
            longitude: coords.longitude
        });
        user.value = response;
        
        // 在线状态已包含在 user 中，无需单独获取
        userStatus.value = {
            is_online: response.is_online ?? false,
            last_online: response.last_online ?? ''
        };
        
        await Promise.all([
            loadUserStats(),
            loadUserMoments()
        ]);
        
    } catch (e: unknown) {
        toast.add({
            title: '加载失败',
            description: getErrorMessage(e),
            color: 'error'
        });
    } finally {
        loading.value = false;
    }
}

async function loadUserStats() {
    try {
        const stats = await userService.getStats({ id: userId });
        userStats.value = {
            momentsCount: stats.moments_count,
            friendsCount: stats.friends_count,
            groupsCount: stats.groups_count
        };
    } catch {
        userStats.value = { momentsCount: 0, friendsCount: 0, groupsCount: 0 };
    }
}

async function loadUserMoments() {
    loadingMoments.value = true;
    try {
        const coords = await geolocation.getCoordinates();
        const resp = await momentsService.getUserMoments({
            id: userId,
            latitude: coords.latitude,
            longitude: coords.longitude,
            limit: 20
        });
        
        userMoments.value = resp?.items || [];
        hasMoreMoments.value = resp?.pagination.has_more || false;
    } catch {
        // 忽略
    } finally {
        loadingMoments.value = false;
    }
}

async function loadMoreMoments() {
    if (loadingMoreMoments.value || !hasMoreMoments.value) return;
    
    loadingMoreMoments.value = true;
    try {
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
    } catch {
        toast.add({ title: '加载失败', color: 'error' });
    } finally {
        loadingMoreMoments.value = false;
    }
}

async function handleFollow() {
    if (operating.value) return;
    
    operating.value = true;
    try {
        await userService.followUser({ id: userId });
        isFollowing.value = true;
        toast.add({
            title: '关注成功',
            description: `已关注 ${user.value?.nickname}`,
            color: 'success'
        });
        await loadUserProfile();
    } catch (e: unknown) {
        toast.add({
            title: '操作失败',
            description: getErrorMessage(e),
            color: 'error'
        });
    } finally {
        operating.value = false;
    }
}

async function handleUnfollow() {
    if (operating.value) return;
    
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
        operating.value = true;
        try {
            await userService.unfollowUser({ id: userId });
            isFollowing.value = false;
            isMutualFollow.value = false;
            toast.add({ title: '已取消关注', color: 'success' });
        } catch (e: unknown) {
            toast.add({
                title: '操作失败',
                description: getErrorMessage(e),
                color: 'error'
            });
        } finally {
            operating.value = false;
        }
    }
}

async function shareUser() {
    if (!user.value) return;
    
    try {
        if (navigator.share) {
            await navigator.share({
                title: `${user.value.nickname} 的个人资料`,
                text: `来看看 ${user.value.nickname} 在 HotSpot 上的精彩内容！`,
                url: window.location.href
            });
        } else {
            await navigator.clipboard.writeText(window.location.href);
            toast.add({
                title: '链接已复制',
                description: '用户链接已复制到剪贴板',
                color: 'success'
            });
        }
    } catch {
        // 忽略
    }
}

async function reportUser() {
    if (!user.value) return;
    
    await modal.showReport('user', String(user.value.user_id));
    toast.add({
        title: '举报已提交',
        description: '我们会尽快处理您的举报',
        color: 'success'
    });
}

async function blockUser() {
    if (!user.value) return;
    
    const confirmed = await modal.showConfirm({
        title: '拉黑用户',
        message: `确定要拉黑 ${user.value.nickname} 吗？拉黑后您将无法看到对方的任何内容，对方也无法与您互动。`,
        confirmText: '确认拉黑',
        cancelText: '取消',
        confirmColor: 'error'
    });
    
    if (!confirmed) return;
    
    try {
        await userService.blockUser({ id: user.value.user_id });
        toast.add({
            title: '拉黑成功',
            description: `已拉黑 ${user.value.nickname}`,
            color: 'success'
        });
        router.back();
    } catch (e: unknown) {
        toast.add({
            title: '拉黑失败',
            description: getErrorMessage(e),
            color: 'error'
        });
    }
}
</script>
