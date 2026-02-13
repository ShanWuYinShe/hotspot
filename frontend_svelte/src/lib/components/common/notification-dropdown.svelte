<script lang="ts">
	import { goto } from '$app/navigation';
	import { notificationStore } from '$lib/stores/notification.svelte';
	import type { NotificationItem } from '$lib/types/api';
	import * as DropdownMenu from '$lib/components/ui/dropdown-menu';
	import { Button } from '$lib/components/ui/button';
	import { Badge } from '$lib/components/ui/badge';
	import { ScrollArea } from '$lib/components/ui/scroll-area';
	import Bell from 'lucide-svelte/icons/bell';
	import BellDot from 'lucide-svelte/icons/bell-dot';
	import Check from 'lucide-svelte/icons/check';
	import CheckCheck from 'lucide-svelte/icons/check-check';
	import Settings from 'lucide-svelte/icons/settings';
	import UserPlus from 'lucide-svelte/icons/user-plus';
	import Heart from 'lucide-svelte/icons/heart';
	import MessageCircle from 'lucide-svelte/icons/message-circle';
	import Users from 'lucide-svelte/icons/users';
	import { isSuccess } from '$lib/api/client';

	let open = $state(false);
	let recentNotifications = $state<NotificationItem[]>([]);
	let loading = $state(false);

	$effect(() => {
		if (open) {
			loadRecent();
		}
	});

	async function loadRecent() {
		loading = true;
		const res = await notificationStore.fetchHistory({ pagination: { limit: 5 } });
		if (isSuccess(res) && res.data) {
			recentNotifications = res.data.items;
		} else {
			recentNotifications = [];
		}
		loading = false;
	}

	const hasMore = $derived(notificationStore.unreadCount > 5);

	function getNotificationIcon(type: NotificationItem['type']) {
		switch (type) {
			case 'new_follower':
			case 'mutual_follow':
				return UserPlus;
			case 'moment_liked':
			case 'comment_liked':
				return Heart;
			case 'moment_commented':
			case 'comment_reply':
			case 'mentioned':
				return MessageCircle;
			case 'group_invite_received':
			case 'group_application_accepted':
				return Users;
			case 'system_announcement':
				return Bell;
			default:
				return Bell;
		}
	}

	function getNotificationTitle(item: NotificationItem): string {
		const p = item.payload;
		switch (p.type) {
			case 'new_follower':
				return '新增关注者';
			case 'mutual_follow':
				return '新好友';
			case 'moment_liked':
				return '动态被点赞';
			case 'moment_commented':
				return '动态被评论';
			case 'comment_liked':
				return '评论被点赞';
			case 'comment_reply':
				return '评论被回复';
			case 'mentioned':
				return '被提及';
			case 'group_invite_received':
				return '群组邀请';
			case 'group_application_accepted':
				return '入群申请通过';
			case 'system_announcement':
				return '系统公告';
			default:
				return '通知';
		}
	}

	function getNotificationContent(item: NotificationItem): string {
		const p = item.payload;
		switch (p.type) {
			case 'new_follower':
				return `${p.actor.nickname} 关注了你`;
			case 'mutual_follow':
				return `你和 ${p.actor.nickname} 成为了好友`;
			case 'moment_liked':
				return `${p.actor.nickname} 赞了你的动态`;
			case 'moment_commented':
				return `${p.actor.nickname}: ${p.comment_preview}`;
			case 'comment_liked':
				return `${p.actor.nickname} 赞了你的评论`;
			case 'comment_reply':
				return `${p.actor.nickname} 回复了你: ${p.comment_preview}`;
			case 'mentioned':
				return `${p.actor.nickname} 在内容中提及了你`;
			case 'group_invite_received':
				return `${p.actor.nickname} 邀请你加入 ${p.group_name}`;
			case 'group_application_accepted':
				return `你已加入群组 ${p.group_name}`;
			case 'system_announcement':
				return p.severity === 'error' ? '重要公告' : p.severity === 'warning' ? '提醒' : '公告';
			default:
				return '';
		}
	}

	function getNotificationLink(item: NotificationItem): string {
		const p = item.payload;
		switch (p.type) {
			case 'new_follower':
			case 'mutual_follow':
				return `/users/${p.actor.user_id}`;
			case 'moment_liked':
			case 'moment_commented':
			case 'comment_liked':
			case 'comment_reply':
			case 'mentioned':
				return `/moments/${p.moment_id}`;
			case 'group_invite_received':
			case 'group_application_accepted':
				return `/groups/${p.group_id}`;
			case 'system_announcement':
				return '#';
			default:
				return '#';
		}
	}

	function formatTimestamp(timestamp: string): string {
		const now = new Date();
		const time = new Date(timestamp);
		const diff = now.getTime() - time.getTime();

		const seconds = Math.floor(diff / 1000);
		const minutes = Math.floor(seconds / 60);
		const hours = Math.floor(minutes / 60);
		const days = Math.floor(hours / 24);

		if (seconds < 60) return `${seconds}秒前`;
		if (minutes < 60) return `${minutes}分钟前`;
		if (hours < 24) return `${hours}小时前`;
		if (days < 7) return `${days}天前`;

		return time.toLocaleDateString('zh-CN', { month: 'short', day: 'numeric' });
	}

	async function handleMarkAllRead() {
		await notificationStore.markAllAsRead();
		recentNotifications = recentNotifications.map((i) => ({ ...i, is_read: true }));
	}

	async function handleMarkAsRead(notificationId: string, e: MouseEvent) {
		e.preventDefault();
		e.stopPropagation();
		const ok = await notificationStore.markAsRead([notificationId]);
		if (ok) {
			recentNotifications = recentNotifications.map((i) =>
				i.notification_id === notificationId ? { ...i, is_read: true } : i
			);
		}
	}

	async function handleItemClick(item: NotificationItem) {
		if (!item.is_read) {
			await notificationStore.markAsRead([item.notification_id]);
			recentNotifications = recentNotifications.map((i) =>
				i.notification_id === item.notification_id ? { ...i, is_read: true } : i
			);
		}
		open = false;
		goto(getNotificationLink(item));
	}

	function handleViewAll() {
		open = false;
		goto('/notifications');
	}
</script>

<DropdownMenu.Root bind:open>
	<DropdownMenu.Trigger>
		{#snippet child({ props })}
			<Button variant="ghost" size="icon" class="relative rounded-full" aria-label="通知" {...props}>
				{#if notificationStore.unreadCount > 0}
					<BellDot class="h-5 w-5" />
					<Badge
						variant="destructive"
						class="absolute -right-1 -top-1 flex h-5 min-w-5 items-center justify-center rounded-full px-1 text-[10px] font-bold ring-2 ring-background"
					>
						{notificationStore.unreadCount > 99 ? '99+' : notificationStore.unreadCount}
					</Badge>
				{:else}
					<Bell class="h-5 w-5" />
				{/if}
			</Button>
		{/snippet}
	</DropdownMenu.Trigger>

	<DropdownMenu.Content class="w-96 p-0" align="end" sideOffset={8}>
		<!-- Header -->
		<div class="flex items-center justify-between border-b border-border/50 px-4 py-3">
			<div class="flex items-center gap-2">
				<h3 class="text-sm font-semibold">通知</h3>
				{#if notificationStore.unreadCount > 0}
					<Badge variant="secondary" class="h-5 rounded-full px-2 text-[10px]">
						{notificationStore.unreadCount}
					</Badge>
				{/if}
			</div>
			<div class="flex items-center gap-1">
				{#if notificationStore.unreadCount > 0}
					<Button
						variant="ghost"
						size="icon"
						class="h-8 w-8"
						onclick={handleMarkAllRead}
						title="全部已读"
						aria-label="全部标为已读"
					>
						<CheckCheck class="h-4 w-4" />
					</Button>
				{/if}
				<Button
					variant="ghost"
					size="icon"
					class="h-8 w-8"
					onclick={() => { open = false; goto('/settings/notifications'); }}
					title="通知设置"
					aria-label="通知设置"
				>
					<Settings class="h-4 w-4" />
				</Button>
			</div>
		</div>

		<!-- Notifications List -->
		{#if loading}
			<div class="flex flex-col items-center justify-center py-12">
				<div class="h-8 w-8 animate-pulse rounded-full bg-muted"></div>
				<p class="mt-3 text-sm text-muted-foreground">加载中...</p>
			</div>
		{:else if recentNotifications.length === 0}
			<div class="flex flex-col items-center justify-center py-12">
				<div class="mb-3 flex h-16 w-16 items-center justify-center rounded-full bg-muted">
					<Bell class="h-8 w-8 text-muted-foreground" />
				</div>
				<p class="text-sm font-medium">暂无通知</p>
				<p class="text-xs text-muted-foreground">你的通知会显示在这里</p>
			</div>
		{:else}
			<ScrollArea class="max-h-96">
				<div class="divide-y divide-border/50">
					{#each recentNotifications as item (item.notification_id)}
						{@const Icon = getNotificationIcon(item.type)}
						<div class="group relative flex gap-3 p-4 transition-colors hover:bg-accent/50">
							<button
								type="button"
								class="absolute inset-0 flex items-start gap-3 text-left"
								onclick={() => handleItemClick(item)}
							>
								<div class="flex h-10 w-10 shrink-0 items-center justify-center rounded-full bg-primary/10 text-primary">
									<Icon class="h-5 w-5" />
								</div>
								<div class="min-w-0 flex-1 pr-8">
									<div class="mb-0.5 text-sm font-medium">{getNotificationTitle(item)}</div>
									<p class="mb-1 text-sm text-muted-foreground line-clamp-2">
										{getNotificationContent(item)}
									</p>
									<span class="text-xs text-muted-foreground">
										{formatTimestamp(item.created_at)}
									</span>
								</div>
							</button>
							{#if !item.is_read}
								<button
									type="button"
									class="absolute right-2 top-2 z-10 flex h-6 w-6 items-center justify-center rounded-full opacity-0 transition-all hover:bg-background group-hover:opacity-100"
									onclick={(e) => handleMarkAsRead(item.notification_id, e)}
									title="标记为已读"
								>
									<Check class="h-3.5 w-3.5" />
								</button>
							{/if}
						</div>
					{/each}
				</div>
			</ScrollArea>

			<!-- Footer -->
			<div class="border-t border-border/50 p-2">
				<Button
					variant="ghost"
					size="sm"
					class="w-full justify-center text-sm"
					onclick={handleViewAll}
				>
					{hasMore ? `查看全部 ${notificationStore.unreadCount} 条通知` : '查看所有通知'}
				</Button>
			</div>
		{/if}
	</DropdownMenu.Content>
</DropdownMenu.Root>
