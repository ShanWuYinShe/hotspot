<template>
  <div class="p-4 bg-white/80 dark:bg-neutral-900/70 backdrop-blur border-t">
    <UChatPrompt
      v-model="messageInput"
      placeholder="发送消息..."
      :loading="sending"
      variant="soft"
      @submit="handleSendMessage"
    >
      <template #trailing>
        <div class="flex items-center gap-1">
          <!-- 表情按钮 -->
          <UTooltip text="表情">
            <UButton
              variant="ghost"
              size="xs"
              icon="i-heroicons-face-smile"
              @click="toggleEmoji"
              :disabled="sending"
            />
          </UTooltip>

          <!-- 文件上传 -->
          <UTooltip text="发送文件">
            <UButton
              variant="ghost"
              size="xs"
              icon="i-heroicons-paper-clip"
              @click="triggerFileUpload"
              :disabled="sending"
            />
          </UTooltip>

          <!-- 位置分享 -->
          <UTooltip text="分享位置">
            <UButton
              variant="ghost"
              size="xs"
              icon="i-heroicons-map-pin"
              @click="$emit('send-location')"
              :disabled="sending"
            />
          </UTooltip>
        </div>
      </template>
    </UChatPrompt>

    <!-- 隐藏的文件输入 -->
    <input
      ref="fileInputRef"
      type="file"
      class="hidden"
      accept="image/*,.pdf,.doc,.docx,.txt"
      @change="handleFileSelect"
    />

    <!-- 表情选择器 -->
    <UPopover v-model:open="showEmoji" :popper="{ placement: 'top-end' }">
      <template #panel>
        <div class="p-4 bg-white dark:bg-neutral-800 rounded-lg shadow-lg border">
          <div class="grid grid-cols-8 gap-2">
            <button
              v-for="emoji in commonEmojis"
              :key="emoji"
              class="text-xl hover:bg-neutral-100 dark:hover:bg-neutral-700 rounded p-1 transition-colors"
              @click="insertEmoji(emoji)"
            >
              {{ emoji }}
            </button>
          </div>
        </div>
      </template>
    </UPopover>
  </div>
</template>

<script setup lang="ts">

interface Emits {
  'send-message': [content: string]
  'send-file': [file: File]
  'send-location': []
}

const emit = defineEmits<Emits>()

const messageInput = ref('')
const sending = ref(false)
const showEmoji = ref(false)
const fileInputRef = ref<HTMLInputElement>()

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

// 处理消息发送
const handleSendMessage = async (event: Event) => {
  event.preventDefault()
  const content = messageInput.value.trim()
  if (!content || sending.value) return

  sending.value = true
  try {
    emit('send-message', content)
    messageInput.value = '' // 清空输入框
  } finally {
    sending.value = false
  }
}

// 切换表情面板
const toggleEmoji = () => {
  showEmoji.value = !showEmoji.value
}

// 插入表情
const insertEmoji = (emoji: string) => {
  messageInput.value += emoji
  showEmoji.value = false
}

// 触发文件上传
const triggerFileUpload = () => {
  fileInputRef.value?.click()
}

// 处理文件选择
const handleFileSelect = (event: Event) => {
  const target = event.target as HTMLInputElement
  const file = target.files?.[0]
  if (file) {
    emit('send-file', file)
    target.value = '' // 清空文件输入
  }
}

// 表情面板由 UPopover 自动管理，无需手动处理点击外部
</script>
