<template>
  <NDialog
    v-model="visible"
    title="回复评论"
    width="500px"
    :show-close="true"
    :close-on-click-modal="false"
    class="comment-reply-modal"
  >
    <!-- 被回复的评论 -->
    <div v-if="replyTo" class="reply-to-comment">
      <div class="comment-header">
        <div class="comment-author">
          <div class="author-avatar">
            <img v-if="replyTo.author.avatar" :src="replyTo.author.avatar" :alt="replyTo.author.name" />
            <span v-else>{{ replyTo.author.name.charAt(0) }}</span>
          </div>
          <span class="author-name">{{ replyTo.author.name }}</span>
        </div>
        <span class="comment-time">{{ formatTime(replyTo.created_at) }}</span>
      </div>
      <div class="comment-content">{{ replyTo.content }}</div>
    </div>

    <!-- 回复输入框 -->
    <div class="reply-form">
      <NInput
        v-model="replyContent"
        type="textarea"
        :rows="4"
        :placeholder="`回复 @${replyTo?.author.name || ''}...`"
        :maxlength="500"
        show-word-limit
        resize="none"
        ref="replyInput"
      />
      
      <!-- 操作按钮 -->
      <div class="reply-actions">
        <NButton @click="handleCancel">
          取消
        </NButton>
        <NButton 
          type="primary" 
          :loading="submitting"
          :disabled="!replyContent.trim()"
          @click="handleSubmit"
        >
          发送回复
        </NButton>
      </div>
    </div>
  </NDialog>
</template>

<script setup lang="ts">
import { ref, computed, watch, nextTick } from 'vue'
import { NDialog, NInput, NButton, useMessage } from 'naive-ui'
import type { Comment } from '~/types'

// API实例
const message = useMessage()

interface Props {
  modelValue: boolean
  replyTo: Comment | null
}

interface Emits {
  (e: 'update:modelValue', value: boolean): void
  (e: 'submit', content: string, replyTo: Comment): void
}

const props = defineProps<Props>()
const emit = defineEmits<Emits>()

// 响应式数据
const replyContent = ref('')
const submitting = ref(false)
const replyInput = ref()

// 计算属性
const visible = computed({
  get: () => props.modelValue,
  set: (value: boolean) => emit('update:modelValue', value)
})

// 监听弹窗打开，自动聚焦输入框
watch(visible, async (newVal) => {
  if (newVal) {
    replyContent.value = ''
    await nextTick()
    replyInput.value?.focus()
  }
})

// 方法
const formatTime = (dateString: string): string => {
  const date = new Date(dateString)
  const now = new Date()
  const diff = now.getTime() - date.getTime()
  const minutes = Math.floor(diff / 60000)
  
  if (minutes < 1) return '刚刚'
  if (minutes < 60) return `${minutes}分钟前`
  
  const hours = Math.floor(minutes / 60)
  if (hours < 24) return `${hours}小时前`
  
  const days = Math.floor(hours / 24)
  if (days < 7) return `${days}天前`
  
  return date.toLocaleDateString()
}

const handleCancel = () => {
  visible.value = false
  replyContent.value = ''
}

const handleSubmit = async () => {
  if (!replyContent.value.trim() || !props.replyTo) {
    return
  }

  submitting.value = true
  try {
    emit('submit', replyContent.value.trim(), props.replyTo)
    message.success('回复发送成功')
    visible.value = false
    replyContent.value = ''
  } catch (error) {
    console.error('发送回复失败:', error)
    message.error('发送回复失败，请重试')
  } finally {
    submitting.value = false
  }
}
</script>

<style scoped lang="scss">
.comment-reply-modal {
  .el-dialog__body {
    padding: 1.5rem;
  }
}

.reply-to-comment {
  background: var(--color-bg-tertiary);
  border-radius: 8px;
  padding: 1rem;
  margin-bottom: 1rem;
  border-left: 3px solid var(--color-primary-500);

  .comment-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    margin-bottom: 0.5rem;

    .comment-author {
      display: flex;
      align-items: center;
      gap: 0.5rem;

      .author-avatar {
        width: 24px;
        height: 24px;
        border-radius: 50%;
        background: var(--color-primary-300);
        display: flex;
        align-items: center;
        justify-content: center;
        color: white;
        font-size: 0.75rem;
        font-weight: 600;
        overflow: hidden;
        flex-shrink: 0;

        img {
          width: 100%;
          height: 100%;
          object-fit: cover;
        }
      }

      .author-name {
        font-size: 0.85rem;
        font-weight: 600;
        color: var(--color-text-primary);
      }
    }

    .comment-time {
      font-size: 0.75rem;
      color: var(--color-text-secondary);
    }
  }

  .comment-content {
    font-size: 0.85rem;
    line-height: 1.5;
    color: var(--color-text-primary);
  }
}

.reply-form {
  .el-textarea {
    margin-bottom: 1rem;
  }

  .reply-actions {
    display: flex;
    justify-content: flex-end;
    gap: 0.75rem;
  }
}

// 深色模式适配
.dark {
  .reply-to-comment {
    background: var(--color-bg-secondary);
  }
}
</style>
