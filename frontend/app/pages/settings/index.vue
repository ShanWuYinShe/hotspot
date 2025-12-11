<template>
  <PageLayout
    title="设置"
    description="管理您的账户、隐私和通知偏好"
  >
    <div class="max-w-4xl mx-auto">
      <!-- 设置分类标签页 -->
      <UTabs v-model="activeTab" :items="tabItems" class="mb-6" />

      <!-- 账户设置 -->
      <div v-show="activeTab === 'account'" class="space-y-6">
        <!-- 个人资料 -->
        <UCard>
          <template #header>
            <div class="flex items-center gap-3">
              <UIcon name="i-heroicons-user-circle" class="w-5 h-5 text-primary" />
              <span class="font-semibold">个人资料</span>
            </div>
          </template>
          
          <div class="flex items-center gap-4">
            <div class="relative">
              <UAvatar :src="authStore.user?.avatar_url" :alt="authStore.user?.nickname" size="xl" />
              <UButton
                icon="i-heroicons-camera"
                size="xs"
                color="primary"
                class="absolute -bottom-1 -right-1 rounded-full"
                @click="changeAvatar"
              />
            </div>
            <div class="flex-1 min-w-0">
              <h3 class="text-lg font-semibold truncate">{{ authStore.user?.nickname || '未设置昵称' }}</h3>
              <p class="text-sm text-secondary-content">
                <UBadge :label="isTemporaryUser ? '临时用户' : '正式用户'" :color="isTemporaryUser ? 'warning' : 'success'" variant="soft" size="xs" />
              </p>
            </div>
            <div class="flex gap-2">
              <UButton variant="outline" size="sm" @click="editNickname">编辑昵称</UButton>
              <UButton v-if="isTemporaryUser" size="sm" @click="upgradeAccount">升级账户</UButton>
            </div>
          </div>
        </UCard>

        <!-- 账户安全 -->
        <UCard v-if="!isTemporaryUser">
          <template #header>
            <div class="flex items-center gap-3">
              <UIcon name="i-heroicons-shield-check" class="w-5 h-5 text-success" />
              <span class="font-semibold">账户安全</span>
            </div>
          </template>
          
          <div class="space-y-4">
            <div class="flex items-center justify-between">
              <div>
                <div class="font-medium">登录密码</div>
                <div class="text-sm text-secondary-content">定期更换密码可以提高账户安全性</div>
              </div>
              <UButton variant="outline" size="sm" @click="changePassword">修改密码</UButton>
            </div>
          </div>
        </UCard>
      </div>

      <!-- 隐私设置 -->
      <div v-show="activeTab === 'privacy'" class="space-y-6">
        <!-- 隐私设置 -->
        <UCard>
          <template #header>
            <div class="flex items-center gap-3">
              <UIcon name="i-heroicons-shield-check" class="w-5 h-5 text-success" />
              <span class="font-semibold">隐私设置</span>
            </div>
          </template>
          
          <div class="space-y-4">
            <SettingToggleItem
              title="位置共享"
              icon="i-heroicons-map-pin"
              description="允许其他用户查看我的位置"
              :model-value="settingsStore.privacy.location_visible"
              @update:model-value="updatePrivacy('location_visible', $event)"
            />
            <SettingToggleItem
              title="在线状态"
              icon="i-heroicons-signal"
              description="允许其他用户查看我的在线状态"
              :model-value="settingsStore.privacy.online_visible"
              @update:model-value="updatePrivacy('online_visible', $event)"
            />
            <SettingToggleItem
              title="最后上线时间"
              icon="i-heroicons-clock"
              description="允许其他用户查看我的最后上线时间"
              :model-value="settingsStore.privacy.last_online_visible"
              @update:model-value="updatePrivacy('last_online_visible', $event)"
            />
          </div>
        </UCard>

        <!-- 社交设置 -->
        <UCard>
          <template #header>
            <div class="flex items-center gap-3">
              <UIcon name="i-heroicons-users" class="w-5 h-5 text-primary" />
              <span class="font-semibold">社交设置</span>
            </div>
          </template>
          
          <div class="space-y-4">
            <UFormField label="谁可以向我发起私聊">
              <USelect 
                :model-value="settingsStore.privacy.allow_private_chat" 
                :items="chatPermissionOptions" 
                class="w-full max-w-xs"
                @update:model-value="(v) => v && updatePrivacy('allow_private_chat', v as 'everyone' | 'friends' | 'nobody')"
              />
            </UFormField>
            <SettingToggleItem
              title="允许关注"
              icon="i-heroicons-user-plus"
              description="允许其他用户关注我"
              :model-value="settingsStore.privacy.allow_follow"
              @update:model-value="updatePrivacy('allow_follow', $event)"
            />
          </div>
        </UCard>

        <!-- 资料可见性 -->
        <UCard>
          <template #header>
            <div class="flex items-center gap-3">
              <UIcon name="i-heroicons-eye" class="w-5 h-5 text-secondary-content" />
              <span class="font-semibold">资料可见性</span>
            </div>
          </template>
          
          <div class="space-y-4">
            <UFormField label="谁可以查看我的资料">
              <USelect 
                :model-value="settingsStore.privacy.privacy_level" 
                :items="privacyOptions" 
                class="w-full max-w-xs"
                @update:model-value="(v) => v && updatePrivacy('privacy_level', v as 'public' | 'friends' | 'private')"
              />
            </UFormField>
            <UFormField label="谁可以查看我的好友列表">
              <USelect 
                :model-value="settingsStore.privacy.friends_visible" 
                :items="privacyOptions" 
                class="w-full max-w-xs"
                @update:model-value="(v) => v && updatePrivacy('friends_visible', v as 'public' | 'friends' | 'private')"
              />
            </UFormField>
            <UFormField label="谁可以查看我的关注列表">
              <USelect 
                :model-value="settingsStore.privacy.following_visible" 
                :items="privacyOptions" 
                class="w-full max-w-xs"
                @update:model-value="(v) => v && updatePrivacy('following_visible', v as 'public' | 'friends' | 'private')"
              />
            </UFormField>
            <UFormField label="谁可以查看我的粉丝列表">
              <USelect 
                :model-value="settingsStore.privacy.followers_visible" 
                :items="privacyOptions" 
                class="w-full max-w-xs"
                @update:model-value="(v) => v && updatePrivacy('followers_visible', v as 'public' | 'friends' | 'private')"
              />
            </UFormField>
            <UFormField label="谁可以查看我的动态">
              <USelect 
                :model-value="settingsStore.privacy.moments_visible" 
                :items="privacyOptions" 
                class="w-full max-w-xs"
                @update:model-value="(v) => v && updatePrivacy('moments_visible', v as 'public' | 'friends' | 'private')"
              />
            </UFormField>
            <UFormField label="谁可以查看我的统计数据">
              <USelect 
                :model-value="settingsStore.privacy.stats_visible" 
                :items="privacyOptions" 
                class="w-full max-w-xs"
                @update:model-value="(v) => v && updatePrivacy('stats_visible', v as 'public' | 'friends' | 'private')"
              />
            </UFormField>
          </div>
        </UCard>

        <!-- 搜索设置 -->
        <UCard>
          <template #header>
            <div class="flex items-center gap-3">
              <UIcon name="i-heroicons-magnifying-glass" class="w-5 h-5 text-secondary-content" />
              <span class="font-semibold">搜索设置</span>
            </div>
          </template>
          
          <div class="space-y-4">
            <SettingToggleItem
              title="允许被搜索"
              icon="i-heroicons-magnifying-glass"
              description="总开关：允许其他用户通过搜索找到我"
              :model-value="settingsStore.privacy.searchable"
              @update:model-value="updatePrivacy('searchable', $event)"
            />
            <SettingToggleItem
              title="昵称搜索"
              icon="i-heroicons-user"
              description="允许通过昵称搜索到我"
              :model-value="settingsStore.privacy.nickname_searchable"
              @update:model-value="updatePrivacy('nickname_searchable', $event)"
            />
            <SettingToggleItem
              title="手机号搜索"
              icon="i-heroicons-phone"
              description="允许通过手机号搜索到我"
              :model-value="settingsStore.privacy.phone_searchable"
              @update:model-value="updatePrivacy('phone_searchable', $event)"
            />
            <SettingToggleItem
              title="位置搜索"
              icon="i-heroicons-map-pin"
              description="在附近的人列表中显示我"
              :model-value="settingsStore.privacy.location_searchable"
              @update:model-value="updatePrivacy('location_searchable', $event)"
            />
          </div>
        </UCard>

        <!-- 安全管理 -->
        <UCard>
          <template #header>
            <div class="flex items-center gap-3">
              <UIcon name="i-heroicons-shield-exclamation" class="w-5 h-5 text-secondary-content" />
              <span class="font-semibold">安全管理</span>
            </div>
          </template>
          
          <div class="space-y-4">
            <div class="flex items-center justify-between">
              <div>
                <div class="font-medium">我的举报记录</div>
                <div class="text-sm text-secondary-content">查看您提交的举报记录和处理状态</div>
              </div>
              <UButton variant="outline" size="sm" icon="i-heroicons-document-text" @click="showReportHistory">查看记录</UButton>
            </div>
            <USeparator />
            <div class="flex items-center justify-between">
              <div>
                <div class="font-medium">黑名单管理</div>
                <div class="text-sm text-secondary-content">管理您拉黑的用户列表</div>
              </div>
              <UButton variant="outline" size="sm" icon="i-heroicons-user-minus" @click="showBlockedUsers">管理黑名单</UButton>
            </div>
          </div>
        </UCard>
      </div>

      <!-- 通知设置 -->
      <div v-show="activeTab === 'notifications'" class="space-y-6">
        <!-- 社交通知 -->
        <UCard>
          <template #header>
            <div class="flex items-center gap-3">
              <UIcon name="i-heroicons-bell" class="w-5 h-5 text-warning" />
              <span class="font-semibold">社交通知</span>
            </div>
          </template>
          
          <div class="space-y-4">
            <SettingToggleItem
              title="新关注者"
              icon="i-heroicons-user-plus"
              description="当有人关注您时接收通知"
              :model-value="settingsStore.notifications.new_follower_enabled"
              @update:model-value="updateNotifications('new_follower_enabled', $event)"
            />
            <SettingToggleItem
              title="互相关注"
              icon="i-heroicons-heart"
              description="当互相关注成为好友时接收通知"
              :model-value="settingsStore.notifications.mutual_follow_enabled"
              @update:model-value="updateNotifications('mutual_follow_enabled', $event)"
            />
          </div>
        </UCard>

        <!-- 群组通知 -->
        <UCard>
          <template #header>
            <div class="flex items-center gap-3">
              <UIcon name="i-heroicons-user-group" class="w-5 h-5 text-primary" />
              <span class="font-semibold">群组通知</span>
            </div>
          </template>
          
          <div class="space-y-4">
            <SettingToggleItem
              title="群组邀请"
              icon="i-heroicons-envelope"
              description="当收到群组邀请时接收通知"
              :model-value="settingsStore.notifications.group_invite_enabled"
              @update:model-value="updateNotifications('group_invite_enabled', $event)"
            />
            <SettingToggleItem
              title="邀请回复"
              icon="i-heroicons-chat-bubble-left-ellipsis"
              description="当您的邀请被回复时接收通知"
              :model-value="settingsStore.notifications.group_invite_response_enabled"
              @update:model-value="updateNotifications('group_invite_response_enabled', $event)"
            />
            <SettingToggleItem
              title="加入申请"
              icon="i-heroicons-user-plus"
              description="当有人申请加入您的群组时接收通知"
              :model-value="settingsStore.notifications.group_application_enabled"
              @update:model-value="updateNotifications('group_application_enabled', $event)"
            />
            <SettingToggleItem
              title="申请回复"
              icon="i-heroicons-check-circle"
              description="当您的加入申请被处理时接收通知"
              :model-value="settingsStore.notifications.group_application_response_enabled"
              @update:model-value="updateNotifications('group_application_response_enabled', $event)"
            />
          </div>
        </UCard>

        <!-- 互动通知 -->
        <UCard>
          <template #header>
            <div class="flex items-center gap-3">
              <UIcon name="i-heroicons-chat-bubble-left-right" class="w-5 h-5 text-success" />
              <span class="font-semibold">互动通知</span>
            </div>
          </template>
          
          <div class="space-y-4">
            <SettingToggleItem
              title="@提及"
              icon="i-heroicons-at-symbol"
              description="当有人在消息中@您时接收通知"
              :model-value="settingsStore.notifications.message_mention_enabled"
              @update:model-value="updateNotifications('message_mention_enabled', $event)"
            />
            <SettingToggleItem
              title="动态点赞"
              icon="i-heroicons-hand-thumb-up"
              description="当有人点赞您的动态时接收通知"
              :model-value="settingsStore.notifications.moment_like_enabled"
              @update:model-value="updateNotifications('moment_like_enabled', $event)"
            />
            <SettingToggleItem
              title="动态评论"
              icon="i-heroicons-chat-bubble-oval-left"
              description="当有人评论您的动态时接收通知"
              :model-value="settingsStore.notifications.moment_comment_enabled"
              @update:model-value="updateNotifications('moment_comment_enabled', $event)"
            />
            <SettingToggleItem
              title="评论回复"
              icon="i-heroicons-arrow-uturn-left"
              description="当有人回复您的评论时接收通知"
              :model-value="settingsStore.notifications.comment_reply_enabled"
              @update:model-value="updateNotifications('comment_reply_enabled', $event)"
            />
          </div>
        </UCard>

        <!-- 系统通知 -->
        <UCard>
          <template #header>
            <div class="flex items-center gap-3">
              <UIcon name="i-heroicons-megaphone" class="w-5 h-5 text-error" />
              <span class="font-semibold">系统通知</span>
            </div>
          </template>
          
          <div class="space-y-4">
            <SettingToggleItem
              title="系统公告"
              icon="i-heroicons-megaphone"
              description="接收系统公告和重要通知"
              :model-value="settingsStore.notifications.system_announcement_enabled"
              @update:model-value="updateNotifications('system_announcement_enabled', $event)"
            />
          </div>
        </UCard>
      </div>

      <!-- 外观设置 -->
      <div v-show="activeTab === 'appearance'" class="space-y-6">
        <UCard>
          <template #header>
            <div class="flex items-center gap-3">
              <UIcon name="i-heroicons-paint-brush" class="w-5 h-5 text-primary" />
              <span class="font-semibold">外观设置</span>
            </div>
          </template>
          
          <div class="space-y-4">
            <SettingToggleItem
              title="动画效果"
              icon="i-heroicons-sparkles"
              description="启用界面动画和过渡效果"
              :model-value="settingsStore.enableAnimations"
              @update:model-value="settingsStore.setEnableAnimations($event)"
            />
            <SettingToggleItem
              title="紧凑模式"
              icon="i-heroicons-view-columns"
              description="使用更紧凑的界面布局"
              :model-value="settingsStore.compactMode"
              @update:model-value="settingsStore.setCompactMode($event)"
            />
          </div>
          
          <template #footer>
            <p class="text-sm text-secondary-content">
              语言设置可在顶部导航栏直接切换
            </p>
          </template>
        </UCard>
      </div>

      <!-- 危险操作 -->
      <div v-show="activeTab === 'danger'" class="space-y-6">
        <UCard class="border-error">
          <template #header>
            <div class="flex items-center gap-3">
              <UIcon name="i-heroicons-exclamation-triangle" class="w-5 h-5 text-error" />
              <span class="font-semibold text-error">危险操作</span>
            </div>
          </template>
          
          <UAlert
            color="error"
            variant="soft"
            title="警告"
            description="以下操作将永久删除您的数据，且无法恢复。"
            class="mb-4"
          />
          
          <div class="flex items-center justify-between p-4 bg-error-light rounded-lg border border-error">
            <div>
              <div class="font-medium">删除账户</div>
              <div class="text-sm text-secondary-content">永久删除您的账户和所有相关数据</div>
            </div>
            <UButton color="error" @click="deleteAccount">删除账户</UButton>
          </div>
        </UCard>
      </div>

      <!-- 关于 -->
      <div v-show="activeTab === 'about'" class="space-y-6">
        <UCard>
          <template #header>
            <div class="flex items-center gap-3">
              <UIcon name="i-heroicons-information-circle" class="w-5 h-5 text-primary" />
              <span class="font-semibold">关于应用</span>
            </div>
          </template>
          
          <div class="text-center py-6">
            <div class="w-16 h-16 bg-primary-subtle rounded-2xl flex items-center justify-center mx-auto mb-4">
              <UIcon name="i-heroicons-map-pin" class="w-8 h-8 text-primary" />
            </div>
            <h3 class="text-xl font-bold mb-1">HotSpot</h3>
            <p class="text-sm text-secondary-content mb-3">基于位置的社交平台</p>
            <UBadge label="v1.0.0" color="primary" variant="soft" />
          </div>
          
          <USeparator class="my-4" />
          
          <div class="space-y-2">
            <UButton variant="ghost" block class="justify-start" @click="clearCache">
              <UIcon name="i-heroicons-trash" class="w-5 h-5 mr-3" />
              <span>清除缓存</span>
            </UButton>
            <UButton variant="ghost" block class="justify-start" @click="showHelp">
              <UIcon name="i-heroicons-question-mark-circle" class="w-5 h-5 mr-3" />
              <span>帮助与支持</span>
            </UButton>
            <UButton variant="ghost" block class="justify-start" @click="showPrivacyPolicy">
              <UIcon name="i-heroicons-document-text" class="w-5 h-5 mr-3" />
              <span>隐私政策</span>
            </UButton>
            <UButton variant="ghost" block class="justify-start" @click="showTerms">
              <UIcon name="i-heroicons-clipboard-document-list" class="w-5 h-5 mr-3" />
              <span>服务条款</span>
            </UButton>
          </div>
        </UCard>
      </div>
    </div>
  </PageLayout>
</template>

<script setup lang="ts">
import { getPageSeoConfig, buildSeoMeta } from '~/config/seo'
import { useUsersService } from '~/services/users.service'
import { useSubscriptionsService } from '~/services/subscriptions.service'
import { isTempUser } from '~/utils/userTypes'
import { useSettingsStore } from '~/stores/settings.store'
import type { UserSettingsData, SubscriptionsData } from '~/types/api'

// SEO
const seoConfig = getPageSeoConfig('settings')
definePageMeta({
  title: seoConfig.title,
  description: seoConfig.description,
  requiresAuth: true,
  layout: 'dashboard'
})
useSeoMeta(buildSeoMeta('settings'))

// 依赖
const authStore = useAuthStore()
const settingsStore = useSettingsStore()
const toast = useToast()
const modal = useModal()

// 状态
const activeTab = ref('account')

// 计算属性
const isTemporaryUser = computed(() => authStore.user?.user_type ? isTempUser(authStore.user.user_type) : false)

// 配置 - 移动端只显示图标，桌面端显示完整标签
const tabItems = [
  { value: 'account', label: '账户', icon: 'i-heroicons-user-circle' },
  { value: 'privacy', label: '隐私', icon: 'i-heroicons-shield-check' },
  { value: 'notifications', label: '通知', icon: 'i-heroicons-bell' },
  { value: 'appearance', label: '外观', icon: 'i-heroicons-paint-brush' },
  { value: 'danger', label: '危险', icon: 'i-heroicons-exclamation-triangle' },
  { value: 'about', label: '关于', icon: 'i-heroicons-information-circle' },
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

// 防抖定时器
let privacySaveTimer: ReturnType<typeof setTimeout> | null = null
let notificationsSaveTimer: ReturnType<typeof setTimeout> | null = null

/**
 * 更新隐私设置
 */
const updatePrivacy = <K extends keyof UserSettingsData>(key: K, value: UserSettingsData[K]) => {
  settingsStore.updatePrivacy({ [key]: value })
  
  // 防抖保存到API
  if (privacySaveTimer) clearTimeout(privacySaveTimer)
  privacySaveTimer = setTimeout(async () => {
    try {
      await userService.updateSettings({ [key]: value })
    } catch (error) {
      console.error('保存隐私设置失败:', error)
      toast.add({ title: '保存失败', description: '请稍后重试', color: 'error' })
    }
  }, 500)
}

/**
 * 更新通知设置
 */
const updateNotifications = <K extends keyof SubscriptionsData>(key: K, value: SubscriptionsData[K]) => {
  settingsStore.updateNotifications({ [key]: value })
  
  // 防抖保存到API
  if (notificationsSaveTimer) clearTimeout(notificationsSaveTimer)
  notificationsSaveTimer = setTimeout(async () => {
    try {
      await subscriptionsService.update({ [key]: value })
    } catch (error) {
      console.error('保存通知设置失败:', error)
      toast.add({ title: '保存失败', description: '请稍后重试', color: 'error' })
    }
  }, 500)
}

// 其他操作
const changeAvatar = () => {
  toast.add({ title: '功能开发中', color: 'info' })
}

const editNickname = () => {
  toast.add({ title: '功能开发中', color: 'info' })
}

const upgradeAccount = () => {
  navigateTo('/auth/register?upgrade=1')
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
    toast.add({ title: '缓存已清除，正在重新加载...', color: 'success' })
    setTimeout(() => window.location.reload(), 500)
  }
}

const showHelp = () => {
  toast.add({ title: '帮助中心即将上线', color: 'info' })
}

const showPrivacyPolicy = () => {
  toast.add({ title: '隐私政策即将上线', color: 'info' })
}

const showTerms = () => {
  toast.add({ title: '服务条款即将上线', color: 'info' })
}

const deleteAccount = () => {
  modal.showDeleteAccount()
}

// 初始化：从API加载设置
onMounted(async () => {
  if (!settingsStore.remoteLoaded) {
    try {
      const [privacySettings, notificationsSettings] = await Promise.all([
        userService.getSettings(),
        subscriptionsService.getConfig()
      ])
      settingsStore.setPrivacy(privacySettings)
      settingsStore.setNotifications(notificationsSettings)
      settingsStore.markRemoteLoaded()
    } catch (error) {
      console.error('加载设置失败:', error)
      toast.add({ title: '加载设置失败', color: 'error' })
    }
  }
})

// 清理定时器
onUnmounted(() => {
  if (privacySaveTimer) clearTimeout(privacySaveTimer)
  if (notificationsSaveTimer) clearTimeout(notificationsSaveTimer)
})
</script>
