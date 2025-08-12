<template>
  <div class="flex flex-col h-screen">
    <!-- 聊天头部 -->
    <div class="border-b bg-white px-4 py-3 flex items-center justify-between">
      <div class="flex items-center space-x-3">
        <UButton 
          variant="ghost" 
          size="sm" 
          icon="i-heroicons-arrow-left"
          @click="goBack"
          class="lg:hidden"
        />
        
        <UAvatar 
          :src="chatInfo?.avatar_url" 
          :alt="chatInfo?.name"
          size="md"
        >
          {{ chatInfo?.name?.charAt(0) || 'C' }}
        </UAvatar>
        
        <div>
          <h1 class="font-semibold text-gray-900">
            {{ chatInfo?.name || '聊天' }}
          </h1>
          <p class="text-sm text-gray-600">
            {{ getStatusText() }}
          </p>
        </div>
      </div>
      
      <!-- 操作按钮 -->
      <div class="flex items-center space-x-2">
        <UButton 
          variant="ghost" 
          size="sm" 
          icon="i-heroicons-phone"
          @click="makeCall"
        />
        <UButton 
          variant="ghost" 
          size="sm" 
          icon="i-heroicons-video-camera"
          @click="makeVideoCall"
        />
        <UDropdown :items="chatActions">
          <UButton 
            variant="ghost" 
            size="sm" 
            icon="i-heroicons-ellipsis-vertical"
          />
        </UDropdown>
      </div>
    </div>

    <!-- 消息列表 -->
    <div 
      ref="messagesContainer"
      class="flex-1 overflow-y-auto px-4 py-4 space-y-4 bg-gray-50"
    >
      <div 
        v-for="message in messages" 
        :key="message.id"
        :class="[
          'flex',
          message.is_own ? 'justify-end' : 'justify-start'
        ]"
      >
        <div 
          :class="[
            'max-w-xs lg:max-w-md px-4 py-2 rounded-lg',
            message.is_own 
              ? 'bg-primary-500 text-white' 
              : 'bg-white text-gray-900 shadow-sm'
          ]"
        >
          <!-- 消息内容 -->
          <div class="mb-1">
            {{ message.content }}
          </div>
          
          <!-- 消息时间 -->
          <div 
            :class="[
              'text-xs',
              message.is_own ? 'text-primary-100' : 'text-gray-500'
            ]"
          >
            {{ formatMessageTime(message.created_at) }}
          </div>
        </div>
      </div>

      <!-- 加载更多 -->
      <div v-if="hasMoreMessages" class="text-center py-4">
        <UButton 
          variant="outline" 
          size="sm" 
          @click="loadMoreMessages"
          :loading="loadingMore"
        >
          加载更多消息
        </UButton>
      </div>
    </div>

    <!-- 消息输入框 -->
    <div class="border-t bg-white p-4">
      <div class="flex items-end space-x-3">
        <!-- 附加功能按钮 -->
        <UButton 
          variant="ghost" 
          size="sm" 
          icon="i-heroicons-plus"
          @click="showAttachMenu"
        />
        
        <!-- 输入框 -->
        <div class="flex-1">
          <UTextarea
            v-model="newMessage"
            placeholder="输入消息..."
            :rows="1"
            resize="none"
            class="resize-none"
            @keydown.enter.prevent="sendMessage"
          />
        </div>
        
        <!-- 发送按钮 -->
        <UButton 
          :disabled="!newMessage.trim()"
          @click="sendMessage"
          :loading="sending"
        >
          <UIcon name="i-heroicons-paper-airplane" class="w-4 h-4" />
        </UButton>
      </div>
    </div>
  </div>
</template>

<script setup>
// 页面元数据
definePageMeta({
  title: '聊天',
  requiresAuth: true,
  layout: false // 使用全屏布局
})

const route = useRoute()
const toast = useToast()

// 聊天参数
const chatType = route.params.type // 'user' 或 'group'
const chatId = route.params.id

// 聊天信息
const chatInfo = ref(null)
const messages = ref([])
const newMessage = ref('')
const sending = ref(false)
const loadingMore = ref(false)
const hasMoreMessages = ref(true)

// DOM 引用
const messagesContainer = ref(null)

// 聊天操作菜单
const chatActions = computed(() => {
  const baseActions = [
    [{
      label: '聊天信息',
      icon: 'i-heroicons-information-circle',
      click: showChatInfo
    }]
  ]
  
  if (chatType === 'group') {
    baseActions.push([{
      label: '群组设置',
      icon: 'i-heroicons-cog-6-tooth',
      click: showGroupSettings
    }])
  }
  
  return baseActions
})

// 获取状态文本
const getStatusText = () => {
  if (chatType === 'group') {
    return `${chatInfo.value?.member_count || 0} 名成员`
  } else {
    return chatInfo.value?.is_online ? '在线' : '离线'
  }
}

// 格式化消息时间
const formatMessageTime = (date) => {
  const messageDate = new Date(date)
  const now = new Date()
  const today = new Date(now.getFullYear(), now.getMonth(), now.getDate())
  const messageDay = new Date(messageDate.getFullYear(), messageDate.getMonth(), messageDate.getDate())
  
  const timeStr = messageDate.toLocaleTimeString('zh-CN', { 
    hour: '2-digit', 
    minute: '2-digit' 
  })
  
  if (messageDay.getTime() === today.getTime()) {
    return timeStr
  } else {
    return `${messageDate.toLocaleDateString('zh-CN')} ${timeStr}`
  }
}

// 初始化聊天数据
const initChat = async () => {
  // 模拟聊天信息
  if (chatType === 'user') {
    chatInfo.value = {
      id: chatId,
      name: '张三',
      avatar_url: '',
      is_online: true
    }
  } else {
    chatInfo.value = {
      id: chatId,
      name: '附近的咖啡爱好者',
      avatar_url: '',
      member_count: 128
    }
  }
  
  // 模拟消息数据
  messages.value = [
    {
      id: 1,
      content: '大家好！',
      is_own: false,
      created_at: new Date(Date.now() - 1000 * 60 * 30)
    },
    {
      id: 2,
      content: '你好！很高兴认识大家',
      is_own: true,
      created_at: new Date(Date.now() - 1000 * 60 * 25)
    },
    {
      id: 3,
      content: '有没有人知道附近哪家咖啡店比较好？',
      is_own: false,
      created_at: new Date(Date.now() - 1000 * 60 * 20)
    }
  ]
  
  // 滚动到底部
  nextTick(() => {
    scrollToBottom()
  })
}

// 发送消息
const sendMessage = async () => {
  if (!newMessage.value.trim() || sending.value) return
  
  sending.value = true
  
  try {
    // 添加到消息列表
    const message = {
      id: Date.now(),
      content: newMessage.value.trim(),
      is_own: true,
      created_at: new Date()
    }
    
    messages.value.push(message)
    newMessage.value = ''
    
    // 滚动到底部
    nextTick(() => {
      scrollToBottom()
    })
    
    // 这里应该调用 API 发送消息
    // await api.sendMessage(chatType, chatId, message.content)
    
  } catch (error) {
    toast.add({
      title: '发送失败',
      description: '请重试',
      color: 'error'
    })
  } finally {
    sending.value = false
  }
}

// 加载更多消息
const loadMoreMessages = async () => {
  if (loadingMore.value) return
  
  loadingMore.value = true
  
  try {
    // 模拟加载更多消息
    await new Promise(resolve => setTimeout(resolve, 1000))
    
    // 这里应该调用 API 加载历史消息
    hasMoreMessages.value = false
    
  } catch (error) {
    toast.add({
      title: '加载失败',
      description: '请重试',
      color: 'error'
    })
  } finally {
    loadingMore.value = false
  }
}

// 滚动到底部
const scrollToBottom = () => {
  if (messagesContainer.value) {
    messagesContainer.value.scrollTop = messagesContainer.value.scrollHeight
  }
}

// 返回上一页
const goBack = () => {
  navigateTo('/messages')
}

// 拨打电话
const makeCall = () => {
  toast.add({
    title: '功能开发中',
    description: '语音通话功能即将上线',
    color: 'info'
  })
}

// 视频通话
const makeVideoCall = () => {
  toast.add({
    title: '功能开发中',
    description: '视频通话功能即将上线',
    color: 'info'
  })
}

// 显示附件菜单
const showAttachMenu = () => {
  toast.add({
    title: '功能开发中',
    description: '文件上传功能即将上线',
    color: 'info'
  })
}

// 显示聊天信息
const showChatInfo = () => {
  toast.add({
    title: '功能开发中',
    description: '聊天信息功能即将上线',
    color: 'info'
  })
}

// 显示群组设置
const showGroupSettings = () => {
  toast.add({
    title: '功能开发中',
    description: '群组设置功能即将上线',
    color: 'info'
  })
}

// 初始化
onMounted(() => {
  initChat()
})
</script>
