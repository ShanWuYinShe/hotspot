<template>
  <div class="container mx-auto px-4 py-8">
    <div class="mb-6">
      <h1 class="text-2xl font-bold text-gray-900 mb-2">消息</h1>
      <p class="text-gray-600">与朋友和群组保持联系</p>
    </div>

    <!-- 消息搜索 -->
    <div class="mb-6">
      <UInput
        v-model="searchQuery"
        placeholder="搜索对话..."
        icon="i-heroicons-magnifying-glass"
        size="lg"
      />
    </div>

    <!-- 消息列表 -->
    <div class="space-y-4">
      <!-- 空状态 -->
      <div v-if="conversations.length === 0" class="text-center py-12">
        <UIcon name="i-heroicons-chat-bubble-left-right" class="w-16 h-16 mx-auto mb-4 text-gray-400" />
        <h3 class="text-lg font-medium text-gray-900 mb-2">暂无对话</h3>
        <p class="text-gray-600 mb-4">开始与其他用户或群组聊天吧</p>
        <UButton @click="startNewChat">
          <UIcon name="i-heroicons-plus" class="w-4 h-4 mr-2" />
          开始新对话
        </UButton>
      </div>

      <!-- 对话列表 -->
      <UCard 
        v-for="conversation in filteredConversations" 
        :key="conversation.id"
        class="cursor-pointer hover:bg-gray-50 transition-colors"
        @click="openChat(conversation)"
      >
        <div class="flex items-center space-x-4">
          <!-- 头像 -->
          <UAvatar 
            :src="conversation.avatar" 
            :alt="conversation.name"
            size="lg"
          >
            {{ conversation.name.charAt(0) }}
          </UAvatar>

          <!-- 对话信息 -->
          <div class="flex-1 min-w-0">
            <div class="flex items-center justify-between mb-1">
              <h3 class="font-medium text-gray-900 truncate">
                {{ conversation.name }}
              </h3>
              <span class="text-sm text-gray-500">
                {{ formatTime(conversation.lastMessageTime) }}
              </span>
            </div>
            <p class="text-sm text-gray-600 truncate">
              {{ conversation.lastMessage }}
            </p>
          </div>

          <!-- 未读消息数 -->
          <UBadge 
            v-if="conversation.unreadCount > 0"
            :label="conversation.unreadCount.toString()"
            color="red"
            variant="solid"
          />
        </div>
      </UCard>
    </div>
  </div>
</template>

<script setup>
// 页面元数据
definePageMeta({
  title: '消息',
  requiresAuth: true
})

// 响应式状态
const searchQuery = ref('')

// 模拟对话数据
const conversations = ref([
  {
    id: 1,
    name: '张三',
    avatar: '',
    lastMessage: '你好，最近怎么样？',
    lastMessageTime: new Date(),
    unreadCount: 2,
    type: 'user'
  },
  {
    id: 2,
    name: '朋友群',
    avatar: '',
    lastMessage: '大家周末有什么计划吗？',
    lastMessageTime: new Date(Date.now() - 3600000),
    unreadCount: 0,
    type: 'group'
  }
])

// 过滤对话
const filteredConversations = computed(() => {
  if (!searchQuery.value) return conversations.value
  
  return conversations.value.filter(conv => 
    conv.name.toLowerCase().includes(searchQuery.value.toLowerCase()) ||
    conv.lastMessage.toLowerCase().includes(searchQuery.value.toLowerCase())
  )
})

// 格式化时间
const formatTime = (time) => {
  const now = new Date()
  const diff = now.getTime() - time.getTime()
  const minutes = Math.floor(diff / 60000)
  const hours = Math.floor(diff / 3600000)
  const days = Math.floor(diff / 86400000)

  if (minutes < 1) return '刚刚'
  if (minutes < 60) return `${minutes}分钟前`
  if (hours < 24) return `${hours}小时前`
  if (days < 7) return `${days}天前`
  return time.toLocaleDateString()
}

// 打开聊天
const openChat = (conversation) => {
  if (conversation.type === 'group') {
    navigateTo(`/group/${conversation.id}/chat`)
  } else {
    navigateTo(`/chat/${conversation.id}`)
  }
}

// 开始新对话
const startNewChat = () => {
  // TODO: 实现新对话逻辑
  const toast = useToast()
  toast.add({
    title: '功能开发中',
    description: '新对话功能即将上线',
    color: 'info'
  })
}
</script>
