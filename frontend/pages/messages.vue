<template>
  <div class="app-page messages-page">
    <div class="page-header">
      <h1 class="page-title">消息中心</h1>
      <p class="page-subtitle">与朋友保持联系</p>
    </div>
    
    <div class="app-container">
      <!-- 搜索栏 -->
      <div class="content-card search-card">
        <div class="search-bar">
          <NInput
            v-model="searchQuery"
            placeholder="搜索对话..."
            :prefix-icon="Search"
            clearable
            @input="handleSearch"
          />
          <NButton 
            type="primary" 
            @click="showNewMessageDialog = true"
          >
            <NIcon><Plus /></NIcon>
            新建对话
          </NButton>
        </div>
      </div>

      <!-- 对话列表 -->
      <div class="content-card">
        <div class="card-header">
          <h3 class="card-title">对话列表</h3>
          <NButton 
            type="primary" 
            size="small" 
            @click="loadConversations"
            :loading="isLoading"
          >
            <NIcon><Refresh /></NIcon>
            刷新
          </NButton>
        </div>
        
        <div v-if="filteredConversations.length > 0" class="conversation-list">
          <div 
            v-for="conversation in filteredConversations" 
            :key="conversation.conversation_id"
            class="conversation-item"
            @click="navigateToChat(conversation.conversation_id)"
          >
            <div class="conversation-item__avatar">
              <NAvatar 
                :size="50"
                :src="conversation.avatar_url || defaultImages.userAvatar" 
              />
              <div 
                v-if="conversation.unread_count > 0" 
                class="conversation-item__badge"
              >
                {{ conversation.unread_count }}
              </div>
            </div>
            
            <div class="conversation-item__content">
              <div class="conversation-item__header">
                <h4 class="conversation-item__name">
                  {{ conversation.display_name || conversation.name || '未知用户' }}
                </h4>
                <span class="conversation-item__time">
                  {{ formatTime(conversation.last_message_time) }}
                </span>
              </div>
              
              <div class="conversation-item__message">
                <p class="conversation-item__text">
                  {{ conversation.last_message_content || '暂无消息' }}
                </p>
                <div class="conversation-item__status">
                  <NIcon v-if="conversation.unread_count === 0" class="read-icon">
                    <Check />
                  </NIcon>
                  <NIcon v-else class="unread-icon">
                    <Clock />
                  </NIcon>
                </div>
              </div>
            </div>
          </div>
        </div>
        
        <div v-else-if="isLoading" class="loading-state">
          <div class="loading-state__spinner">
            <NIcon><Loading /></NIcon>
          </div>
          <span>加载对话中...</span>
        </div>
        
        <div v-else class="empty-state">
          <div class="empty-state__icon">
            <NIcon><Message /></NIcon>
          </div>
          <h4 class="empty-state__title">暂无对话</h4>
          <p class="empty-state__description">开始与朋友聊天吧</p>
          <NButton 
            type="primary" 
            @click="showNewMessageDialog = true"
          >
            <NIcon><Plus /></NIcon>
            发起对话
          </NButton>
        </div>
      </div>
    </div>

    <!-- 新建对话对话框 -->
    <NewMessageDialog 
      v-model="showNewMessageDialog"
      @success="handleNewMessageSuccess"
    />
  </div>
</template>

<script setup lang="ts">
// Vue 3 Composition API
import { ref, computed, onMounted } from 'vue'

// Naive UI 组件
import { NInput, NButton, NIcon, NAvatar, useMessage } from 'naive-ui'
import { Search, Add as Plus, Chatbubble as Message, Reload as Loading, Refresh, Checkmark as Check, Time as Clock } from '@vicons/ionicons5'

// API实例
const message = useMessage()

// 项目组件
import NewMessageDialog from '@/components/message/NewMessageDialog.vue'

// 状态管理
import { useMessageStore } from '@/stores/message'

// 工具函数
import { defaultImages } from '@/utils/assets'

// 类型定义
import type { Conversation } from '@/types/models/message'

// 页面元数据
definePageMeta({
  layout: 'default',
  title: '消息',
  middleware: 'auth'
})

// 响应式数据
const searchQuery = ref<string>('')
const isLoading = ref<boolean>(false)
const showNewMessageDialog = ref<boolean>(false)
const conversations = ref<Conversation[]>([])

// Store
const messageStore = useMessageStore()

// 计算属性
const filteredConversations = computed(() => {
  if (!searchQuery.value) {
    return conversations.value
  }
  
  const query = searchQuery.value.toLowerCase()
  return conversations.value.filter(conv => 
    (conv.display_name || conv.name)?.toLowerCase().includes(query) ||
    (conv.last_message_content && conv.last_message_content.toLowerCase().includes(query))
  )
})

// 生命周期钩子
onMounted(async () => {
  await loadConversations()
})

// 方法定义
const loadConversations = async () => {
  if (isLoading.value) return
  
  isLoading.value = true
  try {
    const data = await messageStore.fetchConversations()
    conversations.value = (data as any)?.items || data || []
  } catch (error) {
    console.error('加载对话失败:', error)
    message.error('加载对话失败')
    conversations.value = []
  } finally {
    isLoading.value = false
  }
}

const handleSearch = () => {
  // 搜索逻辑已在计算属性中处理
}

const navigateToChat = (conversationId: string) => {
  navigateTo(`/chat/${conversationId}`)
}

const handleNewMessageSuccess = async () => {
  showNewMessageDialog.value = false
  message.success('对话已创建')
  // 重新加载对话列表
  await loadConversations()
}

const formatTime = (timestamp?: string) => {
  if (!timestamp) return ''
  
  const now = new Date()
  const time = new Date(timestamp)
  const diff = now.getTime() - time.getTime()
  const minutes = Math.floor(diff / 60000)
  
  if (minutes < 1) return '刚刚'
  if (minutes < 60) return `${minutes}分钟前`
  if (minutes < 1440) return `${Math.floor(minutes / 60)}小时前`
  if (minutes < 10080) return `${Math.floor(minutes / 1440)}天前`
  
  return time.toLocaleDateString()
}

// 设置集成 - 消息页面监听通知设置
import { useAppSettings } from '@/composables/useAppSettings'
const { playNotificationSound, triggerVibration, isInQuietHours } = useAppSettings()

// 监听新消息并应用通知设置
onMounted(() => {
  // 监听WebSocket消息事件
  window.addEventListener('websocket-message-received', (event: any) => {
    const { message_type } = event.detail
    
    // 如果是安静时间，不播放声音和振动
    if (isInQuietHours.value) {
      return
    }
    
    // 根据消息类型播放通知
    if (message_type === 'direct_message' || message_type === 'group_message') {
      playNotificationSound()
      triggerVibration()
    }
  })
})
</script>