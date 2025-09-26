<template>
  <div class="space-y-6" role="group" aria-labelledby="notifications-heading">
    <UCard :ui="cardHeaderUi">
      <template #header>
        <div class="flex items-center gap-4 md:gap-5">
          <div class="p-4 md:p-5 rounded-xl bg-yellow-100 dark:bg-yellow-900/40 flex-shrink-0">
            <UIcon name="i-heroicons-bell" class="w-7 h-7 md:w-9 md:h-9 text-yellow-600 dark:text-yellow-400" />
          </div>
          <div class="min-w-0 flex-1">
            <h2 
              id="notifications-heading" 
              tabindex="-1" 
              class="text-lg md:text-xl lg:text-2xl font-bold text-neutral-900 dark:text-white focus:outline-none focus:ring-2 focus:ring-yellow-500 rounded-sm"
            >
              通知设置
            </h2>
            <p class="text-xs md:text-sm text-neutral-600 dark:text-neutral-400">
              管理您接收通知的方式和频率
            </p>
          </div>
        </div>
      </template>
      
      <div class="space-y-3 md:space-y-4 p-1 pb-4 md:pb-6">
        <!-- 推送通知 -->
        <div class="flex items-center gap-4 md:gap-5 mb-2">
          <div class="p-3 md:p-4 rounded-lg bg-neutral-100 dark:bg-neutral-800 flex-shrink-0">
            <UIcon name="i-heroicons-device-phone-mobile" class="w-5 h-5 md:w-6 md:h-6 text-neutral-600 dark:text-neutral-400" />
          </div>
          <h3 class="text-base md:text-lg font-semibold text-neutral-900 dark:text-white">
            推送通知
          </h3>
        </div>
        
        <div class="space-y-3">
          <SettingToggleItem 
            title="消息通知" 
            icon="i-heroicons-chat-bubble-left-right" 
            color="blue" 
            description="接收新消息的推送通知" 
            v-model:modelValue="localSettings.notifications.message_notifications" 
          />
          <SettingToggleItem 
            title="群组通知" 
            icon="i-heroicons-user-group" 
            color="green" 
            description="接收群组活动的推送通知" 
            v-model:modelValue="localSettings.notifications.group_notifications" 
          />
          <SettingToggleItem 
            title="好友请求" 
            icon="i-heroicons-user-plus" 
            color="neutral" 
            description="接收好友请求的推送通知" 
            v-model:modelValue="localSettings.notifications.friend_requests" 
          />
        </div>
        
        <USeparator />
        
        <!-- 应用内通知 -->
        <div class="space-y-3 md:space-y-4">
          <h4 class="text-sm md:text-base lg:text-lg font-medium text-neutral-900 dark:text-white">
            应用内通知
          </h4>
          <div class="space-y-3">
            <SettingToggleItem 
              title="私聊消息提示" 
              icon="i-heroicons-chat-bubble-oval-left" 
              color="neutral" 
              description="在应用内显示私聊消息弹窗" 
              v-model:modelValue="localSettings.notifications.direct_message_toast" 
            />
            <SettingToggleItem 
              title="群组消息提示" 
              icon="i-heroicons-users" 
              color="neutral" 
              description="在应用内显示群组消息弹窗" 
              v-model:modelValue="localSettings.notifications.group_message_toast" 
            />
            <SettingToggleItem 
              title="系统通知" 
              icon="i-heroicons-exclamation-triangle" 
              color="neutral" 
              description="显示系统重要通知" 
              v-model:modelValue="localSettings.notifications.system_notification_toast" 
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

// UI 配置
const cardHeaderUi = {
  root: 'p-4 md:p-6 rounded-xl border border-yellow-200 dark:border-yellow-800/50 bg-gradient-to-br from-yellow-50 to-orange-100 dark:from-yellow-900/20 dark:to-orange-800/20',
  body: 'p-0'
};
</script>