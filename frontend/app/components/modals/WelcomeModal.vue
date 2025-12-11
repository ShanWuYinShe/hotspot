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
            size="lg"
            class="w-full justify-start"
            @click="handleTempExperience"
          >
            <UIcon
              name="i-heroicons-rocket-launch"
              class="w-5 h-5 mr-3"
            />
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
            size="lg"
            class="w-full justify-start"
            @click="handleRegister"
          >
            <UIcon
              name="i-heroicons-user-plus"
              class="w-5 h-5 mr-3"
            />
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
            size="lg"
            class="w-full justify-start"
            @click="handleLogin"
          >
            <UIcon
              name="i-heroicons-arrow-right-end-on-rectangle"
              class="w-5 h-5 mr-3"
            />
            <article class="text-left">
              <p class="font-semibold">登录</p>
              <p class="text-sm opacity-75">使用现有账户</p>
            </article>
          </UButton>
        </UTooltip>
      </nav>

      <footer class="mt-6 pt-4 border-t border-default">
        <p class="text-sm text-secondary-content text-center mb-3">
          ✨ HotSpot 主要功能
        </p>
        <ul class="flex justify-center gap-6 text-xs text-secondary-content">
          <li class="flex items-center">
            <UIcon
              name="i-heroicons-map-pin"
              class="w-4 h-4 mr-1"
            />
            位置发现
          </li>
          <li class="flex items-center">
            <UIcon
              name="i-heroicons-chat-bubble-left-right"
              class="w-4 h-4 mr-1"
            />
            实时聊天
          </li>
          <li class="flex items-center">
            <UIcon
              name="i-heroicons-user-group"
              class="w-4 h-4 mr-1"
            />
            群组交流
          </li>
          <li class="flex items-center">
            <UIcon
              name="i-heroicons-photo"
              class="w-4 h-4 mr-1"
            />
            动态分享
          </li>
        </ul>
      </footer>
    </template>

    <template #footer>
      <p class="text-xs text-secondary-content">
        使用数字键 1-3 快速选择
      </p>
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
  navigateTo('/auth/register')
}

const handleLogin = () => {
  emit('login')
  emit('close')
  navigateTo('/auth/login')
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
