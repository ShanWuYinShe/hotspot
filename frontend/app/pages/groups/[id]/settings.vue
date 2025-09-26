<template>
    <div
        class="mx-auto max-w-7xl px-4 sm:px-6 lg:px-8 py-14 sm:py-16 space-y-6"
    >
        <UCard>
            <template #header>
                <h1 class="text-3xl font-bold text-neutral-900">群组管理</h1>
            </template>

            <div class="space-y-6">
                <!-- 邀请新成员 -->
                <div class="space-y-2">
                    <div class="flex items-center justify-between">
                        <h2 class="font-medium">邀请新成员</h2>
                        <UButton
                            size="sm"
                            @click="showInviteModal = true"
                            :disabled="!canInvite"
                        >
                            <UIcon name="i-heroicons-user-plus" class="mr-1" />
                            邀请成员
                        </UButton>
                    </div>

                    <!-- 邀请码分享 -->
                    <div
                        class="bg-neutral-50 dark:bg-neutral-900 rounded-lg p-4"
                    >
                        <div class="flex items-center justify-between">
                            <div>
                                <h3 class="font-medium text-sm">群组邀请码</h3>
                                <p class="text-xs text-neutral-500 mt-1">
                                    分享此邀请码让朋友快速加入群组
                                </p>
                            </div>
                            <UButton
                                size="sm"
                                variant="outline"
                                @click="generateInviteCode"
                            >
                                <UIcon name="i-heroicons-link" class="mr-1" />
                                生成邀请码
                            </UButton>
                        </div>
                        <div
                            v-if="inviteCode"
                            class="mt-3 flex items-center gap-2"
                        >
                            <UInput
                                v-model="inviteCode"
                                readonly
                                size="sm"
                                class="flex-1"
                            />
                            <UButton size="sm" @click="copyInviteCode">
                                <UIcon name="i-heroicons-clipboard" />
                            </UButton>
                        </div>
                    </div>
                </div>

                <div class="space-y-2">
                    <h2 class="font-medium">成员管理</h2>

                    <!-- 成员表格 -->
                    <div>
                        <UTable :rows="pagedMembers" :columns="columns">
                            <template #member-cell="{ row }">
                                <div class="flex items-center gap-3 px-1">
                                    <UAvatar :alt="(row as any).nickname">{{
                                        (row as any).nickname?.charAt(0)
                                    }}</UAvatar>
                                    <div class="min-w-0">
                                        <div class="flex items-center gap-2">
                                            <span
                                                class="font-medium truncate"
                                                >{{
                                                    (row as any).nickname
                                                }}</span
                                            >
                                            <UBadge
                                                v-if="(row as any).role"
                                                variant="soft"
                                                size="xs"
                                                >{{ (row as any).role }}</UBadge
                                            >
                                        </div>
                                        <div class="text-xs text-neutral-500">
                                            ID: {{ (row as any).user_id }}
                                        </div>
                                    </div>
                                </div>
                            </template>
                            <template #actions-cell="{ row }">
                                <div
                                    class="flex items-center gap-2 justify-end"
                                >
                                    <UButton
                                        v-if="
                                            isOwner &&
                                            (row as any).role !== 'owner'
                                        "
                                        size="sm"
                                        variant="outline"
                                        color="warning"
                                        @click="
                                            openTransfer((row as any).user_id)
                                        "
                                        >设为群主</UButton
                                    >
                                    <UButton
                                        v-if="
                                            (isOwner || isAdmin) &&
                                            (row as any).role !== 'owner'
                                        "
                                        size="sm"
                                        color="error"
                                        variant="outline"
                                        @click="openKick((row as any).user_id)"
                                        >踢出</UButton
                                    >
                                </div>
                            </template>
                            <template #empty>
                                <EmptyState
                                    icon="i-heroicons-users"
                                    title="暂无成员"
                                    description="邀请朋友加入你的群组"
                                    action-text="邀请"
                                    :action-handler="
                                        () => (showInviteModal = true)
                                    "
                                />
                            </template>
                        </UTable>
                    </div>
                    <div v-if="pageCount > 1" class="mt-4 flex justify-center">
                        <UPagination
                            v-model:page="page"
                            :items-per-page="pageSize"
                            :total="members.length"
                        />
                    </div>
                </div>
            </div>
        </UCard>



    </div>
</template>

<!-- 合并到同一个 <script setup> 内，保持单一 <template> 规则 -->
<script setup lang="ts">
// TableColumn 类型已在组件内部定义
import type { GroupResponse } from "~/types/api";

definePageMeta({ title: "群组管理", requiresAuth: true, layout: 'dashboard' });

const route = useRoute();
const toast = useSmartToast();
const groupsStore = useGroupsStore();
const authStore = useAuthStore();
const groupService = useGroupService();
const userService = useUserService();
const groupId = computed(() =>
    String(
        Array.isArray(route.params.id) ? route.params.id[0] : route.params.id,
    ),
);

const group = ref<GroupResponse | null>(null);
const members = ref<
    Array<{
        user_id: string;
        nickname: string;
        role?: "owner" | "admin" | "member";
        joined_at?: string;
    }>
>([]);

// 表格列定义
interface TableColumn {
    id: string;
    key: string;
    label: string;
    class?: string;
}

// 表格列 & 分页
const columns: TableColumn[] = [
    { id: "member", key: "member", label: "成员" },
    { id: "actions", key: "actions", label: "操作", class: "w-48 text-right" },
];

const page = ref(1);
const pageSize = ref(10);
const pageCount = computed(() =>
    Math.max(1, Math.ceil(members.value.length / pageSize.value)),
);
const pagedMembers = computed(() => {
    const start = (page.value - 1) * pageSize.value;
    return members.value.slice(start, start + pageSize.value);
});

const isOwner = computed(() => group.value?.user_role === "owner");
const isAdmin = computed(
    () =>
        group.value?.user_role === "admin" ||
        group.value?.user_role === "owner",
);

const loadData = async () => {
    try {
        const g = await groupsStore.fetchGroupDetails(groupId.value);
        group.value = g;
        members.value = await groupsStore.fetchGroupMembers(groupId.value);
        page.value = 1;
    } catch (e: any) {
        toast.add({
            title: "加载失败",
            description: e?.message || "无法加载群组信息",
            color: "error",
        });
    }
};

const showKick = ref(false);
const showTransfer = ref(false);
const targetUserId = ref<string | null>(null);

// 邀请功能相关
const showInviteModal = ref(false);
const inviteCode = ref("");
const inviteSearchQuery = ref("");
const searchedUsers = ref<any[]>([]);
const invitingUserId = ref<string | null>(null);

// 权限检查
const canInvite = computed(() => isOwner.value || isAdmin.value);

const openKick = (uid: string) => {
    targetUserId.value = uid;
    showKick.value = true;
};
const openTransfer = (uid: string) => {
    targetUserId.value = uid;
    showTransfer.value = true;
};

const doKick = async () => {
    if (!targetUserId.value) return;
    try {
        await groupsStore.kickMember(groupId.value, targetUserId.value);
        toast.add({ title: "已移出成员", color: "success" });
        await loadData();
    } catch (e: any) {
        toast.add({
            title: "操作失败",
            description: e?.message,
            color: "error",
        });
    } finally {
        showKick.value = false;
        targetUserId.value = null;
    }
};

const doTransfer = async () => {
    if (!targetUserId.value) return;
    try {
        await groupsStore.transferOwnership(groupId.value, targetUserId.value);
        toast.add({ title: "已转让群主", color: "success" });
        await loadData();
    } catch (e: any) {
        toast.add({
            title: "操作失败",
            description: e?.message,
            color: "error",
        });
    } finally {
        showTransfer.value = false;
        targetUserId.value = null;
    }
};

// 邀请功能方法
const generateInviteCode = async () => {
    try {
        inviteCode.value = `HT_${groupId.value}_${Date.now()}`;
        toast.add({ title: "邀请码已生成", color: "success" });
    } catch (error: any) {
        toast.add({
            title: "生成失败",
            description: error?.message || "无法生成邀请码",
            color: "error",
        });
    }
};

const copyInviteCode = async () => {
    try {
        await navigator.clipboard.writeText(inviteCode.value);
        toast.add({ title: "邀请码已复制", color: "success" });
    } catch (error) {
        toast.add({ title: "复制失败", color: "error" });
    }
};

const searchUsers = async () => {
    if (!inviteSearchQuery.value.trim()) {
        searchedUsers.value = [];
        return;
    }
    try {
        const resp = await userService.searchUsers(inviteSearchQuery.value);
        searchedUsers.value = resp?.items || [];
    } catch (error: any) {
        console.error("搜索用户失败:", error);
        searchedUsers.value = [];
        toast.add({
            title: "搜索失败",
            description: error?.message || "无法搜索用户",
            color: "error",
        });
    }
};

const inviteUser = async (userId: string) => {
    invitingUserId.value = userId;
    try {
        await groupService.inviteUser(groupId.value, userId);
        toast.add({ title: "邀请已发送", color: "success" });
        searchedUsers.value = searchedUsers.value.filter(
            (u) => u.user_id !== userId,
        );
    } catch (error: any) {
        toast.add({
            title: "邀请失败",
            description: error?.message || "无法发送邀请",
            color: "error",
        });
    } finally {
        invitingUserId.value = null;
    }
};

const shareViaLink = () => {
    const inviteLink = `${window.location.origin}/groups/${groupId.value}/join?code=${inviteCode.value}`;
    navigator.clipboard
        .writeText(inviteLink)
        .then(() => {
            toast.add({ title: "邀请链接已复制", color: "success" });
        })
        .catch(() => {
            toast.add({ title: "复制失败", color: "error" });
        });
};

const shareViaQR = () => {
    toast.add({ title: "二维码功能开发中", color: "info" });
};

onMounted(async () => {
    try {
        await loadData();
    } catch (error: any) {
        console.error("[GroupSettings] Failed to load initial data:", error);
    }
});
</script>
