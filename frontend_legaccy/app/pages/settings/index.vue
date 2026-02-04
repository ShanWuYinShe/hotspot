<template>
  <div class="w-full max-w-4xl mx-auto pb-20">
    <UTabs v-model="activeTab" :items="tabItems" class="w-full">
      <!-- 账户设置 -->
      <template #account>
        <div class="space-y-6 mt-6">
          <UCard variant="subtle">
            <template #header>
              <h3 class="text-lg font-semibold text-neutral-900 dark:text-white">个人资料</h3>
            </template>
            
            <div class="flex flex-col sm:flex-row items-center gap-6">
              <div class="relative group">
                <UAvatar
                  :src="currentUser?.avatar_url"
                  :alt="currentUser?.nickname"
                  size="3xl"
                  class="ring-4 ring-white dark:ring-neutral-900"
                />
                <UButton
                  icon="i-lucide-camera"
                  size="xs"
                  color="neutral"
                  variant="solid"
                  class="absolute bottom-0 right-0 rounded-full shadow-md"
                  @click="editProfile"
                />
              </div>
              
              <div class="flex-1 text-center sm:text-left space-y-2">
                <div class="text-xl sm:text-2xl font-bold text-neutral-900 dark:text-white">{{ currentUser?.nickname || '未设置昵称' }}</div>
                <div class="flex items-center justify-center sm:justify-start gap-2">
                  <UBadge
                    :label="isTemporaryUser ? '临时用户' : '正式用户'"
                    :color="isTemporaryUser ? 'warning' : 'success'"
                    variant="soft"
                    size="xs"
                  />
                  <UBadge v-if="currentUser?.user_id" color="neutral" variant="soft" size="xs">
                    ID: {{ currentUser.user_id }}
                  </UBadge>
                </div>
              </div>

              <div class="flex flex-col gap-2 w-full sm:w-auto">
                <UButton block icon="i-lucide-edit" @click="editProfile">编辑资料</UButton>
                <UButton v-if="isTemporaryUser" block color="primary" variant="soft" icon="i-lucide-rocket" @click="upgradeAccount">升级账户</UButton>
              </div>
            </div>
          </UCard>

          <UCard v-if="!isTemporaryUser" variant="subtle">
            <template #header>
              <h3 class="text-lg font-semibold text-neutral-900 dark:text-white">账户安全</h3>
            </template>
            <div class="flex items-center justify-between py-2">
              <div>
                <div class="font-semibold text-neutral-900 dark:text-white">登录密码</div>
                <div class="text-sm text-neutral-500 dark:text-neutral-400 mt-0.5">定期更换密码可以提高账户安全性</div>
              </div>
              <UButton color="neutral" variant="outline" size="sm" @click="changePassword">修改密码</UButton>
            </div>
          </UCard>
        </div>
      </template>

      <!-- 隐私设置 -->
      <template #privacy>
        <div class="space-y-6 mt-6">
          <UCard variant="subtle">
            <template #header>
              <h3 class="text-lg font-semibold text-neutral-900 dark:text-white">隐私可见性</h3>
            </template>
            <div class="divide-y divide-neutral-100 dark:divide-neutral-800">
              <SettingToggleItem
                title="位置共享"
                icon="i-lucide-map-pin"
                description="允许其他用户查看我的位置"
                :model-value="privacySettings.location_visible"
                @update:model-value="updatePrivacy('location_visible', $event)"
              />
              <SettingToggleItem
                title="在线状态"
                icon="i-lucide-signal"
                description="允许其他用户查看我的在线状态"
                :model-value="privacySettings.online_visible"
                @update:model-value="updatePrivacy('online_visible', $event)"
              />
            </div>
          </UCard>

          <UCard variant="subtle">
            <template #header>
              <h3 class="text-lg font-semibold text-neutral-900 dark:text-white">社交权限</h3>
            </template>
            <div class="space-y-4">
              <UFormField label="谁可以私聊我">
                <USelect
                  :model-value="privacySettings.allow_private_chat"
                  :items="chatPermissionOptions"
                  class="w-full sm:w-64"
                  @update:model-value="(v) => v && updatePrivacy('allow_private_chat', v)"
                />
              </UFormField>
              <div class="divide-y divide-neutral-100 dark:divide-neutral-800">
                <SettingToggleItem
                  title="允许被关注"
                  icon="i-lucide-user-plus"
                  description="允许其他用户关注我"
                  :model-value="privacySettings.allow_follow"
                  @update:model-value="updatePrivacy('allow_follow', $event)"
                />
                <SettingToggleItem
                  title="允许被搜索"
                  icon="i-lucide-search"
                  description="允许通过昵称或ID搜索到我"
                  :model-value="privacySettings.searchable"
                  @update:model-value="updatePrivacy('searchable', $event)"
                />
              </div>
            </div>
          </UCard>
        </div>
      </template>

      <!-- 通知设置 -->
      <template #notifications>
        <div class="space-y-6 mt-6">
          <UCard variant="subtle">
            <template #header>
              <h3 class="text-lg font-semibold text-neutral-900 dark:text-white">通知偏好</h3>
            </template>
            <div class="divide-y divide-neutral-100 dark:divide-neutral-800">
              <SettingToggleItem
                title="新关注者"
                icon="i-lucide-user-plus"
                :model-value="notificationSettings.new_follower_enabled"
                @update:model-value="updateNotifications('new_follower_enabled', $event)"
              />
              <SettingToggleItem
                title="消息通知"
                icon="i-lucide-message-circle"
                :model-value="notificationSettings.message_mention_enabled"
                @update:model-value="updateNotifications('message_mention_enabled', $event)"
              />
              <SettingToggleItem
                title="动态互动"
                icon="i-lucide-heart"
                description="点赞和评论通知"
                :model-value="notificationSettings.moment_like_enabled"
                @update:model-value="updateNotifications('moment_like_enabled', $event)"
              />
              <SettingToggleItem
                title="群组通知"
                icon="i-lucide-users"
                description="邀请和申请通知"
                :model-value="notificationSettings.group_invite_enabled"
                @update:model-value="updateNotifications('group_invite_enabled', $event)"
              />
            </div>
          </UCard>
        </div>
      </template>

      <!-- 外观设置 -->
      <template #appearance>
        <div class="space-y-6 mt-6">
          <UCard variant="subtle">
            <template #header>
              <h3 class="text-lg font-semibold text-neutral-900 dark:text-white">界面偏好</h3>
            </template>
            <div class="divide-y divide-neutral-100 dark:divide-neutral-800">
              <SettingToggleItem
                title="动画效果"
                icon="i-lucide-sparkles"
                description="启用界面动画和过渡效果"
                :model-value="settingsStore.enableAnimations"
                @update:model-value="settingsStore.setEnableAnimations($event)"
              />
              <SettingToggleItem
                title="紧凑模式"
                icon="i-lucide-layout-list"
                description="更紧凑的界面布局"
                :model-value="settingsStore.compactMode"
                @update:model-value="settingsStore.setCompactMode($event)"
              />
            </div>
          </UCard>
        </div>
      </template>

      <!-- 其他设置 -->
      <template #about>
        <div class="space-y-6 mt-6">
          <UCard variant="subtle" class="text-center py-8">
            <UAvatar icon="i-lucide-flame" size="3xl" class="mx-auto mb-4" />
            <h2 class="text-xl sm:text-2xl font-bold text-neutral-900 dark:text-white mb-1">HotSpot</h2>
            <p class="text-neutral-500 dark:text-neutral-400 text-sm mb-6">v1.0.0 · 基于位置的社交平台</p>
            
            <div class="max-w-xs mx-auto space-y-3">
              <UButton block variant="soft" color="neutral" icon="i-lucide-trash-2" @click="clearCache">清除缓存</UButton>
              <UButton block variant="ghost" color="neutral" icon="i-lucide-file-text" @click="showPrivacyPolicy">隐私政策</UButton>
            </div>
          </UCard>
          
          <div class="text-center">
            <UButton color="error" variant="link" size="sm" icon="i-lucide-log-out" @click="deleteAccount">注销账户</UButton>
          </div>
        </div>
      </template>
    </UTabs>
  </div>
</template>

<script setup lang="ts">
import { useUsersService } from '~/services/users.service'
import { useSubscriptionsService } from '~/services/subscriptions.service'
import { isTempUser } from '~/utils/userTypes'
import { useSettingsStore } from '~/stores/settings.store'
import SettingToggleItem from '~/components/settings/SettingToggleItem.vue'
import type { SubscriptionsData, UserSettingsData } from '~/types/api'

definePageMeta({ 
    title: '设置', 
    layout: 'dashboard',
    requiresAuth: true  // 设置页面必须登录才能访问
})

const authStore = useAuthStore()
const settingsStore = useSettingsStore()
const feedback = useFeedbackStore()
const modal = useModal()
const userService = useUsersService()
const subscriptionsService = useSubscriptionsService()

const activeTab = ref('account')
const currentUser = useHydratedStore(authStore, 'user', null)
const isTemporaryUser = computed(() => currentUser.value?.user_type ? isTempUser(currentUser.value.user_type) : false)

const tabItems = [
  { slot: 'account', label: '账户', icon: 'i-lucide-user', value: 'account' },
  { slot: 'privacy', label: '隐私', icon: 'i-lucide-lock', value: 'privacy' },
  { slot: 'notifications', label: '通知', icon: 'i-lucide-bell', value: 'notifications' },
  { slot: 'appearance', label: '外观', icon: 'i-lucide-palette', value: 'appearance' },
  { slot: 'about', label: '关于', icon: 'i-lucide-info', value: 'about' }
]

const DEFAULT_PRIVACY: UserSettingsData = {
  privacy_level: 'public',
  location_visible: true,
  online_visible: true,
  last_online_visible: true,
  friends_visible: 'public',
  following_visible: 'public',
  followers_visible: 'public',
  moments_visible: 'public',
  stats_visible: 'public',
  allow_follow: true,
  allow_private_chat: 'everyone',
  searchable: true,
  phone_searchable: true,
  nickname_searchable: true,
  location_searchable: true
}

const DEFAULT_NOTIFICATIONS: SubscriptionsData = {
  new_follower_enabled: true,
  mutual_follow_enabled: true,
  group_invite_enabled: true,
  group_invite_response_enabled: true,
  group_application_enabled: true,
  group_application_response_enabled: true,
  message_mention_enabled: true,
  moment_like_enabled: true,
  moment_comment_enabled: true,
  comment_reply_enabled: true,
  system_announcement_enabled: true
}

const privacySettings = useHydratedStore(settingsStore, 'privacy', DEFAULT_PRIVACY)
const notificationSettings = useHydratedStore(settingsStore, 'notifications', DEFAULT_NOTIFICATIONS)

const chatPermissionOptions = [
  { label: '所有人', value: 'everyone' },
  { label: '仅好友', value: 'friends' },
  { label: '禁止私聊', value: 'nobody' }
]

const updatePrivacy = useDebounceFn(async (key: string, value: unknown) => {
  settingsStore.updatePrivacy({ [key]: value })
  try {
    await userService.updateSettings({ ...settingsStore.privacy })
  } catch {
    feedback.error('保存失败')
  }
}, 1000)

const updateNotifications = useDebounceFn(async (key: string, value: boolean) => {
  settingsStore.updateNotifications({ [key]: value })
  try {
    await subscriptionsService.update({ ...settingsStore.notifications })
  } catch {
    feedback.error('保存失败')
  }
}, 1000)

const editProfile = () => modal.showEditProfile()
const upgradeAccount = () => {
  void navigateTo('/auth?mode=register&upgrade=1')
}
const changePassword = () => modal.showChangePassword()
const clearCache = async () => {
  if (await modal.showConfirm({ title: '清除缓存', message: '确定要清除应用缓存吗？', confirmColor: 'error' })) {
    localStorage.clear()
    location.reload()
  }
}
const showPrivacyPolicy = () => {
  feedback.info('暂未开放')
}
const deleteAccount = () => modal.showDeleteAccount()

onMounted(async () => {
  if (!settingsStore.remoteLoaded) {
    try {
      const [p, n] = await Promise.all([userService.getSettings(), subscriptionsService.getConfig()])
      settingsStore.setPrivacy(p)
      settingsStore.setNotifications(n)
      settingsStore.markRemoteLoaded()
    } catch {}
  }
})
</script>
