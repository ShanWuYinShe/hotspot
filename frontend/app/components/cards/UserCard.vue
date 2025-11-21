<template>
    <UCard 
        class="group cursor-pointer card-hover fade-in-up"
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
import { useUserService } from '~/services/users.service'
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

// 获取用户在线状态
const { checkUserOnline } = useOnlineStatus()
const isOnline = ref(false)

// 异步获取在线状态
watchEffect(async () => {
    if (props.user.user_id) {
        isOnline.value = await checkUserOnline(props.user.user_id)
    }
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

// formatDistance 函数已从统一工具函数导入

// 添加好友
const handleAddFriend = async () => {
    try {
        const userService = useUserService()
        await userService.followUser({
            user_id: props.user.user_id
        })
        
        const toast = useToast()
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

// 关注用户
const handleFollowUser = async () => {
    await handleAddFriend() // 关注和添加好友使用相同逻辑
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
