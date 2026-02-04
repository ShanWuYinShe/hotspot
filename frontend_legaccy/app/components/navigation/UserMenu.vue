<template>
    <div v-if="!user" class="flex items-center gap-2 shrink-0">
      <UButton
        variant="ghost"
        color="neutral"
        size="sm"
        leading-icon="i-lucide-zap"
        @click="createTempUser"
        :loading="isCreatingTemp"
        class="hidden sm:flex"
      >
        {{ t('userMenu.quickExperience') }}
      </UButton>
      <UButton
        to="/auth?mode=login"
        color="primary"
        size="sm"
        leading-icon="i-lucide-log-in"
      >
        {{ t('auth.login') }}
      </UButton>
    </div>

    <UDropdownMenu v-else :items="menuItems">
      <UButton 
        variant="ghost" 
        color="neutral"
        class="w-full justify-between"
        :trailing-icon="null"
      >
        <div class="flex items-center gap-2 min-w-0">
          <UAvatar
            :src="user.avatar_url"
            :alt="user.nickname"
            size="xs"
            :icon="isTemp ? 'i-lucide-user' : undefined"
            :chip="isTemp ? { color: 'warning', position: 'top-right', inset: true, size: '2xs' } : undefined"
          />
          <span class="truncate max-w-[100px] hidden sm:inline-block">{{ user.nickname }}</span>
        </div>
        <UIcon name="i-lucide-chevron-down" class="size-4 text-neutral-400 dark:text-neutral-500" />
      </UButton>
      
      <template #header>
        <div class="p-2">
          <div class="flex items-center gap-3 mb-2">
            <UAvatar
              :src="user.avatar_url"
              :alt="user.nickname"
              size="md"
              :icon="isTemp ? 'i-lucide-user' : undefined"
              :chip="isTemp ? { color: 'warning', position: 'top-right', inset: true } : undefined"
            />
            <div class="min-w-0 flex-1">
              <div class="font-medium truncate">{{ user.nickname }}</div>
              <div class="text-sm text-neutral-500 dark:text-neutral-400 truncate flex items-center gap-1">
                <span :class="isTemp ? 'text-warning-500' : 'text-success-500'">●</span>
                {{ isTemp ? t('enums.userType.temp') : t('enums.userType.normal') }}
              </div>
            </div>
          </div>
          
          <UButton
            v-if="isTemp"
            to="/auth?mode=register&upgrade=1"
            color="warning"
            variant="soft"
            size="xs"
            block
            icon="i-lucide-arrow-up-circle"
          >
            {{ t('userMenu.upgradeAccount') }}
          </UButton>
        </div>
      </template>
    </UDropdownMenu>
</template>

<script setup lang="ts">
import { useAuthService } from '~/services/auth.service'
import type { DropdownMenuItem } from '@nuxt/ui'
import type { UserDetailData } from '~/types/client'
import type { AppAuthData } from '~/types/api'
import { resetAppForAccountChange } from '~/utils/account'
import { useHydratedStore } from '~/composables/useHydratedStore'
import type { AuthSession } from '~/stores/auth.store'

const feedback = useFeedbackStore()
const authService = useAuthService()
const authStore = useAuthStore()
const { t } = useI18n()

// 使用 useHydratedStore 确保 SSR/CSR 一致，避免 hydration mismatch
const user = useHydratedStore(authStore, 'user', null)
const accountsList = useHydratedStore(authStore, 'accountsList', [] as AuthSession[])
const isTemp = computed(() => user.value?.user_type && isTempUser(user.value.user_type))
const isCreatingTemp = ref(false)

const otherAccounts = computed(() => {
  const list = accountsList.value || []
  return list.filter((a): a is AuthSession => !!a && !!a.user && a.user.user_id !== user.value?.user_id)
})

const createTempUser = async () => {
  try {
    isCreatingTemp.value = true
    const response = (await authService.guest({ nickname: `${t('userMenu.guestPrefix')}${Date.now().toString(36).slice(-4)}`, client_type: 'app' })) as unknown as AppAuthData
    authStore.setAccountSession(
      { user_id: response.user_id, nickname: response.nickname },
      response.access_token,
      response.refresh_token ?? null,
      new Date(response.access_token_expires_at).getTime(),
      new Date(response.refresh_token_expires_at).getTime()
    )
    feedback.success(t('userMenu.toasts.welcome'))
    navigateTo('/')
  } catch (err: unknown) {
    feedback.error(t('userMenu.toasts.createFailed'), getErrorMessage(err))
  } finally {
    isCreatingTemp.value = false
  }
}

const handleSwitchAccount = async (userId: string) => {
  try {
    await authStore.switchToAccount(userId)
    feedback.success(t('userMenu.toasts.switchSuccess'))
    resetAppForAccountChange()
    if (import.meta.client) window.location.reload()
  } catch (err: unknown) {
    feedback.error(t('userMenu.toasts.switchFailed'), getErrorMessage(err))
  }
}

const handleLogout = async () => {
  try {
    await authService.logout()
    authStore.logoutCurrent()
    feedback.info(t('userMenu.toasts.loggedOut'))
    resetAppForAccountChange()
    if (import.meta.client) window.location.reload()
  } catch (err: unknown) {
    authStore.logoutCurrent()
    feedback.error(t('userMenu.toasts.logoutError'), getErrorMessage(err))
    resetAppForAccountChange()
    if (import.meta.client) window.location.reload()
  }
}

const handleLock = () => {
  authStore.lock()
  feedback.info(t('userMenu.toasts.locked'))
  resetAppForAccountChange()
  if (import.meta.client) window.location.reload()
}

const menuItems = computed((): DropdownMenuItem[][] => {
  if (!user.value) return []

  const items: DropdownMenuItem[][] = [
    [
      { label: t('userMenu.profile'), icon: 'i-lucide-user', onSelect: () => void navigateTo('/profile') },
      { label: t('userMenu.settings'), icon: 'i-lucide-settings', onSelect: () => void navigateTo('/settings') }
    ]
  ]

  if (otherAccounts.value.length > 0) {
    items.push([
      { label: t('userMenu.switchAccount'), icon: 'i-lucide-arrow-left-right', disabled: true },
      ...otherAccounts.value.map((acc: AuthSession) => ({
        label: acc.user.nickname,
        icon: isTempUser(acc.user.user_type ?? 'normal') ? 'i-lucide-clock' : 'i-lucide-circle-user',
        onSelect: () => handleSwitchAccount(String(acc.user.user_id))
      })),
      { label: t('userMenu.addAccount'), icon: 'i-lucide-circle-plus', onSelect: () => void navigateTo('/auth?mode=login&add=1') }
    ])
  }

  items.push([
    { label: t('userMenu.lock'), icon: 'i-lucide-lock', onSelect: handleLock },
    { label: t('userMenu.logout'), icon: 'i-lucide-log-out', color: 'error', onSelect: handleLogout }
  ])

  return items
})
</script>
