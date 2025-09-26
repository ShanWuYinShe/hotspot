<template>
  <UModal 
    title="🎉 欢迎来到 HotSpot！"
    description="发现身边的精彩，连接周围的朋友。选择你的开始方式："
    :dismissible="false"
  >
    <template #default></template>

    <template #body>
      <div class="space-y-4">
        <!-- 快速开始选项 -->
        <div class="grid grid-cols-1 gap-3">
          <UTooltip text="无需注册，立即开始体验所有功能" :kbds="['1']">
            <UButton 
              size="lg" 
              @click="handleTempExperience"
              class="w-full justify-start"
            >
              <UIcon name="i-heroicons-rocket-launch" class="w-5 h-5 mr-3" />
              <div class="text-left">
                <div class="font-semibold">临时体验</div>
                <div class="text-sm opacity-75">无需注册，立即开始</div>
              </div>
            </UButton>
          </UTooltip>
          
          <UTooltip text="创建账户，享受完整功能" :kbds="['2']">
            <UButton 
              variant="outline" 
              size="lg" 
              @click="handleRegister"
              class="w-full justify-start"
            >
              <UIcon name="i-heroicons-user-plus" class="w-5 h-5 mr-3" />
              <div class="text-left">
                <div class="font-semibold">注册账户</div>
                <div class="text-sm opacity-75">完整功能，数据永久保存</div>
              </div>
            </UButton>
          </UTooltip>
          
          <UTooltip text="已有账户？直接登录" :kbds="['3']">
            <UButton 
              variant="ghost" 
              size="lg" 
              @click="handleLogin"
              class="w-full justify-start"
            >
              <UIcon name="i-heroicons-arrow-right-end-on-rectangle" class="w-5 h-5 mr-3" />
              <div class="text-left">
                <div class="font-semibold">登录</div>
                <div class="text-sm opacity-75">使用现有账户</div>
              </div>
            </UButton>
          </UTooltip>
        </div>
        
        <!-- 功能预览 -->
        <div class="mt-6 pt-4 border-t border-neutral-200 dark:border-neutral-700">
          <p class="text-sm text-neutral-600 dark:text-neutral-400 text-center mb-3">
            ✨ HotSpot 主要功能
          </p>
          <div class="flex justify-center space-x-6 text-xs text-neutral-500">
            <div class="flex items-center">
              <UIcon name="i-heroicons-map-pin" class="w-4 h-4 mr-1" />
              位置发现
            </div>
            <div class="flex items-center">
              <UIcon name="i-heroicons-chat-bubble-left-right" class="w-4 h-4 mr-1" />
              实时聊天
            </div>
            <div class="flex items-center">
              <UIcon name="i-heroicons-user-group" class="w-4 h-4 mr-1" />
              群组交流
            </div>
            <div class="flex items-center">
              <UIcon name="i-heroicons-photo" class="w-4 h-4 mr-1" />
              动态分享
            </div>
          </div>
        </div>
      </div>
    </template>

    <template #footer>
      <div class="flex justify-between items-center w-full">
        <p class="text-xs text-neutral-500">
          使用数字键 1-3 快速选择
        </p>
        <UTooltip text="稍后再说，先浏览页面" :kbds="['escape']">
          <UButton variant="ghost" size="sm" @click="handleSkip">
            稍后再说
          </UButton>
        </UTooltip>
      </div>
    </template>
  </UModal>
</template>

<script setup lang="ts">
// Props
interface Props {
  open?: boolean;
}

const props = withDefaults(defineProps<Props>(), {
  open: false
});

// Emits
const emit = defineEmits<{
  'update:open': [value: boolean];
  'temp-experience': [];
  'register': [];
  'login': [];
  'skip': [];
  'close': [];
}>();

// useOverlay 会自动管理 Modal 状态，无需手动绑定 isOpen

// 处理函数
const handleTempExperience = () => {
  emit('temp-experience');
  emit('close');
};

const handleRegister = () => {
  emit('register');
  emit('close');
  navigateTo('/auth/login');
};

const handleLogin = () => {
  emit('login');
  emit('close');
  navigateTo('/auth/login');
};

const handleSkip = () => {
  emit('skip');
  emit('close');
};

// 快捷键支持 - 移除 isOpen 检查，useOverlay 会自动管理
const disposeShortcuts = defineShortcuts({
  '1': {
    usingInput: false,
    handler: () => {
      handleTempExperience();
    },
  },
  '2': {
    usingInput: false,
    handler: () => {
      handleRegister();
    },
  },
  '3': {
    usingInput: false,
    handler: () => {
      handleLogin();
    },
  },
});

onUnmounted(() => {
  try { 
    (disposeShortcuts as any)(); 
  } catch { 
    /* noop */ 
  }
});
</script>

