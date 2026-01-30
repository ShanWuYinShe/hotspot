<script lang="ts">
	import { onMount } from 'svelte';
	import { notificationsService } from '$lib/api/services';
	import { isSuccess } from '$lib/api/client';
	import { toast } from '$lib/stores/toast.svelte';
	import { Button } from '$lib/components/ui/button';
	import { Card, CardContent, CardHeader, CardTitle } from '$lib/components/ui/card';
	import { Avatar, AvatarFallback, AvatarImage } from '$lib/components/ui/avatar';
	import { Skeleton } from '$lib/components/ui/skeleton';
	import { Tabs, TabsContent, TabsList, TabsTrigger } from '$lib/components/ui/tabs';
	import type { NotificationItem, NotificationFilterType } from '$lib/types/api';
	import Bell from 'lucide-svelte/icons/bell';
	import Heart from 'lucide-svelte/icons/heart';
	import MessageCircle from 'lucide-svelte/icons/message-circle';
	import UserPlus from 'lucide-svelte/icons/user-plus';
	import AtSign from 'lucide-svelte/icons/at-sign';
	import Info from 'lucide-svelte/icons/info';
	import Check from 'lucide-svelte/icons/check';
	import CheckCheck from 'lucide-svelte/icons/check-check';

	let notifications = $state<NotificationItem[]>([]);
	let isLoading = $state(true);
	let currentFilter = $state<NotificationFilterType | undefined>(undefined);

	async function loadNotifications(filter?: NotificationFilterType) {
		isLoading = true;
		currentFilter = filter;
		try {
			const response = await notificationsService.getNotifications({
				type: filter,
				limit: 50
			});
			if (isSuccess(response) && response.data) {
				notifications = response.data.items;
			}
		} catch {
			toast.error('加载失败');
		} finally {
			isLoading = false;
		}
	}

	async function markAllRead() {
		try {
			const response = await notificationsService.markRead({
				type: currentFilter
			});
			if (isSuccess(response)) {
				notifications = notifications.map((n) => ({ ...n, is_read: true }));
				toast.success('已全部标记为已读');
			}
		} catch {
			toast.error('操作失败');
		}
	}

	function getNotificationIcon(type: string) {
		if (type.includes('like')) return Heart;
		if (type.includes('comment') || type.includes('reply')) return MessageCircle;
		if (type.includes('follow')) return UserPlus;
		if (type.includes('mention')) return AtSign;
		return Info;
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

	function getNotificationLink(notification: NotificationItem): string {
		if (notification.target_type === 'moment' && notification.target_id) {
			return `/moments/${notification.target_id}`;
		}
		if (notification.target_type === 'user' && notification.sender) {
			return `/users/${notification.sender.user_id}`;
		}
		if (notification.target_type === 'group' && notification.target_id) {
			return `/groups/${notification.target_id}`;
		}
		return '#';
	}

	onMount(() => {
		loadNotifications();
	});
</script>

<svelte:head>
	<title>通知 - HotSpot</title>
</svelte:head>

<div class="space-y-6">
	<div class="flex items-center justify-between">
		<h1 class="text-2xl font-bold">通知</h1>
		<Button variant="outline" size="sm" onclick={markAllRead}>
			<CheckCheck class="mr-2 h-4 w-4" />
			全部已读
		</Button>
	</div>

	<Tabs value={currentFilter ?? 'all'} onValueChange={(v) => loadNotifications(v === 'all' ? undefined : v as NotificationFilterType)}>
		<TabsList>
			<TabsTrigger value="all">全部</TabsTrigger>
			<TabsTrigger value="like">点赞</TabsTrigger>
			<TabsTrigger value="comment">评论</TabsTrigger>
			<TabsTrigger value="follow">关注</TabsTrigger>
			<TabsTrigger value="system">系统</TabsTrigger>
		</TabsList>

		<TabsContent value={currentFilter ?? 'all'} class="mt-4">
			{#if isLoading}
				<div class="space-y-2">
					{#each Array(5) as _}
						<Card>
							<CardContent class="flex items-center gap-4 p-4">
								<Skeleton class="h-10 w-10 rounded-full" />
								<div class="flex-1 space-y-2">
									<Skeleton class="h-4 w-48" />
									<Skeleton class="h-3 w-24" />
								</div>
							</CardContent>
						</Card>
					{/each}
				</div>
			{:else if notifications.length === 0}
				<Card>
					<CardContent class="flex flex-col items-center justify-center py-12">
						<Bell class="mb-4 h-12 w-12 text-muted-foreground" />
						<p class="text-muted-foreground">暂无通知</p>
					</CardContent>
				</Card>
			{:else}
				<div class="space-y-2">
					{#each notifications as notification (notification.notification_id)}
						{@const Icon = getNotificationIcon(notification.type)}
						<a href={getNotificationLink(notification)}>
							<Card class="cursor-pointer transition-colors hover:bg-accent/50 {notification.is_read ? 'opacity-70' : ''}">
								<CardContent class="flex items-start gap-4 p-4">
									{#if notification.sender}
										<Avatar class="h-10 w-10">
											<AvatarImage src={notification.sender.avatar_url} alt={notification.sender.nickname} />
											<AvatarFallback>{notification.sender.nickname.slice(0, 2)}</AvatarFallback>
										</Avatar>
									{:else}
										<div class="flex h-10 w-10 items-center justify-center rounded-full bg-muted">
											<Icon class="h-5 w-5" />
										</div>
									{/if}
									<div class="min-w-0 flex-1">
										<div class="flex items-start justify-between gap-2">
											<div>
												{#if notification.sender}
													<span class="font-medium">{notification.sender.nickname}</span>
												{/if}
												<span class="text-muted-foreground">
													{notification.content || notification.title}
												</span>
											</div>
											{#if !notification.is_read}
												<div class="mt-1 h-2 w-2 rounded-full bg-primary"></div>
											{/if}
										</div>
										{#if notification.target_text_preview}
											<p class="mt-1 truncate text-sm text-muted-foreground">
												{notification.target_text_preview}
											</p>
										{/if}
										<p class="mt-1 text-xs text-muted-foreground">
											{formatTime(notification.created_at)}
										</p>
									</div>
								</CardContent>
							</Card>
						</a>
					{/each}
				</div>
			{/if}
		</TabsContent>
	</Tabs>
</div>
