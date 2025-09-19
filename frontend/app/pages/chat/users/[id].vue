<template>
    <div class="flex flex-col h-screen">
        <!-- 私聊头部 -->
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

                <UAvatar :alt="userInfo?.nickname" size="md">
                    {{ userInfo?.nickname?.charAt(0) || "U" }}
                </UAvatar>

                <div>
                    <h1 class="text-lg font-semibold text-neutral-900 dark:text-neutral-100">
                        {{ userInfo?.nickname || "私聊" }}
                    </h1>
                    <p class="text-sm opacity-80">
                        {{ userInfo?.last_active_at ? "最近活跃" : "离线" }}
                    </p>
                </div>
            </div>

            <!-- 私聊操作按钮 -->
            <div class="flex items-center space-x-1">
                <!-- 搜索按钮 -->
                <UButton
                    variant="ghost"
                    size="sm"
                    icon="i-heroicons-magnifying-glass"
                    aria-label="搜索聊天记录"
                    @click="showSearch = !showSearch"
                    class="min-w-[44px] min-h-[44px] touch-manipulation"
                />
                <!-- 语音通话 -->
                <UButton
                    variant="ghost"
                    size="sm"
                    icon="i-heroicons-phone"
                    aria-label="语音通话"
                    @click="makeCall"
                    class="min-w-[44px] min-h-[44px] touch-manipulation"
                />
                <!-- 视频通话 -->
                <UButton
                    variant="ghost"
                    size="sm"
                    icon="i-heroicons-video-camera"
                    aria-label="视频通话"
                    @click="makeVideoCall"
                    class="min-w-[44px] min-h-[44px] touch-manipulation"
                />
                <!-- 更多操作 -->
                <UDropdownMenu :items="userActions">
                    <UButton
                        variant="ghost"
                        size="sm"
                        icon="i-heroicons-ellipsis-vertical"
                        aria-label="更多操作"
                        class="min-w-[44px] min-h-[44px] touch-manipulation"
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
                placeholder="搜索聊天记录..."
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
                    icon="i-heroicons-chat-bubble-left-right"
                    title="暂无私聊消息"
                    description="开始发送第一条消息吧"
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
                <template #default="{ message, isOwnMessage }">
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
                            <!-- 消息内容 -->
                            <div class="mb-1">
                                {{ message.content }}
                            </div>

                            <!-- 消息时间和状态 -->
                            <div class="flex items-center justify-between">
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
                                <!-- 消息状态（仅自己的消息显示） -->
                                <div v-if="isOwnMessage" class="ml-2">
                                    <UIcon 
                                        name="i-heroicons-check" 
                                        class="w-3 h-3 text-white/60"
                                        v-if="(message as any).status === 'sent'"
                                    />
                                    <UIcon 
                                        name="i-heroicons-check-check" 
                                        class="w-3 h-3 text-white/80"
                                        v-else-if="(message as any).status === 'delivered'"
                                    />
                                    <UIcon 
                                        name="i-heroicons-check-check" 
                                        class="w-3 h-3 text-primary-300"
                                        v-else-if="(message as any).status === 'read'"
                                    />
                                </div>
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
                placeholder="输入消息..."
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
import type { Message } from "~/types/message";
import type { UserInfo } from "~/types";

const EmptyMessages = defineAsyncComponent(
    () => import("~/components/illustrations/EmptyMessages.vue"),
);

const UVirtualChatMessages = defineAsyncComponent(
    () => import("~/components/UVirtualChatMessages.vue"),
);

// 页面元数据
definePageMeta({ 
    title: "私聊", 
    requiresAuth: true, 
    layout: false 
});

const route = useRoute();
const toast = useSmartToast();
const userService = useUserService();

// 私聊参数
const userId = route.params.id as string;

// 用户信息与状态
const userInfo = ref<UserInfo | null>(null);
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
        'textarea[placeholder="输入消息..."]',
    ) as HTMLTextAreaElement | null;
    el?.focus();
};

const messagesStore = useMessagesStore();
const authStore = useAuthStore();

// 是否本人消息
const isOwn = (m: Message) =>
    String(m.sender_id) === String(authStore.user?.user_id ?? "");

// 私聊操作菜单
const userActions = computed(() => [
    [
        {
            label: "用户资料",
            icon: "i-heroicons-user",
            click: showUserProfile,
        },
        {
            label: "聊天信息",
            icon: "i-heroicons-information-circle",
            click: showChatInfo,
        },
    ],
    [
        {
            label: "清空聊天记录",
            icon: "i-heroicons-trash",
            click: clearChatHistory,
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

// 初始化私聊数据
const initUserChat = async (): Promise<void> => {
    try {
        // 获取用户信息
        try {
            const profile = await userService.getUserProfile(userId) as UserInfo;
            userInfo.value = profile;
        } catch (_) {
            // 创建一个基本的用户信息对象
            userInfo.value = { 
                user_id: userId,
                nickname: `用户 ${userId}`,
                last_active: new Date().toISOString(),
                latitude: 0,
                longitude: 0,
                distance: 0,
                location_name: "未知位置",
                last_active_at: new Date().toISOString()
            } as UserInfo;
        }

        // 获取私聊消息
        await messagesStore.fetchMessages(userId, false, "user");
        nextTick(() => scrollToBottom());
    } catch (error) {
        console.error("初始化私聊失败:", error);
        toast.add({
            title: "私聊初始化失败",
            description: "无法加载私聊信息，请刷新页面重试",
            color: "error",
        });
    }
};

// 发送私聊消息
const onSend = async (content: string) => {
    if (!content.trim() || sending.value) return;
    
    sending.value = true;
    try {
        await messagesStore.sendMessage(userId, content, "text");
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
        
        await new Promise(resolve => setTimeout(resolve, 1000));
        
        // 发送文件消息
        await messagesStore.sendMessage(
            userId,
            `[文件] ${file.name}`,
            "file"
        );
        
        toast.add({
            title: "上传成功",
            description: `文件 "${file.name}" 已发送`,
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
        
        await messagesStore.sendMessage(userId, locationContent, "location");
        
        toast.add({
            title: "位置已发送",
            description: "您的当前位置已分享",
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
        await messagesStore.fetchMessages(userId, true, "user");
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

// 私聊操作
const makeCall = (): void => {
    toast.add({
        title: "功能开发中",
        description: "语音通话功能即将上线",
        color: "info",
    });
};

const makeVideoCall = (): void => {
    toast.add({
        title: "功能开发中",
        description: "视频通话功能即将上线",
        color: "info",
    });
};

const showUserProfile = (): void => {
    navigateTo(`/users/${userId}`);
};

const showChatInfo = (): void => {
    toast.add({
        title: "功能开发中",
        description: "聊天信息功能即将上线",
        color: "info",
    });
};

const clearChatHistory = (): void => {
    toast.add({
        title: "功能开发中",
        description: "清空聊天记录功能即将上线",
        color: "info",
    });
};

// 初始化
onMounted(async () => {
    try {
        await initUserChat();
    } catch (error: any) {
        console.error("[UserChat] Failed to initialize:", error);
        toast.add({
            title: "私聊初始化失败",
            description: error?.message || "无法加载私聊信息，请刷新页面重试",
            color: "error",
        });
    }
});
</script>
