<template>
  <UModal
    size="lg"
    title="邀请成员"
    description="从好友列表中选择要邀请的成员"
  >
    <template #body>
      <UInput
        v-model="searchQuery"
        placeholder="搜索好友..."
        icon="i-heroicons-magnifying-glass"
        :disabled="isLoading"
        class="mb-4"
      />

      <!-- 加载状态 -->
      <template v-if="isLoading">
        <USkeleton
          v-for="i in 3"
          :key="i"
          class="h-16 rounded-lg mb-2"
        />
      </template>

      <!-- 好友列表 -->
      <section
        v-else-if="filteredFriends.length > 0"
        class="max-h-96 overflow-y-auto space-y-2"
      >
        <UCard
          v-for="friend in filteredFriends"
          :key="friend.user_id"
          :ui="{ body: 'p-3' }"
        >
          <section class="flex items-center justify-between gap-3">
            <header class="flex items-center gap-3 min-w-0">
              <UAvatar
                :src="friend.avatar_url"
                :alt="friend.nickname"
                size="md"
              />
              <article class="min-w-0">
                <p class="font-medium truncate">{{ friend.nickname }}</p>
                <p
                  v-if="friend.remark"
                  class="text-xs text-secondary-content"
                >
                  备注: {{ friend.remark }}
                </p>
              </article>
            </header>

            <UButton
              v-if="!invitedUsers.has(friend.user_id)"
              size="sm"
              :loading="invitingUsers.has(friend.user_id)"
              @click="inviteFriend(friend)"
            >
              邀请
            </UButton>
            <UBadge
              v-else
              color="success"
              variant="soft"
            >
              已邀请
            </UBadge>
          </section>
        </UCard>
      </section>

      <UAlert
        v-else
        icon="i-heroicons-user-group"
        color="neutral"
        variant="subtle"
        title="暂无好友"
        description="您还没有添加好友"
      />
    </template>

    <template #footer="{ close }">
      <UButton
        variant="ghost"
        class="flex-1"
        :disabled="isLoading"
        @click="close"
      >
        关闭
      </UButton>
    </template>
  </UModal>
</template>

<script setup lang="ts">
import { useGroupsService } from '~/services/groups.service'
import { useUsersService } from '~/services/users.service'

interface Friend {
  user_id: number
  nickname: string
  avatar_url?: string
  remark?: string
}

interface Props {
  groupId: string
  onSuccess?: () => void
}

const props = defineProps<Props>()
const toast = useToast()

const groupsService = useGroupsService()
const userService = useUsersService()

const searchQuery = ref('')
const friends = ref<Friend[]>([])
const isLoading = ref(false)
const invitedUsers = ref(new Set<number>())
const invitingUsers = ref(new Set<number>())

const filteredFriends = computed(() => {
  if (!searchQuery.value.trim()) return friends.value
  const query = searchQuery.value.toLowerCase()
  return friends.value.filter(friend =>
    friend.nickname.toLowerCase().includes(query) ||
    friend.remark?.toLowerCase().includes(query)
  )
})

const loadFriends = async () => {
  isLoading.value = true
  try {
    const response = await userService.getFriends({ limit: 100 })
    friends.value = response.items.map(item => ({
      user_id: item.user.user_id,
      nickname: item.user.nickname,
      avatar_url: item.user.avatar_url,
      remark: item.remark
    }))
  } catch (error) {
    console.error('加载好友列表失败:', error)
    toast.add({ title: '加载失败', description: (error as Error).message, color: 'error' })
  } finally {
    isLoading.value = false
  }
}

const inviteFriend = async (friend: Friend) => {
  invitingUsers.value.add(friend.user_id)

  try {
    await groupsService.inviteMember({
      id: props.groupId,
      user_id: friend.user_id,
      message: '邀请您加入群组'
    })

    invitedUsers.value.add(friend.user_id)
    toast.add({ title: '邀请成功', description: `已向 ${friend.nickname} 发送邀请`, color: 'success' })
    props.onSuccess?.()
  } catch (error) {
    console.error('邀请失败:', error)
    toast.add({ title: '邀请失败', description: (error as Error).message, color: 'error' })
  } finally {
    invitingUsers.value.delete(friend.user_id)
  }
}

onMounted(() => loadFriends())
</script>
