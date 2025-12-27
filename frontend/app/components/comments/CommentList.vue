<template>
  <div class="w-full">
    <!-- 评论统计 -->
    <div class="flex items-center justify-between mb-4">
      <div class="flex items-center gap-2 min-w-0">
        <span class="text-lg font-semibold truncate">评论</span>
        <UBadge color="neutral" variant="soft" size="xs">{{ totalCount }}</UBadge>
      </div>
      <UButton
        variant="ghost"
        size="sm"
        icon="i-lucide-refresh-cw"
        @click="refresh"
        :loading="loading"
      >
        刷新
      </UButton>
    </div>

    <!-- 评论输入框 -->
    <CommentInput
      ref="commentInputRef"
      :moment-id="momentId"
      :reply-to="replyTo"
      @submit="handleCommentSubmit"
      @cancel-reply="replyTo = undefined"
    />

    <!-- 评论列表 -->
    <div class="mt-6 space-y-4">
      <!-- 加载状态 -->
      <UEmpty
        v-if="loading && comments.length === 0"
        icon="i-lucide-refresh-cw"
        title="加载中..."
        description="请稍候"
        class="py-8"
      />

      <!-- 空状态 -->
      <UEmpty
        v-else-if="comments.length === 0"
        icon="i-lucide-messages-square"
        title="还没有评论"
        description="来抢沙发吧！"
        class="py-8"
      />

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
import type { CommentListItem } from '~/types/api'
import { useCommentsService } from '~/services/comments.service'
import CommentItem from './CommentItem.vue'
import CommentInput from './CommentInput.vue'

// 扩展类型：包含前端本地管理的点赞状态
interface CommentWithLikeStatus extends CommentListItem {
  liked_by_me?: boolean
}

const props = defineProps<{
  momentId: string
  initialTotal?: number  // 来自父组件的初始评论总数
}>()

const commentsService = useCommentsService()
const feedback = useFeedbackStore()

// 状态
const comments = ref<CommentWithLikeStatus[]>([])
const loading = ref(false)
const loadingMore = ref(false)
const hasMore = ref(false)
const nextCursor = ref<string | undefined>()
const totalCount = ref(props.initialTotal || 0)  // 使用 prop 传入的初始值
const replyTo = ref<CommentWithLikeStatus | undefined>()
const commentInputRef = ref<InstanceType<typeof CommentInput> | null>(null)

// 监听 initialTotal 的变化
watch(() => props.initialTotal, (newTotal) => {
  if (newTotal !== undefined) {
    totalCount.value = newTotal
  }
})

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
    // ❌ 不要使用 items.length 作为总数！这只是当前分页的数量
    // totalCount 应该由父组件通过 initialTotal prop 传入，或者通过新增/删除评论来增减
  } catch (error: unknown) {
    feedback.error('加载失败', getErrorMessage(error, '加载评论失败'))
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
  } catch (error: unknown) {
    feedback.error('加载失败', getErrorMessage(error, '加载更多评论失败'))
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
const handleCommentSubmit = (newComment: CommentWithLikeStatus) => {
  // 如果是回复，不添加到顶层列表
  if (!replyTo.value) {
    comments.value.unshift(newComment)
    totalCount.value++
    // 发送总数更新事件
    emit('update:total', totalCount.value)
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
const handleReply = (comment: CommentWithLikeStatus) => {
  replyTo.value = comment
  // 使用模板引用滚动到评论输入框
  commentInputRef.value?.$el?.scrollIntoView({ behavior: 'smooth', block: 'center' })
}

// 处理删除
const handleDelete = (commentId: string) => {
  comments.value = comments.value.filter(c => c.id !== commentId)
  totalCount.value--
  // 发送总数更新事件
  emit('update:total', totalCount.value)
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

const emit = defineEmits<{
  (e: 'update:total', count: number): void
}>()
</script>

