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
                    'bg-primary-50 dark:bg-primary-950/20': !notification.is_read,
                    'opacity-75': notification.is_read
                }"
                @click="handleNotificationClick(notification)"
            >
                <!-- 通知图标 -->
                <div class="flex-shrink-0 mt-0.5">
                    <div
                        class="w-8 h-8 rounded-full flex items-center justify-center"
                        :class="getNotificationIconClass(notification.category)"
                    >
                        <UIcon :name="getNotificationIcon(notification.category)" class="w-4 h-4" />
                    </div>
                </div>

                <!-- 通知内容 -->
                <div class="flex-1 min-w-0">
                    <div class="flex items-center gap-2 mb-1">
                        <h4 class="font-medium text-sm truncate">{{ notification.title || '通知' }}</h4>
                        <div v-if="!notification.is_read" class="w-2 h-2 bg-primary-500 rounded-full flex-shrink-0"></div>
                    </div>
                    <p class="text-xs text-neutral-600 dark:text-neutral-400 line-clamp-2 mb-1">
                        {{ notification.content }}
                    </p>
                    <div class="flex items-center justify-between">
                        <span class="text-xs text-neutral-500">
                            {{ formatTime(notification.created_at) }}
                        </span>
                        <UChip
                            :text="getNotificationCategoryText(notification.category)"
                            :color="getNotificationCategoryColor(notification.category)"
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
import { useInboxService } from '~/services/inbox.service'
import type { InboxItemData } from '~/types/api'

const { formatTime } = useFormatters()
const inboxService = useInboxService()
const toast = useToast()

// 通知数据
const notifications = ref<InboxItemData[]>([])
const loading = ref(false)

// 未读通知数量
const unreadCount = computed(() => 
    notifications.value.filter((n: InboxItemData) => !n.is_read).length
)

// 加载通知列表
const loadNotifications = async () => {
    loading.value = true
    try {
        const response = await inboxService.getList({
            limit: 20
        })
        notifications.value = response.items
    } catch (error) {
        console.error('加载通知失败:', error)
    } finally {
        loading.value = false
    }
}

// 组件挂载时加载通知
onMounted(() => {
    loadNotifications()
})

// 获取通知图标
const getNotificationIcon = (category: string) => {
    switch (category) {
        case 'social':
            return 'i-heroicons-user-group'
        case 'system':
            return 'i-heroicons-exclamation-triangle'
        case 'content':
            return 'i-heroicons-heart'
        default:
            return 'i-heroicons-bell'
    }
}

// 获取通知图标样式类
const getNotificationIconClass = (category: string) => {
    switch (category) {
        case 'social':
            return 'bg-blue-100 dark:bg-blue-900/30 text-blue-600 dark:text-blue-400'
        case 'system':
            return 'bg-orange-100 dark:bg-orange-900/30 text-orange-600 dark:text-orange-400'
        case 'content':
            return 'bg-pink-100 dark:bg-pink-900/30 text-pink-600 dark:text-pink-400'
        default:
            return 'bg-neutral-100 dark:bg-neutral-800 text-neutral-600 dark:text-neutral-400'
    }
}

// 获取通知分类文本
const getNotificationCategoryText = (category: string) => {
    switch (category) {
        case 'social':
            return '社交'
        case 'system':
            return '系统'
        case 'content':
            return '内容'
        default:
            return '通知'
    }
}

// 获取通知分类颜色
const getNotificationCategoryColor = (category: string) => {
    switch (category) {
        case 'social':
            return 'primary'
        case 'system':
            return 'warning'
        case 'content':
            return 'error'
        default:
            return 'neutral'
    }
}

// formatTime 函数已从统一工具函数导入

// 处理通知点击
const handleNotificationClick = async (notification: InboxItemData) => {
    // 标记为已读
    if (!notification.is_read) {
        try {
            await inboxService.markAsRead({ ids: [notification.id] })
            notification.is_read = true
        } catch (error) {
            console.error('标记已读失败:', error)
        }
    }

    // 根据 action_url 进行导航
    if (notification.action_url) {
        navigateTo(notification.action_url)
    }
}

// 全部标记为已读
const markAllAsRead = async () => {
    try {
        await inboxService.markAsRead({})
        notifications.value.forEach((n: InboxItemData) => n.is_read = true)
        toast.add({
            title: '已标记为已读',
            description: '所有通知已标记为已读',
            color: 'success'
        })
    } catch (error: any) {
        toast.add({
            title: '操作失败',
            description: error?.message || '标记已读失败',
            color: 'error'
        })
    }
}

// 打开设置
const openSettings = () => {
    navigateTo('/settings')
}

// 查看全部通知
const viewAllNotifications = async () => {
    await navigateTo('/notifications')
}

// 暴露未读数量给父组件
defineExpose({
    unreadCount
})
</script>
