<template>
    <UCard class="w-80 max-h-96 overflow-hidden">
        <template #header>
            <div class="flex items-center justify-between">
                <div class="flex items-center gap-2">
                    <UIcon name="i-heroicons-bell" class="icon-md" />
                    <h3 class="font-semibold text-sm">通知中心</h3>
                    <UChip v-if="unreadCount > 0" size="sm" color="error">
                        {{ unreadCount > 99 ? '99+' : unreadCount }}
                    </UChip>
                </div>
                <div class="flex items-center gap-1">
                    <UButton
                        v-if="notifications.length > 0"
                        variant="ghost"
                        size="xs"
                        @click="handleMarkAllAsRead"
                    >
                        全部已读
                    </UButton>
                    <UButton
                        variant="ghost"
                        size="sm"
                        icon="i-heroicons-cog-6-tooth"
                        @click="openSettings"
                    />
                </div>
            </div>
        </template>

        <div class="py-2">
            <!-- 通知列表或空状态 -->
            <div v-if="notifications.length === 0" class="text-center py-6">
                <UIcon name="i-heroicons-bell-slash" class="icon-xl text-subtle mx-auto mb-3" />
                <p class="text-sm text-secondary-content">
                    暂无新通知
                </p>
            </div>
            <div v-else class="space-y-1 max-h-64 overflow-y-auto">
                <div
                    v-for="notification in notifications"
                    :key="notification.id"
                    class="p-3 rounded-lg cursor-pointer transition-colors"
                    :class="notification.read 
                        ? 'bg-transparent hover-bg-surface' 
                        : 'bg-primary-light-20 hover-bg-primary-base'"
                    @click="handleNotificationClick(notification)"
                >
                    <div class="flex items-start gap-2">
                        <div class="flex-1 min-w-0">
                            <p class="text-sm font-medium truncate" :class="notification.read ? 'text-secondary-content' : 'text-heading'">
                                {{ notification.title }}
                            </p>
                            <p class="text-xs text-secondary-content line-clamp-2 mt-0.5">
                                {{ notification.description }}
                            </p>
                            <p class="text-xs text-muted mt-1">
                                {{ formatTime(notification.timestamp) }}
                            </p>
                        </div>
                        <div v-if="!notification.read" class="status-dot-unread mt-1.5"></div>
                    </div>
                </div>
            </div>
        </div>

        <template #footer>
            <div class="flex gap-2">
                <UButton
                    variant="ghost"
                    size="sm"
                    class="flex-1"
                    @click="openNotifications"
                >
                    查看全部
                </UButton>
                <UButton
                    variant="ghost"
                    size="sm"
                    icon="i-heroicons-cog-6-tooth"
                    @click="openSettings"
                />
            </div>
        </template>
    </UCard>
</template>

<script setup lang="ts">
import type { NotificationItem } from '~/composables/useNotifications'

// 使用全局通知状态
const { notifications, unreadCount, markAsRead, markAllAsRead } = useNotifications()

// 格式化时间
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

// 点击通知
const handleNotificationClick = (notification: NotificationItem) => {
    // 标记为已读
    markAsRead(notification.id)
    
    // 根据类型跳转
    const data = notification.data as Record<string, unknown> | undefined
    switch (notification.type) {
        case 'new_follower':
        case 'mutual_follow':
            if (data?.user && typeof data.user === 'object' && 'user_id' in data.user) {
                navigateTo(`/users/${(data.user as { user_id: number }).user_id}`)
            }
            break
        case 'moment_liked':
        case 'moment_commented':
        case 'comment_reply':
            if (data?.moment_id) {
                navigateTo(`/moments/${data.moment_id}`)
            }
            break
        case 'group_invite':
        case 'group_application':
        case 'group_application_response':
            if (data?.group_chat_id) {
                navigateTo(`/groups/${data.group_chat_id}`)
            }
            break
    }
}

// 标记全部已读
const handleMarkAllAsRead = () => {
    markAllAsRead()
}

// 打开通知页面
const openNotifications = () => {
    navigateTo('/notifications')
}

// 打开设置
const openSettings = () => {
    navigateTo('/settings')
}

// 暴露未读数量给父组件
defineExpose({
    unreadCount
})
</script>
