<!--
  UVirtualChatMessages - 专门为聊天消息优化的虚拟滚动组件
  支持自动滚动到底部、动态高度、消息分组等聊天特性
-->
<template>
  <div class="u-virtual-chat-messages" :class="ui.wrapper">
    <!-- 虚拟滚动容器 -->
    <div
      ref="containerRef"
      :style="{ height: `${containerHeight}px` }"
      class="overflow-auto"
      :class="ui.container"
      @scroll="handleScroll"
    >
      <!-- 虚拟滚动内容 -->
      <div :style="{ height: `${totalHeight}px`, position: 'relative' }">
        <!-- 顶部加载更多 -->
        <div
          v-if="canLoadMore && !loadingMore"
          class="absolute top-0 left-0 right-0 z-10"
        >
          <div class="flex justify-center py-4">
            <UButton
              variant="ghost"
              size="sm"
              icon="i-heroicons-arrow-up"
              @click="$emit('load-more')"
            >
              加载更多消息
            </UButton>
          </div>
        </div>

        <!-- 加载更多状态 -->
        <div
          v-if="loadingMore"
          class="absolute top-0 left-0 right-0 z-10"
        >
          <div class="flex justify-center items-center py-4">
            <UIcon name="i-heroicons-arrow-path" class="animate-spin w-4 h-4 mr-2" />
            <span class="text-sm text-neutral-500 dark:text-neutral-400">加载中...</span>
          </div>
        </div>

        <!-- 消息列表 -->
        <div
          v-for="{ index, item: message, start, key } in visibleItems"
          :key="key"
          :ref="(el) => measureElement(index, el as HTMLElement)"
          :style="{
            position: 'absolute',
            top: 0,
            left: 0,
            width: '100%',
            transform: `translateY(${start + (loadingMore ? 60 : canLoadMore ? 60 : 0)}px)`
          }"
          :class="ui.message.wrapper"
        >
          <!-- 日期分隔符 -->
          <div
            v-if="shouldShowDateSeparator(message, index)"
            :class="ui.dateSeparator.wrapper"
          >
            <div :class="ui.dateSeparator.line" />
            <span :class="ui.dateSeparator.text">
              {{ formatMessageDate(message.created_at || message.sent_at || new Date()) }}
            </span>
            <div :class="ui.dateSeparator.line" />
          </div>

          <!-- 消息内容插槽 -->
          <slot
            :message="message"
            :index="index"
            :is-own-message="isOwnMessage(message)"
            :show-avatar="shouldShowAvatar(message, index)"
            :show-sender="shouldShowSender(message, index)"
          >
            <!-- 默认消息渲染 -->
            <div :class="getMessageClasses(message)">
              <!-- 头像 -->
              <UAvatar
                v-if="shouldShowAvatar(message, index) && !isOwnMessage(message)"
                :alt="message.sender_name"
                size="sm"
                :class="ui.message.avatar"
              >
                {{ message.sender_name?.charAt(0) }}
              </UAvatar>

              <!-- 消息气泡 -->
              <div :class="getMessageBubbleClasses(message)">
                <!-- 发送者名称 -->
                <div
                  v-if="shouldShowSender(message, index) && !isOwnMessage(message)"
                  :class="ui.message.senderName"
                >
                  {{ message.sender_name }}
                </div>

                <!-- 消息内容 -->
                <div :class="ui.message.content">
                  {{ message.content }}
                </div>

                <!-- 消息时间和状态 -->
                <div :class="ui.message.meta">
                  <span :class="ui.message.time">
                    {{ formatMessageTime(message.created_at || message.sent_at || new Date()) }}
                  </span>
                  
                  <!-- 消息状态（仅自己的消息） -->
                  <UIcon
                    v-if="isOwnMessage(message) && (message as ChatMessage).status"
                    :name="getStatusIcon((message as ChatMessage).status!)"
                    :class="getStatusIconClass((message as ChatMessage).status!)"
                  />
                </div>
              </div>
            </div>
          </slot>
        </div>
      </div>
    </div>

    <!-- 滚动到底部按钮 -->
    <Transition
      enter-active-class="transition-all duration-300"
      enter-from-class="opacity-0 scale-90"
      enter-to-class="opacity-100 scale-100"
      leave-active-class="transition-all duration-300"
      leave-from-class="opacity-100 scale-100"
      leave-to-class="opacity-0 scale-90"
    >
      <UButton
        v-if="showScrollToBottom && !isNearBottom"
        :class="ui.scrollToBottom.button"
        size="sm"
        variant="soft"
        icon="i-heroicons-arrow-down"
        @click="scrollToBottom"
      >
        <UBadge
          v-if="unreadCount > 0"
          :label="unreadCount.toString()"
                        color="error"
          size="xs"
          :class="ui.scrollToBottom.badge"
        />
      </UButton>
    </Transition>
  </div>
</template>

<script setup lang="ts">
import { format, isToday, isYesterday, isSameDay } from 'date-fns'
import { zhCN } from 'date-fns/locale'

// 导入 Message 类型
import type { Message } from '~/types/message'

// 扩展 Message 类型以支持聊天特性
interface ChatMessage extends Message {
  status?: 'sending' | 'sent' | 'delivered' | 'read'
  type?: 'text' | 'image' | 'file' | 'system'
}

// Props
interface Props {
  // 消息列表
  messages: Message[]
  // 当前用户ID
  currentUserId: string // 64位整数转换为字符串
  // 容器高度
  containerHeight?: number
  // 估计消息高度
  estimatedMessageHeight?: number
  // 是否可以加载更多
  canLoadMore?: boolean
  // 是否正在加载更多
  loadingMore?: boolean
  // 未读消息数量
  unreadCount?: number
  // 是否显示滚动到底部按钮
  showScrollToBottom?: boolean
  // 是否自动滚动到底部（新消息时）
  autoScrollToBottom?: boolean
  // UI 配置
  ui?: Record<string, unknown>
}

const props = withDefaults(defineProps<Props>(), {
  containerHeight: 400,
  estimatedMessageHeight: 80,
  canLoadMore: false,
  loadingMore: false,
  unreadCount: 0,
  showScrollToBottom: true,
  autoScrollToBottom: true,
  ui: () => ({})
})

// 事件
const emit = defineEmits<{
  'load-more': []
  'message-click': [message: ChatMessage]
  'scroll': [offset: number]
}>()

// 默认 UI 配置
const defaultUI = {
  wrapper: 'relative',
  container: 'scroll-smooth',
  message: {
    wrapper: 'px-4 py-2',
    avatar: 'mr-3',
    senderName: 'text-xs text-neutral-500 mb-1',
    content: 'break-words',
    meta: 'flex items-center justify-end mt-1 gap-1',
    time: 'text-xs text-neutral-400',
  },
  dateSeparator: {
    wrapper: 'flex items-center my-4 px-4',
    line: 'flex-1 h-px bg-neutral-200 dark:bg-neutral-700',
    text: 'px-3 text-xs text-neutral-500 bg-white dark:bg-neutral-900'
  },
  scrollToBottom: {
    button: 'fixed right-4 bottom-20 z-10',
    badge: 'absolute -top-2 -right-2'
  }
}

// 合并 UI 配置
const ui = computed(() => ({
  ...defaultUI,
  ...props.ui
}))

// 虚拟滚动
const messagesRef = toRef(props, 'messages')
const {
  containerRef,
  visibleItems,
  totalHeight,
  scrollToBottom,
  measureElement,
  virtualizer
} = useTanStackVirtualScroll(messagesRef, {
  containerHeight: props.containerHeight,
  estimatedItemHeight: props.estimatedMessageHeight,
  overscan: 10,
  enableDynamicSize: true
})

// 滚动状态
const isNearBottom = ref(true)
const lastMessageCount = ref(0)

// 处理滚动
const handleScroll = () => {
  const container = containerRef.value
  if (!container) return

  const { scrollTop, scrollHeight, clientHeight } = container
  isNearBottom.value = scrollTop + clientHeight >= scrollHeight - 100
  
  emit('scroll', scrollTop)
}

// 消息相关方法
const isOwnMessage = (message: ChatMessage) => {
  return message.sender_id === props.currentUserId
}

const shouldShowAvatar = (message: ChatMessage, index: number) => {
  if (isOwnMessage(message)) return false
  if (index === 0) return true
  
  const prevMessage = props.messages[index - 1]
  return !prevMessage || prevMessage.sender_id !== message.sender_id
}

const shouldShowSender = (message: ChatMessage, index: number) => {
  return shouldShowAvatar(message, index)
}

const shouldShowDateSeparator = (message: ChatMessage, index: number) => {
  if (index === 0) return true
  
  const prevMessage = props.messages[index - 1]
  if (!prevMessage) return true
  
  const currentDate = new Date(message.created_at || message.sent_at || new Date())
  const prevDate = new Date(prevMessage.created_at || prevMessage.sent_at || new Date())
  
  return !isSameDay(currentDate, prevDate)
}

// 格式化方法
const formatMessageDate = (date: string | Date) => {
  const messageDate = new Date(date)
  
  if (isToday(messageDate)) {
    return '今天'
  } else if (isYesterday(messageDate)) {
    return '昨天'
  } else {
    return format(messageDate, 'MM月dd日', { locale: zhCN })
  }
}

const formatMessageTime = (date: string | Date) => {
  return format(new Date(date), 'HH:mm')
}

// 样式方法
const getMessageClasses = (message: ChatMessage) => {
  const baseClasses = 'flex items-end mb-4'
  return isOwnMessage(message) 
    ? `${baseClasses} justify-end` 
    : `${baseClasses} justify-start`
}

const getMessageBubbleClasses = (message: ChatMessage) => {
  const baseClasses = 'max-w-xs lg:max-w-md px-4 py-2 rounded-lg'
  return isOwnMessage(message)
    ? `${baseClasses} bg-primary-500 text-white`
    : `${baseClasses} bg-neutral-100 dark:bg-neutral-800 text-neutral-900 dark:text-neutral-100`
}

const getStatusIcon = (status: string) => {
  const icons = {
    sending: 'i-heroicons-clock',
    sent: 'i-heroicons-check',
    delivered: 'i-heroicons-check-check',
    read: 'i-heroicons-check-check'
  }
  return icons[status as keyof typeof icons] || 'i-heroicons-clock'
}

const getStatusIconClass = (status: string) => {
  const classes = {
    sending: 'w-3 h-3 text-neutral-400',
    sent: 'w-3 h-3 text-neutral-400',
    delivered: 'w-3 h-3 text-primary-400',
    read: 'w-3 h-3 text-success-400'
  }
  return classes[status as keyof typeof classes] || 'w-3 h-3 text-neutral-400'
}

// 自动滚动到底部（新消息时）
watch(() => props.messages.length, (newCount) => {
  if (props.autoScrollToBottom && newCount > lastMessageCount.value && isNearBottom.value) {
    nextTick(() => {
      scrollToBottom()
    })
  }
  lastMessageCount.value = newCount
})

// 初始化时滚动到底部
onMounted(() => {
  if (props.autoScrollToBottom) {
    nextTick(() => {
      scrollToBottom()
    })
  }
})

// 暴露方法给父组件
defineExpose({
  scrollToBottom
})
</script>

<style scoped>
.u-virtual-chat-messages {
  @apply relative;
}
</style>
