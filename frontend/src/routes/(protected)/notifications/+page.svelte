<script lang="ts">
  import { onMount } from 'svelte';
  import { goto } from '$app/navigation';
  import { notificationStore } from '$lib/stores/notification.svelte';
  import { groupsService } from '$lib/api/services';
  import { toast } from '$lib/stores/toast.svelte';
  import { usePagination } from '$lib/hooks/use-pagination.svelte';
  import { platformStore } from '$lib/stores/platform.svelte';
  import { getLocale, t } from '$lib/i18n';
  import type { NotificationItem, NotificationPayload, NotificationType } from '$lib/types/api';
  import EmptyState from '$lib/components/common/empty-state.svelte';
  import {
    formatNotificationTimestamp,
    getNotificationContent,
    getNotificationLink,
    getNotificationTitle
  } from '$lib/utils/notification-display';
  import BackButton from '$lib/components/common/back-button.svelte';
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
        return 'var(--theme-error)';
      case 'new_follower':
      case 'mutual_follow':
        return 'var(--brand)';
      case 'moment_commented':
      case 'comment_reply':
      case 'mentioned':
        return 'var(--map-user)';
      case 'group_invite_received':
      case 'group_invite_accepted':
      case 'group_application_received':
      case 'group_application_accepted':
        return 'var(--map-group)';
      case 'system_announcement':
        return 'var(--theme-warning)';
      default:
        return 'var(--brand)';
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

  const overviewTiles = $derived.by(() => {
    const items = pagination.items;
    const total = items.length;
    const unread = items.filter((i) => !i.is_read).length;
    return [
      { label: '全部', value: String(total), color: 'var(--text-primary)', big: true },
      { label: '未读', value: String(unread), color: 'var(--brand)', big: true },
      {
        label: '当前筛选',
        value: filterLabels[activeFilter],
        color: 'var(--text-primary)',
        big: false
      },
      { label: '已读', value: String(total - unread), color: 'var(--text-secondary)', big: true }
    ];
  });

  const distributionRows = $derived.by(() => {
    const items = pagination.items;
    const countOf = (s: Set<NotificationType>) => items.filter((i) => s.has(i.type)).length;
    const like = countOf(new Set<NotificationType>(['moment_liked', 'comment_liked']));
    const follow = countOf(new Set<NotificationType>(['new_follower', 'mutual_follow']));
    const comment = countOf(
      new Set<NotificationType>(['moment_commented', 'comment_reply', 'mentioned'])
    );
    const system = items.length - like - follow - comment;
    return [
      { label: '点赞', value: like, color: 'var(--theme-error)' },
      { label: '关注', value: follow, color: 'var(--brand)' },
      { label: '评论', value: comment, color: 'var(--map-user)' },
      { label: '系统', value: system, color: 'var(--theme-warning)' }
    ];
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

{#snippet overviewSidebar()}
  <!-- Overview stats -->
  <div class="surface-flat rounded-[22px] p-4">
    <span class="eyebrow">通知概览</span>
    <div class="grid grid-cols-2 gap-3" style="margin-top: 12px;">
      {#each overviewTiles as tile (tile.label)}
        <div class="rounded-[16px]" style="background: var(--border-theme); padding: 12px;">
          <p
            style="font-size: 10px; color: var(--text-secondary); letter-spacing: 0.08em; text-transform: uppercase;"
          >
            {tile.label}
          </p>
          <p
            style="font-size: {tile.big
              ? '22px'
              : '14px'}; color: {tile.color}; margin-top: 6px; font-weight: 600; line-height: 1.1;"
          >
            {tile.value}
          </p>
        </div>
      {/each}
    </div>
  </div>

  <!-- Type distribution -->
  <div class="surface-flat rounded-[22px] p-4" style="margin-top: 16px;">
    <span class="eyebrow">类型分布</span>
    <div class="flex flex-col" style="gap: 10px; margin-top: 12px;">
      {#each distributionRows as row (row.label)}
        <div
          class="flex items-center justify-between rounded-[16px]"
          style="background: var(--border-theme); padding: 10px 12px;"
        >
          <div class="flex items-center gap-2">
            <span class="rounded-full" style="width: 8px; height: 8px; background: {row.color};"
            ></span>
            <span style="font-size: 13px; color: var(--text-primary);">{row.label}</span>
          </div>
          <span style="font-size: 12px; color: var(--text-secondary);">{row.value}</span>
        </div>
      {/each}
    </div>
  </div>
{/snippet}

<div class="flex" class:h-full={platformStore.isWide} class:overflow-hidden={platformStore.isWide}>
  <div
    class="flex min-w-0 flex-1 flex-col"
    class:overflow-y-auto={platformStore.isWide}
    class:no-scrollbar={platformStore.isWide}
  >
    <!-- Sticky header -->
    <div
      class="sticky top-0 z-10 page-pad-x pt-3 pb-3"
      style="background-color: var(--surface-bg); border-bottom: 1px solid var(--border-theme);"
    >
      <div class="page-shell">
        <div class="flex items-center justify-between">
          <div class="flex items-center gap-3">
            {#if !platformStore.isDesktop}
              <BackButton onclick={() => window.history.back()} size={36} />
            {/if}
            <div>
              <h1 style="font-size: 18px; font-weight: 600; color: var(--text-primary);">
                {t('notificationsPage.title')}
              </h1>
              {#if notificationStore.unreadCount > 0}
                <p style="font-size: 12px; color: var(--text-secondary); margin-top: 1px;">
                  {notificationStore.unreadCount} 条未读
                </p>
              {/if}
            </div>
          </div>
          {#if notificationStore.unreadCount > 0}
            <button
              class="surface-flat flex items-center gap-1.5 rounded-full px-3 py-1.5 outline-none transition-transform duration-150 active:scale-[0.93]"
              style="font-size: 12px; color: var(--brand);"
              onclick={markAllRead}
            >
              <CheckCheck size={13} strokeWidth={2} />
              {t('notificationDropdown.markAllRead')}
            </button>
          {/if}
        </div>
      </div>
    </div>

    <!-- Filter pills -->
    <div class="page-pad-x py-3" style="scrollbar-width: none;">
      <div class="page-shell">
        <div class="flex gap-2 overflow-x-auto">
          {#each Object.entries(filterLabels) as [key, label]}
            <button
              class="filter-chip"
              class:filter-chip--active={activeFilter === key}
              onclick={() => (activeFilter = key as FilterKey)}
            >
              {label}
            </button>
          {/each}
        </div>
      </div>
    </div>

    <div class="flex-1 page-pad">
      <div class="page-shell">
        {#if pagination.loading}
          <div class="flex justify-center py-12">
            <Loader2 class="h-8 w-8 animate-spin" color="var(--text-secondary)" />
          </div>
        {:else if pagination.error}
          <EmptyState icon={Bell} title={t('notificationsPage.loadFailed')} />
          <div class="flex justify-center pt-2">
            <button
              class="surface-flat rounded-full px-5 py-2 outline-none"
              style="color: var(--brand); font-size: 13px;"
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
              style="font-size: 10px; letter-spacing: 0.14em; color: var(--text-secondary);"
            >
              新消息
            </p>
            {#each unreadGroup as item (item.notification_id)}
              {@const locale = getLocale()}
              {@const typeColor = getTypeColor(item.type)}
              {@const typeSoft = `color-mix(in srgb, ${typeColor} 7%, transparent)`}
              {@const typeBorder = `color-mix(in srgb, ${typeColor} 18%, transparent)`}
              {@const TypeIcon = getTypeIcon(item.type)}
              {@const avatarUrl = getActorAvatar(item)}
              {@const thumbnailUrl = getPostThumbnail(item)}
              <div
                class="row-hover flex items-start gap-3.5 rounded-[18px] px-3 py-3.5 transition-transform duration-150 active:scale-[0.98]"
                style="background: {typeSoft};
                border: 1px solid {typeBorder};"
              >
                <!-- Avatar with type badge -->
                <div class="relative flex-shrink-0" style="width: 44px; height: 44px;">
                  {#if avatarUrl}
                    <img
                      src={avatarUrl}
                      alt=""
                      style="width: 44px; height: 44px; border-radius: 9999px; object-fit: cover; border: 1px solid var(--border-theme);"
                    />
                  {:else}
                    <div
                      class="input-surface flex items-center justify-center"
                      style="width: 44px; height: 44px; border-radius: 9999px;"
                    >
                      <Bell size={16} strokeWidth={1.5} color="var(--text-secondary)" />
                    </div>
                  {/if}
                  <div
                    class="absolute flex items-center justify-center"
                    style="width: 18px; height: 18px; border-radius: 9999px; background: {typeColor}; border: 2px solid var(--surface-bg); bottom: -2px; right: -2px;"
                  >
                    <TypeIcon size={10} strokeWidth={2} color="var(--text-on-accent)" />
                  </div>
                </div>

                <div class="min-w-0 flex-1">
                  <button
                    class="w-full text-left outline-none"
                    onclick={() => handleNotificationClick(item)}
                  >
                    <div
                      style="font-size: 13px; color: var(--text-primary); font-weight: 500; line-height: 1.4;"
                    >
                      <span class="mr-1">{getNotificationTitle(item, locale)}</span>
                      <span style="color: var(--text-secondary);"
                        >{getNotificationContent(item, locale)}</span
                      >
                    </div>
                    <p style="font-size: 11px; color: var(--text-secondary); margin-top: 2px;">
                      {formatNotificationTimestamp(item.created_at, locale)}
                    </p>
                  </button>
                  {#if canHandleGroupInvite(item)}
                    <div class="mt-2 flex gap-2">
                      <button
                        class="rounded-full px-3 py-1 text-xs outline-none transition-transform duration-150 active:scale-[0.93]"
                        style="background: var(--brand); color: var(--text-on-accent); box-shadow: 0 2px 6px var(--glow);"
                        disabled={isProcessing(item.notification_id)}
                        onclick={() => handleInviteAction(item, 'accept')}
                        >{t('common.accept')}</button
                      >
                      <button
                        class="surface-flat rounded-full px-3 py-1 text-xs outline-none transition-transform duration-150 active:scale-[0.93]"
                        style="color: var(--text-secondary);"
                        disabled={isProcessing(item.notification_id)}
                        onclick={() => handleInviteAction(item, 'reject')}
                        >{t('common.reject')}</button
                      >
                    </div>
                  {:else if canHandleGroupApplication(item)}
                    <div class="mt-2 flex gap-2">
                      <button
                        class="rounded-full px-3 py-1 text-xs outline-none transition-transform duration-150 active:scale-[0.93]"
                        style="background: var(--brand); color: var(--text-on-accent); box-shadow: 0 2px 6px var(--glow);"
                        disabled={isProcessing(item.notification_id)}
                        onclick={() => handleApplicationAction(item, 'approve')}
                        >{t('common.approve')}</button
                      >
                      <button
                        class="surface-flat rounded-full px-3 py-1 text-xs outline-none transition-transform duration-150 active:scale-[0.93]"
                        style="color: var(--text-secondary);"
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
                      style="width: 38px; height: 38px; border-radius: 10px; object-fit: cover; border: 1px solid var(--border-theme);"
                    />
                  {/if}
                  <div
                    style="width: 8px; height: 8px; border-radius: 9999px; background: var(--brand);"
                  ></div>
                </div>
              </div>
            {/each}
            {#if readGroup.length > 0}
              <p
                class="uppercase mt-4 mb-2"
                style="font-size: 10px; letter-spacing: 0.14em; color: var(--text-secondary);"
              >
                较早
              </p>
            {/if}
          {/if}
          <!-- Read group -->
          {#each readGroup as item (item.notification_id)}
            {@const locale = getLocale()}
            {@const typeColor = getTypeColor(item.type)}
            {@const typeSoft = `color-mix(in srgb, ${typeColor} 7%, transparent)`}
            {@const typeBorder = `color-mix(in srgb, ${typeColor} 18%, transparent)`}
            {@const TypeIcon = getTypeIcon(item.type)}
            {@const avatarUrl = getActorAvatar(item)}
            {@const thumbnailUrl = getPostThumbnail(item)}
            <div
              class="row-hover flex items-start gap-3.5 rounded-[18px] px-3 py-3.5 transition-transform duration-150 active:scale-[0.98]"
              style="background: {typeSoft}; border: 1px solid {typeBorder};"
            >
              <div class="relative flex-shrink-0" style="width: 44px; height: 44px;">
                {#if avatarUrl}
                  <img
                    src={avatarUrl}
                    alt=""
                    style="width: 44px; height: 44px; border-radius: 9999px; object-fit: cover; border: 1px solid var(--border-theme);"
                  />
                {:else}
                  <div
                    class="flex items-center justify-center"
                    style="width: 44px; height: 44px; border-radius: 9999px; background: var(--surface-bg); box-shadow: var(--shadow-sm); border: 1px solid var(--border-theme);"
                  >
                    <Bell size={16} strokeWidth={1.5} color="var(--text-secondary)" />
                  </div>
                {/if}
                <div
                  class="absolute flex items-center justify-center"
                  style="width: 18px; height: 18px; border-radius: 9999px; background: {typeColor}; border: 2px solid var(--surface-bg); bottom: -2px; right: -2px;"
                >
                  <TypeIcon size={10} strokeWidth={2} color="var(--text-on-accent)" />
                </div>
              </div>

              <div class="min-w-0 flex-1">
                <button
                  class="w-full text-left outline-none"
                  onclick={() => handleNotificationClick(item)}
                >
                  <div
                    style="font-size: 13px; color: var(--text-primary); font-weight: 400; line-height: 1.4;"
                  >
                    <span class="mr-1">{getNotificationTitle(item, locale)}</span>
                    <span style="color: var(--text-secondary);"
                      >{getNotificationContent(item, locale)}</span
                    >
                  </div>
                  <p style="font-size: 11px; color: var(--text-secondary); margin-top: 2px;">
                    {formatNotificationTimestamp(item.created_at, locale)}
                  </p>
                </button>
                {#if canHandleGroupInvite(item)}
                  <div class="mt-2 flex gap-2">
                    <button
                      class="rounded-full px-3 py-1 text-xs outline-none transition-transform duration-150 active:scale-[0.93]"
                      style="background: var(--brand); color: var(--text-on-accent); box-shadow: 0 2px 6px var(--glow);"
                      disabled={isProcessing(item.notification_id)}
                      onclick={() => handleInviteAction(item, 'accept')}
                      >{t('common.accept')}</button
                    >
                    <button
                      class="rounded-full px-3 py-1 text-xs outline-none transition-transform duration-150 active:scale-[0.93]"
                      style="box-shadow: var(--shadow-md); border: 1px solid var(--border-theme); color: var(--text-secondary); background: var(--surface-bg);"
                      disabled={isProcessing(item.notification_id)}
                      onclick={() => handleInviteAction(item, 'reject')}
                      >{t('common.reject')}</button
                    >
                  </div>
                {:else if canHandleGroupApplication(item)}
                  <div class="mt-2 flex gap-2">
                    <button
                      class="rounded-full px-3 py-1 text-xs outline-none transition-transform duration-150 active:scale-[0.93]"
                      style="background: var(--brand); color: var(--text-on-accent); box-shadow: 0 2px 6px var(--glow);"
                      disabled={isProcessing(item.notification_id)}
                      onclick={() => handleApplicationAction(item, 'approve')}
                      >{t('common.approve')}</button
                    >
                    <button
                      class="rounded-full px-3 py-1 text-xs outline-none transition-transform duration-150 active:scale-[0.93]"
                      style="box-shadow: var(--shadow-md); border: 1px solid var(--border-theme); color: var(--text-secondary); background: var(--surface-bg);"
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
                    style="width: 38px; height: 38px; border-radius: 10px; object-fit: cover; border: 1px solid var(--border-theme);"
                  />
                {/if}
              </div>
            </div>
          {/each}
        </div>
        {#if pagination.hasMore}
          <div class="flex justify-center pt-4">
            <button
              class="surface-flat rounded-full px-5 py-2 outline-none transition-transform duration-150 active:scale-[0.97]"
              style="color: var(--text-secondary); font-size: 13px;"
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
  </div>
  {#if platformStore.isWide}
    <aside
      class="flex-shrink-0 overflow-y-auto no-scrollbar"
      style="width: 300px; border-left: 1px solid var(--border-theme); padding: 20px 16px;"
    >
      {@render overviewSidebar()}
    </aside>
  {/if}
</div>
