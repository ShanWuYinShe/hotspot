<script lang="ts">
	import { onMount } from 'svelte';
	import { goto } from '$app/navigation';
	import { notificationStore } from '$lib/stores/notification.svelte';
	import { groupsService } from '$lib/api/services';
	import { toast } from '$lib/stores/toast.svelte';
	import { usePagination } from '$lib/hooks/use-pagination.svelte';
	import { getLocale, t } from '$lib/i18n';
	import type { NotificationItem, NotificationPayload } from '$lib/types/api';
	import EmptyState from '$lib/components/common/empty-state.svelte';
	import {
		formatNotificationTimestamp,
		getNotificationContent,
		getNotificationIcon,
		getNotificationLink,
		getNotificationTitle
	} from '$lib/utils/notification-display';
	import Bell from 'lucide-svelte/icons/bell';
	import CheckCheck from 'lucide-svelte/icons/check-check';
	import Loader2 from 'lucide-svelte/icons/loader-2';

	const pagination = usePagination<NotificationItem>({
		uniqueKey: (item) => item.notification_id,
		initialLoading: true
	});
	let processingNotifications = $state<Set<string>>(new Set());

	function loadNotifications(refresh = true) {
		return pagination.load(
			(cursor) =>
				notificationStore.fetchHistory({
					pagination: { cursor, limit: 20 }
				}),
			refresh
		);
	}

	onMount(() => {
		loadNotifications(true);
	});

	$effect(() => {
		const syncVersion = notificationStore.readSyncVersion;
		if (syncVersion === 0) return;
		if (notificationStore.lastReadMutationType === 'mark_all_read') {
			pagination.updateItem((i) => (i.is_read ? i : { ...i, is_read: true }));
			return;
		}
		if (notificationStore.lastReadMutationType === 'mark_read' && notificationStore.lastReadNotificationIds.length > 0) {
			const ids = new Set(notificationStore.lastReadNotificationIds);
			pagination.updateItem((i) =>
				ids.has(i.notification_id) ? { ...i, is_read: true } : null
			);
		}
	});

	async function markAllRead() {
		const ok = await notificationStore.markAllAsRead();
		if (ok) {
			pagination.updateItem((i) => (i.is_read ? i : { ...i, is_read: true }));
		}
	}

	function isProcessing(notificationId: string): boolean {
		return processingNotifications.has(notificationId);
	}

	function setProcessing(notificationId: string, processing: boolean) {
		const next = new Set(processingNotifications);
		if (processing) {
			next.add(notificationId);
		} else {
			next.delete(notificationId);
		}
		processingNotifications = next;
	}

	function canHandleGroupInvite(item: NotificationItem): boolean {
		return item.payload.type === 'group_invite_received' && !!item.payload.invite_id;
	}

	function canHandleGroupApplication(item: NotificationItem): boolean {
		return item.payload.type === 'group_application_received' && !!item.payload.application_id;
	}

	function getGroupInvitePayload(
		item: NotificationItem
	): Extract<NotificationPayload, { type: 'group_invite_received' }> | null {
		return item.payload.type === 'group_invite_received' ? item.payload : null;
	}

	function getGroupApplicationPayload(
		item: NotificationItem
	): Extract<NotificationPayload, { type: 'group_application_received' }> | null {
		return item.payload.type === 'group_application_received' ? item.payload : null;
	}

	async function markNotificationRead(item: NotificationItem) {
		if (item.is_read) return;
		const ok = await notificationStore.markAsRead([item.notification_id]);
		if (ok) {
			pagination.updateItem((i) =>
				i.notification_id === item.notification_id ? { ...i, is_read: true } : null
			);
		}
	}

	async function handleInviteAction(item: NotificationItem, action: 'accept' | 'reject') {
		if (!canHandleGroupInvite(item) || isProcessing(item.notification_id)) return;
		const payload = getGroupInvitePayload(item);
		const inviteId = payload?.invite_id;
		if (!inviteId) return;
		setProcessing(item.notification_id, true);
		try {
			const response =
				action === 'accept'
					? await groupsService.acceptInvite({ invite_id: inviteId })
					: await groupsService.rejectInvite({ invite_id: inviteId });
			if (response.code !== 0) {
				toast.error(
					action === 'accept' ? t('notificationsPage.inviteAcceptFailed') : t('notificationsPage.inviteRejectFailed')
				);
				return;
			}
			await markNotificationRead(item);
			toast.success(
				action === 'accept' ? t('notificationsPage.inviteAccepted') : t('notificationsPage.inviteRejected')
			);
			if (action === 'accept' && payload?.group_id) {
				goto(`/groups/${payload.group_id}`);
			}
		} catch {
			toast.error(
				action === 'accept' ? t('notificationsPage.inviteAcceptFailed') : t('notificationsPage.inviteRejectFailed')
			);
		} finally {
			setProcessing(item.notification_id, false);
		}
	}

	async function handleApplicationAction(item: NotificationItem, action: 'approve' | 'reject') {
		if (!canHandleGroupApplication(item) || isProcessing(item.notification_id)) return;
		const payload = getGroupApplicationPayload(item);
		const applicationId = payload?.application_id;
		if (!applicationId) return;
		setProcessing(item.notification_id, true);
		try {
			const response =
				action === 'approve'
					? await groupsService.approveApplication({ application_id: applicationId })
					: await groupsService.rejectApplication({ application_id: applicationId });
			if (response.code !== 0) {
				toast.error(
					action === 'approve'
						? t('notificationsPage.applicationApproveFailed')
						: t('notificationsPage.applicationRejectFailed')
				);
				return;
			}
			await markNotificationRead(item);
			toast.success(
				action === 'approve'
					? t('notificationsPage.applicationApproved')
					: t('notificationsPage.applicationRejected')
			);
		} catch {
			toast.error(
				action === 'approve'
					? t('notificationsPage.applicationApproveFailed')
					: t('notificationsPage.applicationRejectFailed')
			);
		} finally {
			setProcessing(item.notification_id, false);
		}
	}


	async function handleNotificationClick(item: NotificationItem) {
		if (!item.is_read) {
			const ok = await notificationStore.markAsRead([item.notification_id]);
			if (ok) {
				pagination.updateItem((i) =>
					i.notification_id === item.notification_id ? { ...i, is_read: true } : null
				);
			}
		}
		const target = getNotificationLink(item);
		if (target) {
			goto(target);
		}
	}
</script>

<svelte:head>
	<title>{t('notificationsPage.title')} - HotSpot</title>
</svelte:head>

<div class="flex flex-col">
	<!-- Sticky header -->
	<div
		class="sticky top-0 z-10 px-5 pt-3 pb-3"
		style="background-color: var(--neu-bg); border-bottom: 1px solid var(--neu-border);"
	>
		<div class="flex items-center justify-between">
			<div>
				<h1 style="font-size: 20px; font-weight: 600; color: var(--neu-text);">{t('notificationsPage.title')}</h1>
				{#if notificationStore.unreadCount > 0}
					<p style="font-size: 12px; color: var(--neu-text-muted); margin-top: 2px;">{notificationStore.unreadCount} 条未读</p>
				{/if}
			</div>
			{#if notificationStore.unreadCount > 0}
				<button
					class="flex items-center gap-1.5 rounded-full px-3 py-1.5 outline-none transition-transform duration-150 active:scale-[0.93]"
					style="background: var(--neu-bg); border: 1px solid var(--neu-border); box-shadow: var(--neu-shadow-md); font-size: 12px; color: var(--neu-accent-primary);"
					onclick={markAllRead}
				>
					<CheckCheck size={13} strokeWidth={2} />
					{t('notificationDropdown.markAllRead')}
				</button>
			{/if}
		</div>
	</div>

	<div class="flex flex-col px-3 pt-2 pb-4">
		{#if pagination.loading}
			<div class="flex justify-center py-12">
				<Loader2 class="h-8 w-8 animate-spin" color="var(--neu-text-muted)" />
			</div>
		{:else if pagination.error}
			<EmptyState icon={Bell} title={t('notificationsPage.loadFailed')} />
			<div class="flex justify-center pt-2">
				<button
					class="rounded-full px-5 py-2 outline-none"
					style="background: var(--neu-bg); box-shadow: var(--neu-shadow-md); border: 1px solid var(--neu-border); color: var(--neu-accent-primary); font-size: 13px;"
					onclick={() => loadNotifications(true)}
				>{t('common.retry')}</button>
			</div>
		{:else if pagination.items.length === 0}
			<EmptyState icon={Bell} title={t('notificationsPage.emptyTitle')} />
		{:else}
			<div class="space-y-2">
				{#each pagination.items as item (item.notification_id)}
					{@const locale = getLocale()}
					{@const Icon = getNotificationIcon(item.type)}
					<div
						class="flex items-start gap-3 rounded-[18px] px-4 py-3 transition-transform duration-150 active:scale-[0.98]"
						style="background: var(--neu-bg); border: 1px solid var(--neu-border); {item.is_read ? '' : 'box-shadow: var(--neu-shadow-lg);'}"
					>
						<div class="flex flex-shrink-0 items-center justify-center rounded-full" style="width: 44px; height: 44px; box-shadow: var(--neu-shadow-inset); border: 1px solid var(--neu-border);">
							<Icon size={15} strokeWidth={1.5} />
						</div>
						<div class="min-w-0 flex-1">
							<button class="w-full text-left outline-none" onclick={() => handleNotificationClick(item)}>
								<div class="flex items-start justify-between gap-2">
									<div>
										<span class="mr-1.5" style="font-size: 13px; color: var(--neu-text); font-weight: {item.is_read ? 400 : 500};">{getNotificationTitle(item, locale)}</span>
										<span style="font-size: 13px; color: var(--neu-text-muted);">{getNotificationContent(item, locale)}</span>
									</div>
									{#if !item.is_read}
										<div class="mt-1.5 h-2 w-2 flex-shrink-0 rounded-full" style="background: var(--neu-accent-primary);"></div>
									{/if}
								</div>
								<p style="font-size: 11px; color: var(--neu-text-muted); margin-top: 2px;">{formatNotificationTimestamp(item.created_at, locale)}</p>
							</button>
							{#if canHandleGroupInvite(item)}
								<div class="mt-2 flex gap-2">
									<button class="rounded-full px-3 py-1 text-xs outline-none transition-transform duration-150 active:scale-[0.93]" style="background: var(--neu-accent-primary); color: var(--neu-active-text); box-shadow: 0 2px 6px var(--neu-glow);" disabled={isProcessing(item.notification_id)} onclick={() => handleInviteAction(item, 'accept')}>{t('common.accept')}</button>
									<button class="rounded-full px-3 py-1 text-xs outline-none transition-transform duration-150 active:scale-[0.93]" style="box-shadow: var(--neu-shadow-md); border: 1px solid var(--neu-border); color: var(--neu-text-muted); background: var(--neu-bg);" disabled={isProcessing(item.notification_id)} onclick={() => handleInviteAction(item, 'reject')}>{t('common.reject')}</button>
								</div>
							{:else if canHandleGroupApplication(item)}
								<div class="mt-2 flex gap-2">
									<button class="rounded-full px-3 py-1 text-xs outline-none transition-transform duration-150 active:scale-[0.93]" style="background: var(--neu-accent-primary); color: var(--neu-active-text); box-shadow: 0 2px 6px var(--neu-glow);" disabled={isProcessing(item.notification_id)} onclick={() => handleApplicationAction(item, 'approve')}>{t('common.approve')}</button>
									<button class="rounded-full px-3 py-1 text-xs outline-none transition-transform duration-150 active:scale-[0.93]" style="box-shadow: var(--neu-shadow-md); border: 1px solid var(--neu-border); color: var(--neu-text-muted); background: var(--neu-bg);" disabled={isProcessing(item.notification_id)} onclick={() => handleApplicationAction(item, 'reject')}>{t('common.reject')}</button>
								</div>
							{/if}
						</div>
					</div>
				{/each}
			</div>
			{#if pagination.hasMore}
				<div class="flex justify-center pt-4">
					<button class="rounded-full px-5 py-2 outline-none transition-transform duration-150 active:scale-[0.97]" style="background: var(--neu-bg); box-shadow: var(--neu-shadow-md); border: 1px solid var(--neu-border); color: var(--neu-text-muted); font-size: 13px;" disabled={pagination.loadingMore} onclick={() => loadNotifications(false)}>
						{#if pagination.loadingMore}<Loader2 class="mr-2 inline h-4 w-4 animate-spin" />{/if}
						{t('common.loadMore')}
					</button>
				</div>
			{/if}
		{/if}
	</div>
</div>
