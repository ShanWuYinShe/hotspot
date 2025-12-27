<template>
  <div :class="isReply ? 'py-2' : 'py-3'">
    <!-- 评论内容 -->
    <div class="flex gap-3">
      <!-- 评论主体 -->
      <div class="flex-1 min-w-0">
        <div class="flex items-start justify-between gap-2 mb-1">
          <UUser
            class="min-w-0"
            :name="comment.user.nickname"
            :avatar="{ alt: comment.user.nickname, size: 'sm' }"
            size="sm"
          />
          <UBadge color="neutral" variant="soft" size="xs" class="shrink-0">
            {{ formatTime(comment.created_at) }}
          </UBadge>
        </div>

        <!-- 回复对象 -->
        <div v-if="comment.reply_to_user" class="mb-1">
          <UBadge color="neutral" variant="soft" size="xs">
            回复 {{ comment.reply_to_user.nickname }}
          </UBadge>
        </div>

        <!-- 评论内容 -->
        <p class="text-sm whitespace-pre-wrap wrap-break-word leading-relaxed">
          {{ comment.content }}
        </p>

        <!-- 操作按钮 -->
        <div class="flex items-center gap-2 mt-2">
          <!-- 点赞 -->
          <UButton
            variant="ghost"
            size="sm"
            :color="comment.liked_by_me ? 'error' : 'neutral'"
            :icon="comment.liked_by_me ? 'i-lucide-heart' : 'i-lucide-heart'"
            :label="String(comment.likes_count || 0)"
            @click="handleLike"
          />

          <!-- 回复 -->
          <UButton
            variant="ghost"
            size="sm"
            color="neutral"
            icon="i-lucide-message-circle"
            label="回复"
            @click="handleReply"
          />

          <!-- 查看回复 -->
          <UButton
            v-if="comment.replies_count > 0 && !isReply"
            variant="ghost"
            size="sm"
            color="neutral"
            :icon="showReplies ? 'i-lucide-chevron-up' : 'i-lucide-chevron-down'"
            :label="`${comment.replies_count} 条回复`"
            @click="toggleReplies"
          />

          <!-- 删除（仅作者可见） -->
          <UButton
            v-if="canDelete"
            variant="ghost"
            size="sm"
            color="error"
            icon="i-lucide-trash-2"
            label="删除"
            class="ml-auto"
            @click="handleDelete"
          />
        </div>
      </div>
    </div>

    <!-- 回复列表 -->
    <div v-if="showReplies && replies.length > 0" class="mt-3 ml-11 space-y-3">
      <CommentItem
        v-for="reply in replies"
        :key="reply.id"
        :comment="reply"
        :is-reply="true"
        @reply="handleReplyToReply"
        @delete="handleDeleteReply"
      />
      
      <!-- 加载更多回复 -->
      <UButton
        v-if="hasMoreReplies"
        variant="link"
        size="sm"
        color="neutral"
        :label="loadingReplies ? '加载中...' : '查看更多回复'"
        :loading="loadingReplies"
        :disabled="loadingReplies"
        @click="loadMoreReplies"
      />
    </div>
  </div>
</template>

<script setup lang="ts">
import type { CommentListItem } from '~/types/api'
import { useCommentsService } from '~/services/comments.service'
import { useHydratedStore } from '~/composables/useHydratedStore'

// 扩展类型：包含前端本地管理的点赞状态
interface CommentWithLikeStatus extends CommentListItem {
  liked_by_me?: boolean
}

const props = defineProps<{
  comment: CommentWithLikeStatus
  isReply?: boolean
}>()

const emit = defineEmits<{
  reply: [comment: CommentWithLikeStatus]
  delete: [commentId: string]
  like: [commentId: string]
}>()

const { formatTime } = useFormatters()
const { requireAuth } = useRequireAuth()
const commentsService = useCommentsService()
// 使用水合安全的 store 访问
const authStore = useAuthStore()
const hydratedUser = useHydratedStore(authStore, 'user', null as typeof authStore.user)
const feedback = useFeedbackStore()

// 状态
const showReplies = ref(false)
const replies = ref<CommentWithLikeStatus[]>([])
const hasMoreReplies = ref(false)
const nextCursor = ref<string | undefined>()
const loadingReplies = ref(false)

// 是否可以删除（作者本人）
const canDelete = computed(() => {
  return hydratedUser.value?.user_id === props.comment.user.user_id
})

// 点赞
const handleLike = async () => {
  // ✅ 功能级认证检查
  if (!requireAuth('点赞评论')) return;
  
  try {
    const wasLiked = props.comment.liked_by_me
    
    if (wasLiked) {
      await commentsService.unlike({ id: props.comment.id })
    } else {
      await commentsService.like({ id: props.comment.id })
    }
    
    // 更新本地状态
    props.comment.liked_by_me = !props.comment.liked_by_me
    props.comment.likes_count += props.comment.liked_by_me ? 1 : -1
    
    emit('like', props.comment.id)
  } catch (error: unknown) {
    feedback.error('操作失败', getErrorMessage(error, '点赞失败'))
  }
}

// 回复
const handleReply = () => {
  // ✅ 功能级认证检查
  if (!requireAuth('回复评论')) return;
  
  emit('reply', props.comment)
}

// 删除
const handleDelete = async () => {
  const modal = useModal()
  const confirmed = await modal.showConfirm({
    title: '删除评论',
    message: '确定要删除这条评论吗？',
    confirmText: '删除',
    confirmColor: 'error'
  })
  if (!confirmed) return

  try {
    await commentsService.delete({ id: props.comment.id })
    feedback.success('删除成功')
    emit('delete', props.comment.id)
  } catch (error: unknown) {
    feedback.error('删除失败', getErrorMessage(error, '删除评论失败'))
  }
}

// 切换回复显示
const toggleReplies = async () => {
  showReplies.value = !showReplies.value
  
  if (showReplies.value && replies.value.length === 0) {
    await loadReplies()
  }
}

// 加载回复
const loadReplies = async () => {
  if (loadingReplies.value) return
  
  loadingReplies.value = true
  try {
    const response = await commentsService.getReplies({
      id: props.comment.id,
      limit: 10,
      cursor: nextCursor.value
    })
    
    replies.value = response.items
    hasMoreReplies.value = response.pagination.has_more
    nextCursor.value = response.pagination.next_cursor
  } catch (error) {
    console.error('加载回复失败:', error)
  } finally {
    loadingReplies.value = false
  }
}

// 加载更多回复
const loadMoreReplies = async () => {
  if (loadingReplies.value || !nextCursor.value) return
  
  loadingReplies.value = true
  try {
    const response = await commentsService.getReplies({
      id: props.comment.id,
      limit: 10,
      cursor: nextCursor.value
    })
    
    replies.value.push(...response.items)
    hasMoreReplies.value = response.pagination.has_more
    nextCursor.value = response.pagination.next_cursor
  } catch (error) {
    console.error('加载更多回复失败:', error)
  } finally {
    loadingReplies.value = false
  }
}

// 回复的回复
const handleReplyToReply = (reply: CommentWithLikeStatus) => {
  emit('reply', reply)
}

// 删除回复
const handleDeleteReply = (replyId: string) => {
  replies.value = replies.value.filter(r => r.id !== replyId)
  props.comment.replies_count--
}
</script>
