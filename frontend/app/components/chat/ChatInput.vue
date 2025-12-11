<template>
  <div class="p-4 bg-glass-neutral backdrop-blur border-t">
    <ChatPrompt
      v-model="messageInput"
      placeholder="发送消息..."
      :loading="sending"
      variant="soft"
      @submit="handleSendMessage"
    >
      <template #trailing>
        <div class="flex items-center gap-1">
          <!-- 表情选择器 -->
          <UPopover v-model:open="showEmoji">
            <UTooltip text="表情">
              <UButton
                variant="ghost"
                size="xs"
                icon="i-heroicons-face-smile"
                :disabled="sending"
              />
            </UTooltip>
            <template #content>
              <div class="p-3 w-72">
                <div class="grid grid-cols-8 gap-1">
                  <UButton
                    v-for="emoji in commonEmojis"
                    :key="emoji"
                    variant="ghost"
                    size="xs"
                    :label="emoji"
                    class="text-lg"
                    @click="insertEmoji(emoji)"
                  />
                </div>
              </div>
            </template>
          </UPopover>

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
    </ChatPrompt>

    <!-- 文件上传 (使用 UFileUpload) -->
    <UFileUpload
      v-slot="{ open }"
      v-model="selectedFile"
      accept="image/*,.pdf,.doc,.docx,.txt"
      class="hidden"
      @update:model-value="handleFileSelected"
    >
      <button ref="fileUploadBtn" class="hidden" @click="open()" />
    </UFileUpload>

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
// 文件上传 (UFileUpload)
const selectedFile = ref<File | null>(null)
const fileUploadBtn = ref<HTMLButtonElement | null>(null)

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
const handleSendMessage = async () => {
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

// 插入表情
const insertEmoji = (emoji: string) => {
  messageInput.value += emoji
  showEmoji.value = false
}

// 触发文件上传
const triggerFileUpload = () => {
  fileUploadBtn.value?.click()
}

// UFileUpload 回调 - 处理文件选择
const handleFileSelected = (file: File | null | undefined) => {
  if (file) {
    emit('send-file', file)
    // 重置选择
    selectedFile.value = null
  }
}

</script>
