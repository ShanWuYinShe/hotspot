<template>
  <div class="space-y-6" :style="{ viewTransitionName: 'main-content' }">
    <!-- 用户欢迎区域 -->
    <div v-if="displayedUserResponse" class="bg-gradient-to-r from-primary-50 to-blue-50 dark:from-primary-950 dark:to-blue-950 rounded-xl p-6 border border-primary-200 dark:border-primary-800">
      <div class="flex items-center gap-4">
        <div class="relative flex-shrink-0">
          <UChip color="success" size="lg" position="bottom-right">
            <UAvatar :alt="displayedUserResponse.nickname" size="xl" :src="displayedUserResponse.avatar_url">
              {{ displayedUserResponse.nickname?.charAt(0) || "U" }}
            </UAvatar>
          </UChip>
        </div>
        
        <div class="flex-1">
          <h1 class="text-2xl font-bold text-neutral-900 dark:text-white mb-2">
            {{ getGreetingTitle() }}
          </h1>
          <p class="text-neutral-600 dark:text-neutral-400 mb-3">
            {{ getGreetingMessage() }}
          </p>
          
          <div class="flex items-center gap-3">
            <UBadge 
              v-if="displayedUserResponse.is_temp" 
              color="warning" 
              variant="soft" 
              size="sm"
            >
              <UIcon name="i-heroicons-clock" class="w-3 h-3 mr-1" />
              临时用户
            </UBadge>
            <UBadge 
              color="success" 
              variant="soft" 
              size="sm"
            >
              <UIcon name="i-heroicons-signal" class="w-3 h-3 mr-1" />
              在线
            </UBadge>
            <div class="flex items-center gap-1 text-sm text-neutral-600 dark:text-neutral-400">
              <UIcon name="i-heroicons-chat-bubble-left-right" class="w-4 h-4" />
              <span>{{ unreadCount > 0 ? `${unreadCount} 条未读消息` : '暂无未读消息' }}</span>
            </div>
          </div>
        </div>
      </div>
    </div>

    <!-- 访客欢迎区域 -->
    <div v-else class="text-center py-8">
      <div class="w-16 h-16 bg-gradient-to-br from-primary-500 to-blue-600 rounded-2xl flex items-center justify-center mx-auto mb-4 shadow-lg">
        <UIcon name="i-heroicons-fire" class="w-8 h-8 text-white" />
      </div>
      <h1 class="text-3xl font-bold text-neutral-900 dark:text-white mb-2">欢迎来到 HotSpot</h1>
      <p class="text-lg text-neutral-600 dark:text-neutral-400 mb-6">基于位置的社交平台，发现身边有趣的人和事</p>
      <div class="flex gap-3 justify-center">
        <UButton to="/auth/register" color="primary" size="lg">
          <UIcon name="i-heroicons-user-plus" class="w-5 h-5 mr-2" />
          立即注册
        </UButton>
        <UButton to="/auth/login" variant="outline" size="lg">
          <UIcon name="i-heroicons-arrow-right-on-rectangle" class="w-5 h-5 mr-2" />
          登录
        </UButton>
      </div>
    </div>

    <!-- 快速操作区域 -->
    <div>
      <div class="flex items-center gap-3 mb-4">
        <UIcon name="i-heroicons-bolt" class="w-5 h-5 text-primary-500" />
        <h2 class="text-xl font-semibold text-neutral-900 dark:text-white">快速操作</h2>
      </div>
      <p class="text-neutral-600 dark:text-neutral-400 mb-6">选择您想要进行的操作，快速开始使用 HotSpot</p>

      <div class="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-4">
        <UCard
          class="hover:shadow-lg transition-shadow cursor-pointer"
          @click="navigateTo('/users/nearby')"
        >
          <template #header>
            <div class="flex items-center gap-3">
              <div class="w-10 h-10 bg-blue-100 dark:bg-blue-900/50 rounded-lg flex items-center justify-center">
                <UIcon name="i-heroicons-users" class="w-5 h-5 text-blue-600 dark:text-blue-400" />
              </div>
              <div>
                <h3 class="font-semibold text-neutral-900 dark:text-white">附近用户</h3>
                <p class="text-sm text-neutral-600 dark:text-neutral-400">发现周围的朋友</p>
              </div>
            </div>
          </template>
        </UCard>

        <UCard
          class="hover:shadow-lg transition-shadow cursor-pointer"
          @click="navigateTo('/groups')"
        >
          <template #header>
            <div class="flex items-center gap-3">
              <div class="w-10 h-10 bg-purple-100 dark:bg-purple-900/50 rounded-lg flex items-center justify-center">
                <UIcon name="i-heroicons-user-group" class="w-5 h-5 text-purple-600 dark:text-purple-400" />
              </div>
              <div>
                <h3 class="font-semibold text-neutral-900 dark:text-white">我的群组</h3>
                <p class="text-sm text-neutral-600 dark:text-neutral-400">管理群组聊天</p>
              </div>
            </div>
          </template>
        </UCard>

        <UCard
          class="hover:shadow-lg transition-shadow cursor-pointer"
          @click="navigateTo('/moments')"
        >
          <template #header>
            <div class="flex items-center gap-3">
              <div class="w-10 h-10 bg-orange-100 dark:bg-orange-900/50 rounded-lg flex items-center justify-center">
                <UIcon name="i-heroicons-photo" class="w-5 h-5 text-orange-600 dark:text-orange-400" />
              </div>
              <div>
                <h3 class="font-semibold text-neutral-900 dark:text-white">动态分享</h3>
                <p class="text-sm text-neutral-600 dark:text-neutral-400">分享精彩瞬间</p>
              </div>
            </div>
          </template>
        </UCard>

        <UCard
          class="hover:shadow-lg transition-shadow cursor-pointer"
          @click="navigateTo('/map')"
          to="/map"
        >
          <template #header>
            <div class="flex items-center gap-3">
              <div class="w-10 h-10 bg-green-100 dark:bg-green-900/50 rounded-lg flex items-center justify-center">
                <UIcon name="i-heroicons-map" class="w-5 h-5 text-green-600 dark:text-green-400" />
              </div>
              <div>
                <h3 class="font-semibold text-neutral-900 dark:text-white">地图探索</h3>
                <p class="text-sm text-neutral-600 dark:text-neutral-400">查看周边热点</p>
              </div>
            </div>
          </template>
        </UCard>

        <UCard
          class="hover:shadow-lg transition-shadow cursor-pointer relative"
          @click="navigateTo('/chat')"
        >
          <template #header>
            <div class="flex items-center gap-3">
              <div class="w-10 h-10 bg-indigo-100 dark:bg-indigo-900/50 rounded-lg flex items-center justify-center relative">
                <UIcon name="i-heroicons-chat-bubble-left-right" class="w-5 h-5 text-indigo-600 dark:text-indigo-400" />
                <UBadge
                  v-if="unreadCount > 0"
                  :label="unreadCount > 99 ? '99+' : unreadCount.toString()"
                  color="error"
                  size="xs"
                  class="absolute -top-1 -right-1"
                />
              </div>
              <div>
                <h3 class="font-semibold text-neutral-900 dark:text-white">消息中心</h3>
                <p class="text-sm text-neutral-600 dark:text-neutral-400">查看聊天消息</p>
              </div>
            </div>
          </template>
        </UCard>

        <UCard
          class="hover:shadow-lg transition-shadow cursor-pointer"
          @click="navigateTo('/settings')"
        >
          <template #header>
            <div class="flex items-center gap-3">
              <div class="w-10 h-10 bg-neutral-100 dark:bg-neutral-800 rounded-lg flex items-center justify-center">
                <UIcon name="i-heroicons-cog-6-tooth" class="w-5 h-5 text-neutral-600 dark:text-neutral-400" />
              </div>
              <div>
                <h3 class="font-semibold text-neutral-900 dark:text-white">个人设置</h3>
                <p class="text-sm text-neutral-600 dark:text-neutral-400">管理账户信息</p>
              </div>
            </div>
          </template>
        </UCard>
      </div>
    </div>

    <!-- 最近活动 -->
    <div>
      <div class="flex items-center justify-between w-full mb-4">
        <div class="flex items-center gap-3">
          <UIcon name="i-heroicons-clock" class="w-5 h-5 text-primary-500" />
          <h2 class="text-xl font-semibold text-neutral-900 dark:text-white">最近活动</h2>
        </div>
        <UButton 
          v-if="displayedUserResponse"
          variant="outline" 
          size="sm" 
          @click="refreshActivities" 
          :loading="refreshing"
        >
          <UIcon 
            name="i-heroicons-arrow-path" 
            class="w-4 h-4 mr-2" 
            :class="{ 'animate-spin': refreshing }" 
          />
          刷新
        </UButton>
      </div>
      
      <p class="text-neutral-600 dark:text-neutral-400 mb-6">查看最新的社区动态和好友活动</p>

      <!-- 加载状态 -->
      <div v-if="activitiesLoading" class="space-y-4">
        <div v-for="i in 3" :key="i" class="flex items-center gap-3 p-4 bg-neutral-50 dark:bg-neutral-800 rounded-lg">
          <USkeleton class="w-10 h-10 rounded-full" />
          <div class="flex-1 space-y-2">
            <USkeleton class="h-4 w-3/4" />
            <USkeleton class="h-3 w-1/2" />
          </div>
          <USkeleton class="h-3 w-16" />
        </div>
      </div>

      <!-- 错误状态 -->
      <div v-else-if="activitiesError" class="text-center py-12">
        <div class="w-16 h-16 bg-error-100 dark:bg-error-900/50 rounded-2xl flex items-center justify-center mx-auto mb-4">
          <UIcon name="i-heroicons-exclamation-triangle" class="w-8 h-8 text-error-500" />
        </div>
        <h3 class="text-lg font-semibold text-neutral-900 dark:text-white mb-2">{{ activitiesError }}</h3>
        <p class="text-neutral-600 dark:text-neutral-400 mb-4">请稍后重试</p>
        <UButton @click="fetchActivities" variant="outline" size="sm">
          <UIcon name="i-heroicons-arrow-path" class="w-4 h-4 mr-2" />
          重试
        </UButton>
      </div>

      <!-- 活动时间线 -->
      <div v-else-if="activities.length > 0" class="space-y-4">
        <UTimeline :items="timelineItems" />
      </div>

      <!-- 空状态 -->
      <div v-else class="text-center py-12">
        <div class="w-16 h-16 bg-neutral-100 dark:bg-neutral-800 rounded-2xl flex items-center justify-center mx-auto mb-4">
          <UIcon name="i-heroicons-sparkles" class="w-8 h-8 text-neutral-500" />
        </div>
        <h3 class="text-lg font-semibold text-neutral-900 dark:text-white mb-2">暂无活动</h3>
        <p class="text-neutral-600 dark:text-neutral-400 mb-4">开始探索，创建你的第一个活动吧！</p>
        <UButton to="/moments" color="primary" size="sm">
          <UIcon name="i-heroicons-plus" class="w-4 h-4 mr-2" />
          发布动态
        </UButton>
      </div>
    </div>

    <!-- 功能特色 -->
    <div v-if="!displayedUserResponse">
      <div class="flex items-center gap-3 mb-4">
        <UIcon name="i-heroicons-star" class="w-5 h-5 text-primary-500" />
        <h2 class="text-xl font-semibold text-neutral-900 dark:text-white">为什么选择 HotSpot</h2>
      </div>
      
      <p class="text-neutral-600 dark:text-neutral-400 mb-6">基于位置的社交平台，让您轻松发现身边的精彩</p>

      <div class="grid grid-cols-1 lg:grid-cols-2 gap-6">
        <UCard class="lg:col-span-2">
          <template #header>
            <div class="flex items-center gap-4">
              <div class="w-12 h-12 bg-primary-100 dark:bg-primary-900/40 rounded-xl flex items-center justify-center">
                <UIcon name="i-heroicons-map-pin" class="w-6 h-6 text-primary-600 dark:text-primary-300" />
              </div>
              <div>
                <h3 class="text-lg font-semibold text-neutral-900 dark:text-white">基于位置的发现</h3>
                <p class="text-neutral-600 dark:text-neutral-400">发现附近用户与群组，快速开始交流</p>
              </div>
            </div>
          </template>
        </UCard>

        <UCard>
          <template #header>
            <div class="flex items-center gap-4">
              <div class="w-12 h-12 bg-success-100 dark:bg-success-900/40 rounded-xl flex items-center justify-center">
                <UIcon name="i-heroicons-chat-bubble-left-right" class="w-6 h-6 text-success-600 dark:text-success-300" />
              </div>
              <div>
                <h3 class="text-lg font-semibold text-neutral-900 dark:text-white">实时消息</h3>
                <p class="text-neutral-600 dark:text-neutral-400">与朋友和群组实时沟通，支持多种消息类型</p>
              </div>
            </div>
          </template>
        </UCard>

        <UCard>
          <template #header>
            <div class="flex items-center gap-4">
              <div class="w-12 h-12 bg-info-100 dark:bg-info-900/40 rounded-xl flex items-center justify-center">
                <UIcon name="i-heroicons-photo" class="w-6 h-6 text-info-600 dark:text-info-300" />
              </div>
              <div>
                <h3 class="text-lg font-semibold text-neutral-900 dark:text-white">动态与互动</h3>
                <p class="text-neutral-600 dark:text-neutral-400">发布动态，点赞互动，记录精彩</p>
              </div>
            </div>
          </template>
        </UCard>

        <UCard class="lg:col-span-2">
          <template #header>
            <div class="flex items-center gap-4">
              <div class="w-12 h-12 bg-warning-100 dark:bg-warning-900/40 rounded-xl flex items-center justify-center">
                <UIcon name="i-heroicons-shield-check" class="w-6 h-6 text-warning-600 dark:text-warning-300" />
              </div>
              <div>
                <h3 class="text-lg font-semibold text-neutral-900 dark:text-white">隐私安全</h3>
                <p class="text-neutral-600 dark:text-neutral-400">端到端加密，保护您的隐私和数据安全</p>
              </div>
            </div>
          </template>
        </UCard>
      </div>
    </div>

    <!-- CTA 区域 -->
    <div v-if="!displayedUserResponse" class="bg-gradient-to-r from-primary-500 to-blue-600 rounded-xl p-8 text-center text-white">
      <div class="max-w-2xl mx-auto">
        <h2 class="text-3xl font-bold mb-4">准备开始了吗？</h2>
        <p class="text-xl text-primary-100 mb-8">立即加入 HotSpot，开始您的社交之旅</p>
        
        <div class="flex flex-col sm:flex-row gap-4 justify-center">
          <UButton
            to="/auth/register"
            color="primary"
            size="lg"
            class="bg-white text-primary-600 hover:bg-primary-50"
          >
            <UIcon name="i-heroicons-user-plus" class="w-5 h-5 mr-2" />
            立即注册
          </UButton>
          <UButton
            to="/auth/login"
            variant="outline"
            size="lg"
            class="border-white text-white hover:bg-white hover:text-primary-600"
          >
            <UIcon name="i-heroicons-arrow-right-on-rectangle" class="w-5 h-5 mr-2" />
            登录账户
          </UButton>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { getPageSeoConfig, buildSeoMeta } from '~/config/seo'

// 从数据源获取 SEO 配置
const seoConfig = getPageSeoConfig('home')

// 使用 Nuxt v4 的 definePageMeta
definePageMeta({
  title: seoConfig.title,
  description: seoConfig.description,
  layout: 'dashboard',
  // 启用视图过渡
  viewTransition: true,
  // 页面级别的 keepalive
  keepalive: true,
  // 预加载策略
  prerender: true
});

// 直接使用 Nuxt v4 的 useSeoMeta
useSeoMeta(buildSeoMeta('home'));

const authStore = useAuthStore();
const { openWelcomeModal, openLoginPromptModal } = useModals();

// 认证状态
import type { UserResponse } from '~/types/api';
const displayedUserResponse = ref<UserResponse | null>(null);

// 未读消息数量
const messagesStore = useMessagesStore();
const unreadCount = computed(() => messagesStore.totalUnreadCount);


// 问候消息
const getGreetingTitle = () => {
  if (displayedUserResponse.value) {
    return `你好，${displayedUserResponse.value.nickname}！`;
  }
  return '发现身边的精彩';
};

const getGreetingMessage = () => {
  const hour = new Date().getHours();
  if (displayedUserResponse.value) {
    if (hour < 6) return '夜深了，注意休息哦';
    if (hour < 12) return '早上好！新的一天开始了';
    if (hour < 18) return '下午好！今天过得怎么样？';
    if (hour < 22) return '晚上好！放松一下吧';
    return '夜深了，早点休息哦';
  } else {
    return '基于位置的社交平台，与朋友分享生活的精彩时刻';
  }
};

// 活动数据
const refreshing = ref(false);
const activities = ref<any[]>([]);
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

// 生命周期
onMounted(async () => { 
  displayedUserResponse.value = authStore.user || null;
  
  // 智能化用户引导逻辑
  const hasVisited = localStorage.getItem('has-visited-hotspot');
  const lastVisit = localStorage.getItem('last-visit-time');
  const currentTime = Date.now();
  
  if (!hasVisited && !displayedUserResponse.value) {
    // 首次访问：延迟显示欢迎弹窗
    setTimeout(() => {
      openWelcomeModal();
    }, 1000);
    localStorage.setItem('has-visited-hotspot', 'true');
    localStorage.setItem('last-visit-time', currentTime.toString());
  } else if (!displayedUserResponse.value) {
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
      }, 3000);
    }
  }
  
  // 更新最后访问时间
  if (lastVisit) {
    localStorage.setItem('last-visit-time', currentTime.toString());
  }
  
  // 获取活动数据
  await fetchActivities();
});

// 快捷键管理
onMounted(() => {
  const shortcuts = useShortcuts();
  shortcuts.register('meta+h', () => navigateTo('/'), '返回首页', '导航');
  shortcuts.register('meta+m', () => navigateTo('/map'), '地图探索', '功能');
  shortcuts.register('meta+g', () => navigateTo('/groups'), '我的群组', '功能');
  shortcuts.register('meta+c', () => navigateTo('/chat'), '消息聊天', '功能');
  shortcuts.register('alt+u', () => navigateTo('/users/nearby'), '附近用户', '功能');
  shortcuts.register('alt+m', () => navigateTo('/moments'), '动态分享', '功能');
  shortcuts.register('meta+r', refreshActivities, '刷新活动', '操作');
});
</script>