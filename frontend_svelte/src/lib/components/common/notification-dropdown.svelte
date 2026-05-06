<script lang="ts">
	import { goto } from '$app/navigation';
	import { untrack } from 'svelte';
	import { notificationStore } from '$lib/stores/notification.svelte';
	import type { NotificationItem } from '$lib/types/api';
	import * as DropdownMenu from '$lib/components/ui/dropdown-menu';
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
	let loadFailed = $state(false);
	let loadVersion = 0;

	$effect(() => {
		if (open) {
			const version = ++loadVersion;
			untrack(() => loadRecent(version));
		}
	});

	$effect(() => {
		const syncVersion = notificationStore.readSyncVersion;
		if (!open || syncVersion === 0) return;
		if (notificationStore.lastReadMutationType === 'mark_all_read') {
			recentNotifications = recentNotifications.map((i) => (i.is_read ? i : { ...i, is_read: true }));
			return;
		}
		if (notificationStore.lastReadMutationType === 'mark_read' && notificationStore.lastReadNotificationIds.length > 0) {
			const ids = new Set(notificationStore.lastReadNotificationIds);
			recentNotifications = recentNotifications.map((i) =>
				ids.has(i.notification_id) ? { ...i, is_read: true } : i
			);
		}
	});

	async function loadRecent(version: number) {
		loading = true;
		loadFailed = false;
		try {
			const res = await notificationStore.fetchHistory({ pagination: { limit: 5 } });
			if (version !== loadVersion) return;
			if (isSuccess(res) && res.data) {
				recentNotifications = res.data.items;
				loadFailed = false;
			} else {
				recentNotifications = [];
				loadFailed = true;
			}
		} catch {
			if (version !== loadVersion) return;
			recentNotifications = [];
			loadFailed = true;
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

	function retryLoadRecent() {
		const version = ++loadVersion;
		untrack(() => loadRecent(version));
	}
</script>

<DropdownMenu.Root bind:open>
	<DropdownMenu.Trigger>
		{#snippet child({ props })}
			<button class="relative flex items-center justify-center rounded-full outline-none" style="width: 40px; height: 40px; background: var(--neu-bg); box-shadow: var(--neu-shadow-md); border: 1px solid var(--neu-border);" aria-label={t('notificationDropdown.triggerAria')} {...props}>
				{#if notificationStore.unreadCount > 0}
					{@const TriggerIcon = getNotificationBadgeIcon(notificationStore.unreadCount)}
					<TriggerIcon style="width: 20px; height: 20px; color: var(--neu-text);" />
					<span class="absolute flex items-center justify-center rounded-full" style="right: -4px; top: -4px; height: 20px; min-width: 20px; padding: 0 4px; background: var(--neu-error); color: var(--neu-active-text); font-size: 10px; font-weight: 700; border: 2px solid var(--neu-bg);">
						{notificationStore.unreadCount > 99 ? '99+' : notificationStore.unreadCount}
					</span>
				{:else}
					<Bell style="width: 20px; height: 20px; color: var(--neu-text);" />
				{/if}
			</button>
		{/snippet}
	</DropdownMenu.Trigger>

	<DropdownMenu.Content class="w-[min(24rem,calc(100vw-2rem))] p-0" align="end" sideOffset={8}>
		<!-- Header -->
		<div class="flex items-center justify-between px-4 py-3" style="border-bottom: 1px solid var(--neu-border);">
			<div class="flex items-center gap-2">
				<h3 style="font-size: 14px; font-weight: 600; color: var(--neu-text);">{t('notificationDropdown.title')}</h3>
				{#if notificationStore.unreadCount > 0}
					<span class="rounded-full" style="height: 20px; padding: 0 8px; font-size: 10px; line-height: 20px; background: var(--neu-accent-primary); color: var(--neu-active-text);">
						{notificationStore.unreadCount}
					</span>
				{/if}
			</div>
			<div class="flex items-center gap-1">
				{#if notificationStore.unreadCount > 0}
					<button
						class="flex items-center justify-center rounded-full outline-none"
						style="width: 32px; height: 32px; background: var(--neu-bg); box-shadow: var(--neu-shadow-md); border: 1px solid var(--neu-border);"
						onclick={handleMarkAllRead}
						title={t('notificationDropdown.markAllRead')}
						aria-label={t('notificationDropdown.markAllReadAria')}
					>
						<CheckCheck style="width: 16px; height: 16px; color: var(--neu-text);" />
					</button>
				{/if}
				<button
					class="flex items-center justify-center rounded-full outline-none"
					style="width: 32px; height: 32px; background: var(--neu-bg); box-shadow: var(--neu-shadow-md); border: 1px solid var(--neu-border);"
					onclick={() => { open = false; goto('/settings/notifications'); }}
					title={t('notificationDropdown.settings')}
					aria-label={t('notificationDropdown.settings')}
				>
					<Settings style="width: 16px; height: 16px; color: var(--neu-text);" />
				</button>
			</div>
		</div>

		<!-- Notifications List -->
		{#if loading}
			<div class="flex flex-col items-center justify-center py-12">
				<div class="rounded-full animate-pulse" style="width: 32px; height: 32px; background: var(--neu-bg); box-shadow: var(--neu-shadow-inset);"></div>
				<p class="mt-3" style="font-size: 13px; color: var(--neu-text-muted);">{t('common.loading')}</p>
			</div>
		{:else if loadFailed}
			<div class="flex flex-col items-center justify-center py-12 px-4 text-center">
				<div class="mb-3 flex items-center justify-center rounded-full" style="width: 64px; height: 64px; box-shadow: var(--neu-shadow-inset); border: 1px solid var(--neu-border);">
					<Bell style="width: 32px; height: 32px; color: var(--neu-text-muted);" />
				</div>
				<p style="font-size: 14px; font-weight: 500; color: var(--neu-text);">加载通知失败</p>
				<p style="font-size: 12px; color: var(--neu-text-muted);">请稍后重试</p>
				<button class="mt-4 rounded-full px-5 py-2 outline-none" style="background: var(--neu-bg); color: var(--neu-text); box-shadow: var(--neu-shadow-md); border: 1px solid var(--neu-border); font-size: 13px;" onclick={retryLoadRecent}>
					重试
				</button>
			</div>
		{:else if recentNotifications.length === 0}
			<div class="flex flex-col items-center justify-center py-12">
				<div class="mb-3 flex items-center justify-center rounded-full" style="width: 64px; height: 64px; box-shadow: var(--neu-shadow-inset); border: 1px solid var(--neu-border);">
					<Bell style="width: 32px; height: 32px; color: var(--neu-text-muted);" />
				</div>
				<p style="font-size: 14px; font-weight: 500; color: var(--neu-text);">{t('notificationDropdown.emptyTitle')}</p>
				<p style="font-size: 12px; color: var(--neu-text-muted);">{t('notificationDropdown.emptyDescription')}</p>
			</div>
		{:else}
			<div class="overflow-y-auto" style="max-height: 384px;">
				<div>
					{#each recentNotifications as item (item.notification_id)}
						{@const locale = getLocale()}
						{@const Icon = getNotificationIcon(item.type)}
						<div class="group relative flex gap-3 p-4 transition-colors" style="border-bottom: 1px solid var(--neu-border);">
							<button
								type="button"
								class="absolute inset-0 flex items-start gap-3 text-left"
								onclick={() => handleItemClick(item)}
							>
								<div class="flex shrink-0 items-center justify-center rounded-full" style="width: 40px; height: 40px; box-shadow: var(--neu-shadow-inset); border: 1px solid var(--neu-border);">
									<Icon style="width: 20px; height: 20px; color: var(--neu-accent-primary);" />
								</div>
								<div class="min-w-0 flex-1 pr-8">
									<div class="mb-0.5" style="font-size: 14px; font-weight: 500; color: var(--neu-text);">{getNotificationTitle(item, locale)}</div>
									<p class="mb-1" style="font-size: 14px; color: var(--neu-text-muted); display: -webkit-box; -webkit-line-clamp: 2; -webkit-box-orient: vertical; overflow: hidden;">
										{getNotificationContent(item, locale)}
									</p>
									<span style="font-size: 12px; color: var(--neu-text-muted);">
										{formatNotificationTimestamp(item.created_at, locale)}
									</span>
								</div>
							</button>
							{#if !item.is_read}
								<button
									type="button"
									class="absolute right-2 top-2 z-10 flex items-center justify-center rounded-full opacity-0 transition-all group-hover:opacity-100"
									style="width: 24px; height: 24px; background: var(--neu-bg); box-shadow: var(--neu-shadow-md); border: 1px solid var(--neu-border);"
									onclick={(e) => handleMarkAsRead(item.notification_id, e)}
									title={t('notificationDropdown.markAsRead')}
								>
									<Check style="width: 14px; height: 14px; color: var(--neu-text);" />
								</button>
							{/if}
						</div>
					{/each}
				</div>
			</div>

			<!-- Footer -->
			<div class="p-2" style="border-top: 1px solid var(--neu-border);">
				<button
					class="w-full flex items-center justify-center rounded-full px-5 py-2 outline-none"
					style="background: var(--neu-bg); color: var(--neu-text); box-shadow: var(--neu-shadow-md); border: 1px solid var(--neu-border); font-size: 13px;"
					onclick={handleViewAll}
				>
					{hasMore
						? t('notificationDropdown.viewAllWithCount', { count: notificationStore.unreadCount })
						: t('notificationDropdown.viewAll')}
				</button>
			</div>
		{/if}
	</DropdownMenu.Content>
</DropdownMenu.Root>
