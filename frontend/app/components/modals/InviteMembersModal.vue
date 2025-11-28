<template>
  <UModal size="lg">
    <template #body>
      <div class="p-6">
        <!-- 标题 -->
        <div class="flex items-center justify-between mb-6">
          <div class="flex items-center gap-3">
            <UIcon name="i-heroicons-user-plus" class="w-6 h-6 text-neutral-600 dark:text-neutral-300" />
            <h3 class="text-lg font-semibold text-neutral-900 dark:text-white">
              邀请成员
            </h3>
          </div>
          <UButton
            variant="ghost"
            icon="i-heroicons-x-mark"
            size="sm"
            @click="handleClose"
          />
        </div>

        <!-- 搜索好友 -->
        <div class="mb-4">
          <UInput
            v-model="searchQuery"
            placeholder="搜索好友..."
            icon="i-heroicons-magnifying-glass"
            :disabled="isLoading"
          />
        </div>

        <!-- 好友列表 -->
        <div v-if="isLoading" class="space-y-2">
          <USkeleton class="h-16" v-for="i in 3" :key="i" />
        </div>

        <div v-else-if="filteredFriends.length > 0" class="max-h-96 overflow-y-auto space-y-2">
          <div
            v-for="friend in filteredFriends"
            :key="friend.user_id"
            class="flex items-center justify-between p-3 rounded-lg border border-neutral-200 dark:border-neutral-700 hover:bg-neutral-50 dark:hover:bg-neutral-800 transition-colors"
          >
            <div class="flex items-center gap-3">
              <UAvatar
                :src="friend.avatar_url"
                :alt="friend.nickname"
                size="md"
              />
              <div>
                <div class="font-medium text-neutral-900 dark:text-white">
                  {{ friend.nickname }}
                </div>
                <div v-if="friend.remark" class="text-xs text-neutral-500 dark:text-neutral-400">
                  备注: {{ friend.remark }}
                </div>
              </div>
            </div>

            <UButton
              v-if="!invitedUsers.has(friend.user_id)"
              size="sm"
              @click="inviteFriend(friend)"
              :loading="invitingUsers.has(friend.user_id)"
            >
              邀请
            </UButton>
            <UBadge v-else color="success" variant="soft">
              已邀请
            </UBadge>
          </div>
        </div>

        <EmptyState
          v-else
          icon="i-heroicons-user-group"
          title="暂无好友"
          description="您还没有添加好友"
        />

        <!-- 操作按钮 -->
        <div class="flex gap-3 pt-4 mt-6 border-t border-neutral-200 dark:border-neutral-700">
          <UButton
            variant="ghost"
            class="flex-1"
            @click="handleClose"
            :disabled="isLoading"
          >
            关闭
          </UButton>
        </div>
      </div>
    </template>
  </UModal>
</template>

<script setup lang="ts">
import { useGroupsService } from '~/services/groups.service'
import { useUserService } from '~/services/users.service'

interface Friend {
  user_id: string
  nickname: string
  avatar_url?: string
  remark?: string
}

interface Props {
  groupId: string
  onSuccess?: () => void
  onClose?: () => void
}

const props = defineProps<Props>()
const toast = useToast()

const groupsService = useGroupsService()
const userService = useUserService()

const searchQuery = ref('')
const friends = ref<Friend[]>([])
const isLoading = ref(false)
const invitedUsers = ref(new Set<string>())
const invitingUsers = ref(new Set<string>())

// 过滤好友列表
const filteredFriends = computed(() => {
  if (!searchQuery.value.trim()) {
    return friends.value
  }
  
  const query = searchQuery.value.toLowerCase()
  return friends.value.filter(friend => 
    friend.nickname.toLowerCase().includes(query) ||
    friend.remark?.toLowerCase().includes(query)
  )
})

// 加载好友列表
const loadFriends = async () => {
  isLoading.value = true
  try {
    const response = await userService.getFriends({
      status: 'accepted',
      limit: 100
    })
    
    friends.value = response.items.map(item => ({
      user_id: item.user.user_id,
      nickname: item.user.nickname,
      avatar_url: item.user.avatar_url,
      remark: item.remark
    }))
  } catch (error) {
    console.error('加载好友列表失败:', error)
    toast.add({
      title: '加载失败',
      description: (error as Error).message,
      color: 'error'
    })
  } finally {
    isLoading.value = false
  }
}

// 邀请好友
const inviteFriend = async (friend: Friend) => {
  invitingUsers.value.add(friend.user_id)
  
  try {
    await groupsService.inviteMember({
      group_chat_id: props.groupId,
      user_id: friend.user_id,
      message: `邀请您加入群组`
    })
    
    invitedUsers.value.add(friend.user_id)
    
    toast.add({
      title: '邀请成功',
      description: `已向 ${friend.nickname} 发送邀请`,
      color: 'success'
    })
    
    props.onSuccess?.()
  } catch (error) {
    console.error('邀请失败:', error)
    toast.add({
      title: '邀请失败',
      description: (error as Error).message,
      color: 'error'
    })
  } finally {
    invitingUsers.value.delete(friend.user_id)
  }
}

const handleClose = () => {
  if (!isLoading.value) {
    props.onClose?.()
  }
}

// 初始化
onMounted(() => {
  loadFriends()
})
</script>



