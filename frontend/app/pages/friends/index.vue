<template>
    <div class="mx-auto max-w-7xl px-4 sm:px-6 lg:px-8 py-8 sm:py-10">
        <!-- 页面标题和操作栏 -->
        <div
            class="flex flex-col gap-4 sm:flex-row sm:justify-between sm:items-center mb-6"
        >
            <div>
                <h1 class="text-3xl font-bold text-neutral-900 dark:text-white">
                    好友
                </h1>
                <p class="text-neutral-600 dark:text-neutral-300">
                    管理您的好友关系
                </p>
            </div>

            <div class="flex gap-3">
                <UButton @click="navigateTo('/users/search')">
                    <UIcon name="i-heroicons-user-plus" class="w-4 h-4 mr-2" />
                    添加好友
                </UButton>
                <UButton
                    variant="outline"
                    @click="navigateTo('/friends/requests')"
                >
                    <UIcon name="i-heroicons-bell" class="w-4 h-4 mr-2" />
                    好友请求
                    <UBadge
                        v-if="pendingRequestsCount > 0"
                        color="warning"
                        size="xs"
                        class="ml-2"
                    >
                        {{ pendingRequestsCount }}
                    </UBadge>
                </UButton>
            </div>
        </div>

        <!-- 搜索和筛选 -->
        <UCard class="mb-4">
            <div class="flex gap-4">
                <UInput
                    v-model="searchQuery"
                    placeholder="搜索好友..."
                    icon="i-heroicons-magnifying-glass"
                    class="flex-1"
                />
                <USelect
                    v-model="sortBy"
                    :options="sortOptions"
                    placeholder="排序方式"
                    class="w-40"
                />
            </div>
        </UCard>

        <!-- 列表加载骨架屏 -->
        <div v-if="loading && friends.length === 0" class="space-y-4">
            <USkeleton class="h-20" />
            <USkeleton class="h-20" />
            <USkeleton class="h-20" />
        </div>

        <!-- 好友表格（UTable） -->
        <div v-else>
            <div class="w-full">
                <UTable :rows="pagedFriends" :columns="columns">
                    <template #friend-cell="{ row }">
                        <div class="flex items-center gap-3 min-w-0">
                            <div class="relative shrink-0">
                                <UAvatar
                                    :alt="(row as any).nickname"
                                    size="lg"
                                    :src="(row as any).avatar_url"
                                >
                                    {{ (row as any).nickname?.charAt(0) }}
                                </UAvatar>
                                <div
                                    v-if="(row as any).is_online"
                                    class="absolute bottom-0 right-0 w-3.5 h-3.5 bg-green-500 rounded-full border-2 border-white"
                                />
                            </div>
                            <div class="min-w-0">
                                <div class="flex items-center gap-2">
                                    <span class="font-medium truncate">{{
                                        (row as any).display_name ||
                                        (row as any).nickname
                                    }}</span>
                                    <UBadge
                                        v-if="(row as any).is_online"
                                        color="success"
                                        size="xs"
                                        >在线</UBadge
                                    >
                                </div>
                                <div
                                    v-if="
                                        (row as any).remark &&
                                        (row as any).remark !==
                                            (row as any).nickname
                                    "
                                    class="text-xs text-neutral-500 truncate"
                                >
                                    备注: {{ (row as any).remark }}
                                </div>
                            </div>
                        </div>
                    </template>

                    <template #status-cell="{ row }">
                        <span
                            class="text-sm text-neutral-600 dark:text-neutral-400 hidden sm:inline"
                            >{{
                                (row as any).is_online ? "在线" : "离线"
                            }}</span
                        >
                    </template>

                    <template #last-cell="{ row }">
                        <div
                            class="text-sm text-neutral-600 dark:text-neutral-400 max-w-[320px] hidden md:block"
                        >
                            <div
                                v-if="(row as any).last_message"
                                class="flex items-center gap-2 truncate"
                            >
                                <UIcon
                                    name="i-heroicons-chat-bubble-left"
                                    class="w-4 h-4"
                                />
                                <span class="truncate">{{
                                    (row as any).last_message.content
                                }}</span>
                                <span class="text-xs">{{
                                    formatTime(
                                        (row as any).last_message.created_at,
                                    )
                                }}</span>
                            </div>
                            <span v-else class="text-neutral-400">—</span>
                        </div>
                    </template>

                    <template #location-cell="{ row }">
                        <div
                            class="text-sm text-neutral-600 dark:text-neutral-400 hidden lg:block"
                        >
                            <div
                                class="flex items-center gap-1 truncate max-w-[240px]"
                            >
                                <UIcon
                                    name="i-heroicons-map-pin"
                                    class="w-4 h-4"
                                />
                                <span class="truncate">{{
                                    (row as any).location_name || "—"
                                }}</span>
                            </div>
                        </div>
                    </template>

                    <template #since-cell="{ row }">
                        <span
                            class="text-sm text-neutral-600 dark:text-neutral-400 hidden md:inline"
                            >{{
                                formatFriendTime((row as any).friend_since)
                            }}</span
                        >
                    </template>

                    <template #actions-cell="{ row }">
                        <div class="flex items-center justify-end gap-2">
                            <UButton
                                size="sm"
                                @click.stop="
                                    $router.push(
                                        `/chat/users/${(row as any).user_id}`,
                                    )
                                "
                            >
                                <UIcon
                                    name="i-heroicons-chat-bubble-left-right"
                                    class="w-4 h-4 mr-1"
                                />
                                聊天
                            </UButton>
                            <UDropdownMenu
                                :items="getFriendActionItems(row as any)"
                            >
                                <UButton
                                    icon="i-heroicons-ellipsis-horizontal"
                                    variant="ghost"
                                    size="sm"
                                    @click.stop
                                />
                            </UDropdownMenu>
                        </div>
                    </template>

                    <template #empty>
                        <div class="py-10">
                            <EmptyState
                                icon="i-heroicons-user-group"
                                title="还没有好友"
                                description="添加好友开始聊天吧"
                                action-text="查找用户"
                                :action-handler="
                                    () => navigateTo('/users/search')
                                "
                            />
                        </div>
                    </template>
                </UTable>
            </div>

            <div v-if="pageCount > 1" class="flex justify-center mt-6">
                <UPagination v-model:page="page" :page-count="pageCount" />
            </div>
        </div>

    </div>
</template>

<script setup lang="ts">
import type { UserInfo } from "~/types";
// import { buildFriendUrl, FRIEND_ENDPOINTS } from "~/config/api"; // 好友功能暂未实现
import { formatTime } from '~/utils/formatters';

definePageMeta({
    title: "我的好友",
    requiresAuth: true,
});

const api = useApi();
const toast = useSmartToast();
const authStore = useAuthStore();
const { openConfirmModal, openFriendRemarkModal } = useModals();

// 状态变量
const loading = ref(true);
const searchQuery = ref("");
const sortBy = ref("name");
const friends = ref<
    (UserInfo & {
        friend_since?: string;
        remark?: string;
        display_name?: string;
        is_online?: boolean;
        avatar_url?: string;
        location_name?: string;
        last_message?: {
            content: string;
            created_at: string;
        };
    })[]
>([]);

// UTable 列（断言为 any 以适配变体类）
const columns = [
    { id: "friend", key: "friend", label: "好友" },
    { id: "status", key: "status", label: "状态", class: "w-24" },
    { id: "last", key: "last", label: "最近消息", class: "min-w-[260px]" },
    { id: "location", key: "location", label: "位置", class: "min-w-[200px]" },
    { id: "since", key: "since", label: "成为好友", class: "w-40" },
    { id: "actions", key: "actions", label: "操作", class: "w-40 text-right" },
] as any;

// 好友请求相关
const pendingRequestsCount = ref(0);

// 备注设置相关
const selectedFriend = ref<UserInfo | null>(null);

// 本地分页（仅使用本地分页）
const page = ref(1);
const pageSize = ref(10);
const pageCount = computed(() =>
    Math.max(1, Math.ceil(displayFriends.value.length / pageSize.value)),
);
const pagedFriends = computed(() => {
    const start = (page.value - 1) * pageSize.value;
    return displayFriends.value.slice(start, start + pageSize.value);
});

// 排序选项
const sortOptions = [
    { label: "按姓名", value: "name" },
    { label: "按添加时间", value: "time" },
    { label: "按最近聊天", value: "recent" },
];

// 计算属性
const displayFriends = computed(() => {
    let filtered = friends.value;

    // 搜索筛选
    if (searchQuery.value.trim()) {
        const query = searchQuery.value.toLowerCase();
        filtered = filtered.filter(
            (friend) =>
                (friend.display_name || friend.nickname)
                    ?.toLowerCase()
                    .includes(query) ||
                friend.remark?.toLowerCase().includes(query),
        );
    }

    // 排序
    return filtered.sort((a, b) => {
        switch (sortBy.value) {
            case "name":
                return (a.display_name || a.nickname || "").localeCompare(
                    b.display_name || b.nickname || "",
                );
            case "time":
                return (
                    new Date(b.friend_since || 0).getTime() -
                    new Date(a.friend_since || 0).getTime()
                );
            case "recent":
                const aTime = a.last_message?.created_at || "0";
                const bTime = b.last_message?.created_at || "0";
                return new Date(bTime).getTime() - new Date(aTime).getTime();
            default:
                return 0;
        }
    });
});

// 生命周期
onMounted(async () => {
    try {
        await loadFriends();
        await loadPendingRequests();
    } catch (error: any) {
        console.error("[Friends] Failed to load initial data:", error);
        toast.add({
            title: "好友信息加载失败",
            description: error?.message || "无法获取好友信息，请稍后重试",
            color: "error",
        });
    }
});

// 加载好友列表
async function loadFriends() {
    loading.value = true;
    try {
        // 好友功能暂未实现，显示提示信息
        throw new Error('好友功能暂未实现，请等待后端API上线');
    } catch (e: any) {
        toast.add({
            title: "加载失败",
            description: e?.message || "无法加载好友列表",
            color: "error",
        });
    } finally {
        loading.value = false;
    }
}

// 加载待处理的好友请求数量
async function loadPendingRequests() {
    try {
        // 好友功能暂未实现，返回默认值
        const data = { count: 0 };
        pendingRequestsCount.value = data?.count || 0;
    } catch (e: any) {
        console.error("加载待处理请求数量失败:", e);
        pendingRequestsCount.value = 0;
    }
}

// 好友操作
function viewFriendProfile(friend: UserInfo) {
    navigateTo(`/users/${friend.user_id}`);
}

function getFriendActionItems(friend: UserInfo) {
    return [
        [
            {
                label: "设置备注",
                icon: "i-heroicons-pencil",
                click: () => openRemarkModal(friend),
            },
            {
                label: "查看资料",
                icon: "i-heroicons-user",
                click: () => viewFriendProfile(friend),
            },
        ],
        [
            {
                label: "删除好友",
                icon: "i-heroicons-user-minus",
                click: () => removeFriend(friend),
            },
        ],
    ];
}

// 备注管理
function openRemarkModal(friend: UserInfo) {
    selectedFriend.value = friend;
    openFriendRemarkModal({
        friend: friend,
        onSave: saveRemark
    });
}

async function saveRemark(remarkText: string) {
    if (!selectedFriend.value) return;

    try {
        const friend = friends.value.find(
            (f) => f.user_id === selectedFriend.value?.user_id,
        );
        if (friend) {
            friend.remark = remarkText.trim() || undefined;
            friend.display_name = friend.remark || friend.nickname;
        }
        toast.add({ title: "备注已保存", color: "success" });
    } catch (e: any) {
        toast.add({
            title: "保存失败",
            description: e?.message,
            color: "error",
        });
        throw e; // 重新抛出错误，让 Modal 处理
    }
}

async function removeFriend(friend: UserInfo) {
    openConfirmModal({
        title: "确认删除好友",
        message: "确定要删除该好友吗？",
        confirmText: "确认删除",
        cancelText: "取消",
        confirmColor: "error",
        onConfirm: () => doRemoveFriend(friend)
    });
}

async function doRemoveFriend(friend: UserInfo) {
    try {
        const index = friends.value.findIndex(
            (f) => f.user_id === friend.user_id,
        );
        if (index > -1) friends.value.splice(index, 1);
        toast.add({ title: "已删除好友", color: "info" });
    } catch (e: any) {
        toast.add({
            title: "删除失败",
            description: e?.message,
            color: "error",
        });
    }
}

// formatTime 函数已从统一工具函数导入

function formatFriendTime(dateStr?: string): string {
    if (!dateStr) return "";
    const date = new Date(dateStr);
    return `${date.getFullYear()}年${date.getMonth() + 1}月`;
}
</script>
