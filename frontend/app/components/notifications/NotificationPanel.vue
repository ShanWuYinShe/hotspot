<template>
    <UCard class="w-96 max-h-96 overflow-hidden">
        <template #header>
            <div class="flex items-center justify-between">
                <div class="flex items-center gap-2">
                    <UIcon name="i-heroicons-bell" class="w-5 h-5" />
                    <h3 class="font-semibold">通知</h3>
                    <UBadge v-if="unreadCount > 0" :text="String(unreadCount)" color="error" size="xs" />
                </div>
                <div class="flex items-center gap-1">
                    <UTooltip text="全部标记为已读">
                        <UButton
                            variant="ghost"
                            size="xs"
                            icon="i-heroicons-check-circle"
                            @click="markAllAsRead"
                            :disabled="unreadCount === 0"
                        />
                    </UTooltip>
                    <UTooltip text="设置">
                        <UButton
                            variant="ghost"
                            size="xs"
                            icon="i-heroicons-cog-6-tooth"
                            @click="openSettings"
                        />
                    </UTooltip>
                </div>
            </div>
        </template>

        <div class="space-y-1 max-h-80 overflow-y-auto">
            <!-- 空状态 -->
            <div v-if="notifications.length === 0" class="text-center py-8">
                <UIcon name="i-heroicons-bell-slash" class="w-12 h-12 text-neutral-400 mx-auto mb-3" />
                <p class="text-neutral-500 text-sm">暂无通知</p>
            </div>

            <!-- 通知列表 -->
            <div
                v-for="notification in notifications"
                :key="notification.id"
                class="flex items-start gap-3 p-3 rounded-lg hover:bg-neutral-50 dark:hover:bg-neutral-800/50 cursor-pointer transition-colors"
                :class="{
                    'bg-primary-50 dark:bg-primary-950/20': !notification.read,
                    'opacity-75': notification.read
                }"
                @click="handleNotificationClick(notification)"
            >
                <!-- 通知图标 -->
                <div class="flex-shrink-0 mt-0.5">
                    <UAvatar
                        v-if="notification.type === 'message' || notification.type === 'friend_request'"
                        :src="notification.avatar"
                        :alt="notification.title"
                        size="sm"
                    >
                        {{ notification.title?.charAt(0) }}
                    </UAvatar>
                    <div
                        v-else
                        class="w-8 h-8 rounded-full flex items-center justify-center"
                        :class="getNotificationIconClass(notification.type)"
                    >
                        <UIcon :name="getNotificationIcon(notification.type)" class="w-4 h-4" />
                    </div>
                </div>

                <!-- 通知内容 -->
                <div class="flex-1 min-w-0">
                    <div class="flex items-center gap-2 mb-1">
                        <h4 class="font-medium text-sm truncate">{{ notification.title }}</h4>
                        <div v-if="!notification.read" class="w-2 h-2 bg-primary-500 rounded-full flex-shrink-0"></div>
                    </div>
                    <p class="text-xs text-neutral-600 dark:text-neutral-400 line-clamp-2 mb-1">
                        {{ notification.message }}
                    </p>
                    <div class="flex items-center justify-between">
                        <span class="text-xs text-neutral-500">
                            {{ formatTime(notification.created_at) }}
                        </span>
                        <UChip
                            :text="getNotificationTypeText(notification.type)"
                            :color="getNotificationTypeColor(notification.type)"
                            size="xs"
                        />
                    </div>
                </div>
            </div>
        </div>

        <template #footer v-if="notifications.length > 0">
            <div class="text-center">
                <UButton
                    variant="ghost"
                    size="sm"
                    @click="viewAllNotifications"
                    class="w-full"
                >
                    查看全部通知
                </UButton>
            </div>
        </template>
    </UCard>
</template>

<script setup lang="ts">

interface Notification {
    id: string
    type: 'message' | 'group' | 'friend_request' | 'system' | 'moment'
    title: string
    message: string
    avatar?: string
    read: boolean
    created_at: string
    data?: any
}

const { formatTime } = useFormatters()

// 模拟通知数据
const notifications = ref<Notification[]>([
    {
        id: '1',
        type: 'message',
        title: '张三',
        message: '你好，最近怎么样？',
        avatar: '',
        read: false,
        created_at: new Date(Date.now() - 5 * 60 * 1000).toISOString(),
        data: { userId: '123' }
    },
    {
        id: '2',
        type: 'group',
        title: '技术交流群',
        message: '李四: 大家有什么好的学习资源推荐吗？',
        read: false,
        created_at: new Date(Date.now() - 15 * 60 * 1000).toISOString(),
        data: { groupId: '456' }
    },
    {
        id: '3',
        type: 'friend_request',
        title: '王五',
        message: '想要添加您为好友',
        avatar: '',
        read: false,
        created_at: new Date(Date.now() - 30 * 60 * 1000).toISOString(),
        data: { userId: '789' }
    },
    {
        id: '4',
        type: 'system',
        title: '系统通知',
        message: '您的账户安全设置已更新',
        read: true,
        created_at: new Date(Date.now() - 2 * 60 * 60 * 1000).toISOString()
    },
    {
        id: '5',
        type: 'moment',
        title: '赵六',
        message: '点赞了您的动态',
        avatar: '',
        read: true,
        created_at: new Date(Date.now() - 4 * 60 * 60 * 1000).toISOString(),
        data: { momentId: '101' }
    }
])

const toast = useToast()

// 未读通知数量
const unreadCount = computed(() => 
    notifications.value.filter(n => !n.read).length
)

// 获取通知图标
const getNotificationIcon = (type: string) => {
    switch (type) {
        case 'message':
            return 'i-heroicons-chat-bubble-left'
        case 'group':
            return 'i-heroicons-user-group'
        case 'friend_request':
            return 'i-heroicons-user-plus'
        case 'system':
            return 'i-heroicons-exclamation-triangle'
        case 'moment':
            return 'i-heroicons-heart'
        default:
            return 'i-heroicons-bell'
    }
}

// 获取通知图标样式类
const getNotificationIconClass = (type: string) => {
    switch (type) {
        case 'message':
            return 'bg-blue-100 dark:bg-blue-900/30 text-blue-600 dark:text-blue-400'
        case 'group':
            return 'bg-green-100 dark:bg-green-900/30 text-green-600 dark:text-green-400'
        case 'friend_request':
            return 'bg-purple-100 dark:bg-purple-900/30 text-purple-600 dark:text-purple-400'
        case 'system':
            return 'bg-orange-100 dark:bg-orange-900/30 text-orange-600 dark:text-orange-400'
        case 'moment':
            return 'bg-pink-100 dark:bg-pink-900/30 text-pink-600 dark:text-pink-400'
        default:
            return 'bg-neutral-100 dark:bg-neutral-800 text-neutral-600 dark:text-neutral-400'
    }
}

// 获取通知类型文本
const getNotificationTypeText = (type: string) => {
    switch (type) {
        case 'message':
            return '私信'
        case 'group':
            return '群组'
        case 'friend_request':
            return '好友'
        case 'system':
            return '系统'
        case 'moment':
            return '动态'
        default:
            return '通知'
    }
}

// 获取通知类型颜色
const getNotificationTypeColor = (type: string) => {
    switch (type) {
        case 'message':
            return 'primary'
        case 'group':
            return 'success'
        case 'friend_request':
            return 'info'
        case 'system':
            return 'warning'
        case 'moment':
            return 'error'
        default:
            return 'neutral'
    }
}

// formatTime 函数已从统一工具函数导入

// 处理通知点击
const handleNotificationClick = (notification: Notification) => {
    // 标记为已读
    if (!notification.read) {
        notification.read = true
    }

    // 根据通知类型进行导航
    switch (notification.type) {
        case 'message':
            if (notification.data?.userId) {
                navigateTo(`/chat/users/${notification.data.userId}`)
            }
            break
        case 'group':
            if (notification.data?.groupId) {
                navigateTo(`/chat/groups/${notification.data.groupId}`)
            }
            break
        case 'friend_request':
            navigateTo('/friends/requests')
            break
        case 'moment':
            if (notification.data?.momentId) {
                navigateTo(`/moments/${notification.data.momentId}`)
            }
            break
        case 'system':
            // 系统通知可能不需要导航
            break
    }
}

// 全部标记为已读
const markAllAsRead = () => {
    notifications.value.forEach(n => n.read = true)
    toast.add({
        title: '已标记为已读',
        description: '所有通知已标记为已读',
        color: 'success'
    })
}

// 打开设置
const openSettings = () => {
    navigateTo('/settings')
}

// 查看全部通知
const viewAllNotifications = () => {
    // TODO: 创建专门的通知页面
    toast.add({
        title: '功能开发中',
        description: '通知页面即将上线',
        color: 'info'
    })
}

// 暴露未读数量给父组件
defineExpose({
    unreadCount
})
</script>
