import { defineStore } from "pinia";
import type { Conversation } from "~/types/message";
import { convertConversationSummary } from "~/types/message";
import type { ConversationItem, UserInfo, MessageType, ConversationSummary, Message } from "~/types";

/**
 * 消息 Store
 * 仅负责消息数据的缓存，不持久化（数据应从服务器获取最新状态）
 */
export const useMessagesStore = defineStore("messages", () => {
  // 缓存状态 - 不持久化
  const conversations = ref<Conversation[]>([]);
  const currentConversation = ref<Conversation | null>(null);
  const messages = ref<Message[]>([]);

  // 分页状态
  const pagination = ref({
    conversations: {
      hasMore: false,
      nextCursor: undefined as string | undefined,
    },
    messages: { hasMore: false, nextCursor: undefined as string | undefined },
  });

  // 加载状态
  const loading = ref({
    conversations: false,
    messages: false,
    sending: false,
  });

  // 实时消息支持
  const optimisticMessages = ref<Map<string, Message>>(markRaw(new Map()));
  const userOnlineStatus = ref<Map<string, boolean>>(markRaw(new Map()));

  // 计算属性
  const totalUnreadCount = computed(() =>
    conversations.value.reduce((total, conv) => total + conv.unread_count, 0),
  );

  // 业务方法 - 使用 Service 层
  async function fetchConversations(loadMore = false) {
    const messageService = useMessageService();
    loading.value.conversations = true;

    try {
      const cursor = loadMore
        ? pagination.value.conversations.nextCursor
        : undefined;
      const response = await messageService.getConversations(cursor);
      const convertedItems = (response.items || []).map((item: ConversationSummary) => convertConversationSummary(item));

      if (loadMore) {
        conversations.value.push(...convertedItems);
      } else {
        conversations.value = convertedItems;
      }

      pagination.value.conversations = {
        hasMore: response.has_more,
        nextCursor: response.next_cursor,
      };
    } finally {
      loading.value.conversations = false;
    }
  }

  async function searchConversations(query: string, loadMore = false) {
    if (!query.trim()) {
      return fetchConversations(loadMore);
    }

    const messageService = useMessageService();
    loading.value.conversations = true;

    try {
      const cursor = loadMore
        ? pagination.value.conversations.nextCursor
        : undefined;
      // 简单的本地搜索，因为后端没有searchConversations方法
      const allConversations = await messageService.getConversations();
      const filtered = allConversations.items.filter((item: any) =>
        item.peer_name.toLowerCase().includes(query.toLowerCase()),
      );
      const convertedItems = filtered.map((item: ConversationSummary) => convertConversationSummary(item));

      if (loadMore) {
        conversations.value.push(...convertedItems);
      } else {
        conversations.value = convertedItems;
      }

      pagination.value.conversations = {
        hasMore: false,
        nextCursor: undefined,
      };
    } finally {
      loading.value.conversations = false;
    }
  }

  async function createConversation(participantId?: string, groupId?: string) {
    // 后端没有createConversation方法，通过发送消息自动创建会话
    throw new Error("请通过发送消息自动创建会话");
  }

  async function createOrGetConversation(participantId: string) {
    // 首先检查是否已存在对话
    const existingConversation = conversations.value.find(
      (c) => c.type === "user" && c.participant_id === participantId,
    );

    if (existingConversation) {
      return existingConversation;
    }

    // 创建新对话
    return await createConversation(participantId);
  }

  async function searchUsers(query: string) {
    // 使用用户服务搜索用户
    const userService = useUserService();
    return await userService.searchUsers(query);
  }

  async function fetchMessages(
    conversationId: string,
    loadMore = false,
    type: "user" | "group" = "user",
  ) {
    const messageService = useMessageService();
    loading.value.messages = true;

    try {
      const cursor = loadMore
        ? pagination.value.messages.nextCursor
        : undefined;
      const response =
        type === "group"
          ? await messageService.getGroupMessageHistory(
              String(conversationId),
              cursor,
            )
          : await messageService.getDirectMessageHistory(
              { conversationId: String(conversationId) },
              cursor,
            );

      const convertedItems = response.items; // 直接使用后端数据
      if (loadMore) {
        messages.value.push(...convertedItems);
      } else {
        messages.value = convertedItems;
      }
      // 始终按时间升序（旧->新），确保最新消息在底部
      messages.value.sort(
        (a: any, b: any) =>
          new Date(a.created_at).getTime() - new Date(b.created_at).getTime(),
      );

      pagination.value.messages = {
        hasMore: response.has_more,
        nextCursor: response.next_cursor,
      };
    } finally {
      loading.value.messages = false;
    }
  }

  async function sendMessage(
    conversationId: string,
    content: string,
    messageType: MessageType = "text",
  ) {
    const messageService = useMessageService();
    const authStore = useAuthStore();
    loading.value.sending = true;
    // 乐观消息
    const tempId = String(Date.now());
    const optimistic: Message = {
      id: tempId,
      message_id: tempId,
      conversation_id: String(conversationId),
      sender_id: String(authStore.user?.user_id || "0"),
      sender_name: authStore.user?.nickname || "",
      content,
      message_type: messageType,
      sent_at: new Date().toISOString(),
      created_at: new Date().toISOString(),
      status: "sending" as const,
    };
    messages.value.push(optimistic);

    try {
      const resp = await messageService.sendDirectMessage(
        String(conversationId),
        content,
        messageType,
      );
      // 覆盖乐观消息
      const index = messages.value.findIndex((m) => m.id === tempId);
      if (index !== -1) {
        const newMessage: Message = {
          id: String(resp.message_id) || tempId,
          message_id: String(resp.message_id) || tempId,
          conversation_id: resp.conversation_id,
          sender_id: String(authStore.user?.user_id || "0"),
          sender_name: authStore.user?.nickname || "",
          content,
          message_type: resp.message_type,
          created_at: resp.sent_at,
          sent_at: resp.sent_at,
          latitude: 0,
          longitude: 0,
        };
        messages.value.splice(index, 1, newMessage);
      }

      // 更新对话的最后消息
      let conversation = conversations.value.find(
        (c) => c.id === String(resp.conversation_id),
      );
      if (conversation) {
        conversation.last_message = content;
        conversation.last_message_time = resp.sent_at;
      } else {
        // 首次发送创建会话时，插入会话占位并尽力补齐信息
        conversation = {
          id: String(resp.conversation_id),
          name: "",
          avatar: undefined,
          last_message: content,
          last_message_time: resp.sent_at,
          unread_count: 0,
          type: "user",
          participant_id: String((resp as any).recipient_id),
          is_pinned: false,
          is_muted: false,
        };
        conversations.value.unshift(conversation);
        // 异步补齐对方资料
        try {
          const userService = useUserService();
          const profile = await userService.getUserProfile(
            String((resp as any).recipient_id),
          ) as UserInfo;
          const target = conversations.value.find(
            (c) => c.id === String(resp.conversation_id),
          );
          if (target) {
            target.name = profile.nickname || target.name;
            // 后端无头像字段，保持已有 avatar 占位
          }
        } catch (_) {
          // 忽略补齐失败
        }
      }

      return messages.value.find(
        (m) =>
          m.conversation_id === String(resp.conversation_id) &&
          m.sent_at === resp.sent_at,
      ) as Message;
    } catch (e) {
      // 失败：标记乐观消息为失败
      const idx = messages.value.findIndex((m) => m.id === tempId);
      if (idx !== -1) {
        // 安全地更新消息状态
        const message = messages.value[idx];
        if (message) {
          Object.assign(message, {
            status: "failed" as const,
            retry_count: (message as any).retry_count ? (message as any).retry_count + 1 : 1
          });
        }
      }
      throw e;
    } finally {
      loading.value.sending = false;
    }
  }

  async function sendGroupMessage(
    groupId: string,
    content: string,
    messageType: MessageType = "text",
  ) {
    const messageService = useMessageService();
    const authStore = useAuthStore();
    loading.value.sending = true;
    const tempId = String(Date.now());
    const optimistic: Message = {
      id: tempId,
      message_id: tempId,
      group_id: String(groupId),
      sender_id: String(authStore.user?.user_id || "0"),
      sender_name: authStore.user?.nickname || "",
      content,
      message_type: messageType,
      sent_at: new Date().toISOString(),
      created_at: new Date().toISOString(),
      status: "sending" as const,
    };
    messages.value.push(optimistic);
    try {
      const resp = await messageService.sendGroupMessage(
        String(groupId),
        content,
        messageType,
      );
      const index = messages.value.findIndex((m) => m.id === tempId);
      if (index !== -1) {
        const newMessage: Message = {
          id: String((resp as any).message_id) || tempId,
          message_id: String((resp as any).message_id) || tempId,
          group_id: resp.group_id,
          sender_id: String(authStore.user?.user_id || "0"),
          sender_name: authStore.user?.nickname || "",
          content,
          message_type: messageType,
          created_at: resp.sent_at,
          sent_at: resp.sent_at,
          latitude: 0,
          longitude: 0,
        };
        messages.value.splice(index, 1, newMessage);
      }
      // 若会话列表中没有该群会话，插入占位
      const hasGroupConv = conversations.value.some(
        (c) =>
          c.type === "group" &&
          (c.group_id === String(groupId) || c.id === String(groupId)),
      );
      if (!hasGroupConv) {
        conversations.value.unshift({
          id: String(groupId),
          name: "",
          avatar: undefined,
          last_message: content,
          last_message_time: resp.sent_at,
          unread_count: 0,
          type: "group",
          group_id: String(groupId),
          is_pinned: false,
          is_muted: false,
        });
        // 异步补齐群资料
        try {
          const groupService = useGroupService();
          const details = await groupService.getGroupDetails(String(groupId));
          const target = conversations.value.find(
            (c) =>
              c.type === "group" &&
              (c.group_id === String(groupId) || c.id === String(groupId)),
          );
          if (target) {
            target.name = (details as any).name || target.name;
            // 后端无头像字段，保留现有 avatar
            (target as any).member_count =
              (details as any).member_count ?? (target as any).member_count;
          }
        } catch (_) {
          // 忽略补齐失败
        }
      }
      return messages.value.find(
        (m) => m.group_id === String(groupId) && m.sent_at === resp.sent_at,
      ) as Message;
    } catch (e) {
      const idx = messages.value.findIndex((m) => m.id === tempId);
      if (idx !== -1) {
        const message = messages.value[idx];
        if (message) {
          message.status = "failed";
          message.retry_count = (message.retry_count || 0) + 1;
        }
      }
      throw e;
    } finally {
      loading.value.sending = false;
    }
  }

  // 重试失败消息（仅限当前列表中的失败乐观消息）
  async function retryMessage(
    messageId: string,
    type: "user" | "group" = "user",
  ) {
    const m = messages.value.find((mm) => mm.id === messageId) as any;
    if (!m || m.status !== "failed") return;
    m.status = "sending";
    try {
      if (type === "group") {
        await sendGroupMessage(String(m.group_id), m.content, m.message_type);
      } else {
        await sendMessage(
          String(m.conversation_id || m.recipient_id || ""),
          m.content,
          m.message_type,
        );
      }
    } catch (e) {
      m.status = "failed";
      m.retry_count = (m.retry_count || 0) + 1;
      throw e;
    }
  }

  async function markConversationAsRead(
    conversationId: string,
    type: "user" | "group" = "user",
    lastReadMessageId?: string,
  ) {
    const messageService = useMessageService();
    await messageService.markAsRead(
      String(conversationId),
      type,
      lastReadMessageId,
    );

    // 更新本地状态
    const conversation =
      type === "group"
        ? conversations.value.find(
            (c) =>
              c.type === "group" &&
              (c.group_id === String(conversationId) ||
                c.id === String(conversationId)),
          )
        : conversations.value.find((c) => c.id === String(conversationId));
    if (conversation) {
      conversation.unread_count = 0;
    }
  }

  async function deleteConversation(conversationId: string) {
    // 后端没有deleteConversation方法，暂时只从本地移除
    const index = conversations.value.findIndex((c) => c.id === conversationId);
    if (index > -1) {
      conversations.value.splice(index, 1);
    }

    // 如果是当前对话，清空当前对话
    if (currentConversation.value?.id === String(conversationId)) {
      currentConversation.value = null;
      messages.value = [];
    }
  }

  function setCurrentConversation(conversation: Conversation) {
    currentConversation.value = conversation;
    messages.value = []; // 清空之前的消息
    pagination.value.messages = { hasMore: false, nextCursor: undefined };
  }

  function clearMessages() {
    messages.value = [];
    pagination.value.messages = { hasMore: false, nextCursor: undefined };
  }

  // 用户切换时清理本地缓存，避免跨账号数据串扰
  function resetAll() {
    conversations.value = [];
    currentConversation.value = null;
    messages.value = [];
    pagination.value = {
      conversations: { hasMore: false, nextCursor: undefined },
      messages: { hasMore: false, nextCursor: undefined },
    };
    loading.value = { conversations: false, messages: false, sending: false };
    optimisticMessages.value.clear();
    userOnlineStatus.value.clear();
  }

  const auth = useAuthStore();
  watch(
    () => auth.user?.user_id,
    (newId, oldId) => {
      if (newId !== oldId) resetAll();
    },
  );

  // 实时消息功能
  function addRealtimeMessage(message: any) {
    // 检查消息是否已存在
    const existingIndex = messages.value.findIndex((m) => m.id === message.id);
    if (existingIndex === -1) {
      // 按时间顺序插入消息
      const insertIndex = messages.value.findIndex(
        (m) =>
          new Date(m.sent_at || m.created_at || new Date()) > new Date(message.created_at || message.sent_at || new Date()),
      );
      if (insertIndex === -1) {
        messages.value.push(message);
      } else {
        messages.value.splice(insertIndex, 0, message);
      }
    }
  }

  function updateConversationLastMessage(
    conversationId: string,
    messageInfo: any,
  ) {
    const conversation = conversations.value.find(
      (c) => c.id === String(conversationId),
    );
    if (conversation) {
      conversation.last_message = messageInfo.content;
      conversation.last_message_time = messageInfo.time;

      // 将对话移到列表顶部
      const index = conversations.value.indexOf(conversation);
      conversations.value.splice(index, 1);
      conversations.value.unshift(conversation);
    }
  }

  function updateGroupConversationLastMessage(
    groupId: string,
    messageInfo: any,
  ) {
    const conversation = conversations.value.find(
      (c) =>
        c.type === "group" &&
        (c.group_id === String(groupId) || c.id === String(groupId)),
    );
    if (conversation) {
      conversation.last_message = messageInfo.content;
      conversation.last_message_time = messageInfo.time;
      const index = conversations.value.indexOf(conversation);
      conversations.value.splice(index, 1);
      conversations.value.unshift(conversation);
    }
  }

  function setConversationPinned(conversationId: string, pinned: boolean) {
    const conversation = conversations.value.find(
      (c) => c.id === String(conversationId),
    );
    if (conversation) {
      conversation.is_pinned = pinned;
    }
  }

  function setConversationMuted(conversationId: string, muted: boolean) {
    const conversation = conversations.value.find(
      (c) => c.id === String(conversationId),
    );
    if (conversation) {
      conversation.is_muted = muted;
    }
  }

  function incrementUnreadCount(conversationId: string) {
    const conversation = conversations.value.find(
      (c) => c.id === String(conversationId),
    );
    if (conversation) {
      conversation.unread_count = (conversation.unread_count || 0) + 1;
    }
  }

  function resetConversationUnreadCount(conversationId: string) {
    const conversation = conversations.value.find(
      (c) => c.id === String(conversationId),
    );
    if (conversation) {
      conversation.unread_count = 0;
    }
  }

  function getConversationUnreadCount(conversationId: string): number {
    const conversation = conversations.value.find(
      (c) => c.id === String(conversationId),
    );
    return conversation?.unread_count || 0;
  }

  function updateMessageStatus(messageId: string, status: string) {
    const message = messages.value.find((m) => m.id === messageId);
    if (message) {
      // 更新消息状态
    }
  }

  function updateUserOnlineStatus(userId: string, isOnline: boolean) {
    userOnlineStatus.value.set(userId, isOnline);
  }

  function updateMessageId(oldId: string, newId: string) {
    const message = messages.value.find((m) => m.id === oldId);
    if (message) {
      message.id = newId;
    }
    optimisticMessages.value.delete(oldId);
  }

  // 类型转换函数
  // 移除数据转换函数 - 直接信任后端数据
  // 原则：接收后直接使用，不做额外处理和验证
  // 
  // ❌ 错误做法：重新生成ID、强制类型转换、提供默认值
  // ✅ 正确做法：直接使用后端返回的数据结构

  function removeOptimisticMessage(messageId: string) {
    const index = messages.value.findIndex((m) => m.id === messageId);
    if (index > -1) {
      messages.value.splice(index, 1);
    }
    optimisticMessages.value.delete(messageId);
  }

  return {
    // 只读选择器（返回拷贝，避免外部修改内部状态）
    conversations: computed(() => conversations.value.slice()),
    currentConversation: computed(() =>
      currentConversation.value ? { ...currentConversation.value } : null,
    ),
    messages: computed(() => messages.value.slice()),
    pagination: computed(() => ({
      conversations: { ...pagination.value.conversations },
      messages: { ...pagination.value.messages },
    })),
    loading: computed(() => ({ ...loading.value })),

    // 计算属性
    totalUnreadCount,

    // 方法
    fetchConversations,
    searchConversations,
    createConversation,
    createOrGetConversation,
    searchUsers,
    fetchMessages,
    sendMessage,
    sendGroupMessage,
    markConversationAsRead,
    deleteConversation,
    setCurrentConversation,
    clearMessages,

    // 实时消息功能
    addRealtimeMessage,
    updateConversationLastMessage,
    updateGroupConversationLastMessage,
    setConversationPinned,
    setConversationMuted,
    incrementUnreadCount,
    resetConversationUnreadCount,
    getConversationUnreadCount,
    updateMessageStatus,
    updateUserOnlineStatus,
    updateMessageId,
    removeOptimisticMessage,
    retryMessage,
  };
});
