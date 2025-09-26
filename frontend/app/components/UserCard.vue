<template>
    <UCard 
        class="group cursor-pointer hover:shadow-md transition-all duration-200 card-hover fade-in-up"
        @click="$emit('click', user)"
    >
        <div class="flex items-center justify-between">
            <!-- 使用 Nuxt UI User 组件 -->
            <UUser
                :name="user.nickname"
                :description="userDescription"
                :avatar="{
                    src: user.avatar_url,
                    alt: user.nickname,
                    size: 'lg',
                    chip: isOnline ? { color: 'success', position: 'bottom-right' } : false
                }"
                :chip="(user as any).is_temp ? { text: '临时', color: 'neutral' } : false"
                size="lg"
                class="flex-1 min-w-0"
            >
                <!-- 自定义描述内容 -->
                <template #description>
                    <div class="space-y-1">
                        <!-- 位置和距离信息 -->
                        <div class="flex items-center gap-4 text-sm text-muted">
                            <div v-if="user.location_name" class="flex items-center gap-1">
                                <UIcon name="i-heroicons-map-pin" class="w-3 h-3" />
                                <span class="truncate">{{ user.location_name }}</span>
                            </div>
                            
                            <div v-if="user.distance !== undefined" class="flex items-center gap-1">
                                <UIcon name="i-heroicons-signal" class="w-3 h-3" />
                                <span>{{ formatDistance(user.distance) }}</span>
                            </div>
                        </div>

                        <!-- 个人简介 -->
                        <p v-if="user.bio" class="text-sm text-muted line-clamp-2">
                            {{ user.bio }}
                        </p>
                    </div>
                </template>
            </UUser>

            <!-- 操作按钮 -->
            <div class="flex items-center gap-1 opacity-0 group-hover:opacity-100 transition-opacity">
                <UTooltip text="发送消息">
                    <UButton
                        variant="ghost"
                        size="sm"
                        icon="i-heroicons-chat-bubble-left"
                        @click.stop="$emit('message', user)"
                    />
                </UTooltip>
                
                <UDropdownMenu :items="userMenuItems" @click.stop>
                    <UButton
                        variant="ghost"
                        size="sm"
                        icon="i-heroicons-ellipsis-horizontal"
                    />
                </UDropdownMenu>
            </div>
        </div>
    </UCard>
</template>

<script setup lang="ts">
import type { UserResponse } from '~/types/api'

interface Props {
    user: UserResponse
}

interface Emits {
    click: [user: UserResponse]
    message: [user: UserResponse]
    more: [user: UserResponse]
}

const props = defineProps<Props>()
const emit = defineEmits<Emits>()

// 用户描述（基础信息）
const userDescription = computed(() => {
    const parts = []
    if (props.user.location_name) {
        parts.push(props.user.location_name)
    }
    if (props.user.distance !== undefined) {
        parts.push(formatDistance(props.user.distance))
    }
    return parts.join(' • ')
})

// 在线状态（示例，实际需要从WebSocket或其他状态管理获取）
const isOnline = computed(() => {
    // 这里可以从实时状态管理中获取用户在线状态
    return Math.random() > 0.5 // 临时随机状态
})

// 用户操作菜单
const userMenuItems = computed(() => [
    [{
        label: '查看资料',
        icon: 'i-heroicons-user',
        click: () => emit('click', props.user)
    }],
    [{
        label: '添加好友',
        icon: 'i-heroicons-user-plus',
        click: () => handleAddFriend()
    }, {
        label: '关注用户',
        icon: 'i-heroicons-heart',
        click: () => handleFollowUser()
    }],
    [{
        label: '举报用户',
        icon: 'i-heroicons-flag',
        click: () => handleReportUser(),
        class: 'text-error-500'
    }]
])

// 格式化距离
const formatDistance = (distance: number): string => {
    if (distance < 1000) {
        return `${Math.round(distance)}m`
    } else {
        return `${(distance / 1000).toFixed(1)}km`
    }
}

// 添加好友
const handleAddFriend = () => {
    // TODO: 实现添加好友功能
    console.log('添加好友:', props.user.user_id)
}

// 关注用户
const handleFollowUser = () => {
    // TODO: 实现关注用户功能
    console.log('关注用户:', props.user.user_id)
}

// 举报用户
const handleReportUser = () => {
    // TODO: 实现举报用户功能
    console.log('举报用户:', props.user.user_id)
}
</script>
