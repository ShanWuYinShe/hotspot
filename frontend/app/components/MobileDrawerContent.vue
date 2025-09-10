<template>
  <!-- 移动端导航抽屉 - 使用 UDrawer -->
  <UDrawer side="left" :overlay="true">
    <template #header>
      <div class="flex items-center justify-between">
        <div class="flex items-center gap-3">
          <div class="w-8 h-8 rounded-lg bg-gradient-to-br from-orange-500 to-red-600 flex items-center justify-center">
            <UIcon name="i-heroicons-fire" class="w-5 h-5 text-white" />
          </div>
          <span class="text-lg font-bold bg-gradient-to-r from-orange-600 to-red-600 bg-clip-text text-transparent">
            HotSpot
          </span>
        </div>
        <UButton variant="ghost" icon="i-heroicons-x-mark" size="sm" square @click="handleClose" />
      </div>
    </template>

    <template #body>
      <div class="space-y-6 p-4">
        <!-- 用户信息卡片 -->
        <UCard v-if="authStore.user" class="bg-gradient-to-r from-primary-50 to-blue-50 dark:from-primary-900/20 dark:to-blue-900/20">
          <div class="flex items-center gap-3">
            <UAvatar 
              :alt="authStore.user.nickname" 
              size="lg"
              class="ring-2 ring-white dark:ring-neutral-800"
            >
              {{ authStore.user.nickname?.charAt(0) || 'U' }}
            </UAvatar>
            <div class="flex-1">
              <p class="font-semibold text-neutral-900 dark:text-white">
                {{ authStore.user.nickname }}
              </p>
              <p class="text-sm text-neutral-600 dark:text-neutral-400">
                {{ authStore.user.email || '临时用户' }}
              </p>
              <UBadge 
                :color="authStore.user.is_temp ? 'warning' : 'success'" 
                size="xs" 
                class="mt-1"
              >
                {{ authStore.user.is_temp ? '临时账户' : '正式用户' }}
              </UBadge>
            </div>
          </div>
        </UCard>

        <!-- 未登录状态 -->
        <UCard v-else class="bg-gradient-to-r from-neutral-50 to-neutral-100 dark:from-neutral-800 dark:to-neutral-900">
          <div class="text-center py-4">
            <UIcon name="i-heroicons-user-circle" class="w-12 h-12 mx-auto text-neutral-400 mb-2" />
            <p class="text-neutral-600 dark:text-neutral-400 mb-3">未登录</p>
            <UButton to="/auth" size="sm" @click="handleClose">
              立即登录
            </UButton>
          </div>
        </UCard>

        <!-- 主导航菜单 -->
        <div class="space-y-2">
          <h3 class="text-sm font-semibold text-neutral-500 dark:text-neutral-400 uppercase tracking-wide px-2">
            主要功能
          </h3>
          
          <UButton 
            v-for="item in mainNavItems" 
            :key="item.to"
            :to="item.to"
            variant="ghost" 
            class="w-full justify-start"
            @click="item.to ? handleNavigation(item.to) : undefined"
          >
            <UIcon :name="item.icon" class="w-5 h-5 mr-3" />
            {{ item.label }}
            <UBadge v-if="item.badge" :color="item.badgeColor || 'primary'" size="xs" class="ml-auto">
              {{ item.badge }}
            </UBadge>
          </UButton>
        </div>

        <!-- 次要功能 -->
        <div class="space-y-2">
          <h3 class="text-sm font-semibold text-neutral-500 dark:text-neutral-400 uppercase tracking-wide px-2">
            更多功能
          </h3>
          
          <UButton 
            v-for="item in secondaryNavItems" 
            :key="item.to || item.action"
            :to="item.to"
            variant="ghost" 
            class="w-full justify-start"
            @click="handleItemClick(item)"
          >
            <UIcon :name="item.icon" class="w-5 h-5 mr-3" />
            {{ item.label }}
          </UButton>
        </div>

        <!-- 底部操作 -->
        <div class="border-t pt-4 space-y-2">
          <UButton 
            v-if="authStore.user"
            variant="ghost" 
            color="error"
            class="w-full justify-start"
            @click="handleLogout"
          >
            <UIcon name="i-heroicons-arrow-right-end-on-rectangle" class="w-5 h-5 mr-3" />
            退出登录
          </UButton>
        </div>
      </div>
    </template>
  </UDrawer>
</template>

<script setup lang="ts">
// Props
interface Props {
  open?: boolean;
}

const props = withDefaults(defineProps<Props>(), {
  open: false
});

// Emits
const emit = defineEmits<{
  'update:open': [value: boolean];
  'close': [];
}>();

// useOverlay 会自动管理 Drawer 状态，无需手动绑定 isOpen

const authStore = useAuthStore();

// 导航项类型定义
interface NavItem {
  label: string;
  icon: string;
  to?: string;
  action?: string;
  badge?: string;
  badgeColor?: 'primary' | 'secondary' | 'success' | 'info' | 'warning' | 'error' | 'neutral';
}

// 主导航项
const mainNavItems = computed((): NavItem[] => [
  {
    label: '首页',
    icon: 'i-heroicons-home',
    to: '/',
  },
  {
    label: '附近用户',
    icon: 'i-heroicons-users',
    to: '/users/nearby',
  },
  {
    label: '我的群组',
    icon: 'i-heroicons-user-group',
    to: '/groups',
  },
  {
    label: '消息',
    icon: 'i-heroicons-chat-bubble-left-right',
    to: '/chat',
    badge: '3', // 示例徽章
    badgeColor: 'primary'
  },
  {
    label: '动态',
    icon: 'i-heroicons-photo',
    to: '/moments',
  },
  {
    label: '地图',
    icon: 'i-heroicons-map',
    to: '/map',
  },
]);

// 次要导航项
const secondaryNavItems = computed((): NavItem[] => [
  {
    label: '好友',
    icon: 'i-heroicons-heart',
    to: '/friends',
  },
  {
    label: '设置',
    icon: 'i-heroicons-cog-6-tooth',
    to: '/settings',
  },
  {
    label: '帮助',
    icon: 'i-heroicons-question-mark-circle',
    action: 'help'
  },
  {
    label: '关于',
    icon: 'i-heroicons-information-circle',
    action: 'about'
  },
]);

// 处理关闭
const handleClose = () => {
  emit('close');
};

// 处理导航
const handleNavigation = (to: string) => {
  handleClose();
  navigateTo(to);
};

// 处理项目点击
const handleItemClick = (item: any) => {
  if (item.to) {
    handleNavigation(item.to);
  } else if (item.action) {
    handleAction(item.action);
  }
};

// 处理操作
const handleAction = (action: string) => {
  const toast = useSmartToast();
  
  switch (action) {
    case 'help':
      toast.add({
        title: '帮助中心',
        description: '帮助文档正在建设中',
        color: 'info'
      });
      break;
    case 'about':
      toast.add({
        title: 'HotSpot v1.0',
        description: '基于位置的社交平台',
        color: 'info'
      });
      break;
  }
  
  handleClose();
};

// 处理退出登录
const handleLogout = async () => {
  const toast = useSmartToast();
  
  try {
    await authStore.logout();
    toast.add({
      title: '已退出登录',
      color: 'success'
    });
    handleClose();
    navigateTo('/');
  } catch (error) {
    toast.add({
      title: '退出失败',
      description: '请稍后重试',
      color: 'error'
    });
  }
};
</script>
