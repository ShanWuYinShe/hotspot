<template>
  <div class="comment-item" :class="{ 'is-reply': isReply }">
    <!-- 评论内容 -->
    <div class="flex gap-3">
      <!-- 用户头像 -->
      <UAvatar
        :alt="comment.user.nickname"
        size="sm"
        class="flex-shrink-0"
      >
        {{ comment.user.nickname?.charAt(0) }}
      </UAvatar>

      <!-- 评论主体 -->
      <div class="flex-1 min-w-0">
        <!-- 用户信息 -->
        <div class="flex items-center gap-2 mb-1">
          <span class="font-medium text-sm">{{ comment.user.nickname }}</span>
          <span class="text-xs text-neutral-500 dark:text-neutral-400">
            {{ formatTime(comment.created_at) }}
          </span>
        </div>

        <!-- 回复对象 -->
        <div v-if="comment.reply_to_user" class="text-sm text-neutral-600 dark:text-neutral-400 mb-1">
          回复 <span class="text-neutral-600 dark:text-neutral-300">@{{ comment.reply_to_user.nickname }}</span>
        </div>

        <!-- 评论内容 -->
        <p class="text-sm text-neutral-900 dark:text-white whitespace-pre-wrap break-words">
          {{ comment.content }}
        </p>

        <!-- 操作按钮 -->
        <div class="flex items-center gap-4 mt-2">
          <!-- 点赞 -->
          <button
            @click="handleLike"
            class="flex items-center gap-1 text-xs transition-colors"
            :class="comment.liked_by_me 
              ? 'text-error-500' 
              : 'text-neutral-500 dark:text-neutral-400 hover:text-error-500'"
          >
            <UIcon 
              :name="comment.liked_by_me ? 'i-heroicons-heart-solid' : 'i-heroicons-heart'" 
              class="w-4 h-4" 
            />
            <span>{{ comment.likes_count || 0 }}</span>
          </button>

          <!-- 回复 -->
          <button
            @click="handleReply"
            class="flex items-center gap-1 text-xs text-neutral-500 dark:text-neutral-400 hover:text-neutral-600 dark:text-neutral-300 transition-colors"
          >
            <UIcon name="i-heroicons-chat-bubble-left" class="w-4 h-4" />
            <span>回复</span>
          </button>

          <!-- 查看回复 -->
          <button
            v-if="comment.replies_count > 0 && !isReply"
            @click="toggleReplies"
            class="flex items-center gap-1 text-xs text-neutral-500 dark:text-neutral-400 hover:text-neutral-600 dark:text-neutral-300 transition-colors"
          >
            <UIcon 
              :name="showReplies ? 'i-heroicons-chevron-up' : 'i-heroicons-chevron-down'" 
              class="w-4 h-4" 
            />
            <span>{{ comment.replies_count }} 条回复</span>
          </button>

          <!-- 删除（仅作者可见） -->
          <button
            v-if="canDelete"
            @click="handleDelete"
            class="flex items-center gap-1 text-xs text-neutral-500 dark:text-neutral-400 hover:text-error-500 transition-colors ml-auto"
          >
            <UIcon name="i-heroicons-trash" class="w-4 h-4" />
            <span>删除</span>
          </button>
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
      <button
        v-if="hasMoreReplies"
        @click="loadMoreReplies"
        :disabled="loadingReplies"
        class="text-sm text-neutral-600 dark:text-neutral-300 hover:underline transition-colors"
      >
        {{ loadingReplies ? '加载中...' : '查看更多回复' }}
      </button>
    </div>
  </div>
</template>

<script setup lang="ts">
import type { CommentItem as CommentItemType } from '~/services/comments.service'
import { useCommentsService } from '~/services/comments.service'

const props = defineProps<{
  comment: CommentItemType
  isReply?: boolean
}>()

const emit = defineEmits<{
  reply: [comment: CommentItemType]
  delete: [commentId: string]
  like: [commentId: string]
}>()

const { formatTime } = useFormatters()
const { requireAuth } = useRequireAuth()
const commentsService = useCommentsService()
const authStore = useAuthStore()
const toast = useToast()

// 状态
const showReplies = ref(false)
const replies = ref<CommentItemType[]>([])
const hasMoreReplies = ref(false)
const nextCursor = ref<string | undefined>()
const loadingReplies = ref(false)

// 是否可以删除（作者本人）
const canDelete = computed(() => {
  return authStore.user?.user_id === props.comment.user.user_id
})

// 点赞
const handleLike = async () => {
  // ✅ 功能级认证检查
  if (!requireAuth('点赞评论')) return;
  
  try {
    await commentsService.toggleLike({
      id: props.comment.id,
      action: props.comment.liked_by_me ? 'unlike' : 'like'
    })
    
    // 更新本地状态
    props.comment.liked_by_me = !props.comment.liked_by_me
    props.comment.likes_count += props.comment.liked_by_me ? 1 : -1
    
    emit('like', props.comment.id)
  } catch (error: any) {
    toast.add({
      title: '操作失败',
      description: error?.message || '点赞失败',
      color: 'error'
    })
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
  const confirmed = confirm('确定要删除这条评论吗？')
  if (!confirmed) return

  try {
    await commentsService.delete({ id: props.comment.id })
    toast.add({
      title: '删除成功',
      color: 'success'
    })
    emit('delete', props.comment.id)
  } catch (error: any) {
    toast.add({
      title: '删除失败',
      description: error?.message || '删除评论失败',
      color: 'error'
    })
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
const handleReplyToReply = (reply: CommentItemType) => {
  emit('reply', reply)
}

// 删除回复
const handleDeleteReply = (replyId: string) => {
  replies.value = replies.value.filter(r => r.id !== replyId)
  props.comment.replies_count--
}
</script>

<style scoped>
.comment-item {
  @apply py-3;
}

.comment-item.is-reply {
  @apply py-2;
}
</style>

