<template>
    <div class="flex flex-col gap-6 h-full min-h-0">
        <!-- Header -->
        <div class="flex items-center justify-between shrink-0">
            <div>
                <h1 class="text-2xl font-bold text-neutral-900 dark:text-white">
                    消息中心
                </h1>
                <p class="text-neutral-500 dark:text-neutral-400 mt-1">
                    最近的会话
                </p>
            </div>
        </div>

        <div class="flex-1 min-h-0 flex flex-col md:flex-row gap-6">
            <!-- 左侧：会话列表 -->
            <UCard
                class="w-full md:w-80 lg:w-96 flex flex-col h-full overflow-hidden"
            >
                <template #header>
                    <UInput
                        v-model="searchQuery"
                        placeholder="搜索会话..."
                        icon="i-lucide-search"
                        size="sm"
                    >
                        <template v-if="searchQuery" #trailing>
                            <UButton
                                color="neutral"
                                variant="link"
                                icon="i-lucide-x"
                                size="xs"
                                @click="searchQuery = ''"
                            />
                        </template>
                    </UInput>
                </template>

                <!-- 列表 -->
                <div class="flex-1 overflow-y-auto -mx-5 sm:-mx-7">
                    <!-- 加载中 -->
                    <div
                        v-if="loading && conversations.length === 0"
                        class="p-2 space-y-2"
                    >
                        <ConversationListSkeleton />
                    </div>

                    <!-- 搜索结果/会话列表 -->
                    <div
                        v-else-if="filteredConversations.length > 0"
                        class="space-y-2 p-2"
                    >
                        <ConversationItem
                            v-for="conv in filteredConversations"
                            :key="conv.group_id"
                            :conversation="conv"
                            :is-online="conv.is_online"
                            @open="() => handleConversationClick(conv)"
                            @delete="() => handleDeleteConversation(conv)"
                        />
                    </div>

                    <!-- 空状态 -->
                    <div
                        v-else
                        class="h-full flex flex-col items-center justify-center p-6 text-center text-neutral-500 dark:text-neutral-400"
                    >
                        <UIcon
                            name="i-lucide-message-circle-off"
                            class="size-12 mb-2 text-neutral-300 dark:text-neutral-600"
                        />
                        <p class="text-sm">暂无会话</p>
                        <p
                            class="text-xs text-neutral-400 dark:text-neutral-500 mt-1"
                        >
                            开始一个新的聊天吧
                        </p>
                        <UButton
                            to="/users/nearby"
                            variant="soft"
                            color="primary"
                            size="sm"
                            class="mt-4"
                        >
                            找人聊天
                        </UButton>
                    </div>
                </div>
            </UCard>

            <!-- 右侧：空状态提示 (仅在桌面端显示) -->
            <div
                class="hidden md:flex flex-1 items-center justify-center bg-white dark:bg-neutral-900 border border-neutral-200 dark:border-neutral-800 rounded-2xl p-8 text-center text-neutral-500 dark:text-neutral-400 shadow-sm"
            >
                <div class="max-w-xs">
                    <div class="relative inline-block mb-6">
                        <div
                            class="absolute inset-0 bg-primary-500/20 blur-xl rounded-full"
                        />
                        <UIcon
                            name="i-lucide-messages-square"
                            class="size-24 text-primary-500 relative z-10"
                        />
                    </div>
                    <h2
                        class="text-2xl font-bold mb-2 text-neutral-900 dark:text-white"
                    >
                        欢迎来到消息中心
                    </h2>
                    <p class="mb-8">
                        选择左侧的会话开始聊天，或者去发现新朋友。
                    </p>
                    <div class="flex justify-center gap-4">
                        <UButton to="/users/nearby" icon="i-lucide-users"
                            >附近的人</UButton
                        >
                        <UButton
                            to="/groups/nearby"
                            variant="soft"
                            color="neutral"
                            icon="i-lucide-users-round"
                            >附近的群</UButton
                        >
                    </div>
                </div>
            </div>
        </div>
    </div>
</template>

<script setup lang="ts">
import { usePrivateChatsService } from "~/services/private_chats.service";
import { useGroupsService } from "~/services/groups.service";
import type { PrivateChatListItem, JoinedGroupItem, PrivateChatLastMessage } from "~/types/api";
import ConversationItem from "~/components/chat/ConversationItem.vue";
import ConversationListSkeleton from "~/components/skeletons/ConversationListSkeleton.vue";

definePageMeta({
    layout: "dashboard",
    requiresAuth: true,
});

const router = useRouter();
const privateChatService = usePrivateChatsService();
const groupsService = useGroupsService();
const feedback = useFeedbackStore();

// 状态
const loading = ref(true);
const searchQuery = ref("");
const privateChats = ref<PrivateChatListItem[]>([]);
const groupChats = ref<JoinedGroupItem[]>([]);

// 适配 ConversationItem 需要的类型
interface Conversation {
    group_id: string; // 列表API的ID (user_id for private, group_chat_id for group)
    name: string;
    avatar?: string;
    type: "group" | "user";
    target_id: string; // 实际导航用的ID (user_id for private, group_id for group)
    members_count?: number;
    last_message?: PrivateChatLastMessage;
    last_message_time?: string;
    unread_count: number;
    is_pinned: boolean;
    is_muted: boolean;
    updated_at: string;
    is_online?: boolean; // 仅用于辅助 statusChip
}

const conversations = computed(() => {
    const list: Conversation[] = [];

    // 私聊
    privateChats.value.forEach((chat) => {
        list.push({
            group_id: chat.other_user.user_id.toString(), // 使用 other_user_id 作为唯一标识
            type: "user",
            target_id: chat.other_user.user_id.toString(),
            name: chat.other_user.nickname,
            avatar: chat.other_user.avatar_url,
            unread_count: chat.unread_count,
            last_message: chat.last_message,
            last_message_time: chat.last_activity_at,
            is_pinned: false, // 列表API暂不提供
            is_muted: false, // 列表API暂不提供
            updated_at: chat.last_activity_at,
            is_online: chat.is_online,
        });
    });

    // 群聊
    groupChats.value.forEach((group) => {
        list.push({
            group_id: group.group_chat_id,
            type: "group",
            target_id: group.group_chat_id,
            name: group.name,
            avatar: group.avatar_url,
            members_count: group.members_count,
            unread_count: group.my_state.unread_count,
            last_message: undefined,
            last_message_time: group.my_state.joined_at, // 暂用 joined_at 代替，API 文档未提供活跃时间
            is_pinned: group.my_state.is_pinned,
            is_muted: group.my_state.is_muted,
            updated_at: group.my_state.joined_at || "", // 暂用 joined_at 代替
            is_online: undefined,
        });
    });

    // 按时间倒序
    return list.sort(
        (a, b) =>
            new Date(b.updated_at).getTime() - new Date(a.updated_at).getTime(),
    );
});

const filteredConversations = computed(() => {
    if (!searchQuery.value) return conversations.value;
    const query = searchQuery.value.toLowerCase();
    return conversations.value.filter((c) =>
        c.name.toLowerCase().includes(query),
    );
});

// 初始化
onMounted(() => {
    refresh();
});

async function refresh() {
    loading.value = true;
    try {
        const [pRes, gRes] = await Promise.all([
            privateChatService.getPrivateChats({ limit: 50 }),
            groupsService.getJoined({ limit: 50 }),
        ]);
        privateChats.value = pRes.items;
        groupChats.value = gRes.items;
    } catch (e) {
        console.error("Failed to load conversations", e);
    } finally {
        loading.value = false;
    }
}

function handleConversationClick(conv: Conversation) {
    if (conv.type === "user") {
        router.push(`/chat/users/${conv.target_id}`);
    } else {
        router.push(`/chat/groups/${conv.target_id}`);
    }
}

async function handleDeleteConversation(conv: Conversation) {
    if (!confirm("确定要删除该会话吗？聊天记录将被清除。")) return;

    try {
        if (conv.type === "user") {
            await privateChatService.deletePrivateChat({
                target_user_id: parseInt(conv.target_id),
            });
            privateChats.value = privateChats.value.filter(
                (c) => c.other_user.user_id.toString() !== conv.target_id,
            );
        } else {
            // 群聊"删除会话"通常意味着退出群聊或者隐藏，视业务需求而定
            // 这里假设是退出群聊
            if (!confirm("确定要退出该群聊吗？")) return;
            await groupsService.leaveGroup({ group_chat_id: conv.group_id });
            groupChats.value = groupChats.value.filter(
                (g) => g.group_chat_id !== conv.group_id,
            );
        }
        feedback.success("会话已删除");
    } catch (e) {
        feedback.error("操作失败");
    }
}
</script>
