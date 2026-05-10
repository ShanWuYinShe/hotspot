<script lang="ts">
  import { mapAnchorStore } from '$lib/stores/map-anchor.svelte';
  import { feedService } from '$lib/api/services';
  import { usePagination } from '$lib/hooks/use-pagination.svelte';
  import { platformStore } from '$lib/stores/platform.svelte';
  import MomentCard from '$lib/components/moments/moment-card.svelte';
  import EmptyState from '$lib/components/common/empty-state.svelte';
  import LoadMoreButton from '$lib/components/common/load-more-button.svelte';
  import Search from 'lucide-svelte/icons/search';
  import X from 'lucide-svelte/icons/x';
  import Compass from 'lucide-svelte/icons/compass';
  import TrendingUp from 'lucide-svelte/icons/trending-up';
  import Sparkles from 'lucide-svelte/icons/sparkles';
  import type { MomentItem } from '$lib/types/api';

  const TABS = ['热门', '附近', '活动', '关注'];
  const TAGS = ['自然', '城市', '美食', '音乐', '活动', '摄影', '生活', '旅行'];

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

  $effect(() => {
    void loadFeed(true);
  });
</script>

<svelte:head>
  <title>发现 - HotSpot</title>
</svelte:head>

{#if platformStore.isDesktop}
  <!-- Desktop Discover: 2-col feed + sidebar -->
  <div class="flex h-full flex-col">
    <!-- Sticky header -->
    <div
      class="flex-shrink-0 px-6 pt-5 pb-0"
      style="background-color: var(--neu-bg); border-bottom: 1px solid var(--neu-border);"
    >
      <div class="mb-2 flex items-center justify-between">
        <h1 style="font-size: 20px; color: var(--neu-text);">发现</h1>
        <span style="font-size: 12px; color: var(--neu-text-muted);"
          >{getFilteredItems().length} 条内容</span
        >
      </div>
      <!-- Search bar -->
      <div
        class="flex items-center gap-3 rounded-[18px] px-4 py-2.5"
        style="margin-bottom: 10px; background: var(--neu-bg); box-shadow: inset 3px 3px 6px var(--neu-shadow-dark), inset -3px -3px 6px var(--neu-shadow-light); border: 1px solid var(--neu-border);"
      >
        <Search size={15} strokeWidth={1.5} style="color: var(--neu-text-muted); flex-shrink: 0;" />
        <input
          type="text"
          bind:value={searchVal}
          placeholder="地点、人物、关键词…"
          class="flex-1 bg-transparent outline-none"
          style="font-size: 13px; color: var(--neu-text);"
          onkeydown={(e) => e.key === 'Enter' && handleSearch()}
        />
        {#if searchVal.length > 0}
          <button type="button" class="flex-shrink-0 outline-none" onclick={() => (searchVal = '')}>
            <X size={13} strokeWidth={2} style="color: var(--neu-text-muted);" />
          </button>
        {/if}
      </div>
      <!-- Tab bar -->
      <div
        class="flex overflow-x-auto no-scrollbar"
        style="border-bottom: 1px solid var(--neu-border);"
      >
        {#each TABS as tab (tab)}
          <button
            type="button"
            class="relative flex-shrink-0 pb-3 pt-1 outline-none"
            style="padding-left: 18px; padding-right: 18px; color: {activeTab === tab
              ? 'var(--neu-accent-primary)'
              : 'var(--neu-text-muted)'}; font-weight: {activeTab === tab
              ? 500
              : 400}; font-size: 13px; transition: color 0.2s;"
            onclick={() => (activeTab = tab)}
          >
            {tab}
            {#if activeTab === tab}
              <div
                style="position: absolute; bottom: -1px; left: 14%; right: 14%; height: 2px; border-radius: 1px; background: var(--neu-accent-primary);"
              ></div>
            {/if}
          </button>
        {/each}
      </div>
    </div>

    <!-- Body: main feed + right sidebar -->
    <div class="flex flex-1 min-h-0 overflow-hidden">
      <!-- Main feed -->
      <div class="flex-1 overflow-y-auto no-scrollbar">
        <div style="padding: 20px 24px 28px;">
          <!-- Tag chips -->
          <div class="mb-5 flex flex-wrap gap-2">
            {#each TAGS as tag (tag)}
              <button
                type="button"
                class="flex-shrink-0 rounded-full px-3.5 py-1.5 outline-none transition-all"
                style={activeTag === tag
                  ? 'background: var(--neu-accent-primary); color: white; font-size: 12px; box-shadow: 0 3px 10px var(--neu-glow); border: 1px solid transparent;'
                  : 'background: var(--neu-bg); color: var(--neu-text-muted); font-size: 12px; box-shadow: 3px 3px 7px var(--neu-shadow-dark), -3px -3px 7px var(--neu-shadow-light); border: 1px solid var(--neu-border);'}
                onclick={() => handleTagClick(tag)}
              >
                {tag}
              </button>
            {/each}
          </div>

          <!-- 2-column grid -->
          {#if feed.loading}
            <div class="flex flex-col gap-4">
              {#each Array(4) as _}
                <div
                  class="rounded-[22px]"
                  style="background: var(--neu-bg); box-shadow: 7px 7px 16px var(--neu-shadow-dark), -7px -7px 16px var(--neu-shadow-light); border: 1px solid var(--neu-border);"
                >
                  <div
                    style="aspect-ratio: 16/9; background: var(--neu-border); border-radius: 22px 22px 0 0;"
                  ></div>
                  <div style="padding: 12px 16px;">
                    <div
                      class="mb-2 rounded"
                      style="width: 80%; height: 12px; background: var(--neu-border);"
                    ></div>
                    <div
                      class="rounded"
                      style="width: 60%; height: 10px; background: var(--neu-border);"
                    ></div>
                  </div>
                </div>
              {/each}
            </div>
          {:else if getFilteredItems().length === 0}
            <EmptyState icon={Compass} title="暂无内容" description="换个标签或关键词试试" />
          {:else}
            <div
              style="display: grid; grid-template-columns: repeat(2, 1fr); gap: 16px; max-width: 800px;"
            >
              {#each getFilteredItems() as moment (moment.moment_id)}
                <MomentCard {moment} dense />
              {/each}
            </div>
            <LoadMoreButton
              loading={feed.loadingMore}
              hasMore={feed.hasMore}
              onLoadMore={() => loadFeed(false)}
            />
          {/if}
        </div>
      </div>

      <!-- Right sidebar -->
      <div
        class="flex-shrink-0 overflow-y-auto no-scrollbar"
        style="width: 220px; border-left: 1px solid var(--neu-border); padding: 20px 16px;"
      >
        <!-- Trending topics -->
        <div class="mb-6">
          <div class="mb-3 flex items-center gap-1.5">
            <TrendingUp size={13} strokeWidth={2} style="color: var(--neu-accent-primary);" />
            <p
              class="uppercase"
              style="font-size: 10px; letter-spacing: 0.12em; color: var(--neu-text-muted);"
            >
              热门话题
            </p>
          </div>
          {#each ['#城市探索', '#周末去哪', '#隐藏咖啡馆', '#街头摄影', '#本地美食'] as tag, i (tag)}
            <button
              type="button"
              class="mb-1 flex w-full items-center justify-between rounded-[12px] px-3 py-2.5 outline-none"
              onclick={() => {
                searchVal = tag.replace('#', '');
                void loadFeed(true);
              }}
            >
              <span style="font-size: 13px; color: var(--neu-text);">{tag}</span>
              <span style="font-size: 11px; color: var(--neu-text-muted);"
                >{[89, 64, 52, 48, 41][i]}帖</span
              >
            </button>
          {/each}
        </div>

        <div style="height: 1px; background: var(--neu-border); margin-bottom: 20px;"></div>

        <!-- Active users (from feed data) -->
        <div>
          <p
            class="mb-3 uppercase"
            style="font-size: 10px; letter-spacing: 0.12em; color: var(--neu-text-muted);"
          >
            活跃用户
          </p>
          {#each feed.items.slice(0, 4) as item (item.moment_id)}
            <div class="mb-3.5 flex items-center gap-2.5">
              <div
                class="flex-shrink-0 overflow-hidden rounded-full"
                style="width: 32px; height: 32px;"
              >
                <img
                  src={item.user.avatar_url || ''}
                  alt={item.user.nickname}
                  class="h-full w-full object-cover"
                />
              </div>
              <div class="min-w-0 flex-1">
                <p class="truncate" style="font-size: 12px; color: var(--neu-text);">
                  {item.user.nickname}
                </p>
                <p class="truncate" style="font-size: 10px; color: var(--neu-text-muted);">
                  {item.user.nickname}
                </p>
              </div>
              <button
                type="button"
                class="flex-shrink-0 rounded-full px-2.5 py-1 outline-none"
                style="background: var(--neu-accent-primary); font-size: 10px; color: white; box-shadow: 0 2px 6px var(--neu-glow);"
              >
                关注
              </button>
            </div>
          {/each}
        </div>
      </div>
    </div>
  </div>
{:else if platformStore.isTablet}
  <!-- Tablet Discover: 2-column grid, max-width 800px -->
  <div>
    <!-- Sticky header -->
    <div
      class="sticky top-0 z-10 px-6 pt-4 pb-0"
      style="background-color: var(--neu-bg); border-bottom: 1px solid var(--neu-border);"
    >
      <div class="mb-2 flex items-center justify-between">
        <h1 style="font-size: 24px; color: var(--neu-text);">发现</h1>
        <div
          class="flex items-center gap-1.5 rounded-full px-3 py-1.5"
          style="background: var(--neu-accent-primary); box-shadow: 0 3px 10px var(--neu-glow);"
        >
          <Sparkles size={11} strokeWidth={2} style="color: white;" />
          <span style="font-size: 11px; color: white;">精选</span>
        </div>
      </div>
      <!-- Search bar -->
      <div
        class="flex items-center gap-3 rounded-[18px] px-4 py-2.5"
        style="margin-bottom: 10px; background: var(--neu-bg); box-shadow: inset 3px 3px 6px var(--neu-shadow-dark), inset -3px -3px 6px var(--neu-shadow-light); border: 1px solid var(--neu-border);"
      >
        <Search size={15} strokeWidth={1.5} style="color: var(--neu-text-muted); flex-shrink: 0;" />
        <input
          type="text"
          bind:value={searchVal}
          placeholder="地点、人物、关键词…"
          class="flex-1 bg-transparent outline-none"
          style="font-size: 13px; color: var(--neu-text);"
        />
        {#if searchVal.length > 0}
          <button type="button" class="flex-shrink-0 outline-none" onclick={() => (searchVal = '')}>
            <X size={13} strokeWidth={2} style="color: var(--neu-text-muted);" />
          </button>
        {/if}
      </div>
      <!-- Tab bar -->
      <div
        class="flex overflow-x-auto no-scrollbar"
        style="border-bottom: 1px solid var(--neu-border);"
      >
        {#each TABS as tab (tab)}
          <button
            type="button"
            class="relative flex-shrink-0 pb-3 pt-1 outline-none"
            style="padding-left: 18px; padding-right: 18px; color: {activeTab === tab
              ? 'var(--neu-accent-primary)'
              : 'var(--neu-text-muted)'}; font-weight: {activeTab === tab
              ? 500
              : 400}; font-size: 13px; transition: color 0.2s;"
            onclick={() => (activeTab = tab)}
          >
            {tab}
            {#if activeTab === tab}
              <div
                style="position: absolute; bottom: -1px; left: 14%; right: 14%; height: 2px; border-radius: 1px; background: var(--neu-accent-primary);"
              ></div>
            {/if}
          </button>
        {/each}
      </div>
    </div>

    <!-- Content -->
    <div
      style="max-width: 800px; margin-left: auto; margin-right: auto; width: 100%; padding: 16px 20px 28px;"
    >
      <!-- Tag chips -->
      <div class="mb-5 flex gap-2 overflow-x-auto no-scrollbar pb-1">
        {#each TAGS as tag (tag)}
          <button
            type="button"
            class="flex-shrink-0 rounded-full px-3.5 py-1.5 outline-none transition-all"
            style={activeTag === tag
              ? 'background: var(--neu-accent-primary); color: white; font-size: 12px; box-shadow: 0 3px 10px var(--neu-glow); border: 1px solid transparent;'
              : 'background: var(--neu-bg); color: var(--neu-text-muted); font-size: 12px; box-shadow: 3px 3px 7px var(--neu-shadow-dark), -3px -3px 7px var(--neu-shadow-light); border: 1px solid var(--neu-border);'}
            onclick={() => handleTagClick(tag)}
          >
            {tag}
          </button>
        {/each}
      </div>

      <!-- 2-column grid -->
      {#if feed.loading}
        <div style="display: grid; grid-template-columns: repeat(2, 1fr); gap: 16px;">
          {#each Array(4) as _}
            <div
              class="rounded-[22px]"
              style="background: var(--neu-bg); box-shadow: 7px 7px 16px var(--neu-shadow-dark), -7px -7px 16px var(--neu-shadow-light); border: 1px solid var(--neu-border);"
            >
              <div
                style="aspect-ratio: 16/9; background: var(--neu-border); border-radius: 22px 22px 0 0;"
              ></div>
              <div style="padding: 10px 14px;">
                <div
                  class="mb-1.5 rounded"
                  style="width: 80%; height: 10px; background: var(--neu-border);"
                ></div>
                <div
                  class="rounded"
                  style="width: 50%; height: 8px; background: var(--neu-border);"
                ></div>
              </div>
            </div>
          {/each}
        </div>
      {:else if getFilteredItems().length === 0}
        <EmptyState icon={Compass} title="暂无内容" description="换个标签或关键词试试" />
      {:else}
        <div style="display: grid; grid-template-columns: repeat(2, 1fr); gap: 16px;">
          {#each getFilteredItems() as moment (moment.moment_id)}
            <MomentCard {moment} dense />
          {/each}
        </div>
        <LoadMoreButton
          loading={feed.loadingMore}
          hasMore={feed.hasMore}
          onLoadMore={() => loadFeed(false)}
        />
      {/if}
    </div>
  </div>
{:else}
  <!-- Mobile Discover: single column -->
  <div>
    <!-- Sticky header -->
    <div
      class="sticky top-0 z-10 px-5 pt-3 pb-0"
      style="background-color: var(--neu-bg); border-bottom: 1px solid var(--neu-border);"
    >
      <div class="mb-2 flex items-center justify-between">
        <h1 style="font-size: 22px; color: var(--neu-text);">发现</h1>
        <span style="font-size: 12px; color: var(--neu-text-muted);"
          >{getFilteredItems().length} 条内容</span
        >
      </div>
      <!-- Search bar -->
      <div
        class="flex items-center gap-3 rounded-[18px] px-4 py-2.5"
        style="margin-bottom: 10px; background: var(--neu-bg); box-shadow: inset 3px 3px 6px var(--neu-shadow-dark), inset -3px -3px 6px var(--neu-shadow-light); border: 1px solid var(--neu-border);"
      >
        <Search size={15} strokeWidth={1.5} style="color: var(--neu-text-muted); flex-shrink: 0;" />
        <input
          type="text"
          bind:value={searchVal}
          placeholder="地点、人物、关键词…"
          class="flex-1 bg-transparent outline-none"
          style="font-size: 13px; color: var(--neu-text);"
          onkeydown={(e) => e.key === 'Enter' && handleSearch()}
        />
        {#if searchVal.length > 0}
          <button type="button" class="flex-shrink-0 outline-none" onclick={() => (searchVal = '')}>
            <X size={13} strokeWidth={2} style="color: var(--neu-text-muted);" />
          </button>
        {/if}
      </div>
      <!-- Tab bar -->
      <div
        class="flex overflow-x-auto no-scrollbar"
        style="border-bottom: 1px solid var(--neu-border);"
      >
        {#each TABS as tab (tab)}
          <button
            type="button"
            class="relative flex-shrink-0 pb-3 pt-1 outline-none"
            style="padding-left: 18px; padding-right: 18px; color: {activeTab === tab
              ? 'var(--neu-accent-primary)'
              : 'var(--neu-text-muted)'}; font-weight: {activeTab === tab
              ? 500
              : 400}; font-size: 13px; transition: color 0.2s;"
            onclick={() => (activeTab = tab)}
          >
            {tab}
            {#if activeTab === tab}
              <div
                style="position: absolute; bottom: -1px; left: 14%; right: 14%; height: 2px; border-radius: 1px; background: var(--neu-accent-primary);"
              ></div>
            {/if}
          </button>
        {/each}
      </div>
    </div>

    <!-- Content -->
    <div
      style="max-width: 560px; margin-left: auto; margin-right: auto; width: 100%; padding: 12px 16px 24px;"
    >
      <!-- Tag chips -->
      <div class="mb-4 flex gap-2 overflow-x-auto no-scrollbar pb-1">
        {#each TAGS as tag (tag)}
          <button
            type="button"
            class="flex-shrink-0 rounded-full px-3.5 py-1.5 outline-none transition-all"
            style={activeTag === tag
              ? 'background: var(--neu-accent-primary); color: white; font-size: 12px; box-shadow: 0 3px 10px var(--neu-glow); border: 1px solid transparent;'
              : 'background: var(--neu-bg); color: var(--neu-text-muted); font-size: 12px; box-shadow: 3px 3px 7px var(--neu-shadow-dark), -3px -3px 7px var(--neu-shadow-light); border: 1px solid var(--neu-border);'}
            onclick={() => handleTagClick(tag)}
          >
            {tag}
          </button>
        {/each}
      </div>

      <!-- Feed cards -->
      {#if feed.loading}
        <div class="flex flex-col gap-4">
          {#each Array(3) as _}
            <div
              class="rounded-[22px]"
              style="background: var(--neu-bg); box-shadow: 7px 7px 16px var(--neu-shadow-dark), -7px -7px 16px var(--neu-shadow-light); border: 1px solid var(--neu-border);"
            >
              <div
                style="aspect-ratio: 16/9; background: var(--neu-border); border-radius: 22px 22px 0 0;"
              ></div>
              <div style="padding: 12px 16px;">
                <div
                  class="mb-2 rounded"
                  style="width: 80%; height: 12px; background: var(--neu-border);"
                ></div>
                <div
                  class="rounded"
                  style="width: 60%; height: 10px; background: var(--neu-border);"
                ></div>
              </div>
            </div>
          {/each}
        </div>
      {:else if getFilteredItems().length === 0}
        <EmptyState icon={Compass} title="暂无内容" description="换个标签或关键词试试" />
      {:else}
        <div class="flex flex-col gap-4">
          {#each getFilteredItems() as moment (moment.moment_id)}
            <MomentCard {moment} />
          {/each}
        </div>
        <LoadMoreButton
          loading={feed.loadingMore}
          hasMore={feed.hasMore}
          onLoadMore={() => loadFeed(false)}
        />
      {/if}
    </div>
  </div>
{/if}
