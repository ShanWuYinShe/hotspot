<template>
    <div class="mx-auto max-w-4xl px-4 sm:px-6 lg:px-8 py-14 sm:py-16">
        <!-- 面包屑导航 -->
        <div class="mb-6">
            <UBreadcrumb :links="breadcrumbLinks" />
        </div>

        <!-- 加载中 -->
        <div v-if="loading" class="space-y-4">
            <USkeleton class="h-12" />
            <USkeleton class="h-32" />
            <USkeleton class="h-24" />
        </div>

        <!-- 动态不存在 -->
        <UCard v-else-if="!moment">
            <EmptyState
                icon="i-heroicons-exclamation-circle"
                title="动态不存在"
                description="该动态可能已被删除或不存在"
                action-text="返回动态广场"
                :action-handler="() => $router.push('/moments')"
            />
        </UCard>

        <!-- 动态详情 -->
        <div v-else class="space-y-6">
            <!-- 主要内容卡片 -->
            <UCard>
                <!-- 用户信息 -->
                <div class="flex items-center gap-3 mb-4">
                    <UAvatar 
                        :alt="moment.nickname"
                        size="lg"
                    >
                        {{ moment.nickname?.charAt(0) }}
                    </UAvatar>
                    <div class="flex-1 min-w-0">
                        <div class="font-semibold text-lg">{{ moment.nickname }}</div>
                        <div class="text-sm text-neutral-600 dark:text-neutral-400">
                            {{ formatTime(moment.created_at) }}
                            <span 
                                v-if="moment.location_name" 
                                class="ml-2 inline-flex items-center gap-1"
                            >
                                <UIcon name="i-heroicons-map-pin" class="w-3 h-3" />
                                {{ moment.location_name }}
                            </span>
                        </div>
                    </div>
                    <UDropdownMenu :items="getMomentActions(moment)">
                        <UButton
                            icon="i-heroicons-ellipsis-horizontal"
                            variant="ghost"
                            size="sm"
                        />
                    </UDropdownMenu>
                </div>

                <!-- 动态内容 -->
                <div class="mb-4">
                    <p class="text-lg leading-relaxed whitespace-pre-wrap">{{ moment.description }}</p>
                </div>

                <!-- 标签 -->
                <div v-if="moment.tags && moment.tags.length > 0" class="mb-4">
                    <div class="flex flex-wrap gap-2">
                        <UBadge 
                            v-for="tag in moment.tags" 
                            :key="tag"
                            variant="soft"
                            size="xs"
                        >
                            #{{ tag }}
                        </UBadge>
                    </div>
                </div>

                <!-- 统计信息 -->
                <div class="flex items-center gap-6 pt-4 border-t border-neutral-200 dark:border-neutral-700">
                    <UButton
                        variant="ghost"
                        size="sm"
                        :class="moment.liked_by_me ? 'text-red-600' : 'text-neutral-700 dark:text-neutral-300'"
                        @click="toggleLike"
                        :loading="likingState.loading"
                    >
                        <UIcon
                            :name="moment.liked_by_me ? 'i-heroicons-heart-solid' : 'i-heroicons-heart'"
                            class="w-5 h-5 mr-2"
                        />
                        {{ likingState.count }}
                    </UButton>
                    
                    <div class="flex items-center text-neutral-700 dark:text-neutral-300">
                        <UIcon name="i-heroicons-chat-bubble-left-right" class="w-5 h-5 mr-2" />
                        {{ commentCount }}
                    </div>
                    
                    <UButton variant="ghost" size="sm" @click="shareMoment">
                        <UIcon name="i-heroicons-share" class="w-5 h-5 mr-2" />
                        分享
                    </UButton>
                </div>
            </UCard>

            <!-- 评论区域 -->
            <UCard>
                <div class="space-y-4">
                    <h3 class="text-lg font-semibold">评论 ({{ commentCount }})</h3>
                    
                    <!-- 发表评论 -->
                    <div class="flex gap-3">
                        <UAvatar 
                            :alt="authStore.user?.nickname"
                            size="sm"
                        >
                            {{ authStore.user?.nickname?.charAt(0) || "U" }}
                        </UAvatar>
                        <div class="flex-1">
                            <UTextarea
                                v-model="commentText"
                                placeholder="写点评论..."
                                :rows="2"
                                class="mb-2"
                            />
                            <div class="flex justify-end">
                                <UButton
                                    size="sm"
                                    :disabled="!commentText.trim()"
                                    :loading="commenting"
                                    @click="submitComment"
                                >
                                    发表评论
                                </UButton>
                            </div>
                        </div>
                    </div>

                    <!-- 评论列表 -->
                    <div v-if="comments.length === 0" class="py-8 text-center text-neutral-500">
                        还没有评论，快来抢沙发吧~
                    </div>
                    
                    <div v-else class="space-y-4">
                        <div 
                            v-for="comment in comments" 
                            :key="comment.id"
                            class="flex gap-3 p-4 rounded-lg bg-neutral-50 dark:bg-neutral-900"
                        >
                            <UAvatar 
                                :alt="comment.nickname"
                                size="sm"
                            >
                                {{ comment.nickname?.charAt(0) }}
                            </UAvatar>
                            <div class="flex-1">
                                <div class="flex items-center gap-2 mb-1">
                                    <span class="font-medium">{{ comment.nickname }}</span>
                                    <span class="text-xs text-neutral-500">{{ formatTime(comment.created_at) }}</span>
                                </div>
                                <p class="text-sm">{{ comment.content }}</p>
                            </div>
                        </div>
                    </div>

                    <!-- 加载更多评论 -->
                    <div v-if="hasMoreComments" class="text-center">
                        <UButton 
                            variant="ghost" 
                            size="sm"
                            :loading="loadingMoreComments"
                            @click="loadMoreComments"
                        >
                            加载更多评论
                        </UButton>
                    </div>
                </div>
            </UCard>
        </div>
    </div>
</template>

<script setup lang="ts">
import type { MomentResponse } from '~/types/api';

definePageMeta({ 
    title: '动态详情', 
    requiresAuth: true,
    layout: 'dashboard'
});

const route = useRoute();
const authStore = useAuthStore();
const { formatTime } = useFormatters();
const momentsStore = useMomentsStore();
const { success, error, info } = useNotifications()
const toast = { success, error, info };

const momentId = route.params.id as string;

// 状态变量
const loading = ref(true);

// 面包屑导航链接
const breadcrumbLinks = computed(() => [
    {
        label: '首页',
        icon: 'i-heroicons-home',
        to: '/'
    },
    {
        label: '动态',
        icon: 'i-heroicons-photo',
        to: '/moments'
    },
    {
        label: moment.value ? `${moment.value.nickname}的动态` : '动态详情',
        icon: 'i-heroicons-document-text'
    }
]);
const commenting = ref(false);
const commentText = ref('');
const loadingMoreComments = ref(false);

// 点赞状态（本地管理，避免频繁请求）
const likingState = ref({
    loading: false,
    count: 0,
    liked: false
});

// 评论类型定义
interface Comment {
  id: string;
  user_id: string;
  nickname: string;
  avatar_url?: string;
  content: string;
  created_at: string;
}

// 评论数据
const comments = ref<Comment[]>([]);
const hasMoreComments = ref(false);
const commentCount = computed(() => comments.value.length);
const commentCursor = ref<string | null>(null);

// 动态详情
const moment = computed(() => momentsStore.currentMoment);

onMounted(async () => {
    await fetchMomentDetail();
    await fetchComments();
});

async function fetchMomentDetail() {
    loading.value = true;
    try {
        const momentDetail = await momentsStore.fetchMomentDetails(momentId);
        
        // 初始化点赞状态
        likingState.value = {
            loading: false,
            count: momentDetail.likes_count,
            liked: momentDetail.liked_by_me
        };
    } catch (e: any) {
        toast.add({
            title: '加载失败',
            description: e?.message || '获取动态详情失败',
            color: 'error'
        });
    } finally {
        loading.value = false;
    }
}

async function fetchComments() {
    try {
        const api = useApi();
        const response = await api.post('/comments/moment/' + momentId, {
            cursor: null,
            limit: 20
        });
        
        if (response.code === 0 && response.data) {
            comments.value = response.data.items || [];
            hasMoreComments.value = response.data.has_more || false;
            commentCursor.value = response.data.next_cursor;
        }
    } catch (error: any) {
        console.error('获取评论失败:', error);
        // 评论加载失败不显示错误提示，保持静默
    }
}

async function toggleLike() {
    if (likingState.value.loading || !moment.value) return;
    
    likingState.value.loading = true;
    const originalState = { ...likingState.value };
    
    // 乐观更新
    likingState.value.liked = !likingState.value.liked;
    likingState.value.count += likingState.value.liked ? 1 : -1;
    
    try {
        if (originalState.liked) {
            await momentsStore.unlikeMoment(moment.value.moment_id);
        } else {
            await momentsStore.likeMoment(moment.value.moment_id);
        }
    } catch (e: any) {
        // 回滚
        likingState.value = originalState;
        toast.add({
            title: '操作失败',
            description: e?.message,
            color: 'error'
        });
    } finally {
        likingState.value.loading = false;
    }
}

async function submitComment() {
    if (!commentText.value.trim() || commenting.value) return;
    
    commenting.value = true;
    try {
        const api = useApi();
        const response = await api.post('/comments/create', {
            moment_id: momentId,
            content: commentText.value.trim()
        });
        
        if (response.code === 0) {
            // 添加新评论到列表顶部
            comments.value.unshift(response.data);
            commentText.value = '';
            
            toast.add({
                title: '评论成功',
                color: 'success'
            });
        } else {
            throw new Error(response.msg || '评论失败');
        }
        
    } catch (error: any) {
        toast.add({
            title: '评论失败',
            description: error?.message || '请稍后重试',
            color: 'error'
        });
    } finally {
        commenting.value = false;
    }
}

async function loadMoreComments() {
    if (loadingMoreComments.value || !hasMoreComments.value) return;
    
    loadingMoreComments.value = true;
    try {
        const api = useApi();
        const response = await api.post('/comments/moment/' + momentId, {
            cursor: commentCursor.value,
            limit: 20
        });
        
        if (response.code === 0 && response.data) {
            comments.value.push(...(response.data.items || []));
            hasMoreComments.value = response.data.has_more || false;
            commentCursor.value = response.data.next_cursor;
        }
    } catch (error: any) {
        toast.add({
            title: '加载评论失败',
            description: error?.message || '请稍后重试',
            color: 'error'
        });
    } finally {
        loadingMoreComments.value = false;
    }
}

function shareMoment() {
    if (navigator.share) {
        navigator.share({
            title: `${moment.value?.nickname}的动态`,
            text: moment.value?.description,
            url: window.location.href
        });
    } else {
        // 复制链接
        navigator.clipboard.writeText(window.location.href);
        toast.add({
            title: '链接已复制',
            description: '动态链接已复制到剪贴板',
            color: 'success'
        });
    }
}

function getMomentActions(m: MomentResponse) {
    const isOwner = m.user_id === authStore.user?.user_id;
    const items: any[] = [];
    
    if (isOwner) {
        items.push([
            {
                label: "删除",
                icon: "i-heroicons-trash",
                click: () => onDelete(m),
            },
        ]);
    } else {
        items.push([
            {
                label: "举报",
                icon: "i-heroicons-flag",
                click: () => reportMoment(),
            },
        ]);
    }
    
    return items;
}

async function onDelete(m: MomentResponse) {
    try {
        await momentsStore.deleteMoment(m.moment_id);
        toast.add({ title: "已删除", color: "success" });
        // 删除后返回动态广场
        navigateTo('/moments');
    } catch (e: any) {
        toast.add({
            title: "操作失败",
            description: e?.message,
            color: "error",
        });
    }
}

function reportMoment() {
    toast.add({ 
        title: "功能暂未开放", 
        description: "举报功能即将上线",
        color: "info" 
    });
}

// formatTime 函数已从统一工具函数导入

// onMounted 和数据加载逻辑已在下方的现有代码中处理
</script>
