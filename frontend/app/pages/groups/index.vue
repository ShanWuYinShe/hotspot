<template>
    <div class="mx-auto max-w-7xl px-4 sm:px-6 lg:px-8 py-8 sm:py-10">
        <UCard class="mb-4">
            <template #header>
                <div class="flex justify-between items-center">
                    <h1 class="text-3xl font-bold text-neutral-900">我的群组</h1>
                    <CreateGroupModal @success="handleCreateGroupSuccess">
                        <UButton size="lg">
                            <UIcon name="i-heroicons-plus" />
                            创建群组
                        </UButton>
                    </CreateGroupModal>
                </div>
            </template>

            <!-- 搜索与筛选 -->
            <div class="flex flex-col sm:flex-row gap-3 sm:gap-4 mb-6">
                <UInput
                    v-model="searchQuery"
                    placeholder="搜索群组..."
                    size="md"
                    class="flex-1 w-full"
                >
                    <template #leading>
                        <UIcon name="i-heroicons-magnifying-glass" />
                    </template>
                </UInput>
                <USelect
                    v-model="activeFilter"
                    :items="filterOptions"
                    placeholder="筛选"
                    class="w-full sm:w-48"
                />
            </div>

            <!-- 列表 → UTable 风格 -->
            <div v-if="filteredGroups.length > 0">
                <UTable :rows="filteredGroups" :columns="columns as any">
                        <template #group-cell="{ row }">
                            <div class="flex items-center gap-3 min-w-[220px]">
                                <UAvatar :alt="(row as any).name || '群组'" size="lg">{{ (row as any).name?.charAt(0) || '群' }}</UAvatar>
                                <div class="min-w-0">
                                    <div class="flex items-center gap-2">
                                        <span class="font-medium truncate">{{ (row as any).name || '未命名群组' }}</span>
                                        <UBadge v-if="(row as any).user_role === 'owner'" color="primary" variant="soft" size="xs">群主</UBadge>
                                        <UBadge v-else-if="(row as any).user_role === 'admin'" color="success" variant="soft" size="xs">管理员</UBadge>
                                        <UBadge v-if="groupUnreadMap[(row as any).group_id] > 0" variant="soft" size="xs">未读 {{ groupUnreadMap[(row as any).group_id] }}</UBadge>
                                    </div>
                                    <p class="text-xs text-neutral-500 truncate">{{ (row as any).description || '暂无群组描述' }}</p>
                                </div>
                            </div>
                        </template>
                        <template #members-cell="{ row }">
                            <span class="text-sm text-neutral-600 dark:text-neutral-400 flex items-center gap-1">
                                <UIcon name="i-heroicons-users" class="w-4 h-4" />
                                {{ (row as any).member_count }} 人
                            </span>
                        </template>
                        <template #activity-cell="{ row }">
                            <span class="text-sm text-neutral-600 dark:text-neutral-400">{{ (row as any).last_active_at ? formatTime((row as any).last_active_at) : '—' }}</span>
                        </template>
                        <template #actions-cell="{ row }">
                            <div class="flex items-center justify-end gap-2">
                                <UButton size="sm" @click="enterGroup(row as any)">
                                    <UIcon name="i-heroicons-chat-bubble-left-right" class="w-4 h-4 mr-1" />进入
                                </UButton>
                                <UDropdownMenu :items="getGroupActions(row as any)">
                                    <UButton variant="ghost" size="sm" icon="i-heroicons-ellipsis-vertical" @click.stop />
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
                :action-handler="() => openCreateGroup()"
            >
                <template #illustration>
                    <EmptyGroups size="lg" />
                </template>
            </EmptyState>
        </UCard>
        <!-- 退出群组确认 -->
        <UModal v-model:open="showLeave" title="确认退出">
            <template #body>
                <p class="text-gray-600 dark:text-gray-300">
                    确定要退出群组"{{ leavingGroup?.name || '' }}"吗？
                </p>
            </template>
            <template #footer>
                <div class="flex justify-end space-x-2">
                    <UButton variant="ghost" @click="showLeave = false">取消</UButton>
                    <UButton color="error" @click="confirmLeave">确认退出</UButton>
                </div>
            </template>
        </UModal>
    </div>
</template>

<script setup lang="ts">
import { defineAsyncComponent } from 'vue'
definePageMeta({ title: "我的群组", requiresAuth: true });

import CreateGroupModal from "~/components/CreateGroupModal.vue";
const EmptyGroups = defineAsyncComponent(() => import("~/components/illustrations/EmptyGroups.vue"));
const groupsStore = useGroupsStore();
const toast = useSmartToast();

const searchQuery = ref("");
const activeFilter = ref<"all" | "owned" | "admin" | "active">("all");

const filterOptions = [
    { label: "全部群组", value: "all" },
    { label: "我创建的", value: "owned" },
    { label: "我管理的", value: "admin" },
    { label: "最近活跃", value: "active" },
];

import type { GroupInfo } from "~/types";
import { useMessageService } from "~/services/message.service";
const filteredGroups = computed<GroupInfo[]>(() => {
    let result = groupsStore.myGroups;
    if (activeFilter.value === "owned") {
        result = groupsStore.ownedGroups;
    } else if (activeFilter.value === "admin") {
        result = result.filter((g) => g.user_role === "admin");
    } else if (activeFilter.value === "active") {
        const oneDayAgo = new Date(Date.now() - 86400000);
        result = result.filter((g) => new Date(g.last_active_at) > oneDayAgo);
    }
    if (searchQuery.value.trim()) {
        const q = searchQuery.value.toLowerCase();
        result = result.filter(
            (g) =>
                (g.name || "").toLowerCase().includes(q) ||
                (g.description || "").toLowerCase().includes(q),
        );
    }
    return result;
});

// 群未读本地映射
const groupUnreadMap = reactive<Record<number, number>>({});

// UTable 列
const columns = [
  { id: 'group', key: 'group', label: '群组' },
  { id: 'members', key: 'members', label: '成员数', class: 'hidden sm:table-cell w-32' },
  { id: 'activity', key: 'activity', label: '最近活跃', class: 'hidden md:table-cell w-40' },
  { id: 'actions', key: 'actions', label: '操作', class: 'w-32 text-right' }
] as any;

onMounted(async () => {
    try {
        await groupsStore.fetchMyGroups();
    } catch (e: any) {
        toast.add({
            title: "加载失败",
            description: e?.message || "无法加载群组",
            color: "error",
        });
    }
    // 懒加载未读数（合并请求）
    try {
        const ms = useMessageService();
        // 逐个请求，后端若提供批量接口可替换
        const groups = groupsStore.myGroups.slice(0, 30); // 限前30个以控流
        await Promise.all(groups.map(async (g) => {
            try {
                const r: any = await ms.getGroupUnreadCount(String(g.group_id));
                groupUnreadMap[g.group_id] = Number(r?.unread_count ?? r?.total_unread_count ?? 0);
            } catch {}
        }));
    } catch {}
    if (activeFilter.value === "owned") {
        try {
            await groupsStore.fetchOwnedGroups();
        } catch {}
    }
});

watch(activeFilter, async (val) => {
    if (val === "owned" && groupsStore.ownedGroups.length === 0) {
        try {
            await groupsStore.fetchOwnedGroups();
        } catch {}
    }
});

const formatTime = (s: string) => {
    const t = new Date(s);
    const n = Date.now();
    const d = n - t.getTime();
    const m = Math.floor(d / 60000);
    const h = Math.floor(d / 3600000);
    const day = Math.floor(d / 86400000);
    if (m < 60) return `${m}分钟前`;
    if (h < 24) return `${h}小时前`;
    return `${day}天前`;
};

const enterGroup = (g: GroupInfo) => {
    // 进入群聊前标记为已读并清零本地未读
    try {
        const ms = useMessageService();
        void ms.markAsRead(String(g.group_id), 'group');
        groupUnreadMap[g.group_id] = 0;
    } catch {}
    navigateTo(`/chat/group/${g.group_id}`);
};
const manageGroup = (g: GroupInfo) => {
    navigateTo(`/groups/${g.group_id}/settings`);
};
const showLeave = ref(false);
const leavingGroup = ref<GroupInfo | null>(null);
const leaveGroup = (g: GroupInfo) => {
    leavingGroup.value = g;
    showLeave.value = true;
};
const confirmLeave = async () => {
    if (!leavingGroup.value) return;
    try {
        await groupsStore.leaveGroup(leavingGroup.value.group_id);
        toast.success("已退出");
        leavingGroup.value = null;
    } finally {
        showLeave.value = false;
    }
};
const getGroupActions = (g: GroupInfo) => [
    [
        {
            label: "进入群聊",
            icon: "i-heroicons-chat-bubble-left-right",
            click: () => enterGroup(g),
        },
    ],
    ...(g.user_role === "owner"
        ? [
              [
                  {
                      label: "管理群组",
                      icon: "i-heroicons-cog-6-tooth",
                      click: () => manageGroup(g),
                  },
              ],
          ]
        : []),
    [
        {
            label: "退出群组",
            icon: "i-heroicons-arrow-right-on-rectangle",
            click: () => leaveGroup(g),
        },
    ],
];

const handleCreateGroupSuccess = (g: any) => {
    groupsStore
        .fetchMyGroups()
        .finally(() => navigateTo(`/chat/group/${g.group_id}`));
};

// 触发创建群组弹窗（通过触发器引用）
const openCreateGroup = () => {
    // 简化处理：直接跳转到发现页作为替代入口
    navigateTo("/groups/discover");
};
</script>
