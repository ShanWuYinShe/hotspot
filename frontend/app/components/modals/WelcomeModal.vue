<template>
  <UModal 
    :ui="{
      width: 'sm:max-w-md'
    }"
  >
    <template #header>
      <div class="text-center w-full">
        <div class="w-14 h-14 bg-gradient-to-br from-primary-500 to-blue-600 rounded-2xl flex items-center justify-center mx-auto mb-3 shadow-lg">
          <UIcon name="i-heroicons-fire" class="w-7 h-7 text-white" />
        </div>
        <h2 class="text-xl font-bold text-neutral-900 dark:text-white">
          欢迎使用 HotSpot
        </h2>
        <p class="text-sm text-neutral-600 dark:text-neutral-400 mt-1">
          发现身边的精彩，连接周围的朋友
        </p>
      </div>
    </template>

    <template #body>
      <div class="space-y-3">
        <!-- 临时体验 -->
        <button 
          @click="handleTempExperience"
          class="w-full flex items-center gap-4 p-4 rounded-xl bg-gradient-to-r from-primary-500 to-blue-600 text-white hover:opacity-90 transition-opacity"
        >
          <div class="w-10 h-10 bg-white/20 rounded-lg flex items-center justify-center">
            <UIcon name="i-heroicons-rocket-launch" class="w-5 h-5" />
          </div>
          <div class="text-left flex-1">
            <div class="font-semibold">立即体验</div>
            <div class="text-sm opacity-80">无需注册，一键开始</div>
          </div>
          <UIcon name="i-heroicons-arrow-right" class="w-5 h-5" />
        </button>
        
        <!-- 注册 -->
        <button 
          @click="handleRegister"
          class="w-full flex items-center gap-4 p-4 rounded-xl border border-neutral-200 dark:border-neutral-700 bg-white dark:bg-neutral-800 hover:border-primary-300 dark:hover:border-primary-600 transition-colors"
        >
          <div class="w-10 h-10 bg-primary-100 dark:bg-primary-900/30 rounded-lg flex items-center justify-center">
            <UIcon name="i-heroicons-user-plus" class="w-5 h-5 text-primary-600 dark:text-primary-400" />
          </div>
          <div class="text-left flex-1">
            <div class="font-semibold text-neutral-900 dark:text-white">注册账户</div>
            <div class="text-sm text-neutral-500 dark:text-neutral-400">享受完整功能</div>
          </div>
          <UIcon name="i-heroicons-chevron-right" class="w-5 h-5 text-neutral-400" />
        </button>
        
        <!-- 登录 -->
        <button 
          @click="handleLogin"
          class="w-full flex items-center gap-4 p-4 rounded-xl border border-neutral-200 dark:border-neutral-700 bg-white dark:bg-neutral-800 hover:border-primary-300 dark:hover:border-primary-600 transition-colors"
        >
          <div class="w-10 h-10 bg-neutral-100 dark:bg-neutral-700 rounded-lg flex items-center justify-center">
            <UIcon name="i-heroicons-arrow-right-on-rectangle" class="w-5 h-5 text-neutral-600 dark:text-neutral-400" />
          </div>
          <div class="text-left flex-1">
            <div class="font-semibold text-neutral-900 dark:text-white">已有账户</div>
            <div class="text-sm text-neutral-500 dark:text-neutral-400">直接登录</div>
          </div>
          <UIcon name="i-heroicons-chevron-right" class="w-5 h-5 text-neutral-400" />
        </button>
      </div>
    </template>

    <template #footer>
      <div class="flex justify-center w-full">
        <UButton variant="ghost" size="sm" color="neutral" @click="handleSkip">
          稍后再说
        </UButton>
      </div>
    </template>
  </UModal>
</template>

<script setup lang="ts">
const overlay = useOverlay()
const authStore = useAuthStore()
const toast = useToast()

// 处理临时体验 - 创建临时用户
const handleTempExperience = async () => {
  try {
    await authStore.createTempUser({ nickname: '探索者' })
    toast.add({
      title: '欢迎！',
      description: '临时账户已创建，开始探索吧',
      color: 'success'
    })
    overlay.close()
  } catch (error) {
    console.error('创建临时用户失败:', error)
    toast.add({
      title: '创建失败',
      description: '请稍后重试',
      color: 'error'
    })
  }
}

const handleRegister = () => {
  overlay.close()
  navigateTo('/auth/register')
}

const handleLogin = () => {
  overlay.close()
  navigateTo('/auth/login')
}

const handleSkip = () => {
  overlay.close()
}

// 快捷键
defineShortcuts({
  '1': { usingInput: false, handler: handleTempExperience },
  '2': { usingInput: false, handler: handleRegister },
  '3': { usingInput: false, handler: handleLogin },
  'escape': { usingInput: false, handler: handleSkip }
})
</script>
