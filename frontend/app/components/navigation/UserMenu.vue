<template>
  <!-- 用户菜单 - 使用 UDropdownMenu 实现更规范的菜单 -->
  <UDropdownMenu :items="menuItems" :ui="{ content: 'w-56' }">
    <UButton 
      size="lg" 
      variant="ghost" 
      color="neutral"
      square
      class="relative"
    >
      <UAvatar 
        v-if="user"
        size="sm"
        :alt="user.nickname"
        :src="user.avatar_url"
      />
      <UIcon 
        v-else 
        name="i-heroicons-user-circle" 
        class="w-6 h-6" 
      />
    </UButton>
  </UDropdownMenu>
</template>

<script setup lang="ts">
import { isTempUser } from '~/utils/userTypes'
import type { DropdownMenuItem } from '@nuxt/ui'

const { t } = useI18n()
const authStore = useAuthStore()

// 当前用户
const user = computed(() => authStore.user)

// 其他账户列表
const otherAccounts = computed(() => 
  authStore.accountsList.filter((a: any) => a.user_id !== user.value?.user_id)
)

// 创建临时用户
const handleCreateTempUser = async () => {
  try {
    // 创建临时用户，使用默认昵称
    await authStore.createTempUser({ nickname: '访客用户' });
  } catch (error) {
    console.error('创建临时用户失败:', error)
  }
}

// 切换账户
const handleSwitchAccount = async (userId: string) => {
  try {
    await authStore.switchToAccount(userId)
  } catch (error) {
    console.error('切换账户失败:', error)
  }
}

// 登出
const logout = async () => {
  try {
    const authStore = useAuthStore()
    await logout()
    navigateTo('/auth/login')
  } catch (error) {
    console.error('登出失败:', error)
  }
}

// 菜单项配置
const menuItems = computed((): DropdownMenuItem[][] => {
  if (!user.value) {
    // 未登录用户菜单
    return [
      [
        {
          label: t('userMenu.tempExperience'),
          icon: 'i-heroicons-rocket-launch',
          onSelect: handleCreateTempUser
        },
        {
          label: t('userMenu.loginRegister'),
          icon: 'i-heroicons-arrow-right-end-on-rectangle',
          onSelect: () => navigateTo('/auth/login')
        }
      ]
    ]
  }

  // 已登录用户菜单
  const items: DropdownMenuItem[][] = [
    [
      {
        label: t('userMenu.profile'),
        icon: 'i-heroicons-identification',
        onSelect: () => navigateTo('/profile')
      },
      {
        label: t('userMenu.settings'),
        icon: 'i-heroicons-cog-6-tooth',
        onSelect: () => navigateTo('/settings')
      }
    ],
    [
      {
        label: t('userMenu.createGroup'),
        icon: 'i-heroicons-user-group',
        onSelect: () => navigateTo('/groups/create')
      },
      {
        label: t('userMenu.findFriends'),
        icon: 'i-heroicons-magnifying-glass',
        onSelect: () => navigateTo('/search')
      }
    ]
  ]

  // 账户管理部分
  if (isTempUser(user.value.user_id) || otherAccounts.value.length > 0) {
    const accountItems: DropdownMenuItem[] = []
    
    if (isTempUser(user.value.user_id)) {
      accountItems.push({
        label: t('userMenu.upgradeAccount'),
        icon: 'i-heroicons-arrow-up-circle',
        onSelect: () => navigateTo('/auth/register?upgrade=1')
      })
    }
    
    accountItems.push({
      label: t('userMenu.addAccount'),
      icon: 'i-heroicons-plus-circle',
      onSelect: () => navigateTo('/auth/register?add=1')
    })

    // 其他账户切换
    otherAccounts.value.forEach(account => {
      accountItems.push({
        label: account.user?.nickname || '未知用户',
        icon: isTempUser(account.user?.user_id) ? 'i-heroicons-clock' : 'i-heroicons-user-circle',
        onSelect: () => handleSwitchAccount(account.user?.user_id || '')
      })
    })

    if (accountItems.length > 0) {
      items.push(accountItems)
    }
  }

  // 登出
  items.push([
    {
      label: t('userMenu.logout'),
      icon: 'i-heroicons-arrow-left-start-on-rectangle',
      color: 'error',
      onSelect: logout
    }
  ])

  return items
})
</script>