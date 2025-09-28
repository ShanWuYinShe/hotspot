<template>
  <div class="space-y-4 md:space-y-6" role="group" aria-labelledby="appearance-heading">
    <UCard :ui="cardDefaultUi">
      <template #header>
        <div class="flex items-center gap-4 md:gap-5">
          <div class="p-4 md:p-5 rounded-xl bg-primary-100 dark:bg-primary-900/40 flex-shrink-0">
            <UIcon name="i-heroicons-paint-brush" class="w-7 h-7 md:w-9 md:h-9 text-primary-600 dark:text-primary-400" />
          </div>
          <div class="min-w-0 flex-1">
            <h2 
              id="appearance-heading" 
              tabindex="-1" 
              class="text-lg md:text-xl font-semibold text-neutral-900 dark:text-white focus:outline-none focus:ring-2 focus:ring-primary-500 rounded-sm"
            >
              外观设置
            </h2>
            <p class="text-xs md:text-sm text-neutral-600 dark:text-neutral-400">
              自定义应用的外观和显示偏好
            </p>
          </div>
        </div>
      </template>

      <div class="space-y-3 md:space-y-4 p-1 pb-4 md:pb-6">

        <!-- 语言设置 -->
        <div class="space-y-3 md:space-y-4">
          <h4 class="text-sm md:text-base lg:text-lg font-medium text-neutral-900 dark:text-white">
            语言
          </h4>
          <UFormField label="界面语言" class="space-y-2">
            <USelect 
              v-model="localSettings.appearance.language" 
              :options="languageOptions" 
              class="w-full" 
              size="md" 
            />
          </UFormField>
        </div>

        <USeparator />

        <!-- 其他偏好设置 -->
        <div class="space-y-3 md:space-y-4">
          <h4 class="text-sm md:text-base lg:text-lg font-medium text-neutral-900 dark:text-white">
            其他偏好
          </h4>
          <div class="space-y-3">
            <SettingToggleItem 
              title="动画效果" 
              icon="i-heroicons-sparkles" 
              color="purple" 
              description="启用界面动画和过渡效果" 
              v-model:modelValue="localSettings.appearance.animations" 
            />
            <SettingToggleItem 
              title="紧凑模式" 
              icon="i-heroicons-view-columns" 
              color="blue" 
              description="使用更紧凑的界面布局" 
              v-model:modelValue="localSettings.appearance.compact_mode" 
            />
            <SettingToggleItem 
              title="高对比度" 
              icon="i-heroicons-eye" 
              color="neutral" 
              description="提高界面对比度以改善可读性" 
              v-model:modelValue="localSettings.appearance.high_contrast" 
            />
          </div>
        </div>
      </div>
    </UCard>
  </div>
</template>

<script setup lang="ts">
import type { UserSettings } from '~/types/api';

interface Props {
  localSettings: UserSettings;
}

defineProps<Props>();

// 配置选项

const languageOptions = [
  { label: '简体中文', value: 'zh-CN' },
  { label: 'English', value: 'en-US' }
];

// UI 配置
const cardDefaultUi = {
  root: 'p-4 md:p-6 rounded-xl border border-neutral-200 dark:border-neutral-700 bg-white dark:bg-neutral-800',
  body: 'p-0'
};
</script>