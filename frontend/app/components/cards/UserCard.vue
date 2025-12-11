<template>
    <UCard
        class="interactive-card group cursor-pointer"
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
                :chip="user.user_type && isTempUser(user.user_type) ? { text: '临时', color: 'neutral' } : false"
                size="lg"
                class="flex-1 min-w-0"
            >
                <!-- 自定义描述内容 -->
                <template #description>
                    <div class="space-y-1">
                        <!-- 位置和距离信息（仅当有 geo 数据时显示） -->
                        <div v-if="user.geo" class="flex items-center gap-4 text-sm text-secondary-content">
                            <div v-if="user.geo.location_name" class="flex items-center gap-1">
                                <UIcon name="i-heroicons-map-pin" class="icon-sm" />
                                <span class="truncate">{{ user.geo.location_name }}</span>
                            </div>
                            
                            <div v-if="user.geo.distance !== undefined" class="flex items-center gap-1">
                                <UIcon name="i-heroicons-signal" class="icon-sm" />
                                <span>{{ formatDistance(user.geo.distance) }}</span>
                            </div>
                        </div>

                        <!-- 个人简介 -->
                        <p v-if="user.bio" class="text-sm text-secondary-content line-clamp-2">
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
    </UCard>
</template>

<script setup lang="ts">
import type { GeoData } from '~/types/api'
import { useUsersService } from '~/services/users.service'
import { isTempUser } from '~/utils/userTypes'
import { getErrorMessage } from '~/utils/error'

// UserCard 支持的用户数据类型（兼容搜索结果和好友列表）
interface UserCardData {
    user_id: number;
    nickname: string;
    avatar_url?: string;
    user_type?: string;  // 可选，搜索结果可能没有
    bio?: string;
    tags?: string[];
    is_online?: boolean;
    geo?: GeoData;  // 搜索结果有 geo，好友列表没有
    relationship?: {
        is_mutual_follow?: boolean;
        is_following?: boolean;
    };
}

interface Props {
    user: UserCardData
}

interface Emits {
    click: [user: UserCardData]
    message: [user: UserCardData]
    more: [user: UserCardData]
}

const props = defineProps<Props>()
const { formatDistance } = useFormatters()
const emit = defineEmits<Emits>()

// 用户描述（基础信息）
const userDescription = computed(() => {
    const parts = []
    if (props.user.geo?.location_name) {
        parts.push(props.user.geo.location_name)
    }
    if (props.user.geo?.distance !== undefined) {
        parts.push(formatDistance(props.user.geo.distance))
    }
    return parts.join(' • ')
})

// 用户在线状态：直接使用 props 传入的值（展示组件不需要实时订阅）
const isOnline = computed(() => props.user.is_online ?? false)

// 用户关系状态（从 relationship 字段获取）
// 好友 = 互相关注
const isFriend = computed(() => props.user.relationship?.is_mutual_follow ?? false)
const isFollowing = computed(() => props.user.relationship?.is_following ?? false)

// 用户操作菜单（根据关系状态动态调整）
const userMenuItems = computed(() => {
    const items = []

    // 基础操作
    items.push([{
        label: '查看资料',
        icon: 'i-heroicons-user',
        click: () => emit('click', props.user)
    }])

    // 关系操作（好友 = 互相关注，只需要关注操作）
    const relationItems = []
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
        class: 'text-error'
    }])

    return items
})

// formatDistance 函数已从统一工具函数导入

// 关注用户（好友 = 互相关注）
const handleFollowUser = async () => {
    const toast = useToast()
    const userService = useUsersService()
    try {
        await userService.followUser({
            id: props.user.user_id
        })
        
        toast.add({
            title: '关注成功',
            description: `已关注 ${props.user.nickname}`,
            color: 'success'
        })
        
        // 刷新用户数据
        await refreshNuxtData('users')
    } catch (error: unknown) {
        toast.add({
            title: '关注失败',
            description: getErrorMessage(error, '关注用户时发生错误'),
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
            id: props.user.user_id
        })
        
        toast.add({
            title: '已取消关注',
            description: `已取消关注 ${props.user.nickname}`,
            color: 'success'
        })
        
        // 刷新用户数据
        await refreshNuxtData('users')
    } catch (error: unknown) {
        toast.add({
            title: '取消关注失败',
            description: getErrorMessage(error, '取消关注时发生错误'),
            color: 'error'
        })
    }
}

// 举报用户
const handleReportUser = async () => {
    const modal = useModal()
    const toast = useToast()
    
    await modal.showReport('user', String(props.user.user_id))
    toast.add({
        title: '举报已提交',
        description: '感谢您的举报，我们会尽快处理',
        color: 'success'
    })
}
</script>
