<template>
  <UModal
    title="邀请成员"
    description="从好友列表中选择要邀请的成员"
  >
    <template #body>
      <UInput
        v-model="searchQuery"
        placeholder="搜索好友..."
        icon="i-lucide-search"
        :disabled="isLoading"
        class="mb-4"
      />

      <!-- 加载状态 -->
      <template v-if="isLoading">
        <USkeleton
          v-for="i in 3"
          :key="i"
          class="h-16 mb-2"
        />
      </template>

      <!-- 好友列表 -->
      <section
        v-else-if="filteredFriends.length > 0"
        ref="listEl"
        class="max-h-96 overflow-y-auto space-y-2"
        @scroll="onListScroll"
      >
        <UCard
          v-for="friend in filteredFriends"
          :key="friend.user_id"
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
                <UBadge v-if="friend.remark" color="neutral" variant="soft" size="xs">
                  备注: {{ friend.remark }}
                </UBadge>
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

        <!-- 回到顶部（列表很长时） -->
        <div v-if="showScrollTop" class="sticky bottom-3 flex justify-end">
          <UButton
            variant="soft"
            color="neutral"
            size="sm"
            icon="i-lucide-arrow-up"
            @click="scrollToTop"
          />
        </div>
      </section>

      <UAlert
        v-else
        icon="i-lucide-users-round"
        color="neutral"
        variant="soft"
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
const feedback = useFeedbackStore()

const groupsService = useGroupsService()
const userService = useUsersService()

const searchQuery = ref('')
const friends = ref<Friend[]>([])
const isLoading = ref(false)
const invitedUsers = ref(new Set<number>())
const invitingUsers = ref(new Set<number>())

const listEl = ref<HTMLElement | null>(null)
const showScrollTop = ref(false)

const onListScroll = () => {
  const el = listEl.value
  if (!el) return
  showScrollTop.value = el.scrollTop > 160
}

const scrollToTop = () => {
  listEl.value?.scrollTo({ top: 0, behavior: 'smooth' })
}

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
    feedback.error('加载失败', (error as Error).message)
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
    feedback.success('邀请成功', `已向 ${friend.nickname} 发送邀请`)
    props.onSuccess?.()
  } catch (error) {
    console.error('邀请失败:', error)
    feedback.error('邀请失败', (error as Error).message)
  } finally {
    invitingUsers.value.delete(friend.user_id)
  }
}

onMounted(() => loadFriends())
</script>
