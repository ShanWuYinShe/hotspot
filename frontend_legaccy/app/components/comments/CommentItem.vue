<template>
    <div
        :class="[
            'group/comment flex gap-3',
            isReply
                ? 'py-2 pl-10'
                : 'py-4 border-b border-neutral-100 dark:border-neutral-800 last:border-0',
        ]"
    >
            <!-- 头像 -->
            <UAvatar
                :src="comment.user.avatar_url"
                :alt="comment.user.nickname"
                size="sm"
                class="shrink-0 mt-0.5"
            />

            <div class="flex-1 min-w-0">
                <!-- 头部信息 -->
                <div class="flex items-center justify-between gap-2 mb-1">
                    <div class="flex items-center gap-2 min-w-0">
                        <span
                            class="font-semibold text-neutral-900 dark:text-white text-sm truncate"
                        >
                            {{ comment.user.nickname }}
                        </span>
                    <UBadge
                            v-if="comment.user.user_type !== 'normal'"
                        :label="
                            comment.user.user_type === 'admin'
                                ? '管理员'
                                : '官方'
                        "
                        color="primary"
                        variant="subtle"
                        size="xs"
                    />
                    </div>
                    <span
                        class="text-sm text-neutral-500 dark:text-neutral-400 shrink-0"
                    >
                        {{ formatTime(comment.created_at) }}
                    </span>
                </div>

                <!-- 回复引用 -->
                <div
                    v-if="comment.reply_to_user"
                    class="flex items-center gap-1 mb-1 text-sm text-neutral-500 dark:text-neutral-400"
                >
                    <UIcon name="i-lucide-corner-down-right" class="size-3" />
                    <span>回复</span>
                    <span class="font-semibold text-neutral-900 dark:text-white"
                        >@{{ comment.reply_to_user.nickname }}</span
                    >
                </div>

                <!-- 内容 -->
                <p
                    class="text-neutral-600 dark:text-neutral-300 text-sm whitespace-pre-wrap wrap-break-word leading-relaxed"
                >
                    {{ comment.content }}
                </p>

                <!-- 操作栏 -->
                <div class="flex items-center gap-4 mt-2">
                    <UButton
                        variant="ghost"
                        size="xs"
                        :color="comment.liked_by_me ? 'error' : 'neutral'"
                        :icon="comment.liked_by_me ? 'i-lucide-heart-fill' : 'i-lucide-heart'"
                        :label="comment.likes_count ? String(comment.likes_count) : '赞'"
                        class="gap-1"
                        :loading="likeLoading"
                        @click="toggleLike"
                    />

                <UButton
                    variant="ghost"
                    size="xs"
                    color="neutral"
                    icon="i-lucide-message-circle"
                    label="回复"
                    class="gap-1"
                        @click="handleReply"
                        />

                <UButton
                        v-if="canDelete"
                    variant="ghost"
                    size="xs"
                    color="error"
                    icon="i-lucide-trash-2"
                    label="删除"
                    class="ml-auto opacity-0 group-hover/comment:opacity-100 focus:opacity-100 gap-1"
                        @click="handleDelete"
                />
                </div>

                <!-- 查看回复 -->
                <div v-if="comment.replies_count > 0 && !isReply" class="mt-3">
                <UButton
                    variant="ghost"
                    size="xs"
                    color="primary"
                    class="gap-1"
                        @click="toggleReplies"
                    >
                        <span class="w-4 h-px bg-current opacity-50 mr-1" />
                        <span>{{
                            showReplies
                                ? "收起回复"
                                : `展开 ${comment.replies_count} 条回复`
                        }}</span>
                </UButton>
                </div>
            </div>
        </div>

        <!-- 子回复列表 -->
        <div v-if="showReplies && replies.length > 0" class="mt-2 space-y-2">
            <CommentItem
                v-for="reply in replies"
                :key="reply.comment_id"
                :comment="reply"
                :is-reply="true"
                @reply="handleReplyToReply"
                @delete="handleDeleteReply"
                @like="handleLikeReply"
            />

        <UButton
                v-if="hasMoreReplies"
            variant="link"
            size="xs"
            color="neutral"
            :loading="loadingReplies"
            label="加载更多回复"
            class="ml-10 mt-2 pl-3 border-l-2 border-neutral-100 dark:border-neutral-800"
                @click="loadMoreReplies"
        />
    </div>
</template>

<script setup lang="ts">
import type { CommentListItem, UserInfo } from "~/types/api";
import { useCommentsService } from "~/services/comments.service";
import { useHydratedStore } from "~/composables/useHydratedStore";

interface CommentWithLikeStatus extends CommentListItem {
    liked_by_me?: boolean;
    reply_to_user?: UserInfo;
}

const props = defineProps<{
    comment: CommentWithLikeStatus;
    isReply?: boolean;
}>();
const emit = defineEmits<{
    reply: [comment: CommentWithLikeStatus];
    delete: [commentId: string];
    like: [commentId: string];
}>();

const { formatTime } = useFormatters();
const { requireAuth } = useRequireAuth();
const commentsService = useCommentsService();
const authStore = useAuthStore();
const hydratedUser = useHydratedStore(authStore, "user", null);
const feedback = useFeedbackStore();
const modal = useModal();

const showReplies = ref(false);
const replies = ref<CommentWithLikeStatus[]>([]);
const hasMoreReplies = ref(false);
const nextCursor = ref<string | undefined>();
const loadingReplies = ref(false);

const canDelete = computed(
    () => hydratedUser.value?.user_id === props.comment.user.user_id,
);

const likeLoading = ref(false);
const toggleLike = async () => {
    if (!requireAuth("点赞评论")) return;
    if (likeLoading.value) return;

    const previousLiked = props.comment.liked_by_me;
    const previousCount = props.comment.likes_count;

    // Optimistic
    props.comment.liked_by_me = !previousLiked;
    props.comment.likes_count += props.comment.liked_by_me ? 1 : -1;
    emit("like", props.comment.comment_id);

    likeLoading.value = true;
    try {
        if (previousLiked) {
            await commentsService.unlike({ comment_id: props.comment.comment_id });
        } else {
            await commentsService.like({ comment_id: props.comment.comment_id });
        }
    } catch (e) {
        // Revert
        props.comment.liked_by_me = previousLiked;
        props.comment.likes_count = previousCount;
    } finally {
        likeLoading.value = false;
    }
};

const handleReply = () => {
    // 符合设计理念：只在执行操作时才要求登录，提供友好的登录引导
    if (!requireAuth("回复评论")) return;
    emit("reply", props.comment);
};

const handleDelete = async () => {
    if (
        await modal.showConfirm({
            title: "删除评论",
            message: "确定要删除这条评论吗？",
            confirmColor: "error",
        })
    ) {
        try {
            await commentsService.delete({ comment_id: props.comment.comment_id });
            feedback.success("删除成功");
            emit("delete", props.comment.comment_id);
        } catch (e) {
            feedback.error("删除失败");
        }
    }
};

const toggleReplies = async () => {
    showReplies.value = !showReplies.value;
    if (showReplies.value && replies.value.length === 0) await loadReplies();
};

const loadReplies = async () => {
    if (loadingReplies.value) return;
    loadingReplies.value = true;
    try {
        const res = await commentsService.getReplies({
            comment_id: props.comment.comment_id,
            limit: 5,
            cursor: nextCursor.value,
        });
        replies.value = res.items;
        hasMoreReplies.value = res.pagination.has_more;
        nextCursor.value = res.pagination.next_cursor;
    } finally {
        loadingReplies.value = false;
    }
};

const loadMoreReplies = async () => {
    if (loadingReplies.value || !nextCursor.value) return;
    loadingReplies.value = true;
    try {
        const res = await commentsService.getReplies({
            comment_id: props.comment.comment_id,
            limit: 5,
            cursor: nextCursor.value,
        });
        replies.value.push(...res.items);
        hasMoreReplies.value = res.pagination.has_more;
        nextCursor.value = res.pagination.next_cursor;
    } finally {
        loadingReplies.value = false;
    }
};

const handleReplyToReply = (r: CommentWithLikeStatus) => emit("reply", r);
const handleDeleteReply = (rid: string) => {
    replies.value = replies.value.filter((r) => r.comment_id !== rid);
    props.comment.replies_count--;
};
const handleLikeReply = () => {}; // 子组件已处理状态
</script>
