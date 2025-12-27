<template>
    <div class="w-full">
        <!-- 群组详情（仅当数据加载完成后显示） -->
        <UCard v-if="group">
            <template #header>
                <div class="flex items-center justify-between gap-4">
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
                                <h1 class="text-2xl font-bold tracking-tight">
                                    {{ group.name }}
                                </h1>
                                <UBadge v-if="group.has_password" variant="soft" color="warning" size="xs">
                                    <span class="mr-1">
                                        <UIcon name="i-lucide-lock" :size="14" />
                                    </span>
                                    需密码
                                </UBadge>
                            </div>
                            <div class="flex flex-wrap items-center gap-2">
                                <UBadge
                                  color="neutral"
                                  variant="soft"
                                  size="xs"
                                  icon="i-lucide-map-pin"
                                  class="max-w-full truncate"
                                >
                                  {{ group.geo?.location_name || '未知位置' }}
                                </UBadge>
                                <UBadge
                                  color="neutral"
                                  variant="soft"
                                  size="xs"
                                  icon="i-lucide-users"
                                >
                                  {{ group.members_count || 0 }} 人
                                </UBadge>
                                <UBadge
                                  v-if="group.geo?.distance !== undefined"
                                  color="neutral"
                                  variant="soft"
                                  size="xs"
                                  icon="i-lucide-signal"
                                >
                                  {{ formatDistance(group.geo?.distance ?? 0) }}
                                </UBadge>
                            </div>
                            <!-- 我的角色标签 -->
                            <div v-if="userRole !== 'visitor'" class="flex items-center gap-2 mt-1">
                                <UBadge :color="roleConfig.color" variant="soft" size="xs">
                                    <span class="mr-1">
                                        <UIcon :name="roleConfig.icon" :size="14" />
                                    </span>
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
                                icon="i-lucide-messages-square"
                                @click="goChat"
                            >
                                群聊
                            </UButton>
                        </UTooltip>
                        
                        <!-- 访客：加入群组 -->
                        <UButton
                            v-if="userRole === 'visitor'"
                            color="primary"
                            icon="i-lucide-plus"
                            @click="onJoin"
                        >
                            加入群组
                        </UButton>
                        
                        <!-- 成员：更多操作菜单 -->
                        <UDropdownMenu v-else :items="actionMenuItems">
                            <UButton
                                variant="outline"
                                icon="i-lucide-ellipsis"
                            />
                        </UDropdownMenu>
                    </div>
                </div>
            </template>

            <!-- 群组描述 -->
            <div class="space-y-4">
                <p v-if="group.description" class="font-bold tracking-tight">
                    {{ group.description }}
                </p>
                <UBadge v-else color="neutral" variant="soft" size="sm">暂无群组介绍</UBadge>
                
                <!-- 标签 -->
                <div v-if="group.tags?.length" class="flex flex-wrap gap-2">
                    <UBadge
                        v-for="tag in group.tags"
                        :key="tag"
                        variant="soft"
                        color="neutral"
                        size="sm"
                    >
                        #{{ tag }}
                    </UBadge>
                </div>
                
                <!-- 时间信息 -->
                <USeparator class="mt-2" />
                <div class="flex flex-wrap gap-2 pt-2">
                    <UBadge v-if="group.created_at" color="neutral" variant="soft" size="xs" icon="i-lucide-calendar">
                      创建于 {{ formatTime(group.created_at) }}
                    </UBadge>
                    <UBadge v-if="group.last_activity_at" color="neutral" variant="soft" size="xs" icon="i-lucide-clock">
                      最近活跃 {{ formatTime(group.last_activity_at) }}
                    </UBadge>
                </div>
            </div>
        </UCard>

        <!-- 群组不存在（加载完成后仍无数据时显示） -->
        <UCard v-else-if="!loading">
            <UEmpty
                icon="i-lucide-users-round"
                title="群组不存在"
                description="该群组可能已被解散或不存在"
            >
                <template #actions>
                    <UButton to="/groups">返回群组列表</UButton>
                    <UButton color="primary" to="/groups/nearby">去附近群组</UButton>
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
                                icon="i-lucide-user-plus"
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
                            <UBadge color="neutral" variant="soft" size="xs">
                              {{ formatTime(row.original.joined_at) }}
                            </UBadge>
                        </template>
                        <template #actions-cell="{ row }">
                            <div class="flex items-center gap-1">
                                    <UTooltip text="查看资料">
                                    <UButton
                                        variant="ghost"
                                        size="xs"
                                        icon="i-lucide-user"
                                        @click="row.original.user.user_id === currentUserId ? navigateTo('/profile') : navigateTo(`/users/${row.original.user.user_id}`)"
                                    />
                                </UTooltip>
                                <UTooltip text="发送私信">
                                    <UButton
                                        variant="ghost"
                                        size="xs"
                                        icon="i-lucide-message-circle"
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
                                        icon="i-lucide-ellipsis"
                                    />
                                </UDropdownMenu>
                            </div>
                        </template>
                        <template #empty>
                            <UEmpty
                                icon="i-lucide-users"
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

            <!-- 群组设置（仅管理员和创建者可见） -->
            <template v-if="canManageGroup" #settings>
                <UCard class="mt-4">
                    <div class="space-y-4">
                        <div class="grid gap-4 sm:grid-cols-2">
                            <UButton
                                variant="outline"
                                :block="true"
                                class="justify-start"
                                icon="i-lucide-square-pen"
                                @click="navigateTo(`/groups/${groupId}/settings`)"
                            >
                                编辑群组资料
                            </UButton>
                            <UButton
                                variant="outline"
                                :block="true"
                                class="justify-start"
                                icon="i-lucide-users"
                                @click="navigateTo(`/groups/${groupId}/settings?tab=members`)"
                            >
                                成员管理
                            </UButton>
                            <UButton
                                v-if="userRole === 'owner'"
                                variant="outline"
                                :block="true"
                                class="justify-start"
                                icon="i-lucide-circle-arrow-right"
                                @click="showTransferModal"
                            >
                                转让群主
                            </UButton>
                            <UButton
                                v-if="userRole === 'owner'"
                                variant="outline"
                                color="error"
                                :block="true"
                                class="justify-start"
                                icon="i-lucide-trash-2"
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
import { useDashboardHeader } from '~/composables/useDashboardHeader'

// 用户角色类型
type UserRole = 'owner' | 'admin' | 'member' | 'visitor'

// 成员行类型
type MemberRow = GroupMemberItem

definePageMeta({
  title: "群组详情",
  description: "查看群组信息与成员",
  layout: 'dashboard',
  dashboardHeader: true
})

const dashboardHeader = useDashboardHeader()
const router = useRouter()

watchEffect(() => {
  dashboardHeader.setActions([
    {
      key: 'back',
      label: '返回',
      icon: 'i-lucide-arrow-left',
      color: 'neutral',
      variant: 'outline',
      action: () => router.back()
    }
  ])
})

const route = useRoute()
const feedback = useFeedbackStore()
const groupsService = useGroupsService()
const groupsStore = useGroupsStore()
const geolocation = useGeolocation()
const modal = useModal()
const { formatTime, formatDistance } = useFormatters()

const groupId = computed(() =>
    String(Array.isArray(route.params.id) ? route.params.id[0] : route.params.id)
)

// 状态
const loading = ref(true)
const group = ref<GroupDetailData | null>(null)
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

// 当前用户（水合安全）
const authStore = useAuthStore()
const currentUserId = ref<number | undefined>(undefined)

onMounted(() => {
    currentUserId.value = authStore.user?.user_id
    watch(() => authStore.user?.user_id, (userId) => {
        currentUserId.value = userId
    })
})

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
        owner: { label: '群主', color: 'warning', icon: 'i-lucide-crown' },
        admin: { label: '管理员', color: 'primary', icon: 'i-lucide-shield-check' },
        member: { label: '成员', color: 'neutral', icon: 'i-lucide-user' },
        visitor: { label: '访客', color: 'neutral', icon: 'i-lucide-eye' }
    }
    return configs[userRole.value]
})

// 权限判断
const canManageGroup = computed(() => ['owner', 'admin'].includes(userRole.value))
const canInvite = computed(() => userRole.value !== 'visitor')

// Tab 配置
const tabItems = computed(() => {
    const items = [
        { label: '成员', slot: 'members', icon: 'i-lucide-users' }
    ]
    if (canManageGroup.value) {
        items.push({ label: '设置', slot: 'settings', icon: 'i-lucide-settings' })
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
                icon: 'i-lucide-user-plus',
                click: showInviteModal
            }
        ])
    }
    
    // 管理员操作
    if (canManageGroup.value) {
        items.push([
            {
                label: '群组设置',
                icon: 'i-lucide-settings',
                click: () => navigateTo(`/groups/${groupId.value}/settings`)
            },
            {
                label: '成员管理',
                icon: 'i-lucide-users',
                click: () => navigateTo(`/groups/${groupId.value}/settings?tab=members`)
            }
        ])
    }
    
    // 群主特有操作
    if (userRole.value === 'owner') {
        items.push([
            {
                label: '转让群主',
                icon: 'i-lucide-circle-arrow-right',
                click: showTransferModal
            },
            {
                label: '解散群组',
                icon: 'i-lucide-trash-2',
                click: showDismissModal,
                color: 'error'
            }
        ])
    }
    
    // 退出群组（非群主可见）
    if (userRole.value !== 'visitor' && userRole.value !== 'owner') {
        items.push([
            {
                label: '退出群组',
                icon: 'i-lucide-log-out',
                click: onLeave,
                color: 'error'
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
        // 1. 尝试从 store 获取缓存数据
        const cachedGroup = groupsStore.getGroup(groupId.value)
        if (cachedGroup) {
            // 使用部分数据初始化，created_at 和 settings 暂时为空
            group.value = {
                ...cachedGroup,
                created_at: '',
                settings: {},
                geo: cachedGroup.geo // 确保 geo 类型匹配
            } as GroupDetailData
            loading.value = false // 有缓存先显示
        }

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
        
        // 更新 store
        groupsStore.setGroup(groupData)

        group.value = {
            ...groupData,
            settings: settingsData.settings,
            has_password: settingsData.has_password,
            created_at: settingsData.created_at
        } as GroupDetailData
        
        members.value = membersResponse.items || []
        groupUnread.value = 0
        page.value = 1
    } catch (e: unknown) {
        feedback.error("加载失败", getErrorMessage(e, "无法加载群组信息"))
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
        if (group.value?.has_password) {
            const password = await modal.showGroupPassword(group.value?.name)
            if (!password) return
            await groupsService.joinGroup({ id: groupId.value, password })
        } else {
            await groupsService.joinGroup({ id: groupId.value })
        }
        
        feedback.success("加入成功")
        await loadData()
    } catch (e: unknown) {
        feedback.error("加入失败", getErrorMessage(e))
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
            feedback.success("已退出群组")
            await loadData()
        } catch (e: unknown) {
            feedback.error("操作失败", getErrorMessage(e))
        }
    }
}

// 邀请成员（待实现）
const showInviteModal = async () => {
    await modal.showInviteMembers(groupId.value)
}

// 转让群主（待实现）
const showTransferModal = async () => {
    const newOwnerId = await modal.showTransferOwnership(
        groupId.value,
        members.value.filter(m => m.role !== 'owner')
    )
    if (newOwnerId) {
        await loadData()
    }
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
            feedback.success("群组已解散")
            router.back()
        } catch (e: unknown) {
            feedback.error("操作失败", getErrorMessage(e))
        }
    }
}

// 发送私信
const sendMessage = (member: GroupMemberItem) => {
    if (member.user.user_id === currentUserId.value) {
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
                icon: 'i-lucide-arrow-down',
                click: () => handleRoleChange(member, 'member')
            }])
        } else if (member.role === 'member') {
            items.push([{
                label: '设为管理员',
                icon: 'i-lucide-arrow-up',
                click: () => handleRoleChange(member, 'admin')
            }])
        }
    }
    
    // 管理员和群主都可以禁言和移除成员
    if (canManageMember(member)) {
        items.push([
            {
                label: member.muted_until ? '取消禁言' : '禁言成员',
                icon: member.muted_until ? 'i-lucide-volume-2' : 'i-lucide-volume-x',
                click: () => handleMute(member)
            },
            {
                label: '移出群组',
                icon: 'i-lucide-user-minus',
                click: () => handleKickMember(member),
                color: 'error'
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
        
        feedback.success(
            "角色变更成功",
            `已将 ${member.user.nickname} 设为${newRole === 'admin' ? '管理员' : '普通成员'}`
        )
        await loadData()
    } catch (error: unknown) {
        feedback.error("角色变更失败", getErrorMessage(error, "请稍后重试"))
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
            feedback.success("已取消禁言")
        } else {
            // 默认禁言 24 小时
            await groupsService.muteMember({
                id: groupId.value,
                user_id: member.user.user_id,
                duration_minutes: 24 * 60
            })
            feedback.success("已禁言 24 小时")
        }
        await loadData()
    } catch (error: unknown) {
        feedback.error("操作失败", getErrorMessage(error))
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
            
            feedback.success("移出成功", `已将 ${member.user.nickname} 移出群组`)
            await loadData()
        } catch (error: unknown) {
            feedback.error("移出失败", getErrorMessage(error, "请稍后重试"))
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
