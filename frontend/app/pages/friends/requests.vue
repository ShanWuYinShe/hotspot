<template>
    <div class="mx-auto max-w-7xl px-4 sm:px-6 lg:px-8 py-8 sm:py-10">
        <!-- 页面标题 -->
        <div class="mb-8">
            <div class="flex items-center gap-3 mb-2">
                <UButton
                    variant="ghost"
                    icon="i-heroicons-arrow-left"
                    @click="$router.back()"
                />
                <h1 class="text-3xl font-bold text-neutral-900 dark:text-white">
                    {{ $t("friends.requests.title") }}
                </h1>
            </div>
            <p class="text-neutral-600 dark:text-neutral-300">
                {{ $t("friends.requests.subtitle") }}
            </p>
        </div>

        <!-- 标签页：改用 UTabs -->
        <div class="mb-6">
            <UTabs :items="tabItems" v-model="activeTab" />
        </div>

        <!-- 收到的请求 -->
        <div v-if="activeTab === 'received'">
            <!-- 加载中 -->
            <div v-if="loadingReceived" class="space-y-4">
                <USkeleton class="h-20" />
                <USkeleton class="h-20" />
                <USkeleton class="h-20" />
            </div>

            <!-- 表格列表 -->
            <div v-else>
                <UTable :rows="pagedReceived" :columns="columnsReceived as any">
                    <template #avatar_name-cell="{ row }">
                        <div class="flex items-center gap-3 min-w-[220px]">
                            <UAvatar
                                :alt="(row as any).nickname"
                                size="lg"
                                :src="(row as any).avatar_url"
                            >
                                {{ ((row as any).nickname || "").charAt(0) }}
                            </UAvatar>
                            <div class="min-w-0">
                                <div class="flex items-center gap-2 mb-1">
                                    <h3 class="font-semibold truncate">
                                        {{ (row as any).nickname }}
                                    </h3>
                                    <UBadge color="warning" size="xs">{{
                                        $t("friends.requests.status.pending")
                                    }}</UBadge>
                                </div>
                                <div
                                    class="space-y-1 text-sm text-neutral-600 dark:text-neutral-400"
                                >
                                    <div
                                        v-if="(row as any).bio"
                                        class="truncate"
                                    >
                                        {{ (row as any).bio }}
                                    </div>
                                    <div class="flex items-center gap-4">
                                        <span
                                            v-if="(row as any).location_name"
                                            class="flex items-center gap-1"
                                        >
                                            <UIcon
                                                name="i-heroicons-map-pin"
                                                class="w-3 h-3"
                                            />
                                            {{ (row as any).location_name }}
                                        </span>
                                        <span class="flex items-center gap-1">
                                            <UIcon
                                                name="i-heroicons-clock"
                                                class="w-3 h-3"
                                            />
                                            {{
                                                formatTime(
                                                    (row as any).request_time,
                                                )
                                            }}{{
                                                $t(
                                                    "friends.requests.sent_request",
                                                )
                                            }}
                                        </span>
                                    </div>
                                    <div
                                        v-if="(row as any).message"
                                        class="p-2 bg-neutral-100 dark:bg-neutral-800 rounded text-sm"
                                    >
                                        {{ (row as any).message }}
                                    </div>
                                </div>
                            </div>
                        </div>
                    </template>
                    <template #actions-cell="{ row }">
                        <div class="flex gap-2 justify-end">
                            <UButton
                                size="sm"
                                @click="acceptRequest(row as any)"
                                :loading="(row as any).processing"
                            >
                                <UIcon
                                    name="i-heroicons-check"
                                    class="w-4 h-4 mr-1"
                                />
                                {{ $t("friends.requests.actions.accept") }}
                            </UButton>
                            <UButton
                                size="sm"
                                variant="outline"
                                @click="rejectRequest(row as any)"
                                :loading="(row as any).processing"
                            >
                                <UIcon
                                    name="i-heroicons-x-mark"
                                    class="w-4 h-4 mr-1"
                                />
                                {{ $t("friends.requests.actions.reject") }}
                            </UButton>
                            <UButton
                                size="sm"
                                variant="ghost"
                                @click="
                                    navigateTo(`/users/${(row as any).user_id}`)
                                "
                            >
                                <UIcon
                                    name="i-heroicons-user"
                                    class="w-4 h-4 mr-1"
                                />
                                {{ $t("friends.requests.actions.view") }}
                            </UButton>
                        </div>
                    </template>
                    <template #empty>
                        <EmptyState
                            icon="i-heroicons-inbox"
                            :title="$t('friends.requests.empty.received.title')"
                            :description="
                                $t(
                                    'friends.requests.empty.received.description',
                                )
                            "
                            :action-text="
                                $t('friends.requests.empty.received.actionText')
                            "
                            :action-handler="() => navigateTo('/users/search')"
                        />
                    </template>
                </UTable>
                <div
                    v-if="receivedPageCount > 1"
                    class="mt-4 flex justify-center"
                >
                    <UPagination
                        v-model:page="receivedPage"
                        :items-per-page="pageSize"
                        :total="receivedRequests.length"
                    />
                </div>
            </div>
        </div>

        <!-- 发送的请求 -->
        <div v-else>
            <!-- 加载中 -->
            <div v-if="loadingSent" class="space-y-4">
                <USkeleton class="h-20" />
                <USkeleton class="h-20" />
                <USkeleton class="h-20" />
            </div>

            <!-- 表格列表 -->
            <div v-else>
                <UTable :rows="pagedSent" :columns="columnsSent as any">
                    <template #avatar_name-cell="{ row }">
                        <div class="flex items-center gap-3 min-w-[220px]">
                            <UAvatar
                                :alt="(row as any).nickname"
                                size="lg"
                                :src="(row as any).avatar_url"
                            >
                                {{ ((row as any).nickname || "").charAt(0) }}
                            </UAvatar>
                            <div class="min-w-0">
                                <div class="flex items-center gap-2 mb-1">
                                    <h3 class="font-semibold truncate">
                                        {{ (row as any).nickname }}
                                    </h3>
                                    <UBadge
                                        :color="
                                            getRequestStatusChipColor(
                                                (row as any).status,
                                            )
                                        "
                                        size="xs"
                                    >
                                        {{
                                            getRequestStatusText(
                                                (row as any).status,
                                            )
                                        }}
                                    </UBadge>
                                </div>
                                <div
                                    class="space-y-1 text-sm text-neutral-600 dark:text-neutral-400"
                                >
                                    <div
                                        v-if="(row as any).bio"
                                        class="truncate"
                                    >
                                        {{ (row as any).bio }}
                                    </div>
                                    <div class="flex items-center gap-4">
                                        <span
                                            v-if="(row as any).location_name"
                                            class="flex items-center gap-1"
                                        >
                                            <UIcon
                                                name="i-heroicons-map-pin"
                                                class="w-3 h-3"
                                            />
                                            {{ (row as any).location_name }}
                                        </span>
                                        <span class="flex items-center gap-1">
                                            <UIcon
                                                name="i-heroicons-clock"
                                                class="w-3 h-3"
                                            />
                                            {{
                                                formatTime(
                                                    (row as any).request_time,
                                                )
                                            }}{{ $t("friends.requests.sent") }}
                                        </span>
                                        <span
                                            v-if="(row as any).response_time"
                                            class="flex items-center gap-1"
                                        >
                                            <UIcon
                                                name="i-heroicons-check-circle"
                                                class="w-3 h-3"
                                            />
                                            {{
                                                formatTime(
                                                    (row as any).response_time,
                                                )
                                            }}{{
                                                $t("friends.requests.responded")
                                            }}
                                        </span>
                                    </div>
                                    <div
                                        v-if="(row as any).message"
                                        class="p-2 bg-blue-50 dark:bg-blue-950/30 rounded text-sm"
                                    >
                                        <span
                                            class="text-blue-600 dark:text-blue-400 text-xs"
                                            >{{
                                                $t("friends.requests.i_said")
                                            }}</span
                                        >
                                        {{ (row as any).message }}
                                    </div>
                                </div>
                            </div>
                        </div>
                    </template>
                    <template #actions-cell="{ row }">
                        <div class="flex gap-2 justify-end">
                            <UButton
                                v-if="(row as any).status === 'pending'"
                                size="sm"
                                variant="outline"
                                @click="cancelRequest(row as any)"
                                :loading="(row as any).processing"
                            >
                                <UIcon
                                    name="i-heroicons-x-mark"
                                    class="w-4 h-4 mr-1"
                                />
                                {{ $t("friends.requests.actions.cancel") }}
                            </UButton>
                            <UButton
                                v-else-if="(row as any).status === 'accepted'"
                                size="sm"
                                @click="
                                    $router.push(
                                        `/chat/user/${(row as any).user_id}`,
                                    )
                                "
                            >
                                <UIcon
                                    name="i-heroicons-chat-bubble-left-right"
                                    class="w-4 h-4 mr-1"
                                />
                                {{ $t("friends.requests.actions.chat") }}
                            </UButton>
                            <UButton
                                size="sm"
                                variant="ghost"
                                @click="
                                    navigateTo(`/users/${(row as any).user_id}`)
                                "
                            >
                                <UIcon
                                    name="i-heroicons-user"
                                    class="w-4 h-4 mr-1"
                                />
                                {{ $t("friends.requests.actions.view") }}
                            </UButton>
                        </div>
                    </template>
                    <template #empty>
                        <EmptyState
                            icon="i-heroicons-paper-airplane"
                            :title="$t('friends.requests.empty.sent.title')"
                            :description="
                                $t('friends.requests.empty.sent.description')
                            "
                            :action-text="
                                $t('friends.requests.empty.sent.actionText')
                            "
                            :action-handler="() => navigateTo('/users/search')"
                        />
                    </template>
                </UTable>
                <div v-if="sentPageCount > 1" class="mt-4 flex justify-center">
                    <UPagination
                        v-model:page="sentPage"
                        :items-per-page="pageSize"
                        :total="sentRequests.length"
                    />
                </div>
            </div>
        </div>

        <!-- 批量操作（仅收到的请求） -->
        <div
            v-if="activeTab === 'received' && receivedRequests.length > 0"
            class="mt-6 text-center"
        >
            <UButton
                variant="outline"
                @click="batchAcceptAll"
                :loading="batchProcessing"
                class="mr-3"
            >
                <UIcon name="i-heroicons-check-circle" class="w-4 h-4 mr-1" />
                {{ $t("friends.requests.actions.acceptAll") }}
            </UButton>
            <UButton
                variant="outline"
                @click="batchRejectAll"
                :loading="batchProcessing"
            >
                <UIcon name="i-heroicons-x-circle" class="w-4 h-4 mr-1" />
                {{ $t("friends.requests.actions.rejectAll") }}
            </UButton>
        </div>
    </div>
    <!-- 确认弹窗 -->
    <UModal
        v-model:open="confirmOpen"
        :title="
            pendingConfirm?.type === 'cancel'
                ? $t('friends.requests.modal.title.cancel')
                : pendingConfirm?.type === 'batch_accept'
                  ? $t('friends.requests.modal.title.batch_accept')
                  : $t('friends.requests.modal.title.batch_reject')
        "
    >
        <template #body>
            <p class="text-neutral-600 dark:text-neutral-300">
                {{
                    pendingConfirm?.type === "cancel"
                        ? $t("friends.requests.modal.body.cancel")
                        : pendingConfirm?.type === "batch_accept"
                          ? $t("friends.requests.modal.body.batch_accept")
                          : $t("friends.requests.modal.body.batch_reject")
                }}
            </p>
        </template>
        <template #footer>
            <div class="flex justify-end gap-2">
                <UButton variant="ghost" @click="confirmOpen = false">{{
                    $t("common.cancel")
                }}</UButton>
                <UButton
                    color="warning"
                    v-if="pendingConfirm?.type === 'cancel'"
                    @click="onConfirmCancel"
                    >{{
                        $t("friends.requests.modal.buttons.confirmCancel")
                    }}</UButton
                >
                <UButton
                    color="primary"
                    v-else-if="pendingConfirm?.type === 'batch_accept'"
                    @click="onConfirmBatchAccept"
                    >{{
                        $t("friends.requests.modal.buttons.confirmAccept")
                    }}</UButton
                >
                <UButton color="error" v-else @click="onConfirmBatchReject">{{
                    $t("friends.requests.modal.buttons.confirmReject")
                }}</UButton>
            </div>
        </template>
    </UModal>
</template>

<script setup lang="ts">
import type { TableColumn } from "@nuxt/ui";
import type { UserInfo } from "~/types";

import { buildFriendUrl, FRIEND_ENDPOINTS } from "~/config/api";

interface FriendRequest extends UserInfo {
    request_time: string;
    response_time?: string;
    message?: string;
    status: "pending" | "accepted" | "rejected";
    processing?: boolean;
}

definePageMeta({
    title: "friends.requests.title",
    requiresAuth: true,
});

const api = useApi();
const toast = useSmartToast();
const authStore = useAuthStore();

// 状态变量
const activeTab = ref<"received" | "sent">("received");
const { t } = useI18n();
const tabItems = computed(() => [
    {
        key: "received",
        label: `${t("friends.requests.tabs.received")} (${receivedRequests.value.length})`,
    },
    {
        key: "sent",
        label: `${t("friends.requests.tabs.sent")} (${sentRequests.value.length})`,
    },
]);
const loadingReceived = ref(true);
const loadingSent = ref(false);
const batchProcessing = ref(false);

const receivedRequests = ref<FriendRequest[]>([]);
const sentRequests = ref<FriendRequest[]>([]);

// 本地分页
const pageSize = ref(10);
const receivedPage = ref(1);
const sentPage = ref(1);
const receivedPageCount = computed(() =>
    Math.max(1, Math.ceil(receivedRequests.value.length / pageSize.value)),
);
const sentPageCount = computed(() =>
    Math.max(1, Math.ceil(sentRequests.value.length / pageSize.value)),
);
const pagedReceived = computed(() => {
    const start = (receivedPage.value - 1) * pageSize.value;
    return receivedRequests.value.slice(start, start + pageSize.value);
});
const pagedSent = computed(() => {
    const start = (sentPage.value - 1) * pageSize.value;
    return sentRequests.value.slice(start, start + pageSize.value);
});

// 表格列
const columnsReceived: TableColumn<any>[] = [
    { id: "avatar_name", key: "avatar_name", label: t("common.user") },
    {
        id: "actions",
        key: "actions",
        label: t("common.actions"),
        class: "w-48 text-right",
    },
] as any;
const columnsSent: TableColumn<any>[] = [
    { id: "avatar_name", key: "avatar_name", label: t("common.user") },
    {
        id: "actions",
        key: "actions",
        label: t("common.actions"),
        class: "w-48 text-right",
    },
] as any;

// 生命周期
onMounted(async () => {
    try {
        await loadReceivedRequests();
    } catch (error: any) {
        console.error("[FriendRequests] Failed to load initial data:", error);
        toast.add({
            title: t("friends.requests.errors.loadFailed.title"),
            description:
                error?.message ||
                t("friends.requests.errors.loadFailed.description"),
            color: "error",
        });
    }
});

// 监听标签切换
watch(activeTab, async (newTab) => {
    if (newTab === "sent" && sentRequests.value.length === 0) {
        await loadSentRequests();
    }
});

// 加载收到的请求
async function loadReceivedRequests() {
    loadingReceived.value = true;
    try {
        const data = await api.post(
            buildFriendUrl(FRIEND_ENDPOINTS.REQUESTS_RECEIVED),
            {},
            { auth: true },
        );
        receivedRequests.value = (data as any) || [];
        receivedPage.value = 1;
    } catch (e: any) {
        toast.add({
            title: t("common.loadFailed"),
            description:
                e?.message || t("friends.requests.errors.loadReceived"),
            color: "error",
        });
    } finally {
        loadingReceived.value = false;
    }
}

// 加载发送的请求
async function loadSentRequests() {
    loadingSent.value = true;
    try {
        const data = await api.post(
            buildFriendUrl(FRIEND_ENDPOINTS.REQUESTS_SENT),
            {},
            { auth: true },
        );
        sentRequests.value = (data as any) || [];
        sentPage.value = 1;
    } catch (e: any) {
        toast.add({
            title: t("common.loadFailed"),
            description: e?.message || t("friends.requests.errors.loadSent"),
            color: "error",
        });
    } finally {
        loadingSent.value = false;
    }
}

// 处理收到的请求
async function acceptRequest(request: FriendRequest) {
    if (request.processing) return;
    request.processing = true;
    try {
        await api.post(
            buildFriendUrl(`${FRIEND_ENDPOINTS.ACCEPT}/${request.user_id}`),
            {},
            { auth: true },
        );
        const index = receivedRequests.value.findIndex(
            (r) => r.user_id === request.user_id,
        );
        if (index > -1) {
            receivedRequests.value.splice(index, 1);
        }
        toast.add({
            title: t("friends.requests.toasts.acceptedOne", {
                name: request.nickname,
            }),
            color: "success",
        });
    } catch (e: any) {
        toast.add({
            title: t("common.actionFailed"),
            description: e?.message,
            color: "error",
        });
    } finally {
        request.processing = false;
    }
}

async function rejectRequest(request: FriendRequest) {
    if (request.processing) return;
    request.processing = true;
    try {
        await new Promise((resolve) => setTimeout(resolve, 1000));
        const index = receivedRequests.value.findIndex(
            (r) => r.user_id === request.user_id,
        );
        if (index > -1) {
            receivedRequests.value.splice(index, 1);
        }
        toast.add({
            title: t("friends.requests.toasts.rejectedOne", {
                name: request.nickname,
            }),
            color: "info",
        });
    } catch (e: any) {
        toast.add({
            title: t("common.actionFailed"),
            description: e?.message,
            color: "error",
        });
    } finally {
        request.processing = false;
    }
}

// 处理发送的请求
function cancelRequest(request: FriendRequest) {
    if (request.processing) return;
    pendingConfirm.value = { type: "cancel", payload: request };
    confirmOpen.value = true;
}

async function performCancelRequest(request: FriendRequest) {
    if (request.processing) return;
    request.processing = true;
    try {
        await api.post(
            buildFriendUrl(`${FRIEND_ENDPOINTS.CANCEL}/${request.user_id}`),
            {},
            { auth: true },
        );
        const index = sentRequests.value.findIndex(
            (r) => r.user_id === request.user_id,
        );
        if (index > -1) {
            sentRequests.value.splice(index, 1);
        }
        toast.add({
            title: t("friends.requests.toasts.canceledOne", {
                name: request.nickname,
            }),
            color: "info",
        });
    } catch (e: any) {
        toast.add({
            title: t("common.actionFailed"),
            description: e?.message,
            color: "error",
        });
    } finally {
        request.processing = false;
    }
}

// 批量操作
function batchAcceptAll() {
    if (receivedRequests.value.length === 0 || batchProcessing.value) return;
    pendingConfirm.value = { type: "batch_accept" };
    confirmOpen.value = true;
}

async function performBatchAcceptAll() {
    if (receivedRequests.value.length === 0 || batchProcessing.value) return;
    batchProcessing.value = true;
    try {
        await new Promise((resolve) => setTimeout(resolve, 2000));
        const count = receivedRequests.value.length;
        receivedRequests.value = [];
        toast.add({
            title: t("friends.requests.toasts.acceptedAll", { count }),
            color: "success",
        });
    } catch (e: any) {
        toast.add({
            title: t("friends.requests.toasts.batchFailed"),
            description: e?.message,
            color: "error",
        });
    } finally {
        batchProcessing.value = false;
    }
}

function batchRejectAll() {
    if (receivedRequests.value.length === 0 || batchProcessing.value) return;
    pendingConfirm.value = { type: "batch_reject" };
    confirmOpen.value = true;
}

async function performBatchRejectAll() {
    if (receivedRequests.value.length === 0 || batchProcessing.value) return;
    batchProcessing.value = true;
    try {
        await new Promise((resolve) => setTimeout(resolve, 2000));
        const count = receivedRequests.value.length;
        receivedRequests.value = [];
        toast.add({
            title: t("friends.requests.toasts.rejectedAll", { count }),
            color: "info",
        });
    } catch (e: any) {
        toast.add({
            title: t("friends.requests.toasts.batchFailed"),
            description: e?.message,
            color: "error",
        });
    } finally {
        batchProcessing.value = false;
    }
}

// 状态文案/颜色
function getRequestStatusText(status: string): string {
    switch (status) {
        case "pending":
            return t("friends.requests.status.pending");
        case "accepted":
            return t("friends.requests.status.accepted");
        case "rejected":
            return t("friends.requests.status.rejected");
        default:
            return status;
    }
}

function getRequestStatusChipColor(status: string): any {
    switch (status) {
        case "pending":
            return "warning";
        case "accepted":
            return "success";
        case "rejected":
            return "neutral";
        default:
            return "neutral";
    }
}

// 统一确认弹窗
const confirmOpen = ref(false);
const pendingConfirm = ref<{
    type: "cancel" | "batch_accept" | "batch_reject";
    payload?: any;
} | null>(null);

function onConfirmCancel() {
    const payload = pendingConfirm.value?.payload as FriendRequest | undefined;
    confirmOpen.value = false;
    pendingConfirm.value = null;
    if (payload) {
        performCancelRequest(payload);
    }
}

function onConfirmBatchAccept() {
    confirmOpen.value = false;
    pendingConfirm.value = null;
    performBatchAcceptAll();
}

function onConfirmBatchReject() {
    confirmOpen.value = false;
    pendingConfirm.value = null;
    performBatchRejectAll();
}

function formatTime(dateStr: string): string {
    const date = new Date(dateStr);
    const now = new Date();
    const diff = now.getTime() - date.getTime();
    const days = Math.floor(diff / (1000 * 60 * 60 * 24));
    if (days === 0) {
        const hours = Math.floor(diff / (1000 * 60 * 60));
        if (hours === 0) {
            const minutes = Math.floor(diff / (1000 * 60));
            return t("time.minutesAgo", { n: minutes });
        }
        return t("time.hoursAgo", { n: hours });
    } else if (days < 7) {
        return t("time.daysAgo", { n: days });
    } else {
        return date.toLocaleDateString();
    }
}
</script>
