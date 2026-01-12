<template>
    <div
        class="lsyb-card-interactive group p-3"
        @click="openConversation"
    >
        <div class="relative shrink-0">
            <UAvatar
                    v-if="conversation.type === 'user'"
                    :src="conversation.avatar"
                    :alt="conversation.name"
                    size="lg"
                    :chip="
                        isOnline
                            ? {
                                  color: 'success',
                                  position: 'bottom-right',
                                  size: '2xs',
                              }
                            : undefined
                    "
                />
                <UAvatar
                    v-else
                    :src="conversation.avatar"
                    :alt="conversation.name"
                    size="lg"
                />
            </div>

        <div class="flex-1 min-w-0">
            <div class="flex justify-between items-start mb-1">
                <h3
                    class="font-semibold text-neutral-900 dark:text-white truncate pr-2"
                >
                        {{ conversation.name || "未命名对话" }}
                    </h3>
                    <div class="flex flex-col items-end shrink-0 group-hover:opacity-0 transition-opacity duration-200">
                        <span
                            class="text-sm text-neutral-500 dark:text-neutral-400"
                        >
                            {{
                                displayTime(
                                    conversation.last_message_time ||
                                        conversation.updated_at,
                                )
                            }}
                        </span>
                    </div>
                </div>

                <div class="flex justify-between items-center">
                    <div class="flex items-center gap-1.5 flex-1 min-w-0 pr-2">
                        <UIcon
                            v-if="lastMessageType !== 'text'"
                            :name="getMessageTypeIcon(lastMessageType)"
                            class="size-3.5 text-neutral-400 dark:text-neutral-500 shrink-0"
                        />
                        <p
                            class="text-sm text-neutral-500 dark:text-neutral-400 truncate"
                        >
                            {{
                                displayLastMessage(
                                    conversation.last_message?.content || "",
                                    lastMessageType,
                                )
                            }}
                        </p>
                    </div>

                    <div class="flex items-center gap-1.5 shrink-0">
                        <UIcon
                            v-if="conversation.is_pinned"
                            name="i-lucide-pin"
                            class="size-3.5 text-primary-500"
                        />
                        <UIcon
                            v-if="conversation.is_muted"
                            name="i-lucide-bell-off"
                            class="size-3.5 text-neutral-400 dark:text-neutral-500"
                        />
                        <UBadge
                            v-if="conversation.unread_count > 0"
                            color="error"
                            size="xs"
                            variant="solid"
                        >
                            {{ displayUnreadCount(conversation.unread_count) }}
                        </UBadge>
                    </div>
                </div>
            </div>

        <!-- 悬浮操作 -->
        <div
            class="absolute right-2 top-2 opacity-0 group-hover:opacity-100 transition-opacity duration-200 bg-white dark:bg-neutral-800 shadow-sm rounded-2xl border border-neutral-200 dark:border-neutral-700"
        >
            <UDropdownMenu :items="conversationActions">
                    <UButton
                        variant="ghost"
                        color="neutral"
                        icon="i-lucide-more-vertical"
                        size="xs"
                        square
                        @click.stop
                />
            </UDropdownMenu>
        </div>
    </div>
</template>

<script setup lang="ts">
import type { PrivateChatLastMessage } from "~/types/api";
import type { DropdownMenuItem } from "@nuxt/ui";
import { usePrivateMessagesService } from "~/services/private_messages.service";
import { useGroupMessagesService } from "~/services/group_messages.service";

interface Conversation {
    group_id: string;
    name: string;
    avatar?: string;
    type: "group" | "user";
    members_count?: number;
    last_message?: PrivateChatLastMessage;
    last_message_time?: string;
    unread_count: number;
    is_pinned: boolean;
    is_muted: boolean;
    updated_at: string;
}

interface Props {
    conversation: Conversation;
    isOnline?: boolean;
}
const props = defineProps<Props>();

const emit = defineEmits<{
    (e: "open", conversation: Conversation): void;
    (e: "pin", conversation: Conversation): void;
    (e: "mute", conversation: Conversation): void;
    (e: "delete", conversation: Conversation): void;
}>();

const feedback = useFeedbackStore();
const { formatTime } = useFormatters();

const lastMessageType = computed(
    () => props.conversation.last_message?.type || "text",
);

const conversationActions = computed((): DropdownMenuItem[][] => [
    [
        {
            label: props.conversation.is_pinned ? "取消置顶" : "置顶",
            icon: "i-lucide-pin",
            onSelect: togglePin,
        },
        {
            label: props.conversation.is_muted ? "取消免打扰" : "免打扰",
            icon: props.conversation.is_muted
                ? "i-lucide-bell"
                : "i-lucide-bell-off",
            onSelect: toggleMute,
        },
    ],
    [
        {
            label: "标记已读",
            icon: "i-lucide-check-circle",
            onSelect: markAsRead,
        },
    ],
    [
        {
            label: "删除",
            icon: "i-lucide-trash-2",
            color: "error",
            onSelect: deleteConversation,
        },
    ],
]);

const displayTime = (ts?: string) => (ts ? formatTime(ts) : "");
const displayUnreadCount = (n: number) => (n > 99 ? "99+" : n.toString());

const getMessageTypeIcon = (type: string) => {
    const map: Record<string, string> = {
        image: "i-lucide-image",
        file: "i-lucide-file",
        location: "i-lucide-map-pin",
        voice: "i-lucide-mic",
        video: "i-lucide-video",
    };
    return map[type] || "i-lucide-message-square";
};

const displayLastMessage = (msg: string, type: string) => {
    if (type !== "text") {
        const map: Record<string, string> = {
            image: "[图片]",
            file: "[文件]",
            location: "[位置]",
            voice: "[语音]",
            video: "[视频]",
        };
        return map[type] || "[消息]";
    }
    return msg || "暂无消息";
};

const openConversation = () => emit("open", props.conversation);
const togglePin = () => {
    emit("pin", props.conversation);
    feedback.success(props.conversation.is_pinned ? "已取消置顶" : "已置顶");
};
const toggleMute = () => {
    emit("mute", props.conversation);
    feedback.success(
        props.conversation.is_muted ? "已取消免打扰" : "已开启免打扰",
    );
};

const markAsRead = async () => {
    try {
        const messagesStore = useMessagesStore();
        if (props.conversation.type === "group") {
            await useGroupMessagesService().markAsRead({
                group_chat_id: props.conversation.group_id,
            });
        } else {
            await usePrivateMessagesService().markPrivateMessagesAsRead({
                target_user_id: parseInt(props.conversation.group_id),
            });
        }
        messagesStore.updateUnreadCount(props.conversation.group_id, 0);
        feedback.success("已标记为已读");
    } catch (e) {
        feedback.error("操作失败", getErrorMessage(e));
    }
};

const deleteConversation = async () => {
    if (
        await useModal().showConfirm({
            title: "删除对话",
            message: "确定要删除该对话吗？此操作不可撤销。",
            confirmText: "删除",
            confirmColor: "error",
        })
    ) {
        emit("delete", props.conversation);
        feedback.success("对话已删除");
    }
};
</script>
