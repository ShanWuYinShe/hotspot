<template>
    <UCard
        @click="openConversation"
        class="cursor-pointer group"
        :class="{
            'ring-2 ring-primary-500 bg-primary-50 dark:bg-primary-900/10':
                conversation.unread_count > 0,
        }"
    >
        <div class="flex items-center space-x-4">
            <!-- 头像 -->
            <div class="relative">
                <!-- 在线状态指示器 (仅私聊显示) -->
                <UChip 
                    v-if="conversation.type === 'user' && isOnline"
                    color="success" 
                    size="sm" 
                    position="bottom-right"
                >
                    <UAvatar
                        :src="conversation.avatar"
                        :alt="conversation.name || '会话'"
                        size="md"
                    >
                        {{ conversation.name ? conversation.name.charAt(0) : "聊" }}
                    </UAvatar>
                </UChip>

                <!-- 群组成员数量指示器 -->
                <UChip 
                    v-else-if="conversation.type === 'group'"
                    color="info" 
                    size="md" 
                    position="bottom-right"
                    :text="displayMemberCount(conversation.member_count || 0)"
                >
                    <UAvatar
                        :src="conversation.avatar"
                        :alt="conversation.name || '会话'"
                        size="md"
                    >
                        {{ conversation.name ? conversation.name.charAt(0) : "聊" }}
                    </UAvatar>
                </UChip>

                <!-- 普通头像 (无状态指示器) -->
                <UAvatar
                    v-else
                    :src="conversation.avatar"
                    :alt="conversation.name || '会话'"
                    size="md"
                >
                    {{ conversation.name ? conversation.name.charAt(0) : "聊" }}
                </UAvatar>
            </div>

            <!-- 对话信息 -->
            <div class="flex-1 min-w-0">
                <div class="flex justify-between items-start mb-1">
                    <h3 class="text-lg font-semibold truncate">
                        {{ conversation.name }}
                    </h3>
                    <div class="flex flex-col items-end space-y-1">
                        <span class="text-sm text-neutral-500 flex-shrink-0">
                            {{ displayTime(conversation.last_message_time || conversation.updated_at) }}
                        </span>
                        <!-- 未读消息数 -->
                        <UBadge
                            v-if="conversation.unread_count > 0"
                            size="xs"
                            color="error"
                        >
                            {{ displayUnreadCount(conversation.unread_count) }}
                        </UBadge>
                    </div>
                </div>

                <div class="flex items-center justify-between">
                    <div class="flex items-center space-x-2 flex-1 min-w-0">
                        <!-- 消息类型图标 -->
                        <UTooltip 
                            v-if="lastMessageType !== 'text'"
                            :text="getMessageTypeTooltip(lastMessageType)"
                        >
                            <UIcon
                                :name="getMessageTypeIcon(lastMessageType)"
                                class="w-4 h-4 text-neutral-400 flex-shrink-0"
                            />
                        </UTooltip>

                        <!-- 最后一条消息 -->
                        <p class="text-sm text-neutral-600 truncate">
                            {{
                                displayLastMessage(
                                    conversation.last_message?.content || '',
                                    lastMessageType,
                                )
                            }}
                        </p>
                    </div>

                    <!-- 置顶图标 -->
                    <UTooltip v-if="conversation.is_pinned" text="已置顶会话">
                        <UIcon
                            name="i-heroicons-bookmark-solid"
                            class="w-4 h-4 text-yellow-500 flex-shrink-0 ml-2"
                        />
                    </UTooltip>
                </div>
            </div>

            <!-- 操作按钮 -->
            <div class="opacity-0 group-hover:opacity-100 transition-opacity">
                <UDropdownMenu :items="conversationActions">
                    <UTooltip text="更多操作">
                        <UButton
                            variant="ghost"
                            size="md"
                            icon="i-heroicons-ellipsis-horizontal"
                            aria-label="更多操作"
                            @click.stop
                        />
                    </UTooltip>
                </UDropdownMenu>
            </div>
        </div>
    </UCard>
</template>

<script setup lang="ts">
import { computed, ref } from "vue";
import type { SendMessageResponse } from "~/types/api";

// 本地 Conversation 类型定义 - 基于后端 ConversationsResponse 规范
interface Conversation {
  group_id: string;         // 群组ID
  name: string;             // 群组名称或对话名称
  avatar?: string;          // 头像URL
  type: 'group' | 'user';   // 对话类型
  group_type?: string;      // 群组类型：'private_chat'|'public_chat'
  member_count?: number;    // 成员数量
  last_message?: SendMessageResponse;
  last_message_time?: string; // 最后消息时间
  unread_count: number;
  is_pinned: boolean;
  is_muted: boolean;
  updated_at: string;
  id?: string;              // 兼容属性
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

// 移除不存在的 openConfirmModal

const { success, error, info } = useNotifications()
const toast = { success, error, info };

// 计算最后一条消息的类型
const lastMessageType = computed(() => {
    // 从 last_message 对象中获取消息类型
    return props.conversation.last_message?.message_type || "text";
});

// 对话操作菜单
const conversationActions = computed(() => [
    [
        {
            label: props.conversation.is_pinned ? "取消置顶" : "置顶对话",
            icon: props.conversation.is_pinned
                ? "i-heroicons-bookmark"
                : "i-heroicons-bookmark-solid",
            click: () => togglePin(),
        },
        {
            label: props.conversation.is_muted ? "取消免打扰" : "免打扰",
            icon: props.conversation.is_muted
                ? "i-heroicons-speaker-wave"
                : "i-heroicons-speaker-x-mark",
            click: () => toggleMute(),
        },
    ],
    [
        {
            label: "标记为已读",
            icon: "i-heroicons-check-circle",
            click: () => markAsRead(),
        },
        {
            label: "清空聊天记录",
            icon: "i-heroicons-trash",
            click: () => clearHistory(),
        },
    ],
    [
        {
            label: "删除对话",
            icon: "i-heroicons-x-circle",
            click: () => deleteConversation(),
            class: "text-red-600",
        },
    ],
]);

const { formatTime } = useFormatters();

// 业务逻辑函数（非格式化函数）
const displayTime = (timeString?: string) => {
    if (!timeString) return "";
    return formatTime(timeString);
};

// 显示成员数量（带截断）
const displayMemberCount = (count: number) => {
    if (count > 99) return "99+";
    return count.toString();
};

// 显示未读消息数（带截断）
const displayUnreadCount = (count: number) => {
    if (count > 99) return "99+";
    return count.toString();
};

// 获取消息类型图标
const getMessageTypeIcon = (type: string) => {
    switch (type) {
        case "image":
            return "i-heroicons-photo";
        case "file":
            return "i-heroicons-document";
        case "location":
            return "i-heroicons-map-pin";
        case "voice":
            return "i-heroicons-microphone";
        case "video":
            return "i-heroicons-video-camera";
        default:
            return "i-heroicons-chat-bubble-left-right";
    }
};

// 获取消息类型提示文本
const getMessageTypeTooltip = (type: string) => {
    switch (type) {
        case "image":
            return "图片消息";
        case "file":
            return "文件消息";
        case "location":
            return "位置消息";
        case "voice":
            return "语音消息";
        case "video":
            return "视频消息";
        default:
            return "文本消息";
    }
};

// 格式化最后一条消息
const displayLastMessage = (message: string, type: string) => {
    if (type !== "text") {
        switch (type) {
            case "image":
                return "[图片]";
            case "file":
                return "[文件]";
            case "location":
                return "[位置]";
            case "voice":
                return "[语音]";
            case "video":
                return "[视频]";
            default:
                return "[消息]";
        }
    }
    return message || "暂无消息";
};

// 操作函数
const openConversation = () => {
    emit("open", props.conversation);
};

const togglePin = () => {
    emit("pin", props.conversation);
    toast.add({
        title: props.conversation.is_pinned ? "已取消置顶" : "已置顶",
        color: "success",
    });
};

const toggleMute = () => {
    emit("mute", props.conversation);
    toast.add({
        title: props.conversation.is_muted ? "已取消免打扰" : "已开启免打扰",
        color: "success",
    });
};

const markAsRead = async () => {
    try {
        const messagesStore = useMessagesStore();
        const isGroup = props.conversation.type === "group";
        const targetId = props.conversation.group_id || props.conversation.id;
        
        // 使用 updateUnreadCount 方法标记为已读
        messagesStore.updateUnreadCount(String(targetId), 0);
        
        toast.add({ title: "已标记为已读", color: "success" });
    } catch (e: any) {
        toast.add({
            title: "操作失败",
            description: e?.message,
            color: "error",
        });
    }
};

const clearHistory = () => {
    toast.add({
        title: "功能开发中",
        description: "清空聊天记录功能即将上线",
        color: "warning",
    });
};

const deleteConversation = () => {
    if (confirm("确定要删除该对话吗？此操作不可撤销。")) {
        confirmDelete();
    }
};

const confirmDelete = () => {
    emit("delete", props.conversation);
    toast.add({ title: "对话已删除", color: "success" });
};
</script>
