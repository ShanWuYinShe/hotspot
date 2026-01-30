<script lang="ts">
	import { page } from '$app/stores';
	import { onMount } from 'svelte';
	import { privateChatsService, privateMessagesService, usersService } from '$lib/api/services';
	import { isSuccess } from '$lib/api/client';
	import { authStore } from '$lib/stores/auth.svelte';
	import { toast } from '$lib/stores/toast.svelte';
	import { Button } from '$lib/components/ui/button';
	import { Avatar, AvatarFallback, AvatarImage } from '$lib/components/ui/avatar';
	import { Textarea } from '$lib/components/ui/textarea';
	import { Skeleton } from '$lib/components/ui/skeleton';
	import { ScrollArea } from '$lib/components/ui/scroll-area';
	import type { PrivateMessageItem, UserInfo } from '$lib/types/api';
	import ArrowLeft from 'lucide-svelte/icons/arrow-left';
	import Send from 'lucide-svelte/icons/send';
	import Loader2 from 'lucide-svelte/icons/loader-2';
	import MoreVertical from 'lucide-svelte/icons/more-vertical';

	const userId = parseInt($page.params.id!);

	let otherUser = $state<UserInfo | null>(null);
	let messages = $state<PrivateMessageItem[]>([]);
	let isLoading = $state(true);
	let messageText = $state('');
	let isSending = $state(false);

	async function loadChat() {
		try {
			// 获取对方信息
			const userResponse = await usersService.getUser({ user_id: userId });
			if (isSuccess(userResponse) && userResponse.data) {
				otherUser = userResponse.data;
			}

			// 获取消息历史
			const messagesResponse = await privateMessagesService.getHistory({
				target_user_id: userId,
				limit: 50
			});
			if (isSuccess(messagesResponse) && messagesResponse.data) {
				messages = messagesResponse.data.items.reverse();
			}

			// 标记已读
			await privateMessagesService.markAsRead({ target_user_id: userId });
		} catch {
			toast.error('加载失败');
		} finally {
			isLoading = false;
		}
	}

	async function sendMessage() {
		if (!messageText.trim()) return;

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

			if (isSuccess(response)) {
				// 添加到消息列表（简化版，实际应该等 WebSocket 推送）
				messages = [
					...messages,
					{
						message_id: `local-${Date.now()}`,
						type: 'text',
						content: text,
						sender: {
							user_id: authStore.currentUserId!,
							nickname: authStore.currentAccount?.nickname || '',
							avatar_url: authStore.currentAccount?.avatar_url,
							user_type: 'normal'
						},
						sent_at: new Date().toISOString(),
						is_recalled: false
					}
				];
			} else {
				messageText = text;
				toast.error(response.message || '发送失败');
			}
		} catch {
			messageText = text;
			toast.error('发送失败');
		} finally {
			isSending = false;
		}
	}

	function formatTime(dateStr: string): string {
		const date = new Date(dateStr);
		return date.toLocaleTimeString('zh-CN', { hour: '2-digit', minute: '2-digit' });
	}

	function isSelf(message: PrivateMessageItem): boolean {
		return message.sender.user_id === authStore.currentUserId;
	}

	onMount(() => {
		loadChat();
	});
</script>

<svelte:head>
	<title>与 {otherUser?.nickname ?? '用户'} 的对话 - HotSpot</title>
</svelte:head>

<div class="flex h-[calc(100vh-8rem)] flex-col">
	<!-- Header -->
	<div class="flex items-center gap-3 border-b pb-4">
		<Button variant="ghost" size="icon" onclick={() => history.back()}>
			<ArrowLeft class="h-5 w-5" />
		</Button>
		{#if isLoading}
			<Skeleton class="h-10 w-10 rounded-full" />
			<Skeleton class="h-5 w-24" />
		{:else if otherUser}
			<a href="/users/{userId}" class="flex items-center gap-3">
				<Avatar class="h-10 w-10">
					<AvatarImage src={otherUser.avatar_url} alt={otherUser.nickname} />
					<AvatarFallback>{otherUser.nickname.slice(0, 2)}</AvatarFallback>
				</Avatar>
				<span class="font-medium">{otherUser.nickname}</span>
			</a>
		{/if}
		<div class="ml-auto">
			<Button variant="ghost" size="icon">
				<MoreVertical class="h-5 w-5" />
			</Button>
		</div>
	</div>

	<!-- Messages -->
	<ScrollArea class="flex-1 py-4">
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
			<div class="space-y-4 px-4">
				{#each messages as message (message.message_id)}
					<div class="flex {isSelf(message) ? 'justify-end' : ''}">
						<div class="flex max-w-[70%] gap-2 {isSelf(message) ? 'flex-row-reverse' : ''}">
							{#if !isSelf(message)}
								<Avatar class="h-8 w-8">
									<AvatarImage src={message.sender.avatar_url} alt={message.sender.nickname} />
									<AvatarFallback>{message.sender.nickname.slice(0, 2)}</AvatarFallback>
								</Avatar>
							{/if}
							<div>
								<div
									class="rounded-lg px-3 py-2 {isSelf(message)
										? 'bg-primary text-primary-foreground'
										: 'bg-muted'}"
								>
									{#if message.is_recalled}
										<p class="text-sm italic opacity-70">消息已撤回</p>
									{:else if message.type === 'text'}
										<p class="whitespace-pre-wrap text-sm">{message.content}</p>
									{:else if message.type === 'image' && message.media}
										<img
											src={message.media.url}
											alt="图片"
											class="max-w-[200px] rounded"
										/>
									{:else}
										<p class="text-sm">[{message.type}]</p>
									{/if}
								</div>
								<p class="mt-1 text-xs text-muted-foreground {isSelf(message) ? 'text-right' : ''}">
									{formatTime(message.sent_at)}
								</p>
							</div>
						</div>
					</div>
				{/each}
			</div>
		{/if}
	</ScrollArea>

	<!-- Input -->
	<div class="border-t pt-4">
		<div class="flex gap-2">
			<Textarea
				placeholder="输入消息..."
				bind:value={messageText}
				rows={1}
				class="min-h-[40px] resize-none"
				disabled={isSending}
				onkeydown={(e) => {
					if (e.key === 'Enter' && !e.shiftKey) {
						e.preventDefault();
						sendMessage();
					}
				}}
			/>
			<Button
				size="icon"
				onclick={sendMessage}
				disabled={!messageText.trim() || isSending}
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
