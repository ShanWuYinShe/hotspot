<template>
  <div class="flex flex-col h-full">
    <!-- 聊天头部 -->
    <PrivateChatHeader
      :user-info="userInfo"
      :is-online="isUserOnline"
      :last-online="userProfile?.last_online"
      :menu-items="chatMenuItems"
      @go-back="router.back()"
      @toggle-search="toggleSearch"
      @view-profile="viewUserProfile"
      @clear-history="clearChatHistory"
      @block-user="blockUser"
    />

    <!-- 搜索栏 -->
    <Transition
      enter-active-class="transition-all duration-200 ease-out"
      enter-from-class="opacity-0 -translate-y-2"
      enter-to-class="opacity-100 translate-y-0"
      leave-active-class="transition-all duration-150 ease-in"
      leave-from-class="opacity-100 translate-y-0"
      leave-to-class="opacity-0 -translate-y-2"
    >
      <div
        v-if="showSearch"
        class="z-20 absolute top-[60px] w-full"
      >
        <UCard variant="subtle" data-hs-surface="chat-search">
          <UInput
            v-model="searchQuery"
            placeholder="搜索聊天记录..."
            icon="i-lucide-search"
            class="w-full"
            size="sm"
            :autofocus="true"
            @keyup.escape="toggleSearch"
          >
            <template #trailing>
              <div class="flex items-center gap-1">
                <UButton
                  variant="ghost"
                  size="sm"
                  :square="true"
                  icon="i-lucide-chevron-up"
                  :disabled="!hasSearchHits"
                  @click="jumpToPrevHit"
                />
                <UButton
                  variant="ghost"
                  size="sm"
                  :square="true"
                  icon="i-lucide-chevron-down"
                  :disabled="!hasSearchHits"
                  @click="jumpToNextHit"
                />
                <UBadge v-if="searchQuery">
                  {{ hasSearchHits ? `${activeSearchHitIndexDisplay}/${searchHitsCount}` : '0/0' }}
                </UBadge>
                <UButton
                  variant="ghost"
                  size="sm"
                  :square="true"
                  icon="i-lucide-x"
                  @click="clearSearch"
                />
              </div>
            </template>
          </UInput>

          <UAlert
            v-if="searchQuery && searchHitsCount === 0"
            color="neutral"
            variant="soft"
            icon="i-lucide-search-x"
            title="没有找到相关消息"
            class="mt-2"
          />
          <UAlert
            v-else-if="searchQuery"
            color="neutral"
            variant="soft"
            icon="i-lucide-search"
            :title="`找到 ${searchHitsCount} 条相关消息`"
            class="mt-2"
          />
        </UCard>
      </div>
    </Transition>

    <!-- 聊天消息区域 -->
    <div class="flex-1 min-h-0 relative overflow-hidden">
      <!-- 错误状态：与自己聊天 -->
      <div v-if="chatError === 'self'" class="h-full flex items-center justify-center p-4">
        <UEmpty
          icon="i-lucide-triangle-alert"
          title="无法创建私聊"
          description="不能与自己创建私聊"
        >
          <template #actions>
            <UButton variant="outline" color="neutral" @click="router.back()">
              返回
            </UButton>
          </template>
        </UEmpty>
      </div>
      
      <!-- 错误状态：加载失败 -->
      <div v-else-if="chatError === 'load_failed'" class="h-full flex items-center justify-center p-4">
        <UEmpty
          icon="i-lucide-circle-x"
          title="加载失败"
          description="无法加载聊天数据"
        >
          <template #actions>
            <div class="flex items-center gap-2">
              <UButton variant="outline" color="neutral" @click="router.back()">
                返回
              </UButton>
              <UButton color="primary" @click="loadMessages">
                重试
              </UButton>
            </div>
          </template>
        </UEmpty>
      </div>
      
      <!-- 加载中状态 -->
      <div v-else-if="!privateChatId" class="h-full flex items-center justify-center p-4">
        <UEmpty
          icon="i-lucide-messages-square"
          title="正在加载聊天..."
          description="请稍候"
        />
      </div>
      
      <!-- 消息列表 -->
      <ChatMessages
        v-else
        ref="chatMessagesRef"
        :messages="formattedMessages"
        :status="chatStatus"
        :should-auto-scroll="true"
        :should-scroll-to-bottom="true"
        :highlight-message-id="highlightMessageId || undefined"
        :user="{
          avatar: { 
            alt: currentUser?.nickname,
            src: currentUser?.avatar_url,
            fallback: currentUser?.nickname?.charAt(0) || 'U'
          },
          variant: 'solid',
          side: 'right'
        }"
        :assistant="{
          avatar: { 
            alt: userInfo?.nickname,
            src: userInfo?.avatar_url,
            fallback: userInfo?.nickname?.charAt(0) || 'U'
          },
          variant: 'soft',
          side: 'left'
        }"
        class="h-full"
      >
        <!-- 空消息状态 -->
        <template #empty>
          <div class="h-full flex items-center justify-center p-4">
            <UEmpty
              icon="i-lucide-messages-square"
              title="开始聊天吧"
              :description="`向 ${userInfo?.nickname || '对方'} 发送第一条消息，开启你们的对话`"
            />
          </div>
        </template>
        
        <!-- 自定义消息内容 -->
        <template #content="{ message }">
          <div class="space-y-2">
            <!-- 文本消息 -->
            <div v-if="(message as ChatMessage).type === 'text'" class="text-sm leading-relaxed">
              <template v-for="(seg, i) in getHighlightedSegments(String((message as ChatMessage).content || ''), searchQuery)" :key="i">
                <span :class="seg.isMatch ? 'hs-search-hit' : ''">{{ seg.text }}</span>
              </template>
            </div>
            
            <!-- 图片消息 -->
            <div v-else-if="(message as ChatMessage).type === 'image'" class="max-w-xs">
              <NuxtImg
                :src="String((message as ChatMessage).content || '')"
                :alt="(message as ChatMessage).alt || '图片'"
                class="rounded-xl cursor-pointer hover:opacity-95 transition-opacity max-h-64 object-contain ring-1 ring-gray-200 dark:ring-gray-800"
                @click="previewImage(String((message as ChatMessage).content || ''))"
              />
            </div>
            
            <!-- 文件消息 -->
            <UCard v-else-if="(message as ChatMessage).type === 'file'" variant="subtle" class="max-w-xs">
              <div class="flex items-center gap-3">
                <UBadge color="info" variant="soft" size="lg" square icon="i-lucide-file-text" />
                <div class="flex-1 min-w-0">
                  <p class="font-medium truncate text-sm">{{ (message as ChatMessage).fileName }}</p>
                  <UBadge>
                    {{ formatFileSize((message as ChatMessage).fileSize || 0) }}
                  </UBadge>
                </div>
                <UButton
                  variant="ghost"
                  size="sm"
                  :square="true"
                  icon="i-lucide-download"
                  @click="downloadFile(String((message as ChatMessage).content || ''), (message as ChatMessage).fileName || '')"
                />
              </div>
            </UCard>
            
            <!-- 位置消息 -->
            <UCard v-else-if="(message as ChatMessage).type === 'location'" variant="subtle">
              <div class="flex items-center gap-2 mb-2">
                <UBadge color="info" variant="soft" size="lg" square icon="i-lucide-map-pin" />
                <span class="font-medium text-sm">位置信息</span>
              </div>
              <UAlert
                color="neutral"
                variant="soft"
                icon="i-lucide-map-pin"
                :title="String((message as ChatMessage).address || '')"
              />
              <UButton
                variant="soft"
                size="xs"
                leading-icon="i-lucide-map"
                class="mt-2"
                @click="openLocation((message as ChatMessage).latitude || 0, (message as ChatMessage).longitude || 0)"
              >
                查看位置
              </UButton>
            </UCard>
            
            <!-- 系统消息 -->
            <div v-else-if="(message as ChatMessage).type === 'system'" class="text-center py-2">
              <UBadge icon="i-lucide-info">
                {{ (message as ChatMessage).content }}
              </UBadge>
            </div>
          </div>
        </template>

        <!-- 消息操作 -->
        <template #actions="{ message }">
          <div v-if="message.role === 'user'" class="flex items-center gap-0.5 transition-opacity hover:opacity-80">
            <UTooltip text="回复">
              <UButton
                variant="ghost"
                size="xs"
                icon="i-lucide-corner-up-left"
                @click="replyToMessage(message)"
              />
            </UTooltip>
            <UTooltip text="编辑">
              <UButton
                variant="ghost"
                size="xs"
                icon="i-lucide-pencil"
                @click="editMessage(message)"
              />
            </UTooltip>
            <UTooltip text="删除">
              <UButton
                variant="ghost"
                size="xs"
                icon="i-lucide-trash-2"
                color="error"
                @click="deleteMessage(message)"
              />
            </UTooltip>
          </div>
        </template>

        <!-- 对方正在输入指示器 -->
        <template #indicator v-if="otherUserTyping">
          <div class="px-4 py-2">
            <UCard variant="soft" class="w-fit">
              <div class="flex items-center gap-2">
                <div class="flex items-center gap-1">
                  <div class="hs-typing-dot animate-bounce [animation-delay:0ms]" />
                  <div class="hs-typing-dot animate-bounce [animation-delay:150ms]" />
                  <div class="hs-typing-dot animate-bounce [animation-delay:300ms]" />
                </div>
                <UBadge variant="subtle" color="neutral" size="sm">
                  {{ userInfo?.nickname }} 正在输入...
                </UBadge>
              </div>
            </UCard>
          </div>
        </template>
      </ChatMessages>
    </div>

    <!-- 消息输入区域 -->
    <div class="shrink-0 z-10">
      <!-- 回复消息预览 -->
      <Transition
        enter-active-class="transition-all duration-200 ease-out"
        enter-from-class="opacity-0 translate-y-2"
        enter-to-class="opacity-100 translate-y-0"
        leave-active-class="transition-all duration-150 ease-in"
        leave-from-class="opacity-100 translate-y-0"
        leave-to-class="opacity-0 translate-y-2"
      >
        <div v-if="replyingTo" class="px-4 py-2">
          <div class="flex items-center justify-between gap-3">
            <UAlert
              class="flex-1 min-w-0"
              color="neutral"
              variant="soft"
              icon="i-lucide-corner-up-left"
              :title="`回复 ${replyingTo.sender}`"
              :description="replyingTo.content"
            />
            <UButton
              variant="ghost"
              size="xs"
              icon="i-lucide-x"
              @click="cancelReply"
            />
          </div>
        </div>
      </Transition>

      <ChatInput
        :loading="chatStatus === 'submitted'"
        @send-message="handleSendMessage"
        @send-file="uploadFile"
        @send-image="uploadImage"
        @send-location="shareLocation"
        @typing="handleTyping"
      />
    </div>
  </div>
</template>

<script setup lang="ts">
import type { PrivateMessageItem } from '~/types/api'
import type { UserDetailData, ClientMessageType } from '~/types/client'
import type { ChatMessage as BaseChatMessage } from '~/components/chat/ChatMessages.vue'
import type { DropdownMenuItem } from '@nuxt/ui'
import { useTimeoutFn } from '@vueuse/core'
import { useHydratedStore } from '~/composables/useHydratedStore'

// 扩展 ChatMessage 类型，添加私聊特有字段
interface ChatMessage extends BaseChatMessage {
  fileName?: string
  fileSize?: number
  address?: string
  latitude?: number
  longitude?: number
  alt?: string
  sender?: string  // 发送者昵称
  parts?: { type: string; text?: string; content?: string }[]
}

// 需要登录 - 私聊功能需要认证
definePageMeta({
  title: "私聊",
  requiresAuth: true,
  layout: "dashboard",
  showNavbarBack: false,
  fullHeight: true // 全高度布局，用于聊天页面
})

// ============================================================================
// 基础依赖：route / store / services
// ============================================================================

const route = useRoute()
const router = useRouter()
const targetUserId = Number(route.params.id)  // 目标用户 ID（数字）
const privateChatId = ref<string | null>(null)  // 私聊会话 ID（UUID）

const authStore = useAuthStore()
const messagesStore = useMessagesStore()
const feedback = useFeedbackStore()
const { withErrorHandling } = useErrorHandler()
const modal = useModal()

// 水合安全：持久化 store 必须通过 useHydratedStore 访问（避免 SSR hydration mismatch）
const currentUser = useHydratedStore(authStore, 'user', null as UserDetailData | null)

// 导入服务和类型
import { type Message } from '~/stores/messages.store'
import { useUsersService } from '~/services/users.service'
import { usePrivateChatsService } from '~/services/private_chats.service'
import { usePrivateMessagesService } from '~/services/private_messages.service'
import { useUploadService } from '~/services/upload.service'
import { formatFileSize } from '~/utils/file'

const userService = useUsersService()
const privateChatsService = usePrivateChatsService()
const privateMessagesService = usePrivateMessagesService()
const uploadService = useUploadService()
const geolocation = useGeolocation()
const { send: wsSend, on: wsOn, isConnected: wsConnected } = useWebSocket()

// ============================================================================
// 页面状态（UI-only）
// ============================================================================

const showSearch = ref(false)
const searchQuery = ref('')

const isTyping = ref(false) // 我正在输入（用于发送 Typing 事件）
const replyingTo = ref<ChatMessage | null>(null)
const chatStatus = ref<'ready' | 'submitted' | 'streaming' | 'error'>('ready')

const userProfile = ref<UserDetailData | null>(null)
const chatSettings = reactive({
  is_muted: false,
  is_pinned: false,
  is_archived: false
})

const chatMessagesRef = ref<{ scrollToMessage: (id: string, behavior?: ScrollBehavior) => void } | null>(null)
const highlightMessageId = ref<string | null>(null)
const { start: scheduleClearHighlight } = useTimeoutFn(() => {
  highlightMessageId.value = null
}, 1500, { immediate: false })

// ============================================================================
// 计算属性：会话 / 用户 / 消息列表（页面只读 store）
// ============================================================================

const conversation = computed(() => {
  const id = privateChatId.value
  return id ? messagesStore.getConversation(id) : null
})

const userInfo = computed((): UserDetailData => {
  // 使用缓存的用户信息
  return userProfile.value || { 
    user_id: targetUserId,
    nickname: '未知用户', 
    is_online: false,
    geo: { latitude: 0, longitude: 0, location_name: '', distance: 0, degrees: 0 }
  };
});

const isUserOnline = computed(() => {
  return conversation.value?.is_online ?? false
})

// 对方正在输入：以 store.typing 为准，但用本地超时防止断线卡住（不写回 store）
const otherUserTypingVisible = ref(false)
const { start: startClearOtherTyping, stop: stopClearOtherTyping } = useTimeoutFn(() => {
  otherUserTypingVisible.value = false
}, 5000, { immediate: false })

watch(
  () => conversation.value?.typing ?? false,
  (typing) => {
    if (typing) {
      otherUserTypingVisible.value = true
      startClearOtherTyping()
    } else {
      otherUserTypingVisible.value = false
      stopClearOtherTyping()
    }
  },
  { immediate: true }
)

const otherUserTyping = computed(() => otherUserTypingVisible.value)

const messages = computed<ChatMessage[]>(() => {
  // 转换消息格式以适配 ChatMessages 组件
  const chatId = privateChatId.value
  const conversationMessages = chatId ? messagesStore.getMessages(chatId) : []
  
  // 按时间正序排列（最新的在底部）
  const sortedMessages = [...conversationMessages].sort((a, b) => 
    new Date(a.sent_at).getTime() - new Date(b.sent_at).getTime()
  )
  
  const list = sortedMessages.map((msg: Message): ChatMessage => {
    // 映射消息类型（过滤不支持的类型）
    const supportedTypes = ['text', 'image', 'file', 'location', 'system'] as const
    const msgType = supportedTypes.includes(msg.type as typeof supportedTypes[number]) 
      ? msg.type as ClientMessageType
      : 'text'
    return {
      id: msg.id,
      role: msg.sender.user_id === currentUser.value?.user_id ? 'user' : 'assistant',
      parts: [{ type: 'text', text: msg.content }],
      content: msg.media_url || msg.content, // 优先使用 media_url（后端转换后的URL）
      type: msgType,
      timestamp: msg.sent_at,
      sender: msg.sender.nickname,
      // 传递媒体消息字段（后端已转换）
      fileName: msg.file_name,
      fileSize: msg.file_size,
      thumbnail: msg.thumbnail_url,
      duration: msg.duration,
      width: msg.width,
      height: msg.height
    }
  })

  return list
})

// 搜索命中：返回消息 id 列表（用于上/下一条跳转定位）
const searchHitMessageIds = computed<string[]>(() => {
  const q = searchQuery.value.trim().toLowerCase()
  if (!q) return []
  return messages.value
    .filter((m) => {
      const haystack = [
        m.content,
        (m as unknown as { fileName?: string }).fileName,
        (m as unknown as { address?: string }).address
      ]
        .filter(Boolean)
        .join(' ')
        .toLowerCase()
      return haystack.includes(q)
    })
    .map((m) => m.id)
})

const searchHitsCount = computed(() => searchHitMessageIds.value.length)
const hasSearchHits = computed(() => searchHitsCount.value > 0)
const activeSearchHitIndex = ref(0)
const activeSearchHitIndexDisplay = computed(() => (hasSearchHits.value ? activeSearchHitIndex.value + 1 : 0))

const jumpToHit = (idx: number) => {
  if (!hasSearchHits.value) return
  const next = ((idx % searchHitsCount.value) + searchHitsCount.value) % searchHitsCount.value
  activeSearchHitIndex.value = next
  const id = searchHitMessageIds.value[next]
  if (!id) return
  highlightMessageId.value = id
  scheduleClearHighlight()
  nextTick(() => chatMessagesRef.value?.scrollToMessage(id, 'smooth'))
}

const jumpToPrevHit = () => jumpToHit(activeSearchHitIndex.value - 1)
const jumpToNextHit = () => jumpToHit(activeSearchHitIndex.value + 1)

watch([searchQuery, searchHitMessageIds], () => {
  activeSearchHitIndex.value = 0
  if (searchQuery.value.trim() && hasSearchHits.value) {
    jumpToHit(0)
  } else {
    highlightMessageId.value = null
  }
})

// 格式化消息以适配 ChatMessages
const formattedMessages = computed(() => {
  return messages.value.map((msg) => ({
    id: msg.id,
    role: msg.role,
    type: msg.type || 'text', // 消息类型（用于自定义渲染）
    content: msg.content, // 消息内容（对于媒体消息，这里已经是 media_url）
    parts: msg.parts || [{ type: 'text', text: msg.content }],
    timestamp: msg.timestamp, // 时间戳
    // 传递媒体消息字段
    fileName: msg.fileName,
    fileSize: msg.fileSize,
    alt: msg.alt,
    metadata: {
      timestamp: msg.timestamp,
      type: msg.type
    }
  }))
})

type HighlightSegment = { text: string; isMatch: boolean }
const escapeRegExp = (s: string) => s.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')
const getHighlightedSegments = (text: string, query: string): HighlightSegment[] => {
  const q = query.trim()
  if (!q) return [{ text, isMatch: false }]
  if (!text) return [{ text: '', isMatch: false }]

  const re = new RegExp(escapeRegExp(q), 'ig')
  const segments: HighlightSegment[] = []
  let lastIndex = 0
  let match: RegExpExecArray | null

  while ((match = re.exec(text)) !== null) {
    const start = match.index
    const end = start + match[0].length
    if (start > lastIndex) {
      segments.push({ text: text.slice(lastIndex, start), isMatch: false })
    }
    segments.push({ text: text.slice(start, end), isMatch: true })
    lastIndex = end
  }

  if (lastIndex < text.length) {
    segments.push({ text: text.slice(lastIndex), isMatch: false })
  }

  return segments.length ? segments : [{ text, isMatch: false }]
}

// ============================================================================
// 动作：菜单 / 搜索 / 发送 / 上传 / 会话管理
// ============================================================================

const updateChatSettings = async (patch: Partial<typeof chatSettings>) => {
  const chatId = privateChatId.value
  if (!chatId) return
  await withErrorHandling(async () => {
    await privateChatsService.updateSettings({ id: chatId, ...patch })
    Object.assign(chatSettings, patch)

    if (typeof patch.is_muted === 'boolean') {
      feedback.success(patch.is_muted ? '已开启免打扰' : '已关闭免打扰')
    } else if (typeof patch.is_pinned === 'boolean') {
      feedback.success(patch.is_pinned ? '已置顶聊天' : '已取消置顶')
    } else if (typeof patch.is_archived === 'boolean') {
      feedback.success(patch.is_archived ? '已归档聊天' : '已取消归档')
    } else {
      feedback.success('设置已更新')
    }
  }, { title: '操作失败' })
}

const deleteChat = async () => {
  if (!privateChatId.value) return
  const confirmed = await modal.showConfirm({
    title: '删除聊天',
    message: '确定要删除此聊天吗？删除后将从列表移除（再次发起会重新创建会话）。',
    confirmText: '删除',
    confirmColor: 'error'
  })
  if (!confirmed) return

  await withErrorHandling(async () => {
    await privateChatsService.deletePrivateChat({ id: privateChatId.value! })
    messagesStore.removeConversation(privateChatId.value!)
    feedback.success('聊天已删除')
    router.back()
  }, { title: '删除失败' })
}

// 聊天菜单项（Nuxt UI v4：使用 onSelect，而不是 click）
const chatMenuItems = computed((): DropdownMenuItem[][] => [
  [
    { label: '搜索消息', icon: 'i-lucide-search', onSelect: toggleSearch },
    { label: '查看资料', icon: 'i-lucide-user', onSelect: viewUserProfile }
  ],
  [
    {
      label: chatSettings.is_muted ? '取消静音' : '静音',
      icon: chatSettings.is_muted ? 'i-lucide-volume-2' : 'i-lucide-volume-x',
      onSelect: () => updateChatSettings({ is_muted: !chatSettings.is_muted })
    },
    {
      label: chatSettings.is_pinned ? '取消置顶' : '置顶',
      icon: 'i-lucide-bookmark',
      onSelect: () => updateChatSettings({ is_pinned: !chatSettings.is_pinned })
    },
    {
      label: chatSettings.is_archived ? '取消归档' : '归档',
      icon: 'i-lucide-archive',
      onSelect: () => updateChatSettings({ is_archived: !chatSettings.is_archived })
    }
  ],
  [
    { label: '举报用户', icon: 'i-lucide-flag', onSelect: reportUser },
    { label: '屏蔽用户', icon: 'i-lucide-ban', color: 'error', onSelect: blockUser }
  ],
  [
    { label: '删除聊天', icon: 'i-lucide-trash-2', color: 'error', onSelect: deleteChat }
  ]
])

const toggleSearch = () => {
  showSearch.value = !showSearch.value;
  if (!showSearch.value) {
    searchQuery.value = '';
  }
};

const clearSearch = () => {
  searchQuery.value = '';
};

const handleSendMessage = async (content: string) => {
  if (!content || !privateChatId.value) return;
  
  try {
    chatStatus.value = 'submitted';
    
    // 发送消息到后端
    // 遵循 @frontend_rule.mdc 消息发送规范：
    // 1. API 仅负责发送消息，不处理本地数据更新
    // 2. 消息上屏完全依赖 WebSocket 推送的 ChatMessageNew 事件
    // 3. 确保显示的数据（如媒体URL、时间戳）与服务端完全一致
    await privateMessagesService.send({
      id: privateChatId.value,
      type: 'text',
      content,
      reply_to: replyingTo.value?.id
    });
    
    replyingTo.value = null;
    chatStatus.value = 'ready';
    
  } catch (error: unknown) {
    console.error('发送消息失败:', error);
    chatStatus.value = 'error';
  }
};

const uploadFile = async (file: File) => {
  const chatId = privateChatId.value
  if (!file || !chatId) return

  await withErrorHandling(async () => {
    chatStatus.value = 'submitted'

    const uploaded = await uploadService.uploadFile(file, 'chat_file')
    await privateMessagesService.send({
      id: chatId,
      type: 'file',
      file_id: uploaded.file_id
    })

    feedback.success('文件已发送', file.name)
    chatStatus.value = 'ready'
  }, {
    title: '发送失败'
  })
};

const uploadImage = async (file: File) => {
  const chatId = privateChatId.value
  if (!file || !chatId) return

  await withErrorHandling(async () => {
    chatStatus.value = 'submitted'

    const uploaded = await uploadService.uploadFile(file, 'chat_image')
    await privateMessagesService.send({
      id: chatId,
      type: 'image',
      file_id: uploaded.file_id
    })

    feedback.success('图片已发送', file.name)
    chatStatus.value = 'ready'
  }, {
    title: '发送失败'
  })
};

const shareLocation = async () => {
  try {
    const position = await geolocation.getPosition({ silent: false });
    if (!position) return;
    
    const { latitude, longitude } = position.coords;
    
    // 获取位置名称
    let locationName = geolocation.locationName.value;
    if (!locationName) {
      locationName = await geolocation.fetchLocationName(latitude, longitude);
    }
    
    const content = `📍 ${locationName}`;
    
    if (!privateChatId.value) return;
    
    // 发送位置消息到后端
    // 遵循 @frontend_rule.mdc 消息发送规范：API发送 -> WebSocket接收 -> 更新UI
    await privateMessagesService.send({
      id: privateChatId.value,
      type: 'text',
      content
    });
    
  } catch (error) {
    console.error('获取位置失败:', error);
    feedback.error('位置分享失败', '无法获取当前位置')
  }
};

const replyToMessage = (message: ChatMessage) => {
  replyingTo.value = message;
};

const cancelReply = () => {
  replyingTo.value = null;
};

const editMessage = (message: ChatMessage) => {
  // 实现消息编辑功能 - 需要 ChatInput 支持填充内容，目前 ChatInput 组件未暴露设置内容的方法
  // 如果需要支持编辑，需要修改 ChatInput 支持 v-model 或者 expose setContent 方法
  // 暂时仅支持引用回复
  replyingTo.value = message;
  feedback.info('编辑功能', '暂时仅支持回复消息')
};

const deleteMessage = async (message: ChatMessage) => {
  if (!privateChatId.value) return;
  
  const confirmed = await modal.showConfirm({
    title: '删除消息',
    message: '确定要删除这条消息吗？',
    confirmText: '删除',
    confirmColor: 'error'
  })
  if (confirmed) {
    try {
      await privateMessagesService.deletePrivateMessage({ id: message.id });
      messagesStore.removeMessage(privateChatId.value, message.id);
    } catch (error) {
      console.error('删除消息失败:', error);
    }
  }
};

const previewImage = (imageUrl: string) => {
  // 当前页面没有图片预览 modal：直接新标签打开（避免引入新组件/抽象层）
  if (!imageUrl) return
  navigateTo(imageUrl, { external: true, open: { target: '_blank' } })
};


const downloadFile = (fileUrl: string, fileName: string) => {
  if (!import.meta.client) return;
  const link = document.createElement('a');
  link.href = fileUrl;
  link.download = fileName;
  document.body.appendChild(link);
  link.click();
  document.body.removeChild(link);
};

const openLocation = (latitude: number, longitude: number) => {
  const url = `https://maps.google.com/?q=${latitude},${longitude}`;
  navigateTo(url, { external: true, open: { target: '_blank' } });
};

const clearChatHistory = async () => {
  if (!privateChatId.value) return;
  
  const confirmed = await modal.showConfirm({
    title: '清空聊天记录',
    message: '确定要清空与此用户的所有聊天记录吗？此操作无法撤销。',
    confirmText: '清空',
    confirmColor: 'error'
  })
  if (confirmed) {
    try {
      await privateChatsService.deletePrivateChat({ id: privateChatId.value });
      messagesStore.removeConversation(privateChatId.value);
      feedback.success('聊天记录已清空')
    } catch (error) {
      console.error('清空聊天记录失败:', error);
    }
  }
};

const reportUser = () => {
  void modal.showReport('user', String(targetUserId)).then(() => {
    feedback.success('举报已提交', '我们会尽快处理您的举报')
  }).catch(() => {})
};

const blockUser = async () => {
  const confirmed = await modal.showConfirm({
    title: '屏蔽用户',
    message: `确定要屏蔽用户 ${userInfo.value?.nickname} 吗？屏蔽后将无法收到对方的消息。`,
    confirmText: '屏蔽',
    confirmColor: 'error'
  })
  if (confirmed) {
    try {
      await userService.block({ id: targetUserId });
      feedback.success('用户已屏蔽')
      router.back();
    } catch (error) {
      console.error('屏蔽用户失败:', error);
    }
  }
};

// 错误状态
const chatError = ref<string | null>(null)

const loadMessages = async () => {
  try {
    // 检查是否尝试与自己聊天
    if (currentUser.value?.user_id === targetUserId) {
      chatError.value = 'self'
      feedback.warning('无法创建私聊', '不能与自己创建私聊')
      return
    }
    
    // 1. 创建或获取私聊会话
    const chatData = await privateChatsService.create({ target_user_id: targetUserId });
    privateChatId.value = chatData.id;
    
    // 2. 获取私聊详情（包含对方用户信息）
    const chatDetail = await privateChatsService.openTheDoor({ id: chatData.id });
    chatSettings.is_muted = chatDetail.is_muted
    chatSettings.is_pinned = chatDetail.is_pinned
    chatSettings.is_archived = chatDetail.is_archived
    userProfile.value = {
      user_id: chatDetail.other_user.user_id,
      nickname: chatDetail.other_user.nickname,
      avatar_url: chatDetail.other_user.avatar_url,
      user_type: chatDetail.other_user.user_type,
      is_online: false,
      last_online: undefined,
      geo: {
        latitude: 0,
        longitude: 0,
        location_name: '未知位置',
        distance: 0,
        degrees: 0
      }
    };
    
    // 3. 获取消息历史
    const response = await privateMessagesService.getPrivateMessages({ id: chatData.id });
    const messages: Message[] = response.items.map((item: PrivateMessageItem) => ({
      ...item,
      is_recalled: item.is_recalled || false
    }));
    messagesStore.setMessages(chatData.id, messages);

    // 3.1 upsert 会话基础信息（保证 store 有会话，且后续 WS 增量能正常工作）
    messagesStore.upsertConversation({
      chat_id: chatDetail.id,
      chat_type: 'private',
      target_user_id: chatDetail.other_user.user_id,
      name: chatDetail.other_user.nickname,
      avatar_url: chatDetail.other_user.avatar_url,
      unread_count: chatDetail.unread_count,
      last_message: chatDetail.last_message,
      last_activity_at: chatDetail.last_activity_at,
      is_online: false
    })

    // 3.2 当前会话标记为“正在查看”（用于本地 unread 计算）
    messagesStore.setCurrentConversation(chatDetail.id)

    // 3.3 进入会话即标记已读（修复：打开聊天看过消息仍显示未读）
    if (chatDetail.unread_count > 0) {
      const upTo = messages[messages.length - 1]?.id
      try {
        await privateMessagesService.markPrivateMessagesAsRead({
          id: chatDetail.id,
          up_to: upTo
        })
      } catch {
        // 标记已读失败不阻塞 UI；下次进入/刷新再尝试
      }
      messagesStore.updateUnreadCount(chatDetail.id, 0)
    }
    
  } catch (error: unknown) {
    console.error('加载聊天数据失败:', error);
    chatError.value = 'load_failed'
    
    // 根据错误类型显示不同提示
    const errorMessage = error instanceof Error ? error.message : '未知错误'
    if (errorMessage.includes('不能与自己')) {
      chatError.value = 'self'
      feedback.warning('无法创建私聊', '不能与自己创建私聊')
    } else {
      feedback.error('加载失败', '无法加载聊天数据，请稍后重试')
    }
  }
};

// ============================================================================
// 生命周期：加载会话 / 当前会话标记 / WS 辅助事件（仅补 last_online 细节）
// ============================================================================

onMounted(() => {
  loadMessages();
  
  // 注册快捷键
  defineShortcuts({
    'escape': {
      usingInput: false,
      handler: () => {
        if (showSearch.value) toggleSearch();
        // Emoji picker 和 Reply status 现在由 ChatInput 或内部逻辑管理，这里可能需要调整
        if (replyingTo.value) cancelReply();
      },
    },
    'f': {
      usingInput: false,
      handler: toggleSearch,
    },
  });

  // 补 last_online 的细节：store 只维护 is_online（会话亮灯），last_online 仅在本页展示
  wsOn('UserOnline', (data) => {
    if (data.user_id !== targetUserId) return
    if (userProfile.value) {
      userProfile.value.is_online = true
      userProfile.value.last_online = undefined
    }
  })
  wsOn('UserOffline', (data) => {
    if (data.user_id !== targetUserId) return
    if (userProfile.value) {
      userProfile.value.is_online = false
      userProfile.value.last_online = data.offline_at
    }
  })
});

onUnmounted(() => {
  stopStopTyping() // 取消可能存在的停止输入防抖
  stopClearOtherTyping() // 取消对方输入清除防抖

  // 离开会话：让后续新消息能正确计入未读
  messagesStore.setCurrentConversation(null)
  // 发送停止打字状态
  if (isTyping.value) {
    sendTypingStatus(false)
  }
})

// ============================================================================
// 输入状态：发送 Typing（消息显示由 store.typing 驱动）
// ============================================================================

const sendTypingStatus = (isTypingStatus: boolean) => {
  const chatId = privateChatId.value
  if (!chatId || !wsConnected.value) return
  wsSend('Typing', {
    chat_type: 'private_chat' as const,
    chat_id: chatId,
    is_typing: isTypingStatus
  })
}

const { start: startStopTyping, stop: stopStopTyping } = useTimeoutFn(() => {
  isTyping.value = false;
  sendTypingStatus(false);
}, 3000);

const handleTyping = (isTypingNow: boolean) => {
  if (isTypingNow && privateChatId.value) {
    // 发送正在输入状态
    if (!isTyping.value) {
      isTyping.value = true;
      sendTypingStatus(true);
    }
    // 重置停止输入的计时器
    startStopTyping();
  } else if (!isTypingNow && isTyping.value) {
    // 输入框清空时立即停止打字状态
    stopStopTyping();
    isTyping.value = false;
    sendTypingStatus(false);
  }
}

const viewUserProfile = () => {
  const userId = Number(route.params.id);
  if (userId === currentUser.value?.user_id) {
    navigateTo('/profile');
  } else {
    navigateTo(`/users/${userId}`);
  }
};

</script>