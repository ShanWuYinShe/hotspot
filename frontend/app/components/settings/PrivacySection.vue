<template>
  <div class="space-y-6" role="group" aria-labelledby="privacy-heading">
    <!-- 位置隐私设置 -->
    <UCard :ui="cardHeaderUi">
      <template #header>
        <div class="flex items-center gap-4 md:gap-5">
          <div class="p-4 md:p-5 rounded-xl bg-green-100 dark:bg-green-900/40 flex-shrink-0">
            <UIcon name="i-heroicons-shield-check" class="w-7 h-7 md:w-9 md:h-9 text-green-600 dark:text-green-400" />
          </div>
          <div class="min-w-0 flex-1">
            <h2 
              id="privacy-heading" 
              tabindex="-1" 
              class="text-lg md:text-xl lg:text-2xl font-bold text-neutral-900 dark:text-white focus:outline-none focus:ring-2 focus:ring-green-500 rounded-sm"
            >
              隐私设置
            </h2>
            <p class="text-xs md:text-sm text-neutral-600 dark:text-neutral-400">
              控制您的隐私和数据共享偏好
            </p>
          </div>
        </div>
      </template>
      
      <div class="space-y-3 md:space-y-4 p-1 pb-4 md:pb-6">
        <div class="flex items-center gap-4 md:gap-5 mb-2">
          <div class="p-3 md:p-4 rounded-lg bg-neutral-100 dark:bg-neutral-800 flex-shrink-0">
            <UIcon name="i-heroicons-map-pin" class="w-5 h-5 md:w-6 md:h-6 text-neutral-600 dark:text-neutral-400" />
          </div>
          <h3 class="text-base md:text-lg font-semibold text-neutral-900 dark:text-white">
            位置隐私
          </h3>
        </div>
        
        <div class="space-y-3">
          <SettingToggleItem 
            title="位置共享" 
            icon="i-heroicons-map-pin" 
            color="blue" 
            description="允许其他用户查看我的位置" 
            v-model:modelValue="localSettings.privacy.location_sharing" 
          />
          <SettingToggleItem 
            title="在线状态" 
            icon="i-heroicons-eye" 
            color="green" 
            description="显示我的在线状态给其他用户" 
            v-model:modelValue="localSettings.privacy.online_status" 
          />
          <SettingToggleItem 
            title="搜索可见性" 
            icon="i-heroicons-magnifying-glass" 
            color="purple" 
            description="允许其他用户通过搜索找到我" 
            v-model:modelValue="localSettings.privacy.show_in_search" 
          />
        </div>
      </div>
    </UCard>

    <!-- 资料可见性设置 -->
    <UCard :ui="cardDefaultUi">
      <template #header>
        <div class="flex items-center gap-2">
          <UIcon name="i-heroicons-user-group" class="w-4 h-4 md:w-5 md:h-5 text-neutral-600 dark:text-neutral-400" />
          <h3 class="text-base md:text-lg font-semibold text-neutral-900 dark:text-white">
            资料可见性
          </h3>
        </div>
      </template>
      
      <div class="p-1 pb-4 md:pb-6">
        <UFormField label="谁可以查看我的资料" class="space-y-3">
          <USelect 
            v-model="localSettings.privacy.privacy_level" 
            :options="profileVisibilityOptions" 
            class="w-full" 
            size="md" 
          />
        </UFormField>
      </div>
    </UCard>

    <!-- 安全与举报 -->
    <UCard :ui="cardDefaultUi">
      <template #header>
        <div class="flex items-center gap-2">
          <UIcon name="i-heroicons-shield-exclamation" class="w-4 h-4 md:w-5 md:h-5 text-neutral-600 dark:text-neutral-400" />
          <h3 class="text-base md:text-lg font-semibold text-neutral-900 dark:text-white">
            安全与举报
          </h3>
        </div>
      </template>
      
      <div class="p-1 pb-4 md:pb-6 space-y-4">
        <div class="flex items-center justify-between">
          <div>
            <h4 class="text-sm font-medium text-neutral-900 dark:text-white">
              我的举报记录
            </h4>
            <p class="text-xs text-neutral-600 dark:text-neutral-400">
              查看您提交的举报记录和处理状态
            </p>
          </div>
          <UButton
            variant="outline"
            size="sm"
            icon="i-heroicons-document-text"
            @click="showReportHistory"
          >
            查看记录
          </UButton>
        </div>
        
        <div class="flex items-center justify-between">
          <div>
            <h4 class="text-sm font-medium text-neutral-900 dark:text-white">
              黑名单管理
            </h4>
            <p class="text-xs text-neutral-600 dark:text-neutral-400">
              管理您拉黑的用户列表
            </p>
          </div>
          <UButton
            variant="outline"
            size="sm"
            icon="i-heroicons-user-minus"
            @click="openBlockedUsersModal"
          >
            管理黑名单
          </UButton>
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

// Modal 功能
const { showReportHistoryModal, showBlockedUsersModal } = useModal();

// 配置选项
const profileVisibilityOptions = [
  { label: '公开', value: 'public' },
  { label: '仅好友', value: 'friends' },
  { label: '私密', value: 'private' }
];

// 方法
const showReportHistory = () => {
  showReportHistoryModal();
};

const openBlockedUsersModal = () => {
  showBlockedUsersModal();
};

// UI 配置
const cardHeaderUi = {
  root: 'p-4 md:p-6 rounded-xl border border-green-200 dark:border-green-800/50 bg-gradient-to-br from-green-50 to-emerald-100 dark:from-green-900/20 dark:to-emerald-800/20',
  body: 'p-0'
};

const cardDefaultUi = {
  root: 'p-4 md:p-6 rounded-xl bg-white dark:bg-neutral-800 border border-neutral-200 dark:border-neutral-700',
  body: 'p-0'
};
</script>