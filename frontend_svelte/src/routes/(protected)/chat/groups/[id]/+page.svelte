<script lang="ts">
	import { dev } from '$app/environment';
	import { page } from '$app/state';
	import { onMount, tick } from 'svelte';
	import { withMinimumLoadingTime } from '$lib/utils/loading';
	import { groupsService, groupMessagesService } from '$lib/api/services';
	import { isSuccess } from '$lib/api/client';
	import { authStore } from '$lib/stores/auth.svelte';
	import { messageStore } from '$lib/stores/message.svelte';
	import { toast } from '$lib/stores/toast.svelte';
	import { websocketStore, subscriptionManager, eventHandlers } from '$lib/websocket';
	import type { GroupMessageNewPayload } from '$lib/websocket/types';
	import { useBatchMarkRead } from '$lib/hooks/use-batch-mark-read.svelte';
	import { autofocus } from '$lib/attachments';
	import { Button } from '$lib/components/ui/button';
	import { Avatar, AvatarFallback, AvatarImage } from '$lib/components/ui/avatar';
	import { Skeleton } from '$lib/components/ui/skeleton';
	import { Badge } from '$lib/components/ui/badge';
	import type { GroupItem, GroupMessageItem } from '$lib/types/api';
	import ArrowLeft from 'lucide-svelte/icons/arrow-left';
	import Send from 'lucide-svelte/icons/send';
	import Loader2 from 'lucide-svelte/icons/loader-2';
	import MoreVertical from 'lucide-svelte/icons/more-vertical';
	import Users from 'lucide-svelte/icons/users';
	import MessageCircle from 'lucide-svelte/icons/message-circle';
	import MapPin from 'lucide-svelte/icons/map-pin';
	import ChevronDown from 'lucide-svelte/icons/chevron-down';
	import X from 'lucide-svelte/icons/x';

	const groupId = page.params.id!;

	let group = $state<GroupItem | null>(null);
	let messages = $state.raw<GroupMessageItem[]>([]);
	let isLoading = $state(true);
	let messageText = $state('');
	let isSending = $state(false);
	let cursor = $state<string | undefined>(undefined);
	let hasMore = $state(true);
	let isLoadingMore = $state(false);
	let scrollContainer: HTMLDivElement | null = $state(null);
	let showScrollButton = $state(false);
	let selectedImage = $state<string | null>(null);
	let shouldFocusInput = $state(0); // 用于触发重新聚焦
	let isSubscribed = $state(false); // 订阅状态
	let isAttemptingSubscribe = $state(false); // 是否正在尝试订阅

	const canSendMessage = $derived(group?.my_state?.is_joined && !group?.my_state?.is_muted && isSubscribed);

	// ✨ 批量已读逻辑
	const batchMarkRead = useBatchMarkRead({
		markAsRead: async (_messageIds) => {
			// 群聊消息的标记已读是按群组标记的，不需要传 messageIds
			await groupMessagesService.markAsRead({ group_id: groupId });
		},
		debounceMs: 2000,
		visibilityThreshold: 1000
	});

	async function loadGroup() {
		try {
			const response = await groupsService.getGroup({ group_id: groupId });
			if (isSuccess(response) && response.data) {
				group = response.data;
			} else {
				toast.error(response.message || '群组不存在');
			}
		} catch {
			toast.error('加载群组失败');
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
				},
				include_system: true
			});

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
			toast.error('加载消息失败');
		} finally {
			if (loadMore) {
				isLoadingMore = false;
			} else {
				isLoading = false;
			}
		}
	}

	async function sendMessage() {
		if (!messageText.trim() || !canSendMessage) return;

		isSending = true;
		const text = messageText.trim();
		messageText = '';

		try {
			const mentions = extractMentions(text);

			const response = await groupMessagesService.send({
				group_id: groupId,
				type: 'text',
				content: text,
				mentions: mentions.length > 0 ? mentions : undefined,
				idempotencyKey: crypto.randomUUID()
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

	async function init() {
		await withMinimumLoadingTime(async () => {
			await loadGroup();
			await loadMessages();
		});
	}

	onMount(() => {
		// ✅ 立即清除这个群组的未读计数
		messageStore.markConversationAsRead(`group_${groupId}`);

		// 先注册消息监听器（确保不会错过任何消息）
		const unsubscribeMessage = eventHandlers.onGroupMessage((payload: GroupMessageNewPayload) => {
			if (payload.group_id === groupId) {
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

				// ✅ 清除未读计数（因为用户正在查看这个群组）
				messageStore.markConversationAsRead(`group_${groupId}`);

				// 如果在底部附近，自动滚动到新消息
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
				// 更新群组成员数量
				if (group) {
					group = { ...group, group: { ...group.group, members_count: payload.member_count } };
				}
			}
		});

		// 监听群组成员离开事件
		const handleMemberLeft = (e: CustomEvent<{
			group_id: string;
			user_id: string;
			member_count: number;
		}>) => {
			if (e.detail.group_id === groupId) {
				// 更新群组成员数量
				if (group) {
					group = { ...group, group: { ...group.group, members_count: e.detail.member_count } };
				}
			}
		};

		// 监听群组成员被移除事件
		const handleMemberRemoved = (e: CustomEvent<{
			group_id: string;
			user_id: string;
			member_count: number;
			is_current_user: boolean;
		}>) => {
			if (e.detail.group_id === groupId) {
				// 更新群组成员数量
				if (group) {
					group = { ...group, group: { ...group.group, members_count: e.detail.member_count } };
				}
				// 如果被移除的是当前用户，重新加载群组信息
				if (e.detail.is_current_user) {
					loadGroup();
				}
			}
		};

		// 监听群组信息更新事件
		const handleGroupUpdated = (e: CustomEvent<{
			group_id: string;
			updated_fields: {
				name?: string;
				avatar_url?: string;
				description?: string;
			};
		}>) => {
			if (e.detail.group_id === groupId && group) {
				// 更新群组信息
				group = {
					...group,
					group: {
						...group.group,
						...e.detail.updated_fields
					}
				};
			}
		};

		// 注册浏览器事件监听器
		window.addEventListener('group-member-left', handleMemberLeft as EventListener);
		window.addEventListener('group-member-removed', handleMemberRemoved as EventListener);
		window.addEventListener('group-updated', handleGroupUpdated as EventListener);

		// 加载数据（异步，不阻塞）
		init();

		// ✅ 返回清理函数（Svelte 5 的新方式）
		return () => {
			unsubscribeMessage();
			unsubscribeRecalled();
			unsubscribeEdited();
			unsubscribeMemberJoined();
			
			// 移除浏览器事件监听器
			window.removeEventListener('group-member-left', handleMemberLeft as EventListener);
			window.removeEventListener('group-member-removed', handleMemberRemoved as EventListener);
			window.removeEventListener('group-updated', handleGroupUpdated as EventListener);
			
			// 取消订阅
			if (isSubscribed) {
				subscriptionManager.unsubscribe('group', [groupId]);
			}
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
				if (dev) console.log('[GroupChat] WebSocket 断开，重置订阅状态');
				isSubscribed = false;
			}
			return;
		}

		// WebSocket 已连接，尝试订阅
		if (!isSubscribed && !isAttemptingSubscribe) {
			isAttemptingSubscribe = true;
			
			if (dev) console.log('[GroupChat] WebSocket 已连接，订阅群组:', groupId);
			
			subscriptionManager.subscribe('group', [groupId]).then((success) => {
				isSubscribed = success;
				isAttemptingSubscribe = false;
				
				if (success) {
					if (dev) console.log('[GroupChat] ✅ 订阅成功');
				} else {
					if (dev) console.warn('[GroupChat] ❌ 订阅失败，将在重连后重试');
					toast.warning('消息订阅失败，正在重试...');
				}
			});
		}
	});

</script>

<svelte:head>
	<title>{group?.group.name ?? '群聊'} - HotSpot</title>
</svelte:head>

<!-- 聊天容器 - 保留侧边栏布局 -->
<div class="flex h-[calc(100dvh-8rem)] flex-col overflow-hidden rounded-xl border border-border/50 bg-card shadow-lg">
	<!-- Header -->
	<header class="flex h-16 shrink-0 items-center gap-3 border-b border-border/50 bg-card/80 px-4 backdrop-blur-sm">
		<Button variant="ghost" size="icon" class="h-9 w-9" onclick={() => history.back()} aria-label="返回">
			<ArrowLeft class="h-5 w-5" />
		</Button>
		
		{#if isLoading}
			<Skeleton class="h-10 w-10 rounded-full" />
			<Skeleton class="h-5 w-32" />
		{:else if group}
			<a 
				href="/groups/{groupId}" 
				class="flex flex-1 items-center gap-3 overflow-hidden transition-opacity hover:opacity-80"
			>
				<Avatar class="h-10 w-10 shrink-0">
					<AvatarImage src={group.group.avatar_url} alt={group.group.name} />
					<AvatarFallback>
						<Users class="h-5 w-5" />
					</AvatarFallback>
				</Avatar>
				<div class="flex-1 overflow-hidden">
					<div class="truncate font-semibold">{group.group.name}</div>
					<div class="truncate text-xs text-muted-foreground">
						{group.group.members_count} 位成员
					</div>
				</div>
			</a>
		{/if}
		
		<Button variant="ghost" size="icon" class="h-9 w-9 shrink-0" aria-label="更多选项">
			<MoreVertical class="h-5 w-5" />
		</Button>
	</header>

	<!-- Messages Container -->
	<div 
		bind:this={scrollContainer}
		onscroll={handleScroll}
		class="relative flex-1 overflow-y-auto overscroll-contain bg-muted/20"
	>
		{#if isLoading}
			<div class="space-y-4 p-4">
				{#each Array(8) as _}
					<div class="flex gap-3">
						<Skeleton class="h-10 w-10 shrink-0 rounded-full" />
						<div class="flex-1 space-y-2">
							<Skeleton class="h-3 w-24" />
							<Skeleton class="h-16 w-full max-w-xs rounded-2xl" />
						</div>
					</div>
				{/each}
			</div>
		{:else}
			{#if hasMore && messages.length > 0 && !isLoadingMore}
				<div class="flex justify-center py-3">
					<Button
						variant="ghost"
						size="sm"
						class="text-xs"
						onclick={() => loadMessages(true)}
					>
						加载更多消息
					</Button>
				</div>
			{/if}
			
			{#if isLoadingMore}
				<div class="flex justify-center py-3">
					<Loader2 class="h-5 w-5 animate-spin text-muted-foreground" />
				</div>
			{/if}

			{#if messages.length === 0}
				<div class="flex h-full items-center justify-center p-8">
					<div class="text-center">
						<div class="mb-4 inline-flex h-16 w-16 items-center justify-center rounded-full bg-muted">
							<MessageCircle class="h-8 w-8 text-muted-foreground" />
						</div>
						<h3 class="mb-1 font-medium">还没有消息</h3>
						<p class="text-sm text-muted-foreground">发送第一条消息开始聊天</p>
					</div>
				</div>
			{:else}
				<div class="space-y-1 px-4 py-2">
					{#each messages as message, index (message.message_id)}
						{@const showDateDivider = index === 0 || 
							new Date(message.sent_at).toDateString() !== new Date(messages[index - 1]!.sent_at).toDateString()}
						{@const showAvatar = shouldShowAvatar(index)}
						{@const showName = shouldShowName(index)}
						{@const isMe = isSelf(message)}
						
						{#if showDateDivider}
							<div class="flex justify-center py-4">
								<span class="rounded-full bg-muted/50 px-3 py-1 text-xs font-medium text-muted-foreground">
									{formatDateDivider(message.sent_at)}
								</span>
							</div>
						{/if}

						{#if isSystemMessage(message)}
							<div class="flex justify-center py-2">
								<Badge variant="secondary" class="rounded-full bg-muted/50 px-3 py-1 text-xs font-normal">
									{message.content}
								</Badge>
							</div>
						{:else}
							<div
								class="group flex gap-3 py-1 {isMe ? 'flex-row-reverse' : ''}"
								{@attach observeMessage({ messageId: message.message_id, isSelf: isMe })}
							>
								{#if !isMe}
									<div class="w-10 shrink-0">
										{#if showAvatar}
											<a href="/users/{message.sender?.user_id}">
												<Avatar class="h-10 w-10 transition-opacity hover:opacity-80">
													<AvatarImage
														src={message.sender?.avatar_url}
														alt={message.sender?.nickname}
													/>
													<AvatarFallback class="text-xs">
														{(message.sender?.nickname || '用户').slice(0, 2)}
													</AvatarFallback>
												</Avatar>
											</a>
										{/if}
									</div>
								{/if}
								
								<div class="flex max-w-[70%] flex-col gap-0.5 {isMe ? 'items-end' : 'items-start'}">
									{#if !isMe && showName}
										<span class="px-3 text-xs font-medium text-muted-foreground">
											{message.sender?.nickname}
										</span>
									{/if}
									
									<div
										class="relative rounded-2xl shadow-sm {isMe
											? 'bg-primary text-primary-foreground border border-primary/30'
											: 'bg-card text-card-foreground border border-border/50'} 
											{message.type === 'image' ? 'p-1' : 'px-4 py-2.5'}"
									>
										{#if message.recalled_at != null}
											<p class="text-sm italic opacity-70">消息已撤回</p>
										{:else if message.type === 'text'}
											<p class="whitespace-pre-wrap break-words text-sm leading-relaxed">
												{message.content}
											</p>
										{:else if message.type === 'image' && message.media}
											<button
												type="button"
												onclick={() => openImagePreview(message.media!.url)}
												class="block overflow-hidden rounded-xl transition-opacity hover:opacity-90"
											>
												<img
													src={message.media.url}
													alt="图片"
													class="max-h-[300px] max-w-[250px] object-cover"
												/>
											</button>
										{:else if message.type === 'share' && message.share_code}
											<div class="flex items-center gap-2 text-sm">
												<MapPin class="h-4 w-4 shrink-0" />
												<div>
													<p class="font-medium">分享消息</p>
													{#if message.share_code}
														<a href="/share/{message.share_code}" class="text-xs opacity-80 hover:underline">
															查看分享内容
														</a>
													{/if}
												</div>
											</div>
										{:else}
											<p class="text-sm opacity-70">[{message.type}]</p>
										{/if}
									</div>
									
									{#if showAvatar}
										<span class="px-3 text-[10px] text-muted-foreground/60">
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
				type="button"
				onclick={() => scrollToBottom(true)}
				class="fixed bottom-24 right-6 z-10 flex h-10 w-10 items-center justify-center rounded-full bg-primary text-primary-foreground shadow-lg transition-all hover:scale-110"
			>
				<ChevronDown class="h-5 w-5" />
			</button>
		{/if}
	</div>

	<!-- Input Area -->
	<div class="shrink-0 border-t border-border/50 bg-card/80 p-4 backdrop-blur-sm">
		{#if !isSubscribed && !isLoading}
			<div class="rounded-2xl bg-yellow-500/10 border border-yellow-500/30 px-4 py-3 text-center mb-3">
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
		
		{#if !group?.my_state?.is_joined}
			<div class="rounded-2xl bg-muted/50 px-4 py-3 text-center">
				<p class="text-sm text-muted-foreground">你还不是群成员，无法发送消息</p>
			</div>
		{:else if group?.my_state?.is_muted}
			<div class="rounded-2xl bg-muted/50 px-4 py-3 text-center">
				<p class="text-sm text-muted-foreground">你已被禁言，无法发送消息</p>
			</div>
		{:else}
			<div class="flex items-end gap-2">
				<textarea
					{@attach autofocus(shouldFocusInput)}
					placeholder="发送消息..."
					bind:value={messageText}
					rows={1}
					class="max-h-32 min-h-[42px] flex w-full resize-none rounded-3xl border border-muted-foreground/20 bg-muted/30 px-4 py-2.5 text-[15px] shadow-sm shadow-black/5 transition-colors placeholder:text-muted-foreground/50 focus-visible:bg-muted/50 focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-ring disabled:cursor-not-allowed disabled:opacity-50"
					disabled={isSending}
					onkeydown={(e) => {
						if (e.key === 'Enter' && !e.shiftKey) {
							e.preventDefault();
							sendMessage();
						}
					}}
				></textarea>
				<Button
					size="icon"
					class="h-[42px] w-[42px] shrink-0 rounded-full transition-transform active:scale-95"
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
		{/if}
	</div>
</div>

<!-- Image Preview Modal -->
{#if selectedImage}
	<div 
		class="fixed inset-0 z-50 flex items-center justify-center bg-black/90 p-4"
		onclick={closeImagePreview}
		role="button"
		tabindex="0"
		onkeydown={(e) => e.key === 'Escape' && closeImagePreview()}
	>
		<button
			type="button"
			onclick={closeImagePreview}
			class="absolute right-4 top-4 rounded-full bg-white/10 p-2 text-white backdrop-blur transition-colors hover:bg-white/20"
		>
			<X class="h-6 w-6" />
		</button>
		<img
			src={selectedImage}
			alt="预览"
			class="max-h-full max-w-full rounded-lg object-contain"
			onclick={(e) => e.stopPropagation()}
			role="presentation"
		/>
	</div>
{/if}
