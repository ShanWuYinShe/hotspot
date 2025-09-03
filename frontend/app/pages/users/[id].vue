<template>
    <div class="mx-auto max-w-4xl px-4 sm:px-6 lg:px-8 py-14 sm:py-16">
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
                    <!-- 头像区域 -->
                    <div class="flex flex-col items-center sm:items-start">
                        <UAvatar 
                            :alt="user.nickname" 
                            size="2xl"
                            :src="user.avatar_url"
                            class="mb-4"
                        >
                            {{ user.nickname?.charAt(0) }}
                        </UAvatar>
                        
                        <!-- 好友状态徽章 -->
                        <UChip 
                            v-if="friendshipStatus !== 'none'"
                            :variant="friendshipStatusVariant"
                            :color="friendshipStatusColor"
                            size="sm"
                        >
                            {{ friendshipStatusText }}
                        </UChip>
                    </div>
                    
                    <!-- 用户信息 -->
                    <div class="flex-1">
                        <div class="flex flex-col sm:flex-row sm:items-center sm:justify-between gap-4">
                            <div>
                                <h1 class="text-3xl font-bold text-neutral-900 dark:text-white mb-2">
                                    {{ user.nickname }}
                                </h1>
                                <div class="space-y-2 text-neutral-600 dark:text-neutral-400">
                                    <div v-if="user.bio" class="text-lg">{{ user.bio }}</div>
                                    <div class="flex items-center gap-4 text-sm">
                                        <span v-if="user.location_name" class="flex items-center gap-1">
                                            <UIcon name="i-heroicons-map-pin" class="w-4 h-4" />
                                            {{ user.location_name }}
                                        </span>
                                        <span v-if="user.distance !== undefined" class="flex items-center gap-1">
                                            <UIcon name="i-heroicons-signal" class="w-4 h-4" />
                                            {{ formatDistance(user.distance) }}
                                        </span>
                                        <span class="flex items-center gap-1">
                                            <UIcon name="i-heroicons-calendar" class="w-4 h-4" />
                                            {{ formatJoinTime(user.created_at) }}加入
                                        </span>
                                    </div>
                                </div>
                            </div>
                            
                            <!-- 操作按钮 -->
                            <div class="flex gap-3">
                                <UButton
                                    v-if="friendshipStatus === 'none'"
                                    @click="sendFriendRequest"
                                    :loading="operating"
                                >
                                    <UIcon name="i-heroicons-user-plus" class="w-4 h-4 mr-2" />
                                    添加好友
                                </UButton>
                                
                                <UButton
                                    v-else-if="friendshipStatus === 'pending_sent'"
                                    variant="ghost"
                                    disabled
                                >
                                    <UIcon name="i-heroicons-clock" class="w-4 h-4 mr-2" />
                                    已发送请求
                                </UButton>
                                
                                <div v-else-if="friendshipStatus === 'pending_received'" class="flex gap-2">
                                    <UButton
                                        @click="acceptFriendRequest"
                                        :loading="operating"
                                    >
                                        <UIcon name="i-heroicons-check" class="w-4 h-4 mr-2" />
                                        接受
                                    </UButton>
                                    <UButton
                                        variant="outline"
                                        @click="rejectFriendRequest"
                                        :loading="operating"
                                    >
                                        <UIcon name="i-heroicons-x-mark" class="w-4 h-4 mr-2" />
                                        拒绝
                                    </UButton>
                                </div>
                                
                                <UButton
                                    v-else-if="friendshipStatus === 'accepted'"
                                    @click="$router.push(`/chat/user/${user.user_id}`)"
                                >
                                    <UIcon name="i-heroicons-chat-bubble-left-right" class="w-4 h-4 mr-2" />
                                    发消息
                                </UButton>
                                
                                <!-- 更多操作菜单 -->
                                <UDropdownMenu :items="getActionItems()">
                                    <UButton
                                        icon="i-heroicons-ellipsis-horizontal"
                                        variant="ghost"
                                    />
                                </UDropdownMenu>
                            </div>
                        </div>
                        
                        <!-- 统计信息 -->
                        <div class="mt-6 grid grid-cols-3 gap-6">
                            <div class="text-center">
                                <div class="text-2xl font-bold text-neutral-900 dark:text-white">
                                    {{ userStats.momentsCount }}
                                </div>
                                <div class="text-sm text-neutral-600 dark:text-neutral-400">
                                    动态
                                </div>
                            </div>
                            <div class="text-center">
                                <div class="text-2xl font-bold text-neutral-900 dark:text-white">
                                    {{ userStats.friendsCount }}
                                </div>
                                <div class="text-sm text-neutral-600 dark:text-neutral-400">
                                    好友
                                </div>
                            </div>
                            <div class="text-center">
                                <div class="text-2xl font-bold text-neutral-900 dark:text-white">
                                    {{ userStats.groupsCount }}
                                </div>
                                <div class="text-sm text-neutral-600 dark:text-neutral-400">
                                    群组
                                </div>
                            </div>
                        </div>
                    </div>
                </div>
            </UCard>
            
            <!-- 用户详细信息 -->
            <UAccordion :items="userInfoAccordionItems" multiple>
                <template #moments="{ item, open }">
                    <div class="flex items-center justify-between w-full">
                        <div class="flex items-center gap-2">
                            <UIcon name="i-heroicons-photo" class="w-5 h-5" />
                            <span class="font-medium">TA的动态</span>
                            <UBadge v-if="userStats.momentsCount > 0" size="xs" variant="soft">
                                {{ userStats.momentsCount }}
                            </UBadge>
                        </div>
                        <UButton
                            size="sm"
                            variant="ghost"
                            @click="loadUserMoments"
                            :loading="loadingMoments"
                        >
                            <UIcon name="i-heroicons-arrow-path" class="w-4 h-4" />
                        </UButton>
                    </div>
                </template>
                
                <template #moments-content>
                
                <!-- 动态加载中 -->
                <div v-if="loadingMoments" class="space-y-4">
                    <USkeleton class="h-24" />
                    <USkeleton class="h-24" />
                    <USkeleton class="h-24" />
                </div>
                
                <!-- 动态列表 -->
                <div v-else-if="userMoments.length > 0" class="space-y-4">
                    <div
                        v-for="moment in userMoments"
                        :key="moment.moment_id"
                        class="p-4 border border-neutral-200 dark:border-neutral-700 rounded-lg cursor-pointer hover:bg-neutral-50 dark:hover:bg-neutral-900 transition-colors"
                        @click="navigateTo(`/moments/${moment.moment_id}`)"
                    >
                        <div class="flex items-start gap-3 mb-2">
                            <UAvatar :alt="moment.nickname" size="sm">
                                {{ moment.nickname?.charAt(0) }}
                            </UAvatar>
                            <div class="flex-1">
                                <div class="flex items-center gap-2 mb-1">
                                    <span class="font-medium text-sm">{{ moment.nickname }}</span>
                                    <span class="text-xs text-neutral-500">{{ formatTime(moment.created_at) }}</span>
                                </div>
                                <div class="text-xs text-neutral-500 flex items-center gap-1">
                                    <UIcon name="i-heroicons-map-pin" class="w-3 h-3" />
                                    <span v-if="moment.location_name">{{ moment.location_name }}</span>
                                </div>
                            </div>
                        </div>
                        
                        <p class="text-neutral-700 dark:text-neutral-300 mb-3 line-clamp-3">
                            {{ moment.description }}
                        </p>
                        
                        <div class="flex items-center gap-4 text-xs text-neutral-500">
                            <span class="flex items-center gap-1">
                                <UIcon name="i-heroicons-heart" class="w-3 h-3" />
                                {{ moment.likes_count }}
                            </span>
                            <span class="flex items-center gap-1">
                                <UIcon name="i-heroicons-chat-bubble-left-right" class="w-3 h-3" />
                                {{ moment.comments_count }}
                            </span>
                        </div>
                    </div>
                    
                    <!-- 加载更多动态 -->
                    <div v-if="hasMoreMoments" class="text-center">
                        <UButton
                            variant="outline"
                            :loading="loadingMoreMoments"
                            @click="loadMoreMoments"
                        >
                            加载更多动态
                        </UButton>
                    </div>
                </div>
                
                    <!-- 暂无动态 -->
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
            <!-- 删除好友确认 -->
            <UModal v-model:open="showRemoveConfirm" title="确认删除好友">
                <template #body>
                    <p class="text-neutral-600 dark:text-neutral-300">确定要删除这个好友吗？</p>
                </template>
                <template #footer>
                    <div class="flex justify-end gap-2">
                        <UButton variant="ghost" @click="showRemoveConfirm = false">取消</UButton>
                        <UButton color="error" :loading="operating" @click="doRemoveFriend">确认删除</UButton>
                    </div>
                </template>
            </UModal>
        </div>
    </div>
</template>

<script setup lang="ts">
import { useUserService } from '~/services/user.service';
import { useFriendService } from '~/services/friend.service';
import { useApi } from '~/composables/useApi';
import { buildUserUrl, USER_ENDPOINTS, buildFriendUrl, FRIEND_ENDPOINTS } from '~/config/api';
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

const userId = route.params.id as string;

// 状态变量
const loading = ref(true);
const operating = ref(false);
const user = ref<UserInfo | null>(null);
const userMoments = ref<MomentInfo[]>([]);
const loadingMoments = ref(false);
const loadingMoreMoments = ref(false);
const hasMoreMoments = ref(false);
const momentsCursor = ref<number | undefined>(undefined);

// 用户统计信息
const userStats = ref({
    momentsCount: 0,
    friendsCount: 0,
    groupsCount: 0
});

// 好友状态
const friendshipStatus = ref<'none' | 'pending_sent' | 'pending_received' | 'accepted'>('none');

// 面包屑导航链接
const breadcrumbLinks = computed(() => [
    {
        label: '首页',
        icon: 'i-heroicons-home',
        to: '/'
    },
    {
        label: '用户',
        icon: 'i-heroicons-users',
        to: '/users'
    },
    {
        label: user.value ? user.value.nickname : '用户详情',
        icon: 'i-heroicons-user'
    }
]);

// 计算属性
const friendshipStatusText = computed(() => {
    switch (friendshipStatus.value) {
        case 'pending_sent': return '已发送请求';
        case 'pending_received': return '待接受请求';
        case 'accepted': return '好友';
        default: return '';
    }
});

const friendshipStatusVariant = computed(() => {
    switch (friendshipStatus.value) {
        case 'pending_sent': return 'outline';
        case 'pending_received': return 'soft';
        case 'accepted': return 'soft';
        default: return 'outline';
    }
});

const friendshipStatusColor = computed(() => {
    switch (friendshipStatus.value) {
        case 'pending_sent': return 'info';
        case 'pending_received': return 'warning';
        case 'accepted': return 'success';
        default: return 'neutral';
    }
});

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
        } catch {}
        
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

// 加载用户统计信息（改为服务层 POST 接口）
async function loadUserStats() {
    try {
        const resp: any = await api.post(buildUserUrl(`${USER_ENDPOINTS.DETAILS}/${userId}/stats`), {}, { auth: true });
        userStats.value = resp?.data || { momentsCount: 0, friendsCount: 0, groupsCount: 0 };
    } catch (e: any) {
        console.error('加载用户统计失败:', e);
        toast.add({ title: '加载失败', description: e?.message || '无法加载用户统计信息', color: 'error' });
    }
}

// 加载用户动态（改为服务层 POST 接口）
async function loadUserMoments() {
    loadingMoments.value = true;
    try {
        const resp: any = await api.post(buildUserUrl(`${USER_ENDPOINTS.DETAILS}/${userId}/moments`), { cursor: momentsCursor.value || 0 }, { auth: true });
        userMoments.value = resp?.data || [];
        hasMoreMoments.value = !!resp?.has_more;
        momentsCursor.value = resp?.next_cursor;
    } catch (e: any) {
        toast.add({ title: '加载动态失败', description: e?.message, color: 'error' });
    } finally {
        loadingMoments.value = false;
    }
}

async function loadMoreMoments() {
    if (!hasMoreMoments.value || loadingMoreMoments.value) return;
    loadingMoreMoments.value = true;
    try {
        const resp: any = await api.post(buildUserUrl(`${USER_ENDPOINTS.DETAILS}/${userId}/moments`), { cursor: momentsCursor.value || 0 }, { auth: true });
        const items = resp?.data || [];
        userMoments.value.push(...items);
        hasMoreMoments.value = !!resp?.has_more;
        momentsCursor.value = resp?.next_cursor;
    } finally {
        loadingMoreMoments.value = false;
    }
}

// 好友操作：直接通过 api + FRIEND_ENDPOINTS
async function sendFriendRequest() {
    if (!user.value || operating.value) return;
    operating.value = true;
    try {
        await api.post(buildFriendUrl(FRIEND_ENDPOINTS.SEND_REQUEST), { user_id: String(user.value.user_id) }, { auth: true });
        friendshipStatus.value = 'pending_sent';
        user.value.friendship_status = 'pending_sent';
        toast.add({ title: '好友请求已发送', color: 'success' });
    } catch (e: any) {
        toast.add({ title: '发送失败', description: e?.message, color: 'error' });
    } finally {
        operating.value = false;
    }
}

async function acceptFriendRequest() {
    if (!user.value || operating.value) return;
    operating.value = true;
    try {
        await api.post(buildFriendUrl(`${FRIEND_ENDPOINTS.ACCEPT_REQUEST}/${String(user.value.user_id)}`), {}, { auth: true });
        friendshipStatus.value = 'accepted';
        user.value.friendship_status = 'accepted';
        toast.add({ title: '已添加为好友', color: 'success' });
    } catch (e: any) {
        toast.add({ title: '操作失败', description: e?.message, color: 'error' });
    } finally {
        operating.value = false;
    }
}

async function rejectFriendRequest() {
    if (!user.value || operating.value) return;
    operating.value = true;
    try {
        await api.post(buildFriendUrl(`${FRIEND_ENDPOINTS.REJECT_REQUEST}/${String(user.value.user_id)}`), {}, { auth: true });
        friendshipStatus.value = 'none';
        user.value.friendship_status = 'none';
        toast.add({ title: '已拒绝好友请求', color: 'info' });
    } catch (e: any) {
        toast.add({ title: '操作失败', description: e?.message, color: 'error' });
    } finally {
        operating.value = false;
    }
}

function getActionItems() {
    const items: any[] = [];
    if (friendshipStatus.value === 'accepted') {
        items.push([{ label: '删除好友', icon: 'i-heroicons-user-minus', click: () => removeFriend() }]);
    }
    items.push([{ label: '举报用户', icon: 'i-heroicons-flag', click: () => reportUser() }]);
    return items;
}

const showRemoveConfirm = ref(false);
async function removeFriend() {
    if (!user.value || operating.value) return;
    showRemoveConfirm.value = true;
}

function reportUser() {
    toast.add({ title: '功能暂未开放', description: '举报功能即将上线', color: 'info' });
}

function formatDistance(distance?: number): string {
    if (typeof distance !== 'number') return '';
    if (distance < 1000) return `${Math.round(distance)}米`;
    return `${(distance / 1000).toFixed(1)}公里`;
}

function formatJoinTime(dateStr?: string): string {
    if (!dateStr) return '';
    const date = new Date(dateStr);
    return `${date.getFullYear()}年${date.getMonth() + 1}月`;
}

function formatTime(s: string) {
    const t = new Date(s);
    const d = Date.now() - t.getTime();
    const m = Math.floor(d / 60000);
    const h = Math.floor(d / 3600000);
    const day = Math.floor(d / 86400000);
    if (m < 60) return `${m}分钟前`;
    if (h < 24) return `${h}小时前`;
    return `${day}天前`;
}
const doRemoveFriend = async () => {
    if (!user.value) return;
    operating.value = true;
    try {
        await api.post(buildFriendUrl(`${FRIEND_ENDPOINTS.REMOVE_FRIEND}/${String(user.value.user_id)}`), {}, { auth: true });
        friendshipStatus.value = 'none';
        user.value.friendship_status = 'none';
        toast.add({ title: '已删除好友', color: 'info' });
    } catch (e: any) {
        toast.add({ title: '操作失败', description: e?.message, color: 'error' });
    } finally {
        operating.value = false;
        showRemoveConfirm.value = false;
    }
};
 </script>

<style scoped>
.line-clamp-3 {
    display: -webkit-box;
    -webkit-line-clamp: 3;
    -webkit-box-orient: vertical;
    overflow: hidden;
}
</style>
