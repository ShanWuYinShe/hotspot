<script lang="ts">
  import { mapAnchorStore } from '$lib/stores/map-anchor.svelte';
  import { feedService } from '$lib/api/services';
  import { usePagination } from '$lib/hooks/use-pagination.svelte';
  import { platformStore } from '$lib/stores/platform.svelte';
  import MomentCard from '$lib/components/moments/moment-card.svelte';
  import EmptyState from '$lib/components/common/empty-state.svelte';
  import LoadMoreButton from '$lib/components/common/load-more-button.svelte';
  import Avatar from '$lib/components/common/avatar.svelte';
  import Search from 'lucide-svelte/icons/search';
  import X from 'lucide-svelte/icons/x';
  import Compass from 'lucide-svelte/icons/compass';
  import TrendingUp from 'lucide-svelte/icons/trending-up';
  import UsersRound from 'lucide-svelte/icons/users-round';
  import type { MomentItem, TrendingTagItem } from '$lib/types/api';
  import { PAGE_SIZE } from '$lib/config/pagination';

  const TABS = [
    { id: 'all', label: '热门' },
    { id: 'nearby', label: '附近' },
    { id: 'activity', label: '活动' },
    { id: 'following', label: '关注' }
  ];

  let trendingTags: TrendingTagItem[] = $state([]);

  async function loadTrending() {
    try {
      const result = await feedService.getTrending();
      if (result.code === 0) {
        trendingTags = result.data?.trending_tags ?? [];
      }
    } catch {
      // trending data non-critical, silently fail
    }
  }

  let searchVal = $state('');
  let activeTab = $state('all');
  let activeTag = $state<string | null>(null);
  let dismissedIds = $state<Set<string>>(new Set());

  const feed = usePagination<MomentItem>({
    uniqueKey: 'moment_id',
    initialLoading: true
  });

  async function loadFeed(refresh = false) {
    await feed.load(async (cursor) => {
      return await feedService.getMoments({
        reference_point: {
          latitude: mapAnchorStore.latitude,
          longitude: mapAnchorStore.longitude
        },
        source: activeTab === 'following' ? 'following' : 'all',
        radius: mapAnchorStore.radius * 1000,
        pagination: { limit: PAGE_SIZE.default, cursor }
      });
    }, refresh);
  }

  function getFilteredItems(): MomentItem[] {
    const q = searchVal.trim().toLowerCase();
    return feed.items.filter((item) => {
      if (dismissedIds.has(item.moment_id)) return false;
      if (activeTag && !(item.tags && item.tags.includes(activeTag))) return false;
      if (q) {
        const text =
          (item.user.nickname || '').toLowerCase() + (item.description || '').toLowerCase();
        if (!text.includes(q)) return false;
      }
      return true;
    });
  }

  function handleDismiss(momentId: string) {
    dismissedIds = new Set([...dismissedIds, momentId]);
  }

  function handleTagClick(tag: string) {
    activeTag = activeTag === tag ? null : tag;
  }

  function handleSearch() {
    void loadFeed(true);
  }

  function handleTrendingClick(tag: string) {
    searchVal = tag.replace('#', '');
    void loadFeed(true);
  }

  // 首次挂载加载一次；后续刷新由 handleSearch / handleTrendingClick 等显式触发，
  // 避免顶级 $effect 重跑导致重叠 loadFeed(true) 调用堆叠
  let initialized = false;
  $effect(() => {
    if (initialized) return;
    initialized = true;
    void loadFeed(true);
    void loadTrending();
  });

  // Unique users from feed for sidebar
  const topUsers = $derived(
    [...new Map(feed.items.map((m) => [m.user.user_id, m])).values()].slice(0, 4).map((m) => ({
      id: m.user.user_id,
      nickname: m.user.nickname,
      avatar_url: m.user.avatar_url
    }))
  );
</script>

<svelte:head>
  <title>发现 - HotSpot</title>
</svelte:head>

<!-- Shared header content (used by all platforms) -->
{#snippet headerContent()}
  <div class="flex items-center justify-between mb-2">
    <h1 class="page-title page-title--hero" style="color: var(--text-primary);">发现</h1>
    <span class="text-meta" style="color: var(--text-secondary)"
      >{getFilteredItems().length} 条内容</span
    >
  </div>
  <!-- Search bar -->
  <div
    class="input-surface flex items-center gap-3 rounded-[var(--radius-md)] px-4 py-2.5"
    style="margin-bottom: 10px;"
  >
    <Search size={15} strokeWidth={1.5} color="var(--text-secondary)" class="flex-shrink-0" />
    <input
      type="text"
      name="discover-search"
      aria-label="搜索发现内容"
      bind:value={searchVal}
      placeholder="地点、人物、关键词…"
      class="text-body flex-1 bg-transparent outline-none"
      style="color: var(--text-primary)"
      onkeydown={(e) => e.key === 'Enter' && handleSearch()}
    />
    {#if searchVal.length > 0}
      <button type="button" class="flex-shrink-0 outline-none" onclick={() => (searchVal = '')}>
        <X size={13} strokeWidth={2} color="var(--text-secondary)" />
      </button>
    {/if}
  </div>
  <!-- Tab bar -->
  <div
    class="flex overflow-x-auto no-scrollbar h-scroll"
    style="border-bottom: 1px solid var(--border-theme);"
  >
    {#each TABS as tab (tab.id)}
      <button
        type="button"
        class="text-body relative flex-shrink-0 pb-3 pt-1 outline-none transition-colors"
        style="padding-left: 18px; padding-right: 18px; color: {activeTab === tab.id
          ? 'var(--brand)'
          : 'var(--text-secondary)'}; font-weight: {activeTab === tab.id ? 500 : 400}"
        onclick={() => (activeTab = tab.id)}
      >
        {tab.label}
        {#if activeTab === tab.id}
          <div
            style="position: absolute; bottom: -1px; left: 14%; right: 14%; height: 2px; border-radius: 1px; background: var(--brand);"
          ></div>
        {/if}
      </button>
    {/each}
  </div>
{/snippet}

<!-- Shared tag chips (sourced from server trending tags) -->
{#snippet tagChips()}
  {#if trendingTags.length > 0}
    <div class="mb-5 flex flex-wrap gap-2">
      {#each trendingTags as topic (topic.name)}
        <button
          type="button"
          class="filter-chip"
          class:filter-chip--active={activeTag === topic.name}
          onclick={() => handleTagClick(topic.name)}
        >
          #{topic.name}
        </button>
      {/each}
    </div>
  {/if}
{/snippet}

<!-- Shared feed grid — 容器查询自适应列数，不再按设备档传 mode -->
{#snippet feedGrid()}
  {#if feed.loading}
    <div class="density-grid">
      {#each Array(4) as _}
        <div class="surface-flat rounded-[var(--radius-md)] overflow-hidden">
          <div style="aspect-ratio: 16/9; background: var(--border-theme);"></div>
          <div style="padding: 12px 16px;">
            <div
              class="rounded mb-2"
              style="width: 80%; height: 12px; background: var(--border-theme);"
            ></div>
            <div
              class="rounded"
              style="width: 60%; height: 10px; background: var(--border-theme);"
            ></div>
          </div>
        </div>
      {/each}
    </div>
  {:else if getFilteredItems().length === 0}
    <EmptyState
      icon={Compass}
      title="暂无内容"
      description={activeTag
        ? `「#${activeTag}」标签下还没有内容，换个标签或关键词试试吧。`
        : '附近还没有动态，换个标签或关键词试试吧。'}
      variant="inline"
    />
  {:else}
    {@const items = getFilteredItems()}
    <!-- Hero first card -->
    {#if items[0]}
      {@const firstItem = items[0]}
      <div class="ambient-glow mb-4">
        <MomentCard moment={firstItem} onDismiss={() => handleDismiss(firstItem.moment_id)} />
      </div>
    {/if}
    <!-- Remaining items — 统一 density-grid，卡片按容器列宽自适应（手机单列大卡、
         平板两列、桌面多列），不再区分 list/grid2 mode 与 dense 变体。 -->
    <div class="density-grid">
      {#each items.slice(1) as moment (moment.moment_id)}
        <MomentCard {moment} onDismiss={() => handleDismiss(moment.moment_id)} />
      {/each}
    </div>
    <LoadMoreButton
      loading={feed.loadingMore}
      hasMore={feed.hasMore}
      onLoadMore={() => loadFeed(false)}
    />
  {/if}
{/snippet}

<!-- Shared right sidebar (desktop only) -->
{#snippet rightSidebar()}
  <!-- Trending topics -->
  <div class="mb-6">
    {#if trendingTags.length > 0}
      <div class="flex items-center gap-1.5 mb-3">
        <TrendingUp size={13} strokeWidth={2} color="var(--brand)" />
        <span class="eyebrow">热门话题</span>
      </div>
      {#each trendingTags as topic (topic.name)}
        <button
          type="button"
          class="tag-row flex w-full items-center justify-between rounded-[var(--radius-sm)] px-3 py-2.5 outline-none transition-colors"
          onclick={() => handleTrendingClick(`#${topic.name}`)}
        >
          <span class="text-body" style="color: var(--text-primary)">#{topic.name}</span>
          <span class="text-caption" style="color: var(--text-secondary)"
            >{topic.usage_count}帖</span
          >
        </button>
      {/each}
    {:else}
      <div
        class="flex items-center gap-2.5 rounded-[var(--radius-sm)] px-3 py-2.5"
        style="background: color-mix(in srgb, var(--page-bg) 50%, transparent);"
      >
        <div
          class="flex flex-shrink-0 items-center justify-center rounded-full"
          style="width: 32px; height: 32px; background: color-mix(in srgb, var(--brand) 10%, transparent);"
        >
          <TrendingUp size={15} strokeWidth={1.75} color="var(--brand)" />
        </div>
        <span class="text-meta" style="color: var(--text-secondary)">暂无热门话题</span>
      </div>
    {/if}
  </div>

  <div class="divider-soft mb-5"></div>

  <!-- Active users -->
  <div>
    <span class="eyebrow">活跃用户</span>
    <div class="flex flex-col mt-3" style="gap: 10px;">
      {#each topUsers as user (user.id)}
        <div class="flex items-center gap-2.5">
          <Avatar src={user.avatar_url} name={user.nickname} alt={user.nickname} size={32} />
          <p class="text-meta min-w-0 flex-1 truncate" style="color: var(--text-primary)">
            {user.nickname}
          </p>
          <button
            type="button"
            class="text-xs flex-shrink-0 rounded-full px-2.5 py-1 outline-none"
            style="background: var(--brand); color: var(--text-on-accent); box-shadow: var(--shadow-brand-btn)"
          >
            关注
          </button>
        </div>
      {/each}
      {#if topUsers.length === 0}
        <div
          class="flex items-center gap-2.5 rounded-[var(--radius-sm)] px-3 py-2.5"
          style="background: color-mix(in srgb, var(--page-bg) 50%, transparent);"
        >
          <div
            class="flex flex-shrink-0 items-center justify-center rounded-full"
            style="width: 32px; height: 32px; background: color-mix(in srgb, var(--brand) 10%, transparent);"
          >
            <UsersRound size={15} strokeWidth={1.75} color="var(--brand)" />
          </div>
          <span class="text-meta" style="color: var(--text-secondary)">附近暂无活跃用户</span>
        </div>
      {/if}
    </div>
  </div>
{/snippet}

{#if platformStore.useSidebarChrome}
  <!-- Desktop（常驻侧栏档）：双栏 + 右侧 rightSidebar。rightSidebar 是桌面专属补充内容。 -->
  <div class="flex h-full flex-col">
    <div
      class="flex-shrink-0 px-6 pt-5 pb-0"
      style="background: var(--surface-bg); border-bottom: 1px solid var(--border-theme);"
    >
      {@render headerContent()}
    </div>

    <div class="flex flex-1 min-h-0 overflow-hidden">
      <div class="flex-1 overflow-y-auto no-scrollbar feed-density-pad">
        <div class="flex gap-0">
          <div class="feed-density-shell flex-1 min-w-0">
            {@render tagChips()}
            {@render feedGrid()}
          </div>
          <div
            class="flex-shrink-0"
            style="width: 240px; border-left: 1px solid var(--border-theme); padding: 20px 16px; position: sticky; top: 0; align-self: flex-start;"
          >
            {@render rightSidebar()}
          </div>
        </div>
      </div>
    </div>
  </div>
{:else}
  <!-- Tablet + Mobile：统一单列 shell（自适应宽度）+ density-grid 自动列数。
       不再区分 medium/narrow shell 与 grid2/list mode——density-grid 按容器宽度
       自动决定手机单列 / 平板两列，卡片形态统一。 -->
  <div class="flex h-full flex-col">
    <div
      class="sticky top-0 z-10 flex-shrink-0 feed-density-pad-x pt-3 pb-3"
      style="background: var(--surface-bg); border-bottom: 1px solid var(--border-theme);"
    >
      <div class="feed-density-shell">
        {@render headerContent()}
      </div>
    </div>

    <div class="flex-1 overflow-y-auto no-scrollbar feed-density-shell feed-density-pad">
      {@render tagChips()}
      {@render feedGrid()}
    </div>
  </div>
{/if}

<style>
  .tag-row {
    transition: background-color 0.15s ease;
  }
  @media (hover: hover) {
    .tag-row:hover {
      background: var(--border-theme);
    }
  }
  .tag-row:focus-visible {
    outline: 2px solid var(--glow);
    outline-offset: 2px;
  }
</style>
