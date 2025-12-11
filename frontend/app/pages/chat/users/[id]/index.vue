<template>
  <div class="flex flex-col h-full">
    <!-- 聊天头部 -->
    <ChatHeader
      :user-info="userInfo"
      :is-online="isUserOnline"
      :last-online="userProfile?.last_online"
      @go-back="router.back()"
      @toggle-search="toggleSearch"
      @voice-call="makeCall"
      @video-call="makeVideoCall"
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
      <div v-if="showSearch" class="px-4 py-3 border-b border-strong bg-glass-white-deep backdrop-blur-sm">
        <UInput
          v-model="searchQuery"
          placeholder="搜索聊天记录..."
          icon="i-heroicons-magnifying-glass"
          size="md"
          class="w-full"
          autofocus
          @keyup.escape="toggleSearch"
        >
          <template #trailing>
            <UButton
              variant="ghost"
              size="xs"
              icon="i-heroicons-x-mark"
              @click="clearSearch"
            />
          </template>
        </UInput>
        <p v-if="searchQuery && filteredMessages.length === 0" class="text-sm text-secondary-content mt-2">
          没有找到相关消息
        </p>
        <p v-else-if="searchQuery" class="text-sm text-secondary-content mt-2">
          找到 {{ filteredMessages.length }} 条相关消息
        </p>
      </div>
    </Transition>

    <!-- 聊天消息区域 -->
    <div class="flex-1 overflow-hidden">
      <!-- 错误状态：与自己聊天 -->
      <div v-if="chatError === 'self'" class="center-container">
        <div class="text-center px-4">
          <div class="icon-avatar-lg icon-container-rounded bg-warning-subtle mx-auto mb-4">
            <UIcon name="i-heroicons-exclamation-triangle" class="icon-xl text-warning" />
          </div>
          <h3 class="text-lg font-semibold text-primary-content mb-2">
            无法创建私聊
          </h3>
          <p class="text-secondary-content mb-4">
            不能与自己创建私聊
          </p>
          <UButton color="primary" @click="router.back()">
            返回
          </UButton>
        </div>
      </div>
      
      <!-- 错误状态：加载失败 -->
      <div v-else-if="chatError === 'load_failed'" class="center-container">
        <div class="text-center px-4">
          <div class="icon-avatar-lg icon-container-rounded bg-danger-subtle mx-auto mb-4">
            <UIcon name="i-heroicons-x-circle" class="icon-xl text-error" />
          </div>
          <h3 class="text-lg font-semibold text-primary-content mb-2">
            加载失败
          </h3>
          <p class="text-secondary-content mb-4">
            无法加载聊天数据
          </p>
          <div class="flex gap-2 justify-center">
            <UButton variant="outline" @click="router.back()">
              返回
            </UButton>
            <UButton color="primary" @click="loadMessages">
              重试
            </UButton>
          </div>
        </div>
      </div>
      
      <!-- 加载中状态 -->
      <div v-else-if="!privateChatId" class="center-container">
        <div class="text-center">
          <UIcon name="i-heroicons-chat-bubble-left-right" class="icon-2xl text-subtle mx-auto mb-3 animate-pulse" />
          <p class="text-secondary-content">正在加载聊天...</p>
        </div>
      </div>
      
      <!-- 消息列表 -->
      <ChatMessages
        v-else
        :messages="formattedMessages"
        :status="chatStatus"
        :should-auto-scroll="true"
        :should-scroll-to-bottom="true"
        :user="{
          avatar: { 
            alt: authStore.user?.nickname,
            src: authStore.user?.avatar_url,
            fallback: authStore.user?.nickname?.charAt(0) || 'U'
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
          <div class="h-full flex flex-col items-center justify-center text-center px-4">
            <div class="icon-avatar-lg icon-container-rounded bg-gradient-primary-avatar mb-4">
              <UIcon name="i-heroicons-chat-bubble-left-ellipsis" class="icon-xl text-accent" />
            </div>
            <h3 class="text-lg font-semibold text-primary-content mb-2">
              开始聊天吧
            </h3>
            <p class="text-secondary-content max-w-xs">
              向 {{ userInfo?.nickname || '对方' }} 发送第一条消息，开启你们的对话
            </p>
          </div>
        </template>
        
        <!-- 自定义消息内容 -->
        <template #content="{ message }">
          <div class="space-y-2">
            <!-- 文本消息 -->
            <div v-if="(message as ChatMessage).type === 'text'" class="text-sm leading-relaxed">
              {{ (message as ChatMessage).content }}
            </div>
            
            <!-- 图片消息 -->
            <div v-else-if="(message as ChatMessage).type === 'image'" class="max-w-xs">
              <NuxtImg
                :src="String((message as ChatMessage).content || '')"
                :alt="(message as ChatMessage).alt || '图片'"
                class="rounded-xl cursor-pointer hover:opacity-90 transition-all hover:scale-[1.02] shadow-sm"
                @click="previewImage(String((message as ChatMessage).content || ''))"
              />
            </div>
            
            <!-- 文件消息 -->
            <div v-else-if="(message as ChatMessage).type === 'file'" class="flex items-center gap-3 p-3 bg-overlay rounded-xl backdrop-blur-sm">
              <div class="icon-avatar-sm icon-container-rounded-lg bg-primary-subtle">
                <UIcon name="i-heroicons-document" class="icon-md text-accent" />
              </div>
              <div class="flex-1 min-w-0">
                <p class="font-medium truncate text-sm">{{ (message as ChatMessage).fileName }}</p>
                <p class="text-xs text-secondary-content">{{ formatFileSize((message as ChatMessage).fileSize || 0) }}</p>
              </div>
              <UButton
                variant="ghost"
                size="xs"
                icon="i-heroicons-arrow-down-tray"
                @click="downloadFile(String((message as ChatMessage).content || ''), (message as ChatMessage).fileName || '')"
              />
            </div>
            
            <!-- 位置消息 -->
            <div v-else-if="(message as ChatMessage).type === 'location'" class="p-3 bg-overlay rounded-xl backdrop-blur-sm">
              <div class="flex items-center gap-2 mb-2">
                <div class="icon-lg icon-container-rounded-lg bg-primary-subtle">
                  <UIcon name="i-heroicons-map-pin" class="icon-sm text-accent" />
                </div>
                <span class="font-medium text-sm">位置信息</span>
              </div>
              <p class="text-sm text-secondary-strong mb-2">{{ (message as ChatMessage).address }}</p>
              <UButton
                variant="soft"
                size="xs"
                color="primary"
                @click="openLocation((message as ChatMessage).latitude || 0, (message as ChatMessage).longitude || 0)"
              >
                <UIcon name="i-heroicons-map" class="icon-sm mr-1" />
                查看位置
              </UButton>
            </div>
            
            <!-- 系统消息 -->
            <div v-else-if="(message as ChatMessage).type === 'system'" class="text-center py-2">
              <span class="inline-flex items-center gap-1.5 text-xs text-secondary-content bg-overlay px-3 py-1.5 rounded-full backdrop-blur-sm">
                <UIcon name="i-heroicons-information-circle" class="icon-xs" />
                {{ (message as ChatMessage).content }}
              </span>
            </div>
          </div>
        </template>

        <!-- 消息操作 -->
        <template #actions="{ message }">
          <div v-if="message.role === 'user'" class="flex items-center gap-0.5 opacity-0 group-hover:opacity-100 transition-opacity">
            <UTooltip text="回复">
              <UButton
                variant="ghost"
                size="xs"
                icon="i-heroicons-arrow-uturn-left"
                @click="replyToMessage(message)"
              />
            </UTooltip>
            <UTooltip text="编辑">
              <UButton
                variant="ghost"
                size="xs"
                icon="i-heroicons-pencil"
                @click="editMessage(message)"
              />
            </UTooltip>
            <UTooltip text="删除">
              <UButton
                variant="ghost"
                size="xs"
                icon="i-heroicons-trash"
                color="error"
                @click="deleteMessage(message)"
              />
            </UTooltip>
          </div>
        </template>

        <!-- 对方正在输入指示器 -->
        <template #indicator v-if="otherUserTyping">
          <div class="flex items-center gap-2 text-sm text-secondary-content px-4 py-2">
            <div class="typing-indicator">
              <div class="typing-dot"></div>
              <div class="typing-dot"></div>
              <div class="typing-dot"></div>
            </div>
            <span>{{ userInfo?.nickname }} 正在输入...</span>
          </div>
        </template>
      </ChatMessages>
    </div>

    <!-- 消息输入区域 -->
    <div class="shrink-0 border-t border-glass bg-modal backdrop-blur-xl">
      <!-- 回复消息预览 -->
      <Transition
        enter-active-class="transition-all duration-200 ease-out"
        enter-from-class="opacity-0 translate-y-2"
        enter-to-class="opacity-100 translate-y-0"
        leave-active-class="transition-all duration-150 ease-in"
        leave-from-class="opacity-100 translate-y-0"
        leave-to-class="opacity-0 translate-y-2"
      >
        <div v-if="replyingTo" class="mx-3 mt-2 p-2 bg-primary-light-50 border-l-3 border-accent-medium rounded-r-lg flex items-center justify-between">
          <div class="flex-1 min-w-0">
            <p class="text-xs font-medium text-accent mb-0.5">回复 {{ replyingTo.sender }}</p>
            <p class="text-sm text-secondary-strong truncate">{{ replyingTo.content }}</p>
          </div>
          <UButton
            variant="ghost"
            size="xs"
            icon="i-heroicons-x-mark"
            @click="cancelReply"
          />
        </div>
      </Transition>

      <div class="px-3 py-2">
        <!-- 表情选择器（移到输入框上方） -->
        <Transition
          enter-active-class="transition-all duration-200 ease-out"
          enter-from-class="opacity-0 scale-95"
          enter-to-class="opacity-100 scale-100"
          leave-active-class="transition-all duration-150 ease-in"
          leave-from-class="opacity-100 scale-100"
          leave-to-class="opacity-0 scale-95"
        >
          <div v-if="showEmojiPicker" class="mb-2 p-2 bg-overlay rounded-xl backdrop-blur-sm">
            <div class="grid grid-cols-8 sm:grid-cols-12 gap-0.5">
              <UButton
                v-for="emoji in commonEmojis"
                :key="emoji"
                variant="ghost"
                size="xs"
                :label="emoji"
                class="w-8 h-8 text-lg"
                @click="insertEmoji(emoji)"
              />
            </div>
          </div>
        </Transition>
        
        <div class="flex items-center gap-1">
          <!-- 附加按钮 -->
          <UDropdownMenu :items="attachmentMenuItems">
            <UButton
              variant="ghost"
              icon="i-heroicons-plus-circle"
              square
              class="shrink-0 chat-icon-btn text-secondary-content hover:text-accent"
            />
          </UDropdownMenu>

          <!-- 表情按钮 -->
          <UButton
            variant="ghost"
            icon="i-heroicons-face-smile"
            square
            :class="['shrink-0 chat-icon-btn', showEmojiPicker ? 'text-accent' : 'text-secondary-content hover:text-accent']"
            @click="toggleEmojiPicker"
          />

          <!-- 输入框容器 -->
          <div class="chat-input-wrapper">
            <input
              v-model="messageInput"
              placeholder="输入消息..."
              autofocus
              @keydown.enter.exact.prevent="handleSendMessage"
            />
          </div>

          <!-- 发送按钮 -->
          <UButton
            color="primary"
            :icon="chatStatus === 'submitted' ? 'i-heroicons-stop' : 'i-heroicons-paper-airplane'"
            :loading="chatStatus === 'submitted'"
            :disabled="!messageInput.trim() || !privateChatId"
            class="shrink-0 chat-send-btn"
            @click="chatStatus === 'submitted' ? stopSending() : handleSendMessage()"
          />
        </div>

        <!-- 发送错误提示 -->
        <Transition
          enter-active-class="transition-all duration-200 ease-out"
          enter-from-class="opacity-0 translate-y-2"
          enter-to-class="opacity-100 translate-y-0"
          leave-active-class="transition-all duration-150 ease-in"
          leave-from-class="opacity-100 translate-y-0"
          leave-to-class="opacity-0 translate-y-2"
        >
          <div v-if="sendError" class="mt-2 flex items-center justify-between p-2 bg-error-light-50 border border-error rounded-lg">
            <div class="flex items-center gap-2 text-error">
              <UIcon name="i-heroicons-exclamation-circle" class="icon-sm" />
              <span class="text-sm">{{ sendError }}</span>
            </div>
            <UButton
              variant="ghost"
              size="xs"
              color="error"
              @click="reloadChat"
            >
              重试
            </UButton>
          </div>
        </Transition>
      </div>
    </div>
  </div>

  <!-- 文件上传 (使用 UFileUpload) -->
  <UFileUpload
    v-slot="{ open: openFile }"
    v-model="selectedFiles"
    multiple
    class="hidden"
    @update:model-value="handleFilesSelected"
  >
    <button ref="fileUploadBtn" class="hidden" @click="openFile()" />
  </UFileUpload>

  <!-- 图片上传 (使用 UFileUpload) -->
  <UFileUpload
    v-slot="{ open: openImage }"
    v-model="selectedImages"
    accept="image/*"
    multiple
    class="hidden"
    @update:model-value="handleImagesSelected"
  >
    <button ref="imageUploadBtn" class="hidden" @click="openImage()" />
  </UFileUpload>
</template>

<script setup lang="ts">
import type { PrivateMessageItem } from '~/types/api'
import type { UserResponse } from '~/types/client'
import type { ChatMessage as BaseChatMessage } from '~/components/chat/ChatMessages.vue'
import { getErrorMessage } from '~/utils/error'

// 扩展 ChatMessage 类型，添加私聊特有字段
interface ChatMessage extends BaseChatMessage {
  type?: 'text' | 'image' | 'file' | 'location' | 'system'
  fileName?: string
  fileSize?: number
  address?: string
  latitude?: number
  longitude?: number
  alt?: string
  sender?: string  // 发送者昵称
}

// 需要登录 - 私聊功能需要认证
definePageMeta({
  title: "私聊",
  requiresAuth: true,
  layout: "dashboard",
  fullHeight: true // 全高度布局，用于聊天页面
})

const route = useRoute()
const router = useRouter()
const targetUserId = Number(route.params.id)  // 目标用户 ID（数字）
const privateChatId = ref<string | null>(null)  // 私聊会话 ID（UUID）

// 使用水合安全的 store 访问
const authStore = useAuthStore()
const messagesStore = useMessagesStore()

// 导入服务和类型
import { type Message } from '~/stores/messages.store'
import { useUsersService } from '~/services/users.service'
import { usePrivateChatsService } from '~/services/private_chats.service'
import { usePrivateMessagesService } from '~/services/private_messages.service'
import { geocoding } from '~/utils/geocoding'
import { formatFileSize } from '~/utils/file'

const userService = useUsersService();
const privateChatsService = usePrivateChatsService();
const privateMessagesService = usePrivateMessagesService();
const geolocation = useGeolocation();
const toast = useToast();
const modal = useModal();
const { send: wsSend, on: wsOn, isConnected: wsConnected } = useGlobalWebSocket();

// 响应式状态
const messageInput = ref('');
const showSearch = ref(false);
const searchQuery = ref('');
const showEmojiPicker = ref(false);
const isTyping = ref(false);  // 我正在输入
const otherUserTyping = ref(false);  // 对方正在输入
const replyingTo = ref<ChatMessage | null>(null);
const sendError = ref<string | null>(null);
const chatStatus = ref<'ready' | 'submitted' | 'streaming' | 'error'>('ready');
const currentImagePreview = ref<string | null>(null);
const showImagePreview = ref(false);
const userProfile = ref<UserResponse | null>(null);

// 文件上传相关 (UFileUpload)
const selectedFiles = ref<File[] | null>(null);
const selectedImages = ref<File[] | null>(null);
const fileUploadBtn = ref<HTMLButtonElement | null>(null);
const imageUploadBtn = ref<HTMLButtonElement | null>(null);

// 附件菜单项
const attachmentMenuItems = computed(() => [
  [{
    label: '发送图片',
    icon: 'i-heroicons-photo',
    click: () => openImageUpload()
  }, {
    label: '发送文件',
    icon: 'i-heroicons-document',
    click: () => openFileUpload()
  }],
  [{
    label: '分享位置',
    icon: 'i-heroicons-map-pin',
    click: () => shareLocation()
  }]
]);

// 计算属性
const userInfo = computed((): UserResponse => {
  // 使用缓存的用户信息
  return userProfile.value || { 
    user_id: targetUserId,
    nickname: '未知用户', 
    is_online: false,
    geo: { latitude: 0, longitude: 0, location_name: '', distance: 0, degrees: 0 }
  };
});
const messages = computed<ChatMessage[]>(() => {
  // 转换消息格式以适配 ChatMessages 组件
  const conversationMessages = privateChatId.value ? messagesStore.messages.get(privateChatId.value) || [] : []
  
  // 按时间正序排列（最新的在底部）
  const sortedMessages = [...conversationMessages].sort((a, b) => 
    new Date(a.sent_at).getTime() - new Date(b.sent_at).getTime()
  )
  
  return sortedMessages.map((msg: Message): ChatMessage => {
    // 映射消息类型（过滤不支持的类型）
    const supportedTypes = ['text', 'image', 'file', 'location', 'system'] as const
    const msgType = supportedTypes.includes(msg.type as typeof supportedTypes[number]) 
      ? msg.type as 'text' | 'image' | 'file' | 'location' | 'system'
      : 'text'
    return {
      id: msg.id,
      role: msg.sender.user_id === authStore.user?.user_id ? 'user' : 'assistant',
      parts: [{ type: 'text', text: msg.content }],
      content: msg.content,
      type: msgType,
      timestamp: msg.sent_at,
      sender: msg.sender.nickname
    }
  })
})

// 筛选消息（搜索功能）
const filteredMessages = computed<ChatMessage[]>(() => {
  if (!searchQuery.value.trim()) {
    return messages.value
  }
  
  return messages.value.filter((message) => 
    message.content?.toLowerCase().includes(searchQuery.value.toLowerCase())
  )
})

// 格式化消息以适配 ChatMessages
const formattedMessages = computed(() => {
  return filteredMessages.value.map((msg) => ({
    id: msg.id,
    role: msg.role,
    type: msg.type || 'text', // 消息类型（用于自定义渲染）
    content: msg.content, // 消息内容
    parts: msg.parts || [{ type: 'text', text: msg.content }],
    timestamp: msg.timestamp, // 时间戳
    metadata: {
      timestamp: msg.timestamp,
      type: msg.type
    }
  }))
})

// 获取消息内容
const getMessageContent = (message: ChatMessage): string => {
  if (message.parts && message.parts.length > 0) {
    return message.parts[0].text || message.parts[0].content || ''
  }
  return message.content || ''
}

// 聊天菜单项
const chatMenuItems = computed(() => [
  [{
    label: '查看用户资料',
    icon: 'i-heroicons-user',
    click: () => targetUserId === authStore.user?.user_id ? navigateTo('/profile') : navigateTo(`/users/${targetUserId}`)
  }],
  [{
    label: '清空聊天记录',
    icon: 'i-heroicons-trash',
    click: () => clearChatHistory()
  }, {
    label: '举报用户',
    icon: 'i-heroicons-flag',
    click: () => reportUser()
  }],
  [{
    label: '屏蔽用户',
    icon: 'i-heroicons-no-symbol',
    click: () => blockUser(),
    class: 'text-error'
  }]
]);

// 常用表情
const commonEmojis = ['😀', '😂', '😍', '🥰', '😊', '😎', '🤔', '😢', '😭', '😡', '👍', '👎', '❤️', '💯', '🔥', '✨'];

// 方法

const getOnlineStatus = () => {
  if (!userInfo.value) return '离线';
  if (userInfo.value.is_online) return '在线';
  if (userInfo.value.last_active_at) {
    const lastActive = new Date(userInfo.value.last_active_at);
    const now = new Date();
    const diff = now.getTime() - lastActive.getTime();
    
    if (diff < 300000) return '5分钟前在线'; // 5分钟内
    if (diff < 3600000) return `${Math.floor(diff / 60000)}分钟前在线`;
    if (diff < 86400000) return `${Math.floor(diff / 3600000)}小时前在线`;
    return `${Math.floor(diff / 86400000)}天前在线`;
  }
  return '离线';
};

const toggleSearch = () => {
  showSearch.value = !showSearch.value;
  if (!showSearch.value) {
    searchQuery.value = '';
  }
};

const clearSearch = () => {
  searchQuery.value = '';
};

const makeCall = () => {
  toast.add({
    title: '语音通话',
    description: '功能开发中，敬请期待',
    color: 'info'
  });
};

const makeVideoCall = () => {
  toast.add({
    title: '视频通话',
    description: '功能开发中，敬请期待',
    color: 'info'
  });
};

const handleSendMessage = async () => {
  if (!messageInput.value.trim() || !privateChatId.value) return;
  
  try {
    chatStatus.value = 'submitted';
    sendError.value = null;
    
    const content = messageInput.value.trim();
    
    // 发送消息到后端
    // 注意：不立即显示消息，等待 WebSocket 推送后再显示
    // 这样可以确保显示的内容与服务端一致（如媒体 URL 转换等）
    await privateMessagesService.send({
      id: privateChatId.value,
      type: 'text',
      content,
      reply_to: replyingTo.value?.id
    });
    
    messageInput.value = '';
    replyingTo.value = null;
    chatStatus.value = 'ready';
    
  } catch (error: unknown) {
    console.error('发送消息失败:', error);
    sendError.value = getErrorMessage(error, '发送失败');
    chatStatus.value = 'error';
  }
};

const stopSending = () => {
  chatStatus.value = 'ready';
};

const reloadChat = async () => {
  await loadMessages();
};

const toggleEmojiPicker = () => {
  showEmojiPicker.value = !showEmojiPicker.value;
};

const insertEmoji = (emoji: string) => {
  messageInput.value += emoji;
  showEmojiPicker.value = false;
};

const openFileUpload = () => {
  fileUploadBtn.value?.click();
};

const openImageUpload = () => {
  imageUploadBtn.value?.click();
};

// UFileUpload 回调 - 处理选中的文件
const handleFilesSelected = async (files: File[] | null | undefined) => {
  if (!files) return;
  
  for (const file of files) {
    try {
      await uploadFile(file);
    } catch (error) {
      console.error('文件上传失败:', error);
      toast.add({
        title: '上传失败',
        description: `文件 ${file.name} 上传失败`,
        color: 'error'
      });
    }
  }
  // 重置选择
  selectedFiles.value = null;
};

// UFileUpload 回调 - 处理选中的图片
const handleImagesSelected = async (files: File[] | null | undefined) => {
  if (!files) return;
  
  for (const file of files) {
    try {
      await uploadImage(file);
    } catch (error) {
      console.error('图片上传失败:', error);
      toast.add({
        title: '上传失败',
        description: `图片 ${file.name} 上传失败`,
        color: 'error'
      });
    }
  }
  // 重置选择
  selectedImages.value = null;
};

const uploadFile = async (file: File) => {
  // 实现文件上传逻辑
  console.log('上传文件:', file.name);
};

const uploadImage = async (file: File) => {
  // 实现图片上传逻辑
  console.log('上传图片:', file.name);
};

const shareLocation = async () => {
  try {
    const position = await new Promise<GeolocationPosition>((resolve, reject) => {
      navigator.geolocation.getCurrentPosition(resolve, reject);
    });
    
    const { latitude, longitude } = position.coords;
    
    // 获取位置名称
    let locationName = '当前位置';
    try {
      locationName = await geocoding.getLocationName(latitude, longitude);
    } catch {
      // 使用默认名称
    }
    
    const content = `📍 ${locationName}`;
    
    if (!privateChatId.value) return;
    
    // 发送位置消息到后端（不立即显示，等 WebSocket 推送）
    await privateMessagesService.send({
      id: privateChatId.value,
      type: 'text',
      content
    });
    
  } catch (error) {
    console.error('获取位置失败:', error);
    toast.add({
      title: '位置分享失败',
      description: '无法获取当前位置',
      color: 'error'
    });
  }
};

const replyToMessage = (message: ChatMessage) => {
  replyingTo.value = message;
};

const cancelReply = () => {
  replyingTo.value = null;
};

const editMessage = (message: ChatMessage) => {
  // 实现消息编辑功能
  messageInput.value = message.content || '';
  replyingTo.value = message;
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
  // 打开图片预览模态框
  currentImagePreview.value = imageUrl;
  showImagePreview.value = true;
};


const downloadFile = (fileUrl: string, fileName: string) => {
  const link = document.createElement('a');
  link.href = fileUrl;
  link.download = fileName;
  link.click();
};

const openLocation = (latitude: number, longitude: number) => {
  const url = `https://maps.google.com/?q=${latitude},${longitude}`;
  window.open(url, '_blank');
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
      toast.add({
        title: '聊天记录已清空',
        color: 'success'
      });
    } catch (error) {
      console.error('清空聊天记录失败:', error);
    }
  }
};

const reportUser = () => {
  // 实现举报用户功能
  toast.add({
    title: '举报功能',
    description: '功能开发中，敬请期待',
    color: 'info'
  });
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
      await userService.blockUser({ id: targetUserId });
      toast.add({
        title: '用户已屏蔽',
        color: 'success'
      });
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
    if (authStore.user?.user_id === targetUserId) {
      chatError.value = 'self'
      toast.add({
        title: '无法创建私聊',
        description: '不能与自己创建私聊',
        color: 'warning'
      });
      return
    }
    
    // 1. 创建或获取私聊会话
    const chatData = await privateChatsService.create({ target_user_id: targetUserId });
    privateChatId.value = chatData.id;
    
    // 2. 获取私聊详情（包含对方用户信息）
    const chatDetail = await privateChatsService.openTheDoor({ id: chatData.id });
    userProfile.value = {
      user_id: chatDetail.other_user.user_id,
      nickname: chatDetail.other_user.nickname,
      avatar_url: chatDetail.other_user.avatar_url,
      is_online: false,
      last_online: undefined
    } as any;
    
    // 3. 获取消息历史
    const response = await privateMessagesService.getPrivateMessages({ id: chatData.id });
    const messages: Message[] = response.items.map((item: PrivateMessageItem) => ({
      ...item,
      is_recalled: item.is_recalled || false
    }));
    messagesStore.setMessages(chatData.id, messages);
    
    // 4. 用户在线状态：聊天场景适合通过 WebSocket 订阅实时获取
    // 在 onMounted 中已通过 WebSocket 订阅用户状态（source_type: 'user'）
  } catch (error: unknown) {
    console.error('加载聊天数据失败:', error);
    chatError.value = 'load_failed'
    
    // 根据错误类型显示不同提示
    const errorMessage = error instanceof Error ? error.message : '未知错误'
    if (errorMessage.includes('不能与自己')) {
      chatError.value = 'self'
      toast.add({
        title: '无法创建私聊',
        description: '不能与自己创建私聊',
        color: 'warning'
      });
    } else {
      toast.add({
        title: '加载失败',
        description: '无法加载聊天数据，请稍后重试',
        color: 'error'
      });
    }
  }
};

// 生命周期
onMounted(() => {
  loadMessages();
  
  // 注册快捷键
  defineShortcuts({
    'escape': {
      usingInput: false,
      handler: () => {
        if (showSearch.value) toggleSearch();
        if (showEmojiPicker.value) showEmojiPicker.value = false;
        if (replyingTo.value) cancelReply();
      },
    },
    'f': {
      usingInput: false,
      handler: toggleSearch,
    },
  });
});

// 监听输入状态（打字指示器）
let typingTimer: NodeJS.Timeout;
const sendTypingStatus = (isTyping: boolean) => {
  if (!privateChatId.value || !wsConnected.value) return;
  wsSend('Typing', {
    chat_type: 'private_chat' as const,
    chat_id: privateChatId.value,
    is_typing: isTyping
  });
};

watch(messageInput, (newValue, oldValue) => {
  if (newValue !== oldValue && newValue.length > 0 && privateChatId.value) {
    // 发送正在输入状态
    if (!isTyping.value) {
      isTyping.value = true;
      sendTypingStatus(true);
    }
    
    clearTimeout(typingTimer);
    typingTimer = setTimeout(() => {
      // 停止打字状态
      isTyping.value = false;
      sendTypingStatus(false);
    }, 3000);
  } else if (newValue.length === 0 && isTyping.value) {
    // 输入框清空时停止打字状态
    clearTimeout(typingTimer);
    isTyping.value = false;
    sendTypingStatus(false);
  }
});

// 用户在线状态（通过 WebSocket 订阅实时更新，或从 userProfile 中获取初始值）
const isUserOnline = computed(() => {
  return userProfile.value?.is_online ?? false;
});

const viewUserProfile = () => {
  const userId = Number(route.params.id);
  if (userId === authStore.user?.user_id) {
    navigateTo('/profile');
  } else {
    navigateTo(`/users/${userId}`);
  }
};

// 监听对方的打字状态和实时消息
let typingCleanup: (() => void) | null = null;
let messageCleanup: (() => void) | null = null;
let otherUserTypingTimer: NodeJS.Timeout;

// ChatMessageNew 事件的 payload 类型
interface ChatMessageNewPayload {
  id: string;
  chat_type: 'private_chat' | 'group_chat';
  chat_id: string;
  sender: {
    user_id: number;
    nickname: string;
    avatar_url?: string;
    user_type: string;
  };
  message_type: 'text' | 'image' | 'video' | 'audio' | 'file' | 'system';
  content?: string;
  reply_to?: string;
  mentions?: number[];
  media_url?: string;
  thumbnail_url?: string;
  file_name?: string;
  file_size?: number;
  duration?: number;
  width?: number;
  height?: number;
  sent_at: string;
}

onMounted(() => {
  // 订阅 UserTyping 事件
  typingCleanup = wsOn<{
    chat_type: 'private_chat' | 'group_chat';
    chat_id: string;
    user_id: number;
    nickname: string;
    is_typing: boolean;
  }>('UserTyping', (data) => {
    // 只处理当前私聊的打字事件
    if (data.chat_type === 'private_chat' && 
        data.chat_id === privateChatId.value && 
        data.user_id === targetUserId) {
      otherUserTyping.value = data.is_typing;
      
      // 如果对方正在打字，设置超时自动关闭（防止断线导致状态卡住）
      if (data.is_typing) {
        clearTimeout(otherUserTypingTimer);
        otherUserTypingTimer = setTimeout(() => {
          otherUserTyping.value = false;
        }, 5000);
      }
    }
  });
  
  // 订阅 ChatMessageNew 事件 - 实时接收新消息（包括自己发送的）
  // 所有消息都通过 WebSocket 推送后才显示，确保内容与服务端一致
  messageCleanup = wsOn<ChatMessageNewPayload>('ChatMessageNew', (data) => {
    // 只处理当前私聊的消息
    if (data.chat_type === 'private_chat' && data.chat_id === privateChatId.value) {
      // 构造消息对象并添加到 store
      const newMessage: Message = {
        id: data.id,
        type: data.message_type,
        content: data.content || '',
        sender: {
          user_id: data.sender.user_id,
          nickname: data.sender.nickname,
          avatar_url: data.sender.avatar_url,
          user_type: data.sender.user_type as 'temp' | 'normal' | 'admin' | 'system'
        },
        sent_at: data.sent_at,
        is_recalled: false,
        reply_to: data.reply_to,
        media_url: data.media_url,
        thumbnail_url: data.thumbnail_url,
        file_name: data.file_name,
        file_size: data.file_size,
        duration: data.duration,
        width: data.width,
        height: data.height
      };
      
      messagesStore.appendMessage(privateChatId.value!, newMessage);
      
      // 如果是对方的消息，停止显示"正在输入"
      if (data.sender.user_id !== authStore.user?.user_id) {
        otherUserTyping.value = false;
        clearTimeout(otherUserTypingTimer);
      }
      
      if (import.meta.dev) {
        console.log('[WebSocket] 收到新私聊消息:', data.content);
      }
    }
  });
});

// 页面离开时清理
onUnmounted(() => {
  if (typingTimer) {
    clearTimeout(typingTimer);
  }
  if (otherUserTypingTimer) {
    clearTimeout(otherUserTypingTimer);
  }
  // 发送停止打字状态
  if (isTyping.value) {
    sendTypingStatus(false);
  }
  // 清理 WebSocket 事件监听
  if (typingCleanup) {
    typingCleanup();
  }
  if (messageCleanup) {
    messageCleanup();
  }
});
</script>