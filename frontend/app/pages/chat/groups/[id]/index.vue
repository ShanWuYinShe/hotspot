<template>
    <div class="flex flex-col h-full">
        <!-- 群聊头部 -->
        <GroupChatHeader
            :group-info="groupInfo"
            @go-back="router.back()"
            @toggle-search="showSearch = !showSearch"
            @view-members="showGroupMembers"
            @view-info="showGroupInfo"
            @view-settings="showGroupSettings"
            @leave-group="leaveGroupChat"
        />

        <!-- 搜索栏 -->
        <Transition
            enter-active-class="transition-all duration-200 ease-out"
            enter-from-class="opacity-0 -translate-y-2"
            enter-to-class="opacity-100 translate-y-0"
            leave-active-class="transition-all duration-150 ease-in"
            leave-from-class="opacity-100 translate-y-0"
            leave-to-class="opacity-0 -translate-y-2"
        >
            <div
                v-if="showSearch"
                class="z-20 absolute top-[60px] w-full"
            >
                <UCard variant="subtle" data-hs-surface="chat-search">
                    <UInput
                        v-model="searchQuery"
                        placeholder="搜索群聊记录..."
                        icon="i-lucide-search"
                        size="md"
                        class="w-full"
                        :autofocus="true"
                        @keyup.escape="showSearch = false"
                    >
                        <template #trailing>
                            <UButton
                                variant="ghost"
                                color="neutral"
                                size="sm"
                                icon="i-lucide-x"
                                :square="true"
                                @click="searchQuery = ''"
                            />
                        </template>
                    </UInput>
                    <UAlert
                        v-if="searchQuery && searchResults.length === 0"
                        color="neutral"
                        variant="soft"
                        icon="i-lucide-search-x"
                        title="没有找到相关消息"
                        class="mt-2"
                    />
                    <UAlert
                        v-else-if="searchQuery"
                        color="neutral"
                        variant="soft"
                        icon="i-lucide-search"
                        :title="`找到 ${searchResults.length} 条相关消息`"
                        class="mt-2"
                    />
                </UCard>
            </div>
        </Transition>

        <!-- 消息列表 -->
        <div class="flex-1 min-h-0 relative overflow-hidden">
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
                <UEmpty icon="i-lucide-messages-square" title="暂无群聊消息" description="开始在群里发送第一条消息吧" />
            </div>

            <!-- 自定义 ChatMessages 组件 -->
            <ChatMessages v-else :messages="formattedMessages" :status="chatStatus" :should-auto-scroll="true"
                :should-scroll-to-bottom="true" :is-group-chat="true" :user="{
                    side: 'right',
                    variant: 'soft',
                    avatar: { fallback: currentUser?.nickname?.charAt(0) || 'U', src: currentUser?.avatar_url }
                }" class="h-full px-4" @avatar-click="handleAvatarClick">
                <template #content="{ message }: { message: BaseChatMessage }">
                    <!-- 消息内容 -->
                    <div class="text-sm leading-relaxed">
                        {{ getMessageText(message) }}
                    </div>
                </template>
            </ChatMessages>
        </div>

        <!-- 消息输入框 -->
        <div class="shrink-0 z-10">
             <ChatInput
                :loading="sending"
                @send-message="onSend"
                @send-file="onUploadFile"
                @send-image="onUploadFile"
                @send-location="onSendLocation"
                @typing="handleTyping"
            />
        </div>
    </div>
</template>

<script setup lang="ts">
import { promiseTimeout, useTimeoutFn } from '@vueuse/core'
import type { GroupMessageItem } from "~/types/api"
import type { GroupSearchItem } from "~/types/api"
import type { ClientMessageType } from '~/types/client'
import type { ChatMessage as BaseChatMessage } from '~/components/chat/ChatMessages.vue'
import type { UserDetailData } from '~/types/client'
import { getErrorMessage } from '~/utils/error'
import { useHydratedStore } from '~/composables/useHydratedStore'

type GroupChatMessage = BaseChatMessage & { parts?: { type: string; text?: string }[] }


// 页面元数据
definePageMeta({
    title: "群聊",
    requiresAuth: true,
    layout: 'dashboard',
    showNavbarBack: false,
    fullHeight: true // 全高度布局，用于聊天页面
});

const route = useRoute();
const router = useRouter();
const feedback = useFeedbackStore()
const { withErrorHandling } = useErrorHandler()
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
const groupId = String(route.params.id);

// store（页面只读 store，副作用/落地由全局 WS 插件负责）
const authStore = useAuthStore();
const messagesStore = useMessagesStore();
const currentUser = useHydratedStore(authStore, 'user', null as UserDetailData | null)

// 群聊信息与状态
const groupInfo = ref<GroupSearchItem | null>(null)
const messagesList = computed(() => messagesStore.getMessages(groupId));

// 消息输入和状态
const sending = ref(false);
const loadingMore = ref(false);

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
        id: message.id || `${message.sent_at || ''}:${message.sender?.user_id ?? 0}`,
        role: (isOwn(message) ? 'user' : 'assistant') as 'user' | 'assistant' | 'system',
        content: message.media_url || message.content || '', // 优先使用 media_url（后端转换后的URL）
        parts: [{ type: 'text' as const, text: message.content || '' }],
        timestamp: message.sent_at || new Date().toISOString(), // 时间戳
        type: message.type as ClientMessageType, // 消息类型
        // 传递媒体消息字段（后端已转换）
        fileName: message.file_name,
        fileSize: message.file_size,
        thumbnail: message.thumbnail_url,
        duration: message.duration,
        width: message.width,
        height: message.height,
        metadata: {
            senderName: message.sender?.nickname || '未知用户',
            senderAvatar: message.sender?.avatar_url,
            senderId: message.sender?.user_id
        }
    }));
});

// 处理头像点击 - 跳转到用户资料页
const handleAvatarClick = (message: BaseChatMessage) => {
    const senderId = message.metadata?.senderId;
    if (senderId) {
        navigateTo(`/users/${senderId}`);
    }
};

// 获取消息文本内容
const getMessageText = (message: BaseChatMessage) => {
    // 从 parts 中提取文本内容
    const parts = (message as GroupChatMessage).parts
    if (parts && parts.length > 0) return parts[0]?.text || ''
    return String(message.content || '')
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

const hasMoreMessages = computed<boolean>(() => hasMore.value);

// 是否本人消息
const isOwn = (m: GroupMessageItem) =>
    String(m.sender?.user_id) === String(currentUser.value?.user_id ?? "");

// ============================================================================
// 已读：页面不监听 WS 消息本身，只根据 store 的“最新消息”触发 API 已读
// ============================================================================

const lastReadUpTo = ref<string | null>(null)
watch(
    () => {
        const list = messagesStore.getMessages(groupId)
        const last = list[list.length - 1]
        return last ? { id: last.id, senderId: last.sender.user_id } : null
    },
    (last) => {
        if (!last) return
        if (!last.id) return
        if (lastReadUpTo.value === last.id) return
        if (messagesStore.currentConversationId !== groupId) return

        const me = currentUser.value?.user_id
        if (me != null && last.senderId === me) {
            // 自己的消息不需要 markAsRead
            lastReadUpTo.value = last.id
            return
        }

        lastReadUpTo.value = last.id
        void groupMessagesService
            .markAsRead({ id: groupId, up_to: last.id })
            .then(() => messagesStore.updateUnreadCount(groupId, 0))
            .catch((e: unknown) => {
                if (import.meta.dev) console.warn('[GroupChat] markAsRead (watch) failed:', e);
            })
    }
)

// 初始化群聊数据
const initGroupChat = async (): Promise<void> => {
    await withErrorHandling(async () => {
        // 获取群组基本信息（包含成员数量）
        try {
            const coords = geolocation.state.coordinates
            const groupData = await groupsService.openTheDoor({
                id: groupId,
                latitude: coords?.latitude || 35.154921,
                longitude: coords?.longitude || 112.68253
            })
            groupInfo.value = groupData
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

        // ✅ upsert 会话基础信息（保证 store 有会话，且 WS 增量/typing/unread 能正确工作）
        const last = messages[messages.length - 1]
        const lastActivityAt = last?.sent_at || new Date().toISOString()
        messagesStore.upsertConversation({
            chat_id: groupId,
            chat_type: 'group',
            name: groupInfo.value?.name || '群聊',
            avatar_url: groupInfo.value?.avatar_url,
            unread_count: 0,
            last_message: last ? {
                id: last.id,
                content: last.content,
                type: last.type,
                sender_id: last.sender.user_id,
                sent_at: last.sent_at
            } : undefined,
            last_activity_at: lastActivityAt
        })

        // 进入会话：本地置为当前会话并清零未读（UI 立即同步）
        messagesStore.setCurrentConversation(groupId);

        // ✅ 权威未读清零：进入群聊后标记为已读（避免“看过仍未读”）
        const lastMessageId = messages[messages.length - 1]?.id;
        if (lastMessageId) {
            try {
                await groupMessagesService.markAsRead({ id: groupId, up_to: lastMessageId });
                messagesStore.updateUnreadCount(groupId, 0);
            } catch (e: unknown) {
                // 忽略：网络抖动不应阻塞页面渲染；下次进入/刷新再尝试
                if (import.meta.dev) console.warn('[GroupChat] markAsRead failed:', e);
            }
        }

        // 保存分页状态
        hasMore.value = response.pagination.has_more;
        messageCursor.value = response.pagination.next_cursor ?? null;
    }, {
        title: '群聊初始化失败',
        fallbackMessage: '无法加载群聊信息，请刷新页面重试'
    })
};

// 发送群聊消息的通用方法
// 遵循 @frontend_rule.mdc 消息发送规范：
// 1. API 仅负责发送消息，不处理本地数据更新
// 2. 消息上屏完全依赖 WebSocket 推送的 ChatMessageNew 事件
// 3. 确保显示的数据（如媒体URL、时间戳）与服务端完全一致
const sendGroupMessage = async (type: 'text' | 'image' | 'file', content: string, extra?: { file_name?: string; file_size?: number }) => {
    await groupMessagesService.send({
        id: groupId,
        type,
        content,
        ...extra
    });
};

// 发送群聊消息
const onSend = async (content: string) => {
    if (!content.trim() || sending.value) return;

    await withErrorHandling(async () => {
        await sendGroupMessage('text', content);
    }, {
        title: '发送失败',
        loading: sending
    })
};

// 文件上传处理
const onUploadFile = async (file: File) => {
    const maxSize = 10 * 1024 * 1024; // 10MB
    if (file.size > maxSize) {
        feedback.error("文件过大", "文件大小不能超过10MB")
        return;
    }

    const allowedTypes = [
        'image/jpeg', 'image/png', 'image/gif', 'image/webp',
        'application/pdf', 'text/plain',
        'application/msword',
        'application/vnd.openxmlformats-officedocument.wordprocessingml.document'
    ];

    if (!allowedTypes.includes(file.type)) {
        feedback.error("不支持的文件类型", "仅支持图片、PDF、文档等常见格式")
        return;
    }

    try {
        await withErrorHandling(async () => {
            await promiseTimeout(1000)

            // 发送文件消息
            await sendGroupMessage('file', `[文件] ${file.name}`, {
                file_name: file.name,
                file_size: file.size
            });

            feedback.success("上传成功", `文件 "${file.name}" 已发送到群聊`)
        }, {
            title: '上传失败',
            fallbackMessage: '请检查网络连接后重试',
            loading: sending
        })
    } catch (error) {
        // withErrorHandling 已经处理了错误，这里只需要捕获防止崩溃
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

        feedback.success("位置已发送", "您的当前位置已分享到群聊")

    } catch (error: unknown) {
        console.error("发送位置失败:", error);

        let errorMessage = "获取位置失败，请重试";
        const errorMsg = getErrorMessage(error);
        if (errorMsg.includes("权限")) {
            errorMessage = "位置权限被拒绝，请在浏览器设置中允许位置访问";
        } else if (errorMsg.includes("超时")) {
            errorMessage = "获取位置超时，请检查GPS是否开启";
        }

        feedback.error("发送失败", errorMessage)
    } finally {
        sending.value = false;
    }
};

// 加载更多消息
const loadMoreMessages = async (): Promise<void> => {
    if (loadingMore.value || !hasMore.value || !messageCursor.value) return;

    await withErrorHandling(async () => {
        const response = await groupMessagesService.getGroupMessages({
            id: groupId,
            limit: 20,
            cursor: messageCursor.value || undefined
        });
        const messages: Message[] = response.items.map(item => ({
            ...item,
            is_recalled: item.is_recalled || false
        }));
        messagesStore.prependMessages(groupId, messages);

        // 更新分页状态
        hasMore.value = response.pagination.has_more;
        messageCursor.value = response.pagination.next_cursor ?? null;
    }, {
        title: '加载失败',
        loading: loadingMore
    })
};

// 注意：scrollToBottom 方法已由 ChatMessages 组件自动处理

// 群聊操作
const showGroupMembers = (): void => {
    navigateTo(`/groups/${groupId}`);
};

const showGroupInfo = (): void => {
    navigateTo(`/groups/${groupId}`)
};

const showGroupSettings = (): void => {
    navigateTo(`/groups/${groupId}/settings`)
};

// 退出群聊
const leaveGroupChat = async (): Promise<void> => {
    await withErrorHandling(async () => {
        await groupsService.leaveGroup({ id: groupId });
        feedback.success("已退出群聊", "您已成功退出该群聊")
        router.back();
    }, {
        title: '退出失败'
    })
};

// 处理正在输入
const isTyping = ref(false)
const sendTypingStatus = (isTypingStatus: boolean) => {
    // 群聊 typing 事件：服务端会广播给同群其他成员（由全局 WS 插件落到 store.typing）
    wsSend('Typing', {
        chat_type: 'group_chat' as const,
        chat_id: groupId,
        is_typing: isTypingStatus
    })
}
const { start: startStopTyping, stop: stopStopTyping } = useTimeoutFn(() => {
    isTyping.value = false
    sendTypingStatus(false)
}, 3000, { immediate: false })

const handleTyping = (isTypingNow: boolean) => {
    if (isTypingNow) {
        if (!isTyping.value) {
            isTyping.value = true
            sendTypingStatus(true)
        }
        startStopTyping()
        return
    }
    if (isTyping.value) {
        stopStopTyping()
        isTyping.value = false
        sendTypingStatus(false)
    }
}

// WebSocket 实时消息
const { send: wsSend, useSubscription } = useWebSocket();

// 初始化
onMounted(async () => {
    try {
        await initGroupChat();

        // 订阅群聊消息（必须，否则服务端不会推送 group_chat 消息）
        useSubscription([{ source_type: 'group_chat', source_ids: [groupId] }])

    } catch (error: unknown) {
        console.error("[GroupChat] Failed to initialize:", error);
        feedback.error("群聊初始化失败", getErrorMessage(error, "无法加载群聊信息，请刷新页面重试"))
    }
});

// 页面离开时清理
onUnmounted(() => {
    // 离开会话：让后续新消息能正确计入未读
    messagesStore.setCurrentConversation(null)
    stopStopTyping()
    if (isTyping.value) sendTypingStatus(false)
});
</script>