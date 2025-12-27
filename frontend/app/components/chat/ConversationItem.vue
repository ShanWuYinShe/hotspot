<template>
    <UCard
        @click="openConversation"
        variant="outline"
        data-hs-card="interactive"
    >
        <div class="flex items-center gap-4">
            <!-- 头像 -->
            <div class="relative">
                <!-- 在线状态指示器 (仅私聊显示) -->
                <UChip 
                    v-if="conversation.type === 'user' && isOnline"
                    color="success" 
                    size="2xs" 
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
                    size="2xl" 
                    position="bottom-right"
                    :text="displayMemberCount(conversation.members_count || 0)"
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
                        <span class="text-sm shrink-0">
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
                            <span class="shrink-0">
                              <UIcon
                                  :name="getMessageTypeIcon(lastMessageType)"
                                  :size="16"
                              />
                            </span>
                        </UTooltip>

                        <!-- 最后一条消息 -->
                        <p class="text-sm truncate">
                            {{
                                displayLastMessage(
                                    conversation.last_message?.content || '',
                                    lastMessageType,
                                )
                            }}
                        </p>
                    </div>

                    <!-- 置顶图标 -->
                    <UTooltip v-if="conversation.is_pinned" text="置顶会话">
                        <span class="shrink-0 ml-2">
                          <UIcon
                              name="i-lucide-bookmark"
                              :size="16"
                          />
                        </span>
                    </UTooltip>
                </div>
            </div>

            <!-- 操作按钮 -->
            <div class="transition-opacity duration-200 hover:opacity-80">
                <UDropdownMenu :items="conversationActions">
                    <UTooltip text="更多操作">
                        <UButton
                            variant="ghost"
                            size="sm"
                            :square="true"
                            icon="i-lucide-ellipsis"
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
import type { PrivateChatLastMessage } from "~/types/api";
import { getErrorMessage } from '~/utils/error';
import type { DropdownMenuItem } from '@nuxt/ui'
import { usePrivateMessagesService } from '~/services/private_messages.service'
import { useGroupMessagesService } from '~/services/group_messages.service'

// 本地 Conversation 类型定义 - 基于后端 ConversationsResponse 规范
interface Conversation {
  group_id: string;         // 群组ID
  name: string;             // 群组名称或对话名称
  avatar?: string;          // 头像URL
  type: 'group' | 'user';   // 对话类型
  group_type?: string;      // 群组类型：'private_chat'|'public_chat'
  members_count?: number;    // 成员数量
  last_message?: PrivateChatLastMessage;
  last_message_time?: string; // 最后消息时间
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

const feedback = useFeedbackStore()

// 计算最后一条消息的类型
const lastMessageType = computed(() => {
    // 从 last_message 对象中获取消息类型
    return props.conversation.last_message?.type || "text";
});

// 对话操作菜单
const conversationActions = computed((): DropdownMenuItem[][] => [
    [
        {
            label: props.conversation.is_pinned ? "取消置顶" : "置顶对话",
            icon: props.conversation.is_pinned
                ? "i-lucide-bookmark"
                : "i-lucide-bookmark",
            onSelect: () => togglePin(),
        },
        {
            label: props.conversation.is_muted ? "取消免打扰" : "免打扰",
            icon: props.conversation.is_muted
                ? "i-lucide-volume-2"
                : "i-lucide-volume-x",
            onSelect: () => toggleMute(),
        },
    ],
    [
        {
            label: "标记为已读",
            icon: "i-lucide-circle-check",
            onSelect: () => markAsRead(),
        },
    ],
    [
        {
            label: "删除对话",
            icon: "i-lucide-circle-x",
            color: "error",
            onSelect: () => deleteConversation(),
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
            return "i-lucide-image";
        case "file":
            return "i-lucide-file-text";
        case "location":
            return "i-lucide-map-pin";
        case "voice":
            return "i-lucide-mic";
        case "video":
            return "i-lucide-video";
        default:
            return "i-lucide-messages-square";
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
    feedback.success(props.conversation.is_pinned ? "已取消置顶" : "已置顶")
};

const toggleMute = () => {
    emit("mute", props.conversation);
    feedback.success(props.conversation.is_muted ? "已取消免打扰" : "已开启免打扰")
};

const markAsRead = async () => {
    try {
        const messagesStore = useMessagesStore();
        const targetId = props.conversation.group_id;

        // 先调用后端标记已读（兜底：失败也会本地置零，避免 UI 卡住）
        if (props.conversation.type === "group") {
          const groupMessagesService = useGroupMessagesService()
          await groupMessagesService.markAsRead({ id: String(targetId) })
        } else {
          const privateMessagesService = usePrivateMessagesService()
          await privateMessagesService.markPrivateMessagesAsRead({ id: String(targetId) })
        }

        // 本地同步
        messagesStore.updateUnreadCount(String(targetId), 0)
        
        feedback.success("已标记为已读")
    } catch (e: unknown) {
        feedback.error("操作失败", getErrorMessage(e))
    }
};

const deleteConversation = async () => {
    const modal = useModal()
    const confirmed = await modal.showConfirm({
        title: '删除对话',
        message: '确定要删除该对话吗？此操作不可撤销。',
        confirmText: '删除',
        confirmColor: 'error'
    })
    if (confirmed) {
        emit("delete", props.conversation);
        feedback.success("对话已删除")
    }
};
</script>
