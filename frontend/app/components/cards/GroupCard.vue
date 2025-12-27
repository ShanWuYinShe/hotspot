<template>
    <UCard
        variant="outline"
        class="group"
        data-hs-card="interactive"
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
                        :chip="group.has_password ? {
                            color: 'warning',
                            position: 'bottom-right',
                            text: '🔒'
                        } : false"
                    >
                        {{ group.name?.charAt(0) }}
                    </UAvatar>

                    <div class="flex-1 min-w-0">
                        <div class="flex items-center gap-2">
                            <h3 class="font-bold tracking-tight truncate">
                                {{ group.name }}
                            </h3>
                            <UBadge v-if="group.is_member" variant="soft" size="xs" color="primary">
                                已加入
                            </UBadge>
                        </div>

                        <div class="flex items-center gap-2 mt-1 flex-wrap">
                            <UChip v-if="group.my_role === 'owner'" text="群主" color="success" size="sm" />
                            <UChip v-else-if="group.my_role === 'admin'" text="管理员" color="warning" size="sm" />

                            <UBadge
                                :label="`${group.members_count} 人`"
                                icon="i-lucide-users"
                                color="neutral"
                                variant="soft"
                                size="xs"
                            />
                        </div>
                    </div>
                </div>

                <!-- 操作按钮 -->
                <div class="flex items-center gap-1">
                    <UButton
                        v-if="!group.is_member"
                        variant="soft"
                        size="md"
                        color="primary"
                        icon="i-lucide-plus"
                        @click.stop="$emit('join', group)"
                    >
                        加入
                    </UButton>
                    
                    <template v-else>
                        <UTooltip text="进入群聊" :portal="false">
                            <UButton
                                variant="ghost"
                                color="neutral"
                                size="md"
                                icon="i-lucide-messages-square"
                                @click.stop="$emit('enter', group)"
                            />
                        </UTooltip>

                        <UTooltip text="群组设置" :portal="false">
                            <UButton
                                variant="ghost"
                                color="neutral"
                                size="md"
                                icon="i-lucide-settings"
                                @click.stop="handleGroupSettings"
                            />
                        </UTooltip>

                        <UTooltip v-if="group.my_role === 'owner' || group.my_role === 'admin'" text="管理成员" :portal="false">
                            <UButton
                                variant="ghost"
                                color="neutral"
                                size="md"
                                icon="i-lucide-users"
                                @click.stop="handleManageMembers"
                            />
                        </UTooltip>

                        <UTooltip text="退出群组" :portal="false">
                            <UButton
                                variant="ghost"
                                color="error"
                                size="md"
                                icon="i-lucide-log-out"
                                @click.stop="handleLeaveGroup"
                            />
                        </UTooltip>
                    </template>
                </div>
            </div>
        </template>

        <!-- 群组内容 -->
        <div class="space-y-3">
            <!-- 群组描述 -->
            <p
                v-if="group.description"
                class="text-sm line-clamp-2"
            >
                {{ group.description }}
            </p>

            <!-- 位置和距离信息（仅当有 geo 数据时显示） -->
            <div
                v-if="group.geo"
                class="flex items-center gap-2 mt-2"
            >
                <UBadge
                    v-if="group.geo?.location_name"
                    color="info"
                    variant="soft"
                    size="xs"
                    class="flex items-center gap-1"
                >
                    <UIcon
                        name="i-lucide-map-pin"
                        :size="14"
                    />
                    <span class="truncate max-w-[120px] font-medium">{{ group.geo.location_name }}</span>
                </UBadge>

                <UBadge
                    v-if="group.geo?.distance !== undefined"
                    color="success"
                    variant="soft"
                    size="xs"
                    class="flex items-center gap-1"
                >
                    <UIcon
                        name="i-lucide-signal"
                        :size="14"
                    />
                    <span class="font-medium">{{ formatDistance(group.geo.distance) }}</span>
                </UBadge>
            </div>
        </div>

        <!-- 群组底部信息 -->
        <template #footer>
            <div class="flex items-center justify-between gap-2">
                <UBadge
                    v-if="group.last_activity_at"
                    color="neutral"
                    variant="soft"
                    size="xs"
                    icon="i-lucide-clock"
                >
                    活跃: {{ formatTime(group.last_activity_at) }}
                </UBadge>

                <UBadge
                    v-if="group.created_at"
                    color="neutral"
                    variant="soft"
                    size="xs"
                    icon="i-lucide-calendar"
                >
                    {{ formatTime(group.created_at) }}
                </UBadge>
            </div>
        </template>
    </UCard>
</template>

<script setup lang="ts">
import type { GeoData } from '~/types/api'

// 群组卡片数据类型（支持搜索结果和已加入群组列表）
interface GroupCardData {
    id: string;
    name: string;
    avatar_url?: string;
    members_count: number;
    description?: string;
    my_role?: 'owner' | 'admin' | 'member';
    unread_count?: number;
    geo?: GeoData;              // 搜索结果有 geo，已加入列表没有
    has_password?: boolean;     // 搜索结果有，已加入列表没有
    is_member?: boolean;
    last_activity_at?: string;
    created_at?: string;
}
import { useGroupsService } from '~/services/groups.service'

/**
 * GroupCard Props - 接受 GroupCardData 必填字段
 */
interface Props {
    group: GroupCardData
}

interface Emits {
    click: [group: Props['group']]
    join: [group: Props['group']]
    enter: [group: Props['group']]
    more: [group: Props['group']]
}

const props = defineProps<Props>()
const emit = defineEmits<Emits>()
const { formatTime, formatDistance } = useFormatters()

// 群组设置
const handleGroupSettings = () => {
    navigateTo(`/groups/${props.group.id}/settings`)
}

// 管理成员
const handleManageMembers = () => {
    navigateTo(`/groups/${props.group.id}/members`)
}

// 退出群组
const handleLeaveGroup = async () => {
    const { showConfirm } = useModal()

    const confirmed = await showConfirm({
        title: '退出群组',
        message: `确定要退出群组"${props.group.name}"吗？退出后将无法接收群组消息。`,
        confirmText: '确认退出',
        cancelText: '取消',
        confirmColor: 'error'
    })

    if (!confirmed) return

    const feedback = useFeedbackStore()
    const groupService = useGroupsService()
    try {
        await groupService.leaveGroup({
            id: props.group.id
        })

        feedback.success('退出成功', `已退出群组"${props.group.name}"`)

        // 刷新群组列表
        await refreshNuxtData('groups')
    } catch (error: unknown) {
        feedback.error('退出失败', getErrorMessage(error, '退出群组时发生错误'))
    }
}
</script>
