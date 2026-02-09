<script lang="ts">
	import { notificationStore, type RealtimeNotification } from '$lib/stores/notification.svelte';
	import { Button } from '$lib/components/ui/button';
	import { Card, CardContent } from '$lib/components/ui/card';
	import EmptyState from '$lib/components/common/empty-state.svelte';
	import Bell from 'lucide-svelte/icons/bell';
	import Heart from 'lucide-svelte/icons/heart';
	import MessageCircle from 'lucide-svelte/icons/message-circle';
	import UserPlus from 'lucide-svelte/icons/user-plus';
	import AtSign from 'lucide-svelte/icons/at-sign';
	import Info from 'lucide-svelte/icons/info';
	import CheckCheck from 'lucide-svelte/icons/check-check';

	// 通知数据来自 WebSocket 推送（spec 中无通知 REST API）
	let notifications = $derived(notificationStore.unreadNotifications);

	function markAllRead() {
		notificationStore.markAllAsRead();
	}

	function getNotificationIcon(type: string) {
		if (type.includes('like')) return Heart;
		if (type.includes('comment') || type.includes('reply')) return MessageCircle;
		if (type.includes('follow') || type.includes('mutual')) return UserPlus;
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

	function getNotificationLink(notification: RealtimeNotification): string {
		// 基于通知类型推断跳转链接
		if (notification.type === 'like' || notification.type === 'comment' || notification.type === 'reply') {
			return '#'; // 瞬间相关，需要通知中携带 target_id
		}
		if (notification.type === 'follower' || notification.type === 'mutual') {
			return '#'; // 用户相关
		}
		if (notification.type === 'group_invite' || notification.type === 'group_application') {
			return '#'; // 群组相关
		}
		return '#';
	}
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

	{#if notifications.length === 0}
		<EmptyState icon={Bell} title="暂无通知" />
	{:else}
		<div class="space-y-2">
			{#each notifications as notification (notification.id)}
				{@const Icon = getNotificationIcon(notification.type)}
				<a href={getNotificationLink(notification)}>
					<Card class="cursor-pointer transition-colors hover:bg-accent/50">
						<CardContent class="flex items-start gap-4 p-4">
							<div class="flex h-10 w-10 items-center justify-center rounded-full bg-muted">
								<Icon class="h-5 w-5" />
							</div>
							<div class="min-w-0 flex-1">
								<div class="flex items-start justify-between gap-2">
									<div>
										<span class="font-medium">{notification.title}</span>
										<span class="text-muted-foreground">
											{notification.content}
										</span>
									</div>
									{#if !notification.read}
										<div class="mt-1 h-2 w-2 rounded-full bg-primary"></div>
									{/if}
								</div>
								<p class="mt-1 text-xs text-muted-foreground">
									{formatTime(notification.timestamp)}
								</p>
							</div>
						</CardContent>
					</Card>
				</a>
			{/each}
		</div>
	{/if}
</div>
