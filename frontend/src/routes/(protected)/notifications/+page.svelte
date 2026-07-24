<script lang="ts">
  import { onMount, untrack } from 'svelte';
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
  import Avatar from '$lib/components/common/avatar.svelte';
  import CheckCheck from 'lucide-svelte/icons/check-check';
  import Heart from 'lucide-svelte/icons/heart';
  import Spinner from '$lib/components/common/spinner.svelte';
  import MessageCircle from 'lucide-svelte/icons/message-circle';
  import UserPlus from 'lucide-svelte/icons/user-plus';
  import Users from 'lucide-svelte/icons/users';

  type FilterKey = 'all' | 'like' | 'follow' | 'comment' | 'nearby';

  const filterLabels = $derived<Record<FilterKey, string>>({
    all: t('notificationsPage.filterAll'),
    like: t('notificationsPage.filterLike'),
    follow: t('notificationsPage.filterFollow'),
    comment: t('notificationsPage.filterComment'),
    nearby: t('notificationsPage.filterNearby')
  });

  // Hoist constant type sets to module scope — they never change, so building fresh
  // Set<NotificationType> instances on every derived re-evaluation is wasted work.
  const LIKE_TYPES = new Set<NotificationType>(['moment_liked', 'comment_liked']);
  const FOLLOW_TYPES = new Set<NotificationType>(['new_follower', 'mutual_follow']);
  const COMMENT_TYPES = new Set<NotificationType>([
    'moment_commented',
    'comment_reply',
    'mentioned'
  ]);
  const filterTypeSets: Record<FilterKey, Set<NotificationType> | null> = {
    all: null,
    like: LIKE_TYPES,
    follow: FOLLOW_TYPES,
    comment: COMMENT_TYPES,
    nearby: new Set()
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
      untrack(() => pagination.updateItem((i) => (i.is_read ? i : { ...i, is_read: true })));
      return;
    }
    if (
      notificationStore.lastReadMutationType === 'mark_read' &&
      notificationStore.lastReadNotificationIds.length > 0
    ) {
      const ids = new Set(notificationStore.lastReadNotificationIds);
      untrack(() =>
        pagination.updateItem((i) => (ids.has(i.notification_id) ? { ...i, is_read: true } : null))
      );
    }
  });

  // 单次遍历产出所有派生视图：filtered / unread / read / overview / distribution。
  // 旧实现 5 个 $derived 各遍历一次 pagination.items(~5N/重算)，且 distributionRows 每次
  // rebuild 新 Set。合并后为 1N，常量 Set 已 hoist 到模块作用域。
  // 保留 overview/distribution 内联用 t() —— 派生仍是响应式的，locale 切换会重算。
  let notificationViews = $derived.by(() => {
    const items = pagination.items;
    const filterSet = filterTypeSets[activeFilter];

    const filtered: NotificationItem[] = [];
    const unread: NotificationItem[] = [];
    const read: NotificationItem[] = [];
    let totalUnread = 0;
    let distLike = 0;
    let distFollow = 0;
    let distComment = 0;

    for (const item of items) {
      if (filterSet && !filterSet.has(item.type)) continue;
      filtered.push(item);
      if (item.is_read) {
        read.push(item);
      } else {
        unread.push(item);
        totalUnread++;
      }
      if (LIKE_TYPES.has(item.type)) distLike++;
      else if (FOLLOW_TYPES.has(item.type)) distFollow++;
      else if (COMMENT_TYPES.has(item.type)) distComment++;
    }

    const total = items.length;
    const overview = [
      { label: t('notificationsPage.tileTotal'), value: String(total), color: 'var(--text-primary)', big: true },
      { label: t('notificationsPage.tileUnread'), value: String(totalUnread), color: 'var(--brand)', big: true },
      {
        label: t('notificationsPage.tileCurrentFilter'),
        value: filterLabels[activeFilter],
        color: 'var(--text-primary)',
        big: false
      },
      { label: t('notificationsPage.tileRead'), value: String(total - totalUnread), color: 'var(--text-secondary)', big: true }
    ];

    const system = total - distLike - distFollow - distComment;
    const distribution = [
      { label: t('notificationsPage.distributionLike'), value: distLike, color: 'var(--theme-error)' },
      { label: t('notificationsPage.distributionFollow'), value: distFollow, color: 'var(--brand)' },
      { label: t('notificationsPage.distributionComment'), value: distComment, color: 'var(--map-user)' },
      { label: t('notificationsPage.distributionSystem'), value: system, color: 'var(--theme-warning)' }
    ];

    return { filtered, unread, read, overview, distribution };
  });

  let filteredItems = $derived(notificationViews.filtered);
  let unreadGroup = $derived(notificationViews.unread);
  let readGroup = $derived(notificationViews.read);
  const overviewTiles = $derived(notificationViews.overview);
  const distributionRows = $derived(notificationViews.distribution);

  async function markAllRead() {
    await notificationStore.markAllAsRead();
    // $effect at line 133-147 handles syncing is_read to pagination items automatically
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
    await notificationStore.markAsRead([item.notification_id]);
    // $effect at line 133-147 handles syncing is_read to pagination items automatically
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
      await notificationStore.markAsRead([item.notification_id]);
      // $effect at line 133-147 handles syncing is_read to pagination items automatically
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
  <div class="surface-flat rounded-[var(--radius-lg)] p-4">
    <span class="eyebrow">{t('notificationsPage.overviewTitle')}</span>
    <div class="grid grid-cols-2 gap-3 mt-3">
      {#each overviewTiles as tile (tile.label)}
        <div
          class="rounded-[var(--radius-md)]"
          style="background: var(--border-theme); padding: 12px;"
        >
          <p
            class="text-xs"
            style="color: var(--text-secondary); letter-spacing: 0.08em; text-transform: uppercase"
          >
            {tile.label}
          </p>
          <p
            class:text-display={tile.big}
            class:text-body-lg={!tile.big}
            class:font-semibold={true}
            class="mt-1.5"
            style="color: {tile.color}; line-height: 1.1;"
          >
            {tile.value}
          </p>
        </div>
      {/each}
    </div>
  </div>

  <!-- Type distribution -->
  <div class="surface-flat rounded-[var(--radius-lg)] p-4 mt-4">
    <span class="eyebrow">{t('notificationsPage.distributionTitle')}</span>
    <div class="flex flex-col mt-3 gap-2.5">
      {#each distributionRows as row (row.label)}
        <div
          class="flex items-center justify-between rounded-[var(--radius-md)]"
          style="background: var(--border-theme); padding: 10px 12px;"
        >
          <div class="flex items-center gap-2">
            <span class="rounded-full" style="width: 8px; height: 8px; background: {row.color};"
            ></span>
            <span class="text-body" style="color: var(--text-primary)">{row.label}</span>
          </div>
          <span class="text-meta" style="color: var(--text-secondary)">{row.value}</span>
        </div>
      {/each}
    </div>
  </div>
{/snippet}

<div
  class="flex"
  class:h-full={platformStore.useMasterDetail}
  class:overflow-hidden={platformStore.useMasterDetail}
>
  <div
    class="flex min-w-0 flex-1 flex-col"
    class:overflow-y-auto={platformStore.useMasterDetail}
    class:no-scrollbar={platformStore.useMasterDetail}
  >
    <!-- Sticky header -->
    <div
      class="sticky top-0 z-10 page-pad-x pt-3 pb-3"
      style="background-color: var(--surface-bg); border-bottom: 1px solid var(--border-theme);"
    >
      <div class="page-shell">
        <div class="flex items-center justify-between">
          <div class="flex items-center gap-3">
            {#if !platformStore.hideRedundantEntry}
              <BackButton onclick={() => history.back()} size={36} />
            {/if}
            <div>
              <h1 class="page-title" style="color: var(--text-primary);">
                {t('notificationsPage.title')}
              </h1>
              {#if notificationStore.unreadCount > 0}
                <p class="text-meta mt-px" style="color: var(--text-secondary);">
                  {t('notificationsPage.unreadCount', { count: notificationStore.unreadCount })}
                </p>
              {/if}
            </div>
          </div>
          {#if notificationStore.unreadCount > 0}
            <button
              class="text-meta surface-flat flex items-center gap-1.5 rounded-full px-3 py-1.5 outline-none transition-transform duration-150 active:scale-[0.93]"
              style="color: var(--brand)"
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
        <div class="flex gap-2 overflow-x-auto h-scroll">
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
            <Spinner size="xl" color="var(--text-secondary)" />
          </div>
        {:else if pagination.error}
          <EmptyState icon={Bell} title={t('notificationsPage.loadFailed')} />
          <div class="flex justify-center pt-2">
            <button
              class="text-body surface-flat rounded-full px-5 py-2 outline-none"
              style="color: var(--brand)"
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
                class="text-xs uppercase mb-2 tracking-wider"
                style="color: var(--text-secondary)"
              >
                {t('notificationsPage.groupNew')}
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
                  class="list-row flex items-start gap-3.5 rounded-[var(--radius-md)] px-3 py-3.5"
                  style="background: {typeSoft};
                border: 1px solid {typeBorder};"
                >
                  <!-- Avatar with type badge -->
                  <div class="relative flex-shrink-0">
                    <Avatar src={avatarUrl} name={getNotificationTitle(item, locale)} alt="" size={40} />
                    <div
                      class="absolute flex items-center justify-center"
                      style="width: 18px; height: 18px; border-radius: var(--radius-pill); background: {typeColor}; border: 2px solid var(--surface-bg); bottom: -2px; right: -2px;"
                    >
                      <TypeIcon size={10} strokeWidth={2} color="var(--text-on-accent)" />
                    </div>
                  </div>

                  <div class="min-w-0 flex-1">
                    <button
                      class="w-full text-left outline-none"
                      aria-label={t('notificationsPage.notificationCardAria', {
                        title: getNotificationTitle(item, locale)
                      })}
                      onclick={() => handleNotificationClick(item)}
                    >
                      <div
                        class="text-body"
                        style="color: var(--text-primary); font-weight: 500; line-height: 1.4"
                      >
                        <span class="mr-1">{getNotificationTitle(item, locale)}</span>
                        <span style="color: var(--text-secondary);"
                          >{getNotificationContent(item, locale)}</span
                        >
                      </div>
                      <p class="text-caption mt-0.5" style="color: var(--text-secondary);">
                        {formatNotificationTimestamp(item.created_at, locale)}
                      </p>
                    </button>
                    {#if canHandleGroupInvite(item)}
                      <div class="mt-2 flex gap-2">
                        <button
                          class="rounded-full px-3 py-1 text-xs outline-none transition-transform duration-150 active:scale-[0.93]"
                          style="background: var(--brand); color: var(--text-on-accent); box-shadow: var(--shadow-brand-btn);"
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
                          style="background: var(--brand); color: var(--text-on-accent); box-shadow: var(--shadow-brand-btn);"
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
                        loading="lazy"
                        decoding="async"
                        style="width: 38px; height: 38px; border-radius: var(--radius-md); object-fit: cover; border: 1px solid var(--border-theme);"
                      />
                    {/if}
                    <div
                      style="width: 8px; height: 8px; border-radius: var(--radius-pill); background: var(--brand);"
                    ></div>
                  </div>
                </div>
              {/each}
              {#if readGroup.length > 0}
                <p
                  class="text-xs uppercase mt-4 mb-2 tracking-wider"
                  style="color: var(--text-secondary)"
                >
                  {t('notificationsPage.groupEarlier')}
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
                class="list-row flex items-start gap-3.5 rounded-[var(--radius-md)] px-3 py-3.5"
                style="background: {typeSoft}; border: 1px solid {typeBorder};"
              >
                <div class="relative flex-shrink-0">
                  <Avatar src={avatarUrl} name={getNotificationTitle(item, locale)} alt="" size={40} />
                  <div
                    class="absolute flex items-center justify-center"
                    style="width: 18px; height: 18px; border-radius: var(--radius-pill); background: {typeColor}; border: 2px solid var(--surface-bg); bottom: -2px; right: -2px;"
                  >
                    <TypeIcon size={10} strokeWidth={2} color="var(--text-on-accent)" />
                  </div>
                </div>

                <div class="min-w-0 flex-1">
                  <button
                    class="w-full text-left outline-none"
                    aria-label={t('notificationsPage.notificationCardReadAria', {
                      title: getNotificationTitle(item, locale)
                    })}
                    onclick={() => handleNotificationClick(item)}
                  >
                    <div
                      class="text-body"
                      style="color: var(--text-primary); font-weight: 400; line-height: 1.4"
                    >
                      <span class="mr-1">{getNotificationTitle(item, locale)}</span>
                      <span style="color: var(--text-secondary);"
                        >{getNotificationContent(item, locale)}</span
                      >
                    </div>
                    <p class="text-caption" style="color: var(--text-secondary); margin-top: 2px">
                      {formatNotificationTimestamp(item.created_at, locale)}
                    </p>
                  </button>
                  {#if canHandleGroupInvite(item)}
                    <div class="mt-2 flex gap-2">
                      <button
                        class="rounded-full px-3 py-1 text-xs outline-none transition-transform duration-150 active:scale-[0.93]"
                        style="background: var(--brand); color: var(--text-on-accent); box-shadow: var(--shadow-brand-btn);"
                        disabled={isProcessing(item.notification_id)}
                        onclick={() => handleInviteAction(item, 'accept')}
                        >{t('common.accept')}</button
                      >
                      <button
                        class="rounded-full px-3 py-1 text-xs outline-none transition-transform duration-150 active:scale-[0.93]"
                        style="border: 1px solid var(--border-theme); color: var(--text-secondary); background: var(--surface-bg);"
                        disabled={isProcessing(item.notification_id)}
                        onclick={() => handleInviteAction(item, 'reject')}
                        >{t('common.reject')}</button
                      >
                    </div>
                  {:else if canHandleGroupApplication(item)}
                    <div class="mt-2 flex gap-2">
                      <button
                        class="rounded-full px-3 py-1 text-xs outline-none transition-transform duration-150 active:scale-[0.93]"
                        style="background: var(--brand); color: var(--text-on-accent); box-shadow: var(--shadow-brand-btn);"
                        disabled={isProcessing(item.notification_id)}
                        onclick={() => handleApplicationAction(item, 'approve')}
                        >{t('common.approve')}</button
                      >
                      <button
                        class="rounded-full px-3 py-1 text-xs outline-none transition-transform duration-150 active:scale-[0.93]"
                        style="border: 1px solid var(--border-theme); color: var(--text-secondary); background: var(--surface-bg);"
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
                      style="width: 38px; height: 38px; border-radius: var(--radius-md); object-fit: cover; border: 1px solid var(--border-theme);"
                    />
                  {/if}
                </div>
              </div>
            {/each}
          </div>
          {#if pagination.hasMore}
            <div class="flex justify-center pt-4">
              <button
                class="text-body surface-flat rounded-full px-5 py-2 outline-none transition-transform duration-150 active:scale-[0.97]"
                style="color: var(--text-secondary)"
                disabled={pagination.loadingMore}
                onclick={() => loadNotifications(false)}
              >
                {#if pagination.loadingMore}<Spinner size="md" class="mr-2 inline" />{/if}
                {t('common.loadMore')}
              </button>
            </div>
          {/if}
        {/if}
      </div>
    </div>
  </div>
  {#if platformStore.useMasterDetail}
    <aside
      class="flex-shrink-0 overflow-y-auto no-scrollbar"
      style="width: 220px; border-left: 1px solid var(--border-theme); padding: 20px 16px;"
    >
      {@render overviewSidebar()}
    </aside>
  {/if}
</div>
