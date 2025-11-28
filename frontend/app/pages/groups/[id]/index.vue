<template>
    <div class="mx-auto max-w-7xl px-4 sm:px-6 lg:px-8 py-8 sm:py-10 space-y-6">
        <UCard>
            <template #header>
                <div class="flex items-center justify-between">
                    <div class="flex items-center gap-3">
                        <UAvatar :alt="group?.name">
                            {{ group?.name?.charAt(0) }}
                        </UAvatar>
                        <div>
                            <h1 class="text-2xl font-bold text-neutral-900 dark:text-neutral-100">
                                {{ group?.name }}
                            </h1>
                            <p class="text-sm text-neutral-600 dark:text-neutral-300">
                                {{ group?.location_name }} ·
                                {{ group?.member_count || 0 }} 人
                            </p>
                        </div>
                    </div>
                    <div class="flex items-center gap-2">
                        <UBadge v-if="groupUnread > 0" size="xs" variant="soft" color="primary">
                            未读 {{ groupUnread }} 条
                        </UBadge>
                        <UTooltip text="进入群聊">
                            <UButton
                                size="lg"
                                icon="i-heroicons-chat-bubble-left-right"
                                aria-label="进入群聊"
                                @click="goChat"
                            />
                        </UTooltip>
                        <UTooltip v-if="group?.is_member" text="退出群组">
                            <UButton
                                size="lg"
                                color="error"
                                variant="outline"
                                icon="i-heroicons-arrow-right-on-rectangle"
                                aria-label="退出群组"
                                @click="onLeave"
                            />
                        </UTooltip>
                        <UTooltip v-else text="加入群组">
                            <UButton
                                size="lg"
                                color="primary"
                                icon="i-heroicons-plus"
                                aria-label="加入群组"
                                @click="onJoin"
                            />
                        </UTooltip>
                        <UTooltip v-if="isOwner" text="管理群组">
                            <UButton
                                size="lg"
                                variant="outline"
                                icon="i-heroicons-cog-6-tooth"
                                aria-label="管理群组"
                                @click="goSettings"
                            />
                        </UTooltip>
                    </div>
                </div>
            </template>

            <div class="space-y-2">
                <p class="text-neutral-700">{{ group?.description }}</p>
                <div class="text-sm text-neutral-500">
                    创建于：{{ formatTime(group?.created_at) }} · 最近活跃：{{
                        formatTime(group?.last_active_at)
                    }}
                </div>
            </div>
        </UCard>

        <!-- 群组详细信息 -->
        <UAccordion :items="groupInfoAccordionItems" multiple>
            <template #members="{ item, open }">
                <div class="flex items-center justify-between w-full">
                    <div class="flex items-center gap-2">
                        <UIcon name="i-heroicons-users" class="w-5 h-5" />
                        <span class="font-medium">成员列表</span>
                        <UBadge v-if="members.length > 0" size="xs" variant="soft">
                            {{ members.length }}
                        </UBadge>
                    </div>
                    <div class="text-sm text-neutral-500">仅展示当前页</div>
                </div>
            </template>
            
            <template #members-content>
                <!-- 成员表格 -->
                <div>
                    <UTable :rows="pagedMembers" :columns="memberColumns">
                        <template #member-cell="{ row }">
                            <UUser
                                :name="(row as any).nickname"
                                :description="getUserDescription(row as any)"
                                :avatar="{
                                    src: (row as any).avatar_url,
                                    alt: (row as any).nickname,
                                    size: 'md',
                                    chip: isOnlineUser((row as any).user_id) ? { color: 'success', position: 'bottom-right' } : undefined
                                }"
                                :chip="(row as any).role ? { text: (row as any).role, color: 'primary' } : false"
                                size="md"
                            />
                        </template>
                        <template #joined-cell="{ row }">
                            <span class="text-sm text-neutral-600 dark:text-neutral-400">{{ formatTime((row as any).joined_at) }}</span>
                        </template>
                        <template #uid-cell="{ row }">
                            <span class="text-xs text-neutral-500">{{ (row as any).user_id }}</span>
                        </template>
                        <template #actions-cell="{ row }">
                            <div class="flex items-center gap-1">
                                <UTooltip text="发送私信">
                                    <UButton
                                        variant="ghost"
                                        size="sm"
                                        icon="i-heroicons-chat-bubble-left"
                                        @click="sendMessage(row as any)"
                                    />
                                </UTooltip>
                                <UDropdownMenu :items="getMemberMenuItems(row as any)">
                                    <UButton
                                        variant="ghost"
                                        size="sm"
                                        icon="i-heroicons-ellipsis-horizontal"
                                    />
                                </UDropdownMenu>
                            </div>
                        </template>
                        <template #empty>
                            <EmptyState
                                icon="i-heroicons-users"
                                title="暂无成员"
                                description="邀请朋友一起加入群组吧"
                                action-text="去管理"
                                @action="goSettings"
                            />
                        </template>
                    </UTable>
                </div>
                <div v-if="pageCount > 1" class="mt-4 flex justify-center">
                    <UPagination v-model:page="page" :items-per-page="pageSize" :total="members.length" />
                </div>
            </template>
            
            <template #activities="{ item, open }">
                <div class="flex items-center gap-2">
                    <UIcon name="i-heroicons-clock" class="w-5 h-5" />
                    <span class="font-medium">群组动态</span>
                </div>
            </template>
            
            <template #activities-content>
                <div class="py-8 text-center text-neutral-500">
                    群组动态功能开发中...
                </div>
            </template>
            
            <template #settings="{ item, open }">
                <div class="flex items-center gap-2">
                    <UIcon name="i-heroicons-cog-6-tooth" class="w-5 h-5" />
                    <span class="font-medium">群组设置</span>
                </div>
            </template>
            
            <template #settings-content>
                <div class="space-y-2">
                    <UButton
                        v-if="isOwner"
                        variant="outline"
                        block
                        justify="start"
                        @click="goSettings"
                    >
                        <UIcon name="i-heroicons-cog-6-tooth" class="w-5 h-5 mr-3" />
                        管理群组
                    </UButton>
                    <UButton
                        variant="outline"
                        block
                        justify="start"
                        @click="goChat"
                    >
                        <UIcon name="i-heroicons-chat-bubble-left-right" class="w-5 h-5 mr-3" />
                        进入群聊
                    </UButton>
                </div>
            </template>
        </UAccordion>
    </div>
</template>

<script setup lang="ts">
import type { TableColumn } from '@nuxt/ui';
import type { GroupResponse } from "~/types/api";

// 公开页面 - 对应 POST /api/v1/public/group_chats/open_the_door
// 注意：某些操作（如加入群组、发消息）仍需登录
definePageMeta({ title: "群组详情", layout: 'dashboard' });

const route = useRoute();
const toast = useToast()
const groupsStore = useGroupsStore();
const modal = useModal();
const { formatTime } = useFormatters();

// 导入服务
import { useGroupsService } from '~/services/groups.service';
const groupsService = useGroupsService();
const groupId = computed(() =>
    String(
        Array.isArray(route.params.id) ? route.params.id[0] : route.params.id,
    ),
);

const group = ref<GroupResponse | null>(null);
const groupUnread = ref<number>(0);
const members = ref<
    Array<{
        user_id: string; // 64位整数转换为字符串
        nickname: string;
        avatar_url?: string;
        role?: "owner" | "admin" | "member";
        joined_at?: string;
    }>
>([]);

// 成员表格列 & 分页
const memberColumns = [
    { key: 'member', label: '成员', class: 'min-w-[200px]' },
    { key: 'joined', label: '加入时间', class: 'hidden sm:table-cell w-48' },
    { key: 'uid', label: 'ID', class: 'hidden md:table-cell w-32' },
    { key: 'actions', label: '操作', class: 'w-24' }
] as any;

const page = ref(1);
const pageSize = ref(10);
const pageCount = computed(() => Math.max(1, Math.ceil(members.value.length / pageSize.value)));
const pagedMembers = computed(() => {
    const start = (page.value - 1) * pageSize.value;
    return members.value.slice(start, start + pageSize.value);
});

const isOwner = computed(() => group.value?.user_role === "owner");

// UAccordion 配置
const groupInfoAccordionItems = computed(() => [
    {
        label: '成员列表',
        icon: 'i-heroicons-users',
        slot: 'members',
        defaultOpen: true,
        content: 'members-content'
    },
    {
        label: '群组动态',
        icon: 'i-heroicons-clock',
        slot: 'activities',
        defaultOpen: false,
        content: 'activities-content'
    },
    {
        label: '群组设置',
        icon: 'i-heroicons-cog-6-tooth',
        slot: 'settings',
        defaultOpen: false,
        content: 'settings-content'
    }
]);

// formatTime 函数已从统一工具函数导入，已内置空值处理

const loadData = async () => {
    try {
        // 获取群组详情
        const g = await groupsService.openTheDoor({ id: groupId.value });
        group.value = g;
        groupsStore.setGroupDetails(groupId.value, g as any);
        
        // 获取群组成员
        const membersResponse = await groupsService.getMembers({ group_chat_id: groupId.value });
        members.value = membersResponse.items.map(m => ({
            user_id: m.user_id,
            nickname: m.nickname,
            avatar_url: m.avatar_url,
            role: m.role,
            joined_at: m.joined_at
        }));
        
        // 获取群聊未读数（临时简化处理）
        groupUnread.value = 0;
        page.value = 1;
    } catch (e: any) {
        toast.add({
            title: "加载失败",
            description: e?.message || "无法加载群组信息",
            color: "error",
        });
    }
};

const goChat = async () => {
    // 进入群聊页面
    groupUnread.value = 0;
    void navigateTo(`/chat/groups/${groupId.value}`);
};

const goSettings = () => {
    void navigateTo(`/groups/${groupId.value}/settings`);
};

const onJoin = async () => {
    try {
        await groupsService.joinGroup({ group_chat_id: groupId.value });
        toast.add({ title: "加入成功", color: "success" });
        await loadData();
    } catch (e: any) {
        toast.add({
            title: "加入失败",
            description: e?.message,
            color: "error",
        });
    }
};

const onLeave = async () => {
    if (!group.value) return;
    modal.showConfirmModal({
        title: "确认退出",
        message: `确定退出群组"${group.value.name || ''}"吗？`,
        confirmText: "确认退出",
        cancelText: "取消",
        confirmColor: "error",
        onConfirm: confirmLeave
    });
};

onMounted(async () => {
    try {
        await loadData();
    } catch (error: any) {
        console.error('[GroupDetails] Failed to load initial data:', error);
        // loadData 内部已经有错误处理，这里主要是防止未捕获的异常
    }
});

// 离开群组确认
const confirmLeave = async () => {
    try {
        await groupsService.leaveGroup({ group_chat_id: groupId.value });
        toast.add({ title: "已退出群组", color: "success" });
        await loadData();
    } catch (e: any) {
        toast.add({
            title: "操作失败",
            description: e?.message,
            color: "error",
        });
    }
};

// 成员相关辅助函数
const getUserDescription = (member: any) => {
    const parts = []
    if (member.joined_at) {
        parts.push(`加入于 ${formatTime(member.joined_at)}`)
    }
    if (member.location_name) {
        parts.push(member.location_name)
    }
    return parts.join(' • ')
}

// 检查用户在线状态（暂时禁用，需要WebSocket支持）
const isOnlineUser = (userId: string): boolean => {
    // TODO: 实现 WebSocket 在线状态检查
    return false
}

// 发送私信
const sendMessage = async (member: any) => {
    try {
        await navigateTo(`/chat/users/${member.user_id}`)
    } catch (error) {
        toast.add({
            title: "跳转失败",
            description: "无法打开私信会话",
            color: "error"
        })
    }
}

// 成员操作菜单
const getMemberMenuItems = (member: any) => {
    const items = []
    
    // 基础操作
    items.push([
        {
            label: '查看资料',
            icon: 'i-heroicons-user',
            click: () => navigateTo(`/users/${member.user_id}`)
        }
    ])
    
    // 管理员操作（如果当前用户是管理员或群主）
    if (isOwner.value || (group.value as any)?.current_user_role === 'admin') {
        const adminActions = []
        
        if (member.role !== 'owner') {
            if (member.role === 'admin') {
                adminActions.push({
                    label: '取消管理员',
                    icon: 'i-heroicons-arrow-down',
                    click: () => handleRoleChange(member, 'member')
                })
            } else {
                adminActions.push({
                    label: '设为管理员',
                    icon: 'i-heroicons-arrow-up',
                    click: () => handleRoleChange(member, 'admin')
                })
            }
            
            adminActions.push({
                label: '移出群组',
                icon: 'i-heroicons-user-minus',
                click: () => handleKickMember(member),
                class: 'text-error-500'
            })
        }
        
        if (adminActions.length > 0) {
            items.push(adminActions)
        }
    }
    
    return items
}

// 角色变更
const handleRoleChange = async (member: any, newRole: 'admin' | 'member') => {
    try {
        await groupsService.setMemberRole({
            group_chat_id: groupId.value,
            user_id: member.user_id,
            role: newRole
        })
        
        toast.add({
            title: "角色变更成功",
            description: `已将 ${member.nickname} 设为${newRole === 'admin' ? '管理员' : '普通成员'}`,
            color: "success"
        })
        await loadData() // 重新加载数据
    } catch (error: any) {
        console.error('角色变更失败:', error)
        toast.add({
            title: "角色变更失败",
            description: error.message || "请稍后重试",
            color: "error"
        })
    }
}

// 移出成员
const handleKickMember = async (member: any) => {
    modal.showConfirmModal({
        title: "确认移出",
        message: `确定将 "${member.nickname}" 移出群组吗？`,
        confirmText: "确认移出",
        cancelText: "取消",
        confirmColor: "error",
        onConfirm: async () => {
            try {
                await groupsService.removeMember({
                    group_chat_id: groupId.value,
                    user_id: member.user_id
                })
                
                toast.add({
                    title: "移出成功",
                    description: `已将 ${member.nickname} 移出群组`,
                    color: "success"
                })
                await loadData()
            } catch (error: any) {
                console.error('移出成员失败:', error)
                toast.add({
                    title: "移出失败",
                    description: error.message || "请稍后重试",
                    color: "error"
                })
            }
        }
    })
};
</script>
