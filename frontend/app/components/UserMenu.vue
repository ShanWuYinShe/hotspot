<template>
  <!-- 用户菜单 - 使用 Nuxt UI v3 UDropdownMenu -->
  <UDropdownMenu 
    :items="userMenuItems" 
    :popper="{ placement: 'bottom-end', offsetDistance: 8 }"
    mode="hover"
  >
    <!-- 触发按钮 -->
    <UButton 
      size="lg" 
      variant="ghost" 
      color="neutral"
      square
      class="relative hover:bg-neutral-100 dark:hover:bg-neutral-800"
    >
      <UAvatar 
        v-if="authStore.user"
        size="sm"
        :alt="authStore.user.nickname"
        :src="authStore.user.avatar_url"
        class="ring-2 ring-white dark:ring-neutral-800"
      />
      <UIcon 
        v-else 
        name="i-heroicons-user-circle" 
        class="w-6 h-6 text-neutral-600 dark:text-neutral-400" 
      />
      
      <!-- 在线状态指示器 -->
      <div 
        v-if="authStore.user" 
        class="absolute -bottom-0.5 -right-0.5 w-3 h-3 bg-green-500 border-2 border-white dark:border-neutral-800 rounded-full"
      />
    </UButton>

    <!-- 用户信息头部 -->
    <template #header v-if="authStore.user">
      <div class="flex items-center gap-3 p-3 border-b border-neutral-200 dark:border-neutral-700">
        <UAvatar 
          :alt="authStore.user.nickname"
          :src="authStore.user.avatar_url"
          size="md"
          class="ring-2 ring-primary-500/20"
        />
        <div class="flex-1 min-w-0">
          <p class="font-semibold text-neutral-900 dark:text-white truncate">
            {{ authStore.user.nickname }}
          </p>
          <p class="text-sm text-neutral-500 dark:text-neutral-400 truncate">
            {{ authStore.user.email || t('userMenu.tempUser') }}
          </p>
          <div class="flex items-center gap-1 mt-1">
            <UBadge 
              :color="authStore.user.is_temporary ? 'warning' : 'success'" 
              size="xs"
            >
              {{ authStore.user.is_temporary ? t('userMenu.temporary') : t('userMenu.verified') }}
            </UBadge>
          </div>
        </div>
      </div>
    </template>

    <!-- 未登录状态的头部 -->
    <template #header v-else>
      <div class="flex items-center gap-3 p-3 border-b border-neutral-200 dark:border-neutral-700">
        <div class="w-10 h-10 rounded-full bg-neutral-200 dark:bg-neutral-700 flex items-center justify-center">
          <UIcon name="i-heroicons-user" class="w-5 h-5 text-neutral-500" />
        </div>
        <div>
          <p class="font-medium text-neutral-900 dark:text-white">
            {{ t('userMenu.notLoggedIn') }}
          </p>
          <p class="text-sm text-neutral-500 dark:text-neutral-400">
            {{ t('userMenu.loginToAccess') }}
          </p>
        </div>
      </div>
    </template>
  </UDropdownMenu>

  <!-- 临时用户创建模态框 -->
  <UModal v-model:open="isOpenTempModal">
    <TempUserModal 
      :on-create-temp="createTemp"
      @close="isOpenTempModal = false"
    />
  </UModal>

  <!-- 账户切换确认模态框 -->
  <UModal v-model:open="showSwitchConfirm">
    <SwitchAccountModal 
      :account="switchTargetAccount"
      :on-confirm="confirmSwitchAccount"
      @close="showSwitchConfirm = false"
    />
  </UModal>
</template>

<script setup lang="ts">
import type { DropdownMenuItem } from '@nuxt/ui'

const { t } = useI18n()
const authStore = useAuthStore()
const toast = useSmartToast()

// 响应式状态
const isCreatingTemp = ref(false)
const switchingAccount = ref<string | null>(null)

// 其他账户列表
const otherAccounts = computed(() => 
  authStore.accounts.filter((a) => a.user_id !== authStore.user?.user_id)
)

// 用户菜单项
const userMenuItems = computed((): DropdownMenuItem[][] => {
  const items: DropdownMenuItem[][] = []

  if (authStore.user) {
    // 已登录用户的菜单
    items.push([
      {
        label: t('userMenu.profile'),
        icon: 'i-heroicons-identification',
        to: '/profile'
      },
      {
        label: t('userMenu.settings'),
        icon: 'i-heroicons-cog-6-tooth',
        to: '/settings'
      }
    ])

    // 账户管理
    const accountItems: DropdownMenuItem[] = []
    
    if (authStore.user.is_temporary) {
      accountItems.push({
        label: t('userMenu.upgradeAccount'),
        icon: 'i-heroicons-arrow-up-circle',
        to: '/auth?upgrade=1',
        color: 'primary'
      })
    }

    accountItems.push({
      label: t('userMenu.addAccount'),
      icon: 'i-heroicons-plus-circle',
      to: '/auth?add=1'
    })

    if (accountItems.length > 0) {
      items.push(accountItems)
    }

    // 其他账户切换
    if (otherAccounts.value.length > 0) {
      const switchItems: DropdownMenuItem[] = otherAccounts.value.map(acc => ({
        label: acc.nickname,
        icon: acc.is_temporary ? 'i-heroicons-clock' : 'i-heroicons-user',
        click: () => initiateSwitchAccount(acc),
        disabled: switchingAccount.value === acc.user_id,
        description: formatLastLogin(acc.last_login)
      }))

      items.push([
        {
          label: t('userMenu.switchAccount'),
          icon: 'i-heroicons-arrow-path',
          disabled: true
        },
        ...switchItems
      ])
    }

    // 登出
    items.push([
      {
        label: t('userMenu.logout'),
        icon: 'i-heroicons-arrow-left-start-on-rectangle',
        click: logout,
        color: 'error'
      }
    ])
  } else {
    // 未登录用户的菜单
    items.push([
      {
        label: t('userMenu.tempExperience'),
        icon: 'i-heroicons-rocket-launch',
        click: openTempModal,
        color: 'primary'
      },
      {
        label: t('userMenu.loginRegister'),
        icon: 'i-heroicons-arrow-right-end-on-rectangle',
        to: '/auth'
      }
    ])
  }

  return items
})

// 生成随机昵称
const genNickname = () => `用户${Math.floor(Date.now() % 100000)}`

// 格式化最后登录时间
const formatLastLogin = (lastLogin: string) => {
  const date = new Date(lastLogin)
  const now = new Date()
  const diffMs = now.getTime() - date.getTime()
  const diffDays = Math.floor(diffMs / (1000 * 60 * 60 * 24))
  
  if (diffDays === 0) {
    return t('time.justNow')
  } else if (diffDays === 1) {
    return t('time.yesterday')
  } else if (diffDays < 7) {
    return t('time.daysAgo', { n: diffDays })
  } else {
    return date.toLocaleDateString()
  }
}

// 打开临时用户模态框
const isOpenTempModal = ref(false)
const openTempModal = () => {
  isOpenTempModal.value = true
}

// 创建临时用户
const createTemp = async (nickname?: string) => {
  try {
    isCreatingTemp.value = true
    const finalNickname = (nickname || '').trim() || genNickname()
    
    console.log('开始创建临时用户:', { nickname: finalNickname })
    
    const result = await authStore.createTempUser(finalNickname)
    
    console.log('临时用户创建成功:', result)
    
    toast.add({
      title: t('userMenu.toasts.tempCreated'),
      description: t('userMenu.toasts.welcomeTemp', { name: finalNickname }),
      color: 'success'
    })
    
    await navigateTo('/')
  } catch (error: any) {
    console.error('临时用户创建失败:', error)
    
    toast.add({
      title: t('userMenu.toasts.tempFailed'),
      description: error.message || t('userMenu.toasts.tryAgain'),
      color: 'error'
    })
    throw error // 重新抛出错误，让模态框处理
  } finally {
    isCreatingTemp.value = false
  }
}

// 发起账户切换
const showSwitchConfirm = ref(false)
const switchTargetAccount = ref<any>(null)

function initiateSwitchAccount(account: any) {
  switchTargetAccount.value = account
  showSwitchConfirm.value = true
}

// 确认切换账户
async function confirmSwitchAccount() {
  if (!switchTargetAccount.value) return
  
  const userId = switchTargetAccount.value.user_id
  switchingAccount.value = userId
  
  try {
    await authStore.switchToAccount(userId)

    toast.add({
      title: t('userMenu.toasts.switchSuccess'),
      description: t('userMenu.toasts.switchedTo', { name: switchTargetAccount.value.nickname }),
      color: 'success'
    })

    showSwitchConfirm.value = false
    await navigateTo('/')
  } catch (error: any) {
    toast.add({
      title: t('userMenu.toasts.switchFailed'),
      description: error.message || t('userMenu.toasts.accountExpired'),
      color: 'error'
    })
  } finally {
    switchingAccount.value = null
    switchTargetAccount.value = null
  }
}

// 登出
function logout() {
  authStore.logout()
  toast.add({
    title: t('userMenu.toasts.loggedOut'),
    description: t('userMenu.toasts.redirectingToLogin'),
    color: 'neutral'
  })
  navigateTo('/auth')
}
</script>