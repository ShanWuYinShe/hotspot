<template>
  <div class="fixed bottom-4 right-4 z-50">
    <UButton
      @click="isOpen = !isOpen"
      size="sm"
      color="neutral"
      variant="solid"
      class="shadow-lg"
      icon="i-heroicons-bug-ant"
    >
      认证调试
    </UButton>
    
    <UCard v-if="isOpen" class="mt-2 w-80 shadow-xl">
      <template #header>
        <div class="flex items-center justify-between">
          <span class="text-sm font-medium">认证状态</span>
          <UButton
            @click="isOpen = false"
            size="xs"
            variant="ghost"
            icon="i-heroicons-x-mark"
          />
        </div>
      </template>
      
      <template #body>
        <div class="space-y-3 text-xs">
          <div class="flex justify-between">
            <span>认证状态:</span>
            <UBadge :color="authStore.isAuthenticated ? 'success' : 'error'" size="xs">
              {{ authStore.isAuthenticated ? '已认证' : '未认证' }}
            </UBadge>
          </div>
          
          <div class="flex justify-between">
            <span>Token:</span>
            <UBadge :color="authStore.token ? 'success' : 'warning'" size="xs">
              {{ authStore.token ? '有效' : '无' }}
            </UBadge>
          </div>
          
          <div class="flex justify-between">
            <span>RefreshToken:</span>
            <UBadge :color="authStore.refreshToken ? 'success' : 'warning'" size="xs">
              {{ authStore.refreshToken ? '有效' : '无' }}
            </UBadge>
          </div>
          
          <div class="flex justify-between">
            <span>会话状态:</span>
            <UBadge :color="authStore.sessionActive ? 'success' : 'neutral'" size="xs">
              {{ authStore.sessionActive ? '活跃' : '非活跃' }}
            </UBadge>
          </div>
          
          <div v-if="authStore.user" class="pt-2 border-t border-neutral-200 dark:border-neutral-700">
            <div class="text-xs font-medium mb-1">用户信息:</div>
            <div class="space-y-1">
              <div>ID: {{ authStore.user.user_id }}</div>
              <div>昵称: {{ authStore.user.nickname }}</div>
              <div>类型: {{ authStore.user.is_temporary ? '临时' : '正式' }}</div>
            </div>
          </div>
          
          <div class="pt-2 border-t border-neutral-200 dark:border-neutral-700">
            <div class="text-xs font-medium mb-1">账户数量: {{ authStore.accounts.length }}</div>
          </div>
        </div>
      </template>
      
      <template #footer>
        <div class="flex gap-2">
          <UButton
            @click="testAuth"
            size="xs"
            variant="outline"
            :loading="isTesting"
          >
            测试认证
          </UButton>
          <UButton
            @click="clearAuth"
            size="xs"
            variant="outline"
            color="error"
          >
            清除认证
          </UButton>
        </div>
      </template>
    </UCard>
  </div>
</template>

<script setup lang="ts">
const authStore = useAuthStore()
const isOpen = ref(false)
const isTesting = ref(false)

const testAuth = async () => {
  try {
    isTesting.value = true
    await authStore.checkAuth()
    console.log('认证测试成功')
  } catch (error) {
    console.error('认证测试失败:', error)
  } finally {
    isTesting.value = false
  }
}

const clearAuth = () => {
  authStore.clearAuth()
  console.log('认证状态已清除')
}
</script>
