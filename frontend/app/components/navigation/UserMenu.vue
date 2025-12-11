<template>
  <!-- 未登录状态 - 明确的登录按钮 -->
  <div v-if="!user" class="flex items-center gap-2">
    <!-- 快速体验按钮 -->
    <UButton
      variant="ghost"
      color="neutral"
      size="sm"
      @click="createTempUser"
      :loading="isCreatingTemp"
      class="hidden sm:flex"
    >
      <UIcon name="i-heroicons-bolt" class="w-4 h-4 mr-1.5" />
      快速体验
    </UButton>

    <!-- 登录按钮 -->
    <UButton
      to="/auth/login"
      color="primary"
      size="sm"
    >
      <UIcon name="i-heroicons-arrow-right-end-on-rectangle" class="w-4 h-4 mr-1.5" />
      登录
    </UButton>
  </div>

  <!-- 已登录状态 - 带用户信息的菜单 -->
  <UDropdownMenu v-else :items="menuItems" :ui="{ content: 'w-64' }">
    <UButton 
      variant="ghost" 
      color="neutral"
      class="relative pl-2 pr-3 py-1.5 rounded-full border border-default hover-border-default transition-colors"
    >
      <div class="flex items-center gap-2">
        <!-- 头像 -->
        <div class="relative">
          <UAvatar 
            size="sm"
            :alt="user?.nickname"
            :src="user?.avatar_url"
          >
            {{ user?.nickname?.charAt(0) }}
          </UAvatar>
          <!-- 临时用户标记 -->
          <div 
            v-if="isTemp" 
            class="absolute -top-0.5 -right-0.5 w-3 h-3 bg-icon-warning rounded-full border-2 border-white-dark"
            :title="'临时用户'"
          />
          <!-- 在线状态指示器 -->
          <div 
            v-else
            class="absolute -bottom-0.5 -right-0.5 w-3 h-3 bg-icon-success rounded-full border-2 border-white-dark"
          />
        </div>
        
        <!-- 用户名（仅桌面端显示） -->
        <span class="hidden sm:block text-sm font-medium text-secondary-strong max-w-24 truncate">
          {{ user?.nickname }}
        </span>
        
        <!-- 下拉指示器 -->
        <UIcon 
          name="i-heroicons-chevron-down" 
          class="w-4 h-4 text-muted transition-transform ui-open:rotate-180" 
        />
      </div>
    </UButton>
    
    <!-- 菜单头部 - 用户信息卡片 -->
    <template #header>
      <div class="px-3 py-3 border-b border-subtle">
        <div class="flex items-center gap-3">
          <UAvatar 
            size="lg"
            :alt="user?.nickname"
            :src="user?.avatar_url"
          >
            {{ user?.nickname?.charAt(0) }}
          </UAvatar>
          <div class="flex-1 min-w-0">
            <p class="font-semibold text-primary-content truncate">
              {{ user?.nickname }}
            </p>
            <p class="text-xs text-secondary-content truncate">
              {{ isTemp ? '临时用户' : '正式用户' }}
            </p>
          </div>
        </div>
        
        <!-- 临时用户升级提醒 -->
        <UButton
          v-if="isTemp"
          to="/auth/register?upgrade=1"
          variant="soft"
          color="warning"
          size="xs"
          class="w-full mt-3"
        >
          <UIcon name="i-heroicons-arrow-up-circle" class="w-4 h-4 mr-1" />
          升级为正式账户
        </UButton>
      </div>
    </template>
  </UDropdownMenu>
</template>

<script setup lang="ts">
import { isTempUser } from '~/utils/userTypes'
import { getErrorMessage } from '~/utils/error'
import { useAuthService } from '~/services/auth.service'
import type { DropdownMenuItem } from '@nuxt/ui'
import type { UserResponse } from '~/types/client'

const toast = useToast()
const authService = useAuthService()
const authStore = useAuthStore()

// 中间变量：SSR 时使用默认值，onMounted 后从 store 同步真实值
const user = ref<UserResponse | null>(null)

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
  authStore.accountsList.filter(a => a.user.user_id !== user.value?.user_id)
)

// 创建临时用户
const createTempUser = async () => {
  try {
    isCreatingTemp.value = true
    const response = await authService.createTempUser({ nickname: `访客_${Date.now().toString(36).slice(-4)}` })
    // 设置会话
    authStore.setAccountSession(
      { user_id: response.user_id, nickname: response.nickname },
      response.access_token,
      response.refresh_token,
      new Date(response.access_token_expires_at).getTime(),
      new Date(response.refresh_token_expires_at).getTime()
    )
    toast.add({
      title: '欢迎体验 HotSpot',
      description: '临时账户已创建，您可以开始探索了',
      color: 'success'
    })
    navigateTo('/')
  } catch (err: unknown) {
    toast.add({
      title: '创建失败',
      description: getErrorMessage(err, '请稍后重试'),
      color: 'error'
    })
  } finally {
    isCreatingTemp.value = false
  }
}

// 切换账户
const handleSwitchAccount = async (userId: string) => {
  try {
    await authStore.switchToAccount(userId)
    toast.add({
      title: '已切换账户',
      color: 'success'
    })
  } catch (err: unknown) {
    toast.add({
      title: '切换失败',
      description: getErrorMessage(err, '请稍后重试'),
      color: 'error'
    })
  }
}

// 登出 - 停留在当前页面，不跳转
const handleLogout = async () => {
  try {
    // 先调用服务端登出 API
    await authService.logout()
    // 再清除本地会话
    authStore.logoutCurrent()
    toast.add({
      title: '已退出登录',
      color: 'info'
    })
    // 不跳转，停留在当前页面
  } catch (err: unknown) {
    // 即使服务端登出失败，也清除本地会话
    authStore.logoutCurrent()
    toast.add({
      title: '退出失败',
      description: getErrorMessage(err, '请稍后重试'),
      color: 'error'
    })
  }
}

// 菜单项配置 - 只保留账户相关操作，导航由侧栏负责
const menuItems = computed((): DropdownMenuItem[][] => {
  if (!user.value) return []

  const items: DropdownMenuItem[][] = [
    // 账户管理
    [
      {
        label: '个人主页',
        icon: 'i-heroicons-user',
        onSelect: () => navigateTo('/profile')
      },
      {
        label: '设置',
        icon: 'i-heroicons-cog-6-tooth',
        onSelect: () => navigateTo('/settings')
      }
    ]
  ]

  // 多账户管理
  if (otherAccounts.value.length > 0) {
    const accountItems: DropdownMenuItem[] = [
      {
        label: '切换账户',
        icon: 'i-heroicons-arrows-right-left',
        disabled: true
      },
      ...otherAccounts.value.map(account => ({
        label: account.user.nickname || '未知用户',
        icon: account.user.user_type && isTempUser(account.user.user_type) ? 'i-heroicons-clock' : 'i-heroicons-user-circle',
        onSelect: () => handleSwitchAccount(String(account.user.user_id))
      })),
      {
        label: '添加账户',
        icon: 'i-heroicons-plus-circle',
        onSelect: () => navigateTo('/auth/login?add=1')
      }
    ]
    items.push(accountItems)
  }

  // 登出
  items.push([
    {
      label: '退出登录',
      icon: 'i-heroicons-arrow-right-start-on-rectangle',
      color: 'error',
      onSelect: handleLogout
    }
  ])

  return items
})
</script>
