<script lang="ts">
  import { page } from '$app/state';
  import { authStore } from '$lib/stores/auth.svelte';
  import { usersService } from '$lib/api/services';
  import { navigateToNearbyBackTarget } from '../../nearby-back';
  import { mapAnchorStore } from '$lib/stores/map-anchor.svelte';
  import { platformStore } from '$lib/stores/platform.svelte';
  import { usePagination } from '$lib/hooks/use-pagination.svelte';
  import { useLocationAwareLoad } from '$lib/hooks/use-location-aware-load.svelte';
  import { presenceStore } from '$lib/stores/presence.svelte';
  import UserCard from '$lib/components/common/user-card.svelte';
  import EmptyState from '$lib/components/common/empty-state.svelte';
  import LoadMoreButton from '$lib/components/common/load-more-button.svelte';
  import LocationPickerModal from '$lib/components/common/location-picker-modal.svelte';
  import AnchorEntry from '$lib/components/common/anchor-entry.svelte';
  import StatSegments from '$lib/components/common/stat-segments.svelte';
  import type { UserItem } from '$lib/types/api';
  import { PAGE_SIZE } from '$lib/config/pagination';
  import BackButton from '$lib/components/common/back-button.svelte';
  import { t } from '$lib/i18n';
  import type { TranslationKey } from '$lib/i18n';
  import Search from 'lucide-svelte/icons/search';
  import Radar from 'lucide-svelte/icons/radar';
  import Users from 'lucide-svelte/icons/users';
  import AlertTriangle from 'lucide-svelte/icons/triangle-alert';

  const SORT_OPTIONS = ['distance', 'online', 'mutual'] as const;
  const SORT_LABELS: Record<string, TranslationKey> = {
    distance: 'usersNearby.sortDistance',
    online: 'usersNearby.sortOnline',
    mutual: 'usersNearby.sortMutual'
  };

  const pagination = usePagination<UserItem>({
    uniqueKey: (item) => item.user.user_id
  });

  let hasLoadedOnce = $state(false);
  let searchText = $state('');
  let activeSort = $state<string>('distance');
  let showLocationPicker = $state(false);

  const returnTarget = $derived(page.url.searchParams.get('from') || '/search');

  const onlineCount = $derived(
    pagination.items.filter((item) => {
      const uid = item.user.user_id;
      const pStatus = presenceStore.getStatus(uid);
      if (pStatus !== 'offline') return presenceStore.isOnline(uid);
      return item.presence_status === 'online';
    }).length
  );

  function getAnchor() {
    return {
      latitude: mapAnchorStore.latitude,
      longitude: mapAnchorStore.longitude
    };
  }

  async function loadUsers(refresh = false) {
    await pagination.load(async (cursor, opts) => {
      const response = await usersService.search({
        radius: mapAnchorStore.radius * 1000,
        reference_point: getAnchor(),
        pagination: { limit: PAGE_SIZE.default, cursor }
      }, opts);
      const currentUserId = authStore.currentUserId;
      if (currentUserId && response.data?.items) {
        response.data.items = response.data.items.filter(
          (item) => item.user.user_id !== currentUserId
        );
      }
      return response;
    }, refresh);
    hasLoadedOnce = true;
  }

  function handleBack() {
    navigateToNearbyBackTarget(typeof window === 'undefined' ? undefined : window, returnTarget);
  }

  function handleRetry() {
    void loadUsers(true);
  }

  useLocationAwareLoad((refresh) => loadUsers(refresh));
</script>

<svelte:head>
  <title>{t('usersNearby.title')} - HotSpot</title>
</svelte:head>

<div>
  <!-- Sticky header -->
  <div
    class="sticky top-0 z-10 page-pad-x pt-3 pb-3"
    style="background-color: var(--surface-bg); border-bottom: 1px solid var(--border-theme);"
  >
    <!-- 桌面端（常驻侧栏档）内容区全宽出血、无 shell 收束；其余档 shell 居中。
         这是 ui-layout.md 明确的视口级例外（nearby 桌面最大化空间利用）。 -->
    <div class:page-shell={!platformStore.useSidebarChrome}>
      <div class="mb-3 flex items-center gap-3">
        {#if !platformStore.hideRedundantEntry}
          <BackButton onclick={handleBack} size={36} />
        {/if}
        <div class="flex-1">
          <h1 class="text-h1" style="font-weight: 600; color: var(--text-primary)">{t('usersNearby.title')}</h1>
        </div>
      </div>

      <!-- Search -->
      <div
        class="input-surface mb-3 flex items-center gap-3 rounded-[var(--radius-md)] px-4 py-2.5"
      >
        <Search size={15} strokeWidth={1.5} color="var(--text-secondary)" class="flex-shrink-0" />
        <input
          type="text"
          name="nearby-users-search"
          aria-label={t('usersNearby.searchAriaLabel')}
          bind:value={searchText}
          placeholder={t('usersNearby.searchPlaceholder')}
          class="text-body flex-1 bg-transparent outline-none"
          style="color: var(--text-primary)"
        />
      </div>

      <!-- Sort tabs -->
      <div class="no-scrollbar mb-3 flex items-center gap-2 overflow-x-auto">
        {#each SORT_OPTIONS as s (s)}
          <button
            class="filter-chip"
            class:filter-chip--active={activeSort === s}
            onclick={() => (activeSort = s)}
          >
            {t(SORT_LABELS[s])}
          </button>
        {/each}
      </div>
    </div>
  </div>

  <!-- List -->
  <div class="w-full page-pad">
    <div class:page-shell={!platformStore.useSidebarChrome}>
      <!-- Summary — 桌面用 hero 统计卡（与全宽出血协调）；其余档用 anchor pill + 计数。
           location/radius 已在 anchor entry，此处只承载页面统计。 -->
      {#if platformStore.useSidebarChrome}
        <div class="ambient-glow mb-5">
          <div
            class="surface-hero flex items-center gap-4 rounded-[var(--radius-lg)]"
            style="padding: 14px 18px;"
          >
            <div
              class="flex flex-shrink-0 items-center justify-center rounded-full"
              style="width: 44px; height: 44px; background: var(--brand); box-shadow: var(--shadow-brand-btn);"
            >
              <Radar size={19} strokeWidth={2} color="var(--text-on-accent)" />
            </div>
            <div class="min-w-0 flex-1">
              <p class="text-body" style="font-weight: 500; color: var(--text-primary)">{t('usersNearby.overviewTitle')}</p>
              <p class="text-caption" style="color: var(--text-secondary); margin-top: 2px">
                {t('usersNearby.overviewDesc')}
              </p>
            </div>
            <StatSegments
              class="flex-shrink-0"
              stats={[
                { value: onlineCount, label: t('usersNearby.onlineLabel'), color: 'var(--semantic-online)' },
                { value: pagination.items.length, label: t('usersNearby.totalLabel') },
                { value: `${mapAnchorStore.radius}km`, label: t('usersNearby.radiusLabel') }
              ]}
            />
          </div>
        </div>
      {:else}
        <div class="mb-4 flex items-center justify-between gap-3">
          <AnchorEntry variant="pill" onAdjust={() => (showLocationPicker = true)} />
          <p class="text-caption flex-shrink-0" style="color: var(--text-secondary)">
            <span style="color: var(--semantic-online);">{t('usersNearby.onlineCount', { count: onlineCount })}</span>
            <span style="margin-left: 6px;">· {t('usersNearby.totalCount', { count: pagination.items.length })}</span>
          </p>
        </div>
      {/if}

      {#if pagination.loading}
        <div class="density-grid">
          {#each Array(6) as _, i (i)}
            <div
              class="animate-pulse rounded-[var(--radius-lg)]"
              style="height: 72px; background: var(--border-theme);"
            ></div>
          {/each}
        </div>
      {:else if pagination.error}
        <EmptyState
          icon={AlertTriangle}
          title={t('usersNearby.loadFailed')}
          description={pagination.error}
          actionText={t('usersNearby.retry')}
          onAction={handleRetry}
          fill
        />
      {:else if pagination.items.length === 0 && hasLoadedOnce}
        <EmptyState
          icon={Users}
          title={t('usersNearby.emptyTitle')}
          description={t('usersNearby.emptyDesc')}
          actionText={t('usersNearby.refresh')}
          onAction={handleRetry}
          fill
        />
      {:else}
        <div class="density-grid">
          {#each pagination.items as user (user.user.user_id)}
            <div style="content-visibility: auto; contain-intrinsic-size: 100px;">
              <UserCard
                {user}
                showDistance
                showBio
                showTags
                showOnlineStatus
                variant="nearby-row"
                class="overflow-hidden"
              />
            </div>
          {/each}
        </div>

        <LoadMoreButton
          loading={pagination.loadingMore}
          hasMore={pagination.hasMore}
          onLoadMore={() => loadUsers(false)}
        />
      {/if}
    </div>
  </div>
</div>

<LocationPickerModal bind:open={showLocationPicker} />
