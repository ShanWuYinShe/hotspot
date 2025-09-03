<template>
  <div class="min-h-screen bg-gradient-to-br from-neutral-50 to-neutral-100 dark:from-neutral-900 dark:to-neutral-950">
    <UContainer class="py-8">
      <!-- 用户状态卡片 -->
      <div v-if="displayedUser" class="mb-8">
        <UCard>
          <div class="flex items-center space-x-4">
            <UAvatar :alt="displayedUser?.nickname" size="lg">
              {{ displayedUser?.nickname?.charAt(0) || "U" }}
            </UAvatar>
            <div class="flex-1">
              <h3 class="text-xl font-semibold text-neutral-900 dark:text-neutral-100">
                你好，{{ displayedUser?.nickname || "用户" }}！
              </h3>
              <p class="text-neutral-600 dark:text-neutral-400">
                今天想要做什么呢？
              </p>
            </div>
            <div class="flex space-x-3">
              <UButton to="/map" size="lg" class="hidden sm:flex">
                <UIcon name="i-heroicons-map-pin" class="w-5 h-5 mr-2" />
                探索附近
              </UButton>
              <UButton to="/chat" variant="outline" size="lg">
                <UIcon name="i-heroicons-chat-bubble-left-right" class="w-5 h-5 mr-2" />
                <span class="hidden sm:inline">查看消息</span>
                <span class="sm:hidden">消息</span>
              </UButton>
            </div>
          </div>
        </UCard>
      </div>

      <!-- 快速操作区域 -->
      <div class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-4 mb-8">
        <!-- 附近用户 -->
        <UTooltip text="发现附近的用户和朋友 (Alt+U)" :content="{ side: 'bottom' }">
          <UCard class="hover:shadow-lg transition-shadow cursor-pointer" @click="navigateTo('/nearby-users')">
            <div class="flex items-center space-x-3">
              <div class="w-12 h-12 bg-primary-100 dark:bg-primary-900 rounded-lg flex items-center justify-center">
                <UIcon name="i-heroicons-users" class="w-6 h-6 text-primary-600 dark:text-primary-400" />
              </div>
              <div class="flex-1">
                <div class="flex items-center justify-between">
                  <h3 class="font-semibold text-neutral-900 dark:text-neutral-100">附近用户</h3>
                  <UKbd size="xs" class="ml-2">Alt+U</UKbd>
                </div>
                <p class="text-sm text-neutral-600 dark:text-neutral-400">发现周围的朋友</p>
              </div>
            </div>
          </UCard>
        </UTooltip>

        <!-- 我的群组 -->
        <UTooltip text="管理和查看我的群组 (⌘G)" :content="{ side: 'bottom' }">
          <UCard class="hover:shadow-lg transition-shadow cursor-pointer" @click="navigateTo('/groups')">
            <div class="flex items-center space-x-3">
              <div class="w-12 h-12 bg-blue-100 dark:bg-blue-900 rounded-lg flex items-center justify-center">
                <UIcon name="i-heroicons-user-group" class="w-6 h-6 text-blue-600 dark:text-blue-400" />
              </div>
              <div class="flex-1">
                <div class="flex items-center justify-between">
                  <h3 class="font-semibold text-neutral-900 dark:text-neutral-100">我的群组</h3>
                  <UKbd size="xs" class="ml-2">⌘G</UKbd>
                </div>
                <p class="text-sm text-neutral-600 dark:text-neutral-400">管理群组聊天</p>
              </div>
            </div>
          </UCard>
        </UTooltip>

        <!-- 动态分享 -->
        <UTooltip text="分享和查看精彩动态 (Alt+M)" :content="{ side: 'bottom' }">
          <UCard class="hover:shadow-lg transition-shadow cursor-pointer" @click="navigateTo('/moments')">
            <div class="flex items-center space-x-3">
              <div class="w-12 h-12 bg-purple-100 dark:bg-purple-900 rounded-lg flex items-center justify-center">
                <UIcon name="i-heroicons-photo" class="w-6 h-6 text-purple-600 dark:text-purple-400" />
              </div>
              <div class="flex-1">
                <div class="flex items-center justify-between">
                  <h3 class="font-semibold text-neutral-900 dark:text-neutral-100">动态分享</h3>
                  <UKbd size="xs" class="ml-2">Alt+M</UKbd>
                </div>
                <p class="text-sm text-neutral-600 dark:text-neutral-400">分享精彩瞬间</p>
              </div>
            </div>
          </UCard>
        </UTooltip>

        <!-- 地图探索 -->
        <UTooltip text="探索周边热点和地点 (⌘M)" :content="{ side: 'bottom' }">
          <UCard class="hover:shadow-lg transition-shadow cursor-pointer" @click="navigateTo('/map')">
            <div class="flex items-center space-x-3">
              <div class="w-12 h-12 bg-green-100 dark:bg-green-900 rounded-lg flex items-center justify-center">
                <UIcon name="i-heroicons-map" class="w-6 h-6 text-green-600 dark:text-green-400" />
              </div>
              <div class="flex-1">
                <div class="flex items-center justify-between">
                  <h3 class="font-semibold text-neutral-900 dark:text-neutral-100">地图探索</h3>
                  <UKbd size="xs" class="ml-2">⌘M</UKbd>
                </div>
                <p class="text-sm text-neutral-600 dark:text-neutral-400">查看周边热点</p>
              </div>
            </div>
          </UCard>
        </UTooltip>
      </div>

      <!-- 最近活动 -->
      <div class="mb-8">
        <div class="flex items-center justify-between mb-4">
          <h2 class="text-xl font-semibold text-neutral-900 dark:text-neutral-100">最近活动</h2>
          <UTooltip text="刷新活动 (⌘R)">
            <UButton variant="ghost" size="sm" @click="refreshActivities" class="flex items-center gap-2">
              <UIcon name="i-heroicons-arrow-path" class="w-4 h-4" :class="{ 'animate-spin': refreshing }" />
              <UKbd size="xs">⌘R</UKbd>
            </UButton>
          </UTooltip>
        </div>
        
        <!-- 加载状态 -->
        <LoadingState v-if="activitiesLoading" 
          title="加载中..." 
          description="正在获取最新活动"
        />
        
        <!-- 错误状态 -->
        <ErrorState v-else-if="activitiesError"
          :title="activitiesError"
          description="请稍后重试"
          @retry="fetchActivities"
        />
        
        <!-- 活动时间线 -->
        <UTimeline v-else-if="activities.length > 0" :items="timelineItems" />
        
        <!-- 空状态 -->
        <EmptyState v-else 
          title="暂无活动" 
          description="开始探索，创建你的第一个活动吧！"
          icon="i-heroicons-sparkles"
        />
      </div>

      <!-- 未登录用户提示 - 现在通过 Modal 显示 -->
    </UContainer>

    <!-- Modal 组件现在通过 useOverlay 管理 -->
  </div>
</template>

<script setup lang="ts">
// 首页元数据
definePageMeta({
  title: "首页",
});

// 导入组件
const WelcomeModal = defineAsyncComponent(() => import('~/components/WelcomeModal.vue'));
const TempUserModal = defineAsyncComponent(() => import('~/components/TempUserModal.vue'));
const LoginPromptModal = defineAsyncComponent(() => import('~/components/LoginPromptModal.vue'));

const authStore = useAuthStore();
const overlay = useOverlay();

// 创建 Modal 实例
const welcomeModal = overlay.create(WelcomeModal);
const tempUserModal = overlay.create(TempUserModal);
const loginPromptModal = overlay.create(LoginPromptModal);

// 认证状态
const displayedUser = ref<any | null>(null);
onMounted(async () => { 
  displayedUser.value = authStore.user || null;
  
  // 设置 Modal 事件处理
  handleWelcomeModalEvents();
  handleLoginPromptEvents();
  
  // 首次访问显示欢迎弹窗
  const hasVisited = localStorage.getItem('has-visited-hotspot');
  if (!hasVisited && !displayedUser.value) {
    // 延迟显示，确保页面加载完成
    setTimeout(() => {
      welcomeModal.open();
    }, 500);
    localStorage.setItem('has-visited-hotspot', 'true');
  } else if (!displayedUser.value) {
    // 非首次访问的未登录用户，显示登录提示
    const neverShow = localStorage.getItem('never-show-login-prompt');
    if (!neverShow) {
      setTimeout(() => {
        loginPromptModal.open();
      }, 2000); // 2秒后显示，给用户时间浏览页面
    }
  }
  
  // 获取活动数据
  await fetchActivities();
});

// 活动数据类型定义
interface Activity {
  id: string | number;
  user: string;
  content: string;
  time: string;
  icon: string;
}

// 活动数据
const refreshing = ref(false);
const activities = ref<Activity[]>([]);
const activitiesLoading = ref(false);
const activitiesError = ref<string | null>(null);

// 时间线数据转换
const timelineItems = computed(() => {
  return activities.value.map(activity => ({
    title: activity.content,
    description: activity.time,
    icon: activity.icon,
    avatar: {
      alt: activity.user,
      fallback: activity.user?.charAt(0) || 'U'
    },
    color: 'primary'
  }));
});

// 获取活动数据
const fetchActivities = async () => {
  try {
    activitiesLoading.value = true;
    activitiesError.value = null;
    
    // TODO: 实现真实的活动API调用
    // const response = await $fetch('/api/activities/recent');
    // activities.value = response.data;
    
    // 暂时清空，等待后端API实现
    activities.value = [];
  } catch (error) {
    console.error('获取活动数据失败:', error);
    activitiesError.value = '获取活动数据失败';
    activities.value = [];
  } finally {
    activitiesLoading.value = false;
  }
};

// 刷新活动
const refreshActivities = async () => {
  refreshing.value = true;
  await fetchActivities();
  refreshing.value = false;
};

// Modal 事件处理
const handleWelcomeModalEvents = () => {
  // 监听欢迎弹窗事件
  welcomeModal.patch({
    'onTemp-experience': () => {
      tempUserModal.open();
    },
    onRegister: () => {
      navigateTo('/auth');
    },
    onLogin: () => {
      navigateTo('/auth');
    },
    onSkip: () => {
      // 用户选择跳过，什么都不做
    }
  });
};

const handleLoginPromptEvents = () => {
  // 监听登录提示弹窗事件
  loginPromptModal.patch({
    'onTemp-experience': () => {
      tempUserModal.open();
    },
    onLogin: () => {
      navigateTo('/auth');
    },
    onSkip: () => {
      // 用户选择跳过，什么都不做
    },
    'onNever-show': () => {
      // 用户选择不再显示，已在组件内处理 localStorage
    }
  });
};

// 快捷键管理
const { registerShortcut } = useGlobalShortcuts()

onMounted(() => {
  // 注册首页快捷键
  registerShortcut({
    key: 'meta_h',
    description: '返回首页',
    handler: () => navigateTo('/'),
    category: '导航',
    visible: true
  })

  registerShortcut({
    key: 'meta_m',
    description: '地图探索',
    handler: () => navigateTo('/map'),
    category: '功能',
    visible: true
  })

  registerShortcut({
    key: 'meta_g',
    description: '我的群组',
    handler: () => navigateTo('/groups'),
    category: '功能',
    visible: true
  })

  registerShortcut({
    key: 'meta_c',
    description: '消息聊天',
    handler: () => navigateTo('/chat'),
    category: '功能',
    visible: true
  })

  registerShortcut({
    key: 'alt_u',
    description: '附近用户',
    handler: () => navigateTo('/nearby-users'),
    category: '功能',
    visible: true
  })

  registerShortcut({
    key: 'alt_m',
    description: '动态分享',
    handler: () => navigateTo('/moments'),
    category: '功能',
    visible: true
  })

  registerShortcut({
    key: 'meta_r',
    description: '刷新活动',
    handler: refreshActivities,
    category: '操作',
    visible: true
  })

  // 设置快捷键监听
  const dispose = defineShortcuts({
    meta_h: {
      usingInput: true,
      handler: () => navigateTo('/'),
    },
    meta_m: {
      usingInput: true,
      handler: () => navigateTo('/map'),
    },
    meta_g: {
      usingInput: true,
      handler: () => navigateTo('/groups'),
    },
    meta_c: {
      usingInput: true,
      handler: () => navigateTo('/chat'),
    },
    alt_u: {
      usingInput: true,
      handler: () => navigateTo('/nearby-users'),
    },
    alt_m: {
      usingInput: true,
      handler: () => navigateTo('/moments'),
    },
    meta_r: {
      usingInput: true,
      handler: refreshActivities,
    },
  })

  onUnmounted(() => {
    try {
      dispose()
    } catch {
      /* noop */
    }
  })
})
</script>