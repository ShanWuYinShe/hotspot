<template>
  <UModal
    size="2xl"
    title="黑名单管理"
    :description="`共 ${blockedUsers.length} 个被拉黑用户`"
  >
    <template #body="{ close }">
      <UInput
        v-model="searchQuery"
        placeholder="搜索被拉黑的用户..."
        icon="i-heroicons-magnifying-glass"
        size="lg"
        class="mb-4 w-full"
      />

      <section class="max-h-96 overflow-y-auto space-y-3">
        <!-- 用户列表 -->
        <UCard
          v-for="user in filteredUsers"
          :key="user.user_id"
          :ui="{ body: 'p-4' }"
        >
          <section class="flex items-center justify-between gap-4">
            <header class="flex items-center gap-3 min-w-0">
              <UAvatar
                :alt="user.nickname"
                :src="user.avatar_url"
                size="md"
              >
                {{ user.nickname?.charAt(0) || 'U' }}
              </UAvatar>
              <article class="min-w-0">
                <p class="font-medium truncate">{{ user.nickname }}</p>
                <UBadge
                  variant="subtle"
                  color="neutral"
                  size="xs"
                >
                  ID: {{ user.user_id }}
                </UBadge>
              </article>
            </header>

            <UButtonGroup size="sm">
              <UButton
                variant="outline"
                @click="viewUserProfile(user, close)"
              >
                查看资料
              </UButton>
              <UButton
                color="error"
                variant="outline"
                :loading="unblockingUsers.has(user.user_id)"
                @click="unblockUser(user)"
              >
                解除拉黑
              </UButton>
            </UButtonGroup>
          </section>
        </UCard>

        <!-- 空状态 -->
        <UAlert
          v-if="!loading && filteredUsers.length === 0"
          icon="i-heroicons-user-minus"
          color="neutral"
          variant="subtle"
          title="黑名单为空"
          description="您还没有拉黑任何用户"
        />

        <!-- 加载状态 -->
        <template v-if="loading">
          <USkeleton
            v-for="i in 3"
            :key="i"
            class="h-20 rounded-lg"
          />
        </template>
      </section>
    </template>

    <template #footer="{ close }">
      <UButton
        variant="outline"
        icon="i-heroicons-arrow-path"
        :loading="loading"
        @click="loadBlockedUsers"
      >
        刷新
      </UButton>
      <UButton
        variant="outline"
        @click="close"
      >
        关闭
      </UButton>
    </template>
  </UModal>
</template>

<script setup lang="ts">
import { useUsersService } from '~/services/users.service'
import type { UserInfo } from '~/types/api'
import { getErrorMessage } from '~/utils/error'

const authStore = useAuthStore()
const loading = ref(false)
const blockedUsers = ref<UserInfo[]>([])
const searchQuery = ref('')
const unblockingUsers = ref(new Set<number>())
const toast = useToast()

const filteredUsers = computed(() => {
  if (!searchQuery.value.trim()) return blockedUsers.value
  const query = searchQuery.value.toLowerCase()
  return blockedUsers.value.filter(user => user.nickname.toLowerCase().includes(query))
})

const loadBlockedUsers = async () => {
  loading.value = true
  try {
    const userService = useUsersService()
    const response = await userService.getBlockedUsers()
    blockedUsers.value = response.items || []
  } catch (err) {
    console.error('加载拉黑用户列表失败:', err)
    toast.add({ color: 'error', title: '加载失败', description: '无法获取拉黑用户列表' })
  } finally {
    loading.value = false
  }
}

const viewUserProfile = (user: UserInfo, close: () => void) => {
  close()
  if (user.user_id === authStore.user?.user_id) {
    navigateTo('/profile')
  } else {
    navigateTo(`/users/${user.user_id}`)
  }
}

const unblockUser = async (user: UserInfo) => {
  const { showConfirm } = useModal()
  const confirmed = await showConfirm({
    title: '解除拉黑',
    message: `确定要解除对 ${user.nickname} 的拉黑吗？`,
    confirmText: '确认解除',
    cancelText: '取消',
    confirmColor: 'primary'
  })

  if (!confirmed) return

  unblockingUsers.value.add(user.user_id)

  try {
    const userService = useUsersService()
    await userService.unblockUser({ id: user.user_id })

    const index = blockedUsers.value.findIndex(u => u.user_id === user.user_id)
    if (index > -1) blockedUsers.value.splice(index, 1)

    toast.add({ color: 'success', title: '解除成功', description: `已解除对 ${user.nickname} 的拉黑` })
  } catch (err: unknown) {
    console.error('解除拉黑失败:', err)
    toast.add({ color: 'error', title: '解除失败', description: getErrorMessage(err, '无法解除拉黑，请重试') })
  } finally {
    unblockingUsers.value.delete(user.user_id)
  }
}

onMounted(() => loadBlockedUsers())
</script>
