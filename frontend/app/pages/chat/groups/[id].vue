<template>
    <div class="flex flex-col h-screen">
        <!-- 群聊头部 -->
        <div
            class="px-4 py-3 flex items-center justify-between bg-white/80 dark:bg-neutral-900/70 backdrop-blur border-b"
        >
            <div class="flex items-center space-x-3">
                <UButton
                    variant="ghost"
                    size="sm"
                    icon="i-heroicons-arrow-left"
                    aria-label="返回"
                    @click="goBack"
                    class="lg:hidden"
                />

                <UAvatar :alt="groupInfo?.name" size="md">
                    {{ groupInfo?.name?.charAt(0) || "G" }}
                </UAvatar>

                <div>
                    <h1 class="text-lg font-semibold text-neutral-900 dark:text-neutral-100">
                        {{ groupInfo?.name || "群聊" }}
                    </h1>
                    <p class="text-sm opacity-80">
                        {{ groupInfo?.member_count || 0 }} 名成员
                    </p>
                </div>
            </div>

            <!-- 群聊操作按钮 -->
            <div class="flex items-center space-x-2">
                <!-- 搜索按钮 -->
                <UButton
                    variant="ghost"
                    size="sm"
                    icon="i-heroicons-magnifying-glass"
                    aria-label="搜索聊天记录"
                    @click="showSearch = !showSearch"
                />
                <!-- 群成员 -->
                <UButton
                    variant="ghost"
                    size="sm"
                    icon="i-heroicons-users"
                    aria-label="群成员"
                    @click="showGroupMembers"
                />
                <!-- 群设置 -->
                <UDropdownMenu :items="groupActions">
                    <UButton
                        variant="ghost"
                        size="sm"
                        icon="i-heroicons-ellipsis-vertical"
                        aria-label="更多操作"
                    />
                </UDropdownMenu>
            </div>
        </div>

        <!-- 搜索栏 -->
        <div
            v-if="showSearch"
            class="px-4 py-3 bg-neutral-50 dark:bg-neutral-900 border-b border-neutral-200 dark:border-neutral-800"
        >
            <UInput
                v-model="searchQuery"
                placeholder="搜索群聊记录..."
                icon="i-heroicons-magnifying-glass"
                size="sm"
                @keyup.escape="showSearch = false"
            />
            <div
                v-if="searchResults.length > 0"
                class="mt-2 text-sm text-neutral-500"
            >
                找到 {{ searchResults.length }} 条相关消息
            </div>
        </div>

        <!-- 消息列表 -->
        <div class="flex-1">
            <!-- 加载中占位 -->
            <div v-if="messagesStore.loading.messages" class="h-full flex items-center justify-center">
                <div class="space-y-3 w-full max-w-md px-4">
                    <USkeleton class="h-6 w-2/3" />
                    <USkeleton class="h-20 w-5/6" />
                    <USkeleton class="h-6 w-1/2" />
                </div>
            </div>

            <!-- 空状态 -->
            <UCard v-else-if="messagesList.length === 0" class="m-4">
                <EmptyState
                    icon="i-heroicons-user-group"
                    title="暂无群聊消息"
                    description="开始在群里发送第一条消息吧"
                    action-text="立即发送"
                    :action-handler="focusComposer"
                >
                    <template #illustration>
                        <EmptyMessages size="lg" />
                    </template>
                </EmptyState>
            </UCard>

            <!-- 虚拟滚动聊天消息 -->
            <UVirtualChatMessages
                v-else
                :messages="messagesList"
                :current-user-id="String(authStore.user?.user_id || 0)"
                :container-height="400"
                :can-load-more="hasMoreMessages"
                :loading-more="loadingMore"
                :auto-scroll-to-bottom="true"
                @load-more="loadMoreMessages"
                ref="chatMessagesRef"
            >
                <template #default="{ message, isOwnMessage, showSender }">
                    <div
                        :class="[
                            'flex mb-4',
                            isOwnMessage ? 'justify-end' : 'justify-start',
                        ]"
                    >
                        <div
                            :class="[
                                'max-w-xs lg:max-w-md px-4 py-2 rounded-lg',
                                isOwnMessage
                                    ? 'bg-primary-500 text-white'
                                    : 'bg-white/80 dark:bg-neutral-800/80 backdrop-blur border border-neutral-200/60 dark:border-neutral-700/60',
                            ]"
                        >
                            <!-- 发送者信息（群聊中显示） -->
                            <div v-if="!isOwnMessage && showSender" class="text-xs opacity-70 mb-1">
                                {{ message.sender_name || `用户${message.sender_id}` }}
                            </div>

                            <!-- 消息内容 -->
                            <div class="mb-1">
                                {{ message.content }}
                            </div>

                            <!-- 消息时间 -->
                            <div
                                :class="[
                                    'text-xs',
                                    isOwnMessage ? 'text-white/80' : 'opacity-80',
                                ]"
                            >
                                {{
                                    formatMessageTime(
                                        message.created_at || message.sent_at || new Date(),
                                    )
                                }}
                            </div>
                        </div>
                    </div>
                </template>
            </UVirtualChatMessages>
        </div>

        <!-- 消息输入框 -->
        <div
            class="p-4 bg-white/80 dark:bg-neutral-900/70 backdrop-blur border-t"
        >
            <MessageInput
                :sending="sending"
                placeholder="在群里说点什么..."
                @send="onSend"
                @upload-file="onUploadFile"
                @send-location="onSendLocation"
            />
        </div>
    </div>
</template>

<script setup lang="ts">
import {
    defineAsyncComponent,
    ref,
    computed,
    onMounted,
    onUnmounted,
    nextTick,
} from "vue";
import { useRoute, navigateTo } from "nuxt/app";
import { useMessagesStore } from "~/stores/messages.store";
import { useAuthStore } from "~/stores/auth.store";
import type { Message } from "~/types/message";
import { useGroupService } from '~/services/group.service';

const EmptyMessages = defineAsyncComponent(
    () => import("~/components/illustrations/EmptyMessages.vue"),
);

const UVirtualChatMessages = defineAsyncComponent(
    () => import("~/components/UVirtualChatMessages.vue"),
);

// 页面元数据
definePageMeta({ 
    title: "群聊", 
    requiresAuth: true, 
    layout: false 
});

const route = useRoute();
const toast = useSmartToast();
const groupService = useGroupService();

// 群聊参数
const groupId = route.params.id as string;

// 群聊信息与状态
type GroupInfo = {
    name: string;
    avatar_url?: string;
    member_count?: number;
    description?: string;
};

const groupInfo = ref<GroupInfo | null>(null);
const messagesList = computed<Message[]>(() => messagesStore.messages);

// 聊天消息组件引用
const chatMessagesRef = ref<InstanceType<typeof UVirtualChatMessages>>();

const sending = ref(false);
const loadingMore = ref(false);

// 搜索相关
const showSearch = ref(false);
const searchQuery = ref("");

const searchResults = computed(() => {
    if (!searchQuery.value.trim()) return [];
    const keyword = searchQuery.value.toLowerCase();
    return messagesList.value.filter((message) =>
        message.content.toLowerCase().includes(keyword),
    );
});

const displayMessages = computed(() => {
    if (searchQuery.value.trim()) {
        return searchResults.value;
    }
    return messagesList.value;
});

const hasMoreMessages = computed<boolean>(
    () => messagesStore.pagination.messages.hasMore,
);
const focusComposer = () => {
    const el = document.querySelector(
        'textarea[placeholder*="群里"]',
    ) as HTMLTextAreaElement | null;
    el?.focus();
};

const messagesStore = useMessagesStore();
const authStore = useAuthStore();

// 是否本人消息
const isOwn = (m: Message) =>
    String(m.sender_id) === String(authStore.user?.user_id ?? "");

// 群聊操作菜单
const groupActions = computed(() => [
    [
        {
            label: "群聊信息",
            icon: "i-heroicons-information-circle",
            click: showGroupInfo,
        },
        {
            label: "群组设置",
            icon: "i-heroicons-cog-6-tooth",
            click: showGroupSettings,
        },
    ],
    [
        {
            label: "退出群聊",
            icon: "i-heroicons-arrow-right-on-rectangle",
            click: leaveGroup,
        },
    ],
]);

// 格式化消息时间
const formatMessageTime = (date: string | number | Date): string => {
    const d = new Date(date);
    const now = new Date();
    const today = new Date(now.getFullYear(), now.getMonth(), now.getDate());
    const day = new Date(d.getFullYear(), d.getMonth(), d.getDate());
    const timeStr = d.toLocaleTimeString("zh-CN", {
        hour: "2-digit",
        minute: "2-digit",
    });
    return day.getTime() === today.getTime()
        ? timeStr
        : `${d.toLocaleDateString("zh-CN")} ${timeStr}`;
};

// 初始化群聊数据
const initGroupChat = async (): Promise<void> => {
    try {
        // 获取群组信息
        try {
            groupInfo.value = (await groupService.getGroupDetails(groupId)) as any;
        } catch (_) {
            groupInfo.value = { name: `群组 ${groupId}`, member_count: 0 };
        }

        // 获取群聊消息
        await messagesStore.fetchMessages(groupId, false, "group");
        nextTick(() => scrollToBottom());
    } catch (error) {
        console.error("初始化群聊失败:", error);
        toast.add({
            title: "群聊初始化失败",
            description: "无法加载群聊信息，请刷新页面重试",
            color: "error",
        });
    }
};

// 发送群聊消息
const onSend = async (content: string) => {
    if (!content.trim() || sending.value) return;
    
    sending.value = true;
    try {
        await messagesStore.sendGroupMessage(groupId, content, "text");
        nextTick(() => scrollToBottom());
    } catch (error) {
        toast.add({ 
            title: "发送失败", 
            description: "请重试", 
            color: "error" 
        });
    } finally {
        sending.value = false;
    }
};

// 文件上传处理
const onUploadFile = async (file: File) => {
    const maxSize = 10 * 1024 * 1024; // 10MB
    if (file.size > maxSize) {
        toast.add({
            title: "文件过大",
            description: "文件大小不能超过10MB",
            color: "error",
        });
        return;
    }

    const allowedTypes = [
        'image/jpeg', 'image/png', 'image/gif', 'image/webp',
        'application/pdf', 'text/plain',
        'application/msword', 
        'application/vnd.openxmlformats-officedocument.wordprocessingml.document'
    ];
    
    if (!allowedTypes.includes(file.type)) {
        toast.add({
            title: "不支持的文件类型",
            description: "仅支持图片、PDF、文档等常见格式",
            color: "error",
        });
        return;
    }

    try {
        sending.value = true;
        
        // 暂时模拟上传
        await new Promise(resolve => setTimeout(resolve, 1000));
        
        // 发送文件消息
        await messagesStore.sendGroupMessage(
            groupId,
            `[文件] ${file.name}`,
            "file"
        );
        
        toast.add({
            title: "上传成功",
            description: `文件 "${file.name}" 已发送到群聊`,
            color: "success",
        });
        
        nextTick(() => scrollToBottom());
        
    } catch (error) {
        console.error("文件上传失败:", error);
        toast.add({
            title: "上传失败",
            description: "请检查网络连接后重试",
            color: "error",
        });
    } finally {
        sending.value = false;
    }
};

// 发送位置
const onSendLocation = async () => {
    try {
        sending.value = true;
        
        const permissions = usePermissions();
        const hasPermission = await permissions.checkGeolocation();
        
        if (!hasPermission) {
            const granted = await permissions.requestGeolocation();
            if (!granted) {
                toast.add({
                    title: "需要位置权限",
                    description: "请允许访问位置信息以发送位置",
                    color: "warning",
                });
                return;
            }
        }
        
        const geolocation = useGeolocation();
        const coords = await geolocation.getCurrentPosition();
        const address = await geolocation.reverseGeocode(coords.latitude, coords.longitude);
        
        const locationContent = `📍 我的位置\n${address}\n纬度: ${coords.latitude.toFixed(6)}\n经度: ${coords.longitude.toFixed(6)}`;
        
        await messagesStore.sendGroupMessage(groupId, locationContent, "location");
        
        toast.add({
            title: "位置已发送",
            description: "您的当前位置已分享到群聊",
            color: "success",
        });
        
        nextTick(() => scrollToBottom());
        
    } catch (error: any) {
        console.error("发送位置失败:", error);
        
        let errorMessage = "获取位置失败，请重试";
        if (error.message?.includes("权限")) {
            errorMessage = "位置权限被拒绝，请在浏览器设置中允许位置访问";
        } else if (error.message?.includes("超时")) {
            errorMessage = "获取位置超时，请检查GPS是否开启";
        }
        
        toast.add({
            title: "发送失败",
            description: errorMessage,
            color: "error",
        });
    } finally {
        sending.value = false;
    }
};

// 加载更多消息
const loadMoreMessages = async (): Promise<void> => {
    if (loadingMore.value) return;
    loadingMore.value = true;
    try {
        await messagesStore.fetchMessages(groupId, true, "group");
    } catch (error) {
        toast.add({ 
            title: "加载失败", 
            description: "请重试", 
            color: "error" 
        });
    } finally {
        loadingMore.value = false;
    }
};

// 滚动到底部
const scrollToBottom = (): void => {
    chatMessagesRef.value?.scrollToBottom();
};

// 返回上一页
const goBack = (): void => {
    navigateTo("/chat");
};

// 群聊操作
const showGroupMembers = (): void => {
    navigateTo(`/groups/${groupId}`);
};

const showGroupInfo = (): void => {
    toast.add({
        title: "功能开发中",
        description: "群聊信息功能即将上线",
        color: "info",
    });
};

const showGroupSettings = (): void => {
    toast.add({
        title: "功能开发中",
        description: "群组设置功能即将上线",
        color: "info",
    });
};

const leaveGroup = async (): Promise<void> => {
    try {
        await groupService.leaveGroup(groupId);
        toast.add({
            title: "已退出群聊",
            description: "您已成功退出该群聊",
            color: "success",
        });
        navigateTo("/chat");
    } catch (error) {
        toast.add({
            title: "退出失败",
            description: "请重试",
            color: "error",
        });
    }
};

// 初始化
onMounted(async () => {
    try {
        await initGroupChat();
    } catch (error: any) {
        console.error("[GroupChat] Failed to initialize:", error);
        toast.add({
            title: "群聊初始化失败",
            description: error?.message || "无法加载群聊信息，请刷新页面重试",
            color: "error",
        });
    }
});
</script>
