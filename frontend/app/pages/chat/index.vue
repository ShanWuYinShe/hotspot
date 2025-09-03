<template>
    <div class="mx-auto max-w-7xl px-4 sm:px-6 lg:px-8 py-8 sm:py-10">
        <UCard class="mb-6">
            <template #header>
                <div class="flex items-center justify-between">
                    <h1 class="text-2xl font-bold">消息</h1>
                    <UBadge v-if="totalUnread > 0" size="md" variant="soft"
                        >未读 {{ totalUnread }}</UBadge
                    >
                </div>
            </template>

            <UTable :rows="pagedConversations" :columns="columns as any">
                <template #conversation_info-cell="{ row }">
                    <div class="flex items-center gap-3 min-w-0">
                        <UAvatar :alt="(row as any).name">{{
                            ((row as any).name || "").charAt(0)
                        }}</UAvatar>
                        <div class="min-w-0">
                            <h3 class="font-semibold truncate">
                                {{ (row as any).name }}
                            </h3>
                            <p
                                class="text-sm text-neutral-500 truncate hidden sm:block"
                            >
                                {{ (row as any).last_message?.content }}
                            </p>
                        </div>
                    </div>
                </template>
                <template #last_message_time-cell="{ row }">
                    <span class="text-sm text-neutral-500">{{
                        formatTime((row as any).last_message_time)
                    }}</span>
                </template>
                <template #actions-cell="{ row }">
                    <div class="flex items-center justify-end gap-2">
                        <UButton size="sm" @click="openConversation(row as any)"
                            >打开</UButton
                        >
                        <UDropdownMenu :items="getActions(row as any)">
                            <UButton
                                variant="ghost"
                                size="sm"
                                icon="i-heroicons-ellipsis-vertical"
                                @click.stop
                            />
                        </UDropdownMenu>
                    </div>
                </template>
                <template #empty>
                    <EmptyMessages />
                </template>
            </UTable>

            <div v-if="pageCount > 1" class="mt-4 flex justify-center">
                <UPagination
                    v-model:page="page"
                    :items-per-page="pageSize"
                    :total="conversations.length"
                />
            </div>
        </UCard>
    </div>
</template>

<script setup lang="ts">
import type { TableColumn } from "@nuxt/ui";
import EmptyMessages from "~/components/illustrations/EmptyMessages.vue";
import { useMessageService } from "~/services/message.service";

const conversations = ref<any[]>([]);
const page = ref(1);
const pageSize = ref(10);
const pageCount = computed(() =>
    Math.max(1, Math.ceil(conversations.value.length / pageSize.value)),
);
const pagedConversations = computed(() => {
    const start = (page.value - 1) * pageSize.value;
    return conversations.value.slice(start, start + pageSize.value);
});

const columns = [
    { id: "conversation_info", key: "conversation_info", label: "会话" },
    {
        id: "last_message_time",
        key: "last_message_time",
        label: "时间",
        class: "hidden sm:table-cell w-32",
    },
    { id: "actions", key: "actions", label: "操作", class: "w-28 text-right" },
] as any;

function formatTime(t?: string) {
    if (!t) return "";
    const d = new Date(t);
    return d.toLocaleString();
}

async function openConversation(row: any) {
    try {
        if (row.type === "user") {
            await messageService.markAsRead(String(row.id), "user");
            navigateTo(`/chat/user/${row.id}`);
        } else {
            await messageService.markAsRead(String(row.id), "group");
            navigateTo(`/chat/group/${row.id}`);
        }
    } catch (_) {
        if (row.type === "user") navigateTo(`/chat/user/${row.id}`);
        else navigateTo(`/chat/group/${row.id}`);
    }
}

function getActions(row: any) {
    return [
        [
            {
                label: "打开",
                icon: "i-heroicons-chat-bubble-left-right",
                click: () => openConversation(row),
            },
        ],
    ];
}

// 显示汇总未读：进入页面即获取
const messageService = useMessageService();
const totalUnread = ref<number>(0);
onMounted(async () => {
    try {
        const resp = await messageService.getUserUnreadCount();
        totalUnread.value = Number(resp?.total_unread_count || 0);
    } catch (_) {
        totalUnread.value = 0;
    }
});
</script>
