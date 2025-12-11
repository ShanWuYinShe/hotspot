<template>
    <UCard
        class="interactive-card group cursor-pointer"
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
                            <h3 class="font-semibold text-primary-content truncate">
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

                        <div class="flex items-center gap-2 text-xs text-secondary-content mt-1">
                            <UChip
                                v-if="group.my_role === 'owner'"
                                text="群主"
                                color="success"
                                size="sm"
                            />
                            <UChip
                                v-else-if="group.my_role === 'admin'"
                                text="管理员"
                                color="warning"
                                size="sm"
                            />

                            <span class="flex items-center gap-1">
                                <UIcon
                                    name="i-heroicons-users"
                                    class="icon-sm"
                                />
                                {{ group.members_count }} 人
                            </span>
                        </div>
                    </div>
                </div>

                <!-- 操作按钮 -->
                <div class="flex items-center gap-1">
                    <UButton
                        v-if="!group.is_member"
                        variant="soft"
                        size="sm"
                        color="primary"
                        icon="i-heroicons-plus"
                        @click.stop="$emit('join', group)"
                    >
                        加入
                    </UButton>
                    <UButton
                        v-else
                        variant="ghost"
                        size="sm"
                        icon="i-heroicons-chat-bubble-left-right"
                        class="opacity-0 group-hover:opacity-100 transition-opacity"
                        @click.stop="$emit('enter', group)"
                    />

                    <UDropdownMenu
                        :items="groupMenuItems"
                        @click.stop
                    >
                        <UButton
                            variant="ghost"
                            size="sm"
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
            <p
                v-if="group.description"
                class="text-sm text-secondary-content line-clamp-2"
            >
                {{ group.description }}
            </p>

            <!-- 位置和距离信息（仅当有 geo 数据时显示） -->
            <div
                v-if="group.geo"
                class="flex items-center gap-4 text-sm text-secondary-content"
            >
                <div
                    v-if="group.geo.location_name"
                    class="flex items-center gap-1"
                >
                    <UIcon
                        name="i-heroicons-map-pin"
                        class="icon-sm"
                    />
                    <span class="truncate">{{ group.geo.location_name }}</span>
                </div>

                <div
                    v-if="group.geo.distance !== undefined"
                    class="flex items-center gap-1"
                >
                    <UIcon
                        name="i-heroicons-signal"
                        class="icon-sm"
                    />
                    <span>{{ formatDistance(group.geo.distance) }}</span>
                </div>
            </div>
        </div>

        <!-- 群组底部信息 -->
        <template #footer>
            <div class="flex items-center justify-between text-xs text-secondary-content">
                <div class="flex items-center gap-1">
                    <UIcon
                        name="i-heroicons-clock"
                        class="icon-sm"
                    />
                    <span>活跃: {{ formatTime(group.last_activity_at) }}</span>
                </div>

                <div class="flex items-center gap-1">
                    <UIcon
                        name="i-heroicons-calendar"
                        class="icon-sm"
                    />
                    <span>{{ formatTime(group.created_at) }}</span>
                </div>
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
import { getErrorMessage } from '~/utils/error'

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
        if (props.group.my_role === 'owner' || props.group.my_role === 'admin') {
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
            class: 'text-error'
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

    const toast = useToast()
    const groupService = useGroupsService()
    try {
        await groupService.leaveGroup({
            id: props.group.id
        })

        toast.add({
            title: '退出成功',
            description: `已退出群组"${props.group.name}"`,
            color: 'success'
        })

        // 刷新群组列表
        await refreshNuxtData('groups')
    } catch (error: unknown) {
        toast.add({
            title: '退出失败',
            description: getErrorMessage(error, '退出群组时发生错误'),
            color: 'error'
        })
    }
}
</script>
