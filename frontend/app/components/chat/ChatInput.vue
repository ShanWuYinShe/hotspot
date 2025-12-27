<template>
  <UCard variant="subtle" data-hs-surface="chat-input">
    <!-- 文件上传 (使用作用域插槽) -->
    <UFileUpload
      v-slot="{ open }"
      v-model="selectedFile"
      accept="image/*,.pdf,.doc,.docx,.txt"
      @update:model-value="handleFileSelected"
    >
      <div class="flex items-center gap-2 w-full">
        <!-- 附件上传按钮 -->
        <UDropdownMenu :items="attachmentItems(open)">
          <UButton
            variant="ghost"
            color="neutral"
            icon="i-lucide-plus"
            size="sm"
            :square="true"
            class="shrink-0"
            :disabled="isLoading"
          />
        </UDropdownMenu>

        <!-- 表情按钮（放在输入框外，保证同一水平线） -->
        <UPopover v-model:open="showEmoji">
          <UButton
            :variant="showEmoji ? 'soft' : 'ghost'"
            :color="showEmoji ? 'primary' : 'neutral'"
            size="sm"
            :square="true"
            icon="i-lucide-smile"
            class="shrink-0"
            :disabled="isLoading"
          />
          <template #content>
            <div class="p-3">
              <div class="grid grid-cols-8 gap-1">
                <UButton
                  v-for="emoji in commonEmojis"
                  :key="emoji"
                  variant="ghost"
                  size="sm"
                  :square="true"
                  :label="emoji"
                  @click="insertEmoji(emoji)"
                />
              </div>
            </div>
          </template>
        </UPopover>

        <!-- 输入框区域 -->
        <div
          ref="textareaRootRef"
          class="flex-1 min-w-0"
        >
          <UTextarea
            v-model="messageInput"
            placeholder="发送消息..."
            :rows="1"
            :maxrows="3"
            :autoresize="true"
            size="sm"
            @keydown.enter.exact.prevent="handleSendMessage"
            @keydown.enter.shift.stop
          />
        </div>

        <!-- 发送按钮 -->
        <UButton
          color="primary"
          icon="i-lucide-send-horizontal"
          size="sm"
          :square="true"
          :disabled="!canSubmit"
          :loading="isLoading"
          @click="handleSendMessage"
        />
      </div>
    </UFileUpload>
  </UCard>
</template>

<script setup lang="ts">
import type { DropdownMenuItem } from '@nuxt/ui'

interface Props {
  loading?: boolean
}

interface Emits {
  'send-message': [content: string]
  'send-file': [file: File]
  'send-image': [file: File]
  'send-location': []
  'typing': [isTyping: boolean]
}

const props = withDefaults(defineProps<Props>(), {
  loading: false
})

const emit = defineEmits<Emits>()

const messageInput = ref('')
const showEmoji = ref(false)
// 文件上传 (UFileUpload)
const selectedFile = ref<File | null>(null)
const textareaRootRef = ref<HTMLElement | null>(null)

// 合并加载状态
const isLoading = computed(() => props.loading)

// 是否可以提交
const canSubmit = computed(() => {
  return messageInput.value?.trim().length > 0 && !isLoading.value
})

// 常用表情
const commonEmojis = [
  '😀', '😃', '😄', '😁', '😆', '😅', '😂', '🤣',
  '😊', '😇', '🙂', '🙃', '😉', '😌', '😍', '🥰',
  '😘', '😗', '😙', '😚', '😋', '😛', '😝', '😜',
  '🤪', '🤨', '🧐', '🤓', '😎', '🤩', '🥳', '😏',
  '😒', '😞', '😔', '😟', '😕', '🙁', '☹️', '😣',
  '😖', '😫', '😩', '🥺', '😢', '😭', '😤', '😠',
  '😡', '🤬', '🤯', '😳', '🥵', '🥶', '😱', '😨',
  '😰', '😥', '😓', '🤗', '🤔', '🤭', '🤫', '🤥'
]

// 附件菜单项
const attachmentItems = (openFile: () => void): DropdownMenuItem[][] => [
  [{
    label: '发送图片',
    icon: 'i-lucide-image',
    onSelect: () => openFile()
  }, {
    label: '发送文件',
    icon: 'i-lucide-file-text',
    onSelect: () => openFile()
  }],
  [{
    label: '分享位置',
    icon: 'i-lucide-map-pin',
    onSelect: () => emit('send-location')
  }]
]

// 监听输入状态
watch(messageInput, (newVal) => {
  emit('typing', newVal.length > 0)
})

const focusTextarea = () => {
  if (!import.meta.client) return
  const el = textareaRootRef.value?.querySelector('textarea') as HTMLTextAreaElement | null
  el?.focus()
}

// 处理消息发送
const handleSendMessage = () => {
  if (!canSubmit.value) return

  const content = messageInput.value.trim()
  emit('send-message', content)
  messageInput.value = '' // 清空输入框

  // UX：发送后保持输入焦点（高频发消息不需要反复点击输入框）
  nextTick(() => focusTextarea())
}

// 插入表情
const insertEmoji = (emoji: string) => {
  messageInput.value += emoji
  showEmoji.value = false
}

// UFileUpload 回调 - 处理文件选择
const handleFileSelected = (file: File | null | undefined) => {
  if (file) {
    if (file.type.startsWith('image/')) {
      emit('send-image', file)
    } else {
      emit('send-file', file)
    }
    selectedFile.value = null
  }
}

</script>