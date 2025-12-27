<template>
  <div class="w-full">
    <!-- 设置分类标签页 -->
    <UTabs
      v-model="activeTab"
      :items="tabItems"
      class="mb-6"
    />

      <!-- 账户设置 -->
      <div
        v-show="activeTab === 'account'"
        class="space-y-6"
      >
        <!-- 个人资料 -->
        <UCard variant="subtle">
          <template #header>
            <div class="flex items-center gap-3">
              <UBadge color="info" variant="soft" size="lg" square icon="i-lucide-circle-user" />
              <span class="text-sm font-semibold">个人资料</span>
            </div>
          </template>

          <div class="flex items-center gap-4">
            <div class="relative">
              <UAvatar
                :src="currentUser?.avatar_url"
                :alt="currentUser?.nickname"
                size="xl"
              />
              <UButton
                icon="i-lucide-camera"
                size="xs"
                color="primary"
                class="absolute -bottom-1 -right-1"
                @click="editProfile"
              />
            </div>
            <div class="flex-1 min-w-0">
              <div class="text-lg font-semibold truncate">{{ currentUser?.nickname || '未设置昵称' }}</div>
              <div class="mt-1">
                <UBadge
                  :label="isTemporaryUser ? '临时用户' : '正式用户'"
                  :color="isTemporaryUser ? 'warning' : 'success'"
                  variant="soft"
                  size="xs"
                />
              </div>
            </div>
            <div class="flex gap-2">
              <UButton
                variant="outline"
                size="sm"
                color="neutral"
                @click="editProfile"
              >编辑资料</UButton>
              <UButton
                v-if="isTemporaryUser"
                size="sm"
                color="primary"
                @click="upgradeAccount"
              >升级账户</UButton>
            </div>
          </div>
        </UCard>

        <!-- 账户安全 -->
        <UCard v-if="!isTemporaryUser" variant="subtle">
          <template #header>
            <div class="flex items-center gap-3">
              <UBadge color="success" variant="soft" size="lg" square icon="i-lucide-shield-check" />
              <span class="text-sm font-semibold">账户安全</span>
            </div>
          </template>

          <div class="space-y-4">
            <div class="flex items-center justify-between">
              <div>
                <div class="font-medium">登录密码</div>
                <UBadge color="neutral" variant="soft" size="xs" class="mt-1">
                  定期更换密码可以提高账户安全性
                </UBadge>
              </div>
              <UButton
                variant="outline"
                size="sm"
                color="neutral"
                @click="changePassword"
              >修改密码</UButton>
            </div>
          </div>
        </UCard>
      </div>

      <!-- 隐私设置 -->
      <div
        v-show="activeTab === 'privacy'"
        class="space-y-6"
      >
        <!-- 隐私设置 -->
        <UCard variant="subtle">
          <template #header>
            <div class="flex items-center gap-3">
              <UBadge color="success" variant="soft" size="lg" square icon="i-lucide-shield-check" />
              <span class="text-sm font-semibold">隐私设置</span>
            </div>
          </template>

          <div class="space-y-4">
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
            <SettingToggleItem
              title="最后上线时间"
              icon="i-lucide-clock"
              description="允许其他用户查看我的最后上线时间"
              :model-value="privacySettings.last_online_visible"
              @update:model-value="updatePrivacy('last_online_visible', $event)"
            />
          </div>
        </UCard>

        <!-- 社交设置 -->
        <UCard variant="subtle">
          <template #header>
            <div class="flex items-center gap-3">
              <UBadge color="info" variant="soft" size="lg" square icon="i-lucide-users" />
              <span class="text-sm font-semibold">社交设置</span>
            </div>
          </template>

          <div class="space-y-4">
            <UFormField label="谁可以向我发起私聊">
              <USelect
                :model-value="privacySettings.allow_private_chat"
                :items="chatPermissionOptions"
                class="w-full max-w-xs"
                @update:model-value="(v) => v && updatePrivacy('allow_private_chat', v as 'everyone' | 'friends' | 'nobody')"
              />
            </UFormField>
            <SettingToggleItem
              title="允许关注"
              icon="i-lucide-user-plus"
              description="允许其他用户关注我"
              :model-value="privacySettings.allow_follow"
              @update:model-value="updatePrivacy('allow_follow', $event)"
            />
          </div>
        </UCard>

        <!-- 资料可见性 -->
        <UCard variant="subtle">
          <template #header>
            <div class="flex items-center gap-3">
              <UBadge color="neutral" variant="soft" size="lg" square icon="i-lucide-eye" />
              <span class="text-sm font-semibold">资料可见性</span>
            </div>
          </template>

          <div class="space-y-4">
            <UFormField label="谁可以查看我的资料">
              <USelect
                :model-value="privacySettings.privacy_level"
                :items="privacyOptions"
                class="w-full max-w-xs"
                @update:model-value="(v) => v && updatePrivacy('privacy_level', v as 'public' | 'friends' | 'private')"
              />
            </UFormField>
            <UFormField label="谁可以查看我的好友列表">
              <USelect
                :model-value="privacySettings.friends_visible"
                :items="privacyOptions"
                class="w-full max-w-xs"
                @update:model-value="(v) => v && updatePrivacy('friends_visible', v as 'public' | 'friends' | 'private')"
              />
            </UFormField>
            <UFormField label="谁可以查看我的关注列表">
              <USelect
                :model-value="privacySettings.following_visible"
                :items="privacyOptions"
                class="w-full max-w-xs"
                @update:model-value="(v) => v && updatePrivacy('following_visible', v as 'public' | 'friends' | 'private')"
              />
            </UFormField>
            <UFormField label="谁可以查看我的粉丝列表">
              <USelect
                :model-value="privacySettings.followers_visible"
                :items="privacyOptions"
                class="w-full max-w-xs"
                @update:model-value="(v) => v && updatePrivacy('followers_visible', v as 'public' | 'friends' | 'private')"
              />
            </UFormField>
            <UFormField label="谁可以查看我的动态">
              <USelect
                :model-value="privacySettings.moments_visible"
                :items="privacyOptions"
                class="w-full max-w-xs"
                @update:model-value="(v) => v && updatePrivacy('moments_visible', v as 'public' | 'friends' | 'private')"
              />
            </UFormField>
            <UFormField label="谁可以查看我的统计数据">
              <USelect
                :model-value="privacySettings.stats_visible"
                :items="privacyOptions"
                class="w-full max-w-xs"
                @update:model-value="(v) => v && updatePrivacy('stats_visible', v as 'public' | 'friends' | 'private')"
              />
            </UFormField>
          </div>
        </UCard>

        <!-- 搜索设置 -->
        <UCard variant="subtle">
          <template #header>
            <div class="flex items-center gap-3">
              <UBadge color="neutral" variant="soft" size="lg" square icon="i-lucide-search" />
              <span class="text-sm font-semibold">搜索设置</span>
            </div>
          </template>

          <div class="space-y-4">
            <SettingToggleItem
              title="允许被搜索"
              icon="i-lucide-search"
              description="总开关：允许其他用户通过搜索找到我"
              :model-value="privacySettings.searchable"
              @update:model-value="updatePrivacy('searchable', $event)"
            />
            <SettingToggleItem
              title="昵称搜索"
              icon="i-lucide-user"
              description="允许通过昵称搜索到我"
              :model-value="privacySettings.nickname_searchable"
              @update:model-value="updatePrivacy('nickname_searchable', $event)"
            />
            <SettingToggleItem
              title="手机号搜索"
              icon="i-lucide-phone"
              description="允许通过手机号搜索到我"
              :model-value="privacySettings.phone_searchable"
              @update:model-value="updatePrivacy('phone_searchable', $event)"
            />
            <SettingToggleItem
              title="位置搜索"
              icon="i-lucide-map-pin"
              description="在附近的人列表中显示我"
              :model-value="privacySettings.location_searchable"
              @update:model-value="updatePrivacy('location_searchable', $event)"
            />
          </div>
        </UCard>

        <!-- 安全管理 -->
        <UCard variant="subtle">
          <template #header>
            <div class="flex items-center gap-3">
              <UBadge color="neutral" variant="soft" size="lg" square icon="i-lucide-shield-alert" />
              <span class="text-sm font-semibold">安全管理</span>
            </div>
          </template>

          <div class="space-y-4">
            <div class="flex items-center justify-between">
              <div>
                <div class="font-medium">我的举报记录</div>
                <UBadge color="neutral" variant="soft" size="xs" class="mt-1">
                  查看您提交的举报记录和处理状态
                </UBadge>
              </div>
              <UButton
                variant="outline"
                size="sm"
                icon="i-lucide-file-text"
                color="neutral"
                @click="showReportHistory"
              >查看记录
              </UButton>
            </div>
            <USeparator />
            <div class="flex items-center justify-between">
              <div>
                <div class="font-medium">黑名单管理</div>
                <UBadge color="neutral" variant="soft" size="xs" class="mt-1">
                  管理您拉黑的用户列表
                </UBadge>
              </div>
              <UButton
                variant="outline"
                size="sm"
                icon="i-lucide-user-minus"
                color="neutral"
                @click="showBlockedUsers"
              >管理黑名单
              </UButton>
            </div>
          </div>
        </UCard>
      </div>

      <!-- 通知设置 -->
      <div
        v-show="activeTab === 'notifications'"
        class="space-y-6"
      >
        <!-- 社交通知 -->
        <UCard variant="subtle">
          <template #header>
            <div class="flex items-center gap-3">
              <UBadge color="warning" variant="soft" size="lg" square icon="i-lucide-bell" />
              <span class="text-sm font-semibold">社交通知</span>
            </div>
          </template>

          <div class="space-y-4">
            <SettingToggleItem
              title="新关注者"
              icon="i-lucide-user-plus"
              description="当有人关注您时接收通知"
              :model-value="notificationSettings.new_follower_enabled"
              @update:model-value="updateNotifications('new_follower_enabled', $event)"
            />
            <SettingToggleItem
              title="互相关注"
              icon="i-lucide-heart"
              description="当互相关注成为好友时接收通知"
              :model-value="notificationSettings.mutual_follow_enabled"
              @update:model-value="updateNotifications('mutual_follow_enabled', $event)"
            />
          </div>
        </UCard>

        <!-- 群组通知 -->
        <UCard variant="subtle">
          <template #header>
            <div class="flex items-center gap-3">
              <UBadge color="info" variant="soft" size="lg" square icon="i-lucide-users-round" />
              <span class="text-sm font-semibold">群组通知</span>
            </div>
          </template>

          <div class="space-y-4">
            <SettingToggleItem
              title="群组邀请"
              icon="i-lucide-mail"
              description="当收到群组邀请时接收通知"
              :model-value="notificationSettings.group_invite_enabled"
              @update:model-value="updateNotifications('group_invite_enabled', $event)"
            />
            <SettingToggleItem
              title="邀请回复"
              icon="i-lucide-message-square-text"
              description="当您的邀请被回复时接收通知"
              :model-value="notificationSettings.group_invite_response_enabled"
              @update:model-value="updateNotifications('group_invite_response_enabled', $event)"
            />
            <SettingToggleItem
              title="加入申请"
              icon="i-lucide-user-plus"
              description="当有人申请加入您的群组时接收通知"
              :model-value="notificationSettings.group_application_enabled"
              @update:model-value="updateNotifications('group_application_enabled', $event)"
            />
            <SettingToggleItem
              title="申请回复"
              icon="i-lucide-circle-check"
              description="当您的加入申请被处理时接收通知"
              :model-value="notificationSettings.group_application_response_enabled"
              @update:model-value="updateNotifications('group_application_response_enabled', $event)"
            />
          </div>
        </UCard>

        <!-- 互动通知 -->
        <UCard variant="subtle">
          <template #header>
            <div class="flex items-center gap-3">
              <UBadge color="success" variant="soft" size="lg" square icon="i-lucide-messages-square" />
              <span class="text-sm font-semibold">互动通知</span>
            </div>
          </template>

          <div class="space-y-4">
            <SettingToggleItem
              title="@提及"
              icon="i-lucide-at-sign"
              description="当有人在消息中@您时接收通知"
              :model-value="notificationSettings.message_mention_enabled"
              @update:model-value="updateNotifications('message_mention_enabled', $event)"
            />
            <SettingToggleItem
              title="动态点赞"
              icon="i-lucide-thumbs-up"
              description="当有人点赞您的动态时接收通知"
              :model-value="notificationSettings.moment_like_enabled"
              @update:model-value="updateNotifications('moment_like_enabled', $event)"
            />
            <SettingToggleItem
              title="动态评论"
              icon="i-lucide-message-circle"
              description="当有人评论您的动态时接收通知"
              :model-value="notificationSettings.moment_comment_enabled"
              @update:model-value="updateNotifications('moment_comment_enabled', $event)"
            />
            <SettingToggleItem
              title="评论回复"
              icon="i-lucide-corner-up-left"
              description="当有人回复您的评论时接收通知"
              :model-value="notificationSettings.comment_reply_enabled"
              @update:model-value="updateNotifications('comment_reply_enabled', $event)"
            />
          </div>
        </UCard>

        <!-- 系统通知 -->
        <UCard variant="subtle">
          <template #header>
            <div class="flex items-center gap-3">
              <UBadge color="error" variant="soft" size="lg" square icon="i-lucide-megaphone" />
              <span class="text-sm font-semibold">系统通知</span>
            </div>
          </template>

          <div class="space-y-4">
            <SettingToggleItem
              title="系统公告"
              icon="i-lucide-megaphone"
              description="接收系统公告和重要通知"
              :model-value="notificationSettings.system_announcement_enabled"
              @update:model-value="updateNotifications('system_announcement_enabled', $event)"
            />
          </div>
        </UCard>
      </div>

      <!-- 外观设置 -->
      <div
        v-show="activeTab === 'appearance'"
        class="space-y-6"
      >
        <UCard>
          <template #header>
            <div class="flex items-center gap-3">
              <UBadge color="info" variant="soft" size="lg" square icon="i-lucide-paintbrush" />
              <span class="text-sm font-semibold">外观设置</span>
            </div>
          </template>

          <div class="space-y-4">
            <SettingToggleItem
              title="动画效果"
              icon="i-lucide-sparkles"
              description="启用界面动画和过渡效果"
              :model-value="settingsStore.enableAnimations"
              @update:model-value="settingsStore.setEnableAnimations($event)"
            />
            <SettingToggleItem
              title="紧凑模式"
              icon="i-lucide-columns-2"
              description="使用更紧凑的界面布局"
              :model-value="settingsStore.compactMode"
              @update:model-value="settingsStore.setCompactMode($event)"
            />
          </div>

          <template #footer>
            <UBadge color="neutral" variant="soft" size="xs">
              语言设置可在顶部导航栏直接切换
            </UBadge>
          </template>
        </UCard>
      </div>

      <!-- 危险操作 -->
      <div
        v-show="activeTab === 'danger'"
        class="space-y-6"
      >
        <UCard variant="subtle">
          <template #header>
            <div class="flex items-center gap-3">
              <UBadge color="error" variant="soft" size="lg" square icon="i-lucide-triangle-alert" />
              <span class="font-semibold text-base">危险操作</span>
            </div>
          </template>

            <UAlert
              color="error"
              variant="soft"
              title="警告"
              description="以下操作将永久删除您的数据，且无法恢复。"
              class="mb-4"
            />

            <UCard variant="soft">
              <div class="flex items-center justify-between gap-4">
              <UUser
                class="min-w-0"
                name="删除账户"
                description="永久删除您的账户和所有相关数据"
                size="sm"
              />
              <UButton
                color="error"
                @click="deleteAccount"
              >删除账户</UButton>
              </div>
            </UCard>
        </UCard>
      </div>

      <!-- 关于 -->
      <div
        v-show="activeTab === 'about'"
        class="space-y-6"
      >
        <UCard variant="subtle">
          <template #header>
            <div class="flex items-center gap-3">
              <UBadge color="info" variant="soft" size="lg" square icon="i-lucide-info" />
              <span class="text-sm font-semibold">关于应用</span>
            </div>
          </template>

          <div class="text-center py-6">
            <UAvatar
              size="xl"
              icon="i-lucide-map-pin"
              color="primary"
              class="mx-auto mb-4"
            />
            <div class="text-lg font-semibold mb-1">HotSpot</div>
            <UBadge color="neutral" variant="soft" size="xs" class="mb-3">
              基于位置的社交平台
            </UBadge>
            <UBadge
              label="v1.0.0"
              color="primary"
              variant="soft"
            />
          </div>

          <USeparator class="my-4" />

          <div class="space-y-2">
            <UButton
              variant="ghost"
              :block="true"
              class="justify-start"
              @click="clearCache"
            >
              <div class="mr-3">
                <UIcon
                  name="i-lucide-trash-2"
                  :size="20"
                />
              </div>
              <span>清除缓存</span>
            </UButton>
            <UButton
              variant="ghost"
              :block="true"
              class="justify-start"
              @click="showHelp"
            >
              <div class="mr-3">
                <UIcon
                  name="i-lucide-circle-question-mark"
                  :size="20"
                />
              </div>
              <span>帮助与支持</span>
            </UButton>
            <UButton
              variant="ghost"
              :block="true"
              class="justify-start"
              @click="showPrivacyPolicy"
            >
              <div class="mr-3">
                <UIcon
                  name="i-lucide-file-text"
                  :size="20"
                />
              </div>
              <span>隐私政策</span>
            </UButton>
            <UButton
              variant="ghost"
              :block="true"
              class="justify-start"
              @click="showTerms"
            >
              <div class="mr-3">
                <UIcon
                  name="i-lucide-clipboard-list"
                  :size="20"
                />
              </div>
              <span>服务条款</span>
            </UButton>
          </div>
        </UCard>
      </div>
  </div>
</template>

<script setup lang="ts">
import { getPageSeoConfig, buildSeoMeta } from '~/config/seo'
import { useUsersService } from '~/services/users.service'
import { useSubscriptionsService } from '~/services/subscriptions.service'
import { isTempUser } from '~/utils/userTypes'
import { useSettingsStore } from '~/stores/settings.store'
import type { UserSettingsData, SubscriptionsData } from '~/types/api'
import type { UserDetailData } from '~/types/client'

// SEO
const seoConfig = getPageSeoConfig('settings')
definePageMeta({
  title: seoConfig.title,
  description: seoConfig.description,
  requiresAuth: true,
  layout: 'dashboard',
  dashboardHeader: true
})
useSeoMeta(buildSeoMeta('settings'))

// 依赖
const authStore = useAuthStore()
const settingsStore = useSettingsStore()
const feedback = useFeedbackStore()
const modal = useModal()

// 状态
const activeTab = ref('account')

// 水合安全：用户信息
// 使用 useHydratedStore 解决 Hydration Mismatch
const currentUser = useHydratedStore(authStore, 'user', null as UserDetailData | null)

// 默认设置（用于 SSR 初始渲染）
const defaultPrivacy: UserSettingsData = {
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

const defaultNotifications: SubscriptionsData = {
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

// 水合安全：设置
const privacySettings = useHydratedStore(settingsStore, 'privacy', defaultPrivacy)
const notificationSettings = useHydratedStore(settingsStore, 'notifications', defaultNotifications)

// 计算属性
const isTemporaryUser = computed(() => currentUser.value?.user_type ? isTempUser(currentUser.value.user_type) : false)

// 设置 Tabs 的“移动端 icon-only”展示规则由 app.config.ts 统一控制
const tabItems = [
  { value: 'account', label: '账户', icon: 'i-lucide-circle-user' },
  { value: 'privacy', label: '隐私', icon: 'i-lucide-shield-check' },
  { value: 'notifications', label: '通知', icon: 'i-lucide-bell' },
  { value: 'appearance', label: '外观', icon: 'i-lucide-paintbrush' },
  { value: 'danger', label: '危险', icon: 'i-lucide-triangle-alert' },
  { value: 'about', label: '关于', icon: 'i-lucide-info' }
]

const privacyOptions = [
  { label: '公开', value: 'public' },
  { label: '仅好友', value: 'friends' },
  { label: '私密', value: 'private' }
]

const chatPermissionOptions = [
  { label: '所有人', value: 'everyone' },
  { label: '仅好友', value: 'friends' },
  { label: '禁止私聊', value: 'nobody' }
]

// 服务
const userService = useUsersService()
const subscriptionsService = useSubscriptionsService()
const { withErrorHandling } = useErrorHandler()

/**
 * 自动保存隐私设置（防抖）
 */
const savePrivacySettings = useDebounceFn(async () => {
  await withErrorHandling(async () => {
    // 构造请求体，排除不需要的字段（如果有）
    const payload = { ...settingsStore.privacy }
    await userService.updateSettings(payload)
  }, {
    title: '保存隐私设置失败',
    showToast: false // 自动保存通常不需要打扰用户，除非出错
  })
}, 1000)

/**
 * 更新隐私设置
 */
const updatePrivacy = <K extends keyof UserSettingsData>(key: K, value: UserSettingsData[K]) => {
  settingsStore.updatePrivacy({ [key]: value })
  savePrivacySettings()
}

/**
 * 自动保存通知设置（防抖）
 */
const saveNotificationSettings = useDebounceFn(async () => {
  await withErrorHandling(async () => {
    const payload = { ...settingsStore.notifications }
    await subscriptionsService.update(payload)
  }, {
    title: '保存通知设置失败',
    showToast: false
  })
}, 1000)

/**
 * 更新通知设置
 */
const updateNotifications = <K extends keyof SubscriptionsData>(key: K, value: SubscriptionsData[K]) => {
  settingsStore.updateNotifications({ [key]: value })
  saveNotificationSettings()
}

// 其他操作
const editProfile = () => {
  modal.showEditProfile()
}

const upgradeAccount = () => {
  navigateTo('/auth?mode=register&upgrade=1')
}

const changePassword = () => {
  modal.showChangePassword()
}

const showReportHistory = () => {
  modal.showReportHistory()
}

const showBlockedUsers = () => {
  modal.showBlockedUsers()
}

const clearCache = async () => {
  const confirmed = await modal.showConfirm({
    title: '清除缓存',
    message: '确定要清除应用缓存吗？这将清除所有本地数据并重新加载页面。',
    confirmText: '确认清除',
    confirmColor: 'error'
  })

  if (confirmed) {
    localStorage.clear()
    sessionStorage.clear()
    feedback.success('缓存已清除，正在重新加载...')
    useTimeoutFn(() => reloadNuxtApp(), 500)
  }
}

const showHelp = () => {
  feedback.info('暂不提供帮助中心')
}

const showPrivacyPolicy = () => {
  feedback.info('暂不提供隐私政策页面')
}

const showTerms = () => {
  feedback.info('暂不提供服务条款页面')
}

const deleteAccount = () => {
  modal.showDeleteAccount()
}

// 初始化：从API加载设置
onMounted(async () => {
  if (!settingsStore.remoteLoaded) {
    await withErrorHandling(async () => {
      const [privacySettingsData, notificationsSettingsData] = await Promise.all([
        userService.getSettings(),
        subscriptionsService.getConfig()
      ])
      settingsStore.setPrivacy(privacySettingsData)
      settingsStore.setNotifications(notificationsSettingsData)
      settingsStore.markRemoteLoaded()
    }, {
      title: '加载设置失败',
      showToast: true
    })
  }
})
</script>