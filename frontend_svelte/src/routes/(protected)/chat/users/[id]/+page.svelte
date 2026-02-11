<script lang="ts">
	import { dev } from '$app/environment';
	import { page } from '$app/state';
	import { onMount } from 'svelte';
	import { withMinimumLoadingTime } from '$lib/utils/loading';
	import { privateMessagesService, usersService } from '$lib/api/services';
	import { isSuccess } from '$lib/api/client';
	import { authStore } from '$lib/stores/auth.svelte';
	import { messageStore } from '$lib/stores/message.svelte';
	import { toast } from '$lib/stores/toast.svelte';
	import { websocketStore, subscriptionManager, eventHandlers } from '$lib/websocket';
	import type { PrivateMessageNewPayload } from '$lib/websocket/types';
	import { useBatchMarkRead } from '$lib/hooks/use-batch-mark-read.svelte';
	import { autofocus } from '$lib/attachments';
	import { Button } from '$lib/components/ui/button';
	import { Avatar, AvatarFallback, AvatarImage } from '$lib/components/ui/avatar';
	import { Skeleton } from '$lib/components/ui/skeleton';
	import { ScrollArea } from '$lib/components/ui/scroll-area';
	import type { PrivateMessageItem, UserInfo } from '$lib/types/api';
	import ArrowLeft from 'lucide-svelte/icons/arrow-left';
	import Send from 'lucide-svelte/icons/send';
	import Loader2 from 'lucide-svelte/icons/loader-2';
	import MoreVertical from 'lucide-svelte/icons/more-vertical';

	const userId = page.params.id!;

	let otherUser = $state<UserInfo | null>(null);
	let messages = $state.raw<PrivateMessageItem[]>([]);
	let isLoading = $state(true);
	let messageText = $state('');
	let isSending = $state(false);
	let isOtherUserTyping = $state(false);
	let typingTimer: ReturnType<typeof setTimeout> | null = null;
	let shouldFocusInput = $state(0); // 用于触发重新聚焦
	let isSubscribed = $state(false); // 订阅状态
	let isAttemptingSubscribe = $state(false); // 是否正在尝试订阅

	const canSendMessage = $derived(isSubscribed);

	// ✨ 批量已读逻辑
	const batchMarkRead = useBatchMarkRead({
		markAsRead: async (_messageIds) => {
			// 私聊消息的标记已读是按对话标记的，不需要传 messageIds
			await privateMessagesService.markAsRead({ target_user_id: userId });
		},
		debounceMs: 2000,
		visibilityThreshold: 1000
	});

	async function loadChat() {
		try {
			await withMinimumLoadingTime(async () => {
				// 获取对方信息
				const userResponse = await usersService.getUser({ user_id: userId });
				if (isSuccess(userResponse) && userResponse.data) {
					otherUser = userResponse.data.user;
				}

				// 获取消息历史
				const messagesResponse = await privateMessagesService.getHistory({
					target_user_id: userId,
					pagination: { limit: 50 }
				});
				if (isSuccess(messagesResponse) && messagesResponse.data) {
					messages = messagesResponse.data.items.reverse();
				}

				// ✅ 不再立即标记已读，等待消息可见时自动标记
			});
		} catch {
			toast.error('加载失败');
		} finally {
			isLoading = false;
		}
	}

	async function sendMessage() {
		if (!messageText.trim() || !canSendMessage) return;

		isSending = true;
		const text = messageText.trim();
		messageText = '';

		try {
			const response = await privateMessagesService.send({
				target_user_id: userId,
				type: 'text',
				content: text,
				idempotencyKey: crypto.randomUUID()
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

	/**
	 * 处理输入变化（发送正在输入状态）
	 */
	function handleInput() {
		// 发送正在输入状态（使用派生状态）
		if (websocketStore.isConnected) {
			websocketStore.send({
				type: 'PrivateTyping',
				payload: {
					target_user_id: userId,
					is_typing: true
				}
			});
		}

		// 3秒后自动发送停止输入
		if (typingTimer) clearTimeout(typingTimer);
		typingTimer = setTimeout(() => {
			if (websocketStore.isConnected) {
				websocketStore.send({
					type: 'PrivateTyping',
					payload: {
						target_user_id: userId,
						is_typing: false
					}
				});
			}
		}, 3000);
	}

	function formatTime(dateStr: string): string {
		const date = new Date(dateStr);
		return date.toLocaleTimeString('zh-CN', { hour: '2-digit', minute: '2-digit' });
	}

	function isSelf(message: PrivateMessageItem): boolean {
		return message.sender_id === authStore.currentUserId;
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
		// ✅ 立即清除这个对话的未读计数
		messageStore.markConversationAsRead(`private_${userId}`);

		// 先注册消息监听器（确保不会错过任何消息）
		// 监听私聊消息
		const unsubscribeMessage = eventHandlers.onPrivateMessage((payload: PrivateMessageNewPayload) => {
			// 只处理与当前聊天相关的消息
			if (
				payload.target_user_id === userId ||
				(payload.sender_id === userId && payload.target_user_id === authStore.currentUserId)
			) {
				// 将 WebSocket 消息转换为 PrivateMessageItem
				const newMessage: PrivateMessageItem = {
					message_id: payload.message_id,
					type: payload.type,
					content: payload.content,
					sender_id: payload.sender_id,
					sent_at: payload.sent_at,
					media: payload.media,
					reply_to: payload.reply_to
				};

				// 添加到消息列表
				messages = [...messages, newMessage];

				// ✅ 清除未读计数（因为用户正在查看这个对话）
				messageStore.markConversationAsRead(`private_${userId}`);
			}
		});

		// 监听对方正在输入（类型自动推断！）
		const unsubscribeTyping = websocketStore.on('PrivateUserTyping', (payload) => {
			if (payload.user_id === userId) {
				isOtherUserTyping = payload.is_typing;
			}
		});

		// 监听消息撤回（类型自动推断！）
		const unsubscribeRecalled = websocketStore.on('PrivateMessageRecalled', (payload) => {
			if (
				payload.target_user_id === userId ||
				(payload.recalled_by === userId && payload.target_user_id === authStore.currentUserId)
			) {
				messages = messages.map((msg) =>
					msg.message_id === payload.message_id ? { ...msg, recalled_at: payload.recalled_at } : msg
				);
			}
		});

		// 监听消息编辑（类型自动推断！）
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

		// 加载聊天数据（异步，不阻塞）
		loadChat();

		// ✅ 返回清理函数（Svelte 5 的新方式）
		return () => {
			unsubscribeMessage();
			unsubscribeTyping();
			unsubscribeRecalled();
			unsubscribeEdited();

			// 取消订阅
			if (isSubscribed) {
				subscriptionManager.unsubscribe('user', [userId]);
			}

			// 清除正在输入定时器
			if (typingTimer) clearTimeout(typingTimer);
		};
	});

	/**
	 * 响应式订阅管理（等待 WebSocket 连接完成）
	 * 使用 $effect 监听连接状态，自动订阅/恢复订阅
	 */
	$effect(() => {
		// 使用 derived 状态检查连接
		const connected = websocketStore.isConnected;
		
		if (!connected) {
			// 连接断开，重置订阅状态
			if (isSubscribed) {
				if (dev) console.log('[Chat] WebSocket 断开，重置订阅状态');
				isSubscribed = false;
			}
			return;
		}

		// WebSocket 已连接，尝试订阅
		if (!isSubscribed && !isAttemptingSubscribe) {
			isAttemptingSubscribe = true;
			
			if (dev) console.log('[Chat] WebSocket 已连接，订阅用户:', userId);
			
			subscriptionManager.subscribe('user', [userId]).then((success) => {
				isSubscribed = success;
				isAttemptingSubscribe = false;
				
				if (success) {
					if (dev) console.log('[Chat] ✅ 订阅成功');
				} else {
					if (dev) console.warn('[Chat] ❌ 订阅失败，将在重连后重试');
					toast.warning('消息订阅失败，正在重试...');
				}
			});
		}
	});
</script>

<svelte:head>
	<title>与 {otherUser?.nickname ?? '用户'} 的对话 - HotSpot</title>
</svelte:head>

<div class="flex h-[calc(100dvh-8rem)] flex-col overflow-hidden rounded-xl border border-border/50 bg-card shadow-lg">
	<!-- Header -->
	<div class="flex h-16 items-center gap-3 border-b border-border/50 bg-card/80 px-4 backdrop-blur-sm">
		<Button variant="ghost" size="icon" onclick={() => history.back()} aria-label="返回">
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
		{/if}
		<div class="ml-auto">
			<Button variant="ghost" size="icon" aria-label="更多选项">
				<MoreVertical class="h-5 w-5" />
			</Button>
		</div>
	</div>

	<!-- Messages -->
	<ScrollArea class="flex-1 bg-muted/20 py-4">
		{#if isLoading}
			<div class="space-y-4 px-4">
				{#each Array(5) as _, i}
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
		{#if !isSubscribed && !isLoading}
			<div class="rounded-xl bg-yellow-500/10 border border-yellow-500/30 px-4 py-2.5 text-center mb-3">
				<p class="text-sm text-yellow-700 dark:text-yellow-300 flex items-center justify-center gap-2">
					{#if isAttemptingSubscribe}
						<Loader2 class="h-4 w-4 animate-spin" />
						正在连接服务器...
					{:else if !websocketStore.isConnected}
						<Loader2 class="h-4 w-4 animate-spin" />
						等待 WebSocket 连接...
					{:else}
						订阅失败，正在重试...
					{/if}
				</p>
			</div>
		{/if}
		
		<!-- 对方正在输入提示 -->
		{#if isOtherUserTyping}
			<div class="mb-2 text-xs text-muted-foreground">
				{otherUser?.nickname ?? '对方'} 正在输入...
			</div>
		{/if}

		<div class="flex gap-2">
			<textarea
				{@attach autofocus(shouldFocusInput)}
				placeholder="输入消息..."
				bind:value={messageText}
				rows={1}
				class="min-h-[40px] flex w-full resize-none rounded-md border border-input bg-background/80 px-3 py-2 text-base shadow-sm shadow-black/5 placeholder:text-muted-foreground focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-ring disabled:cursor-not-allowed disabled:opacity-50 md:text-sm"
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
