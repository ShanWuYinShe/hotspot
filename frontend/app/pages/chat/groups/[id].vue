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
                    <h1 class="text-lg font-semibold text-neutral-900 dark:text-white">
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
                class="mt-2 text-sm text-neutral-500 dark:text-neutral-400"
            >
                找到 {{ searchResults.length }} 条相关消息
            </div>
        </div>

        <!-- 消息列表 -->
        <div class="flex-1 overflow-hidden">
            <!-- 加载中占位 -->
            <div v-if="messagesStore.loading" class="h-full flex items-center justify-center">
                <div class="space-y-3 w-full max-w-md px-4">
                    <USkeleton class="h-6 w-2/3" />
                    <USkeleton class="h-20 w-5/6" />
                    <USkeleton class="h-6 w-1/2" />
                </div>
            </div>

            <!-- 空状态 -->
            <div v-else-if="messagesList.length === 0" class="h-full flex items-center justify-center p-4">
                <EmptyState
                    icon="i-heroicons-user-group"
                    title="暂无群聊消息"
                    description="开始在群里发送第一条消息吧"
                    action-text="立即发送"
                    @action="focusComposer"
                >
                    <template #illustration>
                        <EmptyMessages size="lg" />
                    </template>
                </EmptyState>
            </div>

            <!-- Nuxt UI v4 ChatMessages 组件 -->
            <UChatMessages
                v-else
                :messages="formattedMessages"
                :status="chatStatus"
                :should-auto-scroll="true"
                :should-scroll-to-bottom="true"
                :assistant="{
                    side: 'left',
                    variant: 'outline',
                    avatar: { fallback: groupInfo?.name?.charAt(0) || 'G', src: groupInfo?.avatar_url }
                }"
                :user="{
                    side: 'right',
                    variant: 'soft',
                    avatar: { fallback: authStore.user?.nickname?.charAt(0) || 'U', src: authStore.user?.avatar_url }
                }"
                class="h-full px-4"
            >
                <template #content="{ message }">
                    <div class="space-y-1">
                        <!-- 群聊中显示发送者名称 -->
                        <div v-if="message.role === 'assistant' && (message.metadata as any)?.senderName" 
                             class="text-xs text-neutral-500 dark:text-neutral-400 font-medium">
                            {{ (message.metadata as any).senderName }}
                        </div>
                        
                        <!-- 消息内容 -->
                        <div class="text-sm leading-relaxed">
                            {{ getMessageText(message) }}
                        </div>
                        
                        <!-- 消息时间 -->
                        <div class="text-xs text-neutral-400 dark:text-neutral-500">
                            {{ formatMessageTime((message.metadata as any)?.timestamp || new Date()) }}
                        </div>
                    </div>
                </template>
            </UChatMessages>
        </div>

        <!-- 消息输入框 -->
        <div class="p-4 bg-white/80 dark:bg-neutral-900/70 backdrop-blur border-t">
            <UChatPrompt
                v-model="messageInput"
                placeholder="在群里说点什么..."
                :loading="sending"
                variant="soft"
                @submit="handleSendMessage"
            >
                <template #trailing>
                    <div class="flex items-center gap-1">
                        <!-- 文件上传 -->
                        <UTooltip text="发送文件">
                            <UButton
                                variant="ghost"
                                size="xs"
                                icon="i-heroicons-paper-clip"
                                @click="triggerFileUpload"
                                :disabled="sending"
                            />
                        </UTooltip>
                        
                        <!-- 位置分享 -->
                        <UTooltip text="分享位置">
                            <UButton
                                variant="ghost"
                                size="xs"
                                icon="i-heroicons-map-pin"
                                @click="onSendLocation"
                                :disabled="sending"
                            />
                        </UTooltip>
                    </div>
                </template>
            </UChatPrompt>
            
            <!-- 隐藏的文件输入 -->
            <input
                ref="fileInputRef"
                type="file"
                class="hidden"
                accept="image/*,.pdf,.doc,.docx,.txt"
                @change="handleFileSelect"
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
import type { SendMessageResponse } from "~/types/api";
import { useGroupsService } from "~/services/groups.service";

const EmptyMessages = defineAsyncComponent(
    () => import("~/components/illustrations/EmptyMessages.vue"),
);

// 页面元数据
definePageMeta({ 
    title: "群聊", 
    requiresAuth: true, 
    layout: 'dashboard'
});

const route = useRoute();
const toast = useToast()
const { formatTime: baseFormatTime } = useFormatters();

// 格式化消息时间（群聊特殊显示需求）
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

// 使用服务实例
const groupService = useGroupsService();

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
const hasMoreMessages = ref(false);
const messagesList = computed(() => messagesStore.getMessages(groupId));

// 消息输入和状态
const messageInput = ref('');
const sending = ref(false);
const loadingMore = ref(false);
const fileInputRef = ref<HTMLInputElement>();

// 聊天状态
const chatStatus = computed(() => {
    if (sending.value) return 'submitted';
    if (loadingMore.value) return 'streaming';
    return 'ready';
});

// 格式化消息为ChatMessages组件需要的格式
const formattedMessages = computed(() => {
    return messagesList.value.map(message => ({
        id: message.id || String(Math.random()),
        role: (isOwn(message) ? 'user' : 'assistant') as 'user' | 'assistant' | 'system',
        parts: [{ type: 'text' as const, text: message.content || '' }],
        metadata: {
            senderName: message.sender.nickname,
            timestamp: message.sent_at || new Date().toISOString(),
            senderId: message.sender.user_id
        }
    }));
});

// 获取消息文本内容
const getMessageText = (message: any) => {
    return message.parts?.[0]?.text || message.content || '';
};

// 搜索相关
const showSearch = ref(false);
const searchQuery = ref("");

const searchResults = computed(() => {
    if (!searchQuery.value.trim()) return [];
    const keyword = searchQuery.value.toLowerCase();
    return messagesList.value.filter((message) =>
        message.content?.toLowerCase().includes(keyword),
    );
});

const displayMessages = computed(() => {
    if (searchQuery.value.trim()) {
        return searchResults.value;
    }
    return messagesList.value;
});
const focusComposer = () => {
    const el = document.querySelector(
        'textarea[placeholder*="群里"]',
    ) as HTMLTextAreaElement | null;
    el?.focus();
};

const messagesStore = useMessagesStore();
const authStore = useAuthStore();

// 是否本人消息
const isOwn = (m: any) =>
    String(m.sender?.user_id) === String(authStore.user?.user_id ?? "");

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
            click: leaveGroupChat,
        },
    ],
]);

// 格式化消息时间

// 初始化群聊数据
const initGroupChat = async (): Promise<void> => {
    try {
        // 获取群组信息
        try {
            const response = await groupService.openTheDoor({ id: groupId });
            groupInfo.value = {
                name: response.name || `群组 ${groupId}`,
                avatar_url: response.avatar_url,
                member_count: response.member_count,
                description: response.description
            };
        } catch (_) {
            groupInfo.value = { name: `群组 ${groupId}`, member_count: 0 };
        }

        // 获取群聊消息
        const pagination = await messagesStore.fetchGroupMessages(groupId);
        hasMoreMessages.value = pagination?.has_more ?? false;
    } catch (error) {
        console.error("初始化群聊失败:", error);
        toast.add({
            title: "群聊初始化失败",
            description: "无法加载群聊信息，请刷新页面重试",
            color: "error",
        });
    }
};

// 处理消息发送（ChatPrompt组件的submit事件）
const handleSendMessage = async (event: Event) => {
    event.preventDefault();
    const content = messageInput.value.trim();
    if (!content || sending.value) return;
    
    sending.value = true;
    try {
        await messagesStore.sendGroupMessage(groupId, { type: 'text', content });
        messageInput.value = ''; // 清空输入框
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

// 发送群聊消息（保留原方法供其他地方调用）
const onSend = async (content: string) => {
    if (!content.trim() || sending.value) return;
    
    sending.value = true;
    try {
        await messagesStore.sendGroupMessage(groupId, { type: 'text', content });
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

// 触发文件上传
const triggerFileUpload = () => {
    fileInputRef.value?.click();
};

// 处理文件选择
const handleFileSelect = (event: Event) => {
    const target = event.target as HTMLInputElement;
    const file = target.files?.[0];
    if (file) {
        onUploadFile(file);
        target.value = ''; // 清空文件输入
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
        
        await new Promise(resolve => setTimeout(resolve, 1000));
        
        // 发送文件消息
        await messagesStore.sendGroupMessage(groupId, {
            type: 'file',
            content: `[文件] ${file.name}`
        });
        
        toast.add({
            title: "上传成功",
            description: `文件 "${file.name}" 已发送到群聊`,
            color: "success",
        });
        
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
        
        const locationStore = useLocationStore();
        const permission = await locationStore.checkPermission();
        
        if (permission?.state !== 'granted') {
            const granted = await locationStore.requestPermission();
            if (!granted) {
                toast.add({
                    title: "需要位置权限",
                    description: "请允许访问位置信息以发送位置",
                    color: "warning",
                });
                return;
            }
        }
        
        const position = await new Promise<GeolocationPosition>((resolve, reject) => {
            navigator.geolocation.getCurrentPosition(resolve, reject);
        });
        
        const locationContent = `📍 我的位置\n纬度: ${position.coords.latitude.toFixed(6)}\n经度: ${position.coords.longitude.toFixed(6)}`;
        
        await messagesStore.sendGroupMessage(groupId, { type: 'text', content: locationContent });
        
        toast.add({
            title: "位置已发送",
            description: "您的当前位置已分享到群聊",
            color: "success",
        });
        
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
    if (loadingMore.value || !hasMoreMessages.value) return;
    loadingMore.value = true;
    try {
        // 获取当前最早的消息作为游标
        const firstMessage = messagesList.value[0];
        const pagination = await messagesStore.fetchGroupMessages(groupId, firstMessage?.id);
        hasMoreMessages.value = pagination?.has_more ?? false;
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

// 注意：scrollToBottom 方法已由 UChatMessages 组件自动处理

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

const leaveGroupChat = async (): Promise<void> => {
    try {
        await groupService.leaveGroup({ group_chat_id: groupId });
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
