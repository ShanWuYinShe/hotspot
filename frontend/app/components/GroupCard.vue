<template>
    <UCard 
        class="group cursor-pointer hover:shadow-md transition-all duration-200 card-hover fade-in-up"
        @click="$emit('click', group)"
    >
        <!-- 群组头部 -->
        <template #header>
            <div class="flex items-center justify-between">
                <div class="flex items-center gap-3">
                    <UAvatar 
                        :alt="group.name" 
                        :src="group.avatar_url"
                        size="md"
                        :chip="group.is_password_required ? { 
                            color: 'warning', 
                            position: 'bottom-right',
                            text: '🔒'
                        } : false"
                    >
                        {{ group.name?.charAt(0) }}
                    </UAvatar>
                    
                    <div class="flex-1 min-w-0">
                        <div class="flex items-center gap-2">
                            <h3 class="font-semibold text-highlighted truncate">
                                {{ group.name }}
                            </h3>
                            <UBadge 
                                v-if="group.is_member" 
                                variant="soft" 
                                size="xs" 
                                color="primary"
                            >
                                已加入
                            </UBadge>
                        </div>
                        
                        <div class="flex items-center gap-2 text-xs text-muted mt-1">
                            <UChip 
                                v-if="group.user_role === 'owner'"
                                text="群主"
                                color="success"
                                size="xs"
                            />
                            <UChip 
                                v-else-if="group.user_role === 'admin'"
                                text="管理员"
                                color="warning"
                                size="xs"
                            />
                            
                            <span class="flex items-center gap-1">
                                <UIcon name="i-heroicons-users" class="w-3 h-3" />
                                {{ group.member_count }} 人
                            </span>
                        </div>
                    </div>
                </div>

                <!-- 操作按钮 -->
                <div class="flex items-center gap-1">
                    <UButton
                        v-if="!group.is_member"
                        variant="soft"
                        size="xs"
                        color="primary"
                        icon="i-heroicons-plus"
                        @click.stop="$emit('join', group)"
                    >
                        加入
                    </UButton>
                    <UButton
                        v-else
                        variant="ghost"
                        size="xs"
                        icon="i-heroicons-chat-bubble-left-right"
                        class="opacity-0 group-hover:opacity-100 transition-opacity"
                        @click.stop="$emit('enter', group)"
                    />
                    
                    <UDropdownMenu :items="groupMenuItems" @click.stop>
                        <UButton
                            variant="ghost"
                            size="xs"
                            icon="i-heroicons-ellipsis-horizontal"
                            class="opacity-0 group-hover:opacity-100 transition-opacity"
                        />
                    </UDropdownMenu>
                </div>
            </div>
        </template>

        <!-- 群组内容 -->
        <div class="space-y-3">
            <!-- 群组描述 -->
            <p v-if="group.description" class="text-sm text-muted line-clamp-2">
                {{ group.description }}
            </p>

            <!-- 位置和距离信息 -->
            <div class="flex items-center gap-4 text-sm text-muted">
                <div v-if="group.location_name" class="flex items-center gap-1">
                    <UIcon name="i-heroicons-map-pin" class="w-3 h-3" />
                    <span class="truncate">{{ group.location_name }}</span>
                </div>
                
                <div v-if="group.distance !== undefined" class="flex items-center gap-1">
                    <UIcon name="i-heroicons-signal" class="w-3 h-3" />
                    <span>{{ formatDistance(group.distance) }}</span>
                </div>
            </div>
        </div>

        <!-- 群组底部信息 -->
        <template #footer>
            <div class="flex items-center justify-between text-xs text-muted">
                <div class="flex items-center gap-1">
                    <UIcon name="i-heroicons-clock" class="w-3 h-3" />
                    <span>活跃: {{ formatTime(group.last_active_at) }}</span>
                </div>
                
                <div class="flex items-center gap-1">
                    <UIcon name="i-heroicons-calendar" class="w-3 h-3" />
                    <span>{{ formatTime(group.created_at) }}</span>
                </div>
            </div>
        </template>
    </UCard>
</template>

<script setup lang="ts">
import type { GroupResponse } from '~/types/api'

interface Props {
    group: GroupResponse
}

interface Emits {
    click: [group: GroupResponse]
    join: [group: GroupResponse]
    enter: [group: GroupResponse]
    more: [group: GroupResponse]
}

const props = defineProps<Props>()
const emit = defineEmits<Emits>()

// 群组操作菜单
const groupMenuItems = computed(() => {
    const items = []
    
    // 基础操作
    items.push([{
        label: '查看详情',
        icon: 'i-heroicons-information-circle',
        click: () => emit('click', props.group)
    }])
    
    // 成员操作
    if (props.group.is_member) {
        items.push([{
            label: '进入群聊',
            icon: 'i-heroicons-chat-bubble-left-right',
            click: () => emit('enter', props.group)
        }, {
            label: '群组设置',
            icon: 'i-heroicons-cog-6-tooth',
            click: () => handleGroupSettings()
        }])
        
        // 管理员操作
        if (props.group.user_role === 'owner' || props.group.user_role === 'admin') {
            items.push([{
                label: '管理成员',
                icon: 'i-heroicons-users',
                click: () => handleManageMembers()
            }])
        }
        
        // 退出群组
        items.push([{
            label: '退出群组',
            icon: 'i-heroicons-arrow-right-on-rectangle',
            click: () => handleLeaveGroup(),
            class: 'text-error-500'
        }])
    } else {
        // 非成员操作
        items.push([{
            label: '加入群组',
            icon: 'i-heroicons-plus',
            click: () => emit('join', props.group)
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

// 群组设置
const handleGroupSettings = () => {
    // TODO: 实现群组设置功能
    console.log('群组设置:', props.group.group_id)
}

// 管理成员
const handleManageMembers = () => {
    // TODO: 实现管理成员功能
    console.log('管理成员:', props.group.group_id)
}

// 退出群组
const handleLeaveGroup = () => {
    // TODO: 实现退出群组功能
    console.log('退出群组:', props.group.group_id)
}
</script>

