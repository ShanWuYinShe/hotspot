<template>
  <div class="min-h-screen bg-gray-50">
    <!-- 顶部导航栏 -->
    <nav class="bg-white shadow-sm border-b">
      <div class="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
        <div class="flex justify-between h-16">
          <div class="flex items-center">
            <!-- Logo -->
            <NuxtLink to="/" class="flex items-center space-x-2">
              <UIcon name="i-heroicons-map-pin" class="w-8 h-8 text-primary-600" />
              <span class="text-xl font-bold text-gray-900">HotSpot</span>
            </NuxtLink>
          </div>
          
          <!-- 用户菜单 -->
          <div class="flex items-center space-x-4">
            <NuxtLink 
              v-if="!authStore.isAuthenticated" 
              to="/auth" 
              class="text-gray-600 hover:text-gray-900"
            >
              <UButton variant="outline">登录</UButton>
            </NuxtLink>
            
            <div v-else class="flex items-center space-x-2">
              <span class="text-sm text-gray-600">
                {{ authStore.currentUser?.nickname }}
              </span>
              <UButton 
                @click="handleLogout" 
                variant="ghost" 
                size="sm"
              >
                退出
              </UButton>
            </div>
          </div>
        </div>
      </div>
    </nav>

    <!-- 主要内容区域 -->
    <main class="flex-1">
      <slot />
    </main>

    <!-- 底部导航栏 (移动端) -->
    <nav class="lg:hidden fixed bottom-0 left-0 right-0 bg-white border-t border-gray-200">
      <div class="flex justify-around py-2">
        <NuxtLink 
          to="/" 
          class="flex flex-col items-center space-y-1 py-2 px-3"
          :class="{ 'text-primary-600': $route.path === '/' }"
        >
          <UIcon name="i-heroicons-home" class="w-6 h-6" />
          <span class="text-xs">首页</span>
        </NuxtLink>
        
        <NuxtLink 
          to="/map" 
          class="flex flex-col items-center space-y-1 py-2 px-3"
          :class="{ 'text-primary-600': $route.path === '/map' }"
        >
          <UIcon name="i-heroicons-map" class="w-6 h-6" />
          <span class="text-xs">地图</span>
        </NuxtLink>
        
        <NuxtLink 
          to="/moments" 
          class="flex flex-col items-center space-y-1 py-2 px-3"
          :class="{ 'text-primary-600': $route.path === '/moments' }"
        >
          <UIcon name="i-heroicons-photo" class="w-6 h-6" />
          <span class="text-xs">动态</span>
        </NuxtLink>
        
        <NuxtLink 
          to="/messages" 
          class="flex flex-col items-center space-y-1 py-2 px-3"
          :class="{ 'text-primary-600': $route.path === '/messages' }"
        >
          <UIcon name="i-heroicons-chat-bubble-left-right" class="w-6 h-6" />
          <span class="text-xs">消息</span>
        </NuxtLink>
        
        <NuxtLink 
          to="/profile" 
          class="flex flex-col items-center space-y-1 py-2 px-3"
          :class="{ 'text-primary-600': $route.path === '/profile' }"
        >
          <UIcon name="i-heroicons-user" class="w-6 h-6" />
          <span class="text-xs">我的</span>
        </NuxtLink>
      </div>
    </nav>
  </div>
</template>

<script setup>
const authStore = useAuthStore()

const handleLogout = async () => {
  await authStore.logout()
  await navigateTo('/auth')
}
</script>
