<template>
    <div class="-mx-4 sm:-mx-6 flex h-full min-h-0 flex-col overflow-hidden bg-white dark:bg-neutral-900">
        <!-- Header (Fixed) -->
        <div class="shrink-0 border-b border-neutral-200 dark:border-neutral-800">
            <GroupChatHeader
                v-if="group"
                :group="group"
                :loading="loading"
                @back="handleBack"
                @info="navigateToGroupProfile"
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
                v-if="loading && !group"
                title="加载群聊中..."
                class="absolute inset-0"
            />

            <ChatMessages
                v-else
                ref="messagesRef"
                :messages="messages"
                :current-user-id="authStore.user?.user_id"
                :loading-more="loadingMore"
                :has-more="hasMore"
                :is-group-chat="true"
                class="h-full"
                @load-more="loadHistory"
            >
                <template #empty>
                    <div
                        class="flex h-full items-center justify-center px-6 py-12"
                    >
                        <UEmpty
                            icon="i-lucide-messages-square"
                            title="暂无消息"
                            description="发一条消息，开始讨论"
                        />
                    </div>
                </template>
            </ChatMessages>
        </div>

        <!-- Footer (Fixed Input) -->
        <div class="shrink-0 border-t border-neutral-200 dark:border-neutral-800 bg-neutral-50/50 dark:bg-neutral-900/50">
            <ChatInput v-if="group" :loading="sending" @send="handleSend" />
        </div>
    </div>
</template>

<script setup lang="ts">
import { useGroupsService } from "~/services/groups.service";
import { useGroupMessagesService } from "~/services/group_messages.service";
import type {
    GroupItem,
    GroupMessageItem,
    SendMessageType,
} from "~/types/api";
import GroupChatHeader from "~/components/chat/GroupChatHeader.vue";
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
const groupsService = useGroupsService();
const messageService = useGroupMessagesService();
const geolocation = useGeolocation();
const feedback = useFeedbackStore();

const groupId = route.params.id as string;
const group = ref<GroupItem | null>(null);

const loading = ref(true);
const sending = ref(false);
const loadingMore = ref(false);
const hasMore = ref(true);
const cursor = ref<string | undefined>(undefined);
const rawMessages = ref<GroupMessageItem[]>([]);
const messagesRef = ref<InstanceType<typeof ChatMessages> | null>(null);

// 转换消息格式以适配 ChatMessages 组件
const messages = computed<ChatMessage[]>(() => {
    const currentUserId = authStore.user?.user_id;
    return rawMessages.value.map((msg) => ({
        ...msg,
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

// WebSocket 订阅与监听
const { on, subscribeToGroupChats, unsubscribeFromGroupChats } = useWebSocket();

on("GroupMessageNew", (payload) => {
    // 确保是当前群组的消息
    if (
        payload.group_chat_id === groupId &&
        !messages.value.some((m) => m.message_id === payload.message_id)
    ) {
        // 转换为 GroupMessageItem
        const newItem: GroupMessageItem = {
            message_id: payload.message_id,
            type: payload.type,
            content: payload.content,
            sender: payload.sender,
            sent_at: payload.sent_at,
            media: payload.media,
            
            is_recalled: false,
            reply_to_message_id: payload.reply_to_message_id,
            mentions: payload.mentions,
        };

        rawMessages.value.push(newItem);
        nextTick(() => messagesRef.value?.scrollToBottom());
    }
});

// 初始化
onMounted(async () => {
    // 订阅当前群组
    subscribeToGroupChats([groupId]);
    await initChat();
});

// 清理：取消订阅
onBeforeUnmount(() => {
    unsubscribeFromGroupChats([groupId]);
});

async function initChat() {
    loading.value = true;
    try {
        const coords = await geolocation.getCoordinates();
        // 获取群组详情
        group.value = await groupsService.openTheDoor({
            group_chat_id: groupId,
            reference_point: {
                latitude: coords.latitude,
                longitude: coords.longitude,
            },
        });

        // 加载历史消息
        await loadHistory(true);

        // 标记已读
        setTimeout(() => {
            if (groupId) messageService.markAsRead({ group_chat_id: groupId });
        }, 500);
    } catch (e) {
        feedback.error("无法加载群聊");
        router.push("/chat");
    } finally {
        loading.value = false;
    }
}

async function loadHistory(refresh = false) {
    if (!groupId) return;
    if (refresh) {
        cursor.value = undefined;
        hasMore.value = true;
        rawMessages.value = [];
    }

    if (!hasMore.value || loadingMore.value) return;

    loadingMore.value = true;
    try {
        const res = await messageService.getGroupMessages({
            group_chat_id: groupId,
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
}) {
    if (!groupId) return;
    sending.value = true;
    try {
        await messageService.send({
            group_chat_id: groupId,
            type: payload.type,
            content: payload.content,
            file_id: payload.fileId,
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

function navigateToGroupProfile() {
    router.push(`/groups/${groupId}`);
}
</script>
