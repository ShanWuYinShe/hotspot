<template>
  <!-- 未登录状态 - 明确的登录按钮 -->
  <div v-if="!user" class="flex items-center gap-2">
    <!-- 快速体验按钮 -->
    <UButton
      variant="ghost"
      color="neutral"
      size="sm"
      leading-icon="i-lucide-zap"
      @click="createTempUser"
      :loading="isCreatingTemp"
      class="hidden sm:flex"
    >
      快速体验
    </UButton>

    <!-- 登录按钮 -->
    <UButton
      to="/auth?mode=login"
      color="primary"
      size="sm"
      leading-icon="i-lucide-log-in"
    >
      登录
    </UButton>
  </div>

  <!-- 已登录状态 - 带用户信息的菜单 -->
  <UDropdownMenu v-else :items="menuItems" width="md">
    <UButton 
      variant="ghost" 
      color="neutral"
      class="w-full group"
    >
      <div class="flex items-center gap-2 w-full min-w-0">
        <UUser
          class="min-w-0"
          :name="user?.nickname || ''"
          :avatar="{
            size: 'sm',
            alt: user?.nickname,
            src: user?.avatar_url,
            chip: isTemp ? { color: 'warning', position: 'top-right', size: '2xs' } : { color: 'success', position: 'bottom-right', size: '2xs' }
          }"
          size="sm"
        />

        <UIcon 
          name="i-lucide-chevron-down"
          :size="16"
          class="shrink-0 transition-transform group-data-[state=open]:rotate-180"
        />
      </div>
    </UButton>
    
    <!-- 菜单头部 - 用户信息卡片 -->
    <template #header>
      <div class="px-3 py-3">
        <div class="flex items-start justify-between gap-3">
          <UUser
            class="min-w-0"
            :name="user?.nickname || ''"
            :description="isTemp ? '临时用户' : '正式用户'"
            :avatar="{ size: 'lg', alt: user?.nickname, src: user?.avatar_url }"
          />
          <UBadge
            :color="isTemp ? 'warning' : 'success'"
            variant="soft"
            size="xs"
            class="shrink-0"
          >
            {{ isTemp ? '临时' : '正式' }}
          </UBadge>
        </div>
        
        <!-- 临时用户升级提醒 -->
        <UButton
          v-if="isTemp"
          to="/auth?mode=register&upgrade=1"
          variant="soft"
          color="warning"
          size="xs"
          leading-icon="i-lucide-circle-arrow-up"
          class="w-full mt-3"
        >
          升级为正式账户
        </UButton>
      </div>
    </template>
  </UDropdownMenu>
</template>

<script setup lang="ts">
import { useAuthService } from '~/services/auth.service'
import type { DropdownMenuItem } from '@nuxt/ui'
import type { UserDetailData } from '~/types/client'
import { resetAppForAccountChange } from '~/utils/account'

const feedback = useFeedbackStore()
const authService = useAuthService()
const authStore = useAuthStore()

// 中间变量：SSR 时使用默认值，onMounted 后从 store 同步真实值
const user = ref<UserDetailData | null>(null)

onMounted(() => {
  // 水合完成后，同步 store 值并监听变化
  user.value = authStore.user
  watch(() => authStore.user, (newUser) => {
    user.value = newUser
  })
})

// 当前用户
const isTemp = computed(() => user.value?.user_type && isTempUser(user.value.user_type))
const isCreatingTemp = ref(false)

// 其他账户列表
const otherAccounts = computed(() => 
  (authStore.accountsList || []).filter(a => a && a.user && a.user.user_id !== user.value?.user_id)
)

// 创建临时用户
const createTempUser = async () => {
  try {
    isCreatingTemp.value = true
    const response = await authService.guest({ nickname: `访客_${Date.now().toString(36).slice(-4)}`, client_type: 'app' })
    // 设置会话
    authStore.setAccountSession(
      { user_id: response.user_id, nickname: response.nickname },
      response.access_token,
      response.refresh_token ?? null,
      new Date(response.access_token_expires_at).getTime(),
      new Date(response.refresh_token_expires_at).getTime()
    )
    feedback.success('欢迎体验 HotSpot', '临时账户已创建，您可以开始探索了')
    navigateTo('/')
  } catch (err: unknown) {
    feedback.error('创建失败', getErrorMessage(err, '请稍后重试'))
  } finally {
    isCreatingTemp.value = false
  }
}

// 切换账户
const handleSwitchAccount = async (userId: string) => {
  try {
    await authStore.switchToAccount(userId)
    feedback.success('已切换账户')
    // 防止跨账户数据串联：切换后清空用户级缓存并刷新页面（重新初始化各 Store/WS）
    resetAppForAccountChange()
    if (import.meta.client) window.location.reload()
  } catch (err: unknown) {
    feedback.error('切换失败', getErrorMessage(err, '请稍后重试'))
  }
}

// 登出 - 停留在当前页面，不跳转
const handleLogout = async () => {
  try {
    // 先调用服务端登出 API
    await authService.logout()
    // 再清除本地会话
    authStore.logoutCurrent()
    feedback.info('已退出登录')
    // 不跳转，停留在当前页面
    resetAppForAccountChange()
    if (import.meta.client) window.location.reload()
  } catch (err: unknown) {
    // 即使服务端登出失败，也清除本地会话
    authStore.logoutCurrent()
    feedback.error('退出失败', getErrorMessage(err, '请稍后重试'))
    resetAppForAccountChange()
    if (import.meta.client) window.location.reload()
  }
}

// 锁定当前会话（保留多账户列表）
const handleLock = () => {
  authStore.lock()
  feedback.info('已锁定', '当前已进入未登录状态')
  resetAppForAccountChange()
  if (import.meta.client) window.location.reload()
}

// 菜单项配置 - 只保留账户相关操作，导航由侧栏负责
const menuItems = computed((): DropdownMenuItem[][] => {
  if (!user.value) return []

  const items: DropdownMenuItem[][] = [
    // 账户管理
    [
      {
        label: '个人主页',
        icon: 'i-lucide-user',
        onSelect: () => navigateTo('/profile')
      },
      {
        label: '设置',
        icon: 'i-lucide-settings',
        onSelect: async () => {
          await navigateTo('/settings')
        }
      }
    ]
  ]

  // 多账户管理
  if (otherAccounts.value.length > 0) {
    const accountItems: DropdownMenuItem[] = [
      {
        label: '切换账户',
        icon: 'i-lucide-arrow-left-right',
        disabled: true
      },
      ...otherAccounts.value.filter(account => account && account.user).map(account => ({
        label: account!.user.nickname || '未知用户',
        icon: account!.user.user_type && isTempUser(account!.user.user_type) ? 'i-lucide-clock' : 'i-lucide-circle-user',
        onSelect: () => handleSwitchAccount(String(account!.user.user_id))
      })),
      {
        label: '添加账户',
        icon: 'i-lucide-circle-plus',
        onSelect: () => navigateTo('/auth?mode=login&add=1')
      }
    ]
    items.push(accountItems)
  }

  // 登出
  items.push([
    {
      label: '锁定（保留账户）',
      icon: 'i-lucide-lock-keyhole',
      onSelect: handleLock
    },
    {
      label: '退出登录',
      icon: 'i-lucide-log-out',
      color: 'error',
      onSelect: handleLogout
    }
  ])

  return items
})
</script>
