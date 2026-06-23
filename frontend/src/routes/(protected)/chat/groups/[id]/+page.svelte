<script lang="ts">
  import { dev } from '$app/environment';
  import { page } from '$app/state';
  import { onMount, tick, untrack } from 'svelte';
  import { navigateToChatBackTarget } from '../../detail-back';
  import { withMinimumLoadingTime } from '$lib/utils/loading';
  import { groupsService, groupMessagesService } from '$lib/api/services';
  import { isSuccess } from '$lib/api/client';
  import { authStore } from '$lib/stores/auth.svelte';
  import { messageStore } from '$lib/stores/message.svelte';
  import { toast } from '$lib/stores/toast.svelte';
  import { websocketStore, eventHandlers } from '$lib/websocket';
  import type { GroupMessageNewPayload } from '$lib/websocket/types';
  import { useBatchMarkRead } from '$lib/hooks/use-batch-mark-read.svelte';
  import {
    createReconnectTracker,
    mergeMessagesByIdAndSentAt,
    resolveLatestVisibleMessageId
  } from '$lib/hooks/use-chat-realtime-sync';
  import { useChatWsSubscription } from '$lib/hooks/use-chat-ws-subscription.svelte';
  import { autofocus, useKeyboardFocus, KEYBOARD_DISMISS_EVENT } from '$lib/attachments';
  import { platformStore } from '$lib/stores/platform.svelte';
  import type { GroupItem, GroupMessageItem } from '$lib/types/api';
  import BackButton from '$lib/components/common/back-button.svelte';
  import Avatar from '$lib/components/common/avatar.svelte';
  import Send from 'lucide-svelte/icons/send';
  import Loader2 from 'lucide-svelte/icons/loader-2';
  import MoreVertical from 'lucide-svelte/icons/more-vertical';
  import MessageCircle from 'lucide-svelte/icons/message-circle';
  import MapPin from 'lucide-svelte/icons/map-pin';
  import ChevronDown from 'lucide-svelte/icons/chevron-down';
  import X from 'lucide-svelte/icons/x';
  import ChatBubble from '$lib/components/chat/chat-bubble.svelte';

  const groupId = $derived(page.params.id!);

  let group = $state<GroupItem | null>(null);
  let messages = $state.raw<GroupMessageItem[]>([]);
  let isLoading = $state(true);
  let messageText = $state('');
  let isSending = $state(false);
  let cursor = $state<string | undefined>(undefined);
  let hasMore = $state(true);
  let isLoadingMore = $state(false);
  let groupTypingUserIds = $state.raw<string[]>([]);
  let scrollContainer: HTMLDivElement | null = $state(null);
  let showScrollButton = $state(false);
  let selectedImage = $state<string | null>(null);
  let shouldFocusInput = $state(0); // 用于触发重新聚焦
  let isMounted = false; // 组件是否挂载（防止异步操作在卸载后更新状态）
  const MESSAGE_CONTENT_MAX = 5000;
  const MESSAGE_MENTIONS_MAX = 20;
  const GROUP_TYPING_INDICATOR_TTL_MS = 4500;
  const GROUP_TYPING_NAME_PREVIEW_LIMIT = 3;
  const groupTypingTimers = new Map<string, ReturnType<typeof setTimeout>>();
  let isHandlingBackNavigation = false;
  const reconnectTracker = createReconnectTracker();
  const wsSubscription = useChatWsSubscription();

  // spec: 群消息发送通过 HTTP API，不应依赖 WS 订阅状态
  const canSendMessage = $derived(group?.my_state?.is_joined && !group?.my_state?.is_muted);

  function clearGroupTypingTimer(userId: string) {
    const timer = groupTypingTimers.get(userId);
    if (!timer) return;
    clearTimeout(timer);
    groupTypingTimers.delete(userId);
  }

  function markGroupUserTypingStopped(userId: string) {
    clearGroupTypingTimer(userId);
    groupTypingUserIds = groupTypingUserIds.filter((id) => id !== userId);
  }

  function markGroupUserTypingActive(userId: string) {
    if (!groupTypingUserIds.includes(userId)) {
      groupTypingUserIds = [...groupTypingUserIds, userId];
    }
    clearGroupTypingTimer(userId);
    groupTypingTimers.set(
      userId,
      setTimeout(() => {
        groupTypingTimers.delete(userId);
        groupTypingUserIds = groupTypingUserIds.filter((id) => id !== userId);
      }, GROUP_TYPING_INDICATOR_TTL_MS)
    );
  }

  function resetGroupTypingUsers() {
    for (const timer of groupTypingTimers.values()) {
      clearTimeout(timer);
    }
    groupTypingTimers.clear();
    groupTypingUserIds = [];
  }

  function resolveGroupTypingDisplayName(userId: string): string {
    for (let index = messages.length - 1; index >= 0; index--) {
      const message = messages[index];
      if (!message || message.sender?.user_id !== userId) continue;
      const nickname = message.sender.nickname?.trim();
      if (nickname) return nickname;
    }
    const suffix = userId.slice(-4);
    return `成员${suffix || userId}`;
  }

  function buildGroupTypingHintText(): string {
    const count = groupTypingUserIds.length;
    if (count === 0) return '';
    const previewNames = groupTypingUserIds
      .slice(0, GROUP_TYPING_NAME_PREVIEW_LIMIT)
      .map((id) => resolveGroupTypingDisplayName(id));
    if (count === 1) {
      return `${previewNames[0]} 正在输入...`;
    }
    if (count <= GROUP_TYPING_NAME_PREVIEW_LIMIT) {
      return `${previewNames.join('、')} 正在输入...`;
    }
    return `${previewNames.join('、')} 等 ${count} 人正在输入...`;
  }

  // ✨ 批量已读逻辑
  const batchMarkRead = useBatchMarkRead({
    markAsRead: async (messageIds) => {
      const upToMessageId = resolveLatestVisibleMessageId(messages, messageIds ?? []);
      await groupMessagesService.markAsRead({
        group_id: groupId,
        ...(upToMessageId ? { up_to_message_id: upToMessageId } : {})
      });
    },
    debounceMs: 2000,
    visibilityThreshold: 1000
  });

  async function loadGroup() {
    try {
      const response = await groupsService.getGroup({ group_id: groupId });
      if (!isMounted) return;
      if (isSuccess(response) && response.data) {
        group = response.data;
      } else {
        toast.error(response.message || '群组不存在');
      }
    } catch {
      if (isMounted) toast.error('加载群组失败');
    }
  }

  async function loadMessages(loadMore = false) {
    if (loadMore) {
      if (!hasMore || isLoadingMore) return;
      isLoadingMore = true;
    }

    try {
      const response = await groupMessagesService.getHistory({
        group_id: groupId,
        pagination: {
          limit: 50,
          cursor: loadMore ? cursor : undefined
        }
      });

      if (!isMounted) return;
      if (isSuccess(response) && response.data) {
        const newMessages = response.data.items.reverse();

        const scrollHeight = scrollContainer?.scrollHeight || 0;
        const scrollTop = scrollContainer?.scrollTop || 0;

        if (loadMore) {
          messages = [...newMessages, ...messages];

          // 加载历史消息后保持滚动位置
          await tick();
          if (scrollContainer) {
            const newScrollHeight = scrollContainer.scrollHeight;
            scrollContainer.scrollTop = newScrollHeight - scrollHeight + scrollTop;
          }
        } else {
          messages = newMessages;
          // 初始加载滚动到底部
          await tick();
          scrollToBottom(false);
        }

        cursor = response.data.next_cursor;
        hasMore = !!response.data.next_cursor;

        // ✅ 不再立即标记已读，等待消息可见时自动标记
      }
    } catch {
      if (isMounted) toast.error('加载消息失败');
    } finally {
      if (loadMore) {
        isLoadingMore = false;
      } else {
        isLoading = false;
      }
    }
  }

  async function backfillRecentMessagesAfterReconnect() {
    try {
      const response = await groupMessagesService.getHistory({
        group_id: groupId,
        pagination: { limit: 50 }
      });
      if (!isMounted || !isSuccess(response) || !response.data) return;
      const recentMessages = response.data.items.reverse();
      if (recentMessages.length === 0) return;
      messages = mergeMessagesByIdAndSentAt(messages, recentMessages);
    } catch (error) {
      if (dev) console.warn('[GroupChat] 重连回补失败:', error);
    }
  }

  async function sendMessage() {
    const text = messageText.trim();
    if (!text || !canSendMessage) return;
    if (text.length > MESSAGE_CONTENT_MAX) {
      toast.error(`消息内容最多 ${MESSAGE_CONTENT_MAX} 字符`);
      return;
    }

    isSending = true;
    messageText = '';

    try {
      const mentions = extractMentions(text);
      if (mentions.length > MESSAGE_MENTIONS_MAX) {
        messageText = text;
        toast.error(`@ 提及最多 ${MESSAGE_MENTIONS_MAX} 人`);
        return;
      }

      const response = await groupMessagesService.send({
        group_id: groupId,
        type: 'text',
        content: text,
        mentions: mentions.length > 0 ? mentions : undefined
      });

      if (!isSuccess(response)) {
        messageText = text;
        toast.error(response.message || '发送失败');
      } else {
        // 发送成功后滚动到底部并重新聚焦输入框
        await tick();
        scrollToBottom(true);
        shouldFocusInput++; // 触发重新聚焦
      }
    } catch {
      messageText = text;
      toast.error('发送失败');
    } finally {
      eventHandlers.sendGroupTyping(groupId, false);
      isSending = false;
    }
  }

  function extractMentions(text: string): string[] {
    const mentionRegex = /@(\d+)/g;
    const mentions: string[] = [];
    let match;

    while ((match = mentionRegex.exec(text)) !== null) {
      const userId = match[1]!;
      if (!mentions.includes(userId)) {
        mentions.push(userId);
        if (mentions.length >= MESSAGE_MENTIONS_MAX) {
          break;
        }
      }
    }

    return mentions;
  }

  function formatDateDivider(dateStr: string): string {
    const date = new Date(dateStr);
    const now = new Date();

    if (date.toDateString() === now.toDateString()) {
      return '今天';
    }

    const yesterday = new Date(now);
    yesterday.setDate(yesterday.getDate() - 1);
    if (date.toDateString() === yesterday.toDateString()) {
      return '昨天';
    }

    return date.toLocaleDateString('zh-CN', { year: 'numeric', month: 'long', day: 'numeric' });
  }

  function isSelf(message: GroupMessageItem): boolean {
    return message.sender?.user_id === authStore.currentUserId;
  }

  function isSystemMessage(message: GroupMessageItem): boolean {
    return message.type === 'system';
  }

  /**
   * Svelte 5 Attachment：观察消息可见性
   */
  function observeMessage(params: { messageId: string; isSelf: boolean }) {
    return (node: HTMLElement) => {
      // 只观察其他人发送的消息（不观察自己发送的消息）
      if (!params.isSelf) {
        batchMarkRead.observe(node, params.messageId);
      }

      return () => {
        batchMarkRead.unobserve(node);
      };
    };
  }

  function shouldShowAvatar(index: number): boolean {
    if (index === messages.length - 1) return true;

    const current = messages[index]!;
    const next = messages[index + 1]!;

    if (isSelf(current) !== isSelf(next)) return true;
    if (current.sender?.user_id !== next.sender?.user_id) return true;

    const timeDiff = new Date(next.sent_at).getTime() - new Date(current.sent_at).getTime();
    return timeDiff > 60000; // 1分钟
  }

  function shouldShowName(index: number): boolean {
    if (index === 0) return true;

    const current = messages[index]!;
    const prev = messages[index - 1]!;

    if (isSelf(current)) return false;
    if (isSelf(prev) !== isSelf(current)) return true;
    if (prev.sender?.user_id !== current.sender?.user_id) return true;

    const timeDiff = new Date(current.sent_at).getTime() - new Date(prev.sent_at).getTime();
    return timeDiff > 60000;
  }

  function scrollToBottom(smooth = true) {
    if (scrollContainer) {
      scrollContainer.scrollTo({
        top: scrollContainer.scrollHeight,
        behavior: smooth ? 'smooth' : 'auto'
      });
    }
  }

  function handleScroll() {
    if (!scrollContainer) return;

    const { scrollTop, scrollHeight, clientHeight } = scrollContainer;
    const isNearBottom = scrollHeight - scrollTop - clientHeight < 200;
    showScrollButton = !isNearBottom;

    // 滚动到顶部时加载更多
    if (scrollTop < 100 && hasMore && !isLoadingMore) {
      loadMessages(true);
    }
  }

  function openImagePreview(url: string) {
    selectedImage = url;
  }

  function closeImagePreview() {
    selectedImage = null;
  }

  async function handleBackClick() {
    if (isHandlingBackNavigation) {
      return;
    }

    isHandlingBackNavigation = true;
    try {
      await navigateToChatBackTarget(typeof window === 'undefined' ? undefined : window);
    } finally {
      isHandlingBackNavigation = false;
    }
  }

  async function init() {
    await withMinimumLoadingTime(async () => {
      await loadGroup();
      await loadMessages();
    });
  }

  onMount(() => {
    isMounted = true;

    let keyboardDismissHandler: (() => void) | null = null;
    if (typeof window !== 'undefined') {
      keyboardDismissHandler = () => {
        eventHandlers.sendGroupTyping(groupId, false);
      };
      window.addEventListener(KEYBOARD_DISMISS_EVENT, keyboardDismissHandler);
    }

    // 注册 WebSocket 监听器（使用 reactive groupId 自动过滤）
    const unsubscribeMessage = eventHandlers.onGroupMessage((payload: GroupMessageNewPayload) => {
      if (payload.group_id === groupId) {
        // spec: 按 message_id 去重，避免 HTTP 补拉与 WS 推送重复渲染
        if (messages.some((m) => m.message_id === payload.message_id)) return;
        markGroupUserTypingStopped(payload.sender.user_id);

        const newMessage: GroupMessageItem = {
          message_id: payload.message_id,
          type: payload.type,
          content: payload.content,
          sender: payload.sender,
          sent_at: payload.sent_at,
          media: payload.media,
          reply_to: payload.reply_to,
          mentions: payload.mentions
        };

        messages = [...messages, newMessage];
        messageStore.markConversationAsRead(`group_${groupId}`);

        if (scrollContainer) {
          const { scrollTop, scrollHeight, clientHeight } = scrollContainer;
          const isNearBottom = scrollHeight - scrollTop - clientHeight < 200;
          if (isNearBottom) {
            setTimeout(() => scrollToBottom(true), 100);
          }
        }
      }
    });

    const unsubscribeRecalled = websocketStore.on('GroupMessageRecalled', (payload) => {
      if (payload.group_id === groupId) {
        messages = messages.map((msg) =>
          msg.message_id === payload.message_id
            ? { ...msg, recalled_at: payload.recalled_at, recalled_by: payload.recalled_by }
            : msg
        );
      }
    });

    const unsubscribeEdited = websocketStore.on('GroupMessageEdited', (payload) => {
      if (payload.group_id === groupId) {
        messages = messages.map((msg) =>
          msg.message_id === payload.message_id ? { ...msg, content: payload.content } : msg
        );
      }
    });

    const unsubscribeMemberJoined = websocketStore.on('GroupMemberJoined', (payload) => {
      if (payload.group_id === groupId) {
        toast.success(`${payload.user.nickname} 加入了群组`);
        if (group) {
          group = { ...group, group: { ...group.group, members_count: payload.member_count } };
        }
      }
    });

    const unsubscribeGroupTyping = websocketStore.on('GroupUserTyping', (payload) => {
      if (payload.group_id !== groupId) return;
      if (payload.user_id === authStore.currentUserId) return;
      if (payload.is_typing) {
        markGroupUserTypingActive(payload.user_id);
        return;
      }
      markGroupUserTypingStopped(payload.user_id);
    });

    return () => {
      isMounted = false;
      eventHandlers.cancelAllTyping();
      resetGroupTypingUsers();
      unsubscribeMessage();
      unsubscribeRecalled();
      unsubscribeEdited();
      unsubscribeMemberJoined();
      unsubscribeGroupTyping();
      if (keyboardDismissHandler && typeof window !== 'undefined') {
        window.removeEventListener(KEYBOARD_DISMISS_EVENT, keyboardDismissHandler);
      }
      wsSubscription.cleanup('group', [groupId]);
    };
  });

  // groupId 变化时重置状态并重新加载（含首次加载）
  $effect(() => {
    const id = groupId;
    untrack(() => {
      eventHandlers.cancelAllTyping();
      group = null;
      messages = [];
      isLoading = true;
      messageText = '';
      cursor = undefined;
      hasMore = true;
      showScrollButton = false;
      selectedImage = null;
      resetGroupTypingUsers();
      wsSubscription.reset();
      reconnectTracker.reset();
      messageStore.markConversationAsRead(`group_${id}`);
      init();
    });
  });

  // WebSocket 事件处理（Svelte 5 $effect，自动清理）
  $effect(() => {
    return websocketStore.on('GroupMemberLeft', (payload) => {
      if (payload.group_id !== groupId) return;
      if (group) {
        group = { ...group, group: { ...group.group, members_count: payload.member_count } };
      }
    });
  });

  $effect(() => {
    return websocketStore.on('GroupMemberRemoved', (payload) => {
      if (payload.group_id !== groupId) return;
      if (group) {
        group = { ...group, group: { ...group.group, members_count: payload.member_count } };
      }
      if (payload.user.user_id === authStore.currentUserId) {
        loadGroup();
      }
    });
  });

  $effect(() => {
    return websocketStore.on('GroupUpdated', (payload) => {
      if (payload.group_id !== groupId || !group) return;
      group = {
        ...group,
        group: {
          ...group.group,
          ...(payload.updated_fields.name && { name: payload.updated_fields.name }),
          ...(payload.updated_fields.avatar_url && {
            avatar_url: payload.updated_fields.avatar_url
          })
        }
      };
    });
  });

  /**
   * 响应式订阅管理（等待 WebSocket 连接完成）
   * 使用 $effect 监听连接状态，自动订阅/恢复订阅
   */
  $effect(() => {
    let cancelled = false;

    // 使用 derived 状态检查连接
    const connected = websocketStore.isConnected;
    const { shouldBackfill } = reconnectTracker.update(
      connected,
      websocketStore.lastDisconnectDurationMs
    );

    if (!connected) {
      // 连接断开，重置订阅状态
      if (wsSubscription.state.isSubscribed) {
        if (dev) console.log('[GroupChat] WebSocket 断开，重置订阅状态');
      }
      resetGroupTypingUsers();
      wsSubscription.sync({
        connected,
        type: 'group',
        ids: [groupId]
      });
      return () => {
        cancelled = true;
      };
    }
    if (shouldBackfill) {
      void backfillRecentMessagesAfterReconnect();
    }

    wsSubscription.sync({
      connected,
      type: 'group',
      ids: [groupId],
      onSubscribeFailed: () => {
        if (cancelled) return;
        if (dev) console.warn('[GroupChat] ❌ 订阅失败，将在重连后重试');
        toast.warning('消息订阅失败，正在重试...');
      }
    });

    return () => {
      cancelled = true;
    };
  });
</script>

<svelte:head>
  <title>{group?.group.name ?? '群聊'} - HotSpot</title>
</svelte:head>

<!-- 聊天容器 - Neumorphic 风格 -->
<div
  class="flex min-w-0 flex-col overflow-hidden"
  style="{platformStore.isWide
    ? 'height: 100%; width: 100%; flex: 1; border: 1px solid var(--border-theme);'
    : 'height: calc(100dvh - env(safe-area-inset-bottom, 0px)); width: 100%;'} background: var(--surface-bg);"
>
  <!-- Header -->
  <header
    class="flex h-16 shrink-0 items-center gap-3 px-4"
    style="background-color: var(--surface-bg); border-bottom: 1px solid var(--border-theme);"
  >
    {#if !platformStore.isWide}
      <BackButton onclick={handleBackClick} size={36} />
    {/if}

    {#if isLoading}
      <div
        class="rounded-full"
        style="width: 40px; height: 40px; background: var(--surface-bg); box-shadow: var(--shadow-inset);"
      ></div>
      <div
        style="width: 128px; height: 20px; border-radius: 8px; background: var(--surface-bg); box-shadow: var(--shadow-inset);"
      ></div>
    {:else if group}
      <a
        href="/groups/{groupId}"
        class="flex flex-1 items-center gap-3 overflow-hidden transition-opacity hover:opacity-80"
      >
        <Avatar
          src={group.group.avatar_url}
          name={group.group.name}
          alt={group.group.name}
          size={40}
          type="group"
        />
        <div class="flex-1 overflow-hidden">
          <div
            class="truncate"
            style="font-weight: 600; font-size: 15px; color: var(--text-primary);"
          >
            {group.group.name}
          </div>
          <div class="truncate" style="font-size: 11px; color: var(--text-secondary);">
            {group.group.members_count} 位成员
          </div>
        </div>
      </a>
    {/if}

    <button
      class="surface-flat flex shrink-0 items-center justify-center rounded-full outline-none transition-transform duration-150 active:scale-[0.88]"
      style="width: 36px; height: 36px;"
      aria-label="更多选项"
    >
      <MoreVertical size={15} strokeWidth={1.5} color="var(--text-secondary)" />
    </button>
  </header>

  <!-- Messages Container -->
  <div
    bind:this={scrollContainer}
    onscroll={handleScroll}
    class="relative flex-1 overflow-y-auto overscroll-contain"
    style="background: var(--surface-bg);"
  >
    {#if isLoading}
      <div class="space-y-4 p-4">
        {#each Array(8) as _, i (i)}
          <div class="flex gap-3">
            <div
              class="shrink-0 rounded-full"
              style="width: 40px; height: 40px; background: var(--surface-bg); box-shadow: var(--shadow-inset);"
            ></div>
            <div class="flex-1 space-y-2">
              <div
                style="width: 96px; height: 12px; border-radius: 6px; background: var(--surface-bg); box-shadow: var(--shadow-inset);"
              ></div>
              <div
                class="max-w-xs rounded-[var(--radius-md)]"
                style="width: 100%; height: 64px; background: var(--surface-bg); box-shadow: var(--shadow-inset);"
              ></div>
            </div>
          </div>
        {/each}
      </div>
    {:else}
      {#if hasMore && messages.length > 0 && !isLoadingMore}
        <div class="flex justify-center py-3">
          <button
            class="rounded-full px-4 py-1.5 text-xs outline-none"
            style="background: var(--surface-bg); box-shadow: var(--shadow-md); border: 1px solid var(--border-theme); color: var(--text-secondary);"
            onclick={() => loadMessages(true)}
          >
            加载更多消息
          </button>
        </div>
      {/if}

      {#if isLoadingMore}
        <div class="flex justify-center py-3">
          <Loader2 class="h-5 w-5 animate-spin" color="var(--text-secondary)" />
        </div>
      {/if}

      {#if messages.length === 0}
        <div class="flex h-full items-center justify-center p-8">
          <div class="text-center">
            <div
              class="mb-4 inline-flex items-center justify-center rounded-full"
              style="width: 64px; height: 64px; box-shadow: var(--shadow-sm); border: 1px solid var(--border-theme);"
            >
              <MessageCircle size={28} color="var(--text-secondary)" strokeWidth={1.5} />
            </div>
            <p
              style="font-weight: 500; font-size: 15px; color: var(--text-primary); margin-bottom: 4px;"
            >
              还没有消息
            </p>
            <p style="font-size: 13px; color: var(--text-secondary);">发送第一条消息开始聊天</p>
          </div>
        </div>
      {:else}
        <div class="space-y-1 px-4 py-2">
          {#each messages as message, index (message.message_id)}
            {@const showDateDivider =
              index === 0 ||
              new Date(message.sent_at).toDateString() !==
                new Date(messages[index - 1]!.sent_at).toDateString()}
            {@const showAvatar = shouldShowAvatar(index)}
            {@const showName = shouldShowName(index)}
            {@const isMe = isSelf(message)}

            {#if showDateDivider}
              <div class="flex justify-center py-4">
                <span
                  class="rounded-full px-3 py-1"
                  style="font-size: 11px; font-weight: 500; color: var(--text-secondary); background: var(--surface-bg); border: 1px solid var(--border-theme);"
                >
                  {formatDateDivider(message.sent_at)}
                </span>
              </div>
            {/if}

            {#if isSystemMessage(message)}
              <div class="flex justify-center py-2">
                <span
                  class="rounded-full px-3 py-1"
                  style="font-size: 12px; color: var(--text-secondary); background: var(--surface-bg); border: 1px solid var(--border-theme);"
                >
                  {message.content}
                </span>
              </div>
            {:else}
              <div
                class="group flex gap-3 py-1 {isMe ? 'flex-row-reverse' : ''} {index ===
                messages.length - 1
                  ? 'chat-msg-enter'
                  : ''}"
                {@attach observeMessage({ messageId: message.message_id, isSelf: isMe })}
              >
                {#if !isMe}
                  <div class="w-10 shrink-0">
                    {#if showAvatar}
                      <a href="/users/{message.sender?.user_id}">
                        <Avatar
                          src={message.sender?.avatar_url}
                          name={message.sender?.nickname}
                          alt={message.sender?.nickname}
                          size={40}
                        />
                      </a>
                    {/if}
                  </div>
                {/if}

                <div class="flex max-w-[70%] flex-col gap-0.5 {isMe ? 'items-end' : 'items-start'}">
                  {#if !isMe && showName}
                    <span
                      class="px-3"
                      style="font-size: 12px; font-weight: 500; color: var(--text-secondary);"
                    >
                      {message.sender?.nickname}
                    </span>
                  {/if}

                  <ChatBubble
                    isSelf={isMe}
                    type={message.type}
                    content={message.content}
                    media={message.media}
                    isRecalled={message.recalled_at != null}
                    onImageClick={openImagePreview}
                  >
                    {#if message.type === 'share' && message.share_code}
                      <div class="flex items-center gap-2" style="font-size: 13px;">
                        <MapPin size={14} style="flex-shrink: 0;" color="var(--text-secondary)" />
                        <div>
                          <p style="font-weight: 500;">分享消息</p>
                          {#if message.share_code}
                            <a
                              href="/share/{message.share_code}"
                              style="font-size: 12px; opacity: 0.8; color: var(--brand);"
                            >
                              查看分享内容
                            </a>
                          {/if}
                        </div>
                      </div>
                    {/if}
                  </ChatBubble>

                  {#if showAvatar}
                    <span
                      class="px-3"
                      style="font-size: 10px; color: var(--text-secondary); opacity: 0.6;"
                    >
                      {new Date(message.sent_at).toLocaleTimeString('zh-CN', {
                        hour: '2-digit',
                        minute: '2-digit'
                      })}
                    </span>
                  {/if}
                </div>
              </div>
            {/if}
          {/each}
        </div>
      {/if}
    {/if}

    <!-- Scroll to Bottom Button -->
    {#if showScrollButton}
      <button
        class="fixed bottom-24 right-6 z-10 flex items-center justify-center rounded-full outline-none"
        style="width: 36px; height: 36px; background: var(--brand); color: var(--text-on-accent); box-shadow: var(--shadow-brand-btn);"
        onclick={() => scrollToBottom(true)}
        aria-label="滚动到底部"
      >
        <ChevronDown size={20} />
      </button>
    {/if}
  </div>

  <!-- Input Area -->
  <div
    class="shrink-0 p-4"
    style="background-color: var(--surface-bg); border-top: 1px solid var(--border-theme);"
  >
    {#if groupTypingUserIds.length > 0}
      <div class="mb-2 px-1">
        <p style="font-size: 12px; color: var(--text-secondary);">
          {buildGroupTypingHintText()}
        </p>
      </div>
    {/if}

    {#if !wsSubscription.state.isSubscribed && !isLoading}
      <div
        class="rounded-[16px] px-4 py-3 text-center mb-3"
        style="border: 1px solid var(--theme-warning-border); background: var(--theme-warning-border);"
      >
        <p
          class="flex items-center justify-center gap-2"
          style="font-size: 13px; color: var(--text-secondary);"
        >
          <Loader2 class="h-4 w-4 animate-spin" />
          {#if wsSubscription.state.isAttemptingSubscribe || !websocketStore.isConnected}
            实时消息连接中（不影响发送）
          {:else}
            实时订阅重试中（不影响发送）
          {/if}
        </p>
      </div>
    {/if}

    {#if !group?.my_state?.is_joined}
      <div
        class="rounded-[16px] px-4 py-3 text-center"
        style="border: 1px solid var(--border-theme);"
      >
        <p style="font-size: 13px; color: var(--text-secondary);">你还不是群成员，无法发送消息</p>
      </div>
    {:else if group?.my_state?.is_muted}
      <div
        class="rounded-[16px] px-4 py-3 text-center"
        style="border: 1px solid var(--border-theme);"
      >
        <p style="font-size: 13px; color: var(--text-secondary);">你已被禁言，无法发送消息</p>
      </div>
    {:else}
      <div class="flex items-end gap-2">
        <textarea
          {@attach autofocus(shouldFocusInput)}
          {@attach useKeyboardFocus()}
          placeholder="发送消息..."
          bind:value={messageText}
          rows={1}
          maxlength={MESSAGE_CONTENT_MAX}
          class="field-inset max-h-32 min-h-[42px] flex w-full resize-none rounded-[var(--radius-md)] px-4 py-2.5 text-[15px] outline-none transition-colors disabled:cursor-not-allowed disabled:opacity-50"
          style="background: var(--surface-bg); color: var(--text-primary);"
          disabled={isSending}
          oninput={() => {
            eventHandlers.sendGroupTyping(groupId, messageText.trim().length > 0);
          }}
          onkeydown={(e) => {
            if (e.key === 'Enter' && !e.shiftKey) {
              e.preventDefault();
              sendMessage();
            }
          }}
        ></textarea>
        <button
          class="flex shrink-0 items-center justify-center rounded-full outline-none transition-transform duration-150 active:scale-[0.9]"
          style="width: 36px; height: 36px; {messageText.trim()
            ? 'background: var(--brand); color: var(--text-on-accent); box-shadow: var(--shadow-brand-btn);'
            : 'background: var(--surface-bg); color: var(--text-secondary); border: 1px solid var(--border-theme);'}"
          onclick={sendMessage}
          onmousedown={(e) => e.preventDefault()}
          disabled={!messageText.trim() || isSending}
          aria-label="发送消息"
        >
          {#if isSending}
            <Loader2 class="h-5 w-5 animate-spin" />
          {:else}
            <Send size={20} />
          {/if}
        </button>
      </div>
    {/if}
  </div>
</div>

<!-- Image Preview Modal -->
{#if selectedImage}
  <div
    class="fixed inset-0 z-50 flex items-center justify-center p-4"
    style="background: var(--theme-overlay-strong);"
    onclick={closeImagePreview}
    role="button"
    tabindex="0"
    onkeydown={(e) => e.key === 'Escape' && closeImagePreview()}
  >
    <button
      type="button"
      class="absolute right-4 top-4 flex items-center justify-center rounded-full outline-none"
      style="width: 40px; height: 40px; background: var(--theme-surface-glass); color: var(--text-on-overlay);"
      onclick={closeImagePreview}
      aria-label="关闭图片预览"
    >
      <X size={24} />
    </button>
    <img
      src={selectedImage}
      alt="预览"
      class="max-h-full max-w-full rounded-[12px] object-contain"
      onclick={(e) => e.stopPropagation()}
      role="presentation"
    />
  </div>
{/if}
