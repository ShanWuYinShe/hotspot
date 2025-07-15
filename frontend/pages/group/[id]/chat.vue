<template>
  <div class="group-chat-page">
    <!-- 大屏幕布局 -->
    <div class="desktop-layout">

      <div class="group-chat-layout">
        <!-- 左侧：群组信息面板 -->
        <div class="group-sidebar">
          <!-- 页面头部 -->
          <div class="sidebar-header">
            <div class="header-navigation">
              <BackButton />
            </div>
            <h2 class="sidebar-title text-responsive-lg text-electric">群组信息</h2>
          </div>

          <div v-if="groupInfo" class="group-profile">
            <div class="group-avatar-section">
              <NAvatar :size="80" :src="groupInfo.avatar_url" class="group-avatar">
                {{ groupInfo.name?.charAt(0) || 'G' }}
              </NAvatar>
              <div class="group-status-indicator"></div>
            </div>

            <div class="group-info">
              <h3 class="group-name text-responsive-lg text-gold">{{ groupInfo.name }}</h3>
              <div class="group-stats">
                <div class="stat-item">
                  <Icon name="mdi:account-group" size="16" />
                  <span class="text-responsive-sm text-bright">{{ groupInfo.member_count || 0 }} 名成员</span>
                </div>
                <div class="stat-item" v-if="onlineMembers.length > 0">
                  <div class="online-dot"></div>
                  <span class="text-responsive-sm text-electric">{{ onlineMembers.length }} 人在线</span>
                </div>
              </div>
              <p class="group-description text-responsive-sm" v-if="groupInfo.description">
                {{ groupInfo.description }}
              </p>
            </div>

            <div class="group-actions">
              <NButton @click="showGroupInfo" class="action-button primary">
                <Icon name="mdi:information" size="16" />
                <span>群组详情</span>
              </NButton>
              <NButton @click="showMemberList" class="action-button">
                <Icon name="mdi:account-group" size="16" />
                <span>成员列表</span>
              </NButton>
              <NButton @click="showGroupMenu" class="action-button">
                <Icon name="mdi:dots-horizontal" size="16" />
                <span>更多操作</span>
              </NButton>
            </div>
          </div>

          <!-- 在线成员列表 -->
          <div class="online-members" v-if="onlineMembers.length > 0">
            <h4>在线成员</h4>
            <div class="member-list">
              <div v-for="member in onlineMembers.slice(0, 8)" :key="member.user_id" class="member-item">
                <NAvatar :size="32" :src="member.avatar_url">
                  {{ member.nickname?.charAt(0) || 'U' }}
                </NAvatar>
                <span class="member-name">{{ member.nickname || '未知用户' }}</span>
              </div>
              <div v-if="onlineMembers.length > 8" class="more-members">
                <NButton text @click="showMemberList">
                  还有 {{ onlineMembers.length - 8 }} 人在线...
                </NButton>
              </div>
            </div>
          </div>

          <!-- 群组功能 -->
          <div class="group-features">
            <h4>群组功能</h4>
            <div class="feature-buttons">
              <NButton class="feature-button" @click="showImageUploadDialog = true">
                <Icon name="mdi:image" size="16" />
                <span>发送图片</span>
              </NButton>
              <NButton class="feature-button" @click="showLocationShareDialog = true">
                <Icon name="mdi:map-marker" size="16" />
                <span>分享位置</span>
              </NButton>
              <NButton class="feature-button" @click="showEmojiPickerDialog = true">
                <Icon name="mdi:emoticon-happy" size="16" />
                <span>表情</span>
              </NButton>
            </div>
          </div>

          <!-- 快速操作 -->
          <div class="quick-actions">
            <h4>快速操作</h4>
            <div class="action-list">
              <NButton text @click="showGroupSettings">
                <Icon name="mdi:cog" size="16" />
                <span>群组设置</span>
              </NButton>
              <NButton text @click="showNotificationSettings">
                <Icon name="mdi:bell" size="16" />
                <span>消息通知</span>
              </NButton>
            </div>
          </div>
        </div>

        <!-- 右侧：聊天主区域 -->
        <div class="group-chat-main">
          <!-- 群组聊天头部信息 -->
          <div class="chat-header">
            <div class="header-info">
              <NAvatar :size="40" :src="groupInfo?.avatar_url">
                {{ groupInfo?.name?.charAt(0) || 'G' }}
              </NAvatar>
              <div class="header-details">
                <h2 class="chat-title">{{ groupInfo?.name || '群聊' }}</h2>
                <div class="chat-status">
                  <span class="member-count">{{ groupInfo?.member_count || 0 }} 名成员</span>
                  <span v-if="onlineMembers.length > 0" class="online-indicator">
                    {{ onlineMembers.length }} 人在线
                  </span>
                </div>
              </div>
            </div>

            <div class="header-actions">
              <NButton link @click="showGroupInfo" :icon="InfoFilled">
                群组信息
              </NButton>
              <NButton link @click="showGroupMenu" :icon="MoreFilled">
                更多
              </NButton>
            </div>
          </div>

          <!-- 聊天消息区域 -->
          <div class="messages-area" ref="messagesContainer">
            <!-- 连接状态提示 -->
            <ConnectionStatus :status="connectionStatus" />

            <!-- 加载更多历史消息 -->
            <div v-if="hasMoreHistory" class="load-more-history">
              <NButton @click="loadMoreHistory" :loading="loadingHistory" size="small" text>
                加载更多历史消息
              </NButton>
            </div>

            <!-- 消息列表 -->
            <div class="messages-list" ref="messagesList">
              <!-- 日期分割线 -->
              <div v-for="(group, index) in messageGroups" :key="index" class="message-group">
                <div class="date-divider">
                  <span>{{ formatDate(group.date || '') }}</span>
                </div>

                <!-- 该日期的消息 -->
                <div v-for="message in group.messages" :key="message.id" class="message-item"
                  :class="{ 'own-message': message.sender_id === currentUserId }" :data-message-id="message.id">
                  <!-- 他人消息 -->
                  <div v-if="message.sender_id !== currentUserId" class="message-wrapper received">
                    <NAvatar :src="message.sender?.avatar_url" :size="36" class="message-avatar">
                      {{ message.sender?.nickname?.charAt(0) }}
                    </NAvatar>
                    <div class="message-content">
                      <div class="sender-name">{{ message.sender?.nickname }}</div>
                      <div class="message-bubble">
                        <!-- 文本消息 -->
                        <div v-if="message.content_type === 'text'" class="message-text">
                          {{ message.content }}
                        </div>
                        <!-- 图片消息 -->
                        <div v-else-if="message.content_type === 'image'" class="message-image">
                          <UImage :src="message.content as string" :preview-src-list="[message.content as string]"
                            fit="cover" class="chat-image" />
                        </div>
                        <!-- 位置消息 -->
                        <div v-else-if="message.content_type === 'location'" class="message-location">
                          <div class="location-content">
                            <NIcon>
                              <Location />
                            </NIcon>
                            <div class="location-info">
                              <div class="location-name">{{ parseLocationContent(message.content)?.name || '位置信息' }}
                              </div>
                              <div class="location-address">{{ parseLocationContent(message.content)?.address || '详细地址'
                                }}</div>
                            </div>
                          </div>
                        </div>
                        <!-- 系统消息 -->
                        <div v-else-if="message.content_type === 'system'" class="message-system">
                          {{ message.content }}
                        </div>
                      </div>
                      <div class="message-info">
                        <span class="message-time">{{ formatTime(message.created_at) }}</span>
                      </div>
                    </div>
                  </div>

                  <!-- 自己的消息 -->
                  <div v-else class="message-wrapper sent">
                    <div class="message-content">
                      <div class="message-bubble">
                        <!-- 文本消息 -->
                        <div v-if="message.content_type === 'text'" class="message-text">
                          {{ message.content }}
                        </div>
                        <!-- 图片消息 -->
                        <div v-else-if="message.content_type === 'image'" class="message-image">
                          <UImage :src="message.content as string" :preview-src-list="[message.content as string]"
                            fit="cover" class="chat-image" />
                        </div>
                        <!-- 位置消息 -->
                        <div v-else-if="message.content_type === 'location'" class="message-location">
                          <div class="location-content">
                            <NIcon>
                              <Location />
                            </NIcon>
                            <div class="location-info">
                              <div class="location-name">{{ parseLocationContent(message.content)?.name || '位置信息' }}
                              </div>
                              <div class="location-address">{{ parseLocationContent(message.content)?.address || '详细地址'
                                }}</div>
                            </div>
                          </div>
                        </div>
                      </div>
                      <div class="message-info">
                        <span class="message-time">{{ formatTime(message.created_at) }}</span>
                        <span class="message-status">
                          <!-- 移除消息状态显示，因为Message类型中没有status字段 -->
                        </span>
                      </div>
                    </div>
                    <NAvatar :src="currentUser?.avatar_url" :size="36" class="message-avatar">
                      {{ currentUser?.nickname?.charAt(0) }}
                    </NAvatar>
                  </div>
                </div>
              </div>
            </div>

            <!-- 正在输入指示器 -->
            <div v-if="typingUsers.length > 0" class="typing-indicator">
              <div class="typing-content">
                <div class="typing-avatars">
                  <NAvatar v-for="user in typingUsers.slice(0, 3)" :key="user.id" :src="user.avatar" :size="24"
                    class="typing-avatar">
                    {{ user.name?.charAt(0) }}
                  </NAvatar>
                </div>
                <div class="typing-text">
                  {{ getTypingText(typingUsers) }}
                </div>
                <div class="typing-dots">
                  <span></span>
                  <span></span>
                  <span></span>
                </div>
              </div>
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
              <div class="message-input">
                <NInput v-model="messageText" type="textarea" :rows="1" :autosize="{ minRows: 1, maxRows: 4 }"
                  placeholder="输入消息..." @keydown.enter.exact.prevent="sendMessage"
                  @keydown.enter.shift.exact="addNewLine" @input="handleTyping" ref="messageInput" />
              </div>

              <NButton type="primary" @click="sendMessage" :disabled="!messageText.trim()" :loading="sending"
                class="send-button">
                发送
              </NButton>
            </div>
          </div>
        </div>
      </div>
    </div>

    <!-- 响应式聊天布局 -->
    <div class="chat-layout">
      <!-- 聊天消息区域 -->
      <div class="chat-container" ref="chatContainer">
        <!-- 加载更多历史消息 -->
        <div v-if="hasMoreHistory" class="load-more-history">
          <NButton @click="loadMoreHistory" :loading="loadingHistory" size="small" text>
            加载更多历史消息
          </NButton>
        </div>

        <!-- 消息列表 -->
        <div class="messages-list" ref="messagesList">
          <!-- 日期分割线 -->
          <div v-for="(group, index) in messageGroups" :key="index" class="message-group">
            <div class="date-divider">
              <span>{{ formatDate(group.date || '') }}</span>
            </div>

            <!-- 该日期的消息 -->
            <div v-for="message in group.messages" :key="message.id" class="message-item"
              :class="{ 'own-message': message.sender_id === currentUserId }" :data-message-id="message.id">
              <!-- 他人消息 -->
              <div v-if="message.sender_id !== currentUserId" class="message-wrapper received">
                <NAvatar :src="message.sender?.avatar_url" :size="36" class="message-avatar">
                  {{ message.sender?.nickname?.charAt(0) }}
                </NAvatar>
                <div class="message-content">
                  <div class="sender-name">{{ message.sender?.nickname }}</div>
                  <div class="message-bubble">
                    <!-- 文本消息 -->
                    <div v-if="message.content_type === 'text'" class="message-text">
                      {{ message.content }}
                    </div>
                    <!-- 图片消息 -->
                    <div v-else-if="message.content_type === 'image'" class="message-image">
                      <UImage :src="message.content as string" :preview-src-list="[message.content as string]"
                        fit="cover" class="chat-image" />
                    </div>
                    <!-- 位置消息 -->
                    <div v-else-if="message.content_type === 'location'" class="message-location">
                      <div class="location-content">
                        <NIcon>
                          <Location />
                        </NIcon>
                        <div class="location-info">
                          <div class="location-name">{{ parseLocationContent(message.content)?.name || '位置信息' }}</div>
                          <div class="location-address">{{ parseLocationContent(message.content)?.address || '详细地址' }}
                          </div>
                        </div>
                      </div>
                    </div>
                    <!-- 系统消息 -->
                    <div v-else-if="message.content_type === 'system'" class="message-system">
                      {{ message.content }}
                    </div>
                  </div>
                  <div class="message-info">
                    <span class="message-time">{{ formatTime(message.created_at) }}</span>
                  </div>
                </div>
              </div>

              <!-- 自己的消息 -->
              <div v-else class="message-wrapper sent">
                <div class="message-content">
                  <div class="message-bubble">
                    <!-- 文本消息 -->
                    <div v-if="message.content_type === 'text'" class="message-text">
                      {{ message.content }}
                    </div>
                    <!-- 图片消息 -->
                    <div v-else-if="message.content_type === 'image'" class="message-image">
                      <UImage :src="message.content as string" :preview-src-list="[message.content as string]"
                        fit="cover" class="chat-image" />
                    </div>
                    <!-- 位置消息 -->
                    <div v-else-if="message.content_type === 'location'" class="message-location">
                      <div class="location-content">
                        <NIcon>
                          <Location />
                        </NIcon>
                        <div class="location-info">
                          <div class="location-name">{{ parseLocationContent(message.content)?.name || '位置信息' }}</div>
                          <div class="location-address">{{ parseLocationContent(message.content)?.address || '详细地址' }}
                          </div>
                        </div>
                      </div>
                    </div>
                  </div>
                  <div class="message-info">
                    <span class="message-time">{{ formatTime(message.created_at) }}</span>
                    <span class="message-status">
                      <!-- 移除消息状态显示，因为Message类型中没有status字段 -->
                    </span>
                  </div>
                </div>
                <NAvatar :src="currentUser?.avatar_url" :size="36" class="message-avatar">
                  {{ currentUser?.nickname?.charAt(0) }}
                </NAvatar>
              </div>
            </div>
          </div>
        </div>

        <!-- 正在输入指示器 -->
        <div v-if="typingUsers.length > 0" class="typing-indicator">
          <div class="typing-content">
            <div class="typing-avatars">
              <NAvatar v-for="user in typingUsers.slice(0, 3)" :key="user.id" :src="user.avatar" :size="24"
                class="typing-avatar">
                {{ user.name?.charAt(0) }}
              </NAvatar>
            </div>
            <div class="typing-text">
              {{ getTypingText(typingUsers) }}
            </div>
            <div class="typing-dots">
              <span></span>
              <span></span>
              <span></span>
            </div>
          </div>
        </div>
      </div>

      <!-- 消息输入区域 -->
      <div class="chat-input-area">
        <div class="input-container">
          <!-- 消息输入框 -->
          <div class="message-input">
            <NInput v-model="messageText" type="textarea" :rows="1" :autosize="{ minRows: 1, maxRows: 4 }"
              placeholder="输入消息..." @keydown.enter.exact.prevent="sendMessage" @keydown.enter.shift.exact="addNewLine"
              @input="handleTyping" ref="messageInput" />
          </div>

          <!-- 工具按钮 -->
          <div class="input-tools">
            <NButton @click="showEmojiPickerDialog = true" circle size="small">
              <NIcon>
                <ChatDotRound />
              </NIcon>
            </NButton>
            <NButton @click="showImageUploadDialog = true" circle size="small">
              <NIcon>
                <Picture />
              </NIcon>
            </NButton>
            <NButton @click="showLocationShareDialog = true" circle size="small">
              <NIcon>
                <Location />
              </NIcon>
            </NButton>
          </div>

          <!-- 发送按钮 -->
          <NButton type="primary" @click="sendMessage" :disabled="!messageText.trim()" :loading="sending" circle
            class="send-btn">
            <NIcon>
              <Promotion />
            </NIcon>
          </NButton>
        </div>
      </div>
    </div>

    <!-- 群组菜单弹窗 -->
    <el-drawer v-model="showGroupMenuDrawer" :title="groupInfo?.name" direction="rtl" size="80%">
      <div class="group-menu-content">
        <!-- 群组信息 -->
        <div class="group-header">
          <NAvatar :src="groupInfo?.avatar_url" :size="60">
            {{ groupInfo?.name?.charAt(0) }}
          </NAvatar>
          <div class="group-info">
            <h3>{{ groupInfo?.name }}</h3>
            <p>{{ groupInfo?.description }}</p>
            <div class="group-stats">
              <span>{{ groupInfo?.member_count || 0 }} 名成员</span>
            </div>
          </div>
        </div>

        <!-- 菜单选项 -->
        <div class="menu-options">
          <NButton @click="viewGroupInfo" text class="menu-item">
            <NIcon>
              <InfoFilled />
            </NIcon>
            群组信息
          </NButton>
          <NButton @click="viewMembers" text class="menu-item">
            <NIcon>
              <User />
            </NIcon>
            查看成员
          </NButton>
          <NButton @click="manageGroup" text class="menu-item" v-if="isGroupAdmin">
            <NIcon>
              <Setting />
            </NIcon>
            群组管理
          </NButton>
          <NButton @click="searchMessages" text class="menu-item">
            <NIcon>
              <Search />
            </NIcon>
            搜索聊天记录
          </NButton>
          <NButton @click="showQRCodeDialog = true" text class="menu-item">
            <NIcon>
              <Promotion />
            </NIcon>
            分享群组二维码
          </NButton>
          <NButton @click="muteNotifications" text class="menu-item">
            <NIcon>
              <BellFilled />
            </NIcon>
            {{ '消息免打扰' }}
          </NButton>
          <NButton @click="clearChatHistory" text class="menu-item text-error">
            <NIcon>
              <Delete />
            </NIcon>
            清空聊天记录
          </NButton>
          <NButton @click="leaveGroup" text class="menu-item text-error">
            <NIcon>
              <SwitchButton />
            </NIcon>
            退出群聊
          </NButton>
        </div>
      </div>
    </el-drawer>

    <!-- 表情选择器 -->
    <CommonEmojiPicker v-model:visible="showEmojiPickerDialog" @select="handleEmojiSelectForInput" />

    <!-- 图片上传组件 -->
    <CommonImageUploader v-model:visible="showImageUploadDialog" :multiple="true" :max-files="9"
      @upload-success="handleImageUploadComplete" />

    <!-- 位置分享组件 -->
    <CommonLocationShare v-model:visible="showLocationShareDialog" @location-selected="handleLocationSelected" />

    <!-- 二维码生成器（用于群组邀请） -->
    <CommonQRCodeGenerator v-model:visible="showQRCodeDialog" :content="groupInviteData"
      :title="`邀请加入 ${groupInfo?.name}`" :description="groupInfo?.description" @share="handleQRCodeShare" />

    <!-- 微信分享组件 -->
    <CommonWechatShare v-model:visible="showWechatShareDialog" title="微信分享群组" :share-options="shareOptions" />

    <!-- 创建群组对话框 -->
    <div v-if="showCreateGroupDialog" class="modal-overlay">
      <div class="modal-content">
        <h3 class="modal-title">创建群组</h3>
        <p class="modal-description">创建群组功能开发中...</p>
        <div class="modal-actions">
          <button @click="showCreateGroupDialog = false" class="modal-btn modal-btn--secondary">
            取消
          </button>
        </div>
      </div>
    </div>

    <!-- 新消息对话框 -->
    <div v-if="showNewMessageDialog" class="modal-overlay">
      <div class="modal-content">
        <h3 class="modal-title">新建聊天</h3>
        <p class="modal-description">新建聊天功能开发中...</p>
        <div class="modal-actions">
          <button @click="showNewMessageDialog = false" class="modal-btn modal-btn--secondary">
            取消
          </button>
        </div>
      </div>
    </div>

    <!-- 隐藏的文件输入 -->
    <input ref="fileInput" type="file" accept="image/*" @change="handleImageSelect" style="display: none" />
  </div>
</template>

<script setup lang="ts">
// 禁用 SSR 以避免 Naive UI 组件的 document.head 错误
definePageMeta({
  ssr: false
})

// Vue 3 APIs
import { ref, computed, onMounted, onUnmounted, nextTick } from 'vue'

// Naive UI components
import { NButton, NIcon, NAvatar, NImage, NInput } from 'naive-ui'
import { NDrawer, useMessage } from 'naive-ui'

// Naive UI icons
import { InformationCircle as InfoFilled, EllipsisHorizontal as MoreFilled, Chatbubble as ChatDotRound, Image as Picture, Location, Megaphone as Promotion, Person as User, Settings as Setting, Search, Notifications as BellFilled, TrashOutline as Delete, ToggleOutline as SwitchButton } from '@vicons/ionicons5'

// 项目组件导入

// Pinia
import { storeToRefs } from 'pinia'

// Stores
import { useAuthStore } from '~/stores/auth'

// 导入组件
import { isUserOnline as checkUserOnlineStatus } from '~/utils/userStatus'

// Services
import { useGroupService, useMessageService } from '~/services'

// 服务实例
const { searchGroupsById, leaveGroup: leaveGroupAPI } = useGroupService()
const { getGroupMessages, sendGroupMessage } = useMessageService()

// Types
import type {
  Message,
  LocationContent,
  MessageGroup,
  TypingUser,
  ShareOptions
} from '~/types/group/chat.types'
import type { GroupInfo } from '~/types/models/group'
// import type { GroupMessageNotification, UserStatusNotification } from '~/types/websocket'

// Utility functions
const formatDate = (date: string | Date) => {
  const d = new Date(date)
  return d.toLocaleDateString('zh-CN')
}

const formatTime = (time: string | Date) => {
  const d = new Date(time)
  return d.toLocaleTimeString('zh-CN', { hour: '2-digit', minute: '2-digit' })
}

const formatFileSize = (bytes: number) => {
  if (bytes === 0) return '0 Bytes'
  const k = 1024
  const sizes = ['Bytes', 'KB', 'MB', 'GB']
  const i = Math.floor(Math.log(bytes) / Math.log(k))
  return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i]
}

const parseLocationContent = (content: string) => {
  try {
    return JSON.parse(content)
  } catch {
    return { latitude: 0, longitude: 0, name: '位置信息' }
  }
}

// 获取路由参数
const route = useRoute()
const router = useRouter()
const groupId = route.params.id as string

// 初始化 Naive UI message API
const message = useMessage()

// 获取用户信息
const authStore = useAuthStore()
const { user } = storeToRefs(authStore)
const currentUserId = computed(() => user.value?.user_id?.toString() || '')
const currentUser = computed(() => user.value)

// WebSocket连接（简化版本）
const isConnected = ref(false)
const connect = () => {
  // WebSocket连接逻辑待实现
  isConnected.value = true
}
const disconnect = () => {
  // WebSocket断开逻辑待实现
  isConnected.value = false
}

// 连接状态计算
const connectionStatus = computed(() => {
  if (!isConnected.value) return 'disconnected'
  return 'connected'
})

// 聊天处理器相关状态 - 不从composable导入，直接定义
const messageText = ref('')
const showEmojiPickerDialog = ref(false)
const messageInput = ref<HTMLTextAreaElement | null>(null)

// 聊天相关状态
const messages = ref<Message[]>([])
const messageGroups = ref<MessageGroup[]>([])
const typingUsers = ref<TypingUser[]>([])
const isLoading = ref(false)
const groupInfo = ref<GroupInfo | null>(null)
const loadingMore = ref(false)
const hasMoreMessages = ref(true)

// 新增：在线成员和相关功能
const onlineMembers = ref<any[]>([])

// 技能栏功能函数
// 快捷发送表情
const sendQuickEmoji = async (emoji: string) => {
  if (!emoji.trim()) return

  messageText.value = emoji
  await sendMessage()
}

// 分享位置
const shareLocation = () => {
  // 位置分享功能待完善
  message.info('位置分享功能开发中...')
}

// 打开图片选择器
const openImagePicker = () => {
  // 图片分享功能待完善
  message.info('图片分享功能开发中...')
}

// 跳转到群组管理
const goToGroupManage = () => {
  router.push(`/group/${groupId}/manage`)
}

// 显示更多选项
const showMoreOptions = () => {
  // 更多选项功能待完善
  message.info('更多选项功能开发中...')
}

// 新增：其他必要的函数和状态
// 使用聊天分享功能

// 数据适配器 - 将API消息格式转换为UI消息格式
const adaptApiMessageToUI = (apiMessage: any): Message => {
  // 处理GroupMessage类型的API响应
  if (apiMessage.message_id && apiMessage.group_id) {
    return {
      id: apiMessage.message_id,
      content: apiMessage.content,
      content_type: apiMessage.message_type === 'text' ? 'text' :
        apiMessage.message_type === 'image' ? 'image' :
          apiMessage.message_type === 'location' ? 'location' : 'text',
      sender_id: apiMessage.sender_id?.toString() || '',
      group_id: apiMessage.group_id,
      created_at: apiMessage.sent_at,
      updated_at: apiMessage.sent_at,
      sender: apiMessage.sender_name ? {
        id: apiMessage.sender_id?.toString() || '',
        nickname: apiMessage.sender_name,
        avatar_url: apiMessage.sender_avatar_url
      } : null
    }
  }

  // 如果已经是UI格式的Message，直接返回
  return apiMessage as Message
}

// WebSocket实时消息处理
const handleRealtimeGroupMessage = (data: any) => {
  // 检查消息是否属于当前群组
  if (data.group_id.toString() !== groupId) return

  // 将新消息添加到消息列表
  const newMessage: Message = {
    id: data.message_id,
    content: data.content,
    content_type: data.message_type === 'text' ? 'text' :
      data.message_type === 'image' ? 'image' :
        data.message_type === 'location' ? 'location' : 'text',
    sender_id: data.sender_id.toString(),
    group_id: data.group_id.toString(),
    created_at: data.sent_at,
    updated_at: data.sent_at,
    sender: {
      id: data.sender_id.toString(),
      nickname: data.sender_name,
      avatar_url: data.sender_avatar
    }
  }

  // 如果不是自己发送的消息，才添加到列表
  if (data.sender_id.toString() !== currentUserId.value) {
    messages.value.push(newMessage)
    updateMessageGroups()

    // 如果用户已经滚动到底部，自动滚动查看新消息
    if (hasScrolledToBottom.value) {
      nextTick(() => {
        scrollToBottom()
      })
    } else {
      // 如果用户没有在底部，显示新消息提示
      message.info(`${data.sender_name}: ${data.content.substring(0, 30)}${data.content.length > 30 ? '...' : ''}`)
    }
  }
}

// 更新群组成员在线状态
const updateMemberOnlineStatus = (userId: number, isOnline: boolean) => {
  // 更新在线成员列表
  const memberIndex = onlineMembers.value.findIndex(member => member.user_id === userId)

  if (isOnline && memberIndex === -1) {
    // 用户上线，添加到在线成员列表
    // 这里可能需要获取完整的用户信息
    onlineMembers.value.push({
      user_id: userId,
      name: `用户${userId}`, // 临时名称，实际应该从群组成员信息中获取
      avatar_url: ''
    })
  } else if (!isOnline && memberIndex > -1) {
    // 用户下线，从在线成员列表移除
    onlineMembers.value.splice(memberIndex, 1)
  }
}

// 页面状态
const showMessageMenu = ref(false)
const showEmojiMenu = ref(false)
const showLocationPicker = ref(false)
const selectedMessage = ref<Message | null>(null)
const chatContainer = ref<HTMLElement | null>(null)
const hasScrolledToBottom = ref(true)
const sending = ref(false)
const hasMoreHistory = computed(() => hasMoreMessages.value)
const loadingHistory = computed(() => loadingMore.value)

// 对话框状态
const showImageUploadDialog = ref(false)
const showLocationShareDialog = ref(false)
const showGroupMenuDrawer = ref(false)
const showQRCodeDialog = ref(false)
const showWechatShareDialog = ref(false)
const showCreateGroupDialog = ref(false)
const showNewMessageDialog = ref(false)

// 群组信息
const isGroupAdmin = computed(() => {
  if (!groupInfo.value || !user.value) return false
  return groupInfo.value.owner_id === user.value.user_id
})

// 邀请信息
const groupInviteData = computed(() => {
  if (!groupInfo.value) return ''
  return `${window.location.origin}/group/join/${groupId}`
})

// 分享选项
const shareOptions = computed(() => {
  if (!groupInfo.value) return {
    title: '群聊',
    desc: '加入我们的群聊',
    link: `${window.location.origin}/group/join/${groupId}`,
    imgUrl: `${window.location.origin}/images/group-default.png`
  }

  return {
    title: groupInfo.value.name || '群聊',
    desc: groupInfo.value.description || '加入我们的群聊',
    link: `${window.location.origin}/group/join/${groupId}`,
    imgUrl: groupInfo.value.avatar_url || `${window.location.origin}/images/group-default.png`
  }
})

// 导航方法
const goBack = () => {
  router.back()
}

// 显示群组信息
const showGroupInfo = () => {
  router.push(`/group/${groupId}/info`)
}

// 显示群组菜单
const showGroupMenu = () => {
  showGroupMenuDrawer.value = true
}

// 显示成员列表
const showMemberList = () => {
  router.push(`/group/${groupId}/members`)
}

// 显示群组设置
const showGroupSettings = () => {
  router.push(`/group/${groupId}/manage`)
}

// 显示通知设置
const showNotificationSettings = () => {
  // TODO: 实现通知设置功能
  console.log('显示通知设置')
}

// 加载群组信息
const loadGroupInfo = async () => {
  try {
    const data = await searchGroupsById({ group_id: groupId })
    groupInfo.value = data
    return data
  } catch (error) {
    console.error('加载群组信息失败', error)
    return null
  }
}

// 加载聊天消息
const loadMessages = async (before?: string) => {
  if (isLoading.value) return

  isLoading.value = true

  try {
    if (before) {
      loadingMore.value = true
    }

    // 从API获取真实聊天记录
    const response = await getGroupMessages({
      group_id: groupId,
      cursor: before ? parseInt(before) : undefined,
      limit: 50
    })
    const rawData = response?.items || []

    // 使用适配器转换数据格式
    const data = rawData.map(adaptApiMessageToUI)

    if (before) {
      // 加载更多旧消息，添加到列表开头
      if (data && data.length > 0) {
        messages.value = [...data, ...messages.value]
      } else {
        hasMoreMessages.value = false
      }
      loadingMore.value = false
    } else {
      // 首次加载，替换消息列表
      messages.value = data || []
    }

    // 更新消息分组
    updateMessageGroups()

    return data
  } catch (error) {
    console.error('加载消息失败', error)
    return []
  } finally {
    isLoading.value = false
  }
}

// 更新消息分组
const updateMessageGroups = () => {
  if (!messages.value || messages.value.length === 0) {
    messageGroups.value = []
    return
  }

  const groups: MessageGroup[] = []
  let currentGroup: MessageGroup | null = null
  let lastSenderId: string | null = null
  let lastDate: string | null = null

  // 对消息进行分组处理
  messages.value.forEach(message => {
    const messageDate = formatDate(message.created_at)

    // 如果日期变了，创建一个日期分隔
    if (messageDate !== lastDate) {
      groups.push({
        id: `date-${Date.now()}-${Math.random().toString(36).substr(2, 9)}`,
        type: 'date',
        date: messageDate
      })
      lastDate = messageDate
      lastSenderId = null
    }

    // 如果发送者变了或者是新的一天，创建一个新的消息组
    if (message.sender_id !== lastSenderId) {
      currentGroup = {
        id: `group-${Date.now()}-${Math.random().toString(36).substr(2, 9)}`,
        type: 'message-group',
        senderId: message.sender_id,
        sender: message.sender,
        messages: [message],
        time: formatTime(message.created_at)
      }
      groups.push(currentGroup)
      lastSenderId = message.sender_id
    } else if (currentGroup && currentGroup.messages) {
      // 否则添加到当前组
      currentGroup.messages.push(message)
      currentGroup.time = formatTime(message.created_at)
    }
  })

  messageGroups.value = groups
}

// 发送消息（重载方法）
const sendMessage = async (messageData?: any) => {
  try {
    sending.value = true

    // 如果没有传入消息数据，使用输入框的文本内容
    const actualMessageData = messageData || {
      group_id: groupId,
      sender_id: user.value?.user_id?.toString(),
      content_type: 'text' as const,
      content: messageText.value
    }

    // 检查消息内容
    if (!actualMessageData.content?.trim()) {
      return
    }

    // 构建临时消息用于立即显示
    const tempMessage: Message = {
      id: `temp-${Date.now()}-${Math.random().toString(36).substr(2, 9)}`,
      sender_id: user.value?.user_id?.toString() || '',
      group_id: groupId,
      content_type: actualMessageData.content_type,
      content: actualMessageData.content,
      created_at: new Date().toISOString(),
      updated_at: new Date().toISOString(),
      sender: user.value ? {
        id: user.value.user_id?.toString() || '',
        nickname: user.value.nickname || '',
        avatar_url: user.value.avatar_url
      } : null
    }

    // 添加到消息列表
    addMessage(tempMessage)

    // 如果是文本消息，清空输入框
    if (!messageData && actualMessageData.content_type === 'text') {
      messageText.value = ''
      // 聚焦回输入框
      nextTick(() => {
        messageInput.value?.focus()
      })
    }

    // 发送到服务器
    await sendGroupMessage({
      group_id: actualMessageData.group_id,
      content: actualMessageData.content,
      message_type: actualMessageData.content_type,
      latitude: 0,
      longitude: 0
    })

    return tempMessage
  } catch (error) {
    console.error('发送消息失败', error)
    throw error
  } finally {
    sending.value = false
  }
}

// 添加消息到列表
const addMessage = (message: Message) => {
  messages.value.push(message)
  updateMessageGroups()

  // 如果已经滚动到底部，继续保持滚动到底部
  if (hasScrolledToBottom.value) {
    nextTick(() => {
      scrollToBottom()
    })
  }
}

// 查看群组信息
const viewGroupInfo = () => {
  router.push(`/group/${groupId}/info`)
  showGroupMenuDrawer.value = false
}

// 查看群组成员
const viewMembers = () => {
  router.push(`/group/${groupId}/members`)
  showGroupMenuDrawer.value = false
}

// 管理群组
const manageGroup = () => {
  router.push(`/group/${groupId}/manage`)
  showGroupMenuDrawer.value = false
}

// 搜索消息
const searchMessages = () => {
  // 实现搜索消息逻辑
  showGroupMenuDrawer.value = false
}

// 静音通知
const muteNotifications = () => {
  // 实现静音通知逻辑
  showGroupMenuDrawer.value = false
}

// 清除聊天历史
const clearChatHistory = async () => {
  try {
    // 从API获取真实数据
    // await chatService.clearGroupChatHistory(groupId)
    messages.value = []
    updateMessageGroups()
    message.success('聊天记录已清除')
  } catch (error) {
    console.error('清除聊天记录失败', error)
    message.error('清除聊天记录失败')
  }
  showGroupMenuDrawer.value = false
}

// 退出群组
const leaveGroup = async () => {
  try {
    await leaveGroupAPI({ group_id: groupId })
    message.success('已退出群组')
    router.push('/my-groups')
  } catch (error) {
    console.error('退出群组失败', error)
    message.error('退出群组失败')
  }
}

// Dialog 成功处理器
const handleCreateGroupSuccess = async (group: any) => {
  message.success('群组创建成功！')
  await navigateTo(`/group/${group.group_id}`)
}

const handleMomentSuccess = async (moment: any) => {
  message.success('动态创建成功！')
  await navigateTo(`/moment/${moment.id}`)
}

const handleNewMessageSuccess = async (conversation: any) => {
  message.success('消息发送成功！')
  await navigateTo(`/chat/${conversation.conversation_id}`)
}

// 移除重复的函数声明，改为引用已存在的方法
const handleEmojiSelectForInput = (emoji: any) => {
  if (!emoji) return

  // 检查emoji的类型，并获取字符
  const char = typeof emoji === 'string' ? emoji : emoji.char
  messageText.value += char
  showEmojiPickerDialog.value = false

  // 聚焦到输入框
  nextTick(() => {
    messageInput.value?.focus()
  })
}

// 处理位置选择
const handleLocationSelected = (location: LocationContent) => {
  try {
    // 构建位置消息内容
    const content = JSON.stringify(location)

    // 发送位置消息
    sendMessage({
      group_id: groupId,
      sender_id: user.value?.user_id?.toString(),
      content_type: 'location',
      content
    })

    // 关闭位置选择对话框
    showLocationShareDialog.value = false
  } catch (error) {
    console.error('发送位置信息失败', error)
    message.error('发送位置信息失败')
  }
}

// 返回上一页
const handleBack = () => {
  router.back()
}

// 分享群组
const handleShare = () => {
  message.info('群组分享功能开发中...')
}

// 处理QR码分享
const handleQRCodeShare = (content: string, qrCodeUrl: string) => {
  // 设置微信分享
  showWechatShareDialog.value = true
}

// 处理图片选择
const handleImageSelect = (event: Event) => {
  const input = event.target as HTMLInputElement
  if (!input.files || input.files.length === 0) return

  const files = Array.from(input.files)

  // 上传图片
  uploadImages(files)
}

// 处理图片上传成功
const handleImageUploadComplete = (uploadedImages: any[]) => {
  if (!uploadedImages || uploadedImages.length === 0) return

  // 发送图片消息
  uploadedImages.forEach(async (image) => {
    try {
      await sendMessage({
        group_id: groupId,
        sender_id: user.value?.user_id?.toString(),
        content_type: 'image',
        content: image.url || image.src
      })
    } catch (error) {
      console.error('发送图片消息失败', error)
      message.error('发送图片消息失败')
    }
  })

  // 关闭图片上传对话框
  showImageUploadDialog.value = false
}

// 上传图片
const uploadImages = async (files: File[]) => {
  try {
    sending.value = true

    // 简化的图片上传逻辑，这里应该调用实际的上传服务
    for (const file of files) {
      // 创建临时URL用于显示
      const imageUrl = URL.createObjectURL(file)

      await sendMessage({
        group_id: groupId,
        sender_id: user.value?.user_id?.toString(),
        content_type: 'image',
        content: imageUrl
      })
    }

    // 关闭图片上传对话框
    showImageUploadDialog.value = false
  } catch (error) {
    console.error('上传图片失败', error)
    message.error('上传图片失败')
  } finally {
    sending.value = false
  }
}

// 移除重复的函数声明
// const handleSendTextMessage = () => {

// 添加换行
const addNewLine = (e: Event) => {
  const keyEvent = e as KeyboardEvent
  keyEvent.preventDefault()
  messageText.value += '\n'
}

// 处理正在输入事件
let typingTimer: NodeJS.Timeout | null = null
const handleTyping = () => {
  // 发送正在输入状态
  if (user.value && groupId) {
    // 这里可以通过WebSocket发送typing状态
    // webSocketService.sendTypingStatus(groupId, user.value.user_id, true)
  }

  // 清除之前的计时器
  if (typingTimer) {
    clearTimeout(typingTimer)
  }

  // 设置3秒后停止typing状态
  typingTimer = setTimeout(() => {
    if (user.value && groupId) {
      // webSocketService.sendTypingStatus(groupId, user.value.user_id, false)
    }
  }, 3000)
}

// 加载更多历史消息
const loadMoreHistory = async () => {
  if (messages.value.length === 0) return

  // 获取最早消息的ID
  const firstMessageId = messages.value[0]?.id

  // 加载更多消息
  await loadMessages(firstMessageId)
}

// 滚动到底部
const scrollToBottom = () => {
  if (!chatContainer.value) return

  chatContainer.value.scrollTop = chatContainer.value.scrollHeight
}

// 处理滚动事件
const handleScroll = () => {
  if (!chatContainer.value) return

  const { scrollTop, scrollHeight, clientHeight } = chatContainer.value

  // 检测是否滚动到底部或接近底部
  hasScrolledToBottom.value = scrollHeight - scrollTop - clientHeight < 100
}

// 获取正在输入的用户文本
const getTypingText = (users: TypingUser[]) => {
  if (users.length === 0) return ''
  if (users.length === 1) return `${users[0].name} 正在输入...`
  if (users.length === 2) return `${users[0].name} 和 ${users[1].name} 正在输入...`
  return `${users.length} 人正在输入...`
}

// 处理图片预览
const handleImagePreview = (imageUrl: string) => {
  console.log('预览图片:', imageUrl)
}

// WebSocket 房间订阅管理（暂时为空实现）
const subscribeToGroupRoom = () => {
  if (groupId) {
    // TODO: 实现房间订阅逻辑
    console.log(`Subscribing to group_${groupId}`)
  }
}

const unsubscribeFromGroupRoom = () => {
  if (groupId) {
    // TODO: 实现房间取消订阅逻辑
    console.log(`Unsubscribing from group_${groupId}`)
  }
}

// 监听路由变化以重新订阅房间
watch(() => route.params.id, (newGroupId, oldGroupId) => {
  if (oldGroupId) {
    // TODO: 取消订阅旧房间
    console.log(`Unsubscribing from group_${oldGroupId}`)
  }
  if (newGroupId) {
    // TODO: 订阅新房间
    console.log(`Subscribing to group_${newGroupId}`)
  }
})

// 生命周期管理
onMounted(async () => {
  // 加载群组信息和消息
  await Promise.all([loadGroupInfo(), loadMessages()])

  // 初始化滚动到底部
  nextTick(() => {
    scrollToBottom()
  })

  // 添加滚动事件监听
  if (chatContainer.value) {
    chatContainer.value.addEventListener('scroll', handleScroll)
  }

  // 订阅WebSocket群组房间
  subscribeToGroupRoom()
})

onUnmounted(() => {
  // 移除滚动事件监听
  if (chatContainer.value) {
    chatContainer.value.removeEventListener('scroll', handleScroll)
  }

  // 取消订阅WebSocket群组房间
  unsubscribeFromGroupRoom()

  // 断开WebSocket连接（如果需要）
  // disconnect() // 可选：如果这是最后一个使用WebSocket的页面
})
</script>

<style lang="scss" scoped>
.group-chat-page {
  height: 100vh;
  display: flex;
  flex-direction: column;
  background: linear-gradient(135deg, #f5f7fa 0%, #c3cfe2 100%);
}

.desktop-layout {
  display: flex;
  flex-direction: column;
  flex: 1;
  padding: 20px;
}

.group-chat-layout {
  display: flex;
  flex: 1;
  overflow: hidden;
  background: white;
  border-radius: 16px;
  box-shadow: 0 8px 32px rgba(0, 0, 0, 0.1);
  max-width: 1200px;
  margin: 0 auto;
  width: 100%;
}

.group-sidebar {
  background: linear-gradient(180deg, #667eea 0%, #764ba2 100%);
  color: white;
  width: 320px;
  display: flex;
  flex-direction: column;
  border-radius: 16px 0 0 16px;

  .sidebar-header {
    padding: 24px 20px 20px;
    background: rgba(255, 255, 255, 0.1);
    backdrop-filter: blur(10px);

    .header-navigation {
      margin-bottom: 16px;

      .back-btn {
        background: rgba(255, 255, 255, 0.2);
        border: none;
        color: white;
        border-radius: 12px;
        padding: 8px 12px;
        transition: all 0.3s ease;

        &:hover {
          background: rgba(255, 255, 255, 0.3);
          transform: translateX(-2px);
        }
      }
    }

    .sidebar-title {
      font-size: 20px;
      font-weight: 600;
      margin: 0;
      color: white;
    }
  }

  .group-profile {
    padding: 20px;
    text-align: center;

    .group-avatar-section {
      position: relative;
      display: inline-block;
      margin-bottom: 16px;

      .group-avatar {
        border: 4px solid rgba(255, 255, 255, 0.3);
        box-shadow: 0 8px 24px rgba(0, 0, 0, 0.2);
      }

      .group-status-indicator {
        position: absolute;
        bottom: 4px;
        right: 4px;
        width: 16px;
        height: 16px;
        background: #4ade80;
        border: 3px solid white;
        border-radius: 50%;
        box-shadow: 0 2px 8px rgba(0, 0, 0, 0.2);
      }
    }

    .group-info {
      margin-bottom: 24px;

      .group-name {
        font-size: 22px;
        font-weight: 700;
        margin-bottom: 12px;
        color: white;
      }

      .group-stats {
        display: flex;
        flex-direction: column;
        gap: 8px;
        margin-bottom: 16px;

        .stat-item {
          display: flex;
          align-items: center;
          justify-content: center;
          gap: 8px;
          color: rgba(255, 255, 255, 0.9);
          font-size: 14px;

          .online-dot {
            width: 8px;
            height: 8px;
            background: #4ade80;
            border-radius: 50%;
            animation: pulse 2s infinite;
          }
        }
      }

      .group-description {
        color: rgba(255, 255, 255, 0.8);
        font-size: 14px;
        line-height: 1.5;
        background: rgba(255, 255, 255, 0.1);
        padding: 12px;
        border-radius: 12px;
        backdrop-filter: blur(10px);
      }
    }

    .group-actions {
      display: flex;
      flex-direction: column;
      gap: 8px;

      .action-button {
        background: rgba(255, 255, 255, 0.1);
        border: 1px solid rgba(255, 255, 255, 0.2);
        color: white;
        border-radius: 12px;
        padding: 12px 16px;
        transition: all 0.3s ease;
        display: flex;
        align-items: center;
        gap: 8px;

        &:hover {
          background: rgba(255, 255, 255, 0.2);
          transform: translateY(-1px);
          box-shadow: 0 4px 12px rgba(0, 0, 0, 0.2);
        }

        &.primary {
          background: rgba(255, 255, 255, 0.2);
          border-color: rgba(255, 255, 255, 0.3);
        }
      }
    }
  }

  .online-members {
    padding: 0 20px;
    margin-bottom: 24px;

    h4 {
      margin: 0 0 16px 0;
      font-size: 16px;
      font-weight: 600;
      color: white;
    }

    .member-list {
      display: flex;
      flex-wrap: wrap;
      gap: 8px;

      .member-item {
        display: flex;
        align-items: center;
        font-size: 13px;
        color: rgba(255, 255, 255, 0.9);
        background: rgba(255, 255, 255, 0.1);
        padding: 6px 12px;
        border-radius: 20px;
        backdrop-filter: blur(10px);

        .member-name {
          margin-left: 6px;
        }
      }

      .more-members {
        margin-top: 8px;
        text-align: center;

        .el-button {
          color: rgba(255, 255, 255, 0.8);
          background: transparent;
          border: none;

          &:hover {
            color: white;
          }
        }
      }
    }
  }

  .group-features {
    padding: 0 20px;
    margin-bottom: 24px;

    h4 {
      margin: 0 0 16px 0;
      font-size: 16px;
      font-weight: 600;
      color: white;
    }

    .feature-buttons {
      display: flex;
      flex-direction: column;
      gap: 8px;

      .feature-button {
        background: rgba(255, 255, 255, 0.1);
        border: 1px solid rgba(255, 255, 255, 0.2);
        color: rgba(255, 255, 255, 0.9);
        border-radius: 12px;
        padding: 10px 16px;
        transition: all 0.3s ease;
        display: flex;
        align-items: center;
        gap: 8px;
        justify-content: flex-start;

        &:hover {
          background: rgba(255, 255, 255, 0.2);
          color: white;
          transform: translateY(-1px);
        }
      }
    }
  }

  .quick-actions {
    padding: 0 20px;

    h4 {
      margin: 0 0 16px 0;
      font-size: 16px;
      font-weight: 600;
      color: white;
    }

    .action-list {
      display: flex;
      flex-direction: column;
      gap: 4px;

      .el-button {
        background: transparent;
        border: none;
        color: rgba(255, 255, 255, 0.8);
        padding: 8px 16px;
        border-radius: 8px;
        transition: all 0.3s ease;
        display: flex;
        align-items: center;
        gap: 8px;
        justify-content: flex-start;

        &:hover {
          background: rgba(255, 255, 255, 0.1);
          color: white;
        }
      }
    }
  }

  .group-features {
    h4 {
      margin: 0 0 16px 0;
      font-size: 16px;
      font-weight: 500;
      color: #333;
    }

    .feature-buttons {
      display: flex;
      flex-direction: column;
      gap: 12px;

      .feature-button {
        display: flex;
        align-items: center;
        justify-content: flex-start;
        padding: 12px 16px;
        border-radius: 8px;
        transition: background 0.3s;

        &:hover {
          background: rgba(30, 136, 229, 0.1);
        }

        svg {
          margin-right: 8px;
        }
      }
    }
  }

  .quick-actions {
    h4 {
      margin: 0 0 16px 0;
      font-size: 16px;
      font-weight: 500;
      color: #333;
    }

    .action-list {
      display: flex;
      flex-direction: column;
      gap: 12px;

      .el-button {
        justify-content: flex-start;
        padding: 12px 16px;
        border-radius: 8px;
        transition: background 0.3s;

        &:hover {
          background: rgba(30, 136, 229, 0.1);
        }

        svg {
          margin-right: 8px;
        }
      }
    }
  }
}

@keyframes pulse {
  0% {
    transform: scale(1);
    opacity: 1;
  }

  50% {
    transform: scale(1.2);
    opacity: 0.7;
  }

  100% {
    transform: scale(1);
    opacity: 1;
  }
}

.group-chat-main {
  flex: 1;
  display: flex;
  flex-direction: column;
  overflow: hidden;
  background: #f8fafc;

  .chat-header {
    background: white;
    border-bottom: 1px solid #e2e8f0;
    padding: 20px 24px;
    display: flex;
    align-items: center;
    justify-content: space-between;
    box-shadow: 0 2px 4px rgba(0, 0, 0, 0.05);

    .header-info {
      display: flex;
      align-items: center;
      gap: 16px;

      .header-details {
        .chat-title {
          margin: 0;
          font-size: 20px;
          font-weight: 700;
          color: #1e293b;
        }

        .chat-status {
          display: flex;
          align-items: center;
          gap: 12px;
          font-size: 14px;
          color: #64748b;
          margin-top: 4px;

          .online-indicator {
            display: flex;
            align-items: center;
            gap: 4px;

            &::before {
              content: '';
              width: 8px;
              height: 8px;
              background: #10b981;
              border-radius: 50%;
              animation: pulse 2s infinite;
            }
          }
        }
      }
    }

    .header-actions {
      display: flex;
      gap: 12px;

      .el-button {
        background: #f1f5f9;
        border: 1px solid #e2e8f0;
        color: #475569;
        border-radius: 10px;
        padding: 8px 16px;
        transition: all 0.3s ease;

        &:hover {
          background: #e2e8f0;
          border-color: #cbd5e1;
          transform: translateY(-1px);
        }
      }
    }
  }

  .messages-area {
    flex: 1;
    overflow-y: auto;
    padding: 16px;
    background: #f5f5f5;

    .load-more-history {
      text-align: center;
      margin-bottom: 16px;
    }

    .messages-list {
      .message-group {
        margin-bottom: 24px;
      }

      .date-divider {
        text-align: center;
        margin: 16px 0;

        span {
          background: rgba(0, 0, 0, 0.1);
          color: #666;
          padding: 4px 12px;
          border-radius: 12px;
          font-size: 12px;
        }
      }

      .message-item {
        margin-bottom: 16px;

        &.own-message {
          display: flex;
          justify-content: flex-end;
        }
      }

      .message-wrapper {
        display: flex;
        max-width: 70%;

        &.received {
          .message-avatar {
            margin-right: 8px;
            flex-shrink: 0;
          }

          .message-bubble {
            background: white;
            border-radius: 18px 18px 18px 4px;
          }
        }

        &.sent {
          justify-content: flex-end;

          .message-avatar {
            margin-left: 8px;
            flex-shrink: 0;
          }

          .message-bubble {
            background: #1e88e5;
            color: white;
            border-radius: 18px 18px 4px 18px;
          }

          .message-content {
            text-align: right;
          }
        }
      }

      .message-content {
        flex: 1;
        min-width: 0;
      }

      .sender-name {
        font-size: 12px;
        color: #666;
        margin-bottom: 4px;
      }

      .message-bubble {
        padding: 12px 16px;
        box-shadow: 0 1px 2px rgba(0, 0, 0, 0.1);

        .message-text {
          word-wrap: break-word;
          white-space: pre-wrap;
        }

        .message-image {
          .chat-image {
            max-width: 200px;
            max-height: 200px;
            border-radius: 8px;
          }
        }

        .message-location {
          .location-content {
            display: flex;
            align-items: center;

            .location-info {
              margin-left: 8px;

              .location-name {
                font-weight: 600;
                font-size: 14px;
              }

              .location-address {
                font-size: 12px;
                color: #666;
                margin-top: 2px;
              }
            }
          }
        }

        .message-system {
          font-style: italic;
          color: #666;
          text-align: center;
        }
      }

      .message-info {
        display: flex;
        align-items: center;
        justify-content: space-between;
        margin-top: 4px;
        font-size: 11px;
        color: #999;

        .message-status {
          margin-left: 4px;
        }
      }
    }

    .typing-indicator {
      margin: 16px 0;

      .typing-content {
        display: flex;
        align-items: center;
        background: white;
        border-radius: 18px;
        padding: 8px 16px;
        max-width: fit-content;
        box-shadow: 0 1px 2px rgba(0, 0, 0, 0.1);

        .typing-avatars {
          display: flex;
          margin-right: 8px;

          .typing-avatar {
            margin-right: -8px;
            border: 2px solid white;
          }
        }

        .typing-text {
          font-size: 12px;
          color: #666;
          margin-right: 8px;
        }

        .typing-dots {
          display: flex;

          span {
            width: 4px;
            height: 4px;
            background: #999;
            border-radius: 50%;
            margin-right: 2px;
            animation: typing 1.4s infinite ease-in-out;

            &:nth-child(1) {
              animation-delay: -0.32s;
            }

            &:nth-child(2) {
              animation-delay: -0.16s;
            }
          }
        }
      }
    }
  }
}

.chat-layout {
  flex: 1;
  display: flex;
  flex-direction: column;
  overflow: hidden;

  .chat-container {
    flex: 1;
    overflow-y: auto;
    padding: 16px;
    background: #f5f5f5;
  }

  .load-more-history {
    text-align: center;
    margin-bottom: 16px;
  }

  .messages-list {
    .message-group {
      margin-bottom: 24px;
    }

    .date-divider {
      text-align: center;
      margin: 16px 0;

      span {
        background: rgba(0, 0, 0, 0.1);
        color: #666;
        padding: 4px 12px;
        border-radius: 12px;
        font-size: 12px;
      }
    }

    .message-item {
      margin-bottom: 16px;

      &.own-message {
        display: flex;
        justify-content: flex-end;
      }
    }

    .message-wrapper {
      display: flex;
      max-width: 70%;

      &.received {
        .message-avatar {
          margin-right: 8px;
          flex-shrink: 0;
        }

        .message-bubble {
          background: white;
          border-radius: 18px 18px 18px 4px;
        }
      }

      &.sent {
        justify-content: flex-end;

        .message-avatar {
          margin-left: 8px;
          flex-shrink: 0;
        }

        .message-bubble {
          background: #1e88e5;
          color: white;
          border-radius: 18px 18px 4px 18px;
        }

        .message-content {
          text-align: right;
        }
      }
    }

    .message-content {
      flex: 1;
      min-width: 0;
    }

    .sender-name {
      font-size: 12px;
      color: #666;
      margin-bottom: 4px;
    }

    .message-bubble {
      padding: 12px 16px;
      box-shadow: 0 1px 2px rgba(0, 0, 0, 0.1);

      .message-text {
        word-wrap: break-word;
        white-space: pre-wrap;
      }

      .message-image {
        .chat-image {
          max-width: 200px;
          max-height: 200px;
          border-radius: 8px;
        }
      }

      .message-location {
        .location-content {
          display: flex;
          align-items: center;

          .location-info {
            margin-left: 8px;

            .location-name {
              font-weight: 600;
              font-size: 14px;
            }

            .location-address {
              font-size: 12px;
              color: #666;
              margin-top: 2px;
            }
          }
        }
      }

      .message-system {
        font-style: italic;
        color: #666;
        text-align: center;
      }
    }

    .message-info {
      display: flex;
      align-items: center;
      justify-content: space-between;
      margin-top: 4px;
      font-size: 11px;
      color: #999;

      .message-status {
        margin-left: 4px;
      }
    }
  }

  .typing-indicator {
    padding: 8px 16px;
    background: #f8f9fa;
    border-top: 1px solid #dee2e6;
    display: flex;
    align-items: center;
    gap: 8px;

    .typing-avatars {
      display: flex;
      gap: -8px;

      .typing-avatar {
        border: 2px solid white;
        margin-left: -4px;

        &:first-child {
          margin-left: 0;
        }
      }
    }

    .typing-text {
      font-size: 14px;
      color: #666;
    }

    .typing-dots {
      display: flex;
      gap: 2px;

      span {
        width: 4px;
        height: 4px;
        background: #666;
        border-radius: 50%;
        animation: typing-dot 1.4s infinite;

        &:nth-child(2) {
          animation-delay: 0.2s;
        }

        &:nth-child(3) {
          animation-delay: 0.4s;
        }
      }
    }
  }
}

@keyframes messageSlideIn {
  from {
    opacity: 0;
    transform: translateY(10px);
  }

  to {
    opacity: 1;
    transform: translateY(0);
  }
}

@keyframes typing {

  0%,
  80%,
  100% {
    transform: scale(0);
    opacity: 0.5;
  }

  40% {
    transform: scale(1);
    opacity: 1;
  }
}

@keyframes typing-dot {

  0%,
  60%,
  100% {
    opacity: 0.4;
    transform: scale(1);
  }

  30% {
    opacity: 1;
    transform: scale(1.2);
  }
}

// 移动端适配
@media (max-width: 768px) {
  .group-chat-page {
    .desktop-layout {
      padding: 10px;

      .group-chat-layout {
        border-radius: 12px;
        margin-top: 10px;
        margin-bottom: 10px;
      }
    }
  }

  .group-sidebar {
    width: 100%;
    border-radius: 12px 12px 0 0;
  }

  .group-chat-main {
    .chat-header {
      padding: 16px;

      .header-info {
        .header-details {
          .chat-title {
            font-size: 18px;
          }
        }
      }
    }

    .messages-area {
      .chat-container {
        padding: 16px;
      }

      .messages-list {
        .message-wrapper {
          max-width: 85%;
        }
      }
    }
  }

  .chat-input-area {
    padding: 12px;

    .input-tools {
      gap: 8px;
    }
  }
}

// 菜单项样式
.menu-item {
  width: 100%;
  text-align: left;
  padding: 0.75rem 1rem;
  color: #374151;
  font-size: 0.875rem;
  transition: all 0.2s ease-in-out;

  &:hover {
    background: #f3f4f6;
  }

  &.text-danger {
    color: #dc2626;

    &:hover {
      background: #fef2f2;
      color: #b91c1c;
    }
  }
}

// 模态框样式
.modal-overlay {
  position: fixed;
  inset: 0;
  z-index: var(--z-dropdown);
  background: rgba(0, 0, 0, 0.5);
  display: flex;
  align-items: center;
  justify-content: center;
}

.modal-content {
  background: #ffffff;
  border-radius: 0.5rem;
  padding: 1.5rem;
  max-width: 28rem;
  width: 100%;
  margin: 0 1rem;
}

.modal-title {
  font-size: 1.125rem;
  font-weight: 600;
  margin-bottom: 1rem;
  color: #111827;
}

.modal-description {
  color: #6b7280;
  margin-bottom: 1rem;
}

.modal-actions {
  display: flex;
  justify-content: flex-end;
  gap: 0.5rem;
}

.modal-btn {
  padding: 0.5rem 1rem;
  border-radius: 0.25rem;
  border: none;
  cursor: pointer;
  transition: all 0.2s ease-in-out;

  &--secondary {
    color: #6b7280;
    background: transparent;

    &:hover {
      background: #f3f4f6;
    }
  }
}

// 深色模式适配
@media (prefers-color-scheme: dark) {
  .modal-content {
    background: #334155;
  }

  .modal-title {
    color: #f1f5f9;
  }

  .modal-description {
    color: #94a3b8;
  }

  .modal-btn--secondary {
    color: #94a3b8;

    &:hover {
      background: #475569;
    }
  }

  .group-chat-main {
    background: #1e293b;

    .chat-header {
      background: #334155;
      border-bottom-color: #475569;

      .header-info {
        .header-details {
          .chat-title {
            color: #f1f5f9;
          }

          .chat-status {
            color: #94a3b8;
          }
        }
      }

      .header-actions {
        .el-button {
          background: #475569;
          border-color: #64748b;
          color: #f1f5f9;

          &:hover {
            background: #64748b;
          }
        }
      }
    }

    .messages-area {
      background: linear-gradient(to bottom, #1e293b, #334155);

      .messages-list {
        .message-wrapper {
          &.received {
            .message-bubble {
              background: #334155;
              color: #f1f5f9;
              border-color: #475569;
            }
          }
        }

        .date-divider {
          span {
            background: rgba(71, 85, 105, 0.3);
            color: #94a3b8;
          }
        }
      }

      .typing-indicator {
        background: rgba(30, 41, 59, 0.8);
        border-top-color: #475569;

        .typing-text {
          color: #94a3b8;
        }
      }
    }
  }
}
</style>
