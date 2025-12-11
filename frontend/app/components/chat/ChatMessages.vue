<template>
  <div 
    ref="containerRef" 
    class="flex-1 overflow-y-auto p-4 space-y-4"
    @scroll="handleScroll"
  >
    <!-- 加载更多指示器 -->
    <div v-if="status === 'streaming'" class="flex justify-center py-2">
      <UIcon name="i-heroicons-arrow-path" class="icon-md animate-spin text-secondary-content" />
    </div>

    <!-- 消息列表 -->
    <div 
      v-for="(message, index) in messages" 
      :key="message.id || index"
      class="flex gap-3"
      :class="[
        message.role === 'user' ? 'flex-row-reverse' : 'flex-row'
      ]"
    >
      <!-- 头像 -->
      <div class="shrink-0">
        <!-- 群聊模式：使用每条消息的发送者头像 -->
        <UAvatar
          v-if="isGroupChat && message.role === 'assistant' && message.metadata?.senderAvatar"
          :alt="message.metadata.senderName || '用户'"
          :src="message.metadata.senderAvatar"
          size="sm"
          class="cursor-pointer hover:ring-2 hover:ring-accent transition-all"
          @click="handleAvatarClick(message)"
        >
          {{ (message.metadata.senderName || '用户').charAt(0) }}
        </UAvatar>
        <!-- 群聊模式：发送者没有头像时显示首字母 -->
        <UAvatar
          v-else-if="isGroupChat && message.role === 'assistant' && message.metadata?.senderName"
          size="sm"
          class="cursor-pointer hover:ring-2 hover:ring-accent transition-all"
          @click="handleAvatarClick(message)"
        >
          {{ message.metadata.senderName.charAt(0) }}
        </UAvatar>
        <!-- 自己发送的消息 -->
        <UAvatar
          v-else-if="message.role === 'user' && user?.avatar"
          :alt="user.avatar.alt"
          :src="user.avatar.src"
          size="sm"
        >
          {{ user.avatar.fallback }}
        </UAvatar>
        <!-- 私聊模式：对方消息使用统一头像 -->
        <UAvatar
          v-else-if="message.role === 'assistant' && assistant?.avatar"
          :alt="assistant.avatar.alt"
          :src="assistant.avatar.src"
          size="sm"
          :class="isGroupChat ? 'cursor-pointer hover:ring-2 hover:ring-accent transition-all' : ''"
          @click="isGroupChat ? handleAvatarClick(message) : undefined"
        >
          {{ assistant.avatar.fallback }}
        </UAvatar>
        <UAvatar v-else size="sm">
          {{ message.role === 'user' ? 'U' : 'A' }}
        </UAvatar>
      </div>

      <!-- 消息内容 -->
      <div 
        class="max-w-[70%] rounded-2xl px-4 py-2.5 shadow-sm"
        :class="[
          message.role === 'user' 
            ? 'message-bubble-user' 
            : 'message-bubble-other'
        ]"
      >
        <!-- 使用插槽自定义内容 -->
        <slot name="content" :message="message">
          <p class="text-sm whitespace-pre-wrap wrap-break-word">{{ getMessageContent(message) }}</p>
        </slot>

        <!-- 时间戳 -->
        <div 
          v-if="getMessageTimestamp(message)"
          class="text-xs mt-1 opacity-70"
          :class="message.role === 'user' ? 'text-right' : 'text-left'"
        >
          {{ displayTime(getMessageTimestamp(message)!) }}
        </div>
      </div>
    </div>

    <!-- 空状态 -->
    <div v-if="messages.length === 0" class="flex flex-col items-center justify-center h-full text-secondary-content">
      <UIcon name="i-heroicons-chat-bubble-left-right" class="icon-2xl mb-4 opacity-50" />
      <p>暂无消息</p>
    </div>
  </div>
</template>

<script setup lang="ts">
// 消息部分类型（兼容 AI SDK 格式）
export interface MessagePart {
  type: string;
  text?: string;
  content?: string;
}

// 消息元数据
export interface MessageMetadata {
  timestamp?: string;
  type?: string;
  senderName?: string;
  senderAvatar?: string;
  senderId?: number;
  [key: string]: unknown;
}

// 聊天消息类型（兼容 AI SDK 格式和简单格式）
export interface ChatMessage {
  id: string;
  role: 'user' | 'assistant' | 'system';
  // AI SDK 格式
  parts?: MessagePart[];
  metadata?: MessageMetadata;
  // 简单格式
  content?: string;
  type?: string;
  createdAt?: string | Date;
  timestamp?: string;
  [key: string]: unknown;
}

export interface ChatAvatarConfig {
  avatar?: {
    alt?: string;
    src?: string;
    fallback?: string;
  };
  variant?: 'solid' | 'soft' | 'outline' | 'ghost';
  side?: 'left' | 'right';
}

interface Props {
  messages: ChatMessage[];
  status?: 'idle' | 'streaming' | 'submitted' | 'ready' | 'error';
  user?: ChatAvatarConfig;
  assistant?: ChatAvatarConfig;
  shouldAutoScroll?: boolean;
  shouldScrollToBottom?: boolean;
  isGroupChat?: boolean;
}

const props = withDefaults(defineProps<Props>(), {
  status: 'idle',
  shouldAutoScroll: true,
  shouldScrollToBottom: true,
  isGroupChat: false,
});

// 定义事件
const emit = defineEmits<{
  (e: 'avatarClick', message: ChatMessage): void;
}>();

// 处理头像点击
const handleAvatarClick = (message: ChatMessage) => {
  if (props.isGroupChat && message.metadata?.senderId) {
    emit('avatarClick', message);
  }
};

const containerRef = ref<HTMLElement | null>(null);
const isNearBottom = ref(true);

// 获取消息内容（兼容两种格式）
const getMessageContent = (message: ChatMessage): string => {
  // 优先使用 content 字段
  if ('content' in message && message.content) {
    return String(message.content);
  }
  // 否则从 parts 中提取
  if (message.parts && message.parts.length > 0) {
    const textPart = message.parts.find(p => p.type === 'text');
    return textPart?.text || (textPart as { content?: string })?.content || '';
  }
  return '';
};

// 获取消息时间戳
const getMessageTimestamp = (message: ChatMessage): string | Date | undefined => {
  if (message.createdAt) return message.createdAt;
  if (message.timestamp) return message.timestamp;
  if (message.metadata?.timestamp) return message.metadata.timestamp;
  return undefined;
};

// 显示时间
const { formatDate } = useFormatters()
const displayTime = (date: string | Date) => formatDate(date, 'time');

// 处理滚动
const handleScroll = () => {
  if (!containerRef.value) return;
  const { scrollTop, scrollHeight, clientHeight } = containerRef.value;
  isNearBottom.value = scrollHeight - scrollTop - clientHeight < 100;
};

// 滚动到底部
const scrollToBottom = () => {
  if (!containerRef.value) return;
  containerRef.value.scrollTop = containerRef.value.scrollHeight;
};

// 监听消息变化，自动滚动
watch(
  () => props.messages.length,
  () => {
    if (props.shouldAutoScroll && isNearBottom.value) {
      nextTick(() => scrollToBottom());
    }
  }
);

// 初始滚动到底部
onMounted(() => {
  if (props.shouldScrollToBottom) {
    nextTick(() => scrollToBottom());
  }
});

// 暴露方法
defineExpose({
  scrollToBottom,
});
</script>
