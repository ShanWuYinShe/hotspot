<script lang="ts">
	import { onMount } from 'svelte';
	import { goto } from '$app/navigation';
	import { notificationStore } from '$lib/stores/notification.svelte';
	import { usePagination } from '$lib/hooks/use-pagination.svelte';
	import type { NotificationItem } from '$lib/types/api';
	import { Button } from '$lib/components/ui/button';
	import { Card, CardContent } from '$lib/components/ui/card';
	import EmptyState from '$lib/components/common/empty-state.svelte';
	import Bell from 'lucide-svelte/icons/bell';
	import Heart from 'lucide-svelte/icons/heart';
	import MessageCircle from 'lucide-svelte/icons/message-circle';
	import UserPlus from 'lucide-svelte/icons/user-plus';
	import Info from 'lucide-svelte/icons/info';
	import Users from 'lucide-svelte/icons/users';
	import CheckCheck from 'lucide-svelte/icons/check-check';
	import Loader2 from 'lucide-svelte/icons/loader-2';

	const pagination = usePagination<NotificationItem>({
		uniqueKey: (item) => item.notification_id,
		initialLoading: true
	});

	onMount(() => {
		pagination.load(
			(cursor) =>
				notificationStore.fetchHistory({
					pagination: { cursor, limit: 20 }
				}),
			true
		);
	});

	async function markAllRead() {
		const ok = await notificationStore.markAllAsRead();
		if (ok) {
			pagination.updateItem((i) => ({ ...i, is_read: true }));
		}
	}

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
		case 'group_invite_accepted':
		case 'group_application_received':
		case 'group_application_accepted':
			return Users;
		case 'system_announcement':
			return Info;
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
		case 'group_invite_accepted':
			return '邀请被接受';
		case 'group_application_received':
			return '入群申请';
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
				return `${p.actor.nickname} 回复了你: ${p.replied_comment_preview}`;
			case 'mentioned':
				return `${p.actor.nickname} 在内容中提及了你`;
		case 'group_invite_received':
			return `${p.actor.nickname} 邀请你加入 ${p.group_name}`;
		case 'group_invite_accepted':
			return `${p.user.nickname} 接受了群组 ${p.group_name} 的邀请`;
		case 'group_application_received':
			return `${p.applicant.nickname} 申请加入 ${p.group_name}`;
		case 'group_application_accepted':
			return `你已加入群组 ${p.group_name}`;
		case 'system_announcement':
			return p.severity === 'error' ? '重要公告' : p.severity === 'warning' ? '提醒' : '公告';
		default:
			return '';
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
		case 'group_invite_accepted':
		case 'group_application_received':
		case 'group_application_accepted':
			return `/groups/${p.group_id}`;
		case 'system_announcement':
			return '#';
		default:
			return '#';
		}
	}

	async function handleNotificationClick(item: NotificationItem) {
		if (!item.is_read) {
			await notificationStore.markAsRead([item.notification_id]);
			pagination.updateItem((i) =>
				i.notification_id === item.notification_id ? { ...i, is_read: true } : null
			);
		}
		goto(getNotificationLink(item));
	}
</script>

<svelte:head>
	<title>通知 - HotSpot</title>
</svelte:head>

<div class="space-y-6">
	<div class="flex items-center justify-between">
		<h1 class="text-2xl font-bold">通知</h1>
		{#if notificationStore.unreadCount > 0}
			<Button variant="outline" size="sm" onclick={markAllRead}>
				<CheckCheck class="mr-2 h-4 w-4" />
				全部已读
			</Button>
		{/if}
	</div>

	{#if pagination.loading}
		<div class="flex justify-center py-12">
			<Loader2 class="h-8 w-8 animate-spin text-muted-foreground" />
		</div>
	{:else if pagination.error}
		<EmptyState icon={Bell} title="加载失败" description={pagination.error} />
	{:else if pagination.items.length === 0}
		<EmptyState icon={Bell} title="暂无通知" />
	{:else}
		<div class="space-y-2">
			{#each pagination.items as item (item.notification_id)}
				{@const Icon = getNotificationIcon(item.type)}
				<button
					type="button"
					class="w-full text-left"
					onclick={() => handleNotificationClick(item)}
				>
					<Card class="cursor-pointer transition-colors hover:bg-accent/50">
						<CardContent class="flex items-start gap-4 p-4">
							<div class="flex h-10 w-10 items-center justify-center rounded-full bg-muted">
								<Icon class="h-5 w-5" />
							</div>
							<div class="min-w-0 flex-1">
								<div class="flex items-start justify-between gap-2">
									<div>
										<span class="font-medium">{getNotificationTitle(item)}</span>
										<span class="text-muted-foreground">
											{getNotificationContent(item)}
										</span>
									</div>
									{#if !item.is_read}
										<div class="mt-1 h-2 w-2 shrink-0 rounded-full bg-primary"></div>
									{/if}
								</div>
								<p class="mt-1 text-xs text-muted-foreground">
									{formatTime(item.created_at)}
								</p>
							</div>
						</CardContent>
					</Card>
				</button>
			{/each}
		</div>
		{#if pagination.hasMore}
			<div class="flex justify-center pt-4">
				<Button
					variant="outline"
					disabled={pagination.loadingMore}
					onclick={() =>
						pagination.load(
							(cursor) =>
								notificationStore.fetchHistory({
									pagination: { cursor, limit: 20 }
								})
						)}
				>
					{#if pagination.loadingMore}
						<Loader2 class="mr-2 h-4 w-4 animate-spin" />
					{/if}
					加载更多
				</Button>
			</div>
		{/if}
	{/if}
</div>
