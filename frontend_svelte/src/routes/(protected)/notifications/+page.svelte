<script lang="ts">
  import { onMount } from 'svelte';
  import { goto } from '$app/navigation';
  import { notificationStore } from '$lib/stores/notification.svelte';
  import { groupsService } from '$lib/api/services';
  import { toast } from '$lib/stores/toast.svelte';
  import { usePagination } from '$lib/hooks/use-pagination.svelte';
  import { getLocale, t } from '$lib/i18n';
  import type { NotificationItem, NotificationPayload, NotificationType } from '$lib/types/api';
  import EmptyState from '$lib/components/common/empty-state.svelte';
  import {
    formatNotificationTimestamp,
    getNotificationContent,
    getNotificationLink,
    getNotificationTitle
  } from '$lib/utils/notification-display';
  import ArrowLeft from 'lucide-svelte/icons/arrow-left';
  import Bell from 'lucide-svelte/icons/bell';
  import Calendar from 'lucide-svelte/icons/calendar';
  import CheckCheck from 'lucide-svelte/icons/check-check';
  import Heart from 'lucide-svelte/icons/heart';
  import Loader2 from 'lucide-svelte/icons/loader-2';
  import MessageCircle from 'lucide-svelte/icons/message-circle';
  import UserPlus from 'lucide-svelte/icons/user-plus';
  import Users from 'lucide-svelte/icons/users';

  type FilterKey = 'all' | 'like' | 'follow' | 'comment' | 'nearby';

  const filterLabels: Record<FilterKey, string> = {
    all: '全部',
    like: '点赞',
    follow: '关注',
    comment: '评论',
    nearby: '附近'
  };

  const filterToTypes: Record<FilterKey, NotificationType[] | null> = {
    all: null,
    like: ['moment_liked', 'comment_liked'],
    follow: ['new_follower', 'mutual_follow'],
    comment: ['moment_commented', 'comment_reply', 'mentioned'],
    nearby: []
  };

  let activeFilter = $state<FilterKey>('all');

  function getTypeColor(type: NotificationType): string {
    switch (type) {
      case 'moment_liked':
      case 'comment_liked':
        return '#EF4444';
      case 'new_follower':
      case 'mutual_follow':
        return '#6366F1';
      case 'moment_commented':
      case 'comment_reply':
      case 'mentioned':
        return '#3B82F6';
      case 'group_invite_received':
      case 'group_invite_accepted':
      case 'group_application_received':
      case 'group_application_accepted':
        return '#8B5CF6';
      case 'system_announcement':
        return '#F59E0B';
      default:
        return 'var(--neu-accent-primary)';
    }
  }

  function getTypeIcon(type: NotificationType) {
    switch (type) {
      case 'moment_liked':
      case 'comment_liked':
        return Heart;
      case 'new_follower':
      case 'mutual_follow':
        return UserPlus;
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
        return Calendar;
      default:
        return Bell;
    }
  }

  function getActorAvatar(item: NotificationItem): string | undefined {
    const p = item.payload;
    if ('actor' in p) return p.actor.avatar_url;
    if ('applicant' in p) return p.applicant.avatar_url;
    if ('user' in p) return p.user.avatar_url;
    return undefined;
  }

  function getPostThumbnail(item: NotificationItem): string | undefined {
    const p = item.payload;
    if ('moment_summary' in p && p.moment_summary?.cover_url) {
      return p.moment_summary.cover_url;
    }
    return undefined;
  }

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
    if (
      notificationStore.lastReadMutationType === 'mark_read' &&
      notificationStore.lastReadNotificationIds.length > 0
    ) {
      const ids = new Set(notificationStore.lastReadNotificationIds);
      pagination.updateItem((i) => (ids.has(i.notification_id) ? { ...i, is_read: true } : null));
    }
  });

  let filteredItems = $derived.by(() => {
    if (activeFilter === 'all') return pagination.items;
    const types = filterToTypes[activeFilter];
    if (!types) return pagination.items;
    const typeSet = new Set(types);
    return pagination.items.filter((i) => typeSet.has(i.type));
  });

  let unreadGroup = $derived(filteredItems.filter((i) => !i.is_read));
  let readGroup = $derived(filteredItems.filter((i) => i.is_read));

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
          action === 'accept'
            ? t('notificationsPage.inviteAcceptFailed')
            : t('notificationsPage.inviteRejectFailed')
        );
        return;
      }
      await markNotificationRead(item);
      toast.success(
        action === 'accept'
          ? t('notificationsPage.inviteAccepted')
          : t('notificationsPage.inviteRejected')
      );
      if (action === 'accept' && payload?.group_id) {
        goto(`/groups/${payload.group_id}`);
      }
    } catch {
      toast.error(
        action === 'accept'
          ? t('notificationsPage.inviteAcceptFailed')
          : t('notificationsPage.inviteRejectFailed')
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
      <div class="flex items-center gap-3">
        <button
          class="flex items-center justify-center outline-none transition-transform duration-150 active:scale-[0.93]"
          style="width: 38px; height: 38px; border-radius: 9999px; box-shadow: 5px 5px 10px var(--neu-shadow-dark), -5px -5px 10px var(--neu-shadow-light); border: 1px solid var(--neu-border); background: var(--neu-bg);"
          onclick={() => window.history.back()}
        >
          <ArrowLeft size={16} strokeWidth={2} color="var(--neu-text)" />
        </button>
        <div>
          <h1 style="font-size: 18px; font-weight: 600; color: var(--neu-text);">
            {t('notificationsPage.title')}
          </h1>
          {#if notificationStore.unreadCount > 0}
            <p style="font-size: 12px; color: var(--neu-text-muted); margin-top: 1px;">
              {notificationStore.unreadCount} 条未读
            </p>
          {/if}
        </div>
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

  <!-- Filter pills -->
  <div class="flex gap-2 overflow-x-auto px-5 py-3" style="scrollbar-width: none;">
    {#each Object.entries(filterLabels) as [key, label]}
      <button
        class="flex-shrink-0 outline-none transition-all duration-200"
        style="padding: 6px 14px; border-radius: 9999px; font-size: 12px; font-weight: 500;
          {activeFilter === key
          ? 'background: var(--neu-accent-primary); color: white; box-shadow: 0 3px 10px var(--neu-glow); border: none;'
          : 'background: var(--neu-bg); color: var(--neu-text); box-shadow: 3px 3px 6px var(--neu-shadow-dark), -3px -3px 6px var(--neu-shadow-light); border: 1px solid var(--neu-border);'}"
        onclick={() => (activeFilter = key as FilterKey)}
      >
        {label}
      </button>
    {/each}
  </div>

  <div class="flex flex-col px-3 pb-4">
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
          onclick={() => loadNotifications(true)}>{t('common.retry')}</button
        >
      </div>
    {:else if filteredItems.length === 0}
      <EmptyState icon={Bell} title={t('notificationsPage.emptyTitle')} />
    {:else}
      <div class="space-y-2.5">
        <!-- Unread group -->
        {#if unreadGroup.length > 0}
          <p
            class="uppercase mb-2"
            style="font-size: 10px; letter-spacing: 0.14em; color: var(--neu-text-muted);"
          >
            新消息
          </p>
          {#each unreadGroup as item (item.notification_id)}
            {@const locale = getLocale()}
            {@const typeColor = getTypeColor(item.type)}
            {@const TypeIcon = getTypeIcon(item.type)}
            {@const avatarUrl = getActorAvatar(item)}
            {@const thumbnailUrl = getPostThumbnail(item)}
            <div
              class="flex items-start gap-3.5 rounded-[18px] px-3 py-3.5 transition-transform duration-150 active:scale-[0.98]"
              style="background: {`${typeColor}07`};
                border: 1px solid {`${typeColor}18`};
                box-shadow: var(--neu-shadow-lg);"
            >
              <!-- Avatar with type badge -->
              <div class="relative flex-shrink-0" style="width: 44px; height: 44px;">
                {#if avatarUrl}
                  <img
                    src={avatarUrl}
                    alt=""
                    style="width: 44px; height: 44px; border-radius: 9999px; object-fit: cover; border: 1px solid var(--neu-border);"
                  />
                {:else}
                  <div
                    class="flex items-center justify-center"
                    style="width: 44px; height: 44px; border-radius: 9999px; background: var(--neu-bg); box-shadow: var(--neu-shadow-inset); border: 1px solid var(--neu-border);"
                  >
                    <Bell size={16} strokeWidth={1.5} color="var(--neu-text-muted)" />
                  </div>
                {/if}
                <div
                  class="absolute flex items-center justify-center"
                  style="width: 18px; height: 18px; border-radius: 9999px; background: {typeColor}; border: 2px solid var(--neu-bg); bottom: -2px; right: -2px;"
                >
                  <TypeIcon size={10} strokeWidth={2} color="white" />
                </div>
              </div>

              <div class="min-w-0 flex-1">
                <button
                  class="w-full text-left outline-none"
                  onclick={() => handleNotificationClick(item)}
                >
                  <div
                    style="font-size: 13px; color: var(--neu-text); font-weight: 500; line-height: 1.4;"
                  >
                    <span class="mr-1">{getNotificationTitle(item, locale)}</span>
                    <span style="color: var(--neu-text-muted);"
                      >{getNotificationContent(item, locale)}</span
                    >
                  </div>
                  <p style="font-size: 11px; color: var(--neu-text-muted); margin-top: 2px;">
                    {formatNotificationTimestamp(item.created_at, locale)}
                  </p>
                </button>
                {#if canHandleGroupInvite(item)}
                  <div class="mt-2 flex gap-2">
                    <button
                      class="rounded-full px-3 py-1 text-xs outline-none transition-transform duration-150 active:scale-[0.93]"
                      style="background: var(--neu-accent-primary); color: var(--neu-active-text); box-shadow: 0 2px 6px var(--neu-glow);"
                      disabled={isProcessing(item.notification_id)}
                      onclick={() => handleInviteAction(item, 'accept')}
                      >{t('common.accept')}</button
                    >
                    <button
                      class="rounded-full px-3 py-1 text-xs outline-none transition-transform duration-150 active:scale-[0.93]"
                      style="box-shadow: var(--neu-shadow-md); border: 1px solid var(--neu-border); color: var(--neu-text-muted); background: var(--neu-bg);"
                      disabled={isProcessing(item.notification_id)}
                      onclick={() => handleInviteAction(item, 'reject')}
                      >{t('common.reject')}</button
                    >
                  </div>
                {:else if canHandleGroupApplication(item)}
                  <div class="mt-2 flex gap-2">
                    <button
                      class="rounded-full px-3 py-1 text-xs outline-none transition-transform duration-150 active:scale-[0.93]"
                      style="background: var(--neu-accent-primary); color: var(--neu-active-text); box-shadow: 0 2px 6px var(--neu-glow);"
                      disabled={isProcessing(item.notification_id)}
                      onclick={() => handleApplicationAction(item, 'approve')}
                      >{t('common.approve')}</button
                    >
                    <button
                      class="rounded-full px-3 py-1 text-xs outline-none transition-transform duration-150 active:scale-[0.93]"
                      style="box-shadow: var(--neu-shadow-md); border: 1px solid var(--neu-border); color: var(--neu-text-muted); background: var(--neu-bg);"
                      disabled={isProcessing(item.notification_id)}
                      onclick={() => handleApplicationAction(item, 'reject')}
                      >{t('common.reject')}</button
                    >
                  </div>
                {/if}
              </div>

              <div class="flex flex-shrink-0 flex-col items-center gap-1.5 pt-1">
                {#if thumbnailUrl}
                  <img
                    src={thumbnailUrl}
                    alt=""
                    style="width: 38px; height: 38px; border-radius: 10px; object-fit: cover; border: 1px solid var(--neu-border);"
                  />
                {/if}
                <div
                  style="width: 8px; height: 8px; border-radius: 9999px; background: var(--neu-accent-primary);"
                ></div>
              </div>
            </div>
          {/each}
          {#if readGroup.length > 0}
            <p
              class="uppercase mt-4 mb-2"
              style="font-size: 10px; letter-spacing: 0.14em; color: var(--neu-text-muted);"
            >
              较早
            </p>
          {/if}
        {/if}
        <!-- Read group -->
        {#each readGroup as item (item.notification_id)}
          {@const locale = getLocale()}
          {@const typeColor = getTypeColor(item.type)}
          {@const TypeIcon = getTypeIcon(item.type)}
          {@const avatarUrl = getActorAvatar(item)}
          {@const thumbnailUrl = getPostThumbnail(item)}
          <div
            class="flex items-start gap-3.5 rounded-[18px] px-3 py-3.5 transition-transform duration-150 active:scale-[0.98]"
            style="background: var(--neu-bg); border: 1px solid var(--neu-border);"
          >
            <div class="relative flex-shrink-0" style="width: 44px; height: 44px;">
              {#if avatarUrl}
                <img
                  src={avatarUrl}
                  alt=""
                  style="width: 44px; height: 44px; border-radius: 9999px; object-fit: cover; border: 1px solid var(--neu-border);"
                />
              {:else}
                <div
                  class="flex items-center justify-center"
                  style="width: 44px; height: 44px; border-radius: 9999px; background: var(--neu-bg); box-shadow: var(--neu-shadow-inset); border: 1px solid var(--neu-border);"
                >
                  <Bell size={16} strokeWidth={1.5} color="var(--neu-text-muted)" />
                </div>
              {/if}
              <div
                class="absolute flex items-center justify-center"
                style="width: 18px; height: 18px; border-radius: 9999px; background: {typeColor}; border: 2px solid var(--neu-bg); bottom: -2px; right: -2px;"
              >
                <TypeIcon size={10} strokeWidth={2} color="white" />
              </div>
            </div>

            <div class="min-w-0 flex-1">
              <button
                class="w-full text-left outline-none"
                onclick={() => handleNotificationClick(item)}
              >
                <div
                  style="font-size: 13px; color: var(--neu-text); font-weight: 400; line-height: 1.4;"
                >
                  <span class="mr-1">{getNotificationTitle(item, locale)}</span>
                  <span style="color: var(--neu-text-muted);"
                    >{getNotificationContent(item, locale)}</span
                  >
                </div>
                <p style="font-size: 11px; color: var(--neu-text-muted); margin-top: 2px;">
                  {formatNotificationTimestamp(item.created_at, locale)}
                </p>
              </button>
              {#if canHandleGroupInvite(item)}
                <div class="mt-2 flex gap-2">
                  <button
                    class="rounded-full px-3 py-1 text-xs outline-none transition-transform duration-150 active:scale-[0.93]"
                    style="background: var(--neu-accent-primary); color: var(--neu-active-text); box-shadow: 0 2px 6px var(--neu-glow);"
                    disabled={isProcessing(item.notification_id)}
                    onclick={() => handleInviteAction(item, 'accept')}>{t('common.accept')}</button
                  >
                  <button
                    class="rounded-full px-3 py-1 text-xs outline-none transition-transform duration-150 active:scale-[0.93]"
                    style="box-shadow: var(--neu-shadow-md); border: 1px solid var(--neu-border); color: var(--neu-text-muted); background: var(--neu-bg);"
                    disabled={isProcessing(item.notification_id)}
                    onclick={() => handleInviteAction(item, 'reject')}>{t('common.reject')}</button
                  >
                </div>
              {:else if canHandleGroupApplication(item)}
                <div class="mt-2 flex gap-2">
                  <button
                    class="rounded-full px-3 py-1 text-xs outline-none transition-transform duration-150 active:scale-[0.93]"
                    style="background: var(--neu-accent-primary); color: var(--neu-active-text); box-shadow: 0 2px 6px var(--neu-glow);"
                    disabled={isProcessing(item.notification_id)}
                    onclick={() => handleApplicationAction(item, 'approve')}
                    >{t('common.approve')}</button
                  >
                  <button
                    class="rounded-full px-3 py-1 text-xs outline-none transition-transform duration-150 active:scale-[0.93]"
                    style="box-shadow: var(--neu-shadow-md); border: 1px solid var(--neu-border); color: var(--neu-text-muted); background: var(--neu-bg);"
                    disabled={isProcessing(item.notification_id)}
                    onclick={() => handleApplicationAction(item, 'reject')}
                    >{t('common.reject')}</button
                  >
                </div>
              {/if}
            </div>

            <div class="flex flex-shrink-0 flex-col items-center gap-1.5 pt-1">
              {#if thumbnailUrl}
                <img
                  src={thumbnailUrl}
                  alt=""
                  style="width: 38px; height: 38px; border-radius: 10px; object-fit: cover; border: 1px solid var(--neu-border);"
                />
              {/if}
            </div>
          </div>
        {/each}
      </div>
      {#if pagination.hasMore}
        <div class="flex justify-center pt-4">
          <button
            class="rounded-full px-5 py-2 outline-none transition-transform duration-150 active:scale-[0.97]"
            style="background: var(--neu-bg); box-shadow: var(--neu-shadow-md); border: 1px solid var(--neu-border); color: var(--neu-text-muted); font-size: 13px;"
            disabled={pagination.loadingMore}
            onclick={() => loadNotifications(false)}
          >
            {#if pagination.loadingMore}<Loader2 class="mr-2 inline h-4 w-4 animate-spin" />{/if}
            {t('common.loadMore')}
          </button>
        </div>
      {/if}
    {/if}
  </div>
</div>
