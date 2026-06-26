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
  import type { MomentItem, TrendingTagItem } from '$lib/types/api';

  const TABS = ['热门', '附近', '活动', '关注'];
  const TAGS = ['自然', '城市', '美食', '音乐', '活动', '摄影', '生活', '旅行'];

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
  let activeTab = $state('热门');
  let activeTag = $state<string | null>(null);

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
        source: activeTab === '关注' ? 'following' : 'all',
        radius: mapAnchorStore.radius * 1000,
        pagination: { limit: 20, cursor }
      });
    }, refresh);
  }

  function getFilteredItems(): MomentItem[] {
    const q = searchVal.trim().toLowerCase();
    return feed.items.filter((item) => {
      if (activeTag && !(item.tags && item.tags.includes(activeTag))) return false;
      if (q) {
        const text =
          (item.user.nickname || '').toLowerCase() + (item.description || '').toLowerCase();
        if (!text.includes(q)) return false;
      }
      return true;
    });
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

  $effect(() => {
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
    class="flex overflow-x-auto no-scrollbar"
    style="border-bottom: 1px solid var(--border-theme);"
  >
    {#each TABS as tab (tab)}
      <button
        type="button"
        class="text-body relative flex-shrink-0 pb-3 pt-1 outline-none transition-colors"
        style="padding-left: 18px; padding-right: 18px; color: {activeTab === tab
          ? 'var(--brand)'
          : 'var(--text-secondary)'}; font-weight: {activeTab === tab ? 500 : 400}"
        onclick={() => (activeTab = tab)}
      >
        {tab}
        {#if activeTab === tab}
          <div
            style="position: absolute; bottom: -1px; left: 14%; right: 14%; height: 2px; border-radius: 1px; background: var(--brand);"
          ></div>
        {/if}
      </button>
    {/each}
  </div>
{/snippet}

<!-- Shared tag chips -->
{#snippet tagChips()}
  <div class="mb-5 flex flex-wrap gap-2">
    {#each TAGS as tag (tag)}
      <button
        type="button"
        class="filter-chip"
        class:filter-chip--active={activeTag === tag}
        onclick={() => handleTagClick(tag)}
      >
        {tag}
      </button>
    {/each}
  </div>
{/snippet}

<!-- Shared feed grid -->
{#snippet feedGrid(mode: 'list' | 'grid2' | 'autofill')}
  {#if feed.loading}
    {#if mode === 'list'}
      <div class="feed-density-list">
        {#each Array(3) as _}
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
    {:else}
      <div class="feed-density-grid feed-density-grid--compact">
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
    {/if}
  {:else if getFilteredItems().length === 0}
    <EmptyState icon={Compass} title="暂无内容" description="换个标签或关键词试试" />
  {:else}
    {@const items = getFilteredItems()}
    <!-- Hero first card -->
    {#if items[0]}
      <div class="ambient-glow mb-4">
        <MomentCard moment={items[0]} />
      </div>
    {/if}
    <!-- Remaining items -->
    {#if mode === 'list'}
      <div class="feed-density-list">
        {#each items.slice(1) as moment (moment.moment_id)}
          <MomentCard {moment} />
        {/each}
      </div>
    {:else}
      <div class="feed-density-grid feed-density-grid--compact">
        {#each items.slice(1) as moment (moment.moment_id)}
          <MomentCard {moment} dense />
        {/each}
      </div>
    {/if}
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
    <div class="flex items-center gap-1.5 mb-3">
      <TrendingUp size={13} strokeWidth={2} color="var(--brand)" />
      <span class="eyebrow">热门话题</span>
    </div>
    {#each trendingTags as topic (topic.name)}
      <button
        type="button"
        class="flex w-full items-center justify-between rounded-[var(--radius-sm)] px-3 py-2.5 outline-none transition-colors"
        onmouseenter={(e) => (e.currentTarget.style.background = 'var(--border-theme)')}
        onmouseleave={(e) => (e.currentTarget.style.background = 'transparent')}
        onclick={() => handleTrendingClick(`#${topic.name}`)}
      >
        <span class="text-body" style="color: var(--text-primary)">#{topic.name}</span>
        <span class="text-caption" style="color: var(--text-secondary)">{topic.usage_count}帖</span>
      </button>
    {/each}
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
        <p class="text-meta" style="color: var(--text-secondary)">附近暂无活跃用户</p>
      {/if}
    </div>
  </div>
{/snippet}

{#if platformStore.isDesktop}
  <div class="flex h-full flex-col">
    <div
      class="flex-shrink-0 px-6 pt-5 pb-0"
      style="background: var(--surface-bg); border-bottom: 1px solid var(--border-theme);"
    >
      {@render headerContent()}
    </div>

    <div class="flex flex-1 min-h-0 overflow-hidden">
      <div class="flex-1 overflow-y-auto no-scrollbar feed-density-pad">
        <div class="feed-density-shell">
          {@render tagChips()}
          {@render feedGrid('autofill')}
        </div>
      </div>

      <div
        class="flex-shrink-0 overflow-y-auto no-scrollbar"
        style="width: 220px; border-left: 1px solid var(--border-theme); padding: 20px 16px;"
      >
        {@render rightSidebar()}
      </div>
    </div>
  </div>
{:else if platformStore.isTablet}
  <div class="flex h-full flex-col">
    <div
      class="sticky top-0 z-10 flex-shrink-0 feed-density-pad-x pt-3 pb-3"
      style="background: var(--surface-bg); border-bottom: 1px solid var(--border-theme);"
    >
      <div class="feed-density-shell feed-density-shell--medium">
        {@render headerContent()}
      </div>
    </div>

    <div
      class="flex-1 overflow-y-auto no-scrollbar feed-density-shell feed-density-shell--medium feed-density-pad"
    >
      {@render tagChips()}
      {@render feedGrid('grid2')}
    </div>
  </div>
{:else}
  <div class="flex h-full flex-col">
    <div
      class="sticky top-0 z-10 flex-shrink-0 feed-density-pad-x pt-3 pb-3"
      style="background: var(--surface-bg); border-bottom: 1px solid var(--border-theme);"
    >
      <div class="feed-density-shell feed-density-shell--narrow">
        {@render headerContent()}
      </div>
    </div>

    <div
      class="flex-1 overflow-y-auto no-scrollbar feed-density-shell feed-density-shell--narrow feed-density-pad"
    >
      {@render tagChips()}
      {@render feedGrid('list')}
    </div>
  </div>
{/if}
