<template>
  <div class="w-full">

    <div class="grid grid-cols-1 md:grid-cols-2 gap-3 mb-4">
      <UPageCard
        to="/search"
        title="发现新朋友"
        description="搜索并添加新的聊天伙伴"
        icon="i-lucide-user-plus"
        orientation="horizontal"
        variant="outline"
        spotlight
      />

      <UPageCard
        to="/groups/nearby"
        title="加入群组"
        description="发现并加入感兴趣的群组"
        icon="i-lucide-users-round"
        orientation="horizontal"
        variant="outline"
        spotlight
      />
    </div>

    <UCard variant="subtle">
      <template #header>
        <div class="flex items-center justify-between gap-3">
          <span class="text-lg font-semibold">最近会话</span>
          <UButton
            variant="ghost"
            color="neutral"
            size="sm"
            icon="i-lucide-filter"
            @click="toggleFilters"
          >
            筛选
          </UButton>
        </div>
      </template>

      <div>
        <UCard
          v-if="showFilters"
          variant="soft"
          class="mb-3"
        >
          <div class="flex flex-wrap gap-2">
            <UButton
              v-for="filter in conversationFilters"
              :key="filter.key"
              :variant="activeFilter === filter.key ? 'solid' : 'ghost'"
              :color="activeFilter === filter.key ? 'primary' : 'neutral'"
              size="sm"
              @click="setFilter(filter.key)"
            >
              {{ filter.label }}
            </UButton>
          </div>
        </UCard>

        <div v-if="loading" class="space-y-3">
          <div v-for="i in 5" :key="i" class="flex items-center gap-3 p-2">
            <UAvatar size="md" icon="i-lucide-user" color="neutral" />
            <div class="flex-1 space-y-2">
              <USkeleton class="h-4 w-3/4" />
              <USkeleton class="h-3 w-1/2" />
            </div>
            <USkeleton class="h-3 w-12" />
          </div>
        </div>

        <UEmpty
          v-else-if="filteredConversations.length === 0"
          icon="i-lucide-messages-square"
          title="暂无会话"
          description="开始与朋友聊天吧！"
          class="py-12"
        >
          <template #actions>
            <UButton @click="navigateTo('/search')" icon="i-lucide-user-plus" color="primary">
              发现新朋友
            </UButton>
          </template>
        </UEmpty>

        <div v-else class="space-y-1">
          <ConversationItem
            v-for="conversation in filteredConversations"
            :key="conversation.chat_id"
            :conversation="toConversationItem(conversation)"
            :is-online="conversation.is_online"
            @open="openConversation(conversation)"
            @pin="togglePin(conversation.chat_id)"
            @mute="toggleMute(conversation.chat_id)"
            @delete="deleteConversation(conversation.chat_id)"
          />
        </div>
      </div>
    </UCard>
  </div>
</template>

<script setup lang="ts">
import { getPageSeoConfig, buildSeoMeta } from '~/config/seo';
import ConversationItem from '~/components/chat/ConversationItem.vue';
import { useDashboardHeader } from '~/composables/useDashboardHeader'

// 从数据源获取 SEO 配置
const seoConfig = getPageSeoConfig('chat')

// 需要登录 - 聊天功能需要认证
definePageMeta({
  title: seoConfig.title,
  description: seoConfig.description,
  requiresAuth: true,
  layout: "dashboard",
  dashboardHeader: true
});

// 设置 SEO meta
useSeoMeta(buildSeoMeta('chat'));

import { type Conversation } from '~/stores/messages.store'
import { usePrivateChatsService } from '~/services/private_chats.service'
import { usePrivateMessagesService } from '~/services/private_messages.service'

// 使用水合安全的 store 访问
const authStore = useAuthStore();
const messagesStore = useMessagesStore();
const privateChatsService = usePrivateChatsService();
const privateMessagesService = usePrivateMessagesService();
const modal = useModal();
const dashboardHeader = useDashboardHeader()
const { formatTime } = useFormatters();
const { withErrorHandling } = useErrorHandler();

// 响应式状态
const loading = ref(false);
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

// 转换函数
const toConversationItem = (c: Conversation) => ({
  group_id: c.chat_id,
  name: c.name,
  avatar: c.avatar_url,
  type: (c.chat_type === 'private' ? 'user' : 'group') as 'user' | 'group',
  members_count: 0, // Store doesn't have this yet, maybe need to fetch or ignore
  last_message: c.last_message || undefined,
  last_message_time: c.last_activity_at,
  unread_count: c.unread_count,
  is_pinned: false,
  is_muted: false,
  updated_at: c.last_activity_at
});

// 方法
const loadConversations = async () => {
  await withErrorHandling(async () => {
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
      // 在线状态由 WebSocket user 订阅驱动（store 内有缓存，会自动合并）
      is_online: undefined
    }));
    messagesStore.setConversations(conversationList);
  }, {
    title: '加载会话失败',
    loading: loading
  });
};

const toggleFilters = () => {
  showFilters.value = !showFilters.value;
};

const setFilter = (filterKey: string) => {
  activeFilter.value = filterKey;
};

const openConversation = (conversation: Conversation) => {
  // 切换到会话：本地先置为当前会话（避免会话列表/徽标继续显示未读）
  messagesStore.setCurrentConversation(conversation.chat_id)
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

// 把 PageLayout 的 header-actions 移到 dashboard 内置 header（避免每页重复包 PageLayout）
watchEffect(() => {
  dashboardHeader.setActions([
    {
      key: 'refresh',
      label: '刷新',
      icon: 'i-lucide-refresh-cw',
      variant: 'outline',
      loading: loading.value,
      action: () => loadConversations()
    },
    {
      key: 'new_chat',
      label: '新建聊天',
      icon: 'i-lucide-user-plus',
      variant: 'outline',
      action: () => openNewChatModal()
    }
  ])
})

const markAsRead = async (conversationId: string) => {
  await withErrorHandling(async () => {
    await privateMessagesService.markPrivateMessagesAsRead({ id: conversationId });
    messagesStore.updateUnreadCount(conversationId, 0);
  }, { title: '标记已读失败', showToast: false }); // 标记已读失败通常不需要打扰用户
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
    await withErrorHandling(async () => {
      await privateChatsService.deletePrivateChat({ id: conversationId });
      messagesStore.removeConversation(conversationId);
    }, { title: '删除聊天失败' });
  }
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

// 页面离开时清理
// useWebSocket 自动管理清理，无需手动 onUnmounted

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