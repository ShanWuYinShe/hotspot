<template>
  <UModal 
    title="🚀 开始你的 HotSpot 之旅"
    description="登录或创建临时账户来体验完整功能"
    :close="{ onClick: () => $emit('close') }"
  >
    <template #body>
      <div class="space-y-6">
        <!-- 功能预览 -->
        <div class="grid grid-cols-2 gap-4">
          <div class="text-center p-4 bg-gradient-to-br from-primary-50 to-primary-100 dark:from-primary-900/20 dark:to-primary-800/20 rounded-lg">
            <div class="w-10 h-10 bg-primary-500 rounded-lg flex items-center justify-center mx-auto mb-2">
              <UIcon name="i-heroicons-users" class="w-5 h-5 text-white" />
            </div>
            <p class="text-sm font-medium text-primary-900 dark:text-primary-100">发现附近用户</p>
          </div>
          
          <div class="text-center p-4 bg-gradient-to-br from-success-50 to-success-100 dark:from-success-900/20 dark:to-success-800/20 rounded-lg">
            <div class="w-10 h-10 bg-success-500 rounded-lg flex items-center justify-center mx-auto mb-2">
              <UIcon name="i-heroicons-user-group" class="w-5 h-5 text-white" />
            </div>
            <p class="text-sm font-medium text-success-900 dark:text-success-100">加入群组聊天</p>
          </div>
          
          <div class="text-center p-4 bg-gradient-to-br from-purple-50 to-purple-100 dark:from-purple-900/20 dark:to-purple-800/20 rounded-lg">
            <div class="w-10 h-10 bg-purple-500 rounded-lg flex items-center justify-center mx-auto mb-2">
              <UIcon name="i-heroicons-photo" class="w-5 h-5 text-white" />
            </div>
            <p class="text-sm font-medium text-purple-900 dark:text-purple-100">分享精彩动态</p>
          </div>
          
          <div class="text-center p-4 bg-gradient-to-br from-orange-50 to-orange-100 dark:from-orange-900/20 dark:to-orange-800/20 rounded-lg">
            <div class="w-10 h-10 bg-orange-500 rounded-lg flex items-center justify-center mx-auto mb-2">
              <UIcon name="i-heroicons-map" class="w-5 h-5 text-white" />
            </div>
            <p class="text-sm font-medium text-orange-900 dark:text-orange-100">探索周边热点</p>
          </div>
        </div>

        <!-- 操作按钮 -->
        <div class="space-y-3">
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
          
          <UTooltip text="创建账户或登录现有账户" :kbds="['2']">
            <UButton 
              variant="outline" 
              size="lg" 
              @click="handleLogin"
              class="w-full justify-start"
            >
              <UIcon name="i-heroicons-user-circle" class="w-5 h-5 mr-3" />
              <div class="text-left">
                <div class="font-semibold">登录 / 注册</div>
                <div class="text-sm opacity-75">完整功能，数据永久保存</div>
              </div>
            </UButton>
          </UTooltip>
        </div>

        <!-- 提示信息 -->
        <div class="text-center p-3 bg-neutral-50 dark:bg-neutral-800 rounded-lg">
          <p class="text-xs text-neutral-600 dark:text-neutral-400">
            💡 提示：临时体验可以使用所有功能，数据会在浏览器中保存
          </p>
        </div>
      </div>
    </template>

    <template #footer>
      <div class="flex justify-between items-center w-full">
        <p class="text-xs text-neutral-500">
          使用数字键 1-2 快速选择
        </p>
        <div class="flex space-x-2">
          <UTooltip text="稍后再说，继续浏览" :kbds="['escape']">
            <UButton variant="ghost" size="sm" @click="handleSkip">
              稍后再说
            </UButton>
          </UTooltip>
          <UTooltip text="不再提醒">
            <UButton variant="ghost" size="sm" @click="handleNeverShow">
              不再提醒
            </UButton>
          </UTooltip>
        </div>
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
  'login': [];
  'skip': [];
  'never-show': [];
  'close': [];
}>();

// useOverlay 会自动管理 Modal 状态，无需手动绑定 isOpen

// 处理函数
const handleTempExperience = () => {
  emit('temp-experience');
  emit('close');
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

const handleNeverShow = () => {
  emit('never-show');
  emit('close');
  // 设置永不显示标记
  localStorage.setItem('never-show-login-prompt', 'true');
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

