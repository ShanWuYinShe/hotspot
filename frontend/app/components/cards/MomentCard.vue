<template>
    <UCard
        class="interactive-card group cursor-pointer"
        @click="$emit('click', moment)"
    >
        <!-- 用户信息头部 -->
        <template #header>
            <div class="flex items-center justify-between">
                <UUser
                    :name="moment.user.nickname"
                    :description="userDescription"
                    :avatar="{
                        src: moment.user.avatar_url,
                        alt: moment.user.nickname,
                        size: 'md',
                        chip: isOnline ? { color: 'success', position: 'bottom-right' } : false
                    }"
                    :chip="isMyMoment ? { text: '我的', color: 'primary' } : false"
                    size="md"
                    class="flex-1 min-w-0"
                >
                    <template #description>
                        <div class="flex items-center gap-3 text-sm text-secondary-content">
                            <div class="flex items-center gap-1">
                                <UIcon name="i-heroicons-clock" class="icon-sm" />
                                <span>{{ formatTime(moment.created_at) }}</span>
                            </div>
                            
                            <div v-if="moment.geo.location_name" class="flex items-center gap-1">
                                <UIcon name="i-heroicons-map-pin" class="icon-sm" />
                                <span class="truncate">{{ moment.geo.location_name }}</span>
                            </div>
                            
                            <div v-if="moment.geo.distance !== undefined" class="flex items-center gap-1">
                                <UIcon name="i-heroicons-signal" class="icon-sm" />
                                <span>{{ formatDistance(moment.geo.distance) }}</span>
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
                <p class="text-primary-content leading-relaxed">
                    {{ moment.description }}
                </p>
            </div>

            <!-- 图片展示（支持最多9张） -->
            <div v-if="moment.images?.length" class="mt-3">
                <!-- 单张图片 -->
                <div v-if="moment.images.length === 1" class="rounded-lg overflow-hidden">
                    <NuxtImg
                        :src="moment.images[0]"
                        :alt="moment.description"
                        class="w-full max-h-96 object-cover cursor-pointer hover:opacity-90 transition-opacity"
                        @click.stop="handleImageClick(moment.images[0], 0)"
                    />
                </div>

                <!-- 2-4张图片（网格布局） -->
                <div 
                    v-else-if="moment.images.length <= 4"
                    :class="[
                        'grid gap-2 rounded-lg overflow-hidden',
                        moment.images.length === 2 ? 'grid-cols-2' : 'grid-cols-2'
                    ]"
                >
                    <NuxtImg
                        v-for="(image, index) in moment.images"
                        :key="index"
                        :src="image"
                        :alt="`${moment.description} - ${index + 1}`"
                        class="w-full h-48 object-cover cursor-pointer hover:opacity-90 transition-opacity"
                        @click.stop="handleImageClick(image, index)"
                    />
                </div>

                <!-- 5-9张图片（3x3网格） -->
                <div 
                    v-else
                    class="grid grid-cols-3 gap-2 rounded-lg overflow-hidden"
                >
                    <template v-for="(image, index) in moment.images.slice(0, 9)" :key="index">
                        <div class="relative">
                            <NuxtImg
                                :src="image"
                                :alt="`${moment.description} - ${index + 1}`"
                                class="w-full h-32 object-cover cursor-pointer hover:opacity-90 transition-opacity"
                                @click.stop="handleImageClick(image, index)"
                            />
                            <!-- 第9张图片显示"更多"遮罩 -->
                            <div
                                v-if="index === 8 && moment.images.length > 9"
                                class="absolute inset-0 bg-black/50 flex items-center justify-center text-white font-semibold"
                                @click.stop="$emit('more-images', moment)"
                            >
                                +{{ moment.images.length - 9 }}
                            </div>
                        </div>
                    </template>
                </div>
            </div>

            <!-- 标签 -->
            <div v-if="moment.tags?.length" class="flex flex-wrap gap-2 mt-3">
                <UBadge
                    v-for="tag in moment.tags"
                    :key="tag"
                    variant="soft"
                    size="sm"
                    color="neutral"
                    class="cursor-pointer hover-primary transition-colors"
                    @click.stop="$emit('tag-click', tag)"
                >
                    #{{ tag }}
                </UBadge>
            </div>
        </div>

        <!-- 互动区域 -->
        <template #footer>
            <div class="flex items-center justify-between">
                <!-- 互动按钮 -->
                <div class="flex items-center gap-1">
                    <UButton
                        variant="ghost"
                        size="sm"
                        @click.stop="handleLike"
                        :color="moment.liked_by_me ? 'error' : 'neutral'"
                        class="transition-colors"
                    >
                        <UIcon
                            :name="moment.liked_by_me ? 'i-heroicons-heart-solid' : 'i-heroicons-heart'"
                            class="icon-sm mr-1"
                        />
                        {{ moment.likes_count }}
                    </UButton>

                    <UButton 
                        variant="ghost" 
                        size="sm" 
                        color="neutral"
                        @click.stop="handleComment"
                    >
                        <UIcon name="i-heroicons-chat-bubble-left" class="icon-sm mr-1" />
                        {{ moment.comments_count }}
                    </UButton>

                    <UButton 
                        variant="ghost" 
                        size="sm" 
                        color="neutral"
                        @click.stop="handleShare"
                    >
                        <UIcon name="i-heroicons-share" class="icon-sm mr-1" />
                        分享
                    </UButton>
                </div>

                <!-- 可见性功能暂未实现，等待后端API支持 -->
            </div>
        </template>
    </UCard>
</template>

<script setup lang="ts">
import { useMomentsService } from '~/services/moments.service';
import { useUsersService } from '~/services/users.service';
import { getErrorMessage } from '~/utils/error';
import type { MomentSearchItem } from '~/types/api'

// MomentCard 使用 MomentSearchItem 类型（包含 geo 和 tags）
// 添加可选的 liked_by_me 字段用于 UI 状态显示
interface MomentCardData extends MomentSearchItem {
  liked_by_me?: boolean;  // 当前用户是否已点赞（客户端状态）
}

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
const toast = useToast()

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
    return props.moment.user.user_id === authStore.user?.user_id
})

// 在线状态：展示组件不需要实时在线状态，默认不显示
const isOnline = computed(() => false)

// 用户描述信息
const userDescription = computed(() => {
    const parts = []
    if (props.moment.geo.location_name) {
        parts.push(props.moment.geo.location_name)
    }
    if (props.moment.geo.distance !== undefined) {
        parts.push(formatDistance(props.moment.geo.distance))
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
            class: 'text-error'
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
            class: 'text-error'
        }])
    }

    return items
})


// 服务和功能管理
const momentService = useMomentsService()
const userService = useUsersService()

// 编辑动态
const handleEditMoment = async () => {
    // 实际实现编辑功能
    const toast = useToast()
    toast.add({
        title: '编辑功能',
        description: '编辑功能正在开发中',
        color: 'info'
    })
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
        toast.add({
            title: '删除成功',
            description: '动态已删除',
            color: 'success'
        })
        // 触发父组件刷新
        emit('deleted', props.moment)
    } catch (error: unknown) {
        toast.add({
            title: '删除失败',
            description: getErrorMessage(error, '删除动态时发生错误'),
            color: 'error'
        })
    }
}

// 关注用户
const handleFollowUser = async () => {
    try {
        const userService = useUsersService()
        await userService.followUser({ id: props.moment.user.user_id })
        toast.add({ color: 'success', title: '关注成功' })
    } catch (error: unknown) {
        toast.add({ color: 'error', title: '关注失败', description: getErrorMessage(error) })
    }
}

// 发送消息
const handleSendMessage = async () => {
    if (props.moment.user.user_id === authStore.user?.user_id) {
        navigateTo('/profile')
    } else {
        navigateTo(`/chat/users/${props.moment.user.user_id}`)
    }
}

// 举报动态
const handleReportMoment = async () => {
    const modal = useModal()
    const toast = useToast()
    
    await modal.showReport('moment', props.moment.id)
    toast.add({
        title: '举报已提交',
        description: '我们会尽快处理您的举报',
        color: 'success'
    })
}

// 图片点击处理
const handleImageClick = (image: string, index: number) => {
    emit('image-click', { image, index })
}
</script>
