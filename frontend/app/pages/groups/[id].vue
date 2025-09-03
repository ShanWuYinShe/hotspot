<template>
    <div class="mx-auto max-w-7xl px-4 sm:px-6 lg:px-8 py-14 sm:py-16 space-y-6">
        <UCard>
            <template #header>
                <div class="flex items-center justify-between">
                    <div class="flex items-center gap-3">
                        <UAvatar :alt="group?.name">
                            {{ group?.name?.charAt(0) }}
                        </UAvatar>
                        <div>
                            <h1 class="text-xl font-semibold text-neutral-900">
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
                    <UTable :rows="pagedMembers" :columns="memberColumns as any">
                        <template #member-cell="{ row }">
                            <div class="flex items-center gap-3">
                                <UAvatar :alt="(row as any).nickname">{{ (row as any).nickname?.charAt(0) }}</UAvatar>
                                <div class="min-w-0">
                                    <div class="flex items-center gap-2">
                                        <span class="font-medium truncate">{{ (row as any).nickname }}</span>
                                        <UBadge v-if="(row as any).role" variant="soft" size="xs">{{ (row as any).role }}</UBadge>
                                    </div>
                                </div>
                            </div>
                        </template>
                        <template #joined-cell="{ row }">
                            <span class="text-sm text-neutral-600 dark:text-neutral-400">{{ formatTime((row as any).joined_at) }}</span>
                        </template>
                        <template #uid-cell="{ row }">
                            <span class="text-xs text-neutral-500">{{ (row as any).user_id }}</span>
                        </template>
                        <template #empty>
                            <EmptyState
                                icon="i-heroicons-users"
                                title="暂无成员"
                                description="邀请朋友一起加入群组吧"
                                action-text="去管理"
                                :action-handler="goSettings"
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
        <!-- 退出群组确认 Modal 放在页面根容器下 -->
        <UModal v-model:open="showLeaveConfirm" title="确认退出">
            <template #body>
                <p class="text-gray-600 dark:text-gray-300">
                    确定退出群组"{{ group?.name || '' }}"吗？
                </p>
            </template>
            <template #footer>
                <div class="flex justify-end space-x-2">
                    <UButton variant="ghost" @click="showLeaveConfirm = false">取消</UButton>
                    <UButton color="error" @click="confirmLeave">确认退出</UButton>
                </div>
            </template>
        </UModal>
    </div>
</template>

<script setup lang="ts">
import type { TableColumn } from '@nuxt/ui';
import type { GroupInfo } from "~/types";
import { useMessageService } from "~/services/message.service";

definePageMeta({ title: "群组详情", requiresAuth: true });

const route = useRoute();
const toast = useSmartToast();
const groupsStore = useGroupsStore();
const groupId = computed(() =>
    String(
        Array.isArray(route.params.id) ? route.params.id[0] : route.params.id,
    ),
);

const group = ref<GroupInfo | null>(null);
const groupUnread = ref<number>(0);
const members = ref<
    Array<{
        user_id: number;
        nickname: string;
        avatar_url?: string;
        role?: "owner" | "admin" | "member";
        joined_at?: string;
    }>
>([]);

// 成员表格列 & 分页
const memberColumns: TableColumn<any>[] = [
    { id: 'member', key: 'member', label: '成员' },
    { id: 'joined', key: 'joined', label: '加入时间', class: 'hidden sm:table-cell w-48' },
    { id: 'uid', key: 'uid', label: 'ID', class: 'hidden md:table-cell w-32' }
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

const formatTime = (t?: string) => {
    if (!t) return "-";
    const d = new Date(t);
    if (Number.isNaN(d.getTime())) return "-";
    return d.toLocaleString();
};

const loadData = async () => {
    try {
        const g = await groupsStore.fetchGroupDetails(groupId.value);
        group.value = g;
        members.value = await groupsStore.fetchGroupMembers(groupId.value);
        // 获取群聊未读数（/chat/group/unread_count）
        try {
            const messageService = useMessageService();
            const resp: any = await messageService.getGroupUnreadCount(groupId.value);
            groupUnread.value = Number(resp?.unread_count ?? resp?.total_unread_count ?? 0);
        } catch (_) {
            groupUnread.value = 0;
        }
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
    // 进入群聊前，标记该群为已读
    try {
        const service = useMessageService();
        await service.markAsRead(groupId.value, 'group');
        groupUnread.value = 0;
    } catch {}
    void navigateTo(`/chat/group/${groupId.value}`);
};
const goSettings = () => {
    void navigateTo(`/groups/${groupId.value}/settings`);
};

const onJoin = async () => {
    try {
        await groupsStore.joinGroup(groupId.value);
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
    showLeaveConfirm.value = true;
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
const showLeaveConfirm = ref(false);
const confirmLeave = async () => {
    try {
        await groupsStore.leaveGroup(groupId.value);
        toast.add({ title: "已退出群组", color: "success" });
        await loadData();
    } catch (e: any) {
        toast.add({
            title: "操作失败",
            description: e?.message,
            color: "error",
        });
    } finally {
        showLeaveConfirm.value = false;
    }
};
</script>
