<template>
  <UModal
    title="🎉 欢迎来到 HotSpot！"
    description="发现身边的精彩，连接周围的朋友。选择你的开始方式："
    :dismissible="false"
  >
    <template #body>
      <nav class="grid grid-cols-1 gap-3">
        <UTooltip
          text="无需注册，立即开始体验所有功能"
          :kbds="['1']"
        >
          <UButton
            class="w-full justify-start"
            leading-icon="i-lucide-rocket"
            @click="handleTempExperience"
          >
            <article class="text-left">
              <p class="font-semibold">临时体验</p>
              <p class="text-sm opacity-75">无需注册，立即开始</p>
            </article>
          </UButton>
        </UTooltip>

        <UTooltip
          text="创建账户，享受完整功能"
          :kbds="['2']"
        >
          <UButton
            variant="outline"
            class="w-full justify-start"
            leading-icon="i-lucide-user-plus"
            @click="handleRegister"
          >
            <article class="text-left">
              <p class="font-semibold">注册账户</p>
              <p class="text-sm opacity-75">完整功能，数据永久保存</p>
            </article>
          </UButton>
        </UTooltip>

        <UTooltip
          text="已有账户？直接登录"
          :kbds="['3']"
        >
          <UButton
            variant="ghost"
            class="w-full justify-start"
            leading-icon="i-lucide-log-in"
            @click="handleLogin"
          >
            <article class="text-left">
              <p class="font-semibold">登录</p>
              <p class="text-sm opacity-75">使用现有账户</p>
            </article>
          </UButton>
        </UTooltip>
      </nav>

      <footer class="mt-6 pt-4">
        <USeparator class="mb-4" />
        <div class="flex items-center justify-center gap-2 flex-wrap">
          <UBadge color="neutral" variant="soft" size="xs" icon="i-lucide-map-pin">位置发现</UBadge>
          <UBadge color="neutral" variant="soft" size="xs" icon="i-lucide-messages-square">实时聊天</UBadge>
          <UBadge color="neutral" variant="soft" size="xs" icon="i-lucide-users-round">群组交流</UBadge>
          <UBadge color="neutral" variant="soft" size="xs" icon="i-lucide-image">动态分享</UBadge>
        </div>
      </footer>
    </template>

    <template #footer>
      <UBadge color="neutral" variant="soft" size="xs">
        使用数字键 1-3 快速选择
      </UBadge>
      <UTooltip
        text="稍后再说，先浏览页面"
        :kbds="['escape']"
      >
        <UButton
          variant="ghost"
          size="sm"
          @click="handleSkip"
        >
          稍后再说
        </UButton>
      </UTooltip>
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
