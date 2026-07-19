<script lang="ts">
  import { dev } from '$app/environment';
  import { goto } from '$app/navigation';
  import { page } from '$app/state';
  import { onMount, untrack } from 'svelte';
  import { navigateToChatBackTarget } from '../../detail-back';
  import { withMinimumLoadingTime } from '$lib/utils/loading';
  import { privateChatsService, privateMessagesService } from '$lib/api/services';
  import { isSuccess } from '$lib/api/client';
  import { authStore } from '$lib/stores/auth.svelte';
  import { messageStore } from '$lib/stores/message.svelte';
  import { toast } from '$lib/stores/toast.svelte';
  import { t } from '$lib/i18n';
  import { websocketStore, eventHandlers } from '$lib/websocket';
  import type { PrivateMessageNewPayload } from '$lib/websocket/types';
  import { useBatchMarkRead } from '$lib/hooks/use-batch-mark-read.svelte';
  import {
    createReconnectTracker,
    mergeMessagesByIdAndSentAt,
    resolveLatestVisibleMessageId
  } from '$lib/hooks/use-chat-realtime-sync';
  import { useChatWsSubscription } from '$lib/hooks/use-chat-ws-subscription.svelte';
  import { KEYBOARD_DISMISS_EVENT } from '$lib/attachments';
  import { platformStore } from '$lib/stores/platform.svelte';
  import {
    DropdownMenu,
    DropdownMenuContent,
    DropdownMenuItem,
    DropdownMenuTrigger
  } from '$lib/components/ui/dropdown-menu';
  import type { PrivateMessageItem, UserInfo } from '$lib/types/api';
  import BackButton from '$lib/components/common/back-button.svelte';
  import Avatar from '$lib/components/common/avatar.svelte';
  import Trash2 from 'lucide-svelte/icons/trash-2';
  import Send from 'lucide-svelte/icons/send';
  import Spinner from '$lib/components/common/spinner.svelte';
  import EmptyState from '$lib/components/common/empty-state.svelte';
  import { Textarea } from '$lib/components/ui/textarea';
  import MoreHorizontal from 'lucide-svelte/icons/more-horizontal';
  import Phone from 'lucide-svelte/icons/phone';
  import Video from 'lucide-svelte/icons/video';
  import Smile from 'lucide-svelte/icons/smile';
  import MessageCircle from 'lucide-svelte/icons/message-circle';
  import ChatBubble from '$lib/components/chat/chat-bubble.svelte';

  const userId = $derived(page.params.id!);

  let otherUser = $state<UserInfo | null>(null);
  let messages = $state.raw<PrivateMessageItem[]>([]);
  let isLoading = $state(true);
  let messageText = $state('');
  let isSending = $state(false);
  let isOtherUserTyping = $state(false);
  let shouldFocusInput = $state(0);
  let isMounted = false;
  let isDeletingChat = $state(false);
  const MESSAGE_CONTENT_MAX = 5000;
  const TYPING_INDICATOR_TTL_MS = 4500;
  let otherUserTypingTimer: ReturnType<typeof setTimeout> | null = null;
  let isHandlingBackNavigation = false;
  const reconnectTracker = createReconnectTracker();
  const wsSubscription = useChatWsSubscription();

  // spec: 私聊消息通过 HTTP API 发送，不依赖 WS 连接或订阅状态
  const canSendMessage = $derived(authStore.isAuthenticated);

  function clearOtherUserTypingTimer() {
    if (otherUserTypingTimer) {
      clearTimeout(otherUserTypingTimer);
      otherUserTypingTimer = null;
    }
  }

  function resetOtherUserTyping() {
    clearOtherUserTypingTimer();
    isOtherUserTyping = false;
  }

  function markOtherUserTypingActive() {
    isOtherUserTyping = true;
    clearOtherUserTypingTimer();
    otherUserTypingTimer = setTimeout(() => {
      otherUserTypingTimer = null;
      isOtherUserTyping = false;
    }, TYPING_INDICATOR_TTL_MS);
  }

  // ✨ 批量已读逻辑
  const batchMarkRead = useBatchMarkRead({
    markAsRead: async (messageIds) => {
      const upToMessageId = resolveLatestVisibleMessageId(messages, messageIds ?? []);
      await privateMessagesService.markAsRead({
        target_user_id: userId,
        ...(upToMessageId ? { up_to_message_id: upToMessageId } : {})
      });
    },
    debounceMs: 2000,
    visibilityThreshold: 1000
  });

  async function loadChat() {
    try {
      await withMinimumLoadingTime(async () => {
        // spec: 使用 open_the_door 获取/创建会话 + 最近消息（首次聊天自动创建会话）
        const chatResponse = await privateChatsService.openChat({ target_user_id: userId });
        if (!isMounted) return;
        if (isSuccess(chatResponse) && chatResponse.data) {
          otherUser = chatResponse.data.chat.other_user;
          messages = (chatResponse.data.recent_messages.items ?? []).reverse();
        }
      });
    } catch {
      if (isMounted) toast.error(t('chat.loadFailed'));
    } finally {
      if (isMounted) isLoading = false;
    }
  }

  async function backfillRecentMessagesAfterReconnect() {
    try {
      const response = await privateMessagesService.getHistory({
        target_user_id: userId,
        pagination: { limit: 50 }
      });
      if (!isMounted || !isSuccess(response) || !response.data) return;
      const recentMessages = response.data.items.reverse();
      if (recentMessages.length === 0) return;
      messages = mergeMessagesByIdAndSentAt(messages, recentMessages);
    } catch (error) {
      if (dev) console.warn('[Chat] 重连回补失败:', error);
    }
  }

  async function sendMessage() {
    const text = messageText.trim();
    if (!text || !canSendMessage) return;
    if (text.length > MESSAGE_CONTENT_MAX) {
      toast.error(t('chat.contentMaxLength', { count: MESSAGE_CONTENT_MAX }));
      return;
    }

    isSending = true;
    messageText = '';

    try {
      const response = await privateMessagesService.send({
        target_user_id: userId,
        type: 'text',
        content: text
      });

      if (!isSuccess(response)) {
        messageText = text;
        toast.error(response.message || t('chat.sendFailed'));
      } else {
        // 发送成功后重新聚焦输入框
        shouldFocusInput++; // 触发重新聚焦
      }
      // ✅ 不在本地添加消息，等待 WebSocket 推送（规范要求）
    } catch {
      messageText = text;
      toast.error(t('chat.sendFailed'));
    } finally {
      isSending = false;
    }
  }

  async function deleteChat() {
    if (isDeletingChat) return;
    if (!confirm(t('chat.deleteConfirm'))) return;

    isDeletingChat = true;
    try {
      const response = await privateChatsService.deleteChat({ target_user_id: userId });
      if (!isSuccess(response)) {
        toast.error(response.message || t('chat.deleteChatFailed'));
        return;
      }
      messageStore.markConversationAsRead(`private_${userId}`);
      eventHandlers.cancelAllTyping();
      toast.success(t('chat.chatDeleted'));
      await goto('/chat');
    } catch {
      toast.error(t('chat.deleteChatFailed'));
    } finally {
      isDeletingChat = false;
    }
  }

  function handleInput() {
    eventHandlers.sendPrivateTyping(userId, true);
  }

  function formatTime(dateStr: string): string {
    const date = new Date(dateStr);
    return date.toLocaleTimeString('zh-CN', { hour: '2-digit', minute: '2-digit' });
  }

  function isSelf(message: PrivateMessageItem): boolean {
    return message.sender.user_id === authStore.currentUserId;
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

  /**
   * Svelte 5 Attachment：观察消息可见性
   */
  function observeMessage(params: { messageId: string; isSelf: boolean }) {
    return (node: HTMLElement) => {
      // 只观察对方发送的消息（不观察自己发送的消息）
      if (!params.isSelf) {
        batchMarkRead.observe(node, params.messageId);
      }

      return () => {
        batchMarkRead.unobserve(node);
      };
    };
  }

  onMount(() => {
    isMounted = true;

    let keyboardDismissHandler: (() => void) | null = null;
    if (typeof window !== 'undefined') {
      keyboardDismissHandler = () => {
        eventHandlers.sendPrivateTyping(userId, false);
      };
      window.addEventListener(KEYBOARD_DISMISS_EVENT, keyboardDismissHandler);
    }

    const unsubscribeMessage = eventHandlers.onPrivateMessage(
      (payload: PrivateMessageNewPayload) => {
        if (
          payload.target_user_id === userId ||
          (payload.sender.user_id === userId && payload.target_user_id === authStore.currentUserId)
        ) {
          // spec: 按 message_id 去重
          if (messages.some((m) => m.message_id === payload.message_id)) return;
          if (payload.sender.user_id === userId) {
            resetOtherUserTyping();
          }

          const newMessage: PrivateMessageItem = {
            message_id: payload.message_id,
            type: payload.type,
            content: payload.content,
            sender: payload.sender,
            sent_at: payload.sent_at,
            media: payload.media,
            reply_to: payload.reply_to,
            is_read: true
          };
          messages = [...messages, newMessage];
          messageStore.markConversationAsRead(`private_${userId}`);
        }
      }
    );

    const unsubscribeTyping = websocketStore.on('PrivateUserTyping', (payload) => {
      if (payload.user_id === userId) {
        if (payload.is_typing) {
          markOtherUserTypingActive();
        } else {
          resetOtherUserTyping();
        }
      }
    });

    const unsubscribeRecalled = websocketStore.on('PrivateMessageRecalled', (payload) => {
      if (
        payload.target_user_id === userId ||
        (payload.recalled_by.user_id === userId &&
          payload.target_user_id === authStore.currentUserId)
      ) {
        messages = messages.map((msg) =>
          msg.message_id === payload.message_id ? { ...msg, recalled_at: payload.recalled_at } : msg
        );
      }
    });

    const unsubscribeEdited = websocketStore.on('PrivateMessageEdited', (payload) => {
      if (payload.target_user_id === userId || payload.target_user_id === authStore.currentUserId) {
        messages = messages.map((msg) =>
          msg.message_id === payload.message_id ? { ...msg, content: payload.content } : msg
        );
      }
    });

    return () => {
      isMounted = false;
      resetOtherUserTyping();
      unsubscribeMessage();
      unsubscribeTyping();
      unsubscribeRecalled();
      unsubscribeEdited();
      if (keyboardDismissHandler && typeof window !== 'undefined') {
        window.removeEventListener(KEYBOARD_DISMISS_EVENT, keyboardDismissHandler);
      }
      wsSubscription.cleanup('user', [userId]);
      eventHandlers.cancelAllTyping();
    };
  });

  // userId 变化时重置状态并重新加载（含首次加载）
  $effect(() => {
    const id = userId;
    untrack(() => {
      eventHandlers.cancelAllTyping();
      otherUser = null;
      messages = [];
      isLoading = true;
      resetOtherUserTyping();
      messageText = '';
      wsSubscription.reset();
      reconnectTracker.reset();
      messageStore.markConversationAsRead(`private_${id}`);
      loadChat();
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
        if (dev) console.log('[Chat] WebSocket 断开，重置订阅状态');
      }
      resetOtherUserTyping();
      wsSubscription.sync({
        connected,
        type: 'user',
        ids: [userId]
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
      type: 'user',
      ids: [userId],
      onSubscribeFailed: () => {
        if (cancelled) return;
        if (dev) console.warn('[Chat] ❌ 订阅失败，将在重连后重试');
        toast.warning(t('chat.subscribeFailedRetrying'));
      }
    });

    return () => {
      cancelled = true;
    };
  });
</script>

<svelte:head>
  <title>{t('chat.privateChatPageTitle', { name: otherUser?.nickname ?? t('chat.user') })}</title>
</svelte:head>

<div
  class="flex min-w-0 flex-col overflow-hidden"
  style="{platformStore.useMasterDetail
    ? 'height: 100%; width: 100%; flex: 1; border: 1px solid var(--border-theme);'
    : 'height: calc(100dvh - env(safe-area-inset-bottom, 0px)); width: 100%;'} background: var(--surface-bg);"
>
  <!-- Header -->
  <div
    class="flex h-16 shrink-0 items-center gap-3 px-4"
    style="background-color: var(--surface-bg); border-bottom: 1px solid var(--border-theme);"
  >
    {#if !platformStore.useMasterDetail}
      <BackButton onclick={handleBackClick} size={36} />
    {/if}

    {#if isLoading}
      <div
        class="rounded-full"
        style="width: 40px; height: 40px; background: var(--surface-bg);"
      ></div>
      <div
        style="width: 96px; height: 20px; border-radius: var(--radius-xs); background: var(--surface-bg);"
      ></div>
    {:else if otherUser}
      <div class="flex items-center gap-3">
        <div class="relative shrink-0">
          <div
            class="surface-flat overflow-hidden rounded-full"
            style="width: 38px; height: 38px; border: 1.5px solid var(--border-theme);"
          >
            <Avatar
              src={otherUser.avatar_url}
              name={otherUser.nickname}
              alt={otherUser.nickname}
              size={40}
              border={false}
            />
          </div>
        </div>
        <div class="min-w-0">
          <p class="text-body-lg truncate" style="color: var(--text-primary)">
            {otherUser.nickname}
          </p>
        </div>
      </div>
    {:else}
      <span class="text-body" style="color: var(--text-secondary)">{t('chat.userNotFound')}</span>
    {/if}
    <div class="ml-auto flex gap-1.5">
      <button
        class="surface-flat flex items-center justify-center rounded-full outline-none active:scale-[0.88]"
        style="width: 36px; height: 36px;"
        aria-label={t('chat.voiceCallAria')}
      >
        <Phone size={14} color="var(--text-secondary)" strokeWidth={1.5} />
      </button>
      <button
        class="surface-flat flex items-center justify-center rounded-full outline-none active:scale-[0.88]"
        style="width: 36px; height: 36px;"
        aria-label={t('chat.videoCallAria')}
      >
        <Video size={14} color="var(--text-secondary)" strokeWidth={1.5} />
      </button>
      <DropdownMenu>
        <DropdownMenuTrigger>
          <button
            class="surface-flat flex items-center justify-center rounded-full outline-none transition-transform duration-150 active:scale-[0.88]"
            style="width: 36px; height: 36px;"
            aria-label={t('chat.moreOptionsAria')}
          >
            <MoreHorizontal size={15} strokeWidth={1.5} color="var(--text-secondary)" />
          </button>
        </DropdownMenuTrigger>
        <DropdownMenuContent align="end" class="w-44">
          <DropdownMenuItem onclick={deleteChat} disabled={isDeletingChat} class="text-destructive">
            {#if isDeletingChat}
              <Spinner size="md" />
              <span>{t('chat.deleting')}</span>
            {:else}
              <Trash2 class="h-4 w-4" />
              <span>{t('chat.deleteChat')}</span>
            {/if}
          </DropdownMenuItem>
        </DropdownMenuContent>
      </DropdownMenu>
    </div>
  </div>

  <!-- Messages -->
  <div class="flex-1 overflow-y-auto py-4" style="background: var(--surface-bg);">
    {#if isLoading}
      <div class="space-y-4 px-4">
        {#each Array(5) as _, i (i)}
          <div class="flex {i % 2 === 0 ? '' : 'justify-end'}">
            <div class="flex max-w-[76%] gap-2">
              {#if i % 2 === 0}
                <div
                  class="shrink-0 rounded-full"
                  style="width: 32px; height: 32px; background: var(--surface-bg);"
                ></div>
              {/if}
              <div
                class="rounded-[var(--radius-md)]"
                style="width: 192px; height: 64px; background: var(--surface-bg);"
              ></div>
            </div>
          </div>
        {/each}
      </div>
    {:else if messages.length === 0}
      <div class="flex h-full items-center justify-center">
        <EmptyState
          icon={MessageCircle}
          title={t('chat.startChat')}
          variant="inline"
        />
      </div>
    {:else}
      <div class="space-y-3 px-4">
        <!-- Date pill -->
        <div class="mb-2 flex justify-center">
          <span
            class="text-xs rounded-full px-3 py-1"
            style="background: var(--surface-bg); color: var(--text-secondary); border: 1px solid var(--border-theme); letter-spacing: 0.05em; opacity: 0.8"
          >
            {t('chat.today')}
          </span>
        </div>
        {#each messages as message (message.message_id)}
          <div
            class="flex {isSelf(message) ? 'justify-end' : 'justify-start'} chat-msg-enter"
            {@attach observeMessage({ messageId: message.message_id, isSelf: isSelf(message) })}
          >
            <div style="max-width: 76%;">
              <ChatBubble
                isSelf={isSelf(message)}
                type={message.type}
                content={message.content}
                media={message.media}
                isRecalled={message.recalled_at != null}
                time={formatTime(message.sent_at)}
                readReceipt={isSelf(message) &&
                  message.message_id === messages.filter((m) => isSelf(m)).at(-1)?.message_id}
              />
            </div>
          </div>
        {/each}
      </div>
    {/if}
  </div>

  <!-- Input -->
  <div
    class="px-4 pt-4 pb-4"
    style="background-color: var(--surface-bg); border-top: 1px solid var(--border-theme);"
  >
    {#if !websocketStore.isConnected && !isLoading}
      <div
        class="text-meta flex items-center justify-center gap-1.5 mb-2"
        style="color: var(--text-secondary)"
      >
        <Spinner size="xs" />
        <span>{t('chat.realtimeConnecting')}</span>
      </div>
    {/if}
    {#if websocketStore.isConnected && !wsSubscription.state.isSubscribed && !isLoading}
      <div
        class="text-meta flex items-center justify-center gap-1.5 mb-2"
        style="color: var(--text-secondary)"
      >
        <Spinner size="xs" />
        <span>
          {#if wsSubscription.state.isAttemptingSubscribe}
            {t('chat.typingSubscribing')}
          {:else}
            {t('chat.typingSubscribingRetrying')}
          {/if}
        </span>
      </div>
    {/if}

    {#if isOtherUserTyping}
      <div class="text-meta mb-2" style="color: var(--text-secondary)">
        {t('chat.typing', { name: otherUser?.nickname ?? t('chat.otherParty') })}
      </div>
    {/if}

    <div class="flex items-center gap-2.5">
      <button
        class="surface-flat flex shrink-0 items-center justify-center rounded-full outline-none active:scale-[0.88]"
        style="width: 36px; height: 36px;"
        aria-label={t('chat.emojiAria')}
      >
        <Smile size={15} color="var(--text-secondary)" strokeWidth={1.5} />
      </button>
      <Textarea
        size="sm"
        autofocusTrigger={shouldFocusInput}
        placeholder={t('chat.inputPlaceholder')}
        bind:value={messageText}
        maxlength={MESSAGE_CONTENT_MAX}
        disabled={isSending}
        oninput={handleInput}
        onkeydown={(e) => {
          if (e.key === 'Enter' && !e.shiftKey) {
            e.preventDefault();
            sendMessage();
          }
        }}
      />
      <button
        class="flex shrink-0 items-center justify-center rounded-full outline-none transition-transform duration-150 active:scale-[0.9]"
        class:surface-pressed={!messageText.trim()}
        style="width: 36px; height: 36px; {messageText.trim()
          ? 'background: var(--brand); color: var(--text-on-accent); box-shadow: var(--shadow-brand-btn);'
          : 'color: var(--text-secondary);'}"
        onclick={sendMessage}
        onmousedown={(e) => e.preventDefault()}
        disabled={!messageText.trim() || isSending}
        aria-label={t('chat.sendAria')}
      >
        {#if isSending}
          <Spinner size="lg" />
        {:else}
          <Send size={20} />
        {/if}
      </button>
    </div>
  </div>
</div>
