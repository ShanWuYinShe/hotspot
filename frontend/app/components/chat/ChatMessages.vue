<template>
  <div class="h-full min-h-0 flex flex-col">
    <!-- 空状态 -->
    <UEmpty
      v-if="messages.length === 0"
      icon="i-lucide-messages-square"
      title="暂无消息"
      description="发送一条消息开始对话"
      class="h-full"
    >
      <template v-if="$slots.empty" #default>
        <slot name="empty" />
      </template>
    </UEmpty>

    <!-- 消息列表容器 -->
    <div
      v-else
      ref="containerRef"
      class="flex-1 overflow-y-auto p-4 space-y-6 scroll-smooth"
    >
      <!-- 消息项 -->
      <div
        v-for="(message, index) in messages"
        :key="message.id"
        :data-message-id="message.id"
        class="group/message flex gap-3 w-full"
        :class="[
          message.role === 'user' ? 'flex-row-reverse' : 'flex-row',
          highlightMessageId === message.id ? 'hs-chat-highlight -mx-4 px-4 py-2' : ''
        ]"
      >
        <!-- 头像 (对方) -->
        <div v-if="message.role !== 'user'" class="shrink-0 flex flex-col justify-end pb-1">
          <slot name="leading" :message="message">
            <UAvatar
              :src="message.metadata?.senderAvatar || assistant?.avatar?.src"
              :alt="message.metadata?.senderName || assistant?.avatar?.alt || assistant?.avatar?.fallback"
              :fallback="message.metadata?.senderName?.charAt(0) || assistant?.avatar?.fallback"
              size="md"
              :class="isGroupChat ? 'cursor-pointer' : ''"
              @click="isGroupChat ? emit('avatarClick', message) : undefined"
            />
          </slot>
        </div>

        <!-- 内容区域 -->
        <div
          class="flex flex-col gap-1 max-w-[85%] sm:max-w-[75%]"
          :class="message.role === 'user' ? 'items-end' : 'items-start'"
        >
          <!-- 群聊发送者昵称 (仅在左侧且是群聊时显示) -->
          <UBadge
            v-if="isGroupChat && message.role !== 'user' && message.metadata?.senderName"
            variant="subtle"
            color="neutral"
            size="xs"
            class="ml-1"
          >
            {{ message.metadata.senderName }}
          </UBadge>

          <!-- 气泡 & 操作栏容器 -->
          <div class="relative group/bubble">
            <!-- 气泡 -->
            <div :class="message.role === 'user' ? 'hs-chat-bubble-user' : 'hs-chat-bubble-other'">
              <slot name="content" :message="message">
                {{ message.content }}
              </slot>
            </div>

            <!-- 操作栏 (悬浮) -->
            <div
              v-if="$slots.actions"
              class="absolute top-0 -translate-y-1/2 opacity-0 group-hover/bubble:opacity-100 transition-opacity z-10 hidden sm:block"
              :class="message.role === 'user' ? 'right-full mr-2' : 'left-full ml-2'"
            >
              <UCard variant="outline" :ui="{ body: 'p-1 flex gap-0.5' }">
                <slot name="actions" :message="message" />
              </UCard>
            </div>
          </div>

          <!-- 时间戳 -->
          <UBadge
            v-if="getMessageTimestamp(message)"
            variant="subtle"
            color="neutral"
            size="xs"
            class="px-1 select-none"
          >
            {{ displayTime(getMessageTimestamp(message)!) }}
          </UBadge>
        </div>

        <!-- 头像 (我) -->
        <div v-if="message.role === 'user'" class="shrink-0 flex flex-col justify-end pb-1">
          <UAvatar
            :src="user?.avatar?.src"
            :alt="user?.avatar?.alt || user?.avatar?.fallback"
            :fallback="user?.avatar?.fallback"
            size="md"
          />
        </div>
      </div>

      <!-- 输入状态指示器 (typing) -->
      <div v-if="$slots.indicator" class="pb-2">
        <slot name="indicator" />
      </div>

      <!-- 底部锚点 -->
      <div ref="bottomRef" class="h-px w-full" />
    </div>
  </div>
</template>

<script setup lang="ts">
import type { ClientMessageType } from '~/types/client';

// 聊天消息类型
export interface ChatMessageMetadata {
  senderId?: number | string
  senderName?: string
  senderAvatar?: string
  timestamp?: string | Date
}

export interface ChatMessage {
  id: string;
  role: 'user' | 'assistant' | 'system';
  content?: string;
  type?: ClientMessageType | 'typing'; // 支持 typing 类型
  createdAt?: string | Date;
  timestamp?: string;
  metadata?: ChatMessageMetadata;
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
  highlightMessageId?: string;
}

const props = withDefaults(defineProps<Props>(), {
  status: 'idle',
  shouldAutoScroll: true,
  shouldScrollToBottom: true,
  isGroupChat: false,
});

const emit = defineEmits<{
  (e: 'avatarClick', message: ChatMessage): void;
}>();

const containerRef = ref<HTMLElement | null>(null);
const bottomRef = ref<HTMLElement | null>(null);

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

const scrollToBottom = (behavior: ScrollBehavior = 'smooth') => {
  if (!bottomRef.value) return;
  bottomRef.value.scrollIntoView({ block: 'end', behavior });
};

const scrollToMessage = (id: string, behavior: ScrollBehavior = 'smooth') => {
  if (!containerRef.value) return;
  if (!id) return;
  
  const target = containerRef.value.querySelector(`[data-message-id="${id}"]`) as HTMLElement | null;
  if (!target) {
    // 可能是消息还没渲染，尝试下一帧
    nextTick(() => {
        const retryTarget = containerRef.value?.querySelector(`[data-message-id="${id}"]`) as HTMLElement | null;
        retryTarget?.scrollIntoView({ block: 'center', behavior });
    })
    return;
  }
  
  target.scrollIntoView({ block: 'center', behavior });
};

// 自动滚动逻辑
watch(
  () => props.messages.length,
  (newLen, oldLen) => {
    if (props.shouldAutoScroll && newLen > oldLen) {
      nextTick(() => scrollToBottom('smooth'));
    }
  }
);

// 初始滚动
onMounted(() => {
  if (props.shouldScrollToBottom) {
    nextTick(() => scrollToBottom('auto')); // 初始加载不需要动画
  }
});

// 暴露方法给父组件
defineExpose({
  scrollToBottom,
  scrollToMessage
});
</script>