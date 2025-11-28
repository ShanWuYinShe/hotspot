<template>
  <UDashboardPanel>
    <!-- 顶部工具栏 -->
    <template #header>
      <UDashboardToolbar>
        <template #left>
          <div class="flex items-center gap-3">
            <div class="w-8 h-8 bg-gradient-to-br from-blue-500 to-purple-600 rounded-lg flex items-center justify-center shadow-lg">
              <UIcon name="i-heroicons-chat-bubble-left-right" class="w-5 h-5 text-white" />
            </div>
            <div>
              <h1 class="text-lg font-semibold text-neutral-900 dark:text-white">消息中心</h1>
              <p class="text-sm text-neutral-500 dark:text-neutral-400">与朋友和群组保持联系</p>
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
            class="cursor-pointer hover:shadow-md transition-shadow"
            @click="navigateTo('/search')"
          >
            <template #icon>
              <div class="w-10 h-10 bg-primary-500 rounded-xl flex items-center justify-center">
                <UIcon name="i-heroicons-user-plus" class="w-5 h-5 text-white" />
              </div>
            </template>
          </UPageCard>

          <UPageCard
            title="加入群组"
            description="发现并加入感兴趣的群组"
            icon="i-heroicons-user-group"
            class="cursor-pointer hover:shadow-md transition-shadow"
            @click="navigateTo('/groups/discover')"
          >
            <template #icon>
              <div class="w-10 h-10 bg-purple-500 rounded-xl flex items-center justify-center">
                <UIcon name="i-heroicons-user-group" class="w-5 h-5 text-white" />
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
            <div v-if="showFilters" class="mb-4 p-4 bg-neutral-50 dark:bg-neutral-800 rounded-lg">
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
            <div v-else-if="error" class="text-center py-12">
              <UIcon name="i-heroicons-exclamation-triangle" class="w-12 h-12 text-error-500 mx-auto mb-4" />
              <h3 class="text-lg font-semibold text-neutral-900 dark:text-white mb-2">加载失败</h3>
              <p class="text-neutral-600 dark:text-neutral-400 mb-4">{{ error }}</p>
              <UButton @click="loadConversations" variant="outline">重试</UButton>
            </div>

            <!-- 空状态 -->
            <div v-else-if="filteredConversations.length === 0" class="text-center py-12">
              <div class="w-16 h-16 bg-neutral-100 dark:bg-neutral-800 rounded-2xl flex items-center justify-center mx-auto mb-4">
                <UIcon name="i-heroicons-chat-bubble-left-ellipsis" class="w-8 h-8 text-neutral-500 dark:text-neutral-400" />
              </div>
              <h3 class="text-lg font-semibold text-neutral-900 dark:text-white mb-2">暂无会话</h3>
              <p class="text-neutral-600 dark:text-neutral-400 mb-4">开始与朋友聊天吧！</p>
              <UButton @click="navigateTo('/search')" icon="i-heroicons-user-plus">
                发现新朋友
              </UButton>
            </div>

            <!-- 会话列表 -->
            <div v-else class="space-y-1">
              <div
                v-for="conversation in filteredConversations"
                :key="conversation.chat_id"
                class="flex items-center gap-3 p-3 rounded-lg hover:bg-neutral-50 dark:hover:bg-neutral-800 cursor-pointer transition-colors"
                @click="openConversation(conversation)"
              >
                <!-- 头像 -->
                <div class="relative flex-shrink-0">
                  <UAvatar
                    :alt="conversation.name"
                    size="lg"
                    :src="conversation.avatar_url"
                  >
                    {{ conversation.name?.charAt(0) || "?" }}
                  </UAvatar>
                  
                  <!-- 在线状态 -->
                  <div
                    v-if="(conversation as any).is_online"
                    class="absolute -bottom-1 -right-1 w-4 h-4 bg-green-500 rounded-full border-2 border-white dark:border-neutral-900"
                  />
                  
                  <!-- 群组标识 -->
                  <div
                    v-if="conversation.chat_type === 'group'"
                    class="absolute -bottom-1 -right-1 w-4 h-4 bg-primary-500 rounded-full border-2 border-white dark:border-neutral-900 flex items-center justify-center"
                  >
                    <UIcon name="i-heroicons-user-group" class="w-2 h-2 text-white" />
                  </div>
                </div>

                <!-- 会话信息 -->
                <div class="flex-1 min-w-0">
                  <div class="flex items-center justify-between mb-1">
                    <h3 class="font-medium text-neutral-900 dark:text-white truncate">
                      {{ conversation.name }}
                    </h3>
                    <span class="text-xs text-neutral-500 dark:text-neutral-400 flex-shrink-0">
                      {{ formatTime(conversation.last_activity_at) }}
                    </span>
                  </div>
                  
                  <div class="flex items-center justify-between">
                    <p class="text-sm text-neutral-600 dark:text-neutral-400 truncate">
                      <span v-if="(conversation as any).last_message_sender && conversation.chat_type === 'group'" class="text-neutral-500 dark:text-neutral-400">
                        {{ (conversation as any).last_message_sender }}:
                      </span>
                      {{ conversation.last_message?.content || '暂无消息' }}
                    </p>
                    
                    <!-- 未读消息数 -->
                    <UBadge
                      v-if="conversation.unread_count > 0"
                      :label="conversation.unread_count > 99 ? '99+' : conversation.unread_count.toString()"
                      color="primary"
                      size="xs"
                      class="flex-shrink-0"
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

// 导入服务
import { useMessagesService } from '~/services/messages.service';

// 使用现有的 stores 和 composables
const messagesStore = useMessagesStore();
const authStore = useAuthStore();
const messageService = useMessagesService();
const modal = useModal();
const { formatTime } = useFormatters();

// 响应式状态
const loading = ref(false);
const error = ref<string | null>(null);
const showFilters = ref(false);
const activeFilter = ref<string>('all');

// 计算属性
const totalUnread = computed(() => messagesStore.totalUnreadCount);
const conversations = computed(() => messagesStore.conversationsList);

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
      filtered = filtered.filter(c => (c as any).is_online);
      break;
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
    const response = await messageService.getPrivateChats({});
    // 转换为通用会话格式
    const conversationsList = response.items.map(item => ({
      chat_id: item.private_chat_id,
      chat_type: 'private' as const,
      name: item.other_user.nickname,
      avatar_url: item.other_user.avatar_url,
      unread_count: item.unread_count,
      last_message: item.last_message,
      last_activity_at: item.last_activity_at
    }));
    messagesStore.setConversations(conversationsList);
  } catch (err: any) {
    console.error('加载会话失败:', err);
    error.value = err.message || '加载会话失败';
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

const openConversation = (conversation: any) => {
  const route = conversation.chat_type === 'group' 
    ? `/chat/groups/${conversation.chat_id}`
    : `/chat/users/${conversation.chat_id}`;
  navigateTo(route);
};

const openNewChatModal = () => {
  // 这里可以打开新建聊天的模态框
  navigateTo('/search');
};

// formatTime 函数已从统一工具函数导入

const getConversationMenuItems = (conversation: any) => {
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
      class: 'text-error-500'
    }]
  ];
  
  return items;
};

const markAsRead = async (chatId: string) => {
  try {
    const conversation = messagesStore.getConversation(chatId);
    if (conversation?.chat_type === 'group') {
      await messagesStore.markGroupMessagesAsRead(chatId);
    } else {
      await messagesStore.markPrivateMessagesAsRead(chatId);
    }
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

const deleteConversation = (chatId: string) => {
  modal.showConfirm({
    title: "删除聊天",
    message: "确定要删除这个聊天吗？此操作无法撤销。",
    confirmText: "删除",
    cancelText: "取消",
    confirmColor: "error"
  }).then(async (confirmed) => {
    if (confirmed) {
      try {
        messagesStore.removeConversation(chatId);
      } catch (err) {
        console.error('删除聊天失败:', err);
      }
    }
  });
};

// 生命周期
onMounted(() => {
  loadConversations();
  
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