<template>
  <div class="comment-list">
    <!-- 评论统计 -->
    <div class="flex items-center justify-between mb-4">
      <h3 class="text-lg font-semibold">
        评论 <span class="text-neutral-500">({{ totalCount }})</span>
      </h3>
      <UButton
        variant="ghost"
        size="sm"
        icon="i-heroicons-arrow-path"
        @click="refresh"
        :loading="loading"
      >
        刷新
      </UButton>
    </div>

    <!-- 评论输入框 -->
    <CommentInput
      :moment-id="momentId"
      :reply-to="replyTo"
      @submit="handleCommentSubmit"
      @cancel-reply="replyTo = undefined"
    />

    <!-- 评论列表 -->
    <div class="mt-6 space-y-4">
      <!-- 加载状态 -->
      <div v-if="loading && comments.length === 0" class="text-center py-8">
        <UIcon name="i-heroicons-arrow-path" class="w-6 h-6 animate-spin text-primary-500 mx-auto" />
        <p class="text-sm text-neutral-500 mt-2">加载中...</p>
      </div>

      <!-- 空状态 -->
      <div v-else-if="comments.length === 0" class="text-center py-8">
        <UIcon name="i-heroicons-chat-bubble-left-ellipsis" class="w-12 h-12 text-neutral-400 mx-auto mb-3" />
        <p class="text-neutral-500 text-sm">还没有评论，来抢沙发吧！</p>
      </div>

      <!-- 评论项 -->
      <CommentItem
        v-for="comment in comments"
        :key="comment.id"
        :comment="comment"
        @reply="handleReply"
        @delete="handleDelete"
        @like="handleLike"
      />

      <!-- 加载更多 -->
      <div v-if="hasMore" class="text-center py-4">
        <UButton
          variant="ghost"
          @click="loadMore"
          :loading="loadingMore"
        >
          {{ loadingMore ? '加载中...' : '加载更多' }}
        </UButton>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import type { CommentItem as CommentItemType } from '~/services/comments.service'
import { useCommentsService } from '~/services/comments.service'
import CommentItem from './CommentItem.vue'
import CommentInput from './CommentInput.vue'

const props = defineProps<{
  momentId: string
}>()

const commentsService = useCommentsService()
const toast = useToast()

// 状态
const comments = ref<CommentItemType[]>([])
const loading = ref(false)
const loadingMore = ref(false)
const hasMore = ref(false)
const nextCursor = ref<string | undefined>()
const totalCount = ref(0)
const replyTo = ref<CommentItemType | undefined>()

// 加载评论列表
const loadComments = async () => {
  loading.value = true
  try {
    const response = await commentsService.getList({
      moment_id: props.momentId,
      limit: 20
    })
    
    comments.value = response.items
    hasMore.value = response.pagination.has_more
    nextCursor.value = response.pagination.next_cursor
    totalCount.value = response.items.length
  } catch (error: any) {
    toast.add({
      title: '加载失败',
      description: error?.message || '加载评论失败',
      color: 'error'
    })
  } finally {
    loading.value = false
  }
}

// 加载更多
const loadMore = async () => {
  if (loadingMore.value || !nextCursor.value) return
  
  loadingMore.value = true
  try {
    const response = await commentsService.getList({
      moment_id: props.momentId,
      limit: 20,
      cursor: nextCursor.value
    })
    
    comments.value.push(...response.items)
    hasMore.value = response.pagination.has_more
    nextCursor.value = response.pagination.next_cursor
  } catch (error: any) {
    toast.add({
      title: '加载失败',
      description: error?.message || '加载更多评论失败',
      color: 'error'
    })
  } finally {
    loadingMore.value = false
  }
}

// 刷新
const refresh = () => {
  nextCursor.value = undefined
  loadComments()
}

// 处理评论提交
const handleCommentSubmit = (newComment: CommentItemType) => {
  // 如果是回复，不添加到顶层列表
  if (!replyTo.value) {
    comments.value.unshift(newComment)
    totalCount.value++
  } else {
    // 找到被回复的评论，增加回复数
    const parentComment = comments.value.find(c => c.id === replyTo.value!.id)
    if (parentComment) {
      parentComment.replies_count++
    }
  }
  
  replyTo.value = undefined
}

// 处理回复
const handleReply = (comment: CommentItemType) => {
  replyTo.value = comment
  // 滚动到评论输入框
  const inputElement = document.querySelector('.comment-input')
  inputElement?.scrollIntoView({ behavior: 'smooth', block: 'center' })
}

// 处理删除
const handleDelete = (commentId: string) => {
  comments.value = comments.value.filter(c => c.id !== commentId)
  totalCount.value--
}

// 处理点赞
const handleLike = (commentId: string) => {
  // 点赞状态已在 CommentItem 中更新
}

// 组件挂载时加载评论
onMounted(() => {
  loadComments()
})

// 暴露刷新方法
defineExpose({
  refresh
})
</script>

<style scoped>
.comment-list {
  @apply w-full;
}
</style>

