<template>
    <div class="mx-auto max-w-7xl px-4 sm:px-6 lg:px-8 py-8 sm:py-10">
        <UCard class="mb-4">
            <template #header>
                <div class="flex justify-between items-center">
                    <h1 class="text-3xl font-bold text-primary-900 dark:text-primary-100">
                        我的群组
                    </h1>
                    <UButton 
                        size="lg" 
                        color="primary" 
                        @click="() => openCreateGroup()"
                    >
                        <UIcon name="i-heroicons-plus" />
                        创建群组
                    </UButton>
                </div>
            </template>

            <!-- 搜索与筛选 -->
            <div class="flex flex-col sm:flex-row gap-3 sm:gap-4 mb-6">
                <UInput
                    v-model="searchQuery"
                    placeholder="搜索群组..."
                    size="md"
                    color="primary"
                    class="flex-1 w-full"
                >
                    <template #leading>
                        <UIcon name="i-heroicons-magnifying-glass" class="text-primary-500" />
                    </template>
                </UInput>
                <USelect
                    v-model="activeFilter"
                    :items="filterOptions"
                    placeholder="筛选"
                    color="primary"
                    class="w-full sm:w-48"
                />
            </div>

            <!-- 列表 → UTable 风格 -->
            <div v-if="filteredGroups.length > 0">
                <UTable :rows="filteredGroups" :columns="columns">
                    <template #group-cell="{ row }">
                        <div class="flex items-center gap-3 min-w-[220px]">
                            <UAvatar
                                :alt="(row as any).name || '群组'"
                                size="lg"
                                color="primary"
                                >{{
                                    (row as any).name?.charAt(0) || "群"
                                }}</UAvatar
                            >
                            <div class="min-w-0">
                                <div class="flex items-center gap-2">
                                    <span class="font-medium truncate text-primary-900 dark:text-primary-100">{{
                                        (row as any).name || "未命名群组"
                                    }}</span>
                                    <UTooltip
                                        v-if="(row as any).user_role === 'owner'"
                                        text="您是该群组的群主，拥有所有管理权限"
                                    >
                                        <UBadge
                                            color="primary"
                                            variant="soft"
                                            size="xs"
                                        >群主</UBadge>
                                    </UTooltip>
                                    <UTooltip
                                        v-else-if="(row as any).user_role === 'admin'"
                                        text="您是该群组的管理员，拥有部分管理权限"
                                    >
                                        <UBadge
                                            color="success"
                                            variant="soft"
                                            size="xs"
                                        >管理员</UBadge>
                                    </UTooltip>
                                    <UTooltip
                                        v-if="getUnreadCount(row as any) > 0"
                                        :text="`该群组有 ${getUnreadCount(row as any)} 条未读消息`"
                                    >
                                        <UBadge
                                            color="warning"
                                            variant="soft"
                                            size="xs"
                                        >未读 {{ getUnreadCount(row as any) }}</UBadge>
                                    </UTooltip>
                                </div>
                                <p class="text-xs text-neutral-600 dark:text-neutral-400 truncate">
                                    {{
                                        (row as any).description ||
                                        "暂无群组描述"
                                    }}
                                </p>
                            </div>
                        </div>
                    </template>
                    <template #members-cell="{ row }">
                        <UBadge 
                            color="neutral" 
                            variant="soft" 
                            size="sm"
                            class="flex items-center gap-1"
                        >
                            <UIcon name="i-heroicons-users" class="w-3 h-3" />
                            {{ (row as any).member_count }} 人
                        </UBadge>
                    </template>
                    <template #activity-cell="{ row }">
                        <span
                            class="text-sm text-neutral-600 dark:text-neutral-400"
                            >{{
                                (row as any).last_active_at
                                    ? formatTime((row as any).last_active_at)
                                    : "—"
                            }}</span
                        >
                    </template>
                    <template #actions-cell="{ row }">
                        <div class="flex items-center justify-end gap-2">
                            <UButton 
                                size="sm" 
                                color="primary" 
                                variant="soft"
                                @click="enterGroup(row as any)"
                            >
                                <UIcon
                                    name="i-heroicons-chat-bubble-left-right"
                                    class="w-4 h-4 mr-1"
                                />进入
                            </UButton>
                            <UDropdownMenu :items="getGroupActions(row as any)">
                                <UButton
                                    variant="ghost"
                                    size="sm"
                                    color="neutral"
                                    icon="i-heroicons-ellipsis-vertical"
                                    @click.stop
                                />
                            </UDropdownMenu>
                        </div>
                    </template>
                    <template #empty>
                        <div class="py-10">
                            <EmptyState
                                icon="i-heroicons-user-group"
                                title="还没有群组"
                                description="加入或创建您的第一个群组开始聊天吧"
                                action-text="创建群组"
                                action-color="primary"
                                :action-handler="() => openCreateGroup()"
                            >
                                <template #illustration>
                                    <EmptyGroups size="lg" />
                                </template>
                            </EmptyState>
                        </div>
                    </template>
                </UTable>
            </div>

            <!-- 空状态 -->
            <EmptyState
                v-else
                icon="i-heroicons-user-group"
                title="还没有群组"
                description="加入或创建您的第一个群组开始聊天吧"
                action-text="创建群组"
                action-color="primary"
                :action-handler="() => openCreateGroup()"
            >
                <template #illustration>
                    <EmptyGroups size="lg" />
                </template>
            </EmptyState>
        </UCard>
        <!-- 确认弹窗已改为 useModals.openConfirmModal 程序化管理 -->
    </div>
</template>

<script setup lang="ts">
import { defineAsyncComponent } from "vue";
definePageMeta({ title: "我的群组", requiresAuth: true, layout: 'dashboard' });

const EmptyGroups = defineAsyncComponent(
    () => import("~/components/illustrations/EmptyGroups.vue"),
);
const groupsStore = useGroupsStore();
const toast = useSmartToast();
const { openConfirmModal, openCreateGroupModal } = useModals();

const searchQuery = ref("");
const activeFilter = ref<"all" | "owned" | "admin" | "active">("all");

const filterOptions = [
    { label: "全部群组", value: "all" },
    { label: "我创建的", value: "owned" },
    { label: "我管理的", value: "admin" },
    { label: "最近活跃", value: "active" },
];

import type { GroupInfo } from "~/types";
const filteredGroups = computed<GroupInfo[]>(() => {
    let result = groupsStore.myGroups;
    
    // 按筛选类型过滤
    switch (activeFilter.value) {
        case "owned":
            result = groupsStore.getOwnedGroups();
            break;
        case "admin":
            result = result.filter((g) => g.user_role === "admin");
            break;
        case "active":
            const oneDayAgo = new Date(Date.now() - 86400000);
            result = result.filter((g) => new Date(g.last_active_at) > oneDayAgo);
            break;
        default:
            // "all" - 保持原始结果
            break;
    }
    
    // 按搜索关键词过滤
    const query = searchQuery.value.trim();
    if (query) {
        const lowerQuery = query.toLowerCase();
        result = result.filter(
            (g) =>
                (g.name || "").toLowerCase().includes(lowerQuery) ||
                (g.description || "").toLowerCase().includes(lowerQuery),
        );
    }
    
    return result;
});

// 群未读本地映射
const groupUnreadMap = reactive<Record<string, number>>({});

// UTable 列配置
interface TableColumn {
    id: string;
    key: string;
    label: string;
    class?: string;
}

const columns: TableColumn[] = [
    { 
        id: "group", 
        key: "group", 
        label: "群组" 
    },
    {
        id: "members",
        key: "members",
        label: "成员数",
        class: "hidden sm:table-cell w-32",
    },
    {
        id: "activity",
        key: "activity",
        label: "最近活跃",
        class: "hidden md:table-cell w-40",
    },
    { 
        id: "actions", 
        key: "actions", 
        label: "操作", 
        class: "w-32 text-right" 
    },
];

// 加载群组未读数
const loadGroupUnreadCounts = async (groups: GroupInfo[]) => {
    const messageService = useMessageService();
    const limitedGroups = groups.slice(0, 30); // 限制前30个以控制流量
    
    await Promise.allSettled(
        limitedGroups.map(async (group) => {
            try {
                const result = await messageService.getGroupUnreadCount(String(group.group_id));
                groupUnreadMap[String(group.group_id)] = Number(
                    result?.unread_count ?? result?.total_unread_count ?? 0,
                );
            } catch (error) {
                // 静默处理单个群组的未读数加载失败
                if (import.meta.dev) {
                    console.warn(`加载群组 ${group.group_id} 未读数失败:`, error);
                }
            }
        })
    );
};

onMounted(async () => {
    try {
        await groupsStore.fetchMyGroups();
        
        // 懒加载未读数
        await loadGroupUnreadCounts(groupsStore.myGroups);
        
        // 如果当前筛选是"我创建的"，预加载相关数据
        if (activeFilter.value === "owned") {
            await groupsStore.getOwnedGroups();
        }
    } catch (error: any) {
        toast.add({
            title: "加载失败",
            description: error?.message || "无法加载群组",
            color: "error",
        });
    }
});

// 监听筛选器变化，按需加载数据
watch(activeFilter, async (newFilter) => {
    if (newFilter === "owned" && groupsStore.getOwnedGroups().length === 0) {
        try {
            await groupsStore.getOwnedGroups();
        } catch (error) {
            if (import.meta.dev) {
                console.warn("加载我创建的群组失败:", error);
            }
        }
    }
});

// 获取群组未读数
const getUnreadCount = (group: any): number => {
    if (!group?.group_id) return 0;
    return groupUnreadMap[String(group.group_id)] || 0;
};

// 格式化时间显示
const formatTime = (timeString: string): string => {
    const targetTime = new Date(timeString);
    const now = Date.now();
    const diffMs = now - targetTime.getTime();
    
    const minutes = Math.floor(diffMs / 60000);
    const hours = Math.floor(diffMs / 3600000);
    const days = Math.floor(diffMs / 86400000);
    
    if (minutes < 60) return `${minutes}分钟前`;
    if (hours < 24) return `${hours}小时前`;
    return `${days}天前`;
};

// 进入群组聊天
const enterGroup = async (group: GroupInfo) => {
    try {
        // 进入群聊前标记为已读并清零本地未读
        const messageService = useMessageService();
        await messageService.markAsRead(String(group.group_id), "group");
        groupUnreadMap[String(group.group_id)] = 0;
    } catch (error) {
        if (import.meta.dev) {
            console.warn("标记群组消息已读失败:", error);
        }
    }
    
    await navigateTo(`/chat/group/${group.group_id}`);
};

// 管理群组
const manageGroup = async (group: GroupInfo) => {
    await navigateTo(`/groups/${group.group_id}/settings`);
};

// 退出群组
const leaveGroup = (group: GroupInfo) => {
    openConfirmModal({
        title: "确认退出",
        message: `确定要退出群组"${group?.name || ""}"吗？`,
        confirmText: "确认退出",
        cancelText: "取消",
        confirmColor: "error",
        onConfirm: async () => {
            try {
                await groupsStore.leaveGroup(group.group_id);
                toast.success("已退出群组");
            } catch (error: any) {
                toast.error({
                    title: "退出失败",
                    description: error?.message || "请稍后重试"
                });
            }
        },
    });
};
// 获取群组操作菜单
const getGroupActions = (group: GroupInfo) => {
    const baseActions = [
        [
            {
                label: "进入群聊",
                icon: "i-heroicons-chat-bubble-left-right",
                click: () => enterGroup(group),
            },
        ]
    ];

    // 如果是群主，添加管理选项
    if (group.user_role === "owner") {
        baseActions.push([
            {
                label: "管理群组",
                icon: "i-heroicons-cog-6-tooth",
                click: () => manageGroup(group),
            },
        ]);
    }

    // 添加退出群组选项
    baseActions.push([
        {
            label: "退出群组",
            icon: "i-heroicons-arrow-right-on-rectangle",
            click: async () => {
                leaveGroup(group);
            },
        },
    ]);

    return baseActions;
};

// 触发创建群组弹窗
const openCreateGroup = () => {
    openCreateGroupModal({
        onSuccess: async (group: any) => {
            try {
                // 刷新群组列表
                await groupsStore.fetchMyGroups();
                
                // 跳转到新创建的群组聊天
                await navigateTo(`/chat/group/${group.group_id}`);
                
                toast.success({
                    title: "群组创建成功",
                    description: `欢迎来到"${group.name}"群组！`
                });
            } catch (error: any) {
                toast.error({
                    title: "跳转失败",
                    description: error?.message || "群组已创建，请手动进入"
                });
            }
        },
        onCancel: () => {
            // 用户取消创建，无需处理
        }
    });
};
</script>
