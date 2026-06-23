<script lang="ts">
  import { page } from '$app/state';
  import { groupsService } from '$lib/api/services';
  import { navigateToNearbyBackTarget } from '../../nearby-back';
  import { mapAnchorStore } from '$lib/stores/map-anchor.svelte';
  import { platformStore } from '$lib/stores/platform.svelte';
  import { usePagination } from '$lib/hooks/use-pagination.svelte';
  import { useLocationAwareLoad } from '$lib/hooks/use-location-aware-load.svelte';
  import GroupCard from '$lib/components/common/group-card.svelte';
  import EmptyState from '$lib/components/common/empty-state.svelte';
  import LoadMoreButton from '$lib/components/common/load-more-button.svelte';
  import CreateGroupModal from '$lib/components/common/create-group-modal.svelte';
  import AnchorEntry from '$lib/components/common/anchor-entry.svelte';
  import StatSegments from '$lib/components/common/stat-segments.svelte';
  import type { GroupItem } from '$lib/types/api';
  import { authStore } from '$lib/stores/auth.svelte';
  import { uiStore } from '$lib/stores/ui.svelte';
  import BackButton from '$lib/components/common/back-button.svelte';
  import Search from 'lucide-svelte/icons/search';
  import UsersRound from 'lucide-svelte/icons/users-round';
  import Plus from 'lucide-svelte/icons/plus';
  import AlertTriangle from 'lucide-svelte/icons/triangle-alert';

  const CATEGORIES = ['全部', '摄影', '音乐', '户外', '阅读', '美食', '健身', '游戏'];

  const pagination = usePagination<GroupItem>({
    uniqueKey: (item) => item.group.group_id
  });

  let hasLoadedOnce = $state(false);
  let showCreateGroup = $state(false);
  let searchText = $state('');
  let activeCategory = $state('全部');

  const returnTarget = $derived(page.url.searchParams.get('from') || '/search');

  const filteredGroups = $derived(
    pagination.items.filter((item) => {
      const matchesCategory =
        activeCategory === '全部' || (item.tags ?? []).includes(activeCategory);
      const matchesSearch =
        searchText === '' || item.group.name.toLowerCase().includes(searchText.toLowerCase());
      return matchesCategory && matchesSearch;
    })
  );

  function handleCreateGroup() {
    if (!authStore.isAuthenticated) {
      uiStore.openLoginModalWithRedirect({});
      return;
    }
    showCreateGroup = true;
  }

  function getAnchor() {
    return {
      latitude: mapAnchorStore.latitude,
      longitude: mapAnchorStore.longitude
    };
  }

  async function loadGroups(refresh = false) {
    await pagination.load(
      (cursor) =>
        groupsService.search({
          radius: mapAnchorStore.radius * 1000,
          reference_point: getAnchor(),
          pagination: { limit: 20, cursor }
        }),
      refresh
    );
    hasLoadedOnce = true;
  }

  function handleBack() {
    navigateToNearbyBackTarget(typeof window === 'undefined' ? undefined : window, returnTarget);
  }

  function handleRetry() {
    void loadGroups(true);
  }

  useLocationAwareLoad((refresh) => loadGroups(refresh));
</script>

<svelte:head>
  <title>附近群组 - HotSpot</title>
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
        <h1 class="flex-1" style="font-size: 20px; font-weight: 600; color: var(--text-primary);">
          附近群组
        </h1>
        {#if !platformStore.isDesktop}
          <button
            class="flex items-center justify-center rounded-full outline-none"
            style="width: 36px; height: 36px; background: var(--brand); box-shadow: var(--shadow-brand-btn);"
            onclick={handleCreateGroup}
            aria-label="创建群组"
          >
            <Plus size={16} strokeWidth={2} color="var(--text-on-accent)" />
          </button>
        {/if}
      </div>

      <!-- Search -->
      <div class="input-surface mb-3 flex items-center gap-3 rounded-[var(--radius-md)] px-4 py-2.5">
        <Search size={15} strokeWidth={1.5} color="var(--text-secondary)" class="flex-shrink-0" />
        <input
          type="text"
          bind:value={searchText}
          placeholder="搜索群组"
          class="flex-1 bg-transparent outline-none"
          style="font-size: 13px; color: var(--text-primary);"
        />
      </div>

      <!-- Category chips -->
      <div class="no-scrollbar mb-3 flex gap-2 overflow-x-auto">
        {#each CATEGORIES as c (c)}
          <button
            class="filter-chip"
            class:filter-chip--active={activeCategory === c}
            onclick={() => (activeCategory = c)}
          >
            {c}
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
              <UsersRound size={19} strokeWidth={2} color="var(--text-on-accent)" />
            </div>
            <div class="min-w-0 flex-1">
              <p style="font-size: 13px; font-weight: 500; color: var(--text-primary);">附近概览</p>
              <p style="font-size: 11px; color: var(--text-secondary); margin-top: 2px;">
                实时统计当前浏览范围内的群组
              </p>
            </div>
            <StatSegments
              class="flex-shrink-0"
              stats={[
                { value: pagination.items.length, label: '群组' },
                { value: `${mapAnchorStore.radius}km`, label: '半径' }
              ]}
            />
          </div>
        </div>
      {:else}
        <div class="mb-4 flex items-center justify-between gap-3">
          <AnchorEntry variant="pill" />
          <p class="flex-shrink-0" style="font-size: 11px; color: var(--text-secondary);">
            共 {pagination.items.length} 个群组
          </p>
        </div>
      {/if}

      {#if pagination.loading}
        <div class={!platformStore.isMobile ? 'feed-density-grid' : 'flex flex-col gap-2.5'}>
          {#each Array(6) as _, i (i)}
            <div
              class="animate-pulse rounded-[var(--radius-lg)]"
              style="height: 84px; background: var(--border-theme);"
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
          icon={UsersRound}
          title="附近暂无群组"
          description="换个浏览位置或稍后再来看看。"
          fill
        />
      {:else if filteredGroups.length === 0}
        <EmptyState
          icon={UsersRound}
          title="没有匹配的群组"
          description="试试其他分类或关键词。"
          fill
        />
      {:else}
        <div class={!platformStore.isMobile ? 'feed-density-grid' : 'flex flex-col gap-2.5'}>
          {#each filteredGroups as group (group.group.group_id)}
            <GroupCard {group} showDistance showMembers variant="default" class="overflow-hidden" />
          {/each}
        </div>

        <LoadMoreButton
          loading={pagination.loadingMore}
          hasMore={pagination.hasMore}
          onLoadMore={() => loadGroups(false)}
        />
      {/if}
    </div>
  </div>
</div>

<CreateGroupModal bind:open={showCreateGroup} onCreated={() => loadGroups(true)} />
