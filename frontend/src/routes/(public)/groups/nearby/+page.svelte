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
  import { PAGE_SIZE } from '$lib/config/pagination';
  import { authStore } from '$lib/stores/auth.svelte';
  import { uiStore } from '$lib/stores/ui.svelte';
  import BackButton from '$lib/components/common/back-button.svelte';
  import Search from 'lucide-svelte/icons/search';
  import UsersRound from 'lucide-svelte/icons/users-round';
  import Plus from 'lucide-svelte/icons/plus';
  import AlertTriangle from 'lucide-svelte/icons/triangle-alert';

  const pagination = usePagination<GroupItem>({
    uniqueKey: (item) => item.group.group_id
  });

  let hasLoadedOnce = $state(false);
  let showCreateGroup = $state(false);
  let searchText = $state('');
  let activeCategory = $state('全部');

  // 从已加载的群组数据中动态提取标签,替代硬编码分类
  const availableCategories = $derived.by(() => {
    const tagSet = new Set<string>();
    for (const item of pagination.items) {
      for (const tag of item.tags ?? []) {
        tagSet.add(tag);
      }
    }
    return ['全部', ...Array.from(tagSet).sort()];
  });

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
      (cursor, opts) =>
        groupsService.search({
          radius: mapAnchorStore.radius * 1000,
          reference_point: getAnchor(),
          pagination: { limit: PAGE_SIZE.default, cursor }
        }, opts),
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
    <!-- 桌面端（常驻侧栏档）内容区全宽出血、无 shell 收束；其余档 shell 居中。
         这是 ui-layout.md 明确的视口级例外（nearby 桌面最大化空间利用）。 -->
    <div class:page-shell={!platformStore.useSidebarChrome}>
      <div class="mb-3 flex items-center gap-3">
        {#if !platformStore.hideRedundantEntry}
          <BackButton onclick={handleBack} size={36} />
        {/if}
        <h1 class="text-h1 flex-1" style="font-weight: 600; color: var(--text-primary)">
          附近群组
        </h1>
        {#if !platformStore.hideRedundantEntry}
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
      <div
        class="input-surface mb-3 flex items-center gap-3 rounded-[var(--radius-md)] px-4 py-2.5"
      >
        <Search size={15} strokeWidth={1.5} color="var(--text-secondary)" class="flex-shrink-0" />
        <input
          type="text"
          bind:value={searchText}
          placeholder="搜索群组"
          class="text-body flex-1 bg-transparent outline-none"
          style="color: var(--text-primary)"
        />
      </div>

      <!-- Category chips (sourced from actual group tags) -->
      <div class="no-scrollbar mb-3 flex gap-2 overflow-x-auto">
        {#each availableCategories as c (c)}
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
              <UsersRound size={19} strokeWidth={2} color="var(--text-on-accent)" />
            </div>
            <div class="min-w-0 flex-1">
              <p class="text-body" style="font-weight: 500; color: var(--text-primary)">附近概览</p>
              <p class="text-caption" style="color: var(--text-secondary); margin-top: 2px">
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
          <p class="text-caption flex-shrink-0" style="color: var(--text-secondary)">
            共 {pagination.items.length} 个群组
          </p>
        </div>
      {/if}

      {#if pagination.loading}
        <div class="density-grid">
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
          description="换个浏览位置或稍后再来看看，也可以发起一个新群组。"
          actionText="刷新"
          onAction={handleRetry}
          fill
        />
      {:else if filteredGroups.length === 0}
        <EmptyState
          icon={UsersRound}
          title="没有匹配的群组"
          description="试试其他分类或关键词，或清除筛选条件。"
          actionText="清除筛选"
          onAction={() => {
            activeCategory = '全部';
            searchText = '';
          }}
          fill
        />
      {:else}
        <div class="density-grid">
          {#each filteredGroups as group (group.group.group_id)}
            <div style="content-visibility: auto; contain-intrinsic-size: 120px;">
              <GroupCard {group} showDistance showMembers variant="default" class="overflow-hidden" />
            </div>
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
