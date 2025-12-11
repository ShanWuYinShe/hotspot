<template>
  <UDashboardPanel>
    <!-- 顶部工具栏 -->
    <template #header>
      <UDashboardToolbar>
        <template #left>
          <div class="flex items-center gap-3">
            <div class="icon-lg icon-container-rounded-lg bg-linear-to-br from-blue-500 to-purple-600 shadow-lg">
              <UIcon name="i-heroicons-chat-bubble-left-right" class="icon-md text-white" />
            </div>
            <div>
              <h1 class="text-lg font-semibold text-primary-content">消息中心</h1>
              <p class="text-sm text-secondary-content">与朋友和群组保持联系</p>
            </div>
          </div>
        </template>

        <template #right>
          <div class="flex items-center gap-2">
            <UBadge 
              v-if="totalUnread > 0" 
              :label="`${totalUnread} 条未读`"
              color="primary" 
              variant="solid"
              size="sm"
            />
            
            <UButton 
              icon="i-heroicons-arrow-path"
              variant="ghost"
              size="sm"
              :loading="loading"
              @click="loadConversations"
            >
              刷新
            </UButton>

            <UButton 
              icon="i-heroicons-user-plus"
              variant="outline"
              size="sm"
              @click="openNewChatModal"
            >
              新建聊天
            </UButton>
          </div>
        </template>
      </UDashboardToolbar>
    </template>

    <!-- 主内容区域 -->
    <template #body>
      <div class="h-full flex flex-col">
        <!-- 快速操作卡片 -->
        <div class="grid grid-cols-1 md:grid-cols-2 gap-4 mb-6">
          <UPageCard
            title="发现新朋友"
            description="搜索并添加新的聊天伙伴"
            icon="i-heroicons-user-plus"
            class="interactive-card cursor-pointer"
            @click="navigateTo('/search')"
          >
            <template #icon>
              <div class="icon-avatar-sm icon-container-rounded-lg bg-icon-primary">
                <UIcon name="i-heroicons-user-plus" class="icon-md text-white" />
              </div>
            </template>
          </UPageCard>

          <UPageCard
            title="加入群组"
            description="发现并加入感兴趣的群组"
            icon="i-heroicons-user-group"
            class="interactive-card cursor-pointer"
            @click="navigateTo('/groups/discover')"
          >
            <template #icon>
              <div class="icon-avatar-sm icon-container-rounded-lg bg-icon-purple">
                <UIcon name="i-heroicons-user-group" class="icon-md text-white" />
              </div>
            </template>
          </UPageCard>
        </div>

        <!-- 会话列表 -->
        <div class="flex-1">
          <UCard class="h-full">
            <template #header>
              <div class="flex items-center justify-between">
                <h2 class="text-lg font-semibold">最近会话</h2>
                <div class="flex items-center gap-2">
                  <UButton
                    variant="ghost"
                    size="xs"
                    icon="i-heroicons-funnel"
                    @click="toggleFilters"
                  >
                    筛选
                  </UButton>
                </div>
              </div>
            </template>

            <!-- 筛选器 -->
            <div v-if="showFilters" class="mb-4 p-4 bg-interactive rounded-lg">
              <div class="flex flex-wrap gap-2">
                <UButton
                  v-for="filter in conversationFilters"
                  :key="filter.key"
                  :variant="activeFilter === filter.key ? 'solid' : 'ghost'"
                  size="xs"
                  @click="setFilter(filter.key)"
                >
                  {{ filter.label }}
                </UButton>
              </div>
            </div>

            <!-- 加载状态 -->
            <div v-if="loading" class="space-y-4">
              <div v-for="i in 5" :key="i" class="flex items-center gap-3 p-3">
                <USkeleton class="w-12 h-12 rounded-full" />
                <div class="flex-1 space-y-2">
                  <USkeleton class="h-4 w-3/4" />
                  <USkeleton class="h-3 w-1/2" />
                </div>
                <USkeleton class="h-3 w-12" />
              </div>
            </div>

            <!-- 错误状态 -->
            <UEmpty
              v-else-if="error"
              icon="i-heroicons-exclamation-triangle"
              title="加载失败"
              :description="error"
            >
              <template #actions>
                <UButton @click="loadConversations" variant="outline">重试</UButton>
              </template>
            </UEmpty>

            <!-- 空状态 -->
            <UEmpty
              v-else-if="filteredConversations.length === 0"
              icon="i-heroicons-chat-bubble-left-ellipsis"
              title="暂无会话"
              description="开始与朋友聊天吧！"
            >
              <template #actions>
                <UButton @click="navigateTo('/search')" icon="i-heroicons-user-plus">
                  发现新朋友
                </UButton>
              </template>
            </UEmpty>

            <!-- 会话列表 -->
            <div v-else class="space-y-2">
              <div
                v-for="conversation in filteredConversations"
                :key="conversation.chat_id"
                class="interactive-list-item flex items-center gap-3 p-3 rounded-lg cursor-pointer"
                @click="openConversation(conversation)"
              >
                <!-- 头像 -->
                <div class="relative shrink-0">
                  <UAvatar
                    :alt="conversation.name"
                    size="lg"
                    :src="conversation.avatar_url"
                  >
                    {{ conversation.name?.charAt(0) || "?" }}
                  </UAvatar>
                  
                  <!-- 在线状态 -->
                  <div
                    v-if="conversation.is_online"
                    class="badge-position-br status-dot-online"
                  />
                  
                  <!-- 群组标识 -->
                  <div
                    v-if="conversation.chat_type === 'group'"
                    class="badge-position-br status-dot-online bg-icon-primary icon-container"
                  >
                    <UIcon name="i-heroicons-user-group" class="icon-xs text-white" />
                  </div>
                </div>

                <!-- 会话信息 -->
                <div class="flex-1 min-w-0">
                  <div class="flex items-center justify-between mb-1">
                    <h3 class="font-medium text-primary-content truncate">
                      {{ conversation.name }}
                    </h3>
                    <span class="text-xs text-secondary-content shrink-0">
                      {{ formatTime(conversation.last_activity_at) }}
                    </span>
                  </div>
                  
                  <div class="flex items-center justify-between">
                    <p class="text-sm text-secondary-content truncate">
                      <span v-if="conversation.last_message?.sender_id && conversation.chat_type === 'group'" class="text-secondary-strong">
                        {{ conversation.last_message?.sender_id }}:
                      </span>
                      {{ conversation.last_message?.content || '暂无消息' }}
                    </p>
                    
                    <!-- 未读消息数 -->
                    <UBadge
                      v-if="conversation.unread_count > 0"
                      :label="conversation.unread_count > 99 ? '99+' : conversation.unread_count.toString()"
                      color="primary"
                      size="xs"
                      class="shrink-0"
                    />
                  </div>
                </div>

                <!-- 操作菜单 -->
                <UDropdownMenu
                  :items="getConversationMenuItems(conversation)"
                  @click.stop
                >
                  <UButton
                    variant="ghost"
                    size="xs"
                    icon="i-heroicons-ellipsis-vertical"
                    square
                  />
                </UDropdownMenu>
              </div>
            </div>
          </UCard>
        </div>
      </div>
    </template>
  </UDashboardPanel>
</template>

<script setup lang="ts">
import { getPageSeoConfig, buildSeoMeta } from '~/config/seo';
// 从数据源获取 SEO 配置
const seoConfig = getPageSeoConfig('chat')

// 需要登录 - 聊天功能需要认证
definePageMeta({
  title: seoConfig.title,
  description: seoConfig.description,
  requiresAuth: true,
  layout: "dashboard"
});

// 设置 SEO meta
useSeoMeta(buildSeoMeta('chat'));

import { type Conversation } from '~/stores/messages.store'
import { getErrorMessage } from '~/utils/error'
import { usePrivateChatsService } from '~/services/private_chats.service'
import { usePrivateMessagesService } from '~/services/private_messages.service'

// 使用水合安全的 store 访问
const authStore = useAuthStore();
const messagesStore = useMessagesStore();
const privateChatsService = usePrivateChatsService();
const privateMessagesService = usePrivateMessagesService();
const modal = useModal();
const { formatTime } = useFormatters();

// 响应式状态
const loading = ref(false);
const error = ref<string | null>(null);
const showFilters = ref(false);
const activeFilter = ref<string>('all');

// 计算属性
const totalUnread = computed(() => messagesStore.totalUnreadCount);
const conversations = computed(() => messagesStore.conversationsList || []);

// 筛选器配置
const conversationFilters = [
  { key: 'all', label: '全部' },
  { key: 'unread', label: '未读' },
  { key: 'private', label: '私聊' },
  { key: 'group', label: '群聊' },
  { key: 'online', label: '在线' }
];

// 筛选后的会话列表
const filteredConversations = computed(() => {
  let filtered = [...conversations.value];
  
  switch (activeFilter.value) {
    case 'unread':
      filtered = filtered.filter(c => c.unread_count > 0);
      break;
    case 'private':
      filtered = filtered.filter(c => c.chat_type === 'private');
      break;
    case 'group':
      filtered = filtered.filter(c => c.chat_type === 'group');
      break;
    case 'online':
      filtered = filtered.filter(c => c.is_online === true)
      break
  }
  
  // 按最后活动时间排序
  return filtered.sort((a, b) => 
    new Date(b.last_activity_at).getTime() - new Date(a.last_activity_at).getTime()
  );
});

// 方法
const loadConversations = async () => {
  try {
    loading.value = true;
    error.value = null;
    // 加载私聊列表
    const response = await privateChatsService.getPrivateChats();
    // 转换为 Conversation 格式并更新 store
    const conversationList: Conversation[] = response.items.map(item => ({
      chat_id: item.id,
      chat_type: 'private' as const,
      target_user_id: item.other_user.user_id,  // 保存对方用户ID
      name: item.other_user.nickname,
      avatar_url: item.other_user.avatar_url,
      unread_count: item.unread_count,
      last_message: item.last_message,
      last_activity_at: item.last_activity_at,
      is_online: false
    }));
    messagesStore.setConversations(conversationList);
  } catch (err: unknown) {
    console.error('加载会话失败:', err);
    error.value = getErrorMessage(err, '加载会话失败');
  } finally {
    loading.value = false;
  }
};

const toggleFilters = () => {
  showFilters.value = !showFilters.value;
};

const setFilter = (filterKey: string) => {
  activeFilter.value = filterKey;
};

const openConversation = (conversation: Conversation) => {
  if (conversation.chat_type === 'group') {
    navigateTo(`/chat/groups/${conversation.chat_id}`);
  } else {
    // 私聊使用目标用户ID导航
    if (conversation.target_user_id) {
      navigateTo(`/chat/users/${conversation.target_user_id}`);
    } else {
      console.error('私聊会话缺少目标用户ID');
    }
  }
};

const openNewChatModal = () => {
  // 这里可以打开新建聊天的模态框
  navigateTo('/search');
};

// formatTime 函数已从统一工具函数导入

const getConversationMenuItems = (conversation: Conversation) => {
  const items = [
    [{
      label: '标记为已读',
      icon: 'i-heroicons-check',
      click: () => markAsRead(conversation.chat_id),
      disabled: conversation.unread_count === 0
    }],
    [{
      label: '置顶聊天',
      icon: 'i-heroicons-bookmark',
      click: () => togglePin(conversation.chat_id)
    }, {
      label: '免打扰',
      icon: 'i-heroicons-bell-slash',
      click: () => toggleMute(conversation.chat_id)
    }],
    [{
      label: '删除聊天',
      icon: 'i-heroicons-trash',
      click: () => deleteConversation(conversation.chat_id),
      class: 'text-error'
    }]
  ];
  
  return items;
};

const markAsRead = async (conversationId: string) => {
  try {
    await privateMessagesService.markPrivateMessagesAsRead({ id: conversationId });
    messagesStore.updateUnreadCount(conversationId, 0);
  } catch (err) {
    console.error('标记已读失败:', err);
  }
};

const togglePin = async (conversationId: string) => {
  // 实现置顶功能
  console.log('置顶聊天:', conversationId);
};

const toggleMute = async (conversationId: string) => {
  // 实现免打扰功能
  console.log('切换免打扰:', conversationId);
};

const deleteConversation = async (conversationId: string) => {
  const confirmed = await modal.showConfirm({
    title: "删除聊天",
    message: "确定要删除这个聊天吗？此操作无法撤销。",
    confirmText: "删除",
    cancelText: "取消",
    confirmColor: "error"
  });
  
  if (confirmed) {
    try {
      await privateChatsService.deletePrivateChat({ id: conversationId });
      messagesStore.removeConversation(conversationId);
    } catch (err) {
      console.error('删除聊天失败:', err);
    }
  }
};

// WebSocket 实时更新
const { on: wsOn } = useGlobalWebSocket();

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
  sent_at: string;
}

let messageCleanup: (() => void) | null = null;

// 生命周期
onMounted(() => {
  loadConversations();
  
  // 监听 ChatMessageNew 事件 - 实时更新会话列表
  messageCleanup = wsOn<ChatMessageNewPayload>('ChatMessageNew', (data) => {
    // 只处理私聊消息（群聊需要订阅）
    if (data.chat_type === 'private_chat') {
      // 忽略自己发送的消息
      if (data.sender.user_id === authStore.user?.user_id) {
        return;
      }
      
      // 更新会话列表中的对应会话
      const existingConversation = conversations.value.find(
        c => c.chat_id === data.chat_id
      );
      
      if (existingConversation) {
        // 更新最后消息和未读数
        messagesStore.updateConversation(data.chat_id, {
          last_message: {
            id: data.id,
            content: data.content,
            type: data.message_type,
            sender_id: data.sender.user_id,
            sent_at: data.sent_at
          },
          last_activity_at: data.sent_at,
          unread_count: existingConversation.unread_count + 1
        });
      } else {
        // 新会话 - 重新加载会话列表
        loadConversations();
      }
      
      if (import.meta.dev) {
        console.log('[WebSocket] 会话列表收到新消息:', data.content);
      }
    }
  });
  
  // 注册快捷键
  defineShortcuts({
    'n': {
      usingInput: false,
      handler: openNewChatModal,
    },
    'r': {
      usingInput: false,
      handler: loadConversations,
    },
  });
});

// 页面离开时清理
onUnmounted(() => {
  if (messageCleanup) {
    messageCleanup();
  }
});

// 监听认证状态变化
watch(
  () => authStore.isAuthenticated,
  (isAuth) => {
    if (isAuth) {
      loadConversations();
    }
  }
);
</script>