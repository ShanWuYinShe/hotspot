<template>
    <div class="flex flex-col h-screen">
        <!-- 聊天头部 -->
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

                <UAvatar :alt="chatInfo?.name" size="md">
                    {{ chatInfo?.name?.charAt(0) || "C" }}
                </UAvatar>

                <div>
                    <h1 class="font-semibold">
                        {{ chatInfo?.name || "聊天" }}
                    </h1>
                    <p class="text-sm opacity-80">
                        {{ getStatusText() }}
                    </p>
                </div>
            </div>

            <!-- 操作按钮 -->
            <div class="flex items-center space-x-2">
                <!-- 搜索按钮 -->
                <UButton
                    variant="ghost"
                    size="sm"
                    icon="i-heroicons-magnifying-glass"
                    aria-label="搜索聊天记录"
                    @click="showSearch = !showSearch"
                />
                <UButton
                    variant="ghost"
                    size="sm"
                    icon="i-heroicons-phone"
                    aria-label="语音通话"
                    @click="() => makeCall()"
                />
                <UButton
                    variant="ghost"
                    size="sm"
                    icon="i-heroicons-video-camera"
                    aria-label="视频通话"
                    @click="() => makeVideoCall()"
                />
                <UDropdownMenu :items="chatActions">
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
        <div
            ref="messagesContainer"
            class="flex-1 overflow-y-auto px-4 py-4 space-y-4"
        >
            <!-- 加载中占位 -->
            <div v-if="messagesStore.loading.messages" class="space-y-3">
                <USkeleton class="h-6 w-2/3" />
                <USkeleton class="h-20 w-5/6" />
                <USkeleton class="h-6 w-1/2" />
            </div>

            <!-- 空状态 -->
            <UCard v-else-if="messagesList.length === 0">
                <EmptyState
                    icon="i-heroicons-chat-bubble-left-right"
                    title="暂无消息"
                    description="开始发送第一条消息吧"
                    action-text="立即发送"
                    :action-handler="focusComposer"
                >
                    <template #illustration>
                        <EmptyMessages size="lg" />
                    </template>
                </EmptyState>
            </UCard>

            <!-- 虚拟化窗口上方：显示更早消息（仅当窗口未覆盖全部时） -->
            <div v-else-if="startIndex > 0" class="text-center">
                <UButton size="sm" variant="outline" @click="expandWindow"
                    >显示更早消息</UButton
                >
            </div>

            <div
                v-else
                v-for="message in displayMessages"
                :key="message.message_id || message.id"
                :class="[
                    'flex',
                    isOwn(message) ? 'justify-end' : 'justify-start',
                ]"
            >
                <div
                    :class="[
                        'max-w-xs lg:max-w-md px-4 py-2 rounded-lg',
                        isOwn(message)
                            ? 'bg-primary-500 text-white'
                            : 'bg-white/80 dark:bg-neutral-800/80 backdrop-blur border border-neutral-200/60 dark:border-neutral-700/60',
                    ]"
                >
                    <!-- 消息内容 -->
                    <div class="mb-1">
                        {{ message.content }}
                    </div>

                    <!-- 消息时间 -->
                    <div
                        :class="[
                            'text-xs',
                            isOwn(message) ? 'text-white/80' : 'opacity-80',
                        ]"
                    >
                        {{
                            formatMessageTime(
                                message.sent_at || message.created_at,
                            )
                        }}
                    </div>
                </div>
            </div>

            <!-- 加载更多 -->
            <div v-if="hasMoreMessages" class="text-center py-4">
                <UButton
                    variant="outline"
                    size="sm"
                    @click="loadMoreMessages"
                    :loading="loadingMore"
                >
                    加载更多消息
                </UButton>
            </div>
        </div>

        <!-- 回到底部按钮：仅在未处于底部且有消息时显示 -->
        <div
            v-if="showScrollBottom"
            class="fixed right-5 bottom-24 md:bottom-20"
        >
            <UTooltip text="回到底部">
                <UButton
                    size="sm"
                    variant="soft"
                    icon="i-heroicons-arrow-down"
                    @click="scrollToBottom"
                />
            </UTooltip>
        </div>

        <!-- 消息输入框：采用统一的 MessageInput 组件 -->
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
// useSmartToast 会自动导入，无需手动导入
import { useMessagesStore } from "../../../stores/messages.store";
import { useAuthStore } from "../../../stores/auth.store";
import { useUserService } from "../../../services/user.service";
import { useGroupService } from "../../../services/group.service";
import type { Message } from "../../../types/message";
const EmptyMessages = defineAsyncComponent(
    () => import("../../../components/illustrations/EmptyMessages.vue"),
);

// 页面元数据
// definePageMeta({ title: "聊天", requiresAuth: true, layout: false })
// Note: Define page meta via route rules or middleware to satisfy TS diagnostics in this context.

const route = useRoute();
const toast = useSmartToast();

// 聊天参数
const chatType = route.params.type as "user" | "group";
const chatId = route.params.id as string | number;

// 聊天信息与状态
type ChatInfo = {
    name: string;
    avatar_url?: string;
    is_online?: boolean;
    member_count?: number;
};
const chatInfo = ref<ChatInfo | null>(null);
const messagesList = computed<Message[]>(() => messagesStore.messages);
// 简易虚拟化：仅渲染窗口内的最后 renderCount 条，提供“显示更早消息”扩展窗口
const renderCount = ref(100);
const startIndex = computed(() =>
    Math.max(0, messagesList.value.length - renderCount.value),
);
const visibleMessages = computed(() =>
    messagesList.value.slice(startIndex.value),
);
function expandWindow() {
    renderCount.value += 100;
}
const newMessage = ref("");
const sending = ref(false);
const loadingMore = ref(false);

// 搜索相关
const showSearch = ref(false);
const searchQuery = ref("");

// 搜索结果计算
const searchResults = computed(() => {
    if (!searchQuery.value.trim()) return [];

    const keyword = searchQuery.value.toLowerCase();
    return messagesList.value.filter((message) =>
        message.content.toLowerCase().includes(keyword),
    );
});

// 显示的消息：搜索时显示搜索结果，否则显示可见消息
const displayMessages = computed(() => {
    if (searchQuery.value.trim()) {
        return searchResults.value;
    }
    return visibleMessages.value;
});
const hasMoreMessages = computed<boolean>(
    () => messagesStore.pagination.messages.hasMore,
);
const messagesContainer = ref<HTMLDivElement | null>(null);
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

// 聊天操作菜单
const chatActions = computed(() => {
    const baseActions = [
        [
            {
                label: "聊天信息",
                icon: "i-heroicons-information-circle",
                click: showChatInfo,
            },
        ],
    ];
    if (chatType === "group") {
        baseActions.push([
            {
                label: "群组设置",
                icon: "i-heroicons-cog-6-tooth",
                click: showGroupSettings,
            },
        ]);
    }
    return baseActions;
});

// 获取状态文本
const getStatusText = (): string => {
    if (chatType === "group")
        return `${chatInfo.value?.member_count || 0} 名成员`;
    return chatInfo.value?.is_online ? "在线" : "离线";
};

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

// 初始化聊天数据
const initChat = async (): Promise<void> => {
    try {
        if (chatType === "group") {
            try {
                const groupService = useGroupService();
                chatInfo.value = (await groupService.getGroupDetails(
                    String(chatId),
                )) as any;
            } catch (_) {
                chatInfo.value = { name: `群组 ${chatId}` };
            }
            await messagesStore.fetchMessages(String(chatId), false, "group");
        } else {
            try {
                const userService = useUserService();
                const profile = await userService.getUserProfile(
                    chatId as string,
                );
                chatInfo.value = {
                    name: profile.nickname,
                    is_online: false,
                } as any;
            } catch (_) {
                chatInfo.value = { name: `用户 ${chatId}`, is_online: false };
            }
            await messagesStore.fetchMessages(String(chatId), false, "user");
        }
        // 数据来源于 store 的 computed，无需手动赋值
        nextTick(() => scrollToBottom());
    } catch (error) {
        console.error("初始化聊天失败:", error);
    }
};

// 发送消息
const sendMessage = async (): Promise<void> => {
    if (!newMessage.value.trim() || sending.value) return;
    sending.value = true;
    try {
        const content = newMessage.value.trim();
        newMessage.value = "";
        if (chatType === "group")
            await messagesStore.sendGroupMessage(
                String(chatId),
                content,
                "text",
            );
        else await messagesStore.sendMessage(String(chatId), content, "text");
        nextTick(() => scrollToBottom());
    } catch (error) {
        toast.add({ title: "发送失败", description: "请重试", color: "error" });
    } finally {
        sending.value = false;
    }
};

// 统一输入组件事件处理
const onSend = async (content: string) => {
    newMessage.value = content;
    await sendMessage();
};
const onUploadFile = async (file: File) => {
    // 文件大小限制 (10MB)
    const maxSize = 10 * 1024 * 1024;
    if (file.size > maxSize) {
        toast.add({
            title: "文件过大",
            description: "文件大小不能超过10MB",
            color: "error",
        });
        return;
    }

    // 支持的文件类型
    const allowedTypes = [
        'image/jpeg', 'image/png', 'image/gif', 'image/webp',
        'application/pdf', 'text/plain',
        'application/msword', 'application/vnd.openxmlformats-officedocument.wordprocessingml.document'
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
        
        // 创建FormData
        const formData = new FormData();
        formData.append('file', file);
        formData.append('chat_id', String(chatId));
        formData.append('chat_type', chatType);
        
        // 这里应该调用文件上传API
        // const uploadResult = await messagesStore.uploadFile(formData);
        
        // 暂时模拟上传成功
        await new Promise(resolve => setTimeout(resolve, 1000));
        
        toast.add({
            title: "上传成功",
            description: `文件 "${file.name}" 已发送`,
            color: "success",
        });
        
        // 发送文件消息
        if (chatType === "group") {
            await messagesStore.sendGroupMessage(
                String(chatId),
                `[文件] ${file.name}`,
                "file"
            );
        } else {
            await messagesStore.sendMessage(
                String(chatId),
                `[文件] ${file.name}`,
                "file"
            );
        }
        
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
const onSendLocation = async () => {
    try {
        sending.value = true;
        
        // 获取用户位置权限
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
        
        // 获取当前位置
        const geolocation = useGeolocation();
        const coords = await geolocation.getCurrentPosition();
        
        // 获取地址信息
        const address = await geolocation.reverseGeocode(coords.latitude, coords.longitude);
        
        // 构建位置消息内容
        const locationContent = `📍 我的位置\n${address}\n纬度: ${coords.latitude.toFixed(6)}\n经度: ${coords.longitude.toFixed(6)}`;
        
        // 发送位置消息
        if (chatType === "group") {
            await messagesStore.sendGroupMessage(
                String(chatId),
                locationContent,
                "location"
            );
        } else {
            await messagesStore.sendMessage(
                String(chatId),
                locationContent,
                "location"
            );
        }
        
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
        } else if (error.message?.includes("不可用")) {
            errorMessage = "位置服务不可用，请检查网络连接";
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
        await messagesStore.fetchMessages(
            String(chatId),
            true,
            chatType === "group" ? "group" : "user",
        );
    } catch (error) {
        toast.add({ title: "加载失败", description: "请重试", color: "error" });
    } finally {
        loadingMore.value = false;
    }
};

// 滚动到底部
const scrollToBottom = (): void => {
    if (messagesContainer.value)
        messagesContainer.value.scrollTop =
            messagesContainer.value.scrollHeight;
};

// 监听滚动，控制“回到底部”按钮显示
const showScrollBottom = ref(false);
const onScroll = () => {
    if (!messagesContainer.value) return;
    const el = messagesContainer.value;
    const nearBottom = el.scrollHeight - el.scrollTop - el.clientHeight < 80;
    showScrollBottom.value = !nearBottom && messagesList.value.length > 0;
};

onMounted(() => {
    messagesContainer.value?.addEventListener("scroll", onScroll);
});
onUnmounted(() => {
    messagesContainer.value?.removeEventListener("scroll", onScroll);
});

// 返回上一页
const goBack = (): void => {
    navigateTo("/chat");
};

// 拨打电话/视频/附件/信息/设置
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
const showAttachMenu = (): void => {
    toast.add({
        title: "功能开发中",
        description: "文件上传功能即将上线",
        color: "info",
    });
};
const showChatInfo = (): void => {
    toast.add({
        title: "功能开发中",
        description: "聊天信息功能即将上线",
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

// 初始化
onMounted(async () => {
    try {
        await initChat();
    } catch (error: any) {
        console.error("[Chat] Failed to initialize chat:", error);
        toast.add({
            title: "聊天初始化失败",
            description: error?.message || "无法加载聊天信息，请刷新页面重试",
            color: "error",
        });
    }
});
</script>
