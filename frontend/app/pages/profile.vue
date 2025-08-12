<template>
  <div class="container mx-auto px-4 py-8">
    <div class="max-w-2xl mx-auto">
      <!-- 个人信息卡片 -->
      <UCard class="mb-6">
        <div class="text-center">
          <!-- 头像 -->
          <div class="mb-4">
            <UAvatar 
              :src="user?.avatar_url" 
              :alt="user?.nickname"
              size="3xl"
              class="mx-auto"
            >
              {{ user?.nickname?.charAt(0) || 'U' }}
            </UAvatar>
          </div>

          <!-- 基本信息 -->
          <h1 class="text-2xl font-bold text-gray-900 mb-2">
            {{ user?.nickname || '用户' }}
          </h1>
          <p class="text-gray-600 mb-4">
            ID: {{ user?.login_id || user?.user_id }}
          </p>
          
          <!-- 临时用户提示 -->
          <div v-if="user?.is_temp" class="mb-4">
            <UAlert 
              color="warning" 
              variant="soft"
              title="临时用户"
              description="升级为正式用户以保存您的数据和享受完整功能"
            >
              <template #actions>
                <UButton color="warning" variant="solid" size="xs">
                  立即升级
                </UButton>
              </template>
            </UAlert>
          </div>

          <!-- 操作按钮 -->
          <div class="flex justify-center space-x-4">
            <UButton @click="editProfile">
              <UIcon name="i-heroicons-pencil-square" class="w-4 h-4 mr-2" />
              编辑资料
            </UButton>
            <UButton variant="outline" @click="viewSettings">
              <UIcon name="i-heroicons-cog-6-tooth" class="w-4 h-4 mr-2" />
              设置
            </UButton>
          </div>
        </div>
      </UCard>

      <!-- 统计信息 -->
      <div class="grid grid-cols-3 gap-4 mb-6">
        <UCard class="text-center">
          <div class="py-4">
            <div class="text-2xl font-bold text-gray-900 mb-1">0</div>
            <div class="text-sm text-gray-600">好友</div>
          </div>
        </UCard>
        <UCard class="text-center">
          <div class="py-4">
            <div class="text-2xl font-bold text-gray-900 mb-1">0</div>
            <div class="text-sm text-gray-600">群组</div>
          </div>
        </UCard>
        <UCard class="text-center">
          <div class="py-4">
            <div class="text-2xl font-bold text-gray-900 mb-1">0</div>
            <div class="text-sm text-gray-600">动态</div>
          </div>
        </UCard>
      </div>

      <!-- 功能菜单 -->
      <UCard>
        <template #header>
          <h3 class="font-semibold">功能菜单</h3>
        </template>
        
        <div class="space-y-1">
          <UButton 
            variant="ghost" 
            block 
            justify="start"
            @click="navigateTo('/groups')"
          >
            <UIcon name="i-heroicons-user-group" class="w-5 h-5 mr-3" />
            我的群组
          </UButton>
          
          <UButton 
            variant="ghost" 
            block 
            justify="start"
            @click="navigateTo('/moments')"
          >
            <UIcon name="i-heroicons-photo" class="w-5 h-5 mr-3" />
            我的动态
          </UButton>
          
          <UButton 
            variant="ghost" 
            block 
            justify="start"
            @click="navigateTo('/settings')"
          >
            <UIcon name="i-heroicons-cog-6-tooth" class="w-5 h-5 mr-3" />
            设置
          </UButton>
          
          <UButton 
            variant="ghost" 
            block 
            justify="start"
            color="red"
            @click="handleLogout"
          >
            <UIcon name="i-heroicons-arrow-right-on-rectangle" class="w-5 h-5 mr-3" />
            退出登录
          </UButton>
        </div>
      </UCard>
    </div>
  </div>
</template>

<script setup>
// 页面元数据
definePageMeta({
  title: '个人资料',
  requiresAuth: true
})

const { logout } = useAuth()
const authStore = useAuthStore()
const toast = useToast()

// 当前用户信息
const user = computed(() => authStore.currentUser)

// 编辑资料
const editProfile = () => {
  toast.add({
    title: '功能开发中',
    description: '编辑资料功能即将上线',
    color: 'info'
  })
}

// 查看设置
const viewSettings = () => {
  navigateTo('/settings')
}

// 退出登录
const handleLogout = async () => {
  try {
    await logout()
    toast.add({
      title: '已退出登录',
      color: 'success'
    })
    await navigateTo('/auth')
  } catch (error) {
    toast.add({
      title: '退出失败',
      description: '请重试',
      color: 'error'
    })
  }
}
</script>
