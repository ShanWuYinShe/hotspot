<template>
    <PageLayout
        title="通知中心"
        description="查看和管理您的所有通知消息"
        :stats="notificationStats"
        :header-actions="headerActions"
    >
        <!-- 通知筛选 -->
        <div class="mb-6">
            <UTabs v-model="currentFilter" :items="filterTabs" />
        </div>
        
        <!-- 通知列表 -->
        <UCard class="bg-white/60 dark:bg-slate-800/60 border border-slate-200/50 dark:border-slate-700/50">
            <DataView
                :data="filteredNotifications"
                :loading="loading"
                :searchable="true"
                :search-fields="['title', 'content']"
                :show-pagination="true"
                :page-size="15"
                :toolbar-actions="toolbarActions"
                item-component="NotificationItem"
                :get-item-key="(item) => item.notification_id"
                :get-item-props="getNotificationProps"
                :get-item-events="getNotificationEvents"
                empty-icon="i-heroicons-bell-slash"
                empty-title="暂无通知"
                empty-description="您目前没有任何通知消息"
                @item-click="handleNotificationClick"
                @search="handleSearch"
            />
        </UCard>
        
        <!-- 批量操作面板 -->
        <ActionPanel
            v-if="selectedNotifications.length > 0"
            title="批量操作"
            :description="`已选择 ${selectedNotifications.length} 条通知`"
            icon="i-heroicons-check-circle"
            :actions="batchActions"
        />
    </PageLayout>
</template>

<script setup lang="ts">
import type { StatItem } from '~/components/data/StatsGrid.vue'
import type { ActionItem } from '~/components/common/ActionPanel.vue'

// 需要登录 - 通知中心需要认证
definePageMeta({
    title: '通知中心',
    description: '查看和管理您的所有通知消息',
    requiresAuth: true,
    layout: 'dashboard'
})

// 通知类型定义
interface NotificationItem {
    notification_id: string
    type: 'system' | 'friend' | 'group' | 'moment' | 'message'
    title: string
    content: string
    is_read: boolean
    created_at: string
    data?: any
    avatar?: string
    sender_name?: string
}

// 状态管理
const loading = ref(false)
const currentFilter = ref('all')
const selectedNotifications = ref<NotificationItem[]>([])
const notifications = ref<NotificationItem[]>([])

// 服务和工具
const toast = useToast()

// 筛选标签
const filterTabs = [
    { value: 'all', label: '全部', icon: 'i-heroicons-bell' },
    { value: 'unread', label: '未读', icon: 'i-heroicons-bell-alert' },
    { value: 'system', label: '系统', icon: 'i-heroicons-cog-6-tooth' },
    { value: 'friend', label: '好友', icon: 'i-heroicons-users' },
    { value: 'group', label: '群组', icon: 'i-heroicons-user-group' },
    { value: 'moment', label: '动态', icon: 'i-heroicons-photo' }
]

// 计算属性
const totalNotifications = computed(() => notifications.value.length)
const unreadCount = computed(() => notifications.value.filter(n => !n.is_read).length)
const todayCount = computed(() => {
    const today = new Date()
    today.setHours(0, 0, 0, 0)
    return notifications.value.filter(n => new Date(n.created_at) >= today).length
})

const filteredNotifications = computed(() => {
    let filtered = notifications.value
    
    switch (currentFilter.value) {
        case 'unread':
            filtered = filtered.filter(n => !n.is_read)
            break
        case 'system':
        case 'friend':
        case 'group':
        case 'moment':
            filtered = filtered.filter(n => n.type === currentFilter.value)
            break
    }
    
    return filtered
})

// 统计数据
const notificationStats = computed<StatItem[]>(() => [
    {
        key: 'total',
        label: '总通知数',
        value: totalNotifications.value,
        icon: 'i-heroicons-bell',
        color: 'blue',
        description: '所有通知消息',
        onClick: () => currentFilter.value = 'all'
    },
    {
        key: 'unread',
        label: '未读通知',
        value: unreadCount.value,
        icon: 'i-heroicons-bell-alert',
        color: 'orange',
        description: '需要您关注的通知',
        onClick: () => currentFilter.value = 'unread'
    },
    {
        key: 'today',
        label: '今日通知',
        value: todayCount.value,
        icon: 'i-heroicons-calendar-days',
        color: 'green',
        description: '今天收到的通知',
        onClick: () => toast.add({ color: 'info', title: '显示今日通知' })
    },
    {
        key: 'system',
        label: '系统通知',
        value: notifications.value.filter(n => n.type === 'system').length,
        icon: 'i-heroicons-cog-6-tooth',
        color: 'purple',
        description: '系统相关通知',
        onClick: () => currentFilter.value = 'system'
    }
])

// 头部操作
const headerActions = computed(() => [
    {
        key: 'mark-all-read',
        label: '全部已读',
        icon: 'i-heroicons-check',
        variant: 'outline' as const,
        action: markAllAsRead
    },
    {
        key: 'settings',
        label: '通知设置',
        icon: 'i-heroicons-cog-6-tooth',
        variant: 'outline' as const,
        action: () => navigateTo('/settings/notifications')
    }
])

// 工具栏操作
const toolbarActions = computed(() => [
    {
        key: 'refresh',
        label: '刷新',
        icon: 'i-heroicons-arrow-path',
        variant: 'outline' as const,
        action: loadNotifications
    },
    {
        key: 'clear-read',
        label: '清除已读',
        icon: 'i-heroicons-trash',
        variant: 'outline' as const,
        action: clearReadNotifications
    }
])

// 批量操作
const batchActions = computed<ActionItem[]>(() => [
    {
        key: 'mark-read',
        label: '标记已读',
        icon: 'i-heroicons-check',
        color: 'primary',
        action: markSelectedAsRead
    },
    {
        key: 'delete',
        label: '删除',
        icon: 'i-heroicons-trash',
        color: 'error',
        action: deleteSelectedNotifications
    }
])

// 通知项属性
const getNotificationProps = (notification: NotificationItem) => ({
    notification,
    selected: selectedNotifications.value.some(n => n.notification_id === notification.notification_id)
})

// 通知项事件
const getNotificationEvents = (notification: NotificationItem) => ({
    select: () => toggleNotificationSelection(notification),
    'mark-read': () => markAsRead(notification),
    delete: () => deleteNotification(notification)
})

// 方法
const loadNotifications = async () => {
    loading.value = true
    try {
        // 模拟加载通知数据
        await new Promise(resolve => setTimeout(resolve, 1000))
        
        notifications.value = [
            {
                notification_id: '1',
                type: 'friend',
                title: '新的好友请求',
                content: '张三想要添加您为好友',
                is_read: false,
                created_at: new Date().toISOString(),
                avatar: 'https://api.dicebear.com/7.x/avataaars/svg?seed=zhang',
                sender_name: '张三'
            },
            {
                notification_id: '2',
                type: 'group',
                title: '群组邀请',
                content: '您被邀请加入"技术交流群"',
                is_read: false,
                created_at: new Date(Date.now() - 3600000).toISOString(),
                avatar: 'https://api.dicebear.com/7.x/shapes/svg?seed=tech',
                sender_name: '技术交流群'
            },
            {
                notification_id: '3',
                type: 'moment',
                title: '动态点赞',
                content: '李四点赞了您的动态',
                is_read: true,
                created_at: new Date(Date.now() - 7200000).toISOString(),
                avatar: 'https://api.dicebear.com/7.x/avataaars/svg?seed=li',
                sender_name: '李四'
            },
            {
                notification_id: '4',
                type: 'system',
                title: '系统维护通知',
                content: '系统将于今晚23:00-24:00进行维护',
                is_read: true,
                created_at: new Date(Date.now() - 86400000).toISOString(),
                avatar: 'https://api.dicebear.com/7.x/shapes/svg?seed=system'
            }
        ]
        
        toast.add({ color: 'success', title: '通知加载完成' })
    } catch (error) {
        toast.add({ color: 'error', title: '加载通知失败' })
    } finally {
        loading.value = false
    }
}

const handleNotificationClick = (notification: NotificationItem) => {
    // 标记为已读
    if (!notification.is_read) {
        markAsRead(notification)
    }
    
    // 根据通知类型跳转
    switch (notification.type) {
        case 'friend':
            navigateTo('/friends/requests')
            break
        case 'group':
            navigateTo('/groups')
            break
        case 'moment':
            navigateTo('/moments')
            break
        case 'message':
            navigateTo('/chat')
            break
        default:
            toast.add({ color: 'info', title: '查看通知详情' })
    }
}

const markAsRead = async (notification: NotificationItem) => {
    try {
        notification.is_read = true
        toast.add({ color: 'success', title: '已标记为已读' })
    } catch (error) {
        toast.add({ color: 'error', title: '操作失败' })
    }
}

const markAllAsRead = async () => {
    try {
        notifications.value.forEach(n => n.is_read = true)
        toast.add({ color: 'success', title: '所有通知已标记为已读' })
    } catch (error) {
        toast.add({ color: 'error', title: '操作失败' })
    }
}

const markSelectedAsRead = async () => {
    try {
        selectedNotifications.value.forEach(n => n.is_read = true)
        selectedNotifications.value = []
        toast.add({ color: 'success', title: '选中通知已标记为已读' })
    } catch (error) {
        toast.add({ color: 'error', title: '操作失败' })
    }
}

const deleteNotification = async (notification: NotificationItem) => {
    const confirmed = confirm('确定要删除这条通知吗？')
    if (!confirmed) return
    
    try {
        const index = notifications.value.findIndex(n => n.notification_id === notification.notification_id)
        if (index > -1) {
            notifications.value.splice(index, 1)
        }
        toast.add({ color: 'success', title: '通知已删除' })
    } catch (error) {
        toast.add({ color: 'error', title: '删除失败' })
    }
}

const deleteSelectedNotifications = async () => {
    const confirmed = confirm(`确定要删除选中的 ${selectedNotifications.value.length} 条通知吗？`)
    if (!confirmed) return
    
    try {
        selectedNotifications.value.forEach(selected => {
            const index = notifications.value.findIndex(n => n.notification_id === selected.notification_id)
            if (index > -1) {
                notifications.value.splice(index, 1)
            }
        })
        selectedNotifications.value = []
        toast.add({ color: 'success', title: '选中通知已删除' })
    } catch (error) {
        toast.add({ color: 'error', title: '删除失败' })
    }
}

const clearReadNotifications = async () => {
    const readCount = notifications.value.filter(n => n.is_read).length
    if (readCount === 0) {
        toast.add({ color: 'info', title: '没有已读通知需要清除' })
        return
    }
    
    const confirmed = confirm(`确定要清除 ${readCount} 条已读通知吗？`)
    if (!confirmed) return
    
    try {
        notifications.value = notifications.value.filter(n => !n.is_read)
        toast.add({ color: 'success', title: '已读通知已清除' })
    } catch (error) {
        toast.add({ color: 'error', title: '清除失败' })
    }
}

const toggleNotificationSelection = (notification: NotificationItem) => {
    const index = selectedNotifications.value.findIndex(n => n.notification_id === notification.notification_id)
    if (index > -1) {
        selectedNotifications.value.splice(index, 1)
    } else {
        selectedNotifications.value.push(notification)
    }
}

const handleSearch = (query: string) => {
    if (query.trim()) {
        toast.add({ color: 'info', title: `搜索: ${query}` })
    }
}

// 页面加载时获取数据
onMounted(() => {
    loadNotifications()
})
</script>
