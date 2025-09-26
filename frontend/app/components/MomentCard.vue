<template>
    <UCard 
        class="group cursor-pointer hover:shadow-md transition-all duration-200"
        @click="$emit('click', moment)"
    >
        <!-- 用户信息头部 -->
        <template #header>
            <div class="flex items-center justify-between">
                <UUser
                    :name="moment.nickname"
                    :description="userDescription"
                    :avatar="{
                        src: moment.user_avatar,
                        alt: moment.nickname,
                        size: 'md',
                        chip: isOnline ? { color: 'success', position: 'bottom-right' } : false
                    }"
                    :chip="isMyMoment ? { text: '我的', color: 'primary' } : false"
                    size="md"
                    class="flex-1 min-w-0"
                >
                    <template #description>
                        <div class="flex items-center gap-3 text-sm text-muted">
                            <div class="flex items-center gap-1">
                                <UIcon name="i-heroicons-clock" class="w-3 h-3" />
                                <span>{{ formatTime(moment.created_at) }}</span>
                            </div>
                            
                            <div v-if="moment.location_name" class="flex items-center gap-1">
                                <UIcon name="i-heroicons-map-pin" class="w-3 h-3" />
                                <span class="truncate">{{ moment.location_name }}</span>
                            </div>
                            
                            <div v-if="moment.distance !== undefined" class="flex items-center gap-1">
                                <UIcon name="i-heroicons-signal" class="w-3 h-3" />
                                <span>{{ formatDistance(moment.distance) }}</span>
                            </div>
                        </div>
                    </template>
                </UUser>

                <!-- 操作菜单 -->
                <UDropdownMenu :items="momentMenuItems" @click.stop>
                    <UButton
                        variant="ghost"
                        size="sm"
                        icon="i-heroicons-ellipsis-horizontal"
                        class="opacity-0 group-hover:opacity-100 transition-opacity"
                    />
                </UDropdownMenu>
            </div>
        </template>

        <!-- 动态内容 -->
        <div class="space-y-4">
            <!-- 文字内容 -->
            <div class="prose prose-sm max-w-none">
                <p class="text-highlighted leading-relaxed">
                    {{ moment.description }}
                </p>
            </div>

            <!-- 标签 -->
            <div v-if="moment.tags?.length" class="flex flex-wrap gap-2">
                <UBadge
                    v-for="tag in moment.tags"
                    :key="tag"
                    variant="soft"
                    size="sm"
                    color="neutral"
                    class="cursor-pointer hover:bg-primary-50 hover:text-primary-600 dark:hover:bg-primary-950 dark:hover:text-primary-400 transition-colors"
                    @click.stop="$emit('tag-click', tag)"
                >
                    #{{ tag }}
                </UBadge>
            </div>

            <!-- 图片功能暂未实现，等待后端API支持 -->
        </div>

        <!-- 互动区域 -->
        <template #footer>
            <div class="flex items-center justify-between">
                <!-- 互动按钮 -->
                <div class="flex items-center gap-1">
                    <UButton
                        variant="ghost"
                        size="sm"
                        @click.stop="$emit('like', moment)"
                        :color="moment.liked_by_me ? 'error' : 'neutral'"
                        class="transition-colors"
                    >
                        <UIcon
                            :name="moment.liked_by_me ? 'i-heroicons-heart-solid' : 'i-heroicons-heart'"
                            class="w-4 h-4 mr-1"
                        />
                        {{ moment.likes_count }}
                    </UButton>

                    <UButton 
                        variant="ghost" 
                        size="sm" 
                        color="neutral"
                        @click.stop="$emit('comment', moment)"
                    >
                        <UIcon name="i-heroicons-chat-bubble-left" class="w-4 h-4 mr-1" />
                        {{ moment.comments_count }}
                    </UButton>

                    <UButton 
                        variant="ghost" 
                        size="sm" 
                        color="neutral"
                        @click.stop="$emit('share', moment)"
                    >
                        <UIcon name="i-heroicons-share" class="w-4 h-4 mr-1" />
                        分享
                    </UButton>
                </div>

                <!-- 可见性功能暂未实现，等待后端API支持 -->
            </div>
        </template>
    </UCard>
</template>

<script setup lang="ts">
import type { MomentResponse, UserResponse } from '~/types/api'

interface Props {
    moment: MomentResponse
}

interface Emits {
    click: [moment: MomentResponse]
    like: [moment: MomentResponse]
    comment: [moment: MomentResponse]
    share: [moment: MomentResponse]
    more: [moment: MomentResponse]
    deleted: [moment: MomentResponse]
    'tag-click': [tag: string]
    'image-click': [data: { image: string; index: number }]
    'more-images': [moment: MomentResponse]
}

const props = defineProps<Props>()
const emit = defineEmits<Emits>()

const authStore = useAuthStore()
const toast = useToast()

// 检查是否是当前用户的动态
const isMyMoment = computed(() => {
    return props.moment.user_id === authStore.user?.user_id
})

// 模拟在线状态（实际应该从实时状态管理获取）
const isOnline = computed(() => {
    // 这里可以从WebSocket或其他实时状态管理中获取用户在线状态
    return Math.random() > 0.5 // 临时随机状态
})

// 用户描述信息
const userDescription = computed(() => {
    const parts = []
    if (props.moment.location_name) {
        parts.push(props.moment.location_name)
    }
    if (props.moment.distance !== undefined) {
        parts.push(formatDistance(props.moment.distance))
    }
    return parts.join(' • ')
})

// 可见性功能暂未实现，等待后端API支持

// 动态操作菜单
const momentMenuItems = computed(() => {
    const items = []

    // 基础操作
    items.push([{
        label: '查看详情',
        icon: 'i-heroicons-eye',
        click: () => emit('click', props.moment)
    }])

    // 我的动态操作
    if (isMyMoment.value) {
        items.push([{
            label: '编辑动态',
            icon: 'i-heroicons-pencil',
            click: () => handleEditMoment()
        }])

        items.push([{
            label: '删除动态',
            icon: 'i-heroicons-trash',
            click: () => handleDeleteMoment(),
            class: 'text-error-500'
        }])
    } else {
        // 他人动态操作
        items.push([{
            label: '关注用户',
            icon: 'i-heroicons-user-plus',
            click: () => handleFollowUser()
        }, {
            label: '发送消息',
            icon: 'i-heroicons-chat-bubble-left',
            click: () => handleSendMessage()
        }])

        items.push([{
            label: '举报动态',
            icon: 'i-heroicons-flag',
            click: () => handleReportMoment(),
            class: 'text-error-500'
        }])
    }

    return items
})

// 格式化距离
const formatDistance = (distance: number): string => {
    if (distance < 1000) {
        return `${Math.round(distance)}m`
    } else {
        return `${(distance / 1000).toFixed(1)}km`
    }
}

// 格式化时间
const formatTime = (time: string): string => {
    const date = new Date(time)
    const now = new Date()
    const diffMs = now.getTime() - date.getTime()
    const diffMins = Math.floor(diffMs / 60000)
    const diffHours = Math.floor(diffMs / 3600000)
    const diffDays = Math.floor(diffMs / 86400000)

    if (diffMins < 1) {
        return '刚刚'
    } else if (diffMins < 60) {
        return `${diffMins}分钟前`
    } else if (diffHours < 24) {
        return `${diffHours}小时前`
    } else if (diffDays < 7) {
        return `${diffDays}天前`
    } else {
        return date.toLocaleDateString('zh-CN', {
            year: 'numeric',
            month: 'short',
            day: 'numeric'
        })
    }
}

// 服务和功能管理
const momentService = useMomentService()
const { executeFeature } = useFeatureStatus()
const friendActions = useFriendActions()

// 编辑动态
const handleEditMoment = async () => {
    await executeFeature('momentEdit', async () => {
        // 实际实现编辑功能
        toast.add({
            title: '编辑功能',
            description: '编辑功能正在开发中',
            color: 'info'
        })
    })
}

// 删除动态
const handleDeleteMoment = async () => {
    const confirmed = confirm('确定要删除这条动态吗？')
    if (!confirmed) return

    await executeFeature('momentDelete', async () => {
        await momentService.deleteMoment(props.moment.moment_id)
        toast.add({
            title: '删除成功',
            description: '动态已删除',
            color: 'success'
        })
        // 触发父组件刷新
        emit('deleted', props.moment)
    })
}

// 关注用户
const handleFollowUser = async () => {
    const user = {
        user_id: props.moment.user_id,
        nickname: props.moment.nickname,
        avatar_url: props.moment.user_avatar
    } as UserResponse
    
    await friendActions.followUser(user)
}

// 发送消息
const handleSendMessage = async () => {
    await executeFeature('privateChat', async () => {
        // 跳转到私聊页面
        await navigateTo(`/chat/users/${props.moment.user_id}`)
    })
}

// 举报动态
const handleReportMoment = async () => {
    await executeFeature('reports', async () => {
        // 实现举报功能
        toast.add({
            title: '举报已提交',
            description: '我们会尽快处理您的举报',
            color: 'success'
        })
    })
}
</script>
