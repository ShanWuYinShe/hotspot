<template>
    <UCard 
        class="group bg-white/60 dark:bg-slate-800/60 border border-slate-200/50 dark:border-slate-700/50 cursor-pointer hover:bg-white/80 dark:hover:bg-slate-800/80 transition-colors"
        @click="$emit('click', moment)"
    >
        <div class="space-y-4">
            <!-- 用户信息 -->
            <div class="flex items-center gap-4">
                <div class="relative">
                    <UAvatar 
                        :alt="moment.nickname" 
                        :src="moment.user_avatar"
                        size="lg"
                    >
                        {{ moment.nickname?.charAt(0) }}
                    </UAvatar>
                    <div class="absolute -bottom-1 -right-1 w-4 h-4 bg-green-500 rounded-full border-2 border-white dark:border-slate-800"></div>
                </div>
                <div class="flex-1 min-w-0">
                    <div class="flex items-center gap-2">
                        <h3 class="font-semibold text-slate-900 dark:text-white truncate">{{ moment.nickname }}</h3>
                        <UBadge 
                            v-if="isMyMoment" 
                            variant="soft" 
                            size="xs" 
                            color="primary"
                        >
                            我的
                        </UBadge>
                    </div>
                    <div class="flex items-center gap-2 text-sm text-slate-600 dark:text-slate-400">
                        <UIcon name="i-heroicons-clock" class="w-4 h-4" />
                        <span>{{ formatTime(moment.created_at) }}</span>
                        <span v-if="moment.location_name" class="flex items-center gap-1">
                            <UIcon name="i-heroicons-map-pin" class="w-4 h-4" />
                            <span class="truncate">{{ moment.location_name }}</span>
                        </span>
                    </div>
                </div>
                <UButton
                    variant="ghost"
                    size="sm"
                    icon="i-heroicons-ellipsis-horizontal"
                    class="opacity-0 group-hover:opacity-100 transition-opacity"
                    @click.stop="$emit('more', moment)"
                />
            </div>

            <!-- 动态内容 -->
            <div class="space-y-4">
                <p class="text-slate-900 dark:text-slate-100 leading-relaxed text-lg">
                    {{ moment.description }}
                </p>

                <!-- 标签 -->
                <div v-if="moment.tags?.length" class="flex flex-wrap gap-2">
                    <UBadge
                        v-for="tag in moment.tags"
                        :key="tag"
                        variant="soft"
                        size="sm"
                        color="neutral"
                        class="cursor-pointer hover:color-primary"
                        @click.stop="$emit('tag-click', tag)"
                    >
                        #{{ tag }}
                    </UBadge>
                </div>
            </div>

            <!-- 互动信息 -->
            <div class="flex items-center justify-between pt-4 border-t border-slate-200/50 dark:border-slate-700/50">
                <div class="flex items-center gap-6">
                    <UButton
                        variant="ghost"
                        size="sm"
                        @click.stop="$emit('like', moment)"
                        :color="moment.liked_by_me ? 'error' : 'neutral'"
                    >
                        <UIcon
                            :name="moment.liked_by_me ? 'i-heroicons-heart-solid' : 'i-heroicons-heart'"
                            class="w-5 h-5 mr-2"
                        />
                        {{ moment.likes_count }}
                    </UButton>

                    <UButton 
                        variant="ghost" 
                        size="sm" 
                        color="neutral"
                        @click.stop="$emit('comment', moment)"
                    >
                        <UIcon name="i-heroicons-chat-bubble-left" class="w-5 h-5 mr-2" />
                        {{ moment.comments_count }}
                    </UButton>

                    <UButton 
                        variant="ghost" 
                        size="sm" 
                        color="neutral"
                        @click.stop="$emit('share', moment)"
                    >
                        <UIcon name="i-heroicons-share" class="w-5 h-5 mr-2" />
                        分享
                    </UButton>
                </div>

                <div v-if="moment.distance !== undefined" class="flex items-center gap-1 text-sm text-slate-500">
                    <UIcon name="i-heroicons-map-pin" class="w-4 h-4" />
                    <span>{{ formatDistance(moment.distance) }}</span>
                </div>
            </div>
        </div>
    </UCard>
</template>

<script setup lang="ts">
import type { Moment } from '~/types'

interface Props {
    moment: Moment
}

interface Emits {
    click: [moment: Moment]
    like: [moment: Moment]
    comment: [moment: Moment]
    share: [moment: Moment]
    more: [moment: Moment]
    'tag-click': [tag: string]
}

const props = defineProps<Props>()
defineEmits<Emits>()

// 检查是否是当前用户的动态
const authStore = useAuthStore()
const isMyMoment = computed(() => {
    return props.moment.user_id === authStore.user?.user_id
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
</script>
