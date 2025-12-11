<template>
  <div class="w-full">
    <!-- 回复提示 -->
    <div v-if="replyTo" class="flex items-center justify-between p-2 mb-2 bg-primary-light-20 rounded-lg">
      <span class="text-sm text-secondary-strong">
        回复 <span class="font-medium">@{{ replyTo.user.nickname }}</span>
      </span>
      <UButton
        variant="ghost"
        size="sm"
        icon="i-heroicons-x-mark"
        @click="handleCancelReply"
      />
    </div>

    <!-- 输入框 -->
    <div class="flex gap-3">
      <!-- 当前用户头像 -->
      <UAvatar
        v-if="authStore.user"
        :alt="authStore.user.nickname"
        size="sm"
        class="shrink-0"
      >
        {{ authStore.user.nickname?.charAt(0) }}
      </UAvatar>

      <!-- 输入区域 -->
      <div class="flex-1">
        <UTextarea
          v-model="content"
          :placeholder="placeholder"
          :rows="3"
          :maxlength="500"
          @keydown.meta.enter="handleSubmit"
          @keydown.ctrl.enter="handleSubmit"
        />
        
        <!-- 字数统计和操作按钮 -->
        <div class="flex items-center justify-between mt-2">
          <span class="text-xs text-secondary-content">
            {{ content.length }} / 500
          </span>
          
          <div class="flex gap-2">
            <UButton
              v-if="replyTo"
              variant="ghost"
              size="sm"
              @click="handleCancelReply"
            >
              取消
            </UButton>
            <UButton
              size="sm"
              :disabled="!canSubmit"
              :loading="submitting"
              @click="handleSubmit"
            >
              {{ replyTo ? '回复' : '评论' }}
            </UButton>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import type { CommentListItem } from '~/types/api'
import { useCommentsService } from '~/services/comments.service'
import { getErrorMessage } from '~/utils/error'

// 扩展类型：包含前端本地管理的点赞状态
interface CommentWithLikeStatus extends CommentListItem {
  liked_by_me?: boolean
}

const props = defineProps<{
  momentId: string
  replyTo?: CommentWithLikeStatus
}>()

const emit = defineEmits<{
  submit: [comment: CommentWithLikeStatus]
  cancelReply: []
}>()

const commentsService = useCommentsService()
// 使用水合安全的 store 访问
const authStore = useAuthStore()
const toast = useToast()

// 状态
const content = ref('')
const submitting = ref(false)

// 占位符
const placeholder = computed(() => {
  if (props.replyTo) {
    return `回复 @${props.replyTo.user.nickname}...`
  }
  return '写下你的评论...'
})

// 是否可以提交
const canSubmit = computed(() => {
  return content.value.trim().length > 0 && !submitting.value
})

// 提交评论
const handleSubmit = async () => {
  if (!canSubmit.value) return
  
  submitting.value = true
  try {
    const response = await commentsService.add({
      moment_id: props.momentId,
      content: content.value.trim(),
      parent_id: props.replyTo?.id
    })
    
    // 构造新评论对象（用于本地显示）
    const newComment: CommentWithLikeStatus = {
      id: response.id,
      user: {
        ...authStore.user!,
        user_type: 'normal' as const
      },
      content: content.value.trim(),
      parent_id: props.replyTo?.id,
      reply_to_user: props.replyTo?.user,
      created_at: new Date().toISOString(),
      likes_count: 0,
      replies_count: 0,
      liked_by_me: false
    }
    
    emit('submit', newComment)
    
    // 清空输入
    content.value = ''
    
    toast.add({
      title: props.replyTo ? '回复成功' : '评论成功',
      color: 'success'
    })
  } catch (error: unknown) {
    toast.add({
      title: '发送失败',
      description: getErrorMessage(error, '评论发送失败'),
      color: 'error'
    })
  } finally {
    submitting.value = false
  }
}

// 取消回复
const handleCancelReply = () => {
  emit('cancelReply')
}
</script>
