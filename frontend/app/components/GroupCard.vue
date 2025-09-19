<template>
    <UCard 
        class="group bg-white/60 dark:bg-slate-800/60 border border-slate-200/50 dark:border-slate-700/50 cursor-pointer hover:bg-white/80 dark:hover:bg-slate-800/80 card-hover fade-in-up"
        @click="$emit('click', group)"
    >
        <div class="space-y-4">
            <!-- 群组头部信息 -->
            <div class="flex items-start gap-4">
                <!-- 群组头像 -->
                <div class="relative">
                    <UAvatar 
                        :alt="group.name" 
                        :src="group.avatar_url"
                        size="lg"
                    >
                        {{ group.name?.charAt(0) }}
                    </UAvatar>
                    <!-- 密码保护指示器 -->
                    <div 
                        v-if="group.is_password_required" 
                        class="absolute -bottom-1 -right-1 w-4 h-4 bg-amber-500 rounded-full border-2 border-white dark:border-slate-800 flex items-center justify-center"
                    >
                        <UIcon name="i-heroicons-lock-closed" class="w-2 h-2 text-white" />
                    </div>
                </div>

                <!-- 群组信息 -->
                <div class="flex-1 min-w-0">
                    <div class="flex items-center gap-2">
                        <h3 class="font-semibold text-slate-900 dark:text-white truncate">
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
                        <UBadge 
                            v-if="group.user_role === 'owner'" 
                            variant="soft" 
                            size="xs" 
                            color="success"
                        >
                            群主
                        </UBadge>
                        <UBadge 
                            v-else-if="group.user_role === 'admin'" 
                            variant="soft" 
                            size="xs" 
                            color="warning"
                        >
                            管理员
                        </UBadge>
                    </div>
                    
                    <div class="flex items-center gap-4 text-sm text-slate-600 dark:text-slate-400 mt-1">
                        <!-- 成员数量 -->
                        <div class="flex items-center gap-1">
                            <UIcon name="i-heroicons-users" class="w-4 h-4" />
                            <span>{{ group.member_count }} 人</span>
                        </div>
                        
                        <!-- 位置信息 -->
                        <div v-if="group.location_name" class="flex items-center gap-1">
                            <UIcon name="i-heroicons-map-pin" class="w-4 h-4" />
                            <span class="truncate">{{ group.location_name }}</span>
                        </div>
                        
                        <!-- 距离信息 -->
                        <div v-if="group.distance !== undefined" class="flex items-center gap-1">
                            <UIcon name="i-heroicons-signal" class="w-4 h-4" />
                            <span>{{ formatDistance(group.distance) }}</span>
                        </div>
                    </div>

                    <!-- 群组描述 -->
                    <p v-if="group.description" class="text-sm text-slate-600 dark:text-slate-400 mt-2 line-clamp-2">
                        {{ group.description }}
                    </p>
                </div>

                <!-- 操作按钮 -->
                <div class="flex items-center gap-2">
                    <UButton
                        v-if="!group.is_member"
                        variant="soft"
                        size="sm"
                        color="primary"
                        @click.stop="$emit('join', group)"
                    >
                        <UIcon name="i-heroicons-plus" class="w-4 h-4 mr-1" />
                        加入
                    </UButton>
                    <UButton
                        v-else
                        variant="ghost"
                        size="sm"
                        icon="i-heroicons-chat-bubble-left-right"
                        class="opacity-0 group-hover:opacity-100 transition-opacity"
                        @click.stop="$emit('enter', group)"
                    >
                        进入
                    </UButton>
                    <UButton
                        variant="ghost"
                        size="sm"
                        icon="i-heroicons-ellipsis-horizontal"
                        class="opacity-0 group-hover:opacity-100 transition-opacity"
                        @click.stop="$emit('more', group)"
                    />
                </div>
            </div>

            <!-- 群组活动信息 -->
            <div class="flex items-center justify-between pt-3 border-t border-slate-200/50 dark:border-slate-700/50">
                <div class="flex items-center gap-1 text-sm text-slate-500 dark:text-slate-400">
                    <UIcon name="i-heroicons-clock" class="w-4 h-4" />
                    <span>最近活跃: {{ formatTime(group.last_active_at) }}</span>
                </div>
                
                <div class="flex items-center gap-1 text-sm text-slate-500 dark:text-slate-400">
                    <UIcon name="i-heroicons-calendar" class="w-4 h-4" />
                    <span>创建于 {{ formatTime(group.created_at) }}</span>
                </div>
            </div>
        </div>
    </UCard>
</template>

<script setup lang="ts">
import type { Group } from '~/types'

interface Props {
    group: Group
}

interface Emits {
    click: [group: Group]
    join: [group: Group]
    enter: [group: Group]
    more: [group: Group]
}

defineProps<Props>()
defineEmits<Emits>()

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
</script>
