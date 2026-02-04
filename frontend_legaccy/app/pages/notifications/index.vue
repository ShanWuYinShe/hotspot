<template>
  <div class="w-full space-y-4">
    <UCard variant="subtle">
      <template #header>
        <div class="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
          <div class="flex items-center gap-2 min-w-0">
            <UIcon name="i-lucide-bell" :size="20" />
            <span class="truncate">全部通知</span>
            <UBadge v-if="unreadCount > 0" color="error" variant="soft" size="xs">
              {{ unreadCount > 99 ? '99+' : unreadCount }}
            </UBadge>
          </div>

          <div class="flex items-center gap-2 shrink-0">
            <UButton
              v-if="notifications.length > 0"
              variant="outline"
              color="neutral"
              size="sm"
              icon="i-lucide-check"
              @click="markAllAsRead"
            >
              全部已读
            </UButton>
            <UButton
              v-if="notifications.length > 0"
              variant="ghost"
              color="neutral"
              size="sm"
              icon="i-lucide-trash-2"
              @click="clearNotifications"
            >
              清空
            </UButton>
          </div>
        </div>
      </template>

      <div class="flex items-center gap-2 pb-4">
        <UButton
          :variant="filter === 'all' ? 'solid' : 'ghost'"
          color="neutral"
          size="sm"
          @click="filter = 'all'"
        >
          全部
        </UButton>
        <UButton
          :variant="filter === 'unread' ? 'solid' : 'ghost'"
          color="neutral"
          size="sm"
          @click="filter = 'unread'"
        >
          未读
        </UButton>
      </div>

      <UEmpty
        v-if="filteredNotifications.length === 0"
        icon="i-lucide-bell-off"
        title="暂无通知"
        description="新的通知会在这里出现"
      />

      <div v-else class="divide-y divide-neutral-100 dark:divide-neutral-800 -mx-5 sm:-mx-7">
        <NuxtLink
          v-for="n in filteredNotifications"
          :key="n.id"
          :to="getNotificationTo(n)"
          class="block py-4 px-5 sm:px-7 hover:bg-neutral-50 dark:hover:bg-neutral-800/50 transition-colors"
          @click="handleOpen(n)"
        >
          <div class="flex items-start gap-4">
            <div class="bg-neutral-100 dark:bg-neutral-800 p-2 rounded-full shrink-0">
              <UIcon :name="getNotificationIcon(n.type)" class="size-5 text-neutral-600 dark:text-neutral-400" />
            </div>
            
            <div class="flex-1 min-w-0">
              <div class="flex items-center justify-between gap-2 mb-1">
                <span class="text-sm font-medium text-neutral-900 dark:text-white truncate">{{ n.title }}</span>
                <span class="text-xs text-neutral-500">{{ formatTime(n.timestamp) }}</span>
              </div>
              <p class="text-sm text-neutral-600 dark:text-neutral-400 line-clamp-2 mb-2">{{ n.description }}</p>
              
              <div class="flex items-center justify-end gap-2" @click.stop>
                <UBadge v-if="!n.read" color="primary" variant="soft" size="xs">未读</UBadge>
                <UButton
                  v-if="!n.read"
                  variant="ghost"
                  color="neutral"
                  size="xs"
                  icon="i-lucide-check"
                  @click.stop="markAsRead(n.id)"
                >
                  标记已读
                </UButton>
              </div>
            </div>
          </div>
        </NuxtLink>
      </div>
    </UCard>
  </div>
</template>

<script setup lang="ts">
definePageMeta({
  requiresAuth: true,
  title: '通知中心',
  description: '查看实时通知与系统公告',
  layout: 'dashboard',
})

import type { NotificationItem } from '~/composables/useNotifications'

const { notifications, unreadCount, markAsRead, markAllAsRead, clearNotifications } = useNotifications()

const filter = ref<'all' | 'unread'>('all')

const filteredNotifications = computed(() => {
  if (filter.value === 'unread') return notifications.value.filter(n => !n.read)
  return notifications.value
})

const formatTime = (timestamp: string) => {
  const date = new Date(timestamp)
  const now = new Date()
  const diff = now.getTime() - date.getTime()

  if (diff < 60000) return '刚刚'
  if (diff < 3600000) return `${Math.floor(diff / 60000)}分钟前`
  if (diff < 86400000) return `${Math.floor(diff / 3600000)}小时前`
  if (diff < 604800000) return `${Math.floor(diff / 86400000)}天前`

  return date.toLocaleDateString('zh-CN', { month: 'short', day: 'numeric' })
}

const getNotificationIcon = (type: string) => {
  switch (type) {
    case 'new_follower':
    case 'mutual_follow':
      return 'i-lucide-users'
    case 'moment_liked':
    case 'moment_commented':
    case 'comment_reply':
      return 'i-lucide-image'
    case 'group_invite':
    case 'group_application':
    case 'group_application_response':
      return 'i-lucide-users-round'
    case 'system_announcement':
      return 'i-lucide-megaphone'
    default:
      return 'i-lucide-bell'
  }
}

const getNotificationTo = (notification: NotificationItem) => {
  const data = notification.data as Record<string, unknown> | undefined

  switch (notification.type) {
    case 'new_follower':
    case 'mutual_follow':
      if (data?.user && typeof data.user === 'object' && 'user_id' in data.user) {
        return `/users/${(data.user as { user_id: number }).user_id}`
      }
      return undefined
    case 'moment_liked':
    case 'moment_commented':
    case 'comment_reply':
      if (data?.moment_id) return `/moments/${String(data.moment_id)}`
      return undefined
    case 'group_invite':
    case 'group_application':
    case 'group_application_response':
      if (data?.group_chat_id) return `/groups/${String(data.group_chat_id)}`
      return undefined
    default:
      return undefined
  }
}

const handleOpen = (notification: NotificationItem) => {
  if (!notification.read) markAsRead(notification.id)
}
</script>
