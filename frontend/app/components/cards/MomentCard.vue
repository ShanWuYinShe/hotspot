<template>
    <UCard
        variant="outline"
        class="group flex flex-col h-full overflow-hidden"
        data-hs-card="interactive"
        @click="$emit('click', moment)"
    >
        <!-- 主体内容 (Header + Body) -->
        <div class="flex flex-col h-full">
            <!-- 头部：用户信息与时间 -->
            <div class="flex items-start justify-between p-3 gap-3">
                <UUser
                    class="min-w-0"
                    :name="moment.user.nickname"
                    :avatar="{ src: moment.user.avatar_url, alt: moment.user.nickname, size: 'md' }"
                />

                <div class="flex items-center gap-2 shrink-0">
                    <UBadge icon="i-lucide-clock">
                        {{ formatTime(moment.created_at) }}
                    </UBadge>
                    <UBadge v-if="moment.geo?.distance !== undefined" icon="i-lucide-plane">
                        {{ formatDistance(moment.geo.distance) }}
                    </UBadge>
                    <UBadge v-if="isMyMoment" label="我的" color="primary" />
                </div>

                <!-- 操作按钮组 -->
                <div class="flex items-center gap-1">
                    <!-- 我的动态：编辑和删除 -->
                    <template v-if="isMyMoment">
                        <UTooltip text="编辑动态" :portal="false">
                            <UButton
                                variant="ghost"
                                color="neutral"
                                size="sm"
                                icon="i-lucide-pencil"
                                @click.stop="handleEditMoment"
                            />
                        </UTooltip>
                        <UTooltip text="删除动态" :portal="false">
                            <UButton
                                variant="ghost"
                                color="error"
                                size="sm"
                                icon="i-lucide-trash-2"
                                @click.stop="handleDeleteMoment"
                            />
                        </UTooltip>
                    </template>

                    <!-- 他人动态：关注、私信、举报 -->
                    <template v-else>
                        <UTooltip v-if="!moment.relationship?.is_following" text="关注用户" :portal="false">
                            <UButton
                                variant="ghost"
                                color="primary"
                                size="sm"
                                icon="i-lucide-user-plus"
                                @click.stop="handleFollowUser"
                            />
                        </UTooltip>
                        
                        <UTooltip text="发送私信" :portal="false">
                            <UButton
                                variant="ghost"
                                color="neutral"
                                size="sm"
                                icon="i-lucide-message-square-text"
                                @click.stop="handleSendMessage"
                            />
                        </UTooltip>

                        <UTooltip text="举报内容" :portal="false">
                            <UButton
                                variant="ghost"
                                color="error"
                                size="sm"
                                icon="i-lucide-flag"
                                @click.stop="handleReportMoment"
                            />
                        </UTooltip>
                    </template>
                </div>
            </div>

            <!-- 动态内容 -->
            <div class="px-3 pb-3 space-y-3 flex-1">
                <!-- 文字内容 -->
                <div v-if="moment.description" class="text-sm leading-relaxed line-clamp-4">
                    {{ moment.description }}
                </div>

                <!-- 图片展示 -->
                <div v-if="moment.images?.length" class="mt-2">
                    <!-- 单张图片 -->
                    <div v-if="moment.images.length === 1" class="rounded-xl overflow-hidden aspect-video ring-1 ring-gray-200 dark:ring-gray-800">
                        <NuxtImg
                            :src="moment.images[0]"
                            :alt="moment.description"
                            class="w-full h-full object-cover cursor-pointer hover:opacity-95 transition-opacity"
                            @click.stop="handleImageClick(moment.images?.[0] || '', 0)"
                        />
                    </div>

                    <div 
                        v-else-if="moment.images.length <= 4"
                        class="grid gap-0.5 rounded-xl overflow-hidden ring-1 ring-gray-200 dark:ring-gray-800"
                        :class="moment.images.length === 2 ? 'grid-cols-2 aspect-2/1' : 'grid-cols-2 aspect-square'"
                    >
                        <div v-for="(image, index) in moment.images" :key="index" class="relative w-full h-full overflow-hidden">
                            <NuxtImg
                                :src="image"
                                :alt="`${moment.description} - ${index + 1}`"
                                class="w-full h-full object-cover cursor-pointer hover:opacity-95 transition-opacity absolute inset-0"
                                @click.stop="handleImageClick(image, index)"
                            />
                        </div>
                    </div>

                    <!-- 5-9张图片（3x3网格） -->
                    <div 
                        v-else
                        class="grid grid-cols-3 gap-0.5 rounded-xl overflow-hidden aspect-square ring-1 ring-gray-200 dark:ring-gray-800"
                    >
                        <div v-for="(image, index) in moment.images.slice(0, 9)" :key="index" class="relative w-full h-full overflow-hidden">
                            <NuxtImg
                                :src="image"
                                :alt="`${moment.description} - ${index + 1}`"
                                class="w-full h-full object-cover cursor-pointer hover:opacity-95 transition-opacity absolute inset-0"
                                @click.stop="handleImageClick(image, index)"
                            />
                            <!-- 第9张图片显示"更多"遮罩 -->
                            <div
                                v-if="index === 8 && moment.images.length > 9"
                                class="hs-image-more-overlay"
                                @click.stop="$emit('more-images', moment)"
                            >
                                <UBadge color="neutral" variant="solid" size="xl">
                                    +{{ moment.images.length - 9 }}
                                </UBadge>
                            </div>
                        </div>
                    </div>
                </div>

                <!-- 位置和标签 -->
                <div v-if="moment.geo?.location_name || moment.tags?.length" class="flex flex-wrap gap-2 pt-1">
                    <!-- 位置标签 -->
                    <UBadge
                        v-if="moment.geo?.location_name"
                        :label="moment.geo.location_name || ''"
                        icon="i-lucide-map-pin"
                        color="info"
                        variant="soft"
                        size="xs"
                        class="max-w-full truncate"
                    />
                    
                    <!-- 话题标签 -->
                    <UButton
                        v-for="tag in moment.tags"
                        :key="tag"
                        type="button"
                        variant="soft"
                        color="primary"
                        size="xs"
                        class="cursor-pointer"
                        @click.stop="$emit('tag-click', tag)"
                    >
                        #{{ tag }}
                    </UButton>
                </div>
            </div>
        </div>

        <!-- 底部互动栏 -->
        <template #footer>
            <div class="flex items-center gap-2 w-full">
                <!-- 点赞 -->
                <UButton
                    variant="ghost"
                    size="md"
                    :color="moment.liked_by_me ? 'error' : 'neutral'"
                    :icon="'i-lucide-heart'"
                    :label="String(moment.likes_count || 0)"
                    class="flex-1 justify-center"
                    @click.stop="handleLike"
                />

                <!-- 评论 -->
                <UButton 
                    variant="ghost" 
                    size="md" 
                    color="neutral"
                    icon="i-lucide-message-circle"
                    :label="String(moment.comments_count || 0)"
                    class="flex-1 justify-center"
                    @click.stop="handleComment"
                />

                <!-- 分享 -->
                <UButton 
                    variant="ghost" 
                    size="md" 
                    color="neutral"
                    icon="i-lucide-share-2"
                    label="分享"
                    class="flex-1 justify-center"
                    @click.stop="handleShare"
                />
            </div>
        </template>
    </UCard>
</template>

<script setup lang="ts">
import { useMomentsService } from '~/services/moments.service';
import { useUsersService } from '~/services/users.service';
import type { MomentCardData } from '~/types/client';

interface Props {
    moment: MomentCardData
}

interface Emits {
    click: [moment: MomentCardData]
    like: [moment: MomentCardData]
    comment: [moment: MomentCardData]
    share: [moment: MomentCardData]
    more: [moment: MomentCardData]
    deleted: [moment: MomentCardData]
    updated: [momentId: string]
    'tag-click': [tag: string]
    'image-click': [data: { image: string; index: number }]
    'more-images': [moment: MomentCardData]
}

const props = defineProps<Props>()
const emit = defineEmits<Emits>()
const { formatTime, formatDistance } = useFormatters()
const { requireAuth } = useRequireAuth()

// 使用水合安全的 store 访问
const authStore = useAuthStore()
const hydratedUser = useHydratedStore(authStore, 'user', null as typeof authStore.user)
const feedback = useFeedbackStore()

// ============================================================================
// 功能级认证处理函数
// ============================================================================

// 点赞（需要认证）
const handleLike = () => {
    if (!requireAuth('点赞')) return
    emit('like', props.moment)
}

// 评论（需要认证）
const handleComment = () => {
    if (!requireAuth('发表评论')) return
    emit('comment', props.moment)
}

// 分享（需要认证）
const handleShare = () => {
    if (!requireAuth('分享动态')) return
    emit('share', props.moment)
}

// 检查是否是当前用户的动态
const isMyMoment = computed(() => {
    return props.moment.user.user_id === hydratedUser.value?.user_id
})

// 服务和功能管理
const momentService = useMomentsService()
const userService = useUsersService()

// 编辑动态
const handleEditMoment = async () => {
    if (!requireAuth('编辑动态')) return

    const modal = useModal()
    const updatedId = await modal.showEditMoment({
        id: props.moment.id,
        description: props.moment.description,
        tags: props.moment.tags,
        images: props.moment.images,
        geo: props.moment.geo
    })
    if (updatedId) emit('updated', updatedId)
}

// 删除动态
const handleDeleteMoment = async () => {
    const modal = useModal()
    const confirmed = await modal.showConfirm({
        title: '删除动态',
        message: '确定要删除这条动态吗？此操作无法撤销。',
        confirmText: '删除',
        confirmColor: 'error'
    })
    if (!confirmed) return

    try {
        await momentService.deleteMoment({
            id: props.moment.id
        })
        feedback.success('删除成功', '动态已删除')
        // 触发父组件刷新
        emit('deleted', props.moment)
    } catch (error: unknown) {
        feedback.error('删除失败', getErrorMessage(error, '删除动态时发生错误'))
    }
}

// 关注用户
const handleFollowUser = async () => {
    if (!requireAuth('关注用户')) return
    try {
        const userService = useUsersService()
        await userService.follow({ id: props.moment.user.user_id })
        feedback.success('关注成功')
    } catch (error: unknown) {
        feedback.error('关注失败', getErrorMessage(error))
    }
}

// 发送消息
const handleSendMessage = async () => {
    if (!requireAuth('发送消息')) return
    if (props.moment.user.user_id === hydratedUser.value?.user_id) {
        navigateTo('/profile')
    } else {
        navigateTo(`/chat/users/${props.moment.user.user_id}`)
    }
}

// 举报动态
const handleReportMoment = async () => {
    if (!requireAuth('提交举报')) return
    const modal = useModal()
    
    await modal.showReport('moment', props.moment.id)
    feedback.success('举报已提交', '我们会尽快处理您的举报')
}

// 图片点击处理
const handleImageClick = (image: string, index: number) => {
    emit('image-click', { image, index })
}
</script>