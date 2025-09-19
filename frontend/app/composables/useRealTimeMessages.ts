/**
 * 实时消息处理 - 简化版
 * 专注核心实时消息功能，移除过度复杂的设计
 */

export const useRealTimeMessages = () => {
  const webSocket = useWebSocket();
  const messagesStore = useMessagesStore();
  const authStore = useAuthStore();
  const toast = useSmartToast();

  // 当前活跃的对话
  const activeConversationId = ref<string | null>(null);
  const activeRoomType = ref<"user" | "group" | null>(null);

  // 设置活跃对话
  const setActiveConversation = (
    conversationId: string | null,
    type: "user" | "group" = "user",
  ) => {
    // 离开之前的房间
    if (activeConversationId.value && webSocket.isConnected) {
      const prevType =
        activeRoomType.value === "group" ? "group" : "conversation";
      webSocket.leaveRoom(String(activeConversationId.value), prevType);
    }

    activeConversationId.value = conversationId;
    activeRoomType.value = conversationId ? type : null;

    // 加入新的房间
    if (conversationId && webSocket.isConnected) {
      const roomType = type === "group" ? "group" : "conversation";
      webSocket.joinRoom(String(conversationId), roomType);
    }
  };

  // 处理新消息
  const handleNewMessage = (data: any) => {
    try {
      // 适配消息格式
      const message = {
        id: String(Date.now()),
        conversation_id: String(data.room_id || ""),
        sender_id: String(data.sender_id), // 转换为字符串
        sender_name: String(data.sender_nickname || ""),
        content: String(data.content || ""),
        message_type: "text" as const,
        created_at: data.created_at || new Date().toISOString(),
        is_read: false,
      };

      // 更新消息存储
      messagesStore.addRealtimeMessage(message);

      // 如果不是当前活跃对话，显示通知
      if (String(message.conversation_id) !== activeConversationId.value) {
        if (message.sender_id !== String(authStore.user?.user_id)) {
          showNotification(message);
        }
      }
    } catch (error) {
      console.error("处理新消息失败:", error);
    }
  };

  // 显示通知
  const showNotification = (message: any) => {
    // 只在页面不可见时显示系统通知
    if (
      typeof document !== "undefined" &&
      document.visibilityState === "hidden"
    ) {
      if ("Notification" in window && Notification.permission === "granted") {
        try {
          const notification = new Notification(`${message.sender_name}`, {
            body: message.content?.slice(0, 50) || "新消息",
          });
          if (notification && typeof window !== "undefined") {
            notification.onclick = () => {
              window.focus();
              notification.close();
            };
          }
        } catch (e) {
          console.warn("显示通知失败:", e);
        }
      }
    }

    // 显示应用内提示
    if (message.sender_id !== String(authStore.user?.user_id)) {
      toast.info({
        title: message.sender_name,
        description: message.content?.slice(0, 50) || "新消息"
      });
    }
  };

  // 初始化实时消息
  const initializeRealTime = () => {
    // 注册消息监听器
    webSocket.addEventListener("newMessage", handleNewMessage);

    // 监听连接状态变化
    watch(
      () => webSocket.isConnected,
      (connected) => {
        if (connected && activeConversationId.value) {
          // 重新加入当前房间
          const roomType =
            activeRoomType.value === "group" ? "group" : "conversation";
          webSocket.joinRoom(activeConversationId.value.toString(), roomType);
        }
      },
    );

    // 清理函数
    if (getCurrentInstance()) {
      onUnmounted(() => {
        webSocket.removeEventListener("newMessage", handleNewMessage);
      });
    }
  };

  // 发送消息
  const sendRealtimeMessage = async (
    conversationId: string,
    content: string,
    type: "text" | "image" | "file" = "text",
  ) => {
    try {
      // 乐观更新
      const optimisticId = String(Date.now());
      const optimisticMessage = {
        id: optimisticId,
        conversation_id: String(conversationId),
        sender_id: String(authStore.user?.user_id || "0"),
        sender_name: authStore.user?.nickname || "我",
        content,
        message_type: type,
        created_at: new Date().toISOString(),
        is_read: false,
      };

      // 立即更新UI
      messagesStore.addRealtimeMessage(optimisticMessage);

      // 发送到服务器
      const sentMessage = await messagesStore.sendMessage(
        String(conversationId),
        content,
        type,
      );

      // 更新消息ID
      const messageId = sentMessage.message_id || sentMessage.id;
      if (messageId) {
        messagesStore.updateMessageId(optimisticId, messageId);
      }

      return sentMessage;
    } catch (error) {
      console.error("发送消息失败:", error);
      throw error;
    }
  };

  return {
    // 状态
    activeConversationId: readonly(activeConversationId),

    // 方法
    initializeRealTime,
    setActiveConversation,
    sendRealtimeMessage,
  };
};
