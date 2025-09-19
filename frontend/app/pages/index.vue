<template>
  <div class="min-h-screen bg-gradient-to-br from-slate-50 via-slate-100/50 to-slate-50 dark:from-slate-900 dark:via-slate-800/30 dark:to-slate-900">
    <!-- 背景装饰 -->
    <div class="absolute inset-0 overflow-hidden pointer-events-none">
      <div class="absolute -top-40 -right-40 w-80 h-80 bg-gradient-to-br from-slate-300/10 to-slate-400/10 rounded-full blur-3xl"></div>
      <div class="absolute -bottom-40 -left-40 w-80 h-80 bg-gradient-to-tr from-slate-300/10 to-slate-400/10 rounded-full blur-3xl"></div>
    </div>

    <div class="relative mx-auto max-w-7xl px-4 sm:px-6 lg:px-8 py-8 sm:py-12">
      <!-- 用户状态卡片 -->
      <div v-if="displayedUser" class="mb-8 sm:mb-12">
        <UCard class="bg-white/70 dark:bg-slate-800/70 border border-slate-200/50 dark:border-slate-700/50">
          <div class="flex flex-col sm:flex-row items-start sm:items-center space-y-4 sm:space-y-0 sm:space-x-6 p-6">
            <div class="relative flex-shrink-0">
              <UAvatar :alt="displayedUser?.nickname" size="2xl">
                {{ displayedUser?.nickname?.charAt(0) || "U" }}
              </UAvatar>
              <!-- 在线状态指示器 -->
              <div class="absolute -bottom-2 -right-2 w-6 h-6 bg-green-500 rounded-full border-3 border-white dark:border-slate-800"></div>
            </div>
            <div class="flex-1 min-w-0 space-y-3">
              <div>
                <h3 class="text-2xl sm:text-3xl font-bold text-slate-800 dark:text-slate-100 mb-2">
                  你好，{{ displayedUser?.nickname || "用户" }}！
                </h3>
                <p class="text-lg sm:text-xl text-slate-600 dark:text-slate-400 leading-relaxed">
                  {{ getGreetingMessage() }}
                </p>
              </div>
              <!-- 用户类型徽章 -->
              <div class="flex items-center gap-3">
                <UBadge 
                  v-if="displayedUser?.is_temp" 
                  color="warning" 
                  variant="soft" 
                  size="lg"
                >
                  <UIcon name="i-heroicons-clock" class="w-4 h-4 mr-2" />
                  临时用户
                </UBadge>
                <UBadge 
                  color="success" 
                  variant="soft" 
                  size="lg"
                >
                  <UIcon name="i-heroicons-signal" class="w-4 h-4 mr-2" />
                  在线
                </UBadge>
              </div>
            </div>
            <div class="flex flex-col sm:flex-row gap-3 w-full sm:w-auto">
              <UButton 
                to="/map" 
                size="lg" 
                color="primary"
              >
                <UIcon name="i-heroicons-map-pin" class="w-5 h-5 mr-2" />
                <span class="hidden sm:inline">探索附近</span>
                <span class="sm:hidden">探索</span>
              </UButton>
              <UButton 
                to="/chat" 
                variant="outline" 
                size="lg" 
                class="relative"
              >
                <UIcon name="i-heroicons-chat-bubble-left-right" class="w-5 h-5 mr-2" />
                <span class="hidden sm:inline">查看消息</span>
                <span class="sm:hidden">消息</span>
                <!-- 未读消息指示器 -->
                <div v-if="unreadCount > 0" class="absolute -top-2 -right-2 w-5 h-5 bg-red-500 text-white text-xs font-bold rounded-full flex items-center justify-center">
                  {{ unreadCount > 99 ? '99+' : unreadCount }}
                </div>
              </UButton>
            </div>
          </div>
        </UCard>
      </div>

      <!-- 快速操作区域 -->
      <div class="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-6 sm:gap-8 mb-16">
        <!-- 附近用户 -->
        <UCard 
          class="group cursor-pointer bg-white/60 dark:bg-slate-800/60 border border-slate-200/50 dark:border-slate-700/50 hover:bg-white/80 dark:hover:bg-slate-800/80 transition-colors"
          @click="navigateTo('/users/nearby')"
        >
          <div class="flex flex-col items-center text-center p-6 space-y-4">
            <div class="w-16 h-16 bg-blue-100 dark:bg-blue-900/50 rounded-2xl flex items-center justify-center">
              <UIcon name="i-heroicons-users" class="w-8 h-8 text-blue-600 dark:text-blue-400" />
            </div>
            <div class="space-y-1">
              <h3 class="font-semibold text-lg text-slate-900 dark:text-white">附近用户</h3>
              <p class="text-sm text-slate-600 dark:text-slate-400">发现周围的朋友</p>
            </div>
          </div>
        </UCard>

        <!-- 我的群组 -->
        <UCard 
          class="group cursor-pointer bg-white/60 dark:bg-slate-800/60 border border-slate-200/50 dark:border-slate-700/50 hover:bg-white/80 dark:hover:bg-slate-800/80 transition-colors"
          @click="navigateTo('/groups')"
        >
          <div class="flex flex-col items-center text-center p-6 space-y-4">
            <div class="w-16 h-16 bg-purple-100 dark:bg-purple-900/50 rounded-2xl flex items-center justify-center">
              <UIcon name="i-heroicons-user-group" class="w-8 h-8 text-purple-600 dark:text-purple-400" />
            </div>
            <div class="space-y-1">
              <h3 class="font-semibold text-lg text-slate-900 dark:text-white">我的群组</h3>
              <p class="text-sm text-slate-600 dark:text-slate-400">管理群组聊天</p>
            </div>
          </div>
        </UCard>

        <!-- 动态分享 -->
        <UCard 
          class="group cursor-pointer bg-white/60 dark:bg-slate-800/60 border border-slate-200/50 dark:border-slate-700/50 hover:bg-white/80 dark:hover:bg-slate-800/80 transition-colors"
          @click="navigateTo('/moments')"
        >
          <div class="flex flex-col items-center text-center p-6 space-y-4">
            <div class="w-16 h-16 bg-orange-100 dark:bg-orange-900/50 rounded-2xl flex items-center justify-center">
              <UIcon name="i-heroicons-photo" class="w-8 h-8 text-orange-600 dark:text-orange-400" />
            </div>
            <div class="space-y-1">
              <h3 class="font-semibold text-lg text-slate-900 dark:text-white">动态分享</h3>
              <p class="text-sm text-slate-600 dark:text-slate-400">分享精彩瞬间</p>
            </div>
          </div>
        </UCard>

        <!-- 地图探索 -->
        <UCard 
          class="group cursor-pointer bg-white/60 dark:bg-slate-800/60 border border-slate-200/50 dark:border-slate-700/50 hover:bg-white/80 dark:hover:bg-slate-800/80 transition-colors"
          @click="navigateTo('/map')"
        >
          <div class="flex flex-col items-center text-center p-6 space-y-4">
            <div class="w-16 h-16 bg-green-100 dark:bg-green-900/50 rounded-2xl flex items-center justify-center">
              <UIcon name="i-heroicons-map" class="w-8 h-8 text-green-600 dark:text-green-400" />
            </div>
            <div class="space-y-1">
              <h3 class="font-semibold text-lg text-slate-900 dark:text-white">地图探索</h3>
              <p class="text-sm text-slate-600 dark:text-slate-400">查看周边热点</p>
            </div>
          </div>
        </UCard>
      </div>

      <!-- 最近活动 -->
      <div class="mb-8">
        <UCard class="bg-white/60 dark:bg-slate-800/60 border border-slate-200/50 dark:border-slate-700/50">
          <div class="flex flex-col sm:flex-row items-start sm:items-center justify-between gap-4 mb-6 p-6 pb-0">
            <div class="space-y-2">
              <h2 class="text-2xl font-bold text-slate-800 dark:text-slate-100">最近活动</h2>
              <p class="text-slate-600 dark:text-slate-400">查看最新的社区动态</p>
            </div>
            <UButton 
              variant="outline" 
              size="lg" 
              @click="refreshActivities" 
              :disabled="refreshing"
            >
              <UIcon 
                name="i-heroicons-arrow-path" 
                class="w-5 h-5 mr-2" 
                :class="{ 'animate-spin': refreshing }" 
              />
              刷新活动
            </UButton>
          </div>
          
          <div class="px-6 pb-6">
            <!-- 加载状态 -->
            <div v-if="activitiesLoading" class="flex items-center justify-center py-12">
              <div class="text-center space-y-4">
                <div class="w-12 h-12 bg-slate-200 dark:bg-slate-700 rounded-full mx-auto animate-pulse"></div>
                <div class="space-y-1">
                  <h3 class="text-lg font-semibold text-slate-900 dark:text-white">加载中...</h3>
                  <p class="text-sm text-slate-600 dark:text-slate-400">正在获取最新活动</p>
                </div>
              </div>
            </div>
            
            <!-- 错误状态 -->
            <div v-else-if="activitiesError" class="flex items-center justify-center py-12">
              <div class="text-center space-y-4">
                <div class="w-12 h-12 bg-red-100 dark:bg-red-900/50 rounded-xl flex items-center justify-center mx-auto">
                  <UIcon name="i-heroicons-exclamation-triangle" class="w-6 h-6 text-red-500" />
                </div>
                <div class="space-y-1">
                  <h3 class="text-lg font-semibold text-slate-900 dark:text-white">{{ activitiesError }}</h3>
                  <p class="text-sm text-slate-600 dark:text-slate-400">请稍后重试</p>
                </div>
                <UButton 
                  @click="fetchActivities" 
                  variant="outline" 
                  size="sm"
                >
                  <UIcon name="i-heroicons-arrow-path" class="w-4 h-4 mr-2" />
                  重试
                </UButton>
              </div>
            </div>
            
            <!-- 活动时间线 -->
            <div v-else-if="activities.length > 0" class="py-4">
              <UTimeline :items="timelineItems" />
            </div>
            
            <!-- 空状态 -->
            <div v-else class="flex items-center justify-center py-12">
              <div class="text-center space-y-4 max-w-sm">
                <div class="w-16 h-16 bg-slate-100 dark:bg-slate-700 rounded-2xl flex items-center justify-center mx-auto">
                  <UIcon name="i-heroicons-sparkles" class="w-8 h-8 text-slate-500" />
                </div>
                <div class="space-y-2">
                  <h3 class="text-lg font-bold text-slate-900 dark:text-white">暂无活动</h3>
                  <p class="text-sm text-slate-600 dark:text-slate-400">开始探索，创建你的第一个活动吧！</p>
                </div>
                <UButton 
                  to="/moments" 
                  color="primary" 
                  size="sm"
                >
                  <UIcon name="i-heroicons-plus" class="w-4 h-4 mr-2" />
                  发布动态
                </UButton>
              </div>
            </div>
          </div>
        </UCard>
      </div>
    </div>
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

// 未读消息数量（示例）
const unreadCount = ref(0);

// 问候消息
const getGreetingMessage = () => {
  const hour = new Date().getHours();
  if (hour < 6) return '夜深了，注意休息哦';
  if (hour < 12) return '早上好！新的一天开始了';
  if (hour < 18) return '下午好！今天过得怎么样？';
  if (hour < 22) return '晚上好！放松一下吧';
  return '夜深了，早点休息哦';
};

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

// 快捷键管理
const shortcuts = useShortcuts()

onMounted(() => {
  // 注册首页快捷键
  shortcuts.register('meta+h', () => navigateTo('/'), '返回首页', '导航')
  shortcuts.register('meta+m', () => navigateTo('/map'), '地图探索', '功能')
  shortcuts.register('meta+g', () => navigateTo('/groups'), '我的群组', '功能')
  shortcuts.register('meta+c', () => navigateTo('/chat'), '消息聊天', '功能')
  shortcuts.register('alt+u', () => navigateTo('/users/nearby'), '附近用户', '功能')
  shortcuts.register('alt+m', () => navigateTo('/moments'), '动态分享', '功能')
  shortcuts.register('meta+r', refreshActivities, '刷新活动', '操作')
})
</script>