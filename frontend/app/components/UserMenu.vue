<template>
  <!-- 用户菜单 - 使用 UPopover 实现 hover 触发的菜单 -->
  <UPopover 
    mode="hover" 
    :popper="{ placement: 'bottom-end', offsetDistance: 8 }"
  >
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

    <template #content>
      <div class="w-56 p-1">
        <!-- 未登录用户菜单 -->
        <template v-if="!user">
          <UButton
            variant="ghost"
            size="sm"
            class="w-full justify-start px-3 py-2 text-sm"
            @click="createTempUser"
          >
            <UIcon name="i-heroicons-rocket-launch" class="w-4 h-4 mr-3" />
            {{ t('userMenu.tempExperience') }}
          </UButton>
          <UButton
            variant="ghost"
            size="sm"
            class="w-full justify-start px-3 py-2 text-sm"
            @click="navigateTo('/auth')"
          >
            <UIcon name="i-heroicons-arrow-right-end-on-rectangle" class="w-4 h-4 mr-3" />
            {{ t('userMenu.loginRegister') }}
          </UButton>
        </template>

        <!-- 已登录用户菜单 -->
        <template v-else>
          <!-- 个人功能 -->
          <UButton
            variant="ghost"
            size="sm"
            class="w-full justify-start px-3 py-2 text-sm"
            @click="navigateTo('/profile')"
          >
            <UIcon name="i-heroicons-identification" class="w-4 h-4 mr-3" />
            {{ t('userMenu.profile') }}
          </UButton>
          <UButton
            variant="ghost"
            size="sm"
            class="w-full justify-start px-3 py-2 text-sm"
            @click="navigateTo('/settings')"
          >
            <UIcon name="i-heroicons-cog-6-tooth" class="w-4 h-4 mr-3" />
            {{ t('userMenu.settings') }}
          </UButton>

          <!-- 分隔线 -->
          <div class="h-px bg-neutral-200 dark:bg-neutral-700 my-1" />

          <!-- 社交功能 -->
          <UButton
            variant="ghost"
            size="sm"
            class="w-full justify-start px-3 py-2 text-sm"
            @click="navigateTo('/groups/create')"
          >
            <UIcon name="i-heroicons-user-group" class="w-4 h-4 mr-3" />
            {{ t('userMenu.createGroup') }}
          </UButton>
          <UButton
            variant="ghost"
            size="sm"
            class="w-full justify-start px-3 py-2 text-sm"
            @click="navigateTo('/users/search')"
          >
            <UIcon name="i-heroicons-magnifying-glass" class="w-4 h-4 mr-3" />
            {{ t('userMenu.findFriends') }}
          </UButton>

          <!-- 账户管理 -->
          <template v-if="user.is_temp || otherAccounts.length > 0">
            <div class="h-px bg-neutral-200 dark:bg-neutral-700 my-1" />
            
            <UButton
              v-if="user.is_temp"
              variant="ghost"
              size="sm"
              class="w-full justify-start px-3 py-2 text-sm"
              @click="navigateTo('/auth?upgrade=1')"
            >
              <UIcon name="i-heroicons-arrow-up-circle" class="w-4 h-4 mr-3" />
              {{ t('userMenu.upgradeAccount') }}
            </UButton>
            
            <UButton
              variant="ghost"
              size="sm"
              class="w-full justify-start px-3 py-2 text-sm"
              @click="navigateTo('/auth?add=1')"
            >
              <UIcon name="i-heroicons-plus-circle" class="w-4 h-4 mr-3" />
              {{ t('userMenu.addAccount') }}
            </UButton>

            <!-- 其他账户切换 -->
            <template v-if="otherAccounts.length > 0">
              <UButton
                v-for="account in otherAccounts"
                :key="account.user_id"
                variant="ghost"
                size="sm"
                class="w-full justify-start px-3 py-2 text-sm"
                @click="switchAccount(account.user_id)"
              >
                <UIcon 
                  :name="account.is_temp ? 'i-heroicons-clock' : 'i-heroicons-user-circle'" 
                  class="w-4 h-4 mr-3" 
                />
                {{ account.nickname }}
              </UButton>
            </template>
          </template>

          <!-- 登出 -->
          <div class="h-px bg-neutral-200 dark:bg-neutral-700 my-1" />
          <UButton
            variant="ghost"
            size="sm"
            class="w-full justify-start px-3 py-2 text-sm"
            @click="logout"
          >
            <UIcon name="i-heroicons-arrow-left-start-on-rectangle" class="w-4 h-4 mr-3" />
            {{ t('userMenu.logout') }}
          </UButton>
        </template>
      </div>
    </template>
  </UPopover>
</template>

<script setup lang="ts">
const { t } = useI18n()
const authStore = useAuthStore()

// 当前用户
const user = computed(() => authStore.user)

// 其他账户列表
const otherAccounts = computed(() => 
  authStore.accounts.filter(a => a.user_id !== user.value?.user_id)
)

// 创建临时用户
const createTempUser = async () => {
  try {
    await authStore.createTempUser()
  } catch (error) {
    console.error('创建临时用户失败:', error)
  }
}

// 切换账户
const switchAccount = async (userId: string) => {
  try {
    await authStore.switchToAccount(userId)
  } catch (error) {
    console.error('切换账户失败:', error)
  }
}

// 登出
const logout = () => {
  authStore.logout()
  navigateTo('/auth')
}
</script>