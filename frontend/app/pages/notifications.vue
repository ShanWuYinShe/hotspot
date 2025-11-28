<template>
  <div class="min-h-screen bg-neutral-50 dark:bg-neutral-900">
    <div class="max-w-4xl mx-auto px-4 py-8">
      <!-- 标题 -->
      <div class="mb-6">
        <h1 class="text-2xl font-bold text-neutral-900 dark:text-white mb-2">
          通知中心
        </h1>
        <p class="text-neutral-600 dark:text-neutral-400">
          查看所有系统通知和消息提醒
        </p>
      </div>

      <!-- 标签页 -->
      <UTabs v-model="selectedTab" :items="tabs" class="mb-6" />

      <!-- 操作栏 -->
      <div class="flex justify-between items-center mb-4">
        <div class="flex gap-2">
          <UButton
            v-if="unreadCount > 0"
            variant="ghost"
            size="sm"
            icon="i-heroicons-check"
            @click="markAllAsRead"
            :loading="isMarkingRead"
          >
            全部标记已读
          </UButton>
        </div>
        <UBadge v-if="unreadCount > 0" color="primary" variant="soft">
          {{ unreadCount }} 条未读
        </UBadge>
      </div>

      <!-- 通知列表 -->
      <div v-if="loading" class="space-y-3">
        <USkeleton class="h-24" v-for="i in 5" :key="i" />
      </div>

      <div v-else-if="notifications.length > 0" class="space-y-3">
        <UCard
          v-for="notification in notifications"
          :key="notification.id"
          :class="[
            'transition-colors',
            notification.is_read ? 'opacity-70' : ''
          ]"
        >
          <div class="flex gap-4">
            <!-- 图标 -->
            <div class="flex-shrink-0">
              <div
                :class="[
                  'w-10 h-10 rounded-full flex items-center justify-center',
                  getSeverityColor(notification.severity)
                ]"
              >
                <UIcon :name="getIcon(notification.type)" class="w-5 h-5" />
              </div>
            </div>

            <!-- 内容 -->
            <div class="flex-1 min-w-0">
              <h3
                v-if="notification.title"
                class="font-semibold text-neutral-900 dark:text-white mb-1"
              >
                {{ notification.title }}
              </h3>
              <p class="text-sm text-neutral-600 dark:text-neutral-400 mb-2">
                {{ notification.content }}
              </p>
              <div class="flex items-center gap-4 text-xs text-neutral-500">
                <span>{{ formatTime(notification.created_at) }}</span>
                <UBadge
                  v-if="notification.category"
                  :color="getCategoryColor(notification.category)"
                  variant="soft"
                  size="xs"
                >
                  {{ getCategoryLabel(notification.category) }}
                </UBadge>
              </div>
            </div>

            <!-- 操作 -->
            <div class="flex-shrink-0 flex items-start gap-2">
              <UButton
                v-if="!notification.is_read"
                variant="ghost"
                size="xs"
                icon="i-heroicons-check"
                @click="markAsRead(notification.id)"
              />
              <UButton
                variant="ghost"
                size="xs"
                icon="i-heroicons-trash"
                color="error"
                @click="deleteNotification(notification.id)"
              />
            </div>
          </div>
        </UCard>
      </div>

      <EmptyState
        v-else
        icon="i-heroicons-bell"
        title="暂无通知"
        description="您的通知列表是空的"
      />

      <!-- 加载更多 -->
      <div v-if="hasMore" class="mt-6 text-center">
        <UButton
          variant="ghost"
          @click="loadMore"
          :loading="loadingMore"
        >
          加载更多
        </UButton>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { useInboxService } from '~/services/inbox.service'
import type { InboxItemData } from '~/types/api'

definePageMeta({
  requiresAuth: true
})

const inboxService = useInboxService()
const toast = useToast()

const selectedTab = ref(0)
const tabs = [
  { label: '全部', value: 'all' },
  { label: '社交', value: 'social' },
  { label: '系统', value: 'system' },
  { label: '内容', value: 'content' }
]

const notifications = ref<InboxItemData[]>([])
const loading = ref(false)
const loadingMore = ref(false)
const isMarkingRead = ref(false)
const hasMore = ref(false)
const nextCursor = ref<string>()
const unreadCount = ref(0)

// 获取通知列表
const loadNotifications = async (loadMore = false) => {
  if (loadMore) {
    loadingMore.value = true
  } else {
    loading.value = true
    notifications.value = []
  }

  try {
    const category = tabs[selectedTab.value].value
    const params: any = {
      limit: 20,
      cursor: loadMore ? nextCursor.value : undefined
    }

    if (category !== 'all') {
      params.category = category
    }

    const response = await inboxService.getList(params)
    
    if (loadMore) {
      notifications.value.push(...response.items)
    } else {
      notifications.value = response.items
    }

    hasMore.value = response.pagination.has_more
    nextCursor.value = response.pagination.next_cursor

    // 获取未读数
    const unreadResponse = await inboxService.getUnreadCount()
    unreadCount.value = unreadResponse.total
  } catch (error) {
    console.error('加载通知失败:', error)
    toast.add({
      title: '加载失败',
      description: (error as Error).message,
      color: 'error'
    })
  } finally {
    loading.value = false
    loadingMore.value = false
  }
}

// 标记所有为已读
const markAllAsRead = async () => {
  isMarkingRead.value = true
  try {
    await inboxService.markAsRead({})
    unreadCount.value = 0
    notifications.value.forEach(n => n.is_read = true)
    toast.add({
      title: '已标记全部为已读',
      color: 'success'
    })
  } catch (error) {
    console.error('标记失败:', error)
    toast.add({
      title: '操作失败',
      description: (error as Error).message,
      color: 'error'
    })
  } finally {
    isMarkingRead.value = false
  }
}

// 标记单个为已读
const markAsRead = async (id: string) => {
  try {
    await inboxService.markAsRead({ ids: [id] })
    const notification = notifications.value.find(n => n.id === id)
    if (notification) {
      notification.is_read = true
      unreadCount.value = Math.max(0, unreadCount.value - 1)
    }
  } catch (error) {
    console.error('标记失败:', error)
  }
}

// 删除通知
const deleteNotification = async (id: string) => {
  try {
    await inboxService.remove({ ids: [id] })
    notifications.value = notifications.value.filter(n => n.id !== id)
    toast.add({
      title: '已删除',
      color: 'success'
    })
  } catch (error) {
    console.error('删除失败:', error)
    toast.add({
      title: '删除失败',
      description: (error as Error).message,
      color: 'error'
    })
  }
}

// 加载更多
const loadMore = () => {
  loadNotifications(true)
}

// 获取图标
const getIcon = (type: string) => {
  const icons: Record<string, string> = {
    friend_request: 'i-heroicons-user-plus',
    friend_accepted: 'i-heroicons-user-group',
    group_invite: 'i-heroicons-user-group',
    message_mention: 'i-heroicons-at-symbol',
    moment_like: 'i-heroicons-heart',
    moment_comment: 'i-heroicons-chat-bubble-left',
    comment_reply: 'i-heroicons-chat-bubble-left-right',
    system_announcement: 'i-heroicons-megaphone'
  }
  return icons[type] || 'i-heroicons-bell'
}

// 获取严重程度颜色
const getSeverityColor = (severity?: string) => {
  switch (severity) {
    case 'error':
      return 'bg-error-100 dark:bg-error-900/30 text-error-600'
    case 'warning':
      return 'bg-warning-100 dark:bg-warning-900/30 text-warning-600'
    default:
      return 'bg-primary-100 dark:bg-primary-900/30 text-primary-600'
  }
}

// 获取分类颜色
const getCategoryColor = (category: string) => {
  switch (category) {
    case 'social':
      return 'primary'
    case 'system':
      return 'warning'
    case 'content':
      return 'success'
    default:
      return 'neutral'
  }
}

// 获取分类标签
const getCategoryLabel = (category: string) => {
  const labels: Record<string, string> = {
    social: '社交',
    system: '系统',
    content: '内容'
  }
  return labels[category] || category
}

// 格式化时间
const formatTime = (time: string) => {
  const date = new Date(time)
  const now = new Date()
  const diff = now.getTime() - date.getTime()
  const minutes = Math.floor(diff / 60000)
  const hours = Math.floor(minutes / 60)
  const days = Math.floor(hours / 24)

  if (minutes < 1) return '刚刚'
  if (minutes < 60) return `${minutes}分钟前`
  if (hours < 24) return `${hours}小时前`
  if (days < 7) return `${days}天前`
  return date.toLocaleDateString('zh-CN')
}

// 监听标签页切换
watch(selectedTab, () => {
  loadNotifications()
})

// 初始加载
onMounted(() => {
  loadNotifications()
})
</script>



