<template>
  <UModal
    title="邀请成员"
    description="从好友列表中选择要邀请的成员"
    @update:open="(val) => !val && emit('update:open', false)"
  >
    <template #body>
      <div class="sticky top-0 z-10 mb-4 bg-white dark:bg-neutral-900">
        <UInput
          v-model="searchQuery"
          placeholder="搜索好友..."
          icon="i-lucide-search"
          :disabled="isLoading"
        />
      </div>

      <!-- 加载状态 -->
      <div v-if="isLoading" class="space-y-3">
        <USkeleton shape="rectangle" class="h-16 w-full" v-for="i in 3" :key="i" />
      </div>

      <!-- 好友列表 -->
      <div
        v-else-if="filteredFriends.length > 0"
        ref="listEl"
        class="max-h-[50vh] overflow-y-auto space-y-2 pr-1"
        @scroll="onListScroll"
      >
        <UCard
          v-for="friend in filteredFriends"
          :key="friend.user_id"
          variant="outline"
        >
          <div class="flex items-center justify-between gap-3">
            <div class="flex items-center gap-3 min-w-0">
              <UAvatar
                :src="friend.avatar_url"
                :alt="friend.nickname"
                size="md"
              />
              <div class="min-w-0">
                <p class="font-semibold text-neutral-900 dark:text-white text-sm truncate">{{ friend.nickname }}</p>
                <span v-if="friend.remark" class="text-neutral-400 dark:text-neutral-500 text-xs mt-0.5 block">
                  备注: {{ friend.remark }}
                </span>
              </div>
            </div>

            <UButton
              v-if="!invitedUsers.has(friend.user_id)"
              size="xs"
              variant="soft"
              color="primary"
              :loading="invitingUsers.has(friend.user_id)"
              @click="inviteFriend(friend)"
            >
              邀请
            </UButton>
            <UBadge
              v-else
              color="success"
              variant="soft"
              size="xs"
              icon="i-lucide-check"
            >
              已邀请
            </UBadge>
          </div>
        </UCard>
      </div>

      <UEmpty
        v-else
        icon="i-lucide-users-round"
        title="暂无好友"
        description="您还没有添加好友，或者没有找到匹配的好友"
        class="py-8"
      />
      
      <!-- 回到顶部按钮 -->
      <div 
        v-show="showScrollTop" 
        class="absolute bottom-4 right-4 z-20 transition-opacity duration-300 pointer-events-none"
        :class="showScrollTop ? 'opacity-100 pointer-events-auto' : 'opacity-0'"
      >
        <UButton
          variant="soft"
          color="neutral"
          size="sm"
          icon="i-lucide-arrow-up"
          :square="true"
          @click="scrollToTop"
        />
      </div>
    </template>

    <template #footer>
      <UButton
        variant="ghost"
        color="neutral"
        class="w-full"
        :disabled="isLoading"
        @click="emit('update:open', false)"
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
}

const props = defineProps<Props>()
const emit = defineEmits<{
    'update:open': [value: boolean]
    'success': []
}>()

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
    feedback.error('加载失败')
  } finally {
    isLoading.value = false
  }
}

const inviteFriend = async (friend: Friend) => {
  invitingUsers.value.add(friend.user_id)

  try {
    await groupsService.inviteMember({
      group_chat_id: props.groupId,
      user_id: friend.user_id,
      message: '邀请您加入群组'
    })

    invitedUsers.value.add(friend.user_id)
    feedback.success(`已邀请 ${friend.nickname}`)
    emit('success')
  } catch (error) {
    feedback.error('邀请失败')
  } finally {
    invitingUsers.value.delete(friend.user_id)
  }
}

onMounted(() => loadFriends())
</script>
