<template>
    <UCard 
        class="group bg-white/60 dark:bg-slate-800/60 border border-slate-200/50 dark:border-slate-700/50 cursor-pointer hover:bg-white/80 dark:hover:bg-slate-800/80 card-hover fade-in-up"
        @click="$emit('click', user)"
    >
        <div class="flex items-center gap-4">
            <!-- 用户头像 -->
            <div class="relative">
                <UAvatar 
                    :alt="user.nickname" 
                    :src="user.avatar_url"
                    size="lg"
                >
                    {{ user.nickname?.charAt(0) }}
                </UAvatar>
                <!-- 在线状态指示器 -->
                <div class="absolute -bottom-1 -right-1 w-4 h-4 bg-green-500 rounded-full border-2 border-white dark:border-slate-800"></div>
            </div>

            <!-- 用户信息 -->
            <div class="flex-1 min-w-0">
                <div class="flex items-center gap-2">
                    <h3 class="font-semibold text-slate-900 dark:text-white truncate">
                        {{ user.nickname }}
                    </h3>
                    <UBadge 
                        v-if="user.is_temp" 
                        variant="soft" 
                        size="xs" 
                        color="neutral"
                    >
                        临时用户
                    </UBadge>
                </div>
                
                <div class="flex items-center gap-4 text-sm text-slate-600 dark:text-slate-400 mt-1">
                    <!-- 位置信息 -->
                    <div v-if="user.location_name" class="flex items-center gap-1">
                        <UIcon name="i-heroicons-map-pin" class="w-4 h-4" />
                        <span class="truncate">{{ user.location_name }}</span>
                    </div>
                    
                    <!-- 距离信息 -->
                    <div v-if="user.distance !== undefined" class="flex items-center gap-1">
                        <UIcon name="i-heroicons-signal" class="w-4 h-4" />
                        <span>{{ formatDistance(user.distance) }}</span>
                    </div>
                </div>

                <!-- 个人简介 -->
                <p v-if="user.bio" class="text-sm text-slate-600 dark:text-slate-400 mt-2 line-clamp-2">
                    {{ user.bio }}
                </p>
            </div>

            <!-- 操作按钮 -->
            <div class="flex items-center gap-2">
                <UButton
                    variant="ghost"
                    size="sm"
                    icon="i-heroicons-chat-bubble-left"
                    class="opacity-0 group-hover:opacity-100 transition-opacity btn-press"
                    @click.stop="$emit('message', user)"
                >
                    聊天
                </UButton>
                <UButton
                    variant="ghost"
                    size="sm"
                    icon="i-heroicons-ellipsis-horizontal"
                    class="opacity-0 group-hover:opacity-100 transition-opacity btn-press"
                    @click.stop="$emit('more', user)"
                />
            </div>
        </div>
    </UCard>
</template>

<script setup lang="ts">
import type { User } from '~/types'

interface Props {
    user: User
}

interface Emits {
    click: [user: User]
    message: [user: User]
    more: [user: User]
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
</script>
