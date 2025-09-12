<template>
    <div class="min-h-screen bg-gradient-to-br from-neutral-50 via-white to-neutral-100 dark:from-neutral-900 dark:via-neutral-950 dark:to-neutral-900">
        <div class="mx-auto max-w-4xl px-4 sm:px-6 lg:px-8 py-8">
            <!-- 页面标题 -->
            <div class="flex items-center justify-between mb-8">
                <div>
                    <h1 class="text-3xl font-bold text-neutral-900 dark:text-white">消息</h1>
                    <p class="text-neutral-600 dark:text-neutral-400 mt-1">与朋友和群组保持联系</p>
                </div>
                <div class="flex items-center gap-3">
                    <UBadge 
                        v-if="totalUnread > 0" 
                        :label="`${totalUnread} 条未读`"
                        color="primary" 
                        variant="solid"
                        size="lg"
                        class=""
                    />
                    <UButton 
                        icon="i-heroicons-arrow-path"
                        variant="ghost"
                        size="sm"
                        :loading="loading"
                        @click="loadConversations"
                        class="shrink-0"
                    >
                        刷新
                    </UButton>
                </div>
            </div>

            <!-- 快速操作 -->
            <div class="grid grid-cols-2 gap-4 mb-8">
                <UCard 
                    class="cursor-pointer transition-colors duration-200 hover:bg-blue-100 dark:hover:bg-neutral-800 bg-gradient-to-r from-blue-50 to-blue-100 dark:from-blue-950/50 dark:to-blue-900/50 border-blue-200 dark:border-blue-800"
                    @click="navigateTo('/users/search')"
                >
                    <div class="flex items-center gap-4 p-4">
                        <div class="flex-shrink-0">
                            <div class="w-12 h-12 bg-blue-500 rounded-xl flex items-center justify-center transition-colors">
                                <UIcon name="i-heroicons-user-plus" class="w-6 h-6 text-white" />
                            </div>
                        </div>
                        <div>
                            <h3 class="font-semibold text-blue-900 dark:text-blue-100">开始新聊天</h3>
                            <p class="text-sm text-blue-700 dark:text-blue-300">搜索用户并发起对话</p>
                        </div>
                    </div>
                </UCard>
                
                <UCard 
                    class="cursor-pointer transition-colors duration-200 hover:bg-green-100 dark:hover:bg-neutral-800 bg-gradient-to-r from-green-50 to-green-100 dark:from-green-950/50 dark:to-green-900/50 border-green-200 dark:border-green-800"
                    @click="navigateTo('/groups')"
                >
                    <div class="flex items-center gap-4 p-4">
                        <div class="flex-shrink-0">
                            <div class="w-12 h-12 bg-green-500 rounded-xl flex items-center justify-center transition-colors">
                                <UIcon name="i-heroicons-user-group" class="w-6 h-6 text-white" />
                            </div>
                        </div>
                        <div>
                            <h3 class="font-semibold text-green-900 dark:text-green-100">群组聊天</h3>
                            <p class="text-sm text-green-700 dark:text-green-300">查看和管理群组消息</p>
                        </div>
                    </div>
                </UCard>
            </div>

            <!-- 会话列表 -->
            <UCard class="bg-white/80 dark:bg-neutral-800/80 backdrop-blur-sm">
                <template #header>
                    <div class="flex items-center justify-between">
                        <h2 class="text-xl font-semibold text-neutral-900 dark:text-white">最近会话</h2>
                        <UButton 
                            v-if="conversations.length > 0"
                            variant="ghost" 
                            size="sm"
                            icon="i-heroicons-funnel"
                        >
                            筛选
                        </UButton>
                    </div>
                </template>

                <!-- 加载状态 -->
                <LoadingState v-if="loading" message="正在加载会话列表..." />

                <!-- 错误状态 -->
                <ErrorState 
                    v-else-if="error" 
                    icon="i-heroicons-exclamation-triangle"
                    title="加载失败"
                    :description="error"
                    action-text="重试"
                    @action="loadConversations"
                />

                <!-- 空状态 -->
                <EmptyState
                    v-else-if="conversations.length === 0"
                    icon="i-heroicons-chat-bubble-left-right"
                    title="暂无会话"
                    description="开始与朋友聊天，或加入群组开启对话"
                >
                    <template #actions>
                        <div class="flex gap-3">
                            <UButton @click="navigateTo('/users/search')" color="primary">
                                <UIcon name="i-heroicons-user-plus" class="w-4 h-4 mr-2" />
                                开始新聊天
                            </UButton>
                            <UButton @click="navigateTo('/groups')" variant="outline">
                                <UIcon name="i-heroicons-user-group" class="w-4 h-4 mr-2" />
                                查看群组
                            </UButton>
                        </div>
                    </template>
                </EmptyState>

                <!-- 会话列表 -->
                <div v-else class="divide-y divide-neutral-200 dark:divide-neutral-700">
                    <div
                        v-for="conversation in pagedConversations"
                        :key="conversation.id"
                        class="group p-4 hover:bg-neutral-200 dark:hover:bg-neutral-600/50 cursor-pointer transition-colors"
                        @click="openConversation(conversation)"
                    >
                        <div class="flex items-center gap-4">
                            <!-- 头像 -->
                            <div class="relative">
                                <UAvatar 
                                    :src="conversation.avatar_url"
                                    :alt="conversation.name"
                                    size="lg"
                                    :class="conversation.type === 'group' ? 'bg-primary-500 dark:bg-primary-400' : 'bg-green-500 dark:bg-green-400'"
                                >
                                    <UIcon 
                                        :name="conversation.type === 'group' ? 'i-heroicons-user-group' : 'i-heroicons-user'"
                                        class="w-6 h-6"
                                    />
                                </UAvatar>
                                <!-- 在线状态指示器 -->
                                <div 
                                    v-if="conversation.type === 'user' && conversation.is_online"
                                    class="absolute -bottom-1 -right-1 w-4 h-4 bg-green-500 border-2 border-white dark:border-neutral-800 rounded-full"
                                />
                                <!-- 未读消息指示器 -->
                                <div 
                                    v-if="conversation.unread_count > 0"
                                    class="absolute -top-1 -right-1 min-w-[20px] h-5 bg-red-500 text-white text-xs font-medium rounded-full flex items-center justify-center px-1"
                                >
                                    {{ conversation.unread_count > 99 ? '99+' : conversation.unread_count }}
                                </div>
                            </div>

                            <!-- 会话信息 -->
                            <div class="flex-1 min-w-0">
                                <div class="flex items-center justify-between mb-1">
                                    <h3 class="font-semibold text-neutral-900 dark:text-white truncate">
                                        {{ conversation.name }}
                                    </h3>
                                    <span class="text-xs text-neutral-500 dark:text-neutral-400 flex-shrink-0 ml-2">
                                        {{ formatTime(conversation.last_message_time) }}
                                    </span>
                                </div>
                                <div class="flex items-center justify-between">
                                    <p class="text-sm text-neutral-600 dark:text-neutral-400 truncate">
                                        <span v-if="conversation.last_message?.sender_name && conversation.type === 'group'" class="font-medium">
                                            {{ conversation.last_message.sender_name }}:
                                        </span>
                                        {{ conversation.last_message?.content || '暂无消息' }}
                                    </p>
                                    <div class="flex items-center gap-2 ml-2">
                                        <!-- 消息类型图标 -->
                                        <UIcon 
                                            v-if="conversation.last_message?.message_type === 'image'"
                                            name="i-heroicons-photo"
                                            class="w-4 h-4 text-neutral-400"
                                        />
                                        <UIcon 
                                            v-else-if="conversation.last_message?.message_type === 'voice'"
                                            name="i-heroicons-microphone"
                                            class="w-4 h-4 text-neutral-400"
                                        />
                                        <!-- 群组类型标识 -->
                                        <UBadge 
                                            v-if="conversation.type === 'group'"
                                            color="primary"
                                            variant="soft"
                                            size="xs"
                                        >
                                            群组
                                        </UBadge>
                                    </div>
                                </div>
                            </div>

                            <!-- 操作按钮 -->
                            <div class="opacity-0 group-hover:opacity-100 transition-opacity">
                                <UDropdownMenu :items="getActions(conversation)">
                                    <UButton
                                        variant="ghost"
                                        size="sm"
                                        icon="i-heroicons-ellipsis-vertical"
                                        @click.stop
                                    />
                                </UDropdownMenu>
                            </div>
                        </div>
                    </div>
                </div>

                <!-- 分页 -->
                <div v-if="pageCount > 1" class="p-4 border-t border-neutral-200 dark:border-neutral-700">
                    <UPagination
                        v-model:page="page"
                        :items-per-page="pageSize"
                        :total="conversations.length"
                        class="justify-center"
                    />
                </div>
            </UCard>
        </div>
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
                avatar_url: null,
                is_online: true,
                last_message: { 
                    content: '你好，最近怎么样？',
                    message_type: 'text',
                    sender_name: '张三'
                },
                last_message_time: new Date(Date.now() - 3600000).toISOString(),
                unread_count: 2
            },
            {
                id: 2,
                type: 'group',
                name: '工作群',
                avatar_url: null,
                is_online: false,
                last_message: { 
                    content: '明天的会议改到下午3点',
                    message_type: 'text',
                    sender_name: '李四'
                },
                last_message_time: new Date(Date.now() - 7200000).toISOString(),
                unread_count: 0
            },
            {
                id: 3,
                type: 'user',
                name: '王五',
                avatar_url: null,
                is_online: false,
                last_message: { 
                    content: '[图片]',
                    message_type: 'image',
                    sender_name: '王五'
                },
                last_message_time: new Date(Date.now() - 86400000).toISOString(),
                unread_count: 1
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
