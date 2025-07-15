<template>
  <div class="chat-page">
    <!-- 响应式头部 -->
    <div class="chat-header compact-only">
      <div class="chat-header-content">
        <BackButton size="small" />
        <div class="header-info">
          <NAvatar :size="32" :src="peerInfo?.avatar_url">
            {{ peerInfo?.nickname?.charAt(0) || 'U' }}
          </NAvatar>
          <div class="header-details">
            <h1 class="chat-title">{{ peerInfo?.nickname || '私聊' }}</h1>
            <span class="status-text">{{ peerOnlineStatus ? '在线' : '离线' }}</span>
          </div>
        </div>
        <NButton link @click="handleMore" class="more-btn">
          <Icon name="mdi:dots-vertical" size="18" />
        </NButton>
      </div>
    </div>

    <!-- 桌面端布局 -->
    <div class="desktop-layout desktop-only">
      <div class="chat-layout">
        <!-- 左侧：对话信息面板 -->
        <div class="chat-sidebar">
          <div v-if="peerInfo" class="peer-profile">
            <div class="peer-avatar">
              <NAvatar :size="80" :src="peerInfo.avatar_url">
                {{ peerInfo.nickname?.charAt(0) || 'U' }}
              </NAvatar>
              <div v-if="peerOnlineStatus" class="online-indicator"></div>
            </div>

            <div class="peer-info">
              <h3 class="peer-name heading-lg text-gradient-fire">{{ peerInfo.nickname }}</h3>
              <div class="peer-status">
                <span class="status-text text-responsive-base text-electric">
                  {{ peerOnlineStatus ? '在线' : '离线' }}
                </span>
                <span v-if="!isConnected" class="connection-status text-responsive-sm text-muted">
                  (连接断开)
                </span>
              </div>
            </div>

            <div class="peer-actions">
              <NButton @click="handleUserProfile" class="action-button text-responsive-sm"
                v-tooltip="{ category: 'chat', key: 'peerProfile' }">
                <Icon name="mdi:account" size="16" />
                <span class="text-bright">查看资料</span>
              </NButton>
              <NButton @click="handleVideoCall" class="action-button text-responsive-sm"
                v-tooltip="{ category: 'chat', key: 'videoCall' }">
                <Icon name="mdi:video" size="16" />
                <span class="text-bright">视频通话</span>
              </NButton>
              <NButton @click="handleMore" class="action-button text-responsive-sm"
                v-tooltip="{ category: 'chat', key: 'moreActions' }">
                <Icon name="mdi:dots-horizontal" size="16" />
                <span class="text-bright">更多</span>
              </NButton>
            </div>
          </div>

          <!-- 聊天功能 -->
          <div class="chat-features">
            <h4 class="heading-base text-electric">聊天功能</h4>
            <div class="feature-buttons">
              <NButton class="feature-button text-responsive-sm" @click="showImageUploadDialog = true"
                v-tooltip="{ category: 'chat', key: 'sendImage' }">
                <Icon name="mdi:image" size="16" />
                <span class="text-bright">发送图片</span>
              </NButton>
              <NButton class="feature-button text-responsive-sm" @click="showLocationShareDialog = true"
                v-tooltip="{ category: 'chat', key: 'shareLocation' }">
                <Icon name="mdi:map-marker" size="16" />
                <span class="text-bright">分享位置</span>
              </NButton>
              <NButton class="feature-button text-responsive-sm" @click="showEmojiPickerDialog = true"
                v-tooltip="{ category: 'chat', key: 'emojiPicker' }">
                <Icon name="mdi:emoticon-happy" size="16" />
                <span class="text-bright">表情</span>
              </NButton>
            </div>
          </div>

          <!-- 快速操作 -->
          <div class="quick-actions">
            <h4>快速操作</h4>
            <div class="action-list">
              <NButton text @click="clearChat" v-tooltip="{ category: 'chat', key: 'clearChat' }">
                <Icon name="mdi:delete" size="16" />
                <span>清空聊天记录</span>
              </NButton>
              <NButton text @click="blockUser" v-tooltip="{ category: 'chat', key: 'blockUser' }">
                <Icon name="mdi:block-helper" size="16" />
                <span>屏蔽用户</span>
              </NButton>
            </div>
          </div>
        </div>

        <!-- 右侧：聊天主区域 -->
        <div class="chat-main">
          <!-- 聊天头部信息 -->
          <div class="chat-header">
            <div class="header-info">
              <NAvatar :size="40" :src="peerInfo?.avatar_url">
                {{ peerInfo?.nickname?.charAt(0) || 'U' }}
              </NAvatar>
              <div class="header-details">
                <h2 class="chat-title heading-lg text-gradient-fire">{{ peerInfo?.nickname || '私聊' }}</h2>
                <div class="chat-status">
                  <span v-if="peerOnlineStatus" class="online-indicator"></span>
                  <span class="status-text text-responsive-base text-electric">
                    {{ peerOnlineStatus ? '在线' : '离线' }}
                  </span>
                </div>
              </div>
            </div>

            <div class="header-actions">
              <NButton link @click="handleUserProfile" :icon="User"
                v-tooltip="{ category: 'chat', key: 'peerProfile' }">
                查看资料
              </NButton>
              <NButton link @click="handleMore" :icon="MoreFilled"
                v-tooltip="{ category: 'chat', key: 'moreActions' }">
                更多
              </NButton>
            </div>
          </div>

          <!-- 消息列表区域 -->
          <div class="messages-area" ref="messagesContainer">
            <!-- 连接状态提示 -->
            <ConnectionStatus />

            <!-- 加载状态 -->
            <div v-if="loading && !messages.length" class="loading-state">
              <NSkeleton :rows="5" animated />
            </div>

            <!-- 历史消息加载 -->
            <div v-if="hasMoreMessages" class="load-more-section">
              <NButton @click="loadMoreMessages" :loading="loadingMore" size="small" text
                v-tooltip="{ category: 'chat', key: 'loadMore' }">
                加载更多消息
              </NButton>
            </div>

            <!-- 消息列表 -->
            <div class="messages-list">
              <div v-for="message in messages" :key="message.message_id" class="message-item" :class="{
                'message-sent': message.sender_id === currentUserId,
                'message-received': message.sender_id !== currentUserId
              }">
                <!-- 消息气泡 -->
                <div class="message-bubble">
                  <div class="message-content">
                    <!-- 文本消息 -->
                    <p v-if="message.message_type === 'text'" class="message-text">{{ message.content }}</p>

                    <!-- 图片消息 -->
                    <div v-else-if="message.message_type === 'image'" class="message-image">
                      <NImage :src="message.content" :preview-src-list="[message.content]" fit="cover"
                        class="chat-image" />
                    </div>

                    <!-- 位置消息 -->
                    <div v-else-if="message.message_type === 'location'" class="message-location">
                      <div class="location-content">
                        <Icon name="mdi:map-marker" size="16" />
                        <div class="location-info">
                          <div class="location-name">{{ JSON.parse(message.content).name }}</div>
                          <div class="location-address">{{ JSON.parse(message.content).address }}</div>
                        </div>
                      </div>
                    </div>
                  </div>

                  <div class="message-meta">
                    <span class="message-time">{{ formatTime(message.sent_at) }}</span>
                    <div v-if="message.sender_id === currentUserId" class="message-status">
                      <Icon v-if="message.read_at" name="mdi:check" size="12" class="read-icon" />
                      <Icon v-else name="mdi:clock" size="12" class="sent-icon" />
                    </div>
                  </div>
                </div>
              </div>
            </div>

            <!-- 空状态 -->
            <div v-if="!loading && messages.length === 0" class="empty-state">
              <div class="empty-icon">💬</div>
              <h3>开始聊天</h3>
              <p>发送第一条消息开始对话吧</p>
            </div>
          </div>

          <!-- 消息输入区域 -->
          <div class="message-input-area">
            <div class="input-toolbar">
              <NButton link @click="showEmojiPickerDialog = true" :icon="ChatDotRound" title="选择表情"
                v-tooltip="{ category: 'chat', key: 'emojiPicker' }" />
              <NButton link @click="showImageUploadDialog = true" :icon="Picture" title="发送图片"
                v-tooltip="{ category: 'chat', key: 'sendImage' }" />
              <NButton link @click="showLocationShareDialog = true" :icon="Location" title="发送位置"
                v-tooltip="{ category: 'chat', key: 'shareLocation' }" />
            </div>

            <div class="input-container">
              <NInput v-model="messageInput" placeholder="输入消息..." @keyup.enter="handleSendMessage" :disabled="!isConnected"
                class="message-input" type="textarea" :autosize="{ minRows: 1, maxRows: 4 }" resize="none"
                v-tooltip="{ category: 'chat', key: 'messageInput' }" />

              <NButton type="primary" @click="handleSendMessage" :loading="sending"
                :disabled="!messageInput.trim() || !isConnected" class="send-button"
                v-tooltip="{ category: 'chat', key: 'sendBtn' }">
                发送
              </NButton>
            </div>
          </div>
        </div>
      </div>
    </div>

    <!-- 响应式聊天布局 -->
    <div class="chat-layout">
      <div class="chat-container">
        <!-- 聊天头部信息 -->
        <div v-if="peerInfo" class="chat-header">
          <div class="peer-info">
            <NAvatar :size="40" :src="peerInfo.avatar_url">
              {{ peerInfo.nickname?.charAt(0) || 'U' }}
            </NAvatar>
            <div class="peer-details">
              <h2 class="peer-name">{{ peerInfo.nickname }}</h2>
              <div class="peer-status">
                <span v-if="peerOnlineStatus" class="online-indicator"></span>
                <span class="status-text">
                  {{ peerOnlineStatus ? '在线' : '离线' }}
                </span>
                <span v-if="!isConnected" class="connection-status">
                  (连接断开)
                </span>
              </div>
            </div>
          </div>

          <div class="chat-actions">
            <NButton link @click="handleUserProfile" :icon="User">
              查看资料
            </NButton>
            <NButton link @click="handleMore" :icon="MoreFilled">
              更多
            </NButton>
          </div>
        </div>

        <!-- 消息列表区域 -->
        <div class="messages-area" ref="messagesContainer">
          <!-- 连接状态提示 -->
          <ConnectionStatus />

          <!-- 加载状态 -->
          <div v-if="loading && !messages.length" class="loading-state">
            <NSkeleton :rows="5" animated />
          </div>

          <!-- 历史消息加载 -->
          <div v-if="hasMoreMessages" class="load-more-section">
            <NButton @click="loadMoreMessages" :loading="loadingMore" size="small" text>
              加载更多消息
            </NButton>
          </div>

          <!-- 消息列表 -->
          <div class="messages-list">
            <div v-for="message in messages" :key="message.message_id" class="message-item" :class="{
              'message-sent': message.sender_id === currentUserId,
              'message-received': message.sender_id !== currentUserId
            }">
              <!-- 消息气泡 -->
              <div class="message-bubble">
                <div class="message-content">
                  <!-- 文本消息 -->
                  <p v-if="message.message_type === 'text'" class="message-text">{{ message.content }}</p>

                  <!-- 图片消息 -->
                  <div v-else-if="message.message_type === 'image'" class="message-image">
                    <NImage :src="message.content" :preview-src-list="[message.content]" fit="cover"
                      class="chat-image" />
                  </div>

                  <!-- 位置消息 -->
                  <div v-else-if="message.message_type === 'location'" class="message-location">
                    <div class="location-content">
                      <Icon name="mdi:map-marker" size="16" />
                      <div class="location-info">
                        <div class="location-name">{{ JSON.parse(message.content).name }}</div>
                        <div class="location-address">{{ JSON.parse(message.content).address }}</div>
                      </div>
                    </div>
                  </div>
                </div>

                <div class="message-meta">
                  <span class="message-time">{{ formatTime(message.sent_at) }}</span>
                  <div v-if="message.sender_id === currentUserId" class="message-status">
                    <Icon v-if="message.read_at" name="mdi:check" size="12" class="read-icon" />
                    <Icon v-else name="mdi:clock" size="12" class="sent-icon" />
                  </div>
                </div>
              </div>
            </div>
          </div>

          <!-- 空状态 -->
          <div v-if="!loading && messages.length === 0" class="empty-state">
            <div class="empty-icon">💬</div>
            <h3>开始聊天</h3>
            <p>发送第一条消息开始对话吧</p>
          </div>
        </div>

        <!-- 消息输入区域 -->
        <div class="message-input-area">
          <div class="input-toolbar">
            <NButton link @click="showEmojiPickerDialog = true" :icon="ChatDotRound" title="选择表情" />
            <NButton link @click="showImageUploadDialog = true" :icon="Picture" title="发送图片" />
            <NButton link @click="showLocationShareDialog = true" :icon="Location" title="发送位置" />
          </div>

          <div class="input-container">
            <NInput v-model="messageInput" placeholder="输入消息..." @keyup.enter="handleSendMessage" :disabled="!isConnected"
              class="message-input" type="textarea" :autosize="{ minRows: 1, maxRows: 4 }" resize="none" />

            <NButton type="primary" @click="handleSendMessage" :loading="sending"
              :disabled="!messageInput.trim() || !isConnected" class="send-button">
              发送
            </NButton>
          </div>
        </div>
      </div>
    </div>

    <!-- 新组件 -->
    <!-- 表情选择器 -->
    <CommonEmojiPicker v-model:visible="showEmojiPickerDialog" @select="handleEmojiSelect" />

    <!-- 图片上传组件 -->
    <CommonImageUploader v-model:visible="showImageUploadDialog" :multiple="false" :max-files="1"
      @upload-success="handleImageUploadSuccess" />

    <!-- 位置分享组件 -->
    <CommonLocationShare v-model:visible="showLocationShareDialog" @location-selected="handleLocationSelected" />

    <!-- 二维码生成器 -->
    <CommonQRCodeGenerator v-model:visible="showQRCodeDialog" :content="shareOptions.link"
      :title="`与 ${peerInfo?.nickname || '好友'} 的聊天`" :description="'扫描二维码加入聊天'" />

    <!-- 微信分享组件 -->
    <CommonWechatShare v-model:visible="showWechatShareDialog" title="分享聊天" :share-options="shareOptions" />

    <!-- 创建群组对话框 -->
    <CreateGroupDialog v-model="showCreateGroupDialog" @success="handleCreateGroupSuccess" />

    <!-- 新消息对话框 -->
    <NewMessageDialog v-model="showNewMessageDialog" @success="handleNewMessageSuccess" />
  </div>
</template>

<script setup lang="ts">

import { NAvatar, NButton, NSkeleton, NImage, NInput, useMessage, useDialog } from 'naive-ui'
import { Person as User, EllipsisHorizontal as MoreFilled, Location, Image as Picture, Chatbubble as ChatDotRound } from '@vicons/ionicons5'
import { nextTick } from 'vue'
// 项目组件导入
import { useAuthStore } from '~/stores/auth'
import { useMessageStore } from '~/stores/message'
import { useUserStore } from '~/stores/user'
import { useMessageService, useUserService, useWechatService } from '~/services'
import { isUserOnline as checkUserOnlineStatus } from '~/utils/userStatus'
import type { DirectMessage } from '~/types/models/message'
import ConnectionStatus from '~/components/common/ConnectionStatus.vue'
import CreateGroupDialog from '~/components/group/CreateGroupDialog.vue'
import NewMessageDialog from '~/components/message/NewMessageDialog.vue'
import type { UserProfile } from '~/types/models/user'

// API 实例
const message = useMessage()
const dialog = useDialog()

// 服务实例
const { getDirectMessages, sendDirectMessage } = useMessageService()
const { searchUsersById } = useUserService()
const wechatService = useWechatService()

// 页面配置
definePageMeta({
  title: '私聊',
  middleware: 'auth'
})

// 路由参数
const route = useRoute()
const router = useRouter()
const conversationId = computed(() => route.params.id as string)
const userId = computed(() => route.query.userId as string)

// 状态管理
const authStore = useAuthStore()
const messageStore = useMessageStore()
const userStore = useUserStore()

// 页面状态
const loading = ref(false)
const loadingMore = ref(false)
const sending = ref(false)
const messageInput = ref('')
const messagesContainer = ref<HTMLElement>()

// 数据状态
const messages = ref<DirectMessage[]>([])
const peerInfo = ref<UserProfile | null>(null)
const hasMoreMessages = ref(true)

// WebSocket 连接状态
const isConnected = ref(true) // 简化实现，暂时设为true
const peerOnlineStatus = computed(() => false) // 暂时返回false，等待API完善

// 新组件状态
const showEmojiPickerDialog = ref(false)
const showImageUploadDialog = ref(false)
const showLocationShareDialog = ref(false)
const showWechatShareDialog = ref(false)
const showQRCodeDialog = ref(false)
const showCreateGroupDialog = ref(false)
const showNewMessageDialog = ref(false)

// 微信分享选项
const shareOptions = ref({
  title: computed(() => `与 ${peerInfo.value?.nickname || peerInfo.value?.user_id || '好友'} 的聊天`),
  desc: computed(() => `来自HotSpot的私人聊天邀请`),
  link: computed(() => `${window.location.origin}/chat?userId=${userId.value}`),
  imgUrl: computed(() => peerInfo.value?.avatar_url || useRuntimeConfig().public.shareDefaultImage || '')
})

// 计算属性
const currentUserId = computed(() => authStore.user?.user_id)
const chatTitle = computed(() => peerInfo.value?.nickname || '私聊')

// 更新peerInfo中的在线状态
const updatePeerOnlineStatus = () => {
  // 暂时注释掉，等待API完善
}

// 发送消息
const initChat = async () => {
  loading.value = true

  try {
    // 如果有用户ID，先获取用户信息
    if (userId.value) {
      peerInfo.value = await searchUsersById({
        user_id: parseInt(userId.value)
      })

      // 更新在线状态
      updatePeerOnlineStatus()

      // 简化WebSocket订阅逻辑
      if (isConnected.value) {
        console.log(`订阅用户状态: user:${userId.value}`)
      }
    }

    // 加载消息历史
    await loadMessages()

    // 滚动到底部
    nextTick(() => {
      scrollToBottom()
    })
  } catch (error) {
    console.error('初始化聊天失败:', error)
    message.error('加载聊天记录失败')
  } finally {
    loading.value = false
  }
}

// 加载消息
const loadMessages = async (cursor?: number) => {
  try {
    if (!userId.value) {
      throw new Error('用户ID缺失')
    }

    const response = await getDirectMessages({
      conversation_id: userId.value,
      cursor: cursor
    })

    if (cursor) {
      messages.value = [...(response.items as any), ...messages.value]
    } else {
      messages.value = response.items as any
    }

    hasMoreMessages.value = response.has_more
  } catch (error) {
    console.error('加载消息失败:', error)
    throw error
  }
}

// 加载更多消息
const loadMoreMessages = async () => {
  if (!hasMoreMessages.value || loadingMore.value) return

  loadingMore.value = true
  try {
    await loadMessages(messages.value.length)
  } catch (error) {
    message.error('加载更多消息失败')
  } finally {
    loadingMore.value = false
  }
}

// 发送消息
const handleSendMessage = async () => {
  if (!messageInput.value.trim() || sending.value) return

  sending.value = true
  try {
    if (!userId.value) {
      throw new Error('用户ID缺失')
    }

    // 调用API发送消息
    const response = await sendDirectMessage({
      recipient_id: userId.value,
      content: messageInput.value.trim(),
      message_type: 'text',
      latitude: 0,
      longitude: 0
    })

    // 清空输入框
    messageInput.value = ''

    // 重新加载消息列表
    await loadMessages()

    // 滚动到底部
    nextTick(() => {
      scrollToBottom()
    })

    message.success('消息发送成功')
  } catch (error) {
    console.error('发送消息失败:', error)
    message.error('发送消息失败')
  } finally {
    sending.value = false
  }
}

// 处理换行
const handleNewLine = () => {
  messageInput.value += '\n'
}

// 表情选择处理
const handleEmojiSelect = (emoji: { char: string; name: string; category: string }) => {
  messageInput.value += emoji.char
  showEmojiPickerDialog.value = false
  // 聚焦到输入框
  nextTick(() => {
    const inputEl = document.querySelector('.message-input textarea')
    if (inputEl) {
      (inputEl as HTMLTextAreaElement).focus()
    }
  })
}

// 图片上传成功处理
const handleImageUploadSuccess = (files: any[]) => {
  if (files.length > 0) {
    const file = files[0]
    showImageUploadDialog.value = false
    // 发送图片消息
    handleSendImageMessage(file.url)
  }
}

// 位置选择处理
const handleLocationSelected = (location: any) => {
  showLocationShareDialog.value = false
  // 发送位置消息
  handleSendLocationMessage(location)
}

// 发送图片消息
const handleSendImageMessage = async (imageUrl: string) => {
  if (!userId.value || sending.value) return

  sending.value = true
  try {
    // 调用API发送图片消息
    await sendDirectMessage({
      recipient_id: userId.value,
      content: imageUrl,
      message_type: 'image',
      latitude: 0,
      longitude: 0
    })

    // 重新加载消息列表
    await loadMessages()

    // 滚动到底部
    nextTick(() => {
      scrollToBottom()
    })

    message.success('图片发送成功')
  } catch (error) {
    console.error('发送图片失败:', error)
    message.error('发送图片失败')
  } finally {
    sending.value = false
  }
}

// 发送位置消息
const handleSendLocationMessage = async (location: any) => {
  if (!userId.value || sending.value) return

  sending.value = true
  try {
    // 调用API发送位置消息
    await sendDirectMessage({
      recipient_id: userId.value,
      content: JSON.stringify(location),
      message_type: 'location',
      latitude: location.latitude,
      longitude: location.longitude
    })

    // 重新加载消息列表
    await loadMessages()

    // 滚动到底部
    nextTick(() => {
      scrollToBottom()
    })

    message.success('位置发送成功')
  } catch (error) {
    console.error('发送位置失败:', error)
    message.error('发送位置失败')
  } finally {
    sending.value = false
  }
}

// 查看用户资料
const handleUserProfile = () => {
  if (userId.value) {
    navigateTo(`/user/${userId.value}`)
  }
}

// 返回处理
const handleBack = () => {
  router.back()
}

// 视频通话处理
const handleVideoCall = () => {
  message.info('视频通话功能开发中...')
}

// 更多操作
const handleMore = () => {
  message.info('更多功能开发中...')
}

// 清空聊天记录
const clearChat = async () => {
  try {
    await dialog.warning({
      title: '提示',
      content: '确定要清空所有聊天记录吗？',
      positiveText: '确定',
      negativeText: '取消'
    })

    // 这里应该调用清空聊天记录的API
    message.info('清空聊天记录功能开发中...')
  } catch {
    // 用户取消
  }
}

// 屏蔽用户
const blockUser = async () => {
  try {
    await dialog.warning({
      title: '提示',
      content: '确定要屏蔽该用户吗？',
      positiveText: '确定',
      negativeText: '取消'
    })

    // 这里应该调用屏蔽用户的API
    message.info('屏蔽用户功能开发中...')
  } catch {
    // 用户取消
  }
}

// Dialog 成功处理器
const handleCreateGroupSuccess = async (group: any) => {
  message.success('群组创建成功！')
  await navigateTo(`/group/${group.group_id}/chat`)
}

const handleMomentSuccess = async (moment: any) => {
  message.success('动态创建成功！')
  await navigateTo(`/moment/${moment.id}`)
}

const handleNewMessageSuccess = async (conversation: any) => {
  message.success('消息发送成功！')
  await navigateTo(`/chat/${conversation.conversation_id}`)
}

// 格式化时间
const formatTime = (timeStr: string) => {
  const time = new Date(timeStr)
  const now = new Date()
  const diff = now.getTime() - time.getTime()

  if (diff < 1000 * 60) {
    return '刚刚'
  } else if (diff < 1000 * 60 * 60) {
    return `${Math.floor(diff / (1000 * 60))}分钟前`
  } else if (diff < 1000 * 60 * 60 * 24) {
    return time.toLocaleTimeString('zh-CN', { hour: '2-digit', minute: '2-digit' })
  } else {
    return time.toLocaleDateString('zh-CN', { month: 'short', day: 'numeric' })
  }
}

// 滚动到底部
const scrollToBottom = () => {
  if (messagesContainer.value) {
    messagesContainer.value.scrollTop = messagesContainer.value.scrollHeight
  }
}

// 组件生命周期
onMounted(() => {
  initChat()
})
</script>

<style scoped>
.chat-page {
  height: 100vh;
  background: #f5f5f5;
}

/* 响应式头部 */
.chat-header {
  position: fixed;
  top: 0;
  left: 0;
  right: 0;
  z-index: var(--z-sticky);
  background: var(--pixel-bg-primary);
  border-bottom: 2px solid var(--pixel-border-primary);
  backdrop-filter: blur(10px);
}

.chat-header-content {
  display: flex;
  align-items: center;
  padding: 12px 16px;
  gap: 12px;
}

.header-info {
  display: flex;
  align-items: center;
  gap: 12px;
  flex: 1;
}

.header-details {
  flex: 1;
}

.chat-title {
  font-size: 16px;
  font-weight: 600;
  color: var(--pixel-text-primary);
  margin: 0;
  line-height: 1.2;
}

.status-text {
  font-size: 12px;
  color: var(--pixel-text-secondary);
}

.more-btn {
  padding: 8px;
  color: var(--pixel-text-secondary);
}

/* 响应式聊天区域 */
.compact-chat-container {
  height: 100vh;
  display: flex;
  flex-direction: column;
  padding-top: 60px;
  /* 为头部留出空间 */
}

.messages-area {
  flex: 1;
  overflow-y: auto;
  padding: 16px;
  background: var(--pixel-bg-secondary);
}

.messages-list {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.message-item {
  display: flex;
  flex-direction: column;
}

.message-sent {
  align-items: flex-end;
}

.message-received {
  align-items: flex-start;
}

.message-bubble {
  max-width: 80%;
  padding: 12px 16px;
  border-radius: 18px;
  background: var(--pixel-accent-primary);
  box-shadow: var(--pixel-shadow-sm);
}

.message-sent .message-bubble {
  background: var(--pixel-accent-primary);
  color: var(--pixel-text-on-accent);
}

.message-received .message-bubble {
  background: var(--pixel-bg-primary);
  color: var(--pixel-text-primary);
  border: 1px solid var(--pixel-border-primary);
}

.message-text {
  margin: 0;
  line-height: 1.4;
  word-wrap: break-word;
}

.message-meta {
  display: flex;
  align-items: center;
  gap: 4px;
  margin-top: 4px;
  font-size: 11px;
  color: var(--pixel-text-muted);
}

.message-time {
  font-size: 11px;
  color: var(--pixel-text-muted);
}

.message-status {
  display: flex;
  align-items: center;
}

.read-icon {
  color: var(--pixel-accent-secondary);
}

.sent-icon {
  color: var(--pixel-text-muted);
}

/* 消息输入区域 */
.message-input-area {
  background: var(--pixel-bg-primary);
  border-top: 1px solid var(--pixel-border-primary);
  padding: 16px;
}

.input-toolbar {
  display: flex;
  gap: 8px;
  margin-bottom: 12px;
}

.toolbar-btn {
  padding: 8px;
  color: var(--pixel-text-secondary);
  background: var(--pixel-bg-secondary);
  border-radius: 8px;
}

.input-area {
  display: flex;
  gap: 8px;
  align-items: flex-end;
}

.message-input {
  flex: 1;
}

.send-btn {
  padding: 10px 16px;
  border-radius: 18px;
  background: var(--pixel-accent-primary);
  color: var(--pixel-text-on-accent);
  border: none;
}

.send-btn:disabled {
  background: var(--pixel-bg-muted);
  color: var(--pixel-text-muted);
}

/* 加载和空状态 */
.loading-state,
.empty-state {
  text-align: center;
  padding: 40px 20px;
  color: var(--pixel-text-secondary);
}

.empty-icon {
  font-size: 48px;
  margin-bottom: 16px;
}

.load-more-section {
  text-align: center;
  padding: 16px;
}

/* 图片和位置消息 */
.message-image .chat-image {
  max-width: 200px;
  max-height: 200px;
  border-radius: 8px;
}

.message-location {
  padding: 12px;
  background: var(--pixel-bg-secondary);
  border-radius: 8px;
  border: 1px solid var(--pixel-border-primary);
}

.location-content {
  display: flex;
  align-items: center;
  gap: 8px;
}

.location-info {
  flex: 1;
}

.location-name {
  font-weight: 500;
  color: var(--pixel-text-primary);
}

.location-address {
  font-size: 12px;
  color: var(--pixel-text-secondary);
  margin-top: 2px;
}

/* 桌面端布局 */
.desktop-layout {
  display: none;
}

@media (min-width: 768px) {
  .chat-header,
  .compact-chat-container {
    display: none;
  }

  .full-layout {
    display: block;
    height: 100vh;
  }
}
</style>
