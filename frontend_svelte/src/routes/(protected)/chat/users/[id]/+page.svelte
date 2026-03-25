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
	import { websocketStore, eventHandlers } from '$lib/websocket';
	import type { PrivateMessageNewPayload } from '$lib/websocket/types';
	import { useBatchMarkRead } from '$lib/hooks/use-batch-mark-read.svelte';
	import { createReconnectTracker, mergeMessagesByIdAndSentAt, resolveLatestVisibleMessageId } from '$lib/hooks/use-chat-realtime-sync';
	import { useChatWsSubscription } from '$lib/hooks/use-chat-ws-subscription.svelte';
	import { autofocus, useKeyboardFocus, KEYBOARD_DISMISS_EVENT } from '$lib/attachments';
	import { Button } from '$lib/components/ui/button';
	import { Avatar, AvatarFallback, AvatarImage } from '$lib/components/ui/avatar';
	import {
		DropdownMenu,
		DropdownMenuContent,
		DropdownMenuItem,
		DropdownMenuTrigger
	} from '$lib/components/ui/dropdown-menu';
	import { Skeleton } from '$lib/components/ui/skeleton';
	import { ScrollArea } from '$lib/components/ui/scroll-area';
	import type { PrivateMessageItem, UserInfo } from '$lib/types/api';
	import ArrowLeft from 'lucide-svelte/icons/arrow-left';
	import Trash2 from 'lucide-svelte/icons/trash-2';
	import Send from 'lucide-svelte/icons/send';
	import Loader2 from 'lucide-svelte/icons/loader-2';
	import MoreVertical from 'lucide-svelte/icons/more-vertical';

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
			if (isMounted) toast.error('加载失败');
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
			toast.error(`消息内容最多 ${MESSAGE_CONTENT_MAX} 字符`);
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
				toast.error(response.message || '发送失败');
			} else {
				// 发送成功后重新聚焦输入框
				shouldFocusInput++; // 触发重新聚焦
			}
			// ✅ 不在本地添加消息，等待 WebSocket 推送（规范要求）
		} catch {
			messageText = text;
			toast.error('发送失败');
		} finally {
			isSending = false;
		}
	}

	async function deleteChat() {
		if (isDeletingChat) return;
		if (!confirm('删除后仅清空你的会话视图，不会影响对方。确认删除吗？')) return;

		isDeletingChat = true;
		try {
			const response = await privateChatsService.deleteChat({ target_user_id: userId });
			if (!isSuccess(response)) {
				toast.error(response.message || '删除会话失败');
				return;
			}
			messageStore.markConversationAsRead(`private_${userId}`);
			eventHandlers.cancelAllTyping();
			toast.success('会话已删除');
			await goto('/chat');
		} catch {
			toast.error('删除会话失败');
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

		const unsubscribeMessage = eventHandlers.onPrivateMessage((payload: PrivateMessageNewPayload) => {
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
		});

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
				(payload.recalled_by.user_id === userId && payload.target_user_id === authStore.currentUserId)
			) {
				messages = messages.map((msg) =>
					msg.message_id === payload.message_id ? { ...msg, recalled_at: payload.recalled_at } : msg
				);
			}
		});

		const unsubscribeEdited = websocketStore.on('PrivateMessageEdited', (payload) => {
			if (
				payload.target_user_id === userId ||
				payload.target_user_id === authStore.currentUserId
			) {
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
				toast.warning('消息订阅失败，正在重试...');
			}
		});

		return () => { cancelled = true; };
	});
</script>

<svelte:head>
	<title>与 {otherUser?.nickname ?? '用户'} 的对话 - HotSpot</title>
</svelte:head>

<div class="flex h-[calc(100dvh-8rem)] flex-col overflow-hidden rounded-xl border border-border/50 bg-card shadow-lg">
	<!-- Header -->
	<div class="flex h-16 items-center gap-3 border-b border-border/50 bg-card/80 px-4 backdrop-blur-sm">
		<Button variant="ghost" size="icon" onclick={handleBackClick} aria-label="返回">
			<ArrowLeft class="h-5 w-5" />
		</Button>
		{#if isLoading}
			<Skeleton class="h-10 w-10 rounded-full" />
			<Skeleton class="h-5 w-24" />
		{:else if otherUser}
			<a href="/users/{userId}" class="flex items-center gap-3">
				<Avatar class="h-10 w-10">
					<AvatarImage src={otherUser.avatar_url} alt={otherUser.nickname} />
					<AvatarFallback>{(otherUser.nickname || '用户').slice(0, 2)}</AvatarFallback>
				</Avatar>
				<span class="font-medium">{otherUser.nickname}</span>
			</a>
		{:else}
			<span class="text-sm text-muted-foreground">用户不存在或加载失败</span>
		{/if}
		<div class="ml-auto">
			<DropdownMenu>
				<DropdownMenuTrigger>
					<Button variant="ghost" size="icon" aria-label="更多选项">
						<MoreVertical class="h-5 w-5" />
					</Button>
				</DropdownMenuTrigger>
				<DropdownMenuContent align="end" class="w-44">
					<DropdownMenuItem onclick={deleteChat} disabled={isDeletingChat} class="text-destructive">
						{#if isDeletingChat}
							<Loader2 class="h-4 w-4 animate-spin" />
							<span>删除中...</span>
						{:else}
							<Trash2 class="h-4 w-4" />
							<span>删除会话</span>
						{/if}
					</DropdownMenuItem>
				</DropdownMenuContent>
			</DropdownMenu>
		</div>
	</div>

	<!-- Messages -->
	<ScrollArea class="flex-1 bg-muted/20 py-4">
		{#if isLoading}
			<div class="space-y-4 px-4">
				{#each Array(5) as _, i (i)}
					<div class="flex {i % 2 === 0 ? '' : 'justify-end'}">
						<div class="flex max-w-[70%] gap-2">
							{#if i % 2 === 0}
								<Skeleton class="h-8 w-8 rounded-full" />
							{/if}
							<Skeleton class="h-16 w-48 rounded-lg" />
						</div>
					</div>
				{/each}
			</div>
		{:else if messages.length === 0}
			<div class="flex h-full items-center justify-center">
				<p class="text-muted-foreground">开始聊天吧</p>
			</div>
		{:else}
			<div class="space-y-3 px-4">
				{#each messages as message (message.message_id)}
					<div
						class="flex {isSelf(message) ? 'justify-end' : ''}"
						{@attach observeMessage({ messageId: message.message_id, isSelf: isSelf(message) })}
					>
						<div class="flex max-w-[70%] flex-col gap-1">
							<div
								class="rounded-2xl px-4 py-2.5 {isSelf(message)
									? 'bg-primary text-primary-foreground shadow-md shadow-primary/20 border border-primary/30'
									: 'bg-card text-card-foreground shadow-md border border-border/50'}"
							>
								{#if message.recalled_at != null}
									<p class="text-sm italic opacity-70">消息已撤回</p>
								{:else if message.type === 'text'}
									<p class="whitespace-pre-wrap text-sm leading-relaxed">{message.content}</p>
								{:else if message.type === 'image' && message.media}
									<img
										src={message.media.url}
										alt="图片"
										class="max-w-[250px] rounded-lg"
									/>
								{:else}
									<p class="text-sm">[{message.type}]</p>
								{/if}
							</div>
							<p class="px-2 text-xs text-muted-foreground {isSelf(message) ? 'text-right' : ''}">
								{formatTime(message.sent_at)}
							</p>
						</div>
					</div>
				{/each}
			</div>
		{/if}
	</ScrollArea>

	<!-- Input -->
	<div class="border-t border-border/50 bg-card/80 pt-4 px-4 pb-4 backdrop-blur-sm">
		{#if !websocketStore.isConnected && !isLoading}
			<div class="flex items-center justify-center gap-1.5 mb-2 text-xs text-muted-foreground">
				<Loader2 class="h-3 w-3 animate-spin" />
				<span>实时消息连接中…</span>
			</div>
		{/if}
		{#if websocketStore.isConnected && !wsSubscription.state.isSubscribed && !isLoading}
			<div class="flex items-center justify-center gap-1.5 mb-2 text-xs text-muted-foreground">
				<Loader2 class="h-3 w-3 animate-spin" />
				<span>
					{#if wsSubscription.state.isAttemptingSubscribe}
						输入状态订阅中（不影响消息收发）
					{:else}
						输入状态订阅重试中（不影响消息收发）
					{/if}
				</span>
			</div>
		{/if}

		{#if isOtherUserTyping}
			<div class="mb-2 text-xs text-muted-foreground">
				{otherUser?.nickname ?? '对方'} 正在输入...
			</div>
		{/if}

		<div class="flex gap-2">
			<textarea
				{@attach autofocus(shouldFocusInput)}
				{@attach useKeyboardFocus()}
				placeholder="输入消息..."
				bind:value={messageText}
				rows={1}
				maxlength={MESSAGE_CONTENT_MAX}
				class="min-h-[40px] flex w-full resize-none rounded-md border border-input bg-background/80 px-3 py-2 text-base shadow-sm shadow-black/5 dark:shadow-black/20 placeholder:text-muted-foreground focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-ring disabled:cursor-not-allowed disabled:opacity-50 md:text-sm"
				disabled={isSending}
				oninput={handleInput}
				onkeydown={(e) => {
					if (e.key === 'Enter' && !e.shiftKey) {
						e.preventDefault();
						sendMessage();
					}
				}}
			></textarea>
			<Button
				size="icon"
				onclick={sendMessage}
				onmousedown={(e) => e.preventDefault()}
				disabled={!messageText.trim() || isSending}
				aria-label="发送消息"
			>
				{#if isSending}
					<Loader2 class="h-5 w-5 animate-spin" />
				{:else}
					<Send class="h-5 w-5" />
				{/if}
			</Button>
		</div>
	</div>
</div>
