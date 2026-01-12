<template>
    <UCard class="w-80 max-h-96 overflow-hidden">
        <template #header>
            <div class="flex items-center justify-between">
                <div class="flex items-center gap-2">
                    <UIcon name="i-lucide-bell" :size="20" />
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
                        icon="i-lucide-settings"
                        @click="openSettings"
                    />
                </div>
            </div>
        </template>

        <div class="-m-5 sm:-m-7">
            <UEmpty
                v-if="notifications.length === 0"
                icon="i-lucide-bell-off"
                title="暂无新通知"
                description="新的通知会在这里出现"
                class="py-6"
            />

            <div v-else class="max-h-64 overflow-y-auto divide-y divide-neutral-100 dark:divide-neutral-800">
                <div
                    v-for="n in notifications"
                    :key="n.id"
                    class="p-3 hover:bg-neutral-50 dark:hover:bg-neutral-800/50 cursor-pointer transition-colors"
                    @click="handleNotificationClick(n)"
                >
                    <div class="flex items-start gap-3">
                        <div class="bg-neutral-100 dark:bg-neutral-800 p-1.5 rounded-full shrink-0">
                           <UIcon :name="getNotificationIcon(n.type)" class="size-4 text-neutral-600 dark:text-neutral-400" />
                        </div>
                        <div class="flex-1 min-w-0">
                            <h4 class="text-sm font-medium text-neutral-900 dark:text-white truncate">{{ n.title }}</h4>
                            <p class="text-xs text-neutral-500 dark:text-neutral-400 line-clamp-2 mt-0.5">{{ n.description }}</p>
                            <div class="flex items-center justify-between gap-2 mt-2">
                                <UBadge color="neutral" variant="soft" size="xs">{{ formatTime(n.timestamp) }}</UBadge>
                                <UBadge v-if="!n.read" color="primary" variant="soft" size="xs">未读</UBadge>
                            </div>
                        </div>
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
                    icon="i-lucide-settings"
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
const { requireAuth } = useRequireAuth()

// 使用统一的格式化工具
const { formatTime: formatTimeUtil } = useFormatters()

// 格式化时间（相对时间）
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
    if (!requireAuth('查看通知')) return
    navigateTo('/notifications')
}

// 打开设置
const openSettings = () => {
    if (!requireAuth('进入隐私设置')) return
    navigateTo('/settings')
}

// 暴露未读数量给父组件
defineExpose({
    unreadCount
})
</script>
