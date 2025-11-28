<template>
  <UModal size="2xl">
    <template #body>
      <div class="p-6">
        <!-- 标题 -->
        <div class="flex items-center justify-between mb-6">
          <div class="flex items-center gap-3">
            <UIcon name="i-heroicons-user-plus" class="w-6 h-6 text-neutral-600 dark:text-neutral-300" />
            <h3 class="text-lg font-semibold text-neutral-900 dark:text-white">
              好友请求
            </h3>
          </div>
          <UButton
            variant="ghost"
            icon="i-heroicons-x-mark"
            size="sm"
            @click="handleClose"
          />
        </div>

        <!-- 标签页 -->
        <div class="mb-6">
          <UTabs v-model="activeTab" :items="tabItems" />
        </div>

        <!-- 收到的请求 -->
        <div v-if="activeTab === 'received'" class="space-y-4">
          <div v-if="receivedRequests.length > 0">
            <div
              v-for="request in receivedRequests"
              :key="request.id"
              class="flex items-center justify-between p-4 border border-neutral-200 dark:border-neutral-700 rounded-lg hover:bg-neutral-50 dark:hover:bg-neutral-800 transition-colors"
            >
              <div class="flex items-center gap-3">
                <UAvatar
                  :alt="request.from_user.nickname"
                  :src="request.from_user.avatar_url"
                  size="md"
                >
                  {{ request.from_user.nickname?.charAt(0) || 'U' }}
                </UAvatar>
                
                <div>
                  <h4 class="font-medium text-neutral-900 dark:text-white">
                    {{ request.from_user.nickname }}
                  </h4>
                  <p class="text-sm text-neutral-600 dark:text-neutral-400">
                    {{ formatTime(request.created_at) }}
                  </p>
                  <p v-if="request.message" class="text-sm text-neutral-500 dark:text-neutral-400 mt-1">
                    "{{ request.message }}"
                  </p>
                </div>
              </div>
              
              <div class="flex items-center gap-2">
                <UButton
                  color="green"
                  size="sm"
                  @click="acceptRequest(request)"
                  :loading="processingRequests.has(request.id)"
                >
                  接受
                </UButton>
                <UButton
                  color="red"
                  variant="outline"
                  size="sm"
                  @click="rejectRequest(request)"
                  :loading="processingRequests.has(request.id)"
                >
                  拒绝
                </UButton>
              </div>
            </div>
          </div>
          
          <!-- 空状态 -->
          <div v-else class="text-center py-8">
            <UIcon name="i-heroicons-inbox" class="w-12 h-12 text-neutral-400 dark:text-neutral-500 mx-auto mb-3" />
            <h4 class="text-sm font-medium text-neutral-900 dark:text-white mb-1">
              暂无好友请求
            </h4>
            <p class="text-xs text-neutral-600 dark:text-neutral-400">
              还没有人向您发送好友请求
            </p>
          </div>
        </div>

        <!-- 发送的请求 -->
        <div v-if="activeTab === 'sent'" class="space-y-4">
          <div v-if="sentRequests.length > 0">
            <div
              v-for="request in sentRequests"
              :key="request.id"
              class="flex items-center justify-between p-4 border border-neutral-200 dark:border-neutral-700 rounded-lg"
            >
              <div class="flex items-center gap-3">
                <UAvatar
                  :alt="request.to_user.nickname"
                  :src="request.to_user.avatar_url"
                  size="md"
                >
                  {{ request.to_user.nickname?.charAt(0) || 'U' }}
                </UAvatar>
                
                <div>
                  <h4 class="font-medium text-neutral-900 dark:text-white">
                    {{ request.to_user.nickname }}
                  </h4>
                  <p class="text-sm text-neutral-600 dark:text-neutral-400">
                    {{ formatTime(request.created_at) }}
                  </p>
                </div>
              </div>
              
              <div class="flex items-center gap-2">
                <UBadge
                  :color="getStatusColor(request.status)"
                  :label="getStatusLabel(request.status)"
                  size="sm"
                />
                <UButton
                  v-if="request.status === 'pending'"
                  color="red"
                  variant="outline"
                  size="sm"
                  @click="cancelRequest(request)"
                  :loading="processingRequests.has(request.id)"
                >
                  撤销
                </UButton>
              </div>
            </div>
          </div>
          
          <!-- 空状态 -->
          <div v-else class="text-center py-8">
            <UIcon name="i-heroicons-paper-airplane" class="w-12 h-12 text-neutral-400 dark:text-neutral-500 mx-auto mb-3" />
            <h4 class="text-sm font-medium text-neutral-900 dark:text-white mb-1">
              暂无发送记录
            </h4>
            <p class="text-xs text-neutral-600 dark:text-neutral-400">
              您还没有发送过好友请求
            </p>
          </div>
        </div>

        <!-- 加载状态 -->
        <div v-if="loading" class="space-y-4">
          <USkeleton class="h-16" v-for="i in 3" :key="i" />
        </div>

        <!-- 底部操作 -->
        <div class="flex justify-between items-center mt-6 pt-4 border-t border-neutral-200 dark:border-neutral-700">
          <div class="text-sm text-neutral-600 dark:text-neutral-400">
            收到 {{ receivedRequests.length }} 条，发送 {{ sentRequests.length }} 条
          </div>
          <div class="flex gap-2">
            <UButton
              variant="outline"
              size="sm"
              icon="i-heroicons-arrow-path"
              @click="loadRequests"
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
import type { FriendRequestItem } from '~/types/api'

interface Props {
  onClose?: () => void
}

const props = defineProps<Props>()

// 状态管理
const loading = ref(false)
const activeTab = ref('received')
const receivedRequests = ref<FriendRequestItem[]>([])
const sentRequests = ref<FriendRequestItem[]>([])
const processingRequests = ref(new Set<string>())

// 标签页配置
const tabItems = [
  { value: 'received', label: '收到的请求' },
  { value: 'sent', label: '发送的请求' }
]

// 工具函数
const { formatTime } = useFormatters()
const toast = useToast()

// Toast 辅助函数
const success = (title: string, description?: string) => {
  toast.add({ title, description, color: 'success' })
}
const error = (title: string, description?: string) => {
  toast.add({ title, description, color: 'error' })
}

const getStatusColor = (status: string): 'orange' | 'green' | 'red' | 'neutral' => {
  const colors: Record<string, 'orange' | 'green' | 'red' | 'neutral'> = {
    pending: 'orange',
    accepted: 'green',
    rejected: 'red'
  }
  return colors[status] || 'neutral'
}

const getStatusLabel = (status: string): string => {
  const labels: Record<string, string> = {
    pending: '等待回复',
    accepted: '已接受',
    rejected: '已拒绝'
  }
  return labels[status] || status
}

// 方法
const loadRequests = async () => {
  loading.value = true
  try {
    const userService = useUsersService()
    const [receivedResponse, sentResponse] = await Promise.all([
      userService.getFriendRequests({ status: 'received' }),
      userService.getFriendRequests({ status: 'sent' })
    ])
    receivedRequests.value = receivedResponse.items || []
    sentRequests.value = sentResponse.items || []
  } catch (err) {
    console.error('加载好友请求失败:', err)
    error('加载失败', '无法获取好友请求列表')
  } finally {
    loading.value = false
  }
}

const acceptRequest = async (request: FriendRequestItem) => {
  processingRequests.value.add(request.id)
  
  try {
    const userService = useUsersService()
    await userService.acceptFriendRequest({ id: request.id })
    
    // 从列表中移除
    const index = receivedRequests.value.findIndex(r => r.id === request.id)
    if (index > -1) {
      receivedRequests.value.splice(index, 1)
    }
    
    success('已接受好友请求', `您和 ${request.from_user.nickname} 现在是好友了`)
  } catch (err: any) {
    console.error('接受好友请求失败:', err)
    error('操作失败', err?.message || '无法接受好友请求')
  } finally {
    processingRequests.value.delete(request.id)
  }
}

const rejectRequest = async (request: FriendRequestItem) => {
  processingRequests.value.add(request.id)
  
  try {
    const userService = useUsersService()
    await userService.rejectFriendRequest({ id: request.id })
    
    // 从列表中移除
    const index = receivedRequests.value.findIndex(r => r.id === request.id)
    if (index > -1) {
      receivedRequests.value.splice(index, 1)
    }
    
    success('已拒绝好友请求')
  } catch (err: any) {
    console.error('拒绝好友请求失败:', err)
    error('操作失败', err?.message || '无法拒绝好友请求')
  } finally {
    processingRequests.value.delete(request.id)
  }
}

const cancelRequest = async (request: FriendRequestItem) => {
  processingRequests.value.add(request.id)
  
  try {
    const userService = useUsersService()
    await userService.cancelFriendRequest({ id: request.id })
    
    // 从列表中移除
    const index = sentRequests.value.findIndex(r => r.id === request.id)
    if (index > -1) {
      sentRequests.value.splice(index, 1)
    }
    
    success('已撤销好友请求')
  } catch (err: any) {
    console.error('撤销好友请求失败:', err)
    error('操作失败', err?.message || '无法撤销好友请求')
  } finally {
    processingRequests.value.delete(request.id)
  }
}

const handleClose = () => {
  props.onClose?.()
}

// 页面加载
onMounted(() => {
  loadRequests()
})
</script>
