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

            <!-- 加载状态 -->
            <div v-if="loading" class="space-y-4">
                <SkeletonLoader type="table-row" :count="5" />
            </div>

            <!-- 错误状态 -->
            <div v-else-if="error" class="text-center py-8">
                <UIcon name="i-heroicons-exclamation-triangle" class="w-12 h-12 mx-auto text-error-500 mb-4" />
                <h3 class="text-lg font-semibold mb-2">加载失败</h3>
                <p class="text-neutral-500 mb-4">{{ error }}</p>
                <UButton @click="loadConversations" variant="outline">重试</UButton>
            </div>

            <!-- 会话列表 -->
            <UTable v-else :rows="pagedConversations" :columns="columns">
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
// TableColumn 类型已在组件内部定义
import { defineAsyncComponent } from 'vue';
import { useMessageService } from '~/services/message.service';
import { formatTime } from '~/utils/formatters';

// 懒加载组件
const EmptyMessages = defineAsyncComponent(() => import("~/components/illustrations/EmptyMessages.vue"));
const SkeletonLoader = defineAsyncComponent(() => import("~/components/SkeletonLoader.vue"));

definePageMeta({ 
    title: '消息', 
    requiresAuth: true 
});

const conversations = ref<any[]>([]);
const loading = ref(true);
const error = ref<string | null>(null);
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

// formatTime 函数已从统一工具函数导入

async function openConversation(row: any) {
    try {
        if (row.type === "user") {
            await messageService.markAsRead(String(row.id), "user");
            navigateTo(`/chat/users/${row.id}`);
        } else {
            await messageService.markAsRead(String(row.id), "group");
            navigateTo(`/chat/groups/${row.id}`);
        }
    } catch (_) {
        if (row.type === "user") navigateTo(`/chat/users/${row.id}`);
        else navigateTo(`/chat/groups/${row.id}`);
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
const toast = useSmartToast();

async function loadConversations() {
    loading.value = true;
    error.value = null;
    
    try {
        // 注意：会话列表功能需要后端 API 支持，当前使用模拟数据
        // const response = await messageService.getConversations();
        // conversations.value = response.items || [];
        
        // 模拟数据，实际应该从 API 获取
        conversations.value = [
            {
                id: 1,
                type: 'user',
                name: '张三',
                last_message: { content: '你好，最近怎么样？' },
                last_message_time: new Date(Date.now() - 3600000).toISOString(),
                unread_count: 2
            },
            {
                id: 2,
                type: 'group',
                name: '工作群',
                last_message: { content: '明天的会议改到下午3点' },
                last_message_time: new Date(Date.now() - 7200000).toISOString(),
                unread_count: 0
            }
        ];
        
        // 获取未读消息总数
        const resp = await messageService.getUserUnreadCount();
        totalUnread.value = Number(resp?.total_unread_count || 0);
        
    } catch (e: any) {
        error.value = e?.message || '加载会话列表失败';
        console.error('[ChatIndex] Failed to load conversations:', e);
        toast.add({
            title: '加载失败',
            description: error.value || '未知错误',
            color: 'error'
        });
    } finally {
        loading.value = false;
    }
}

onMounted(() => {
    loadConversations();
});
</script>
