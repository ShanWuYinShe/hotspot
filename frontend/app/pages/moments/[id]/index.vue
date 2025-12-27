<template>
    <div class="w-full">
        <!-- 加载中 -->
        <div v-if="loading" class="space-y-4">
            <USkeleton class="h-12" />
            <USkeleton class="h-32" />
            <USkeleton class="h-24" />
        </div>

        <!-- 动态不存在 -->
        <UCard v-else-if="!moment">
            <UEmpty
                icon="i-lucide-circle-alert"
                title="动态不存在"
                description="该动态可能已被删除或不存在"
            >
                <template #actions>
                    <UButton to="/moments/nearby">返回动态列表</UButton>
                </template>
            </UEmpty>
        </UCard>

        <!-- 动态详情 -->
        <div v-else class="space-y-6">
            <!-- 主要内容卡片 -->
            <div v-if="momentCardData" class="w-full">
                <MomentCard
                    :moment="momentCardData"
                    class="w-full"
                    @like="toggleLike"
                    @comment="scrollToComments"
                    @share="shareMoment"
                    @deleted="onDeleteSuccess"
                    @updated="fetchMomentDetail"
                />
            </div>

            <!-- 评论区域 -->
            <UCard id="comments">
                <CommentList 
                    :moment-id="momentId" 
                    :initial-total="moment?.comments_count"
                    @update:total="updateCommentsCount"
                />
            </UCard>
        </div>
    </div>
</template>

<script setup lang="ts">
import type { MomentSearchItem, UserRelationship } from '~/types/api';
import type { MomentCardData } from '~/types/client';
import { useMomentsService } from '~/services/moments.service';
import { useUsersService } from '~/services/users.service';
import CommentList from '~/components/comments/CommentList.vue';
import MomentCard from '~/components/cards/MomentCard.vue';
import { useDashboardHeader } from '~/composables/useDashboardHeader'

definePageMeta({ 
    title: '动态详情',
    description: '查看动态内容与评论',
    layout: 'dashboard',
    dashboardHeader: true
});

const route = useRoute();
const router = useRouter();
const dashboardHeader = useDashboardHeader()

watchEffect(() => {
    dashboardHeader.setActions([
        {
            key: 'back',
            label: '返回',
            icon: 'i-lucide-arrow-left',
            color: 'neutral',
            variant: 'outline',
            action: () => router.back()
        }
    ])
})
// 水合安全：用户信息
const authStore = useAuthStore();
const authUser = ref<typeof authStore.user>(null);

onMounted(() => {
    authUser.value = authStore.user;
    watch(() => authStore.user, (user) => {
        authUser.value = user;
    });
});
const { requireAuth } = useRequireAuth();
const momentsStore = useMomentsStore();
const geolocation = useGeolocation();
const feedback = useFeedbackStore()
const { withErrorHandling, handleError } = useErrorHandler()
const { share, isSupported: isShareSupported } = useShare()
const { copy, isSupported: isClipboardSupported } = useClipboard()

const momentsService = useMomentsService();
const userService = useUsersService();

const momentId = route.params.id as string;

// 状态变量
const loading = ref(true);

// 动态详情
const momentData = ref<MomentSearchItem | null>(null);
// 优先使用 store 中的数据（保持响应式），如果没有则使用本地 fetching 的数据
const moment = computed(() => momentsStore.getMoment(momentId) || momentData.value);

// 扩展数据：关系和点赞状态
const relationship = ref<UserRelationship | null>(null);
const isLiked = ref(false);

// 组合成 MomentCard 需要的数据格式
const momentCardData = computed((): MomentCardData | null => {
    if (!moment.value) return null;
    
    return {
        ...moment.value,
        relationship: relationship.value ? {
            is_following: relationship.value.is_following,
            is_mutual_follow: relationship.value.is_mutual_follow,
            // 其他 UserRelationship 字段...
            is_followed_by: relationship.value.is_followed_by,
            is_blocked: relationship.value.is_blocked,
            is_blocking: relationship.value.is_blocking
        } : undefined,
        liked_by_me: isLiked.value,
        // 如果点赞数变了，使用本地状态
        likes_count: moment.value.likes_count + (isLiked.value ? (initialLikedStatus.value ? 0 : 1) : (initialLikedStatus.value ? -1 : 0)),
        // 评论数优先使用 moment.value 中的值（由 CommentList 更新）
        comments_count: moment.value.comments_count
    };
});

// 记录初始点赞状态，用于计算乐观更新后的数量
const initialLikedStatus = ref(false);

onMounted(async () => {
    // 1. 尝试从 store 获取缓存数据（优先显示）
    const cachedMoment = momentsStore.getMoment(momentId);
    if (cachedMoment) {
        momentData.value = cachedMoment;
        if (cachedMoment.liked_by_me !== undefined) {
            isLiked.value = cachedMoment.liked_by_me;
            initialLikedStatus.value = cachedMoment.liked_by_me;
        }
        loading.value = false;
    }

    // 2. 获取最新数据
    await fetchMomentDetail();
});

async function fetchMomentDetail() {
    await withErrorHandling(async () => {
        const coords = await geolocation.getCoordinates();
        
        // 1. 获取动态详情
        const momentDetail = await momentsService.openTheDoor({ 
            id: momentId,
            latitude: coords.latitude,
            longitude: coords.longitude
        })
        
        momentData.value = momentDetail
        momentsStore.setMomentDetails(momentId, momentDetail)
        
        // 2. 并行获取补充信息：点赞状态、用户关系
        const fetchPromises: Promise<any>[] = [
            momentsService.checkLiked({ id: momentId }),
        ];
        
        const isOwner = authUser.value && momentDetail.user.user_id === authUser.value.user_id;
        if (!isOwner) {
            fetchPromises.push(userService.getRelationship({ id: momentDetail.user.user_id }));
        }
        
        const results = await Promise.allSettled(fetchPromises);
        
        // 处理点赞状态
        const likedRes = results[0];
        if (likedRes?.status === 'fulfilled') {
            if (likedRes.value?.code === 0) {
                isLiked.value = likedRes.value.data?.is_liked ?? false;
                initialLikedStatus.value = isLiked.value;
            }
        }
        
        // 处理用户关系
        if (!isOwner && results.length > 1) {
            const relationRes = results[1];
            if (relationRes?.status === 'fulfilled') {
                if (relationRes.value?.code === 0) {
                    relationship.value = relationRes.value.data;
                }
            }
        }

    }, {
        title: '加载失败',
        fallbackMessage: '获取动态详情失败',
        loading: loading
    })
}

async function toggleLike() {
    // ✅ 功能级认证检查
    if (!requireAuth('点赞')) return;
    if (!moment.value) return;
    
    const wasLiked = isLiked.value;
    
    // 乐观更新
    isLiked.value = !wasLiked;
    
    try {
        if (wasLiked) {
            await momentsService.unlikeMoment({ id: moment.value.id });
        } else {
            await momentsService.likeMoment({ id: moment.value.id });
        }
        // 更新 store 状态
        momentsStore.setLikeStatus(moment.value.id, !wasLiked);
    } catch (e: unknown) {
        // 回滚
        isLiked.value = wasLiked;
        handleError(e, { title: '操作失败' })
    }
}

function shareMoment() {
    const shareData = {
        title: `${moment.value?.user.nickname}的动态`,
        text: moment.value?.description,
        url: useRequestURL().href
    }

    if (isShareSupported.value) {
        share(shareData).catch((err) => {
            console.error('分享失败:', err)
        })
    } else if (isClipboardSupported.value) {
        copy(shareData.url)
        feedback.success('链接已复制', '动态链接已复制到剪贴板')
    } else {
        feedback.error('无法分享', '您的浏览器不支持分享或复制功能')
    }
}

function scrollToComments() {
    const commentsEl = document.getElementById('comments');
    if (commentsEl) {
        commentsEl.scrollIntoView({ behavior: 'smooth' });
    }
}

function onDeleteSuccess() {
    router.back();
}

function updateCommentsCount(count: number) {
    if (momentData.value) {
        momentData.value = {
            ...momentData.value,
            comments_count: count
        }
        // 同时更新 store
        momentsStore.setMomentDetails(momentId, momentData.value)
    }
}

</script>
