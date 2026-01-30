<script lang="ts">
	import { onMount } from 'svelte';
	import { privateChatsService } from '$lib/api/services';
	import { isSuccess } from '$lib/api/client';
	import { toast } from '$lib/stores/toast.svelte';
	import { Card, CardContent, CardHeader, CardTitle } from '$lib/components/ui/card';
	import { Avatar, AvatarFallback, AvatarImage } from '$lib/components/ui/avatar';
	import { Badge } from '$lib/components/ui/badge';
	import { Skeleton } from '$lib/components/ui/skeleton';
	import type { PrivateChatListItem } from '$lib/types/api';
	import MessageCircle from 'lucide-svelte/icons/message-circle';

	let chats = $state<PrivateChatListItem[]>([]);
	let isLoading = $state(true);

	async function loadChats() {
		try {
			const response = await privateChatsService.getChats({ limit: 50 });
			if (isSuccess(response) && response.data) {
				chats = response.data.items;
			}
		} catch {
			toast.error('加载失败');
		} finally {
			isLoading = false;
		}
	}

	function formatTime(dateStr: string): string {
		const date = new Date(dateStr);
		const now = new Date();
		const diff = now.getTime() - date.getTime();
		const minutes = Math.floor(diff / 60000);
		const hours = Math.floor(diff / 3600000);
		const days = Math.floor(diff / 86400000);

		if (minutes < 1) return '刚刚';
		if (minutes < 60) return `${minutes}分钟前`;
		if (hours < 24) return `${hours}小时前`;
		if (days < 7) return `${days}天前`;
		return date.toLocaleDateString('zh-CN');
	}

	onMount(() => {
		loadChats();
	});
</script>

<svelte:head>
	<title>消息 - HotSpot</title>
</svelte:head>

<div class="space-y-6">
	<h1 class="text-2xl font-bold">消息</h1>

	{#if isLoading}
		<div class="space-y-2">
			{#each Array(5) as _}
				<Card>
					<CardContent class="flex items-center gap-4 p-4">
						<Skeleton class="h-12 w-12 rounded-full" />
						<div class="flex-1 space-y-2">
							<Skeleton class="h-4 w-24" />
							<Skeleton class="h-3 w-48" />
						</div>
					</CardContent>
				</Card>
			{/each}
		</div>
	{:else if chats.length === 0}
		<Card>
			<CardContent class="flex flex-col items-center justify-center py-12">
				<MessageCircle class="mb-4 h-12 w-12 text-muted-foreground" />
				<p class="text-lg font-medium">暂无消息</p>
				<p class="mt-1 text-muted-foreground">去探索发现感兴趣的人吧</p>
			</CardContent>
		</Card>
	{:else}
		<div class="space-y-2">
			{#each chats as chat (chat.other_user.user_id)}
				<a href="/chat/users/{chat.other_user.user_id}">
					<Card class="cursor-pointer transition-colors hover:bg-accent/50">
						<CardContent class="flex items-center gap-4 p-4">
							<div class="relative">
								<Avatar class="h-12 w-12">
									<AvatarImage src={chat.other_user.avatar_url} alt={chat.other_user.nickname} />
									<AvatarFallback>{chat.other_user.nickname.slice(0, 2)}</AvatarFallback>
								</Avatar>
								{#if chat.is_online}
									<div class="absolute bottom-0 right-0 h-3 w-3 rounded-full border-2 border-background bg-green-500"></div>
								{/if}
							</div>
							<div class="min-w-0 flex-1">
								<div class="flex items-center justify-between">
									<p class="font-medium">{chat.other_user.nickname}</p>
									<span class="text-xs text-muted-foreground">
										{formatTime(chat.last_activity_at)}
									</span>
								</div>
								<div class="flex items-center justify-between">
									<p class="truncate text-sm text-muted-foreground">
										{#if chat.last_message}
											{chat.last_message.content || '[媒体消息]'}
										{:else}
											暂无消息
										{/if}
									</p>
									{#if chat.unread_count > 0}
										<Badge variant="destructive" class="ml-2">
											{chat.unread_count > 99 ? '99+' : chat.unread_count}
										</Badge>
									{/if}
								</div>
							</div>
						</CardContent>
					</Card>
				</a>
			{/each}
		</div>
	{/if}
</div>
