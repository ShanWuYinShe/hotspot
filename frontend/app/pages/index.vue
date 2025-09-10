<template>
  <div class="min-h-screen bg-gradient-to-br from-neutral-50 to-neutral-100 dark:from-neutral-900 dark:to-neutral-950">
    <div class="mx-auto max-w-7xl px-4 sm:px-6 lg:px-8 py-8 sm:py-10">
      <!-- 用户状态卡片 -->
      <div v-if="displayedUser" class="mb-6 sm:mb-8">
        <UCard>
          <div class="flex items-center space-x-3 sm:space-x-4">
            <UAvatar :alt="displayedUser?.nickname" size="md" class="sm:w-12 sm:h-12">
              {{ displayedUser?.nickname?.charAt(0) || "U" }}
            </UAvatar>
            <div class="flex-1 min-w-0">
              <h3 class="text-lg sm:text-xl font-semibold text-neutral-900 dark:text-neutral-100 truncate">
                你好，{{ displayedUser?.nickname || "用户" }}！
              </h3>
              <p class="text-sm sm:text-base text-neutral-600 dark:text-neutral-400 truncate">
                今天想要做什么呢？
              </p>
            </div>
            <div class="flex space-x-2 sm:space-x-3 flex-shrink-0">
              <UButton to="/map" size="md" class="hidden sm:flex sm:size-lg">
                <UIcon name="i-heroicons-map-pin" class="w-4 h-4 sm:w-5 sm:h-5 mr-1 sm:mr-2" />
                探索附近
              </UButton>
              <UButton to="/chat" variant="outline" size="md" class="touch-manipulation sm:size-lg">
                <UIcon name="i-heroicons-chat-bubble-left-right" class="w-4 h-4 sm:w-5 sm:h-5 mr-1 sm:mr-2" />
                <span class="hidden sm:inline">查看消息</span>
                <span class="sm:hidden">消息</span>
              </UButton>
            </div>
          </div>
        </UCard>
      </div>

      <!-- 快速操作区域 -->
      <div class="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-3 sm:gap-4 mb-8">
        <!-- 附近用户 -->
        <UCard 
          class="hover:shadow-lg active:scale-95 transition-all duration-200 cursor-pointer touch-manipulation" 
          @click="navigateTo('/users/nearby')"
        >
          <div class="flex items-center space-x-3 p-1">
            <div class="w-12 h-12 bg-primary-100 dark:bg-primary-900 rounded-lg flex items-center justify-center flex-shrink-0">
              <UIcon name="i-heroicons-users" class="w-6 h-6 text-primary-600 dark:text-primary-400" />
            </div>
            <div class="flex-1 min-w-0">
              <h3 class="font-semibold text-neutral-900 dark:text-neutral-100 truncate">附近用户</h3>
              <p class="text-sm text-neutral-600 dark:text-neutral-400 truncate">发现周围的朋友</p>
            </div>
          </div>
        </UCard>

        <!-- 我的群组 -->
        <UCard 
          class="hover:shadow-lg active:scale-95 transition-all duration-200 cursor-pointer touch-manipulation" 
          @click="navigateTo('/groups')"
        >
          <div class="flex items-center space-x-3 p-1">
            <div class="w-12 h-12 bg-group-100 dark:bg-group-900 rounded-lg flex items-center justify-center flex-shrink-0">
              <UIcon name="i-heroicons-user-group" class="w-6 h-6 text-group-600 dark:text-group-400" />
            </div>
            <div class="flex-1 min-w-0">
              <h3 class="font-semibold text-neutral-900 dark:text-neutral-100 truncate">我的群组</h3>
              <p class="text-sm text-neutral-600 dark:text-neutral-400 truncate">管理群组聊天</p>
            </div>
          </div>
        </UCard>

        <!-- 动态分享 -->
        <UCard 
          class="hover:shadow-lg active:scale-95 transition-all duration-200 cursor-pointer touch-manipulation" 
          @click="navigateTo('/moments')"
        >
          <div class="flex items-center space-x-3 p-1">
            <div class="w-12 h-12 bg-moment-100 dark:bg-moment-900 rounded-lg flex items-center justify-center flex-shrink-0">
              <UIcon name="i-heroicons-photo" class="w-6 h-6 text-moment-600 dark:text-moment-400" />
            </div>
            <div class="flex-1 min-w-0">
              <h3 class="font-semibold text-neutral-900 dark:text-neutral-100 truncate">动态分享</h3>
              <p class="text-sm text-neutral-600 dark:text-neutral-400 truncate">分享精彩瞬间</p>
            </div>
          </div>
        </UCard>

        <!-- 地图探索 -->
        <UCard 
          class="hover:shadow-lg active:scale-95 transition-all duration-200 cursor-pointer touch-manipulation" 
          @click="navigateTo('/map')"
        >
          <div class="flex items-center space-x-3 p-1">
            <div class="w-12 h-12 bg-location-100 dark:bg-location-900 rounded-lg flex items-center justify-center flex-shrink-0">
              <UIcon name="i-heroicons-map" class="w-6 h-6 text-location-600 dark:text-location-400" />
            </div>
            <div class="flex-1 min-w-0">
              <h3 class="font-semibold text-neutral-900 dark:text-neutral-100 truncate">地图探索</h3>
              <p class="text-sm text-neutral-600 dark:text-neutral-400 truncate">查看周边热点</p>
            </div>
          </div>
        </UCard>
      </div>

      <!-- 最近活动 -->
      <div class="mb-8">
        <div class="flex items-center justify-between mb-4">
          <h2 class="text-xl font-semibold text-neutral-900 dark:text-neutral-100">最近活动</h2>
          <UButton variant="ghost" size="sm" @click="refreshActivities" class="flex items-center gap-2">
            <UIcon name="i-heroicons-arrow-path" class="w-4 h-4" :class="{ 'animate-spin': refreshing }" />
            刷新
          </UButton>
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
    </div>

    <!-- Modal 组件现在通过 useOverlay 管理 -->
  </div>
</template>

<script setup lang="ts">
// 首页元数据
definePageMeta({
  title: "首页",
});

const authStore = useAuthStore();
const { openWelcomeModal, openLoginPromptModal } = useModals();

// 认证状态
import type { User } from '~/types';
const displayedUser = ref<User | null>(null);
onMounted(async () => { 
  displayedUser.value = authStore.user || null;
  
  // 智能化用户引导逻辑
  const hasVisited = localStorage.getItem('has-visited-hotspot');
  const lastVisit = localStorage.getItem('last-visit-time');
  const currentTime = Date.now();
  
  if (!hasVisited && !displayedUser.value) {
    // 首次访问：延迟显示欢迎弹窗
    setTimeout(() => {
      openWelcomeModal();
    }, 1000); // 增加延迟，让用户先浏览页面
    localStorage.setItem('has-visited-hotspot', 'true');
    localStorage.setItem('last-visit-time', currentTime.toString());
  } else if (!displayedUser.value) {
    // 非首次访问的未登录用户
    const neverShow = localStorage.getItem('never-show-login-prompt');
    const lastPromptTime = localStorage.getItem('last-login-prompt-time');
    
    // 智能提示：24小时内不重复提示
    const shouldShowPrompt = !neverShow && 
      (!lastPromptTime || (currentTime - parseInt(lastPromptTime)) > 24 * 60 * 60 * 1000);
    
    if (shouldShowPrompt) {
      setTimeout(() => {
        openLoginPromptModal();
        localStorage.setItem('last-login-prompt-time', currentTime.toString());
      }, 3000); // 3秒后显示，给用户更多浏览时间
    }
  }
  
  // 更新最后访问时间
  if (lastVisit) {
    localStorage.setItem('last-visit-time', currentTime.toString());
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
    
    // 注意：活动功能需要后端 API 支持，当前使用模拟数据
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

// Modal 事件处理现在在 useModals composable 中管理

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
    handler: () => navigateTo('/users/nearby'),
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

  // 移除了与系统快捷键冲突的快捷键定义
})
</script>