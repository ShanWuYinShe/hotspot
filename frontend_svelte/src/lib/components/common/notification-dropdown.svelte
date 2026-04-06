<script lang="ts">
	import { goto } from '$app/navigation';
	import { untrack } from 'svelte';
	import { notificationStore } from '$lib/stores/notification.svelte';
	import type { NotificationItem } from '$lib/types/api';
	import * as DropdownMenu from '$lib/components/ui/dropdown-menu';
	import { Button } from '$lib/components/ui/button';
	import { Badge } from '$lib/components/ui/badge';
	import { ScrollArea } from '$lib/components/ui/scroll-area';
	import { getLocale, t } from '$lib/i18n';
	import {
		formatNotificationTimestamp,
		getNotificationBadgeIcon,
		getNotificationContent,
		getNotificationIcon,
		getNotificationLink,
		getNotificationTitle
	} from '$lib/utils/notification-display';
	import Bell from 'lucide-svelte/icons/bell';
	import Check from 'lucide-svelte/icons/check';
	import CheckCheck from 'lucide-svelte/icons/check-check';
	import Settings from 'lucide-svelte/icons/settings';
	import { isSuccess } from '$lib/api/client';

	let open = $state(false);
	let recentNotifications = $state.raw<NotificationItem[]>([]);
	let loading = $state(false);
	let loadVersion = 0;

	$effect(() => {
		if (open) {
			const version = ++loadVersion;
			untrack(() => loadRecent(version));
		}
	});

	async function loadRecent(version: number) {
		loading = true;
		try {
			const res = await notificationStore.fetchHistory({ pagination: { limit: 5 } });
			if (version !== loadVersion) return; // 丢弃过期响应
			if (isSuccess(res) && res.data) {
				recentNotifications = res.data.items;
			} else {
				recentNotifications = [];
			}
		} catch {
			if (version !== loadVersion) return;
			recentNotifications = [];
		} finally {
			if (version === loadVersion) {
				loading = false;
			}
		}
	}

	const hasMore = $derived(notificationStore.unreadCount > 5);

	async function handleMarkAllRead() {
		const ok = await notificationStore.markAllAsRead();
		if (ok) {
			recentNotifications = recentNotifications.map((i) => (i.is_read ? i : { ...i, is_read: true }));
		}
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
			const ok = await notificationStore.markAsRead([item.notification_id]);
			if (ok) {
				recentNotifications = recentNotifications.map((i) =>
					i.notification_id === item.notification_id ? { ...i, is_read: true } : i
				);
			}
		}
		const target = getNotificationLink(item);
		open = false;
		if (target) {
			goto(target);
		}
	}

	function handleViewAll() {
		open = false;
		goto('/notifications');
	}
</script>

<DropdownMenu.Root bind:open>
	<DropdownMenu.Trigger>
		{#snippet child({ props })}
			<Button variant="ghost" size="icon" class="relative rounded-2xl" aria-label={t('notificationDropdown.triggerAria')} {...props}>
				{@const TriggerIcon = getNotificationBadgeIcon(notificationStore.unreadCount)}
				{#if notificationStore.unreadCount > 0}
					<TriggerIcon class="h-5 w-5" />
					<Badge
						variant="destructive"
						class="absolute -right-1 -top-1 flex h-5 min-w-5 items-center justify-center rounded-xl px-1 text-[10px] font-bold ring-2 ring-background"
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
				<h3 class="text-sm font-semibold">{t('notificationDropdown.title')}</h3>
				{#if notificationStore.unreadCount > 0}
					<Badge variant="secondary" class="h-5 rounded-xl px-2 text-[10px]">
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
						title={t('notificationDropdown.markAllRead')}
						aria-label={t('notificationDropdown.markAllReadAria')}
					>
						<CheckCheck class="h-4 w-4" />
					</Button>
				{/if}
				<Button
					variant="ghost"
					size="icon"
					class="h-8 w-8"
					onclick={() => { open = false; goto('/settings/notifications'); }}
					title={t('notificationDropdown.settings')}
					aria-label={t('notificationDropdown.settings')}
				>
					<Settings class="h-4 w-4" />
				</Button>
			</div>
		</div>

		<!-- Notifications List -->
		{#if loading}
			<div class="flex flex-col items-center justify-center py-12">
				<div class="h-8 w-8 animate-pulse rounded-full bg-muted"></div>
				<p class="mt-3 text-sm text-muted-foreground">{t('common.loading')}</p>
			</div>
		{:else if recentNotifications.length === 0}
			<div class="flex flex-col items-center justify-center py-12">
				<div class="mb-3 flex h-16 w-16 items-center justify-center rounded-2xl bg-muted">
					<Bell class="h-8 w-8 text-muted-foreground" />
				</div>
				<p class="text-sm font-medium">{t('notificationDropdown.emptyTitle')}</p>
				<p class="text-xs text-muted-foreground">{t('notificationDropdown.emptyDescription')}</p>
			</div>
		{:else}
			<ScrollArea class="max-h-96">
				<div class="divide-y divide-border/50">
					{#each recentNotifications as item (item.notification_id)}
						{@const locale = getLocale()}
						{@const Icon = getNotificationIcon(item.type)}
						<div class="group relative flex gap-3 p-4 transition-colors hover:bg-accent/50">
							<button
								type="button"
								class="absolute inset-0 flex items-start gap-3 text-left"
								onclick={() => handleItemClick(item)}
							>
								<div class="flex h-10 w-10 shrink-0 items-center justify-center rounded-2xl bg-primary/10 text-primary">
									<Icon class="h-5 w-5" />
								</div>
								<div class="min-w-0 flex-1 pr-8">
									<div class="mb-0.5 text-sm font-medium">{getNotificationTitle(item, locale)}</div>
									<p class="mb-1 text-sm text-muted-foreground line-clamp-2">
										{getNotificationContent(item, locale)}
									</p>
									<span class="text-xs text-muted-foreground">
										{formatNotificationTimestamp(item.created_at, locale)}
									</span>
								</div>
							</button>
							{#if !item.is_read}
								<button
									type="button"
									class="absolute right-2 top-2 z-10 flex h-6 w-6 items-center justify-center rounded-xl opacity-0 transition-all hover:bg-background group-hover:opacity-100"
									onclick={(e) => handleMarkAsRead(item.notification_id, e)}
									title={t('notificationDropdown.markAsRead')}
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
					{hasMore
						? t('notificationDropdown.viewAllWithCount', { count: notificationStore.unreadCount })
						: t('notificationDropdown.viewAll')}
				</Button>
			</div>
		{/if}
	</DropdownMenu.Content>
</DropdownMenu.Root>
