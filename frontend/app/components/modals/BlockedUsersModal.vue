<template>
  <UModal size="2xl">
    <template #body>
      <div class="p-6">
        <!-- 标题 -->
        <div class="flex items-center justify-between mb-6">
          <div class="flex items-center gap-3">
            <UIcon name="i-heroicons-user-minus" class="w-6 h-6 text-red-500" />
            <h3 class="text-lg font-semibold text-neutral-900 dark:text-white">
              黑名单管理
            </h3>
          </div>
          <UButton
            variant="ghost"
            icon="i-heroicons-x-mark"
            size="sm"
            @click="handleClose"
          />
        </div>

        <!-- 搜索栏 -->
        <div class="mb-6">
          <UInput
            v-model="searchQuery"
            placeholder="搜索被拉黑的用户..."
            icon="i-heroicons-magnifying-glass"
            size="lg"
            class="w-full"
          />
        </div>

        <!-- 拉黑用户列表 -->
        <div class="max-h-96 overflow-y-auto">
          <div v-if="filteredUsers.length > 0" class="space-y-3">
            <div
              v-for="user in filteredUsers"
              :key="user.user_id"
              class="flex items-center justify-between p-4 border border-neutral-200 dark:border-neutral-700 rounded-lg hover:bg-neutral-50 dark:hover:bg-neutral-800 transition-colors"
            >
              <div class="flex items-center gap-3">
                <UAvatar
                  :alt="user.nickname"
                  :src="user.avatar_url"
                  size="md"
                >
                  {{ user.nickname?.charAt(0) || 'U' }}
                </UAvatar>
                
                <div>
                  <h4 class="font-medium text-neutral-900 dark:text-white">
                    {{ user.nickname }}
                  </h4>
                  <p class="text-sm text-neutral-500 dark:text-neutral-400">
                    已加入黑名单
                  </p>
                </div>
              </div>
              
              <div class="flex items-center gap-2">
                <UButton
                  variant="outline"
                  size="sm"
                  @click="viewUserProfile(user)"
                >
                  查看资料
                </UButton>
                <UButton
                  color="red"
                  variant="outline"
                  size="sm"
                  @click="unblockUser(user)"
                  :loading="unblockingUsers.has(user.user_id)"
                >
                  解除拉黑
                </UButton>
              </div>
            </div>
          </div>

          <!-- 空状态 -->
          <div v-else-if="!loading" class="text-center py-8">
            <UIcon name="i-heroicons-user-minus" class="w-12 h-12 text-neutral-400 dark:text-neutral-500 mx-auto mb-3" />
            <h4 class="text-sm font-medium text-neutral-900 dark:text-white mb-1">
              黑名单为空
            </h4>
            <p class="text-xs text-neutral-600 dark:text-neutral-400">
              您还没有拉黑任何用户
            </p>
          </div>

          <!-- 加载状态 -->
          <div v-if="loading" class="space-y-3">
            <USkeleton class="h-16" v-for="i in 3" :key="i" />
          </div>
        </div>

        <!-- 底部操作 -->
        <div class="flex justify-between items-center mt-6 pt-4 border-t border-neutral-200 dark:border-neutral-700">
          <div class="text-sm text-neutral-600 dark:text-neutral-400">
            共 {{ blockedUsers.length }} 个被拉黑用户
          </div>
          <div class="flex gap-2">
            <UButton
              variant="outline"
              size="sm"
              icon="i-heroicons-arrow-path"
              @click="loadBlockedUsers"
              :loading="loading"
            >
              刷新
            </UButton>
            <UButton
              variant="outline"
              size="sm"
              @click="handleClose"
            >
              关闭
            </UButton>
          </div>
        </div>
      </div>
    </template>
  </UModal>
</template>

<script setup lang="ts">
import { useUsersService } from '~/services/users.service'
import type { UserResponse } from '~/types/api'

interface Props {
  onClose?: () => void
}

const props = defineProps<Props>()

// 状态管理
const loading = ref(false)
const blockedUsers = ref<UserResponse[]>([])
const searchQuery = ref('')
const unblockingUsers = ref(new Set<string>())

// 计算属性
const filteredUsers = computed(() => {
  if (!searchQuery.value.trim()) return blockedUsers.value
  
  const query = searchQuery.value.toLowerCase()
  return blockedUsers.value.filter(user => 
    user.nickname.toLowerCase().includes(query)
  )
})

// 工具函数
const { formatTime } = useFormatters()
const toast = useToast()
const { showConfirmModal } = useModal()

// Toast 辅助函数
const success = (title: string, description?: string) => {
  toast.add({ title, description, color: 'success' })
}
const error = (title: string, description?: string) => {
  toast.add({ title, description, color: 'error' })
}

// 方法
const loadBlockedUsers = async () => {
  loading.value = true
  try {
    const userService = useUsersService()
    const response = await userService.getBlockedUsers()
    blockedUsers.value = response.items || []
  } catch (err) {
    console.error('加载拉黑用户列表失败:', err)
    error('加载失败', '无法获取拉黑用户列表')
  } finally {
    loading.value = false
  }
}

const viewUserProfile = (user: any) => {
  // 关闭当前Modal并导航到用户详情页
  handleClose()
  navigateTo(`/users/${user.user_id}`)
}

const unblockUser = async (user: any) => {
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
    await userService.unblockUser({ user_id: user.user_id })
    
    // 从列表中移除
    const index = blockedUsers.value.findIndex(u => u.user_id === user.user_id)
    if (index > -1) {
      blockedUsers.value.splice(index, 1)
    }
    
    success('解除成功', `已解除对 ${user.nickname} 的拉黑`)
  } catch (err: any) {
    console.error('解除拉黑失败:', err)
    error('解除失败', err?.message || '无法解除拉黑，请重试')
  } finally {
    unblockingUsers.value.delete(user.user_id)
  }
}

const handleClose = () => {
  props.onClose?.()
}

// 页面加载
onMounted(() => {
  loadBlockedUsers()
})
</script>
