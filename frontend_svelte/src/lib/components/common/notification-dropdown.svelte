<script lang="ts">
	import { goto } from '$app/navigation';
	import { notificationStore } from '$lib/stores/notification.svelte';
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
	
	let open = $state(false);
	
	// 显示最近5条通知
	const recentNotifications = $derived(notificationStore.unreadNotifications.slice(0, 5));
	const hasMore = $derived(notificationStore.unreadNotifications.length > 5);
	
	function getNotificationIcon(type: string) {
		switch (type) {
			case 'follower':
			case 'mutual':
				return UserPlus;
			case 'like':
				return Heart;
			case 'comment':
			case 'reply':
				return MessageCircle;
			case 'group_invite':
			case 'group_application':
				return Users;
			default:
				return Bell;
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
	
	function handleMarkAllRead() {
		notificationStore.markAllAsRead();
	}
	
	function handleMarkAsRead(id: string, e: MouseEvent) {
		e.stopPropagation();
		notificationStore.markAsRead(id);
	}
	
	function handleViewAll() {
		open = false;
		goto('/notifications');
	}
</script>

<DropdownMenu.Root bind:open>
	<DropdownMenu.Trigger>
		{#snippet child({ props })}
			<Button variant="ghost" size="icon" class="relative rounded-full" {...props}>
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
					>
						<CheckCheck class="h-4 w-4" />
					</Button>
				{/if}
				<Button
					variant="ghost"
					size="icon"
					class="h-8 w-8"
					onclick={handleViewAll}
					title="通知设置"
				>
					<Settings class="h-4 w-4" />
				</Button>
			</div>
		</div>
		
		<!-- Notifications List -->
		{#if recentNotifications.length === 0}
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
					{#each recentNotifications as notification (notification.id)}
						{@const Icon = getNotificationIcon(notification.type)}
						<div class="group relative flex gap-3 p-4 transition-colors hover:bg-accent/50">
							<!-- Icon -->
							<div class="flex h-10 w-10 shrink-0 items-center justify-center rounded-full bg-primary/10 text-primary">
								<Icon class="h-5 w-5" />
							</div>
							
							<!-- Content -->
							<div class="min-w-0 flex-1">
								<div class="mb-0.5 text-sm font-medium">{notification.title}</div>
								<p class="mb-1 text-sm text-muted-foreground line-clamp-2">
									{notification.content}
								</p>
								<span class="text-xs text-muted-foreground">
									{formatTimestamp(notification.timestamp)}
								</span>
							</div>
							
							<!-- Mark as read button -->
							<button
								type="button"
								class="absolute right-2 top-2 flex h-6 w-6 items-center justify-center rounded-full opacity-0 transition-all hover:bg-background group-hover:opacity-100"
								onclick={(e) => handleMarkAsRead(notification.id, e)}
								title="标记为已读"
							>
								<Check class="h-3.5 w-3.5" />
							</button>
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
