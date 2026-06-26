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
  import BackButton from '$lib/components/common/back-button.svelte';
  import Search from 'lucide-svelte/icons/search';
  import Radar from 'lucide-svelte/icons/radar';
  import Users from 'lucide-svelte/icons/users';
  import AlertTriangle from 'lucide-svelte/icons/triangle-alert';

  const SORT_OPTIONS = ['距离', '在线', '共同好友'] as const;
  const FILTER_TAGS = ['全部', '摄影', '旅行', '美食', '户外', '城市', '咖啡', '音乐', '运动'];

  const pagination = usePagination<UserItem>({
    uniqueKey: (item) => item.user.user_id
  });

  let hasLoadedOnce = $state(false);
  let searchText = $state('');
  let activeSort = $state<string>('距离');
  let activeTag = $state<string>('全部');
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
    await pagination.load(async (cursor) => {
      const response = await usersService.search({
        radius: mapAnchorStore.radius * 1000,
        reference_point: getAnchor(),
        pagination: { limit: 20, cursor }
      });
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
  <title>附近的人 - HotSpot</title>
</svelte:head>

<div>
  <!-- Sticky header -->
  <div
    class="sticky top-0 z-10 page-pad-x pt-3 pb-3"
    style="background-color: var(--surface-bg); border-bottom: 1px solid var(--border-theme);"
  >
    <div class:page-shell={!platformStore.isDesktop}>
      <div class="mb-3 flex items-center gap-3">
        {#if !platformStore.isDesktop}
          <BackButton onclick={handleBack} size={36} />
        {/if}
        <div class="flex-1">
          <h1 class="text-h1" style="font-weight: 600; color: var(--text-primary)">附近的人</h1>
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
          aria-label="搜索附近用户"
          bind:value={searchText}
          placeholder="搜索附近用户"
          class="text-body flex-1 bg-transparent outline-none"
          style="color: var(--text-primary)"
        />
      </div>

      <!-- Sort tabs + Filter tags -->
      <div class="no-scrollbar mb-3 flex items-center gap-2 overflow-x-auto">
        {#each SORT_OPTIONS as s (s)}
          <button
            class="filter-chip"
            class:filter-chip--active={activeSort === s}
            onclick={() => (activeSort = s)}
          >
            {s}
          </button>
        {/each}
        <div
          class="mx-1 flex-shrink-0"
          style="width: 1px; height: 18px; background: var(--border-theme);"
        ></div>
        {#each FILTER_TAGS as t (t)}
          <button
            class="filter-chip"
            class:filter-chip--active={activeTag === t && t !== '全部'}
            onclick={() => (activeTag = t)}
          >
            {t}
          </button>
        {/each}
      </div>
    </div>
  </div>

  <!-- List -->
  <div class="w-full page-pad">
    <div class:page-shell={!platformStore.isDesktop}>
      <!-- Summary — location/radius lives in the anchor entry, this only carries page stats -->
      {#if platformStore.isDesktop}
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
              <p class="text-body" style="font-weight: 500; color: var(--text-primary)">附近概览</p>
              <p class="text-caption" style="color: var(--text-secondary); margin-top: 2px">
                实时统计当前浏览范围内的用户
              </p>
            </div>
            <StatSegments
              class="flex-shrink-0"
              stats={[
                { value: onlineCount, label: '在线', color: 'var(--semantic-online)' },
                { value: pagination.items.length, label: '总数' },
                { value: `${mapAnchorStore.radius}km`, label: '半径' }
              ]}
            />
          </div>
        </div>
      {:else}
        <div class="mb-4 flex items-center justify-between gap-3">
          <AnchorEntry variant="pill" onAdjust={() => (showLocationPicker = true)} />
          <p class="text-caption flex-shrink-0" style="color: var(--text-secondary)">
            <span style="color: var(--semantic-online);">{onlineCount} 在线</span>
            <span style="margin-left: 6px;">· 共 {pagination.items.length} 人</span>
          </p>
        </div>
      {/if}

      {#if pagination.loading}
        <div class={!platformStore.isMobile ? 'feed-density-grid' : 'flex flex-col gap-2.5'}>
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
          title="加载失败"
          description={pagination.error}
          actionText="重试"
          onAction={handleRetry}
          fill
        />
      {:else if pagination.items.length === 0 && hasLoadedOnce}
        <EmptyState
          icon={Users}
          title="附近暂无其他用户"
          description="换个浏览位置或稍后再来看看。"
          fill
        />
      {:else}
        <div class={!platformStore.isMobile ? 'feed-density-grid' : 'flex flex-col gap-2.5'}>
          {#each pagination.items as user (user.user.user_id)}
            <UserCard
              {user}
              showDistance
              showBio
              showTags
              showOnlineStatus
              variant="nearby-row"
              class="overflow-hidden"
            />
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
