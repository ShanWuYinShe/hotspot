<template>
  <div class="min-h-screen bg-gradient-to-br from-primary-50 to-green-50">
    <!-- 主要内容区域 -->
    <div class="container mx-auto px-4 py-8">
      <!-- 欢迎横幅 -->
      <div class="text-center mb-16">
        <div class="mb-6">
          <div class="w-20 h-20 bg-primary-500 rounded-full flex items-center justify-center mx-auto mb-4">
            <UIcon name="i-heroicons-map-pin" class="w-10 h-10 text-white" />
          </div>
        </div>
        <h1 class="text-4xl lg:text-6xl font-bold text-gray-900 mb-6">
          欢迎来到 
          <span class="text-primary-500">HotSpot</span>
        </h1>
        <p class="text-xl lg:text-2xl text-gray-600 max-w-3xl mx-auto leading-relaxed">
          发现身边的精彩，连接周围的朋友<br class="hidden sm:block">
          在这里开始你的社交之旅
        </p>
      </div>

      <!-- 用户状态卡片 -->
      <div v-if="isAuthenticated" class="max-w-4xl mx-auto mb-16">
        <UCard class="bg-white/80 backdrop-blur-sm">
          <div class="flex items-center space-x-4 p-2">
            <UAvatar 
              :src="currentUser?.avatar_url" 
              :alt="currentUser?.nickname"
              size="lg"
            >
              {{ currentUser?.nickname?.charAt(0) || 'U' }}
            </UAvatar>
            <div class="flex-1">
              <h3 class="font-semibold text-gray-900">
                你好，{{ currentUser?.nickname || '用户' }}！
              </h3>
              <p class="text-gray-600">
                今天想要做什么呢？
              </p>
            </div>
            <div class="flex space-x-2">
              <UButton @click="navigateTo('/map')" class="hidden sm:flex">
                <UIcon name="i-heroicons-map-pin" class="w-4 h-4 mr-2" />
                探索附近
              </UButton>
              <UButton variant="outline" @click="navigateTo('/messages')">
                <UIcon name="i-heroicons-chat-bubble-left-right" class="w-4 h-4 mr-2" />
                <span class="hidden sm:inline">查看消息</span>
                <span class="sm:hidden">消息</span>
              </UButton>
            </div>
          </div>
        </UCard>
      </div>

      <!-- 功能特色卡片 -->
      <div class="max-w-6xl mx-auto mb-16">
        <h2 class="text-3xl font-bold text-center text-gray-900 mb-12">
          平台特色功能
        </h2>
        
        <div class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6 lg:gap-8">
          <UCard class="group hover:shadow-xl transition-all duration-300 hover:-translate-y-1 bg-white/80 backdrop-blur-sm">
            <div class="text-center p-6">
              <div class="mb-6">
                <div class="w-16 h-16 bg-primary-100 rounded-full flex items-center justify-center mx-auto group-hover:bg-primary-200 transition-colors">
                  <UIcon name="i-heroicons-map-pin" class="w-8 h-8 text-primary-600" />
                </div>
              </div>
              <h3 class="text-xl font-semibold mb-3 text-gray-900">位置发现</h3>
              <p class="text-gray-600 leading-relaxed">
                基于地理位置发现周围的用户和有趣的地点，拓展你的社交圈
              </p>
            </div>
          </UCard>

          <UCard class="group hover:shadow-xl transition-all duration-300 hover:-translate-y-1 bg-white/80 backdrop-blur-sm">
            <div class="text-center p-6">
              <div class="mb-6">
                <div class="w-16 h-16 bg-green-100 rounded-full flex items-center justify-center mx-auto group-hover:bg-green-200 transition-colors">
                  <UIcon name="i-heroicons-chat-bubble-left-right" class="w-8 h-8 text-green-600" />
                </div>
              </div>
              <h3 class="text-xl font-semibold mb-3 text-gray-900">实时聊天</h3>
              <p class="text-gray-600 leading-relaxed">
                支持文字、图片、语音等多种消息类型，与朋友保持密切联系
              </p>
            </div>
          </UCard>

          <UCard class="group hover:shadow-xl transition-all duration-300 hover:-translate-y-1 bg-white/80 backdrop-blur-sm">
            <div class="text-center p-6">
              <div class="mb-6">
                <div class="w-16 h-16 bg-blue-100 rounded-full flex items-center justify-center mx-auto group-hover:bg-blue-200 transition-colors">
                  <UIcon name="i-heroicons-user-group" class="w-8 h-8 text-blue-600" />
                </div>
              </div>
              <h3 class="text-xl font-semibold mb-3 text-gray-900">群组聊天</h3>
              <p class="text-gray-600 leading-relaxed">
                创建或加入兴趣群组，与志同道合的朋友一起交流分享
              </p>
            </div>
          </UCard>

          <UCard class="group hover:shadow-xl transition-all duration-300 hover:-translate-y-1 bg-white/80 backdrop-blur-sm">
            <div class="text-center p-6">
              <div class="mb-6">
                <div class="w-16 h-16 bg-purple-100 rounded-full flex items-center justify-center mx-auto group-hover:bg-purple-200 transition-colors">
                  <UIcon name="i-heroicons-photo" class="w-8 h-8 text-purple-600" />
                </div>
              </div>
              <h3 class="text-xl font-semibold mb-3 text-gray-900">动态分享</h3>
              <p class="text-gray-600 leading-relaxed">
                分享生活中的精彩瞬间，记录美好时光，与朋友互动点赞
              </p>
            </div>
          </UCard>

          <UCard class="group hover:shadow-xl transition-all duration-300 hover:-translate-y-1 bg-white/80 backdrop-blur-sm">
            <div class="text-center p-6">
              <div class="mb-6">
                <div class="w-16 h-16 bg-orange-100 rounded-full flex items-center justify-center mx-auto group-hover:bg-orange-200 transition-colors">
                  <UIcon name="i-heroicons-bell" class="w-8 h-8 text-orange-600" />
                </div>
              </div>
              <h3 class="text-xl font-semibold mb-3 text-gray-900">智能通知</h3>
              <p class="text-gray-600 leading-relaxed">
                及时接收消息提醒和附近活动通知，不错过任何重要信息
              </p>
            </div>
          </UCard>

          <UCard class="group hover:shadow-xl transition-all duration-300 hover:-translate-y-1 bg-white/80 backdrop-blur-sm">
            <div class="text-center p-6">
              <div class="mb-6">
                <div class="w-16 h-16 bg-red-100 rounded-full flex items-center justify-center mx-auto group-hover:bg-red-200 transition-colors">
                  <UIcon name="i-heroicons-shield-check" class="w-8 h-8 text-red-600" />
                </div>
              </div>
              <h3 class="text-xl font-semibold mb-3 text-gray-900">隐私安全</h3>
              <p class="text-gray-600 leading-relaxed">
                完善的隐私设置和安全保护，让你的社交体验更加安心
              </p>
            </div>
          </UCard>
        </div>
      </div>

      <!-- 快速开始按钮 -->
      <div class="text-center">
        <div v-if="!isAuthenticated" class="space-y-4 sm:space-y-0 sm:space-x-6 sm:flex sm:justify-center">
          <UButton size="xl" @click="navigateTo('/auth')" class="w-full sm:w-auto">
            <UIcon name="i-heroicons-rocket-launch" class="w-5 h-5 mr-2" />
            立即开始
          </UButton>
          <UButton variant="outline" size="xl" @click="scrollToFeatures" class="w-full sm:w-auto">
            <UIcon name="i-heroicons-information-circle" class="w-5 h-5 mr-2" />
            了解更多
          </UButton>
        </div>
        <div v-else class="space-y-4 sm:space-y-0 sm:space-x-6 sm:flex sm:justify-center">
          <UButton size="xl" @click="navigateTo('/map')" class="w-full sm:w-auto">
            <UIcon name="i-heroicons-map-pin" class="w-5 h-5 mr-2" />
            探索附近
          </UButton>
          <UButton variant="outline" size="xl" @click="navigateTo('/messages')" class="w-full sm:w-auto">
            <UIcon name="i-heroicons-chat-bubble-left-right" class="w-5 h-5 mr-2" />
            查看消息
          </UButton>
          <UButton variant="soft" size="xl" @click="navigateTo('/groups')" class="w-full sm:w-auto">
            <UIcon name="i-heroicons-user-group" class="w-5 h-5 mr-2" />
            我的群组
          </UButton>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
// 首页元数据
definePageMeta({
  title: '首页'
})

const authStore = useAuthStore()

// 认证状态
const isAuthenticated = computed(() => authStore.isAuthenticated)
const currentUser = computed(() => authStore.currentUser)

// 滚动到功能介绍
const scrollToFeatures = () => {
  // 这里可以添加平滑滚动到功能介绍区域的逻辑
  const featuresSection = document.querySelector('h2')
  if (featuresSection) {
    featuresSection.scrollIntoView({ behavior: 'smooth' })
  }
}
</script>
