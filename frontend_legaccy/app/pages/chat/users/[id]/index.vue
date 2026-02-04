<template>
    <div class="-mx-4 sm:-mx-6 flex h-full min-h-0 flex-col overflow-hidden bg-white dark:bg-neutral-900">
        <!-- Header (Fixed) -->
        <div class="shrink-0 border-b border-neutral-200 dark:border-neutral-800">
            <PrivateChatHeader
                v-if="chatDetail"
                :chat="chatDetail"
                :loading="loading"
                @go-back="handleBack"
                @view-profile="navigateToProfile(chatDetail.other_user.user_id)"
            />
            <div v-else class="flex h-14 items-center justify-between px-4">
                <USkeleton class="h-8 w-32" />
                <div class="flex items-center gap-2">
                    <USkeleton class="h-9 w-9" />
                    <USkeleton class="h-9 w-9" />
                </div>
            </div>
        </div>

        <!-- Messages (Scrollable) -->
        <div class="relative flex-1 min-h-0">
            <LoadingState
                v-if="loading && !chatDetail"
                title="加载会话中..."
                class="absolute inset-0"
            />

            <ChatMessages
                v-else
                ref="messagesRef"
                :messages="messages"
                :loading-more="loadingMore"
                :has-more="hasMore"
                :user="{ avatar: { src: authStore.user?.avatar_url, fallback: authStore.user?.nickname?.[0] } }"
                class="h-full"
                @load-more="() => loadHistory(false)"
                @avatar-click="handleAvatarClick"
            >
                <template #empty>
                    <div
                        class="flex h-full items-center justify-center px-6 py-12"
                    >
                        <UEmpty
                            icon="i-lucide-messages-square"
                            title="暂无消息"
                            description="发一条消息，开始对话"
                        />
                    </div>
                </template>
            </ChatMessages>
        </div>

        <!-- Footer (Fixed Input) -->
        <div class="shrink-0 border-t border-neutral-200 dark:border-neutral-800 bg-neutral-50/50 dark:bg-neutral-900/50">
            <ChatInput
                v-if="chatDetail"
                :loading="sending"
                @send="handleSend"
            />
        </div>
    </div>
</template>

<script setup lang="ts">
import { usePrivateChatsService } from "~/services/private_chats.service";
import { usePrivateMessagesService } from "~/services/private_messages.service";
import { useUsersService } from "~/services/users.service";
import { useUploadService } from "~/services/upload.service";
import { isChatNotFoundCode, extractApiErrorSnapshot } from "~/utils/apiError";
import type {
    PrivateChatDetailData,
    PrivateMessageItem,
    SendMessageType,
} from "~/types/api";
import PrivateChatHeader from "~/components/chat/PrivateChatHeader.vue";
import ChatMessages, {
    type ChatMessage,
} from "~/components/chat/ChatMessages.vue";
import ChatInput from "~/components/chat/ChatInput.vue";
import LoadingState from "~/components/common/LoadingState.vue";

definePageMeta({
    layout: "dashboard",
    requiresAuth: true,
});

const route = useRoute();
const router = useRouter();
const authStore = useAuthStore();
const privateChatsService = usePrivateChatsService();
const privateMessagesService = usePrivateMessagesService();
const usersService = useUsersService();
const uploadService = useUploadService();
const feedback = useFeedbackStore();
const geolocation = useGeolocation();

const targetUserId = parseInt(route.params.id as string);
const chatDetail = ref<PrivateChatDetailData | null>(null);
const isNewChat = ref(false);

const loading = ref(true);
const sending = ref(false);
const loadingMore = ref(false);
const hasMore = ref(true);
const cursor = ref<string | undefined>(undefined);
const rawMessages = ref<PrivateMessageItem[]>([]);
const messagesRef = ref<InstanceType<typeof ChatMessages> | null>(null);

// 转换消息格式以适配 ChatMessages 组件
const messages = computed<ChatMessage[]>(() => {
    const currentUserId = authStore.user?.user_id;
    return rawMessages.value.map((msg) => ({
        ...msg,
        // ChatMessages 组件用 `id` 做 DOM key / data-message-id
        id: msg.message_id,
        role:
            msg.sender.user_id === currentUserId
                ? ("user" as const)
                : ("assistant" as const),
        createdAt: msg.sent_at,
        timestamp: msg.sent_at,
        metadata: {
            senderId: msg.sender.user_id,
            senderName: msg.sender.nickname,
            senderAvatar: msg.sender.avatar_url,
            timestamp: msg.sent_at,
        },
    }));
});

// WebSocket 监听
const { on } = useWebSocket();

on("PrivateMessageNew", (payload) => {
    const currentUserId = authStore.user?.user_id;
    
    // 确保是当前会话的消息 (User-Centric Matching)
    // 1. 我发给对方 (Sync)
    // 2. 对方发给我 (Receive)
    const isRelevant = 
        (payload.sender.user_id === currentUserId && payload.target_user_id === targetUserId) ||
        (payload.sender.user_id === targetUserId && payload.target_user_id === currentUserId);

    if (isRelevant && !messages.value.some((m) => m.message_id === payload.message_id)) {
        const newItem: PrivateMessageItem = {
            message_id: payload.message_id,
            type: payload.type,
            content: payload.content,
            sender: payload.sender,
            sent_at: payload.sent_at,
            media: payload.media,
            reply_to_message_id: payload.reply_to_message_id,
            reply_to: payload.reply_to,
            is_recalled: false,
        };

        rawMessages.value.push(newItem);
        nextTick(() => messagesRef.value?.scrollToBottom());

        // 如果是对方发的，标记已读
        if (payload.sender.user_id === targetUserId) {
            privateMessagesService.markPrivateMessagesAsRead({
                target_user_id: targetUserId,
            });
        }
    }
});

// 初始化
onMounted(async () => {
    await initChat();
});

async function initChat() {
    loading.value = true;
    isNewChat.value = false;
    try {
        try {
            chatDetail.value = await privateChatsService.openTheDoor({
                target_user_id: targetUserId,
            });
        } catch (e: unknown) {
            const snapshot = extractApiErrorSnapshot(e);
            if (isChatNotFoundCode(snapshot.code)) {
                isNewChat.value = true;
                // 会话不存在（新会话），获取用户信息以渲染空状态
                // 注意：这里需要地理位置是因为 usersService.openTheDoor 接口要求 reference_point
                const coords = await geolocation.getCoordinates();
                const user = await usersService.openTheDoor({
                    user_id: targetUserId,
                    reference_point: {
                        latitude: coords.latitude,
                        longitude: coords.longitude,
                    },
                });

                // 构造临时会话详情
                chatDetail.value = {
                    other_user: user,
                    unread_count: 0,
                    is_muted: false,
                    is_pinned: false,
                    last_activity_at: new Date().toISOString(),
                    created_at: new Date().toISOString(),
                    can_message: true,
                };
            } else {
                throw e;
            }
        }

        // 2. 加载历史消息 (仅当非新会话时)
        if (chatDetail.value && !isNewChat.value) {
            await loadHistory(true);

            // 3. 标记已读 (延迟一下确保加载完成)
            setTimeout(() => {
                privateMessagesService
                    .markPrivateMessagesAsRead({
                        target_user_id: targetUserId,
                    })
                    .catch(() => {});
            }, 500);
        }
    } catch (e) {
        feedback.error("无法加载会话");
        router.push("/chat");
    } finally {
        loading.value = false;
    }
}

async function loadHistory(refresh = false) {
    if (isNewChat.value) return;
    
    if (refresh) {
        cursor.value = undefined;
        hasMore.value = true;
        rawMessages.value = [];
    }

    if (!hasMore.value || loadingMore.value) return;

    loadingMore.value = true;
    try {
        const res = await privateMessagesService.getPrivateMessages({
            target_user_id: targetUserId,
            limit: 20,
            cursor: cursor.value,
        });

        const newMessages = [...res.items].reverse();
        if (refresh) {
            rawMessages.value = newMessages;
        } else {
            rawMessages.value = [...newMessages, ...rawMessages.value];
        }

        cursor.value = res.pagination.next_cursor;
        hasMore.value = res.pagination.has_more;

        if (refresh) {
            nextTick(() => messagesRef.value?.scrollToBottom());
        }
    } catch (e) {
        // error
    } finally {
        loadingMore.value = false;
    }
}

async function handleSend(payload: {
    content: string;
    fileId?: string;
    type: SendMessageType;
    file?: File;
}) {
    sending.value = true;
    try {
        let fileId = payload.fileId;

        if (payload.file && !fileId) {
            const category = payload.type === 'image' ? 'chat_image' : 'chat_file';
            const uploadResult = await uploadService.uploadFile(payload.file, category);
            fileId = uploadResult.file_id;
        }

        await privateMessagesService.send({
            target_user_id: targetUserId,
            type: payload.type,
            content: payload.content,
            file_id: fileId,
        });
        // 成功发送后不直接插入消息，等待 WebSocket 推送保持一致性
    } catch (e) {
        feedback.error("发送失败");
    } finally {
        sending.value = false;
    }
}

function handleBack() {
    router.push("/chat");
}

function navigateToProfile(userId: number | undefined) {
    if (userId) {
        router.push(`/users/${userId}`);
    }
}

function handleAvatarClick(message: ChatMessage) {
    const senderId = message.metadata?.senderId;
    if (senderId) {
        navigateToProfile(Number(senderId));
    }
}
</script>
