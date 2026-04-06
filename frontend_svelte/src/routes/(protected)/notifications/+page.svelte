<script lang="ts">
	import { onMount } from 'svelte';
	import { goto } from '$app/navigation';
	import { notificationStore } from '$lib/stores/notification.svelte';
	import { groupsService } from '$lib/api/services';
	import { toast } from '$lib/stores/toast.svelte';
	import { usePagination } from '$lib/hooks/use-pagination.svelte';
	import { getLocale, t } from '$lib/i18n';
	import type { NotificationItem, NotificationPayload } from '$lib/types/api';
	import { Button } from '$lib/components/ui/button';
	import { Card } from '$lib/components/ui/card';
	import PageHeader from '$lib/components/layout/page-header.svelte';
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

<div class="space-y-6">
	<PageHeader title={t('notificationsPage.title')}>
		{#snippet actions()}
			{#if notificationStore.unreadCount > 0}
				<Button variant="outline" size="sm" onclick={markAllRead}>
					<CheckCheck class="mr-2 h-4 w-4" />
					{t('notificationDropdown.markAllRead')}
				</Button>
			{/if}
		{/snippet}
	</PageHeader>

	{#if pagination.loading}
		<div class="flex justify-center py-12">
			<Loader2 class="h-8 w-8 animate-spin text-muted-foreground" />
		</div>
	{:else if pagination.error}
		<div class="space-y-4">
			<EmptyState icon={Bell} title={t('notificationsPage.loadFailed')} />
			<div class="flex justify-center">
				<Button variant="outline" onclick={() => loadNotifications(true)}>
					<Loader2 class="mr-2 h-4 w-4" />
					重试
				</Button>
			</div>
		</div>
	{:else if pagination.items.length === 0}
		<EmptyState icon={Bell} title={t('notificationsPage.emptyTitle')} />
	{:else}
		<div class="space-y-2">
			{#each pagination.items as item (item.notification_id)}
				{@const locale = getLocale()}
				{@const Icon = getNotificationIcon(item.type)}
				<Card class="overflow-hidden transition-colors hover:bg-accent/50">
					<div class="flex flex-col gap-3 p-4 sm:flex-row sm:items-start sm:gap-4">
						<Button
							type="button"
							variant="ghost"
							class="h-auto w-full justify-start rounded-2xl p-0 text-left hover:bg-transparent sm:flex-1"
							onclick={() => handleNotificationClick(item)}
						>
							<div class="flex w-full items-start gap-4">
								<div class="flex h-10 w-10 shrink-0 items-center justify-center rounded-2xl bg-muted">
									<Icon class="h-5 w-5" />
								</div>
								<div class="min-w-0 flex-1">
									<div class="flex items-start justify-between gap-2">
										<div>
											<span class="font-medium">{getNotificationTitle(item, locale)}</span>
											<span class="text-muted-foreground">
												{getNotificationContent(item, locale)}
											</span>
										</div>
										{#if !item.is_read}
											<div class="mt-1 h-2 w-2 shrink-0 rounded-full bg-primary"></div>
										{/if}
									</div>
									<p class="mt-1 text-xs text-muted-foreground">
										{formatNotificationTimestamp(item.created_at, locale)}
									</p>
								</div>
							</div>
						</Button>
						{#if canHandleGroupInvite(item)}
							<div class="flex flex-wrap gap-2 sm:mt-0 sm:shrink-0">
								<Button
									type="button"
									size="sm"
									disabled={isProcessing(item.notification_id)}
									onclick={() => handleInviteAction(item, 'accept')}
								>
									{t('common.accept')}
								</Button>
								<Button
									type="button"
									variant="outline"
									size="sm"
									disabled={isProcessing(item.notification_id)}
									onclick={() => handleInviteAction(item, 'reject')}
								>
									{t('common.reject')}
								</Button>
							</div>
						{:else if canHandleGroupApplication(item)}
							<div class="flex flex-wrap gap-2 sm:mt-0 sm:shrink-0">
								<Button
									type="button"
									size="sm"
									disabled={isProcessing(item.notification_id)}
									onclick={() => handleApplicationAction(item, 'approve')}
								>
									{t('common.approve')}
								</Button>
								<Button
									type="button"
									variant="outline"
									size="sm"
									disabled={isProcessing(item.notification_id)}
									onclick={() => handleApplicationAction(item, 'reject')}
								>
									{t('common.reject')}
								</Button>
							</div>
						{/if}
					</div>
				</Card>
			{/each}
		</div>
		{#if pagination.hasMore}
			<div class="flex justify-center pt-4">
				<Button
					variant="outline"
					disabled={pagination.loadingMore}
					onclick={() => loadNotifications(false)}
				>
					{#if pagination.loadingMore}
						<Loader2 class="mr-2 h-4 w-4 animate-spin" />
					{/if}
					{t('common.loadMore')}
				</Button>
			</div>
		{/if}
	{/if}
</div>
