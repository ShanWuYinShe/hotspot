<template>
  <UDashboardPanel>
    <!-- 聊天头部 -->
    <template #header>
      <ChatHeader
        :user-info="userInfo"
        :is-online="isUserOnline"
        @back="goBack"
        @toggle-search="toggleSearch"
        @voice-call="makeCall"
        @video-call="makeVideoCall"
        @view-profile="viewUserProfile"
        @clear-history="clearChatHistory"
        @block-user="blockUser"
      />
    </template>

    <!-- 搜索栏 -->
    <div v-if="showSearch" class="p-4 border-b bg-neutral-50 dark:bg-neutral-800">
      <UInput
        v-model="searchQuery"
        placeholder="搜索聊天记录..."
        icon="i-heroicons-magnifying-glass"
        size="sm"
        class="w-full"
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
    </div>

    <!-- 聊天消息区域 -->
    <template #body>
      <div class="h-full flex flex-col">
        <UChatMessages
          :messages="formattedMessages"
          :status="chatStatus"
          :should-auto-scroll="true"
          :should-scroll-to-bottom="true"
          :user="{
            avatar: { 
              alt: currentUser?.nickname,
              fallback: currentUser?.nickname?.charAt(0) || 'U'
            },
            variant: 'soft',
            side: 'right'
          }"
          :assistant="{
            avatar: { 
              alt: userInfo?.nickname,
              fallback: userInfo?.nickname?.charAt(0) || 'U'
            },
            variant: 'naked',
            side: 'left'
          }"
          class="flex-1"
        >
          <!-- 自定义消息内容 -->
          <template #content="{ message }">
            <div class="space-y-2">
              <!-- 文本消息 -->
              <div v-if="(message as any).type === 'text'" class="prose prose-sm dark:prose-invert max-w-none">
                <p>{{ (message as any).content }}</p>
              </div>
              
              <!-- 图片消息 -->
              <div v-else-if="(message as any).type === 'image'" class="max-w-xs">
                <img 
                  :src="(message as any).content" 
                  :alt="(message as any).alt || '图片'"
                  class="rounded-lg cursor-pointer hover:opacity-90 transition-opacity"
                  @click="previewImage((message as any).content)"
                />
              </div>
              
              <!-- 文件消息 -->
              <div v-else-if="(message as any).type === 'file'" class="flex items-center gap-3 p-3 bg-neutral-100 dark:bg-neutral-800 rounded-lg">
                <UIcon name="i-heroicons-document" class="w-8 h-8 text-neutral-500" />
                <div class="flex-1 min-w-0">
                  <p class="font-medium truncate">{{ (message as any).fileName }}</p>
                  <p class="text-sm text-neutral-500">{{ formatFileSize((message as any).fileSize) }}</p>
                </div>
                <UButton
                  variant="ghost"
                  size="xs"
                  icon="i-heroicons-arrow-down-tray"
                  @click="downloadFile((message as any).content, (message as any).fileName)"
                />
              </div>
              
              <!-- 位置消息 -->
              <div v-else-if="(message as any).type === 'location'" class="p-3 bg-neutral-100 dark:bg-neutral-800 rounded-lg">
                <div class="flex items-center gap-2 mb-2">
                  <UIcon name="i-heroicons-map-pin" class="w-4 h-4 text-primary-500" />
                  <span class="font-medium">位置信息</span>
                </div>
                <p class="text-sm text-neutral-600 dark:text-neutral-400">{{ (message as any).address }}</p>
                <UButton
                  variant="ghost"
                  size="xs"
                  class="mt-2"
                  @click="openLocation((message as any).latitude, (message as any).longitude)"
                >
                  查看位置
                </UButton>
              </div>
              
              <!-- 系统消息 -->
              <div v-else-if="(message as any).type === 'system'" class="text-center">
                <span class="text-xs text-neutral-500 bg-neutral-100 dark:bg-neutral-800 px-2 py-1 rounded-full">
                  {{ (message as any).content }}
                </span>
              </div>
            </div>
          </template>

          <!-- 消息操作 -->
          <template #actions="{ message }">
            <div v-if="message.role === 'user'" class="flex items-center gap-1 opacity-0 group-hover:opacity-100 transition-opacity">
              <UButton
                variant="ghost"
                size="xs"
                icon="i-heroicons-arrow-uturn-left"
                @click="replyToMessage(message)"
              />
              <UButton
                variant="ghost"
                size="xs"
                icon="i-heroicons-pencil"
                @click="editMessage(message)"
              />
              <UButton
                variant="ghost"
                size="xs"
                icon="i-heroicons-trash"
                @click="deleteMessage(message)"
              />
            </div>
          </template>

          <!-- 加载指示器 -->
          <template #indicator v-if="isTyping">
            <div class="flex items-center gap-2 text-sm text-neutral-500">
              <div class="flex gap-1">
                <div class="w-2 h-2 bg-neutral-400 rounded-full animate-bounce" style="animation-delay: 0ms"></div>
                <div class="w-2 h-2 bg-neutral-400 rounded-full animate-bounce" style="animation-delay: 150ms"></div>
                <div class="w-2 h-2 bg-neutral-400 rounded-full animate-bounce" style="animation-delay: 300ms"></div>
              </div>
              <span>{{ userInfo?.nickname }} 正在输入...</span>
            </div>
          </template>
        </UChatMessages>
      </div>
    </template>

    <!-- 消息输入区域 -->
    <template #footer>
      <div class="p-4 border-t bg-white dark:bg-neutral-900">
        <!-- 回复消息预览 -->
        <div v-if="replyingTo" class="mb-3 p-2 bg-neutral-100 dark:bg-neutral-800 rounded-lg flex items-center justify-between">
          <div class="flex-1 min-w-0">
            <p class="text-xs text-neutral-500">回复 {{ replyingTo.sender }}</p>
            <p class="text-sm truncate">{{ replyingTo.content }}</p>
          </div>
          <UButton
            variant="ghost"
            size="xs"
            icon="i-heroicons-x-mark"
            @click="cancelReply"
          />
        </div>

        <UChatPrompt
          v-model="messageInput"
          placeholder="输入消息..."
          :error="sendError ? new Error(sendError) : undefined"
          @submit="handleSendMessage"
        >
          <!-- 左侧工具按钮 -->
          <template #leading>
            <div class="flex items-center gap-1">
              <!-- 表情按钮 -->
              <UButton
                variant="ghost"
                size="sm"
                icon="i-heroicons-face-smile"
                @click="toggleEmojiPicker"
              />
              
              <!-- 文件上传 -->
              <UButton
                variant="ghost"
                size="sm"
                icon="i-heroicons-paper-clip"
                @click="openFileUpload"
              />
              
              <!-- 图片上传 -->
              <UButton
                variant="ghost"
                size="sm"
                icon="i-heroicons-photo"
                @click="openImageUpload"
              />
              
              <!-- 位置分享 -->
              <UButton
                variant="ghost"
                size="sm"
                icon="i-heroicons-map-pin"
                @click="shareLocation"
              />
            </div>
          </template>

          <!-- 发送按钮 -->
          <UChatPromptSubmit
            :status="chatStatus"
            @stop="stopSending"
            @reload="reloadChat"
          />
        </UChatPrompt>

        <!-- 表情选择器 -->
        <div v-if="showEmojiPicker" class="mt-2 p-2 bg-neutral-50 dark:bg-neutral-800 rounded-lg">
          <div class="grid grid-cols-8 gap-2">
            <button
              v-for="emoji in commonEmojis"
              :key="emoji"
              class="p-2 hover:bg-neutral-200 dark:hover:bg-neutral-700 rounded text-lg"
              @click="insertEmoji(emoji)"
            >
              {{ emoji }}
            </button>
          </div>
        </div>
      </div>
    </template>
  </UDashboardPanel>

  <!-- 文件上传 -->
  <input
    ref="fileInput"
    type="file"
    class="hidden"
    multiple
    @change="handleFileUpload"
  />

  <!-- 图片上传 -->
  <input
    ref="imageInput"
    type="file"
    accept="image/*"
    class="hidden"
    multiple
    @change="handleImageUpload"
  />
</template>

<script setup lang="ts">
// 需要登录 - 私聊功能需要认证
definePageMeta({
  title: "私聊",
  requiresAuth: true,
  layout: "dashboard"
});

const route = useRoute();
const userId = route.params.id as string;

// 使用现有的 stores 和 composables
const messagesStore = useMessagesStore();
const authStore = useAuthStore();

// 导入服务
import { useUsersService } from '~/services/users.service';
import { useMessagesService } from '~/services/messages.service';

const userService = useUsersService();
const messageService = useMessagesService();
const toast = useToast();
const modal = useModal();
const { formatFileSize } = useFormatters();

// 响应式状态
const messageInput = ref('');
const showSearch = ref(false);
const searchQuery = ref('');
const showEmojiPicker = ref(false);
const isTyping = ref(false);
const replyingTo = ref<any>(null);
const sendError = ref<string | null>(null);
const chatStatus = ref<'ready' | 'submitted' | 'streaming' | 'error'>('ready');

// 文件上传引用
const fileInput = ref<HTMLInputElement>();
const imageInput = ref<HTMLInputElement>();

// 计算属性
const currentUser = computed(() => authStore.user);
// 用户信息状态
const userInfoData = ref<any>(null);

// 异步加载用户信息
const loadUserInfo = async () => {
  try {
    const response = await userService.openTheDoor({ id: route.params.id as string });
    userInfoData.value = response;
  } catch (error) {
    console.error('加载用户信息失败:', error);
  }
};

const userInfo = computed(() => {
  return userInfoData.value || { 
    user_id: route.params.id as string,
    nickname: '未知用户', 
    isOnline: false, 
    lastActiveAt: null 
  };
});
const messages = computed(() => {
  // 转换消息格式以适配 UChatMessages 组件
  const conversationMessages = messagesStore.messages.get(userId) || [];
  return conversationMessages.map((msg: any) => ({
    id: msg.message_id,
    role: msg.sender_id === authStore.user?.user_id ? 'user' : 'assistant',
    parts: [{ type: 'text', text: msg.content }],
    ...msg
  }));
});

// 筛选消息（搜索功能）
const filteredMessages = computed(() => {
  if (!searchQuery.value.trim()) {
    return messages.value;
  }
  
  return messages.value.filter((message: any) => 
    message.content?.toLowerCase().includes(searchQuery.value.toLowerCase())
  );
});

// 格式化消息以适配 UChatMessages
const formattedMessages = computed(() => {
  return filteredMessages.value.map((msg: any) => ({
    id: msg.id,
    role: msg.role,
    parts: msg.parts || [{ type: 'text', text: msg.content }],
    metadata: {
      timestamp: msg.timestamp,
      type: msg.type
    }
  }));
});

// 获取消息内容
const getMessageContent = (message: any) => {
  if (message.parts && message.parts.length > 0) {
    return message.parts[0].text || message.parts[0].content || '';
  }
  return message.content || '';
};

// 聊天菜单项
const chatMenuItems = computed(() => [
  [{
    label: '查看用户资料',
    icon: 'i-heroicons-user',
    click: () => navigateTo(`/users/${userId}`)
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
    class: 'text-error-500'
  }]
]);

// 常用表情
const commonEmojis = ['😀', '😂', '😍', '🥰', '😊', '😎', '🤔', '😢', '😭', '😡', '👍', '👎', '❤️', '💯', '🔥', '✨'];

// 方法
const goBack = () => {
  navigateTo('/chat');
};

const getOnlineStatus = () => {
  if (!userInfo.value) return '离线';
  if (userInfo.value.isOnline) return '在线';
  if (userInfo.value.lastActiveAt) {
    const lastActive = new Date(userInfo.value.lastActiveAt);
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

const handleSendMessage = async (event: Event) => {
  event.preventDefault();
  
  if (!messageInput.value.trim()) return;
  
  try {
    chatStatus.value = 'submitted';
    sendError.value = null;
    
    const messageData = {
      content: messageInput.value.trim(),
      type: 'text',
      replyTo: replyingTo.value?.id
    };
    
    // 添加消息到 store
    const newMessage = {
      message_id: Date.now().toString(),
      content: messageData.content,
      sender_id: authStore.user?.user_id,
      timestamp: new Date().toISOString(),
      type: 'text'
    };
    
    // 发送消息到后端
    try {
      await messagesStore.sendPrivateMessage(userId, {
        type: 'text',
        content: newMessage.content
      });
    } catch (error) {
      console.error('发送消息失败:', error);
    }
    
    messageInput.value = '';
    replyingTo.value = null;
    chatStatus.value = 'ready';
    
  } catch (error: any) {
    console.error('发送消息失败:', error);
    sendError.value = error.message || '发送失败';
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
  fileInput.value?.click();
};

const openImageUpload = () => {
  imageInput.value?.click();
};

const handleFileUpload = async (event: Event) => {
  const files = (event.target as HTMLInputElement).files;
  if (!files) return;
  
  for (const file of Array.from(files)) {
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
};

const handleImageUpload = async (event: Event) => {
  const files = (event.target as HTMLInputElement).files;
  if (!files) return;
  
  for (const file of Array.from(files)) {
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
    
    // 发送位置消息
    const locationMessage = {
      message_id: Date.now().toString(),
      content: `位置：${latitude},${longitude}`,
      sender_id: authStore.user?.user_id,
      timestamp: new Date().toISOString(),
      type: 'location'
    };
    
    // 发送位置消息到后端
    try {
      await messagesStore.sendPrivateMessage(userId, {
        type: 'text',
        content: `📍 位置: ${latitude}, ${longitude}`
      });
    } catch (error) {
      console.error('发送位置消息失败:', error);
    }
    
  } catch (error) {
    console.error('获取位置失败:', error);
    toast.add({
      title: '位置分享失败',
      description: '无法获取当前位置',
      color: 'error'
    });
  }
};

const replyToMessage = (message: any) => {
  replyingTo.value = message;
};

const cancelReply = () => {
  replyingTo.value = null;
};

const editMessage = (message: any) => {
  // 实现消息编辑功能
  messageInput.value = message.content;
  replyingTo.value = message;
};

const deleteMessage = async (message: any) => {
  if (confirm('确定要删除这条消息吗？')) {
    try {
      await messagesStore.deleteMessage(userId, message.id, false);
    } catch (error) {
      console.error('删除消息失败:', error);
    }
  }
};

// 图片预览状态
const currentImagePreview = ref<string | null>(null);
const showImagePreview = ref(false);

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
  if (confirm('确定要清空与此用户的所有聊天记录吗？此操作无法撤销。')) {
    try {
      messagesStore.removeConversation(userId);
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
  if (confirm(`确定要屏蔽用户 ${userInfo.value?.nickname} 吗？屏蔽后将无法收到对方的消息。`)) {
    try {
      await userService.blockUser({ user_id: userId });
      toast.add({
        title: '用户已屏蔽',
        color: 'success'
      });
      navigateTo('/chat');
    } catch (error) {
      console.error('屏蔽用户失败:', error);
    }
  }
};

const loadMessages = async () => {
  try {
    // 加载聊天数据
    await messagesStore.fetchPrivateMessages(userId);
    await loadUserInfo();
  } catch (error) {
    console.error('加载聊天数据失败:', error);
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

// 监听输入状态（模拟打字指示器）
let typingTimer: NodeJS.Timeout;
watch(messageInput, (newValue, oldValue) => {
  if (newValue !== oldValue && newValue.length > 0) {
    // 发送正在输入状态
    // 发送打字状态
    console.log('发送打字状态:', userId);
    
    clearTimeout(typingTimer);
    typingTimer = setTimeout(() => {
      // 停止打字状态
      console.log('停止打字状态:', userId);
    }, 3000);
  }
});

// 缺失的方法实现
const isUserOnline = computed(() => {
  // 简单的在线状态检查
  return false;
});

const viewUserProfile = () => {
  // 查看用户资料
  navigateTo(`/users/${route.params.id}`);
};

// 页面离开时清理
onUnmounted(() => {
  if (typingTimer) {
    clearTimeout(typingTimer);
  }
  // 停止打字状态
  console.log('停止打字状态:', userId);
});
</script>