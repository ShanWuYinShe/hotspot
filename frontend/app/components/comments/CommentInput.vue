<template>
  <div class="w-full">
    <!-- 回复提示卡片 -->
    <div
      v-if="replyTo"
      class="flex items-center justify-between gap-2 p-3 bg-neutral-50/80 dark:bg-neutral-900/55 rounded-t-3xl border-b border-neutral-100 dark:border-neutral-800"
    >
      <div class="flex items-center gap-1.5 text-sm text-neutral-500 dark:text-neutral-400">
        <UIcon name="i-lucide-corner-up-left" class="size-3" />
        <span>回复</span>
        <span class="font-semibold text-neutral-900 dark:text-white">@{{ replyTo.user.nickname }}</span>
      </div>
      <UButton
        variant="ghost"
        color="neutral"
        icon="i-lucide-x"
        size="xs"
        @click="$emit('cancelReply')"
      />
    </div>

    <!-- 输入框卡片 -->
    <div
      class="border border-neutral-200 dark:border-neutral-800 bg-white dark:bg-neutral-900 overflow-hidden"
      :class="[
        !!replyTo ? 'rounded-b-3xl border-t-0' : 'rounded-3xl'
      ]"
    >
      <UTextarea
        v-model="content"
        :placeholder="placeholder"
        size="sm"
        :rows="1"
        :maxrows="5"
        autoresize
        variant="none"
        @keydown.meta.enter="handleSubmit"
        @keydown.ctrl.enter="handleSubmit"
      />
      
      <div class="flex items-center justify-between px-3 pb-3">
        <!-- 工具栏 (表情等) - 预留位置 -->
        <div class="flex gap-1">
          <UButton variant="ghost" color="neutral" icon="i-lucide-smile" size="xs" disabled />
          <UButton variant="ghost" color="neutral" icon="i-lucide-image" size="xs" disabled />
        </div>

        <div class="flex items-center gap-3">
          <span v-if="content.length > 0" class="text-sm text-neutral-500 dark:text-neutral-400">{{ content.length }}/500</span>
          <UButton
            size="xs"
            color="primary"
            variant="solid"
            :disabled="!canSubmit"
            :loading="submitting"
            @click="handleSubmit"
          >
            {{ replyTo ? '回复' : '发送' }}
          </UButton>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import type { CommentListItem, UserInfo } from '~/types/api'
import { useCommentsService } from '~/services/comments.service'

interface CommentWithLikeStatus extends CommentListItem { liked_by_me?: boolean; reply_to_user?: UserInfo }

const props = defineProps<{ momentId: string; replyTo?: CommentWithLikeStatus }>()
const emit = defineEmits<{
  submit: [comment: CommentWithLikeStatus]
  cancelReply: []
}>()

const commentsService = useCommentsService()
const authStore = useAuthStore()
const feedback = useFeedbackStore()

const content = ref('')
const submitting = ref(false)

const placeholder = computed(() => props.replyTo ? `回复 @${props.replyTo.user.nickname}...` : '写下你的评论...')
const canSubmit = computed(() => content.value.trim().length > 0 && !submitting.value)

const handleSubmit = async () => {
  if (!canSubmit.value) return
  submitting.value = true
  try {
    const res = await commentsService.create({
      moment_id: props.momentId,
      content: content.value.trim(),
      parent_id: props.replyTo?.comment_id
    })
    
    emit('submit', {
      comment_id: res.comment_id,
      user: { ...authStore.user!, user_type: 'normal' },
      content: content.value.trim(),
      parent_id: props.replyTo?.comment_id,
      reply_to_user: props.replyTo?.user,
      created_at: new Date().toISOString(),
      likes_count: 0,
      replies_count: 0,
      liked_by_me: false
    })
    content.value = ''
    feedback.success(props.replyTo ? '回复成功' : '评论成功')
  } catch (e) {
    feedback.error('发送失败', getErrorMessage(e))
  } finally {
    submitting.value = false
  }
}
</script>
