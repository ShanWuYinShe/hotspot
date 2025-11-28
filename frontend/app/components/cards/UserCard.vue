<template>
    <InteractiveCard 
        class="group user-card fade-in-up"
        :hover="true"
        :active="true"
        :glow="true"
        shadow="md"
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
                :chip="isTempUser(user.user_id) ? { text: '临时', color: 'neutral' } : false"
                size="lg"
                class="flex-1 min-w-0"
            >
                <!-- 自定义描述内容 -->
                <template #description>
                    <div class="space-y-1">
                        <!-- 位置和距离信息 -->
                        <div class="flex items-center gap-4 text-sm text-neutral-600 dark:text-neutral-400">
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
                        <p v-if="user.bio" class="text-sm text-neutral-600 dark:text-neutral-400 line-clamp-2">
                            {{ user.bio }}
                        </p>
                    </div>
                </template>
            </UUser>

            <!-- 操作按钮 -->
            <div class="flex items-center gap-2">
                <!-- 好友状态标记（始终可见） -->
                <UBadge
                    v-if="isFriend"
                    variant="soft"
                    color="success"
                    size="sm"
                >
                    好友
                </UBadge>
                <UBadge
                    v-else-if="isFollowing"
                    variant="soft"
                    color="primary"
                    size="sm"
                >
                    已关注
                </UBadge>

                <!-- 操作按钮（hover时显示） -->
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
        </div>
    </InteractiveCard>
</template>

<script setup lang="ts">
import type { UserResponse } from '~/types/api'
import { useUsersService } from '~/services/users.service'
import { isTempUser } from '~/utils/userTypes'

interface Props {
    user: UserResponse
}

interface Emits {
    click: [user: UserResponse]
    message: [user: UserResponse]
    more: [user: UserResponse]
}

const props = defineProps<Props>()
const { formatDistance } = useFormatters()
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

// 用户在线状态（从 props 获取或默认为 false）
// TODO: 通过 WebSocket 订阅用户在线状态更新
const isOnline = computed(() => props.user.is_online ?? false)

// 用户关系状态（从 props 获取，如果有 relationship 字段）
const isFriend = computed(() => (props.user as any).is_friend ?? false)
const isFollowing = computed(() => (props.user as any).is_following ?? false)

// 用户操作菜单（根据关系状态动态调整）
const userMenuItems = computed(() => {
    const items = []

    // 基础操作
    items.push([{
        label: '查看资料',
        icon: 'i-heroicons-user',
        click: () => emit('click', props.user)
    }])

    // 关系操作
    const relationItems = []
    if (!isFriend.value) {
        relationItems.push({
            label: '添加好友',
            icon: 'i-heroicons-user-plus',
            click: () => handleAddFriend()
        })
    }
    if (!isFollowing.value) {
        relationItems.push({
            label: '关注用户',
            icon: 'i-heroicons-heart',
            click: () => handleFollowUser()
        })
    } else {
        relationItems.push({
            label: '取消关注',
            icon: 'i-heroicons-heart-slash',
            click: () => handleUnfollowUser()
        })
    }
    if (relationItems.length > 0) {
        items.push(relationItems)
    }

    // 举报操作
    items.push([{
        label: '举报用户',
        icon: 'i-heroicons-flag',
        click: () => handleReportUser(),
        class: 'text-error-500'
    }])

    return items
})

// formatDistance 函数已从统一工具函数导入

// 添加好友
const handleAddFriend = async () => {
    const toast = useToast()
    const userService = useUsersService()
    try {
        await userService.sendFriendRequest({
            to_user_id: props.user.user_id
        })
        
        toast.add({
            title: '好友申请已发送',
            description: `已向 ${props.user.nickname} 发送好友申请`,
            color: 'success'
        })
        
        // 刷新用户数据
        await refreshNuxtData('users')
    } catch (error: any) {
        toast.add({
            title: '发送失败',
            description: error.message || '发送好友申请时发生错误',
            color: 'error'
        })
    }
}

// 关注用户
const handleFollowUser = async () => {
    const toast = useToast()
    const userService = useUsersService()
    try {
        await userService.followUser({
            to_user_id: props.user.user_id
        })
        
        toast.add({
            title: '关注成功',
            description: `已关注 ${props.user.nickname}`,
            color: 'success'
        })
        
        // 刷新用户数据
        await refreshNuxtData('users')
    } catch (error: any) {
        toast.add({
            title: '关注失败',
            description: error.message || '关注用户时发生错误',
            color: 'error'
        })
    }
}

// 取消关注用户
const handleUnfollowUser = async () => {
    const toast = useToast()
    const userService = useUsersService()
    try {
        await userService.unfollowUser({
            to_user_id: props.user.user_id
        })
        
        toast.add({
            title: '已取消关注',
            description: `已取消关注 ${props.user.nickname}`,
            color: 'success'
        })
        
        // 刷新用户数据
        await refreshNuxtData('users')
    } catch (error: any) {
        toast.add({
            title: '取消关注失败',
            description: error.message || '取消关注时发生错误',
            color: 'error'
        })
    }
}

// 举报用户
const handleReportUser = async () => {
    const { showReport } = useModal()
    
    await showReport({
        targetType: 'user',
        targetId: props.user.user_id,
        onSuccess: () => {
            const toast = useToast()
            toast.add({
                title: '举报已提交',
                description: '感谢您的举报，我们会尽快处理',
                color: 'success'
            })
        }
    })
}
</script>
