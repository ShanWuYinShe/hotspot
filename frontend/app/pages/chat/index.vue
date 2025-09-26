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
              <p class="text-sm text-neutral-500">与朋友和群组保持联系</p>
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
            @click="navigateTo('/users/search')"
          >
            <template #icon>
              <div class="w-10 h-10 bg-blue-500 rounded-xl flex items-center justify-center">
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
                <UIcon name="i-heroicons-chat-bubble-left-ellipsis" class="w-8 h-8 text-neutral-500" />
              </div>
              <h3 class="text-lg font-semibold text-neutral-900 dark:text-white mb-2">暂无会话</h3>
              <p class="text-neutral-600 dark:text-neutral-400 mb-4">开始与朋友聊天吧！</p>
              <UButton @click="navigateTo('/users/search')" icon="i-heroicons-user-plus">
                发现新朋友
              </UButton>
            </div>

            <!-- 会话列表 -->
            <div v-else class="space-y-1">
              <div
                v-for="conversation in filteredConversations"
                :key="conversation.id"
                class="flex items-center gap-3 p-3 rounded-lg hover:bg-neutral-50 dark:hover:bg-neutral-800 cursor-pointer transition-colors"
                @click="openConversation(conversation)"
              >
                <!-- 头像 -->
                <div class="relative flex-shrink-0">
                  <UAvatar
                    :alt="conversation.name"
                    size="lg"
                    :src="conversation.avatar"
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
                    v-if="conversation.type === 'group'"
                    class="absolute -bottom-1 -right-1 w-4 h-4 bg-blue-500 rounded-full border-2 border-white dark:border-neutral-900 flex items-center justify-center"
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
                    <span class="text-xs text-neutral-500 flex-shrink-0">
                      {{ formatTime(conversation.last_message_time) }}
                    </span>
                  </div>
                  
                  <div class="flex items-center justify-between">
                    <p class="text-sm text-neutral-600 dark:text-neutral-400 truncate">
                      <span v-if="(conversation as any).last_message_sender && conversation.type === 'group'" class="text-neutral-500">
                        {{ (conversation as any).last_message_sender }}:
                      </span>
                      {{ conversation.last_message || '暂无消息' }}
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
// 从数据源获取 SEO 配置
const seoConfig = getPageSeoConfig('chat')

definePageMeta({
  title: seoConfig.title,
  description: seoConfig.description,
  layout: "dashboard"
});

// 设置 SEO meta
useSeoMeta(buildSeoMeta('chat'));

// 使用现有的 stores 和 composables
const messagesStore = useMessagesStore();
const authStore = useAuthStore();
const { openConfirmModal } = useModals();

// 响应式状态
const loading = ref(false);
const error = ref<string | null>(null);
const showFilters = ref(false);
const activeFilter = ref<string>('all');

// 计算属性
const totalUnread = computed(() => messagesStore.totalUnreadCount);
const conversations = computed(() => messagesStore.conversations || []);

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
      filtered = filtered.filter(c => c.type === 'user');
      break;
    case 'group':
      filtered = filtered.filter(c => c.type === 'group');
      break;
    case 'online':
      filtered = filtered.filter(c => (c as any).is_online);
      break;
  }
  
  // 按最后消息时间排序
  return filtered.sort((a, b) => 
    new Date(b.last_message_time).getTime() - new Date(a.last_message_time).getTime()
  );
});

// 方法
const loadConversations = async () => {
  try {
    loading.value = true;
    error.value = null;
    // await messagesStore.loadConversations(); // 需要实现此方法
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
  const route = conversation.type === 'group' 
    ? `/chat/groups/${conversation.id}`
    : `/chat/users/${conversation.id}`;
  navigateTo(route);
};

const openNewChatModal = () => {
  // 这里可以打开新建聊天的模态框
  navigateTo('/users/search');
};

const formatTime = (timestamp: string) => {
  const date = new Date(timestamp);
  const now = new Date();
  const diff = now.getTime() - date.getTime();
  
  if (diff < 60000) return '刚刚';
  if (diff < 3600000) return `${Math.floor(diff / 60000)}分钟前`;
  if (diff < 86400000) return `${Math.floor(diff / 3600000)}小时前`;
  if (diff < 604800000) return `${Math.floor(diff / 86400000)}天前`;
  
  return date.toLocaleDateString();
};

const getConversationMenuItems = (conversation: any) => {
  const items = [
    [{
      label: '标记为已读',
      icon: 'i-heroicons-check',
      click: () => markAsRead(conversation.id),
      disabled: conversation.unread_count === 0
    }],
    [{
      label: '置顶聊天',
      icon: 'i-heroicons-bookmark',
      click: () => togglePin(conversation.id)
    }, {
      label: '免打扰',
      icon: 'i-heroicons-bell-slash',
      click: () => toggleMute(conversation.id)
    }],
    [{
      label: '删除聊天',
      icon: 'i-heroicons-trash',
      click: () => deleteConversation(conversation.id),
      class: 'text-error-500'
    }]
  ];
  
  return items;
};

const markAsRead = async (conversationId: string) => {
  try {
    await messagesStore.markConversationAsRead(conversationId);
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

const deleteConversation = (conversationId: string) => {
  openConfirmModal({
    title: "删除聊天",
    message: "确定要删除这个聊天吗？此操作无法撤销。",
    confirmText: "删除",
    cancelText: "取消",
    confirmColor: "error",
    onConfirm: async () => {
      try {
        await messagesStore.deleteConversation(conversationId);
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
  const shortcuts = useShortcuts();
  shortcuts.register('meta+n', openNewChatModal, '新建聊天', '消息');
  shortcuts.register('meta+r', loadConversations, '刷新会话', '消息');
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