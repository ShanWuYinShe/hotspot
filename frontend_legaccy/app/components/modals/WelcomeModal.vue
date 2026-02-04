<template>
  <UModal
    title="🎉 欢迎来到 HotSpot！"
    description="发现身边的精彩，连接周围的朋友。选择你的开始方式："
    :dismissible="false"
  >
    <template #body>
      <div class="p-6">
        <nav class="grid grid-cols-1 gap-4">
          <UTooltip
          text="无需注册，立即开始体验所有功能"
          :kbds="['1']"
          placement="left"
        >
          <UCard
            variant="outline"
            @click="handleTempExperience"
          >
            <template #body>
              <div class="flex items-center gap-4">
                <UAvatar
                  icon="i-lucide-rocket"
                  size="md"
                  color="primary"
                  variant="soft"
                />
                <div class="flex-1">
                  <h3 class="font-semibold text-neutral-900 dark:text-white">临时体验</h3>
                  <p class="text-sm text-neutral-500 dark:text-neutral-400">无需注册，立即开始</p>
                </div>
                <UIcon name="i-lucide-chevron-right" />
              </div>
            </template>
          </UCard>
        </UTooltip>

        <UTooltip
          text="创建账户，享受完整功能"
          :kbds="['2']"
          placement="left"
        >
          <UCard
            variant="outline"
            @click="handleRegister"
          >
            <template #body>
              <div class="flex items-center gap-4">
                <UAvatar
                  icon="i-lucide-user-plus"
                  size="md"
                  color="primary"
                  variant="soft"
                />
                <div class="flex-1">
                  <h3 class="font-semibold text-neutral-900 dark:text-white">注册账户</h3>
                  <p class="text-sm text-neutral-500 dark:text-neutral-400">完整功能，数据永久保存</p>
                </div>
                <UIcon name="i-lucide-chevron-right" />
              </div>
            </template>
          </UCard>
        </UTooltip>

        <UTooltip
          text="已有账户？直接登录"
          :kbds="['3']"
          placement="left"
        >
          <UCard
            variant="outline"
            @click="handleLogin"
          >
            <template #body>
              <div class="flex items-center gap-4">
                <UAvatar
                  icon="i-lucide-log-in"
                  size="md"
                  color="primary"
                  variant="soft"
                />
                <div class="flex-1">
                  <h3 class="font-semibold text-neutral-900 dark:text-white">登录</h3>
                  <p class="text-sm text-neutral-500 dark:text-neutral-400">使用现有账户</p>
                </div>
                <UIcon name="i-lucide-chevron-right" />
              </div>
            </template>
          </UCard>
        </UTooltip>
      </nav>

      <div class="mt-8">
        <div class="flex items-center gap-4 justify-center flex-wrap opacity-75">
          <div class="flex items-center gap-1.5 text-xs text-neutral-500 dark:text-neutral-400">
            <UIcon name="i-lucide-map-pin" size="14" />
            <span>位置发现</span>
          </div>
          <div class="flex items-center gap-1.5 text-xs text-neutral-500 dark:text-neutral-400">
            <UIcon name="i-lucide-messages-square" size="14" />
            <span>实时聊天</span>
          </div>
          <div class="flex items-center gap-1.5 text-xs text-neutral-500 dark:text-neutral-400">
            <UIcon name="i-lucide-users-round" size="14" />
            <span>群组交流</span>
          </div>
          <div class="flex items-center gap-1.5 text-xs text-neutral-500 dark:text-neutral-400">
            <UIcon name="i-lucide-image" size="14" />
            <span>动态分享</span>
          </div>
        </div>
      </div>
    </template>

    <template #footer>
      <div class="flex items-center justify-between w-full">
        <span class="text-xs text-neutral-400 dark:text-neutral-500">
          使用数字键 1-3 快速选择
        </span>
        <UTooltip
          text="稍后再说，先浏览页面"
          :kbds="['escape']"
        >
          <UButton
            variant="ghost"
            color="neutral"
            size="sm"
            @click="handleSkip"
          >
            稍后再说
          </UButton>
        </UTooltip>
      </div>
    </template>
  </UModal>
</template>

<script setup lang="ts">
const emit = defineEmits<{
  'temp-experience': []
  'register': []
  'login': []
  'skip': []
  'close': []
}>()

const handleTempExperience = () => {
  emit('temp-experience')
  emit('close')
}

const handleRegister = () => {
  emit('register')
  emit('close')
  navigateTo('/auth?mode=register')
}

const handleLogin = () => {
  emit('login')
  emit('close')
  navigateTo('/auth?mode=login')
}

const handleSkip = () => {
  emit('skip')
  emit('close')
}

defineShortcuts({
  '1': { usingInput: false, handler: handleTempExperience },
  '2': { usingInput: false, handler: handleRegister },
  '3': { usingInput: false, handler: handleLogin }
})
</script>
