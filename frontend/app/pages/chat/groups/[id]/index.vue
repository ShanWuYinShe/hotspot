<template>
    <div class="flex flex-col h-full">
        <!-- 群聊头部 -->
        <div
            class="px-3 py-2.5 flex items-center justify-between bg-modal backdrop-blur-xl border-b border-glass shrink-0"
        >
            <div class="flex items-center gap-2">
                <!-- 返回按钮 -->
                <UButton
                    variant="ghost"
                    size="sm"
                    icon="i-heroicons-arrow-left"
                    square
                    class="shrink-0"
                    @click="router.back()"
                />
                
                <UAvatar :alt="groupInfo?.name" size="md">
                    {{ groupInfo?.name?.charAt(0) || "G" }}
                </UAvatar>

                <div class="min-w-0">
                    <h1 class="text-sm font-semibold text-primary-content truncate">
                        {{ groupInfo?.name || "群聊" }}
                    </h1>
                    <p class="text-xs text-secondary-content">
                        {{ groupInfo?.members_count || 0 }} 名成员
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
            class="px-4 py-3 bg-surface border-b border-strong"
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
                class="mt-2 text-sm text-secondary-content"
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
                <UEmpty
                    icon="i-heroicons-chat-bubble-left-right"
                    title="暂无群聊消息"
                    description="开始在群里发送第一条消息吧"
                    :actions="[{ label: '立即发送', onClick: focusComposer }]"
                />
            </div>

            <!-- 自定义 ChatMessages 组件 -->
            <ChatMessages
                v-else
                :messages="formattedMessages"
                :status="chatStatus"
                :should-auto-scroll="true"
                :should-scroll-to-bottom="true"
                :is-group-chat="true"
                :user="{
                    side: 'right',
                    variant: 'soft',
                    avatar: { fallback: authStore.user?.nickname?.charAt(0) || 'U', src: authStore.user?.avatar_url }
                }"
                class="h-full px-4"
                @avatar-click="handleAvatarClick"
            >
                <template #content="{ message }: { message: GroupChatMessage }">
                    <div class="space-y-1">
                        <!-- 群聊中显示发送者名称 -->
                        <div v-if="message.role === 'assistant' && message.metadata?.senderName" 
                             class="text-xs opacity-70 font-medium">
                            {{ message.metadata.senderName }}
                        </div>
                        
                        <!-- 消息内容 -->
                        <div class="text-sm leading-relaxed">
                            {{ getMessageText(message) }}
                        </div>
                    </div>
                </template>
            </ChatMessages>
        </div>

        <!-- 消息输入框 -->
        <div class="shrink-0 px-3 py-2 border-t border-glass bg-modal backdrop-blur-xl">
            <div class="flex items-center gap-1">
                <!-- 附加按钮 -->
                <UDropdownMenu :items="attachmentMenuItems">
                    <UButton
                        variant="ghost"
                        icon="i-heroicons-plus-circle"
                        square
                        class="shrink-0 chat-icon-btn text-secondary-content hover:text-accent"
                    />
                </UDropdownMenu>
                
                <!-- 输入框容器 -->
                <div class="chat-input-wrapper">
                    <input
                        ref="messageInputRef"
                        v-model="messageInput"
                        placeholder="在群里说点什么..."
                        autofocus
                        @keydown.enter.exact.prevent="handleSendMessage"
                    />
                </div>
                
                <!-- 发送按钮 -->
                <UButton
                    color="primary"
                    :icon="sending ? 'i-heroicons-stop' : 'i-heroicons-paper-airplane'"
                    :loading="sending"
                    :disabled="!messageInput.trim()"
                    class="shrink-0 chat-send-btn"
                    @click="handleSendMessage"
                />
            </div>
            
            <!-- 文件上传 (使用 UFileUpload) -->
            <UFileUpload
                v-slot="{ open }"
                v-model="selectedFile"
                accept="image/*,.pdf,.doc,.docx,.txt"
                class="hidden"
                @update:model-value="handleFileSelected"
            >
                <button ref="fileUploadTrigger" class="hidden" @click="open()" />
            </UFileUpload>
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
} from "vue"
import { useRoute, navigateTo } from "nuxt/app"
import type { GroupMessageItem } from "~/types/api"
import type { GroupDetailData } from "~/types/client"
import type { ChatMessage as BaseChatMessage, MessageMetadata } from '~/components/chat/ChatMessages.vue'
import { getErrorMessage } from '~/utils/error'

// 群聊消息类型（扩展 ChatMessages 组件的基础类型）
interface GroupChatMessage extends BaseChatMessage {
    metadata?: MessageMetadata & {
        senderName?: string
    }
}


// 页面元数据
definePageMeta({
    title: "群聊",
    requiresAuth: true,
    layout: 'dashboard',
    fullHeight: true // 全高度布局，用于聊天页面
});

const route = useRoute();
const router = useRouter();
const toast = useToast()
const { formatTime: baseFormatTime } = useFormatters();

// 显示消息时间（群聊特殊显示需求）
const displayMessageTime = (date: string | number | Date): string => {
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
import { useGroupsService } from '~/services/groups.service';
import { useGroupMessagesService } from '~/services/group_messages.service';
import { geocoding } from '~/utils/geocoding'
import { type Message } from '~/stores/messages.store';
const groupsService = useGroupsService();
const groupMessagesService = useGroupMessagesService();
const geolocation = useGeolocation();

// 群聊参数
const groupId = route.params.id as string;

// 群聊信息与状态
const groupInfo = ref<GroupDetailData | null>(null)
const messagesList = computed(() => messagesStore.messages.get(groupId) || []);

// 消息输入和状态
const messageInput = ref('');
const sending = ref(false);
const loadingMore = ref(false);
// 文件上传相关
const selectedFile = ref<File | null>(null);
const fileUploadTrigger = ref<HTMLButtonElement | null>(null);
// 消息输入框模板引用（用于程序化聚焦）
const messageInputRef = ref<{ $el: HTMLElement } | null>(null);

// 分页状态
const messageCursor = ref<string | null>(null);
const hasMore = ref(false);

// 聊天状态
const chatStatus = computed(() => {
    if (sending.value) return 'submitted';
    if (loadingMore.value) return 'streaming';
    return 'ready';
});

// 格式化消息为ChatMessages组件需要的格式
const formattedMessages = computed(() => {
    // 按时间正序排列（最新的在底部）
    const sortedMessages = [...messagesList.value].sort((a, b) =>
        new Date(a.sent_at || 0).getTime() - new Date(b.sent_at || 0).getTime()
    )

    return sortedMessages.map((message: GroupMessageItem) => ({
        id: message.id || String(Math.random()),
        role: (isOwn(message) ? 'user' : 'assistant') as 'user' | 'assistant' | 'system',
        content: message.content || '', // 消息内容
        parts: [{ type: 'text' as const, text: message.content || '' }],
        timestamp: message.sent_at || new Date().toISOString(), // 时间戳
        metadata: {
            senderName: message.sender?.nickname || '未知用户',
            senderAvatar: message.sender?.avatar_url,
            senderId: message.sender?.user_id,
            timestamp: message.sent_at || new Date().toISOString()
        }
    }));
});

// 处理头像点击 - 跳转到用户资料页
const handleAvatarClick = (message: GroupChatMessage) => {
    const senderId = message.metadata?.senderId;
    if (senderId) {
        navigateTo(`/users/${senderId}`);
    }
};

// 获取消息文本内容
const getMessageText = (message: GroupChatMessage) => {
    // 从 parts 中提取文本内容
    if (message.parts && message.parts.length > 0) {
        return message.parts[0].text || '';
    }
    return '';
};

// 搜索相关
const showSearch = ref(false);
const searchQuery = ref("");

const searchResults = computed(() => {
    if (!searchQuery.value.trim()) return [];
    const keyword = searchQuery.value.toLowerCase();
    return messagesList.value.filter((message: GroupMessageItem) =>
        message.content?.toLowerCase().includes(keyword),
    );
});

const displayMessages = computed(() => {
    if (searchQuery.value.trim()) {
        return searchResults.value;
    }
    return messagesList.value;
});

const hasMoreMessages = computed<boolean>(() => hasMore.value);
const focusComposer = () => {
    // 使用 Nuxt UI 组件的模板引用来聚焦
    messageInputRef.value?.$el?.querySelector('input')?.focus();
};

// 使用水合安全的 store 访问
const authStore = useAuthStore();
const messagesStore = useMessagesStore();

// 是否本人消息
const isOwn = (m: GroupMessageItem) =>
    String(m.sender?.user_id) === String(authStore.user?.user_id ?? "");

// 附件菜单项
const attachmentMenuItems = computed(() => [
    [{
        label: '发送图片',
        icon: 'i-heroicons-photo',
        click: () => triggerFileUpload()
    }, {
        label: '发送文件',
        icon: 'i-heroicons-document',
        click: () => triggerFileUpload()
    }],
    [{
        label: '分享位置',
        icon: 'i-heroicons-map-pin',
        click: () => onSendLocation()
    }]
]);

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
        // 获取群组基本信息（包含成员数量）
        try {
            const coords = geolocation.state.coordinates
            const groupData = await groupsService.openTheDoor({ 
                id: groupId,
                latitude: coords?.latitude || 35.154921,
                longitude: coords?.longitude || 112.68253
            })
            groupInfo.value = {
                id: groupData.id,
                name: groupData.name,
                description: groupData.description,
                avatar_url: groupData.avatar_url,
                members_count: groupData.members_count,
                has_password: groupData.has_password,
                tags: groupData.tags,
                geo: groupData.geo
            } as any
        } catch (_) {
            // 无法获取群组信息时使用默认值
            groupInfo.value = null
        }

        // 获取群聊消息
        const response = await groupMessagesService.getGroupMessages({ id: groupId, limit: 20 });
        const messages: Message[] = response.items.map(item => ({
            ...item,
            is_recalled: item.is_recalled || false
        }));
        messagesStore.setMessages(groupId, messages);
        
        // 保存分页状态
        hasMore.value = response.pagination.has_more;
        messageCursor.value = response.pagination.next_cursor ?? null;
    } catch (error) {
        console.error("初始化群聊失败:", error);
        toast.add({
            title: "群聊初始化失败",
            description: "无法加载群聊信息，请刷新页面重试",
            color: "error",
        });
    }
};

// 发送群聊消息的通用方法
// 注意：不立即显示消息，等待 WebSocket 推送后再显示
// 这样可以确保显示的内容与服务端一致（如媒体 URL 转换等）
const sendGroupMessage = async (type: 'text' | 'image' | 'file', content: string, extra?: { file_name?: string; file_size?: number }) => {
    await groupMessagesService.send({
        id: groupId,
        type,
        content,
        ...extra
    });
};

// 处理消息发送（ChatPrompt组件的submit事件）
const handleSendMessage = async () => {
    const content = messageInput.value.trim();
    if (!content || sending.value) return;
    
    sending.value = true;
    try {
        await sendGroupMessage('text', content);
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
        await sendGroupMessage('text', content);
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
    fileUploadTrigger.value?.click();
};

// 处理文件选择 (UFileUpload 回调)
const handleFileSelected = (file: File | null | undefined) => {
    if (file) {
        onUploadFile(file);
        // 重置选择
        selectedFile.value = null;
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
        await sendGroupMessage('file', `[文件] ${file.name}`, {
            file_name: file.name,
            file_size: file.size
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
        
        const geolocation = useGeolocation();
        const position = await geolocation.getPosition();
        
        if (!position) {
            // getPosition 内部已经显示了 Toast 提示
            return;
        }
        
        // 获取位置名称
        let locationName = '当前位置';
        try {
            locationName = await geocoding.getLocationName(
                position.coords.latitude,
                position.coords.longitude
            );
        } catch {
            // 使用默认名称
        }
        
        const locationContent = `📍 ${locationName}`;
        
        await sendGroupMessage('text', locationContent);
        
        toast.add({
            title: "位置已发送",
            description: "您的当前位置已分享到群聊",
            color: "success",
        });
        
    } catch (error: unknown) {
        console.error("发送位置失败:", error);
        
        let errorMessage = "获取位置失败，请重试";
        const errorMsg = getErrorMessage(error);
        if (errorMsg.includes("权限")) {
            errorMessage = "位置权限被拒绝，请在浏览器设置中允许位置访问";
        } else if (errorMsg.includes("超时")) {
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
    if (loadingMore.value || !hasMore.value || !messageCursor.value) return;
    loadingMore.value = true;
    try {
        const response = await groupMessagesService.getGroupMessages({ 
            id: groupId,
            limit: 20,
            cursor: messageCursor.value
        });
        const messages: Message[] = response.items.map(item => ({
            ...item,
            is_recalled: item.is_recalled || false
        }));
        messagesStore.prependMessages(groupId, messages);
        
        // 更新分页状态
        hasMore.value = response.pagination.has_more;
        messageCursor.value = response.pagination.next_cursor ?? null;
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

// 注意：scrollToBottom 方法已由 ChatMessages 组件自动处理

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
        await groupsService.leaveGroup({ id: groupId });
        toast.add({
            title: "已退出群聊",
            description: "您已成功退出该群聊",
            color: "success",
        });
        router.back();
    } catch (error) {
        toast.add({
            title: "退出失败",
            description: "请重试",
            color: "error",
        });
    }
};

// WebSocket 实时消息
const { on: wsOn, subscribeToGroupChats, unsubscribeFromGroupChats } = useGlobalWebSocket();

// ChatMessageNew 事件的 payload 类型
interface ChatMessageNewPayload {
  id: string;
  chat_type: 'private_chat' | 'group_chat';
  chat_id: string;
  sender: {
    user_id: number;
    nickname: string;
    avatar_url?: string;
    user_type: string;
  };
  message_type: 'text' | 'image' | 'video' | 'audio' | 'file' | 'system';
  content?: string;
  reply_to?: string;
  mentions?: number[];
  media_url?: string;
  thumbnail_url?: string;
  file_name?: string;
  file_size?: number;
  duration?: number;
  width?: number;
  height?: number;
  sent_at: string;
}

let messageCleanup: (() => void) | null = null;

// 初始化
onMounted(async () => {
    try {
        await initGroupChat();
        
        // 订阅群聊消息
        subscribeToGroupChats([groupId]);
        
        // 监听 ChatMessageNew 事件 - 实时接收新消息（包括自己发送的）
        // 所有消息都通过 WebSocket 推送后才显示，确保内容与服务端一致
        messageCleanup = wsOn<ChatMessageNewPayload>('ChatMessageNew', (data) => {
            // 只处理当前群聊的消息
            if (data.chat_type === 'group_chat' && data.chat_id === groupId) {
                // 构造消息对象并添加到 store
                const newMessage: Message = {
                    id: data.id,
                    type: data.message_type,
                    content: data.content || '',
                    sender: {
                        user_id: data.sender.user_id,
                        nickname: data.sender.nickname,
                        avatar_url: data.sender.avatar_url,
                        user_type: data.sender.user_type as 'temp' | 'normal' | 'admin' | 'system'
                    },
                    sent_at: data.sent_at,
                    is_recalled: false,
                    reply_to: data.reply_to,
                    mentions: data.mentions,
                    media_url: data.media_url,
                    thumbnail_url: data.thumbnail_url,
                    file_name: data.file_name,
                    file_size: data.file_size,
                    duration: data.duration,
                    width: data.width,
                    height: data.height
                };
                
                messagesStore.appendMessage(groupId, newMessage);
                
                if (import.meta.dev) {
                    console.log('[WebSocket] 收到新群聊消息:', data.content);
                }
            }
        });
        
    } catch (error: unknown) {
        console.error("[GroupChat] Failed to initialize:", error);
        toast.add({
            title: "群聊初始化失败",
            description: getErrorMessage(error, "无法加载群聊信息，请刷新页面重试"),
            color: "error",
        });
    }
});

// 页面离开时清理
onUnmounted(() => {
    // 取消订阅群聊消息
    unsubscribeFromGroupChats([groupId]);
    
    // 清理 WebSocket 事件监听
    if (messageCleanup) {
        messageCleanup();
    }
});
</script>
