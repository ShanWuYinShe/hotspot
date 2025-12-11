<template>
    <div class="mx-auto max-w-7xl space-y-6">
        <!-- 群组详情（仅当数据加载完成后显示） -->
        <UCard v-if="group">
            <template #header>
                <div class="flex flex-col sm:flex-row sm:items-center justify-between gap-4">
                    <div class="flex items-center gap-4">
                        <UAvatar 
                            :alt="group.name" 
                            :src="group.avatar_url"
                            size="xl"
                        >
                            {{ group.name?.charAt(0) }}
                        </UAvatar>
                        <div class="space-y-1">
                            <div class="flex items-center gap-2">
                                <h1 class="text-2xl font-bold text-primary-content">
                                    {{ group.name }}
                                </h1>
                                <UBadge v-if="group.has_password" variant="subtle" color="warning" size="xs">
                                    <UIcon name="i-heroicons-lock-closed" class="icon-xs mr-1" />
                                    需密码
                                </UBadge>
                            </div>
                            <div class="flex flex-wrap items-center gap-2 text-sm text-secondary-content">
                                <span class="flex items-center gap-1">
                                    <UIcon name="i-heroicons-map-pin" class="icon-sm" />
                                    {{ group.geo?.location_name || '未知位置' }}
                                </span>
                                <span>·</span>
                                <span class="flex items-center gap-1">
                                    <UIcon name="i-heroicons-users" class="icon-sm" />
                                    {{ group.members_count || 0 }} 人
                                </span>
                                <span v-if="group.geo?.distance !== undefined">·</span>
                                <span v-if="group.geo?.distance !== undefined" class="flex items-center gap-1">
                                    <UIcon name="i-heroicons-signal" class="icon-sm" />
                                    {{ formatDistance(group.geo.distance) }}
                                </span>
                            </div>
                            <!-- 我的角色标签 -->
                            <div v-if="userRole !== 'visitor'" class="flex items-center gap-2 mt-1">
                                <UBadge :color="roleConfig.color" variant="soft" size="xs">
                                    <UIcon :name="roleConfig.icon" class="icon-xs mr-1" />
                                    {{ roleConfig.label }}
                                </UBadge>
                                <UBadge v-if="groupUnread > 0" color="primary" variant="soft" size="xs">
                                    {{ groupUnread }} 条未读
                                </UBadge>
                            </div>
                        </div>
                    </div>
                    
                    <!-- 操作按钮 - 根据角色显示不同内容 -->
                    <div class="flex items-center gap-2">
                        <!-- 进入群聊（所有人可见，但只有成员能使用） -->
                        <UTooltip :text="userRole === 'visitor' ? '加入群组后可进入群聊' : '进入群聊'">
                            <UButton
                                :disabled="userRole === 'visitor'"
                                icon="i-heroicons-chat-bubble-left-right"
                                @click="goChat"
                            >
                                群聊
                            </UButton>
                        </UTooltip>
                        
                        <!-- 访客：加入群组 -->
                        <UButton
                            v-if="userRole === 'visitor'"
                            color="primary"
                            icon="i-heroicons-plus"
                            @click="onJoin"
                        >
                            加入群组
                        </UButton>
                        
                        <!-- 成员：更多操作菜单 -->
                        <UDropdownMenu v-else :items="actionMenuItems">
                            <UButton
                                variant="outline"
                                icon="i-heroicons-ellipsis-horizontal"
                            />
                        </UDropdownMenu>
                    </div>
                </div>
            </template>

            <!-- 群组描述 -->
            <div class="space-y-4">
                <p v-if="group.description" class="text-emphasis">
                    {{ group.description }}
                </p>
                <p v-else class="text-secondary-content italic">
                    暂无群组介绍
                </p>
                
                <!-- 标签 -->
                <div v-if="group.tags?.length" class="flex flex-wrap gap-2">
                    <UBadge
                        v-for="tag in group.tags"
                        :key="tag"
                        variant="subtle"
                        color="neutral"
                        size="sm"
                    >
                        #{{ tag }}
                    </UBadge>
                </div>
                
                <!-- 时间信息 -->
                <div class="flex flex-wrap gap-4 text-sm text-secondary-content pt-2 border-t border-default">
                    <span class="flex items-center gap-1">
                        <UIcon name="i-heroicons-calendar" class="icon-sm" />
                        创建于 {{ formatTime(group.created_at) }}
                    </span>
                    <span v-if="group.last_activity_at" class="flex items-center gap-1">
                        <UIcon name="i-heroicons-clock" class="icon-sm" />
                        最近活跃 {{ formatTime(group.last_activity_at) }}
                    </span>
                </div>
            </div>
        </UCard>

        <!-- 群组不存在（加载完成后仍无数据时显示） -->
        <UCard v-else-if="!loading">
            <UEmpty
                icon="i-heroicons-user-group"
                title="群组不存在"
                description="该群组可能已被解散或不存在"
            >
                <template #actions>
                    <UButton to="/groups">返回群组列表</UButton>
                    <UButton color="primary" to="/groups/discover">发现群组</UButton>
                </template>
            </UEmpty>
        </UCard>

        <!-- 群组详细信息（仅当群组存在时显示） -->
        <UTabs v-if="group" :items="tabItems" class="w-full">
            <!-- 成员列表 -->
            <template #members>
                <UCard class="mt-4">
                    <template #header>
                        <div class="flex items-center justify-between">
                            <div class="flex items-center gap-2">
                                <span class="font-medium">成员列表</span>
                                <UBadge v-if="members.length > 0" variant="soft" size="xs">
                                    {{ members.length }}
                                </UBadge>
                            </div>
                            <UButton
                                v-if="canInvite"
                                variant="outline"
                                size="sm"
                                icon="i-heroicons-user-plus"
                                @click="showInviteModal"
                            >
                                邀请成员
                            </UButton>
                        </div>
                    </template>
                    
                    <UTable :data="pagedMembers" :columns="memberColumns">
                        <template #member-cell="{ row }">
                            <UUser
                                :name="row.original.user.nickname"
                                :description="getMemberDescription(row.original)"
                                :avatar="{
                                    src: row.original.user.avatar_url,
                                    alt: row.original.user.nickname,
                                    size: 'md'
                                }"
                                size="md"
                            />
                        </template>
                        <template #role-cell="{ row }">
                            <UBadge 
                                :color="getMemberRoleColor(row.original.role)" 
                                variant="soft" 
                                size="xs"
                            >
                                {{ getMemberRoleLabel(row.original.role) }}
                            </UBadge>
                        </template>
                        <template #joined-cell="{ row }">
                            <span class="text-sm text-secondary-content">
                                {{ formatTime(row.original.joined_at) }}
                            </span>
                        </template>
                        <template #actions-cell="{ row }">
                            <div class="flex items-center gap-1">
                                    <UTooltip text="查看资料">
                                    <UButton
                                        variant="ghost"
                                        size="xs"
                                        icon="i-heroicons-user"
                                        @click="row.original.user.user_id === authStore.user?.user_id ? navigateTo('/profile') : navigateTo(`/users/${row.original.user.user_id}`)"
                                    />
                                </UTooltip>
                                <UTooltip text="发送私信">
                                    <UButton
                                        variant="ghost"
                                        size="xs"
                                        icon="i-heroicons-chat-bubble-left"
                                        @click="sendMessage(row.original)"
                                    />
                                </UTooltip>
                                <UDropdownMenu 
                                    v-if="canManageMember(row.original)" 
                                    :items="getMemberMenuItems(row.original)"
                                >
                                    <UButton
                                        variant="ghost"
                                        size="xs"
                                        icon="i-heroicons-ellipsis-horizontal"
                                    />
                                </UDropdownMenu>
                            </div>
                        </template>
                        <template #empty>
                            <UEmpty
                                icon="i-heroicons-users"
                                title="暂无成员"
                                description="邀请朋友一起加入群组吧"
                            />
                        </template>
                    </UTable>
                    
                    <template v-if="pageCount > 1" #footer>
                        <div class="flex justify-center">
                            <UPagination v-model:page="page" :items-per-page="pageSize" :total="members.length" />
                        </div>
                    </template>
                </UCard>
            </template>

            <!-- 群组动态 -->
            <template #activities>
                <UCard class="mt-4">
                    <UEmpty
                        icon="i-heroicons-clock"
                        title="群组动态"
                        description="功能开发中，敬请期待..."
                    />
                </UCard>
            </template>

            <!-- 群组设置（仅管理员和创建者可见） -->
            <template v-if="canManageGroup" #settings>
                <UCard class="mt-4">
                    <div class="space-y-4">
                        <div class="grid gap-4 sm:grid-cols-2">
                            <UButton
                                variant="outline"
                                block
                                class="justify-start"
                                icon="i-heroicons-pencil-square"
                                @click="navigateTo(`/groups/${groupId}/settings`)"
                            >
                                编辑群组资料
                            </UButton>
                            <UButton
                                variant="outline"
                                block
                                class="justify-start"
                                icon="i-heroicons-users"
                                @click="navigateTo(`/groups/${groupId}/settings?tab=members`)"
                            >
                                成员管理
                            </UButton>
                            <UButton
                                v-if="userRole === 'owner'"
                                variant="outline"
                                block
                                class="justify-start"
                                icon="i-heroicons-arrow-right-circle"
                                @click="showTransferModal"
                            >
                                转让群主
                            </UButton>
                            <UButton
                                v-if="userRole === 'owner'"
                                variant="outline"
                                color="error"
                                block
                                class="justify-start"
                                icon="i-heroicons-trash"
                                @click="showDismissModal"
                            >
                                解散群组
                            </UButton>
                        </div>
                    </div>
                </UCard>
            </template>
        </UTabs>
    </div>
</template>

<script setup lang="ts">
import type { TableColumn } from '@nuxt/ui'
import type { GroupMemberItem } from "~/types/api"
import type { GroupDetailData } from "~/types/client"
import { useGroupsService } from '~/services/groups.service'
import { getErrorMessage } from '~/utils/error'

// 扩展群组详情类型
interface GroupPageData extends GroupDetailData {
    last_activity_at?: string;
    avatar_url?: string;
    tags?: string[];
}

// 用户角色类型
type UserRole = 'owner' | 'admin' | 'member' | 'visitor'

// 成员行类型
type MemberRow = GroupMemberItem

definePageMeta({ title: "群组详情", layout: 'dashboard' })

const route = useRoute()
const router = useRouter()
const toast = useToast()
const groupsService = useGroupsService()
const geolocation = useGeolocation()
const modal = useModal()
const { formatTime, formatDistance } = useFormatters()

const groupId = computed(() =>
    String(Array.isArray(route.params.id) ? route.params.id[0] : route.params.id)
)

// 状态
const loading = ref(true)
const group = ref<GroupPageData | null>(null)
const groupUnread = ref(0)
const members = ref<GroupMemberItem[]>([])

// 分页
const page = ref(1)
const pageSize = ref(10)
const pageCount = computed(() => Math.max(1, Math.ceil(members.value.length / pageSize.value)))
const pagedMembers = computed(() => {
    const start = (page.value - 1) * pageSize.value
    return members.value.slice(start, start + pageSize.value)
})

// 当前用户
const authStore = useAuthStore()
const currentUserId = computed(() => authStore.user?.user_id)

// 用户角色判断 - 根据成员列表确定
const userRole = computed<UserRole>(() => {
    if (!currentUserId.value || !members.value.length) return 'visitor'
    
    const currentMember = members.value.find(m => m.user.user_id === currentUserId.value)
    if (!currentMember) return 'visitor'
    
    return currentMember.role || 'member'
})

// 角色配置
const roleConfig = computed(() => {
    const configs: Record<UserRole, { label: string; color: string; icon: string }> = {
        owner: { label: '群主', color: 'warning', icon: 'i-heroicons-crown' },
        admin: { label: '管理员', color: 'primary', icon: 'i-heroicons-shield-check' },
        member: { label: '成员', color: 'neutral', icon: 'i-heroicons-user' },
        visitor: { label: '访客', color: 'neutral', icon: 'i-heroicons-eye' }
    }
    return configs[userRole.value]
})

// 权限判断
const canManageGroup = computed(() => ['owner', 'admin'].includes(userRole.value))
const canInvite = computed(() => userRole.value !== 'visitor')

// Tab 配置
const tabItems = computed(() => {
    const items = [
        { label: '成员', slot: 'members', icon: 'i-heroicons-users' },
        { label: '动态', slot: 'activities', icon: 'i-heroicons-clock' }
    ]
    if (canManageGroup.value) {
        items.push({ label: '设置', slot: 'settings', icon: 'i-heroicons-cog-6-tooth' })
    }
    return items
})

// 操作菜单 - 根据角色显示不同内容
const actionMenuItems = computed(() => {
    const items: Array<Array<{ label: string; icon: string; click: () => void; class?: string }>> = []
    
    // 基础操作（所有成员可见）
    if (userRole.value !== 'visitor') {
        items.push([
            {
                label: '邀请成员',
                icon: 'i-heroicons-user-plus',
                click: showInviteModal
            },
            {
                label: '消息免打扰',
                icon: 'i-heroicons-bell-slash',
                click: toggleMute
            }
        ])
    }
    
    // 管理员操作
    if (canManageGroup.value) {
        items.push([
            {
                label: '群组设置',
                icon: 'i-heroicons-cog-6-tooth',
                click: () => navigateTo(`/groups/${groupId.value}/settings`)
            },
            {
                label: '成员管理',
                icon: 'i-heroicons-users',
                click: () => navigateTo(`/groups/${groupId.value}/settings?tab=members`)
            }
        ])
    }
    
    // 群主特有操作
    if (userRole.value === 'owner') {
        items.push([
            {
                label: '转让群主',
                icon: 'i-heroicons-arrow-right-circle',
                click: showTransferModal
            },
            {
                label: '解散群组',
                icon: 'i-heroicons-trash',
                click: showDismissModal,
                class: 'text-error'
            }
        ])
    }
    
    // 退出群组（非群主可见）
    if (userRole.value !== 'visitor' && userRole.value !== 'owner') {
        items.push([
            {
                label: '退出群组',
                icon: 'i-heroicons-arrow-right-on-rectangle',
                click: onLeave,
                class: 'text-error'
            }
        ])
    }
    
    return items
})

// 成员表格列
const memberColumns: TableColumn<MemberRow>[] = [
    { id: 'member', accessorKey: 'user', header: '成员' },
    { id: 'role', accessorKey: 'role', header: '角色' },
    { id: 'joined', accessorKey: 'joined_at', header: '加入时间' },
    { id: 'actions', accessorKey: 'user', header: '操作' }
]

// 加载数据
const loadData = async () => {
    loading.value = true
    try {
        const coords = await geolocation.getCoordinates()
        
        const [groupData, settingsData, membersResponse] = await Promise.all([
            groupsService.openTheDoor({
                id: groupId.value,
                latitude: coords.latitude,
                longitude: coords.longitude
            }),
            groupsService.getSettings({ id: groupId.value }),
            groupsService.getMembers({ id: groupId.value })
        ])
        
        group.value = {
            ...groupData,
            settings: settingsData.settings,
            has_password: settingsData.has_password,
            created_at: settingsData.created_at
        } as GroupPageData
        
        members.value = membersResponse.items || []
        groupUnread.value = 0
        page.value = 1
    } catch (e: unknown) {
        toast.add({
            title: "加载失败",
            description: getErrorMessage(e, "无法加载群组信息"),
            color: "error"
        })
    } finally {
        loading.value = false
    }
}

// 进入群聊
const goChat = () => {
    if (userRole.value === 'visitor') return
    groupUnread.value = 0
    navigateTo(`/chat/groups/${groupId.value}`)
}

// 加入群组
const onJoin = async () => {
    try {
        // 如果群组有密码，暂时提示功能开发中
        if (group.value?.has_password) {
            toast.add({ 
                title: "该群组需要密码", 
                description: "密码输入功能开发中",
                color: "warning" 
            })
            return
        }
        
        await groupsService.joinGroup({ id: groupId.value })
        toast.add({ title: "加入成功", color: "success" })
        await loadData()
    } catch (e: unknown) {
        toast.add({
            title: "加入失败",
            description: getErrorMessage(e),
            color: "error"
        })
    }
}

// 退出群组
const onLeave = async () => {
    if (!group.value) return
    
    const confirmed = await modal.showConfirm({
        title: "确认退出",
        message: `确定退出群组"${group.value.name || ''}"吗？`,
        confirmText: "确认退出",
        cancelText: "取消",
        confirmColor: "error"
    })
    
    if (confirmed) {
        try {
            await groupsService.leaveGroup({ id: groupId.value })
            toast.add({ title: "已退出群组", color: "success" })
            await loadData()
        } catch (e: unknown) {
            toast.add({
                title: "操作失败",
                description: getErrorMessage(e),
                color: "error"
            })
        }
    }
}

// 消息免打扰（待实现）
const toggleMute = () => {
    toast.add({ title: "功能开发中", color: "info" })
}

// 邀请成员（待实现）
const showInviteModal = () => {
    toast.add({ title: "功能开发中", color: "info" })
}

// 转让群主（待实现）
const showTransferModal = () => {
    toast.add({ title: "功能开发中", color: "info" })
}

// 解散群组
const showDismissModal = async () => {
    if (!group.value) return
    
    const confirmed = await modal.showConfirm({
        title: "解散群组",
        message: `确定解散群组"${group.value.name || ''}"吗？此操作不可恢复！`,
        confirmText: "确认解散",
        cancelText: "取消",
        confirmColor: "error"
    })
    
    if (confirmed) {
        try {
            await groupsService.deleteGroup({ id: groupId.value })
            toast.add({ title: "群组已解散", color: "success" })
            router.back()
        } catch (e: unknown) {
            toast.add({
                title: "操作失败",
                description: getErrorMessage(e),
                color: "error"
            })
        }
    }
}

// 发送私信
const sendMessage = (member: GroupMemberItem) => {
    if (member.user.user_id === authStore.user?.user_id) {
        navigateTo('/profile')
    } else {
        navigateTo(`/chat/users/${member.user.user_id}`)
    }
}

// 成员描述
const getMemberDescription = (member: GroupMemberItem): string => {
    if (member.muted_until) {
        return `禁言至 ${formatTime(member.muted_until)}`
    }
    return `加入于 ${formatTime(member.joined_at)}`
}

// 成员角色颜色
const getMemberRoleColor = (role: string): string => {
    const colors: Record<string, string> = {
        owner: 'warning',
        admin: 'primary',
        member: 'neutral'
    }
    return colors[role] || 'neutral'
}

// 成员角色标签
const getMemberRoleLabel = (role: string): string => {
    const labels: Record<string, string> = {
        owner: '群主',
        admin: '管理员',
        member: '成员'
    }
    return labels[role] || '成员'
}

// 是否可以管理该成员
const canManageMember = (member: GroupMemberItem): boolean => {
    if (userRole.value === 'visitor') return false
    if (member.role === 'owner') return false
    if (userRole.value === 'owner') return true
    if (userRole.value === 'admin' && member.role === 'member') return true
    return false
}

// 成员操作菜单
const getMemberMenuItems = (member: GroupMemberItem) => {
    const items: Array<Array<{ label: string; icon: string; click: () => void; class?: string }>> = []
    
    if (userRole.value === 'owner') {
        // 群主可以设置/取消管理员
        if (member.role === 'admin') {
            items.push([{
                label: '取消管理员',
                icon: 'i-heroicons-arrow-down',
                click: () => handleRoleChange(member, 'member')
            }])
        } else if (member.role === 'member') {
            items.push([{
                label: '设为管理员',
                icon: 'i-heroicons-arrow-up',
                click: () => handleRoleChange(member, 'admin')
            }])
        }
    }
    
    // 管理员和群主都可以禁言和移除成员
    if (canManageMember(member)) {
        items.push([
            {
                label: member.muted_until ? '取消禁言' : '禁言成员',
                icon: member.muted_until ? 'i-heroicons-speaker-wave' : 'i-heroicons-speaker-x-mark',
                click: () => handleMute(member)
            },
            {
                label: '移出群组',
                icon: 'i-heroicons-user-minus',
                click: () => handleKickMember(member),
                class: 'text-error'
            }
        ])
    }
    
    return items
}

// 角色变更
const handleRoleChange = async (member: GroupMemberItem, newRole: 'admin' | 'member') => {
    try {
        await groupsService.setMemberRole({
            id: groupId.value,
            user_id: member.user.user_id,
            role: newRole
        })
        
        toast.add({
            title: "角色变更成功",
            description: `已将 ${member.user.nickname} 设为${newRole === 'admin' ? '管理员' : '普通成员'}`,
            color: "success"
        })
        await loadData()
    } catch (error: unknown) {
        toast.add({
            title: "角色变更失败",
            description: getErrorMessage(error, "请稍后重试"),
            color: "error"
        })
    }
}

// 禁言/取消禁言
const handleMute = async (member: GroupMemberItem) => {
    try {
        if (member.muted_until) {
            await groupsService.unmuteMember({
                id: groupId.value,
                user_id: member.user.user_id
            })
            toast.add({ title: "已取消禁言", color: "success" })
        } else {
            // 默认禁言 24 小时
            await groupsService.muteMember({
                id: groupId.value,
                user_id: member.user.user_id,
                duration_minutes: 24 * 60
            })
            toast.add({ title: "已禁言 24 小时", color: "success" })
        }
        await loadData()
    } catch (error: unknown) {
        toast.add({
            title: "操作失败",
            description: getErrorMessage(error),
            color: "error"
        })
    }
}

// 移出成员
const handleKickMember = async (member: GroupMemberItem) => {
    const confirmed = await modal.showConfirm({
        title: "确认移出",
        message: `确定将 "${member.user.nickname}" 移出群组吗？`,
        confirmText: "确认移出",
        cancelText: "取消",
        confirmColor: "error"
    })
    
    if (confirmed) {
        try {
            await groupsService.removeMember({
                id: groupId.value,
                user_id: member.user.user_id
            })
            
            toast.add({
                title: "移出成功",
                description: `已将 ${member.user.nickname} 移出群组`,
                color: "success"
            })
            await loadData()
        } catch (error: unknown) {
            toast.add({
                title: "移出失败",
                description: getErrorMessage(error, "请稍后重试"),
                color: "error"
            })
        }
    }
}

onMounted(async () => {
    try {
        await loadData()
    } catch (error: unknown) {
        console.error('[GroupDetails] Failed to load initial data:', error)
    }
})
</script>
