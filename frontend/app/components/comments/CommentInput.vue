<template>
  <div class="comment-input">
    <!-- 回复提示 -->
    <div v-if="replyTo" class="flex items-center justify-between p-2 mb-2 bg-primary-50 dark:bg-primary-950/20 rounded-lg">
      <span class="text-sm text-neutral-700 dark:text-neutral-300">
        回复 <span class="font-medium">@{{ replyTo.user.nickname }}</span>
      </span>
      <UButton
        variant="ghost"
        size="xs"
        icon="i-heroicons-x-mark"
        @click="handleCancelReply"
      />
    </div>

    <!-- 输入框 -->
    <div class="flex gap-3">
      <!-- 当前用户头像 -->
      <UAvatar
        v-if="currentUser"
        :alt="currentUser.nickname"
        size="sm"
        class="flex-shrink-0"
      >
        {{ currentUser.nickname?.charAt(0) }}
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
          <span class="text-xs text-neutral-500 dark:text-neutral-400">
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
import type { CommentItem } from '~/services/comments.service'
import { useCommentsService } from '~/services/comments.service'

const props = defineProps<{
  momentId: string
  replyTo?: CommentItem
}>()

const emit = defineEmits<{
  submit: [comment: CommentItem]
  cancelReply: []
}>()

const commentsService = useCommentsService()
const authStore = useAuthStore()
const toast = useToast()

// 状态
const content = ref('')
const submitting = ref(false)

// 当前用户
const currentUser = computed(() => authStore.user)

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
    const response = await commentsService.create({
      moment_id: props.momentId,
      content: content.value.trim(),
      parent_id: props.replyTo?.id
    })
    
    // 构造新评论对象（用于本地显示）
    const newComment: CommentItem = {
      id: response.id,
      user: currentUser.value!,
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
  } catch (error: any) {
    toast.add({
      title: '发送失败',
      description: error?.message || '评论发送失败',
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

<style scoped>
.comment-input {
  @apply w-full;
}
</style>

