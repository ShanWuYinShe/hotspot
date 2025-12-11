<template>
    <div class="mx-auto max-w-4xl space-y-6">
        <!-- 加载中 -->
        <div v-if="loading" class="space-y-4">
            <USkeleton class="h-12" />
            <USkeleton class="h-32" />
            <USkeleton class="h-24" />
        </div>

        <!-- 动态不存在 -->
        <UCard v-else-if="!moment">
            <UEmpty
                icon="i-heroicons-exclamation-circle"
                title="动态不存在"
                description="该动态可能已被删除或不存在"
            >
                <template #actions>
                    <UButton to="/moments">返回动态列表</UButton>
                </template>
            </UEmpty>
        </UCard>

        <!-- 动态详情 -->
        <div v-else class="space-y-6">
            <!-- 主要内容卡片 -->
            <UCard>
                <!-- 用户信息 -->
                <div class="flex items-center gap-3 mb-4">
                    <UAvatar 
                        :alt="moment.user.nickname"
                        :src="moment.user.avatar_url"
                        size="lg"
                    >
                        {{ moment.user.nickname?.charAt(0) }}
                    </UAvatar>
                    <div class="flex-1 min-w-0">
                        <div class="font-semibold text-lg">{{ moment.user.nickname }}</div>
                        <div class="text-sm text-secondary-content">
                            {{ formatTime(moment.created_at) }}
                            <span 
                                v-if="moment.location?.name" 
                                class="ml-2 inline-flex items-center gap-1"
                            >
                                <UIcon name="i-heroicons-map-pin" class="icon-xs" />
                                {{ moment.location.name }}
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
                <div class="flex items-center gap-6 pt-4 border-t border-default">
                    <UButton
                        variant="ghost"
                        size="sm"
                        :class="likingState.liked ? 'text-red' : 'text-secondary-strong'"
                        @click="toggleLike"
                        :loading="likingState.loading"
                    >
                        <UIcon
                            :name="likingState.liked ? 'i-heroicons-heart-solid' : 'i-heroicons-heart'"
                            class="w-5 h-5 mr-2"
                        />
                        {{ likingState.count }}
                    </UButton>
                    
                    <div class="flex items-center text-secondary-strong">
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
                            :alt="authUser?.nickname"
                            size="sm"
                        >
                            {{ authUser?.nickname?.charAt(0) || "U" }}
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
                    <div v-if="comments.length === 0" class="py-8 text-center text-secondary-content">
                        还没有评论，快来抢沙发吧~
                    </div>
                    
                    <div v-else class="space-y-4">
                        <div 
                            v-for="comment in comments" 
                            :key="comment.id"
                            class="flex gap-3 p-4 rounded-lg bg-surface"
                        >
                            <UAvatar 
                                :alt="comment.user?.nickname"
                                size="sm"
                            >
                                {{ comment.user?.nickname?.charAt(0) }}
                            </UAvatar>
                            <div class="flex-1">
                                <div class="flex items-center gap-2 mb-1">
                                    <span class="font-medium">{{ comment.user?.nickname }}</span>
                                    <span class="text-xs text-secondary-content">{{ formatTime(comment.created_at) }}</span>
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
import type { MomentSearchItem } from '~/types/api';
import { getErrorMessage } from '~/utils/error';
import { useMomentsService } from '~/services/moments.service';
import { useCommentsService } from '~/services/comments.service';

definePageMeta({ 
    title: '动态详情',
    layout: 'dashboard'
});

const route = useRoute();
const router = useRouter();
// 使用水合安全的 store 访问
const authStore = useAuthStore();
const authUser = computed(() => authStore.user);
const { formatTime } = useFormatters();
const { requireAuth } = useRequireAuth();
const momentsStore = useMomentsStore();
const geolocation = useGeolocation();
const toast = useToast()

const momentsService = useMomentsService();
const commentsService = useCommentsService();

const momentId = route.params.id as string;

// 状态变量
const loading = ref(true);
const commenting = ref(false);
const commentText = ref('');
const loadingMoreComments = ref(false);

// 点赞状态（本地管理，避免频繁请求）
const likingState = ref({
    loading: false,
    count: 0,
    liked: false
});

// 使用 API 定义的评论类型
import type { CommentListItem } from '~/types/api';

// 扩展类型：包含前端本地管理的点赞状态
interface CommentWithLikeStatus extends CommentListItem {
  liked_by_me?: boolean
}

// 评论数据
const comments = ref<CommentWithLikeStatus[]>([]);
const hasMoreComments = ref(false);
const commentCount = computed(() => comments.value.length);
const commentCursor = ref<string | undefined>(undefined);

// 动态详情
const momentData = ref<any>(null);
const moment = computed(() => momentData.value);

onMounted(async () => {
    await fetchMomentDetail();
    await fetchComments();
});

async function fetchMomentDetail() {
    loading.value = true;
    try {
        const coords = await geolocation.getCoordinates();
        const momentDetail = await momentsService.openTheDoor({ 
            id: momentId,
            latitude: coords.latitude,
            longitude: coords.longitude
        })
        momentData.value = momentDetail
        momentsStore.setMomentDetails(momentId, momentDetail as any)
        
        // 初始化点赞状态（API 不返回 liked_by_me，默认为 false）
        likingState.value = {
            loading: false,
            count: momentDetail.likes_count || 0,
            liked: false
        };
    } catch (e: unknown) {
        toast.add({
            title: '加载失败',
            description: getErrorMessage(e, '获取动态详情失败'),
            color: 'error'
        });
    } finally {
        loading.value = false;
    }
}

async function fetchComments() {
    try {
        const response = await commentsService.getList({
            moment_id: momentId,
            limit: 20
        });
        
        comments.value = response.items || [];
        hasMoreComments.value = response.pagination.has_more || false;
        commentCursor.value = response.pagination.next_cursor;
    } catch (error: unknown) {
        console.error('获取评论失败:', error);
        // 评论加载失败不显示错误提示，保持静默
    }
}

async function toggleLike() {
    // ✅ 功能级认证检查
    if (!requireAuth('点赞')) return;
    
    if (likingState.value.loading || !moment.value) return;
    
    likingState.value.loading = true;
    const originalState = { ...likingState.value };
    const wasLiked = likingState.value.liked;
    
    // 乐观更新
    likingState.value.liked = !likingState.value.liked;
    likingState.value.count += likingState.value.liked ? 1 : -1;
    
    try {
        // 根据当前状态调用对应 API
        if (wasLiked) {
            await momentsService.unlikeMoment({ id: moment.value.id });
        } else {
            await momentsService.likeMoment({ id: moment.value.id });
        }
        // 更新 store 状态
        momentsStore.setLikeStatus(moment.value.id, !wasLiked);
    } catch (e: unknown) {
        // 回滚
        likingState.value = originalState;
        toast.add({
            title: '操作失败',
            description: getErrorMessage(e),
            color: 'error'
        });
    } finally {
        likingState.value.loading = false;
    }
}

async function submitComment() {
    // ✅ 功能级认证检查
    if (!requireAuth('发表评论')) return;
    
    if (!commentText.value.trim() || commenting.value) return;
    
    commenting.value = true;
    try {
        const newCommentData = await commentsService.add({
            moment_id: momentId,
            content: commentText.value.trim()
        });
        
        // 重新获取评论列表（包含完整的评论信息）
        await fetchComments();
        commentText.value = '';
        
        toast.add({
            title: '评论成功',
            color: 'success'
        });
        
    } catch (error: unknown) {
        toast.add({
            title: '评论失败',
            description: getErrorMessage(error, '请稍后重试'),
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
        const response = await commentsService.getList({
            moment_id: momentId,
            cursor: commentCursor.value,
            limit: 20
        });
        
        comments.value.push(...(response.items || []));
        hasMoreComments.value = response.pagination.has_more || false;
        commentCursor.value = response.pagination.next_cursor;
    } catch (error: unknown) {
        toast.add({
            title: '加载评论失败',
            description: getErrorMessage(error, '请稍后重试'),
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

function getMomentActions(m: MomentSearchItem) {
    const isOwner = m.user.user_id === authStore.user?.user_id;
    const items: { label: string; icon: string; click: () => void; class?: string }[][] = [];
    
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

async function onDelete(m: MomentSearchItem) {
    const modal = useModal()
    const confirmed = await modal.showConfirm({
        title: '删除动态',
        message: '确定要删除这条动态吗？此操作无法撤销。',
        confirmText: '删除',
        confirmColor: 'error'
    })
    if (!confirmed) return
    
    try {
        await momentsService.deleteMoment({ id: m.id });
        toast.add({ title: "已删除", color: "success" });
        // 删除后返回上一页
        router.back();
    } catch (e: unknown) {
        toast.add({
            title: "操作失败",
            description: getErrorMessage(e),
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
