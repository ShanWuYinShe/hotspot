<script lang="ts">
  import { authStore } from '$lib/stores/auth.svelte';
  import { uiStore } from '$lib/stores/ui.svelte';
  import { momentsService } from '$lib/api/services';
  import { mapAnchorStore } from '$lib/stores/map-anchor.svelte';
  import { platformStore } from '$lib/stores/platform.svelte';
  import { usePagination } from '$lib/hooks/use-pagination.svelte';
  import { useLocationAwareLoad } from '$lib/hooks/use-location-aware-load.svelte';
  import MomentCard from '$lib/components/moments/moment-card.svelte';
  import EmptyState from '$lib/components/common/empty-state.svelte';
  import LoadMoreButton from '$lib/components/common/load-more-button.svelte';
  import type { MomentItem } from '$lib/types/api';
  import Search from 'lucide-svelte/icons/search';
  import Image from 'lucide-svelte/icons/image';
  import AlertTriangle from 'lucide-svelte/icons/triangle-alert';
  import LogIn from 'lucide-svelte/icons/log-in';
  import TrendingUp from 'lucide-svelte/icons/trending-up';
  import Sparkles from 'lucide-svelte/icons/sparkles';
  import X from 'lucide-svelte/icons/x';
  import { t } from '$lib/i18n';

  const TABS = ['热门', '附近', '活动', '关注'];
  const TAGS = ['自然', '城市', '美食', '音乐', '活动', '摄影', '生活', '旅行'];

  const pagination = usePagination<MomentItem>({
    uniqueKey: 'moment_id'
  });

  let hasLoadedOnce = $state(false);
  let activeTab = $state('附近');
  let activeTag = $state<string | null>(null);
  let searchVal = $state('');

  function getAnchor() {
    return {
      latitude: mapAnchorStore.latitude,
      longitude: mapAnchorStore.longitude
    };
  }

  async function loadMoments(refresh = false) {
    await pagination.load(
      (cursor) =>
        momentsService.search({
          radius: mapAnchorStore.radius * 1000,
          reference_point: getAnchor(),
          pagination: { limit: 20, cursor }
        }),
      refresh
    );
    hasLoadedOnce = true;
  }

  function handleRetry() {
    void loadMoments(true);
  }

  useLocationAwareLoad((refresh) => loadMoments(refresh));
</script>

<svelte:head>
  <title>{t('discover.title')} - HotSpot</title>
</svelte:head>

{#if platformStore.isMobile}
  <!-- Mobile Discover — single-column feed, max-width 560px centred -->
  <div class="flex flex-col">
    <!-- Sticky header -->
    <div
      class="sticky top-0 z-10 px-5 pt-3 pb-0"
      style="background-color: var(--surface-bg); border-bottom: 1px solid var(--border-theme);"
    >
      <!-- Top bar: title + content count -->
      <div class="mb-2 flex items-center justify-between">
        <h1 style="font-size: 22px; color: var(--text-primary);">{t('discover.title')}</h1>
        <span style="font-size: 12px; color: var(--text-secondary);">
          {pagination.items.length} 条内容
        </span>
      </div>

      <!-- Search bar -->
      <div class="mb-2.5 flex items-center gap-3 rounded-[18px] px-4 py-2.5 input-surface">
        <Search size={15} strokeWidth={1.5} color="var(--text-secondary)" class="flex-shrink-0" />
        <input
          type="text"
          bind:value={searchVal}
          placeholder="地点、人物、关键词…"
          class="flex-1 bg-transparent outline-none"
          style="font-size: 13px; color: var(--text-primary);"
        />
        {#if searchVal.length > 0}
          <button
            class="flex-shrink-0 outline-none transition-transform duration-150 active:scale-[0.85]"
            onclick={() => {
              searchVal = '';
            }}
          >
            <X size={13} color="var(--text-secondary)" strokeWidth={2} />
          </button>
        {/if}
      </div>

      <!-- Tab bar -->
      <div
        class="relative flex overflow-x-auto no-scrollbar"
        style="border-bottom: 1px solid var(--border-theme);"
      >
        {#each TABS as tab}
          <button
            class="tab-btn relative flex-shrink-0 pb-3 pt-1 outline-none transition-transform duration-150 active:scale-[0.94]"
            style="padding-left: 18px; padding-right: 18px; color: {activeTab === tab
              ? 'var(--brand)'
              : 'var(--text-secondary)'}; font-weight: {activeTab === tab
              ? 500
              : 400}; font-size: 13px; transition: color 0.2s;"
            onclick={() => {
              activeTab = tab;
            }}
            data-tab-id={tab}
          >
            {tab}
            {#if activeTab === tab}
              <div
                style="position: absolute; bottom: -1px; left: 14%; right: 14%; height: 2px; border-radius: 1px; background: var(--brand); transition: all 0.28s cubic-bezier(0.22, 1, 0.36, 1);"
              ></div>
            {/if}
          </button>
        {/each}
      </div>
    </div>

    <!-- Content — centred column, max 560px -->
    <div
      class="w-full pb-6"
      style="max-width: 560px; margin-left: auto; margin-right: auto; padding-left: 16px; padding-right: 16px; padding-top: 12px;"
    >
      <!-- Tag chips — interactive filter -->
      <div class="mb-4 flex items-center gap-2 overflow-x-auto no-scrollbar pb-1">
        {#each TAGS as tag}
          <button
            class="flex-shrink-0 rounded-full px-3.5 py-1.5 outline-none transition-all duration-150 active:scale-[0.93]"
            class:surface-flat={activeTag !== tag}
            style={activeTag === tag
              ? 'background: var(--brand); color: white; font-size: 12px; box-shadow: 0 3px 10px var(--glow); border: 1px solid transparent;'
              : 'color: var(--text-secondary); font-size: 12px;'}
            onclick={() => {
              activeTag = activeTag === tag ? null : tag;
            }}
          >
            {tag}
          </button>
        {/each}
      </div>

      <!-- Feed cards -->
      <div class="flex flex-col gap-4">
        {#if pagination.loading}
          {#each Array(4) as _}
            <div class="surface-flat rounded-[22px] overflow-hidden">
              <div
                class="overflow-hidden rounded-t-[22px]"
                style="aspect-ratio: 16/9; background: var(--border-theme);"
              ></div>
              <div class="p-4">
                <div
                  class="mb-2 rounded"
                  style="width: 80%; height: 12px; background: var(--border-theme);"
                ></div>
                <div
                  class="rounded"
                  style="width: 60%; height: 10px; background: var(--border-theme);"
                ></div>
              </div>
            </div>
          {/each}
        {:else if pagination.error}
          {#if !authStore.isAuthenticated}
            <EmptyState
              icon={LogIn}
              title="登录后查看"
              description="登录后即可浏览附近的动态"
              actionText="立即登录"
              onAction={() => uiStore.openLoginModalWithRedirect({})}
            />
          {:else}
            <EmptyState
              icon={AlertTriangle}
              title={t('discover.loadFailed')}
              description={pagination.error}
              actionText={t('common.retry')}
              onAction={handleRetry}
            />
          {/if}
        {:else if pagination.items.length === 0 && hasLoadedOnce}
          <EmptyState
            icon={Image}
            title={t('discover.noMoments')}
            description={t('discover.noMomentsDesc')}
          />
        {:else}
          {#each pagination.items as moment (moment.moment_id)}
            <MomentCard {moment} />
          {/each}
          <LoadMoreButton
            loading={pagination.loadingMore}
            hasMore={pagination.hasMore}
            onLoadMore={() => loadMoments(false)}
          />
        {/if}
      </div>
    </div>
  </div>
{:else if platformStore.isTablet}
  <!-- Tablet Discover — 2-column feed, centred, no sidebar -->
  <div class="flex h-full flex-col">
    <!-- Sticky header -->
    <div
      class="flex-shrink-0 px-6 pt-4 pb-0"
      style="background-color: var(--surface-bg); border-bottom: 1px solid var(--border-theme);"
    >
      <!-- Top bar -->
      <div class="mb-2 flex items-center justify-between">
        <h1 style="font-size: 24px; color: var(--text-primary);">{t('discover.title')}</h1>
        <span
          class="flex items-center gap-1 rounded-full px-3 py-1.5"
          style="background: var(--brand); font-size: 11px; color: white;"
        >
          <Sparkles size={11} color="white" strokeWidth={2} />
          精选
        </span>
      </div>

      <!-- Search bar -->
      <div
        class="mb-2.5 flex items-center gap-3 rounded-[18px] px-4 py-2.5 input-surface"
        style="border: 1px solid var(--border-theme);"
      >
        <Search size={15} strokeWidth={1.5} color="var(--text-secondary)" class="flex-shrink-0" />
        <input
          type="text"
          bind:value={searchVal}
          placeholder="地点、人物、关键词…"
          class="flex-1 bg-transparent outline-none"
          style="font-size: 13px; color: var(--text-primary);"
        />
        {#if searchVal.length > 0}
          <button
            class="flex-shrink-0 outline-none transition-transform duration-150 active:scale-[0.85]"
            onclick={() => {
              searchVal = '';
            }}
          >
            <X size={13} color="var(--text-secondary)" strokeWidth={2} />
          </button>
        {/if}
      </div>

      <!-- Tab bar -->
      <div
        class="relative flex overflow-x-auto no-scrollbar"
        style="border-bottom: 1px solid var(--border-theme);"
      >
        {#each TABS as tab}
          <button
            class="relative flex-shrink-0 pb-3 pt-1 outline-none transition-transform duration-150 active:scale-[0.94]"
            style="padding-left: 18px; padding-right: 18px; color: {activeTab === tab
              ? 'var(--brand)'
              : 'var(--text-secondary)'}; font-weight: {activeTab === tab
              ? 500
              : 400}; font-size: 13px; transition: color 0.2s;"
            onclick={() => {
              activeTab = tab;
            }}
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
    </div>

    <!-- Content — 2-column grid, centred, max 800px -->
    <div class="flex-1 overflow-y-auto no-scrollbar">
      <div
        style="max-width: 800px; margin-left: auto; margin-right: auto; padding: 16px 20px 28px;"
      >
        <!-- Tag chips -->
        <div class="mb-5 flex flex-wrap items-center gap-2">
          {#each TAGS as tag}
            <button
              class="flex-shrink-0 rounded-full px-3.5 py-1.5 outline-none transition-all duration-150 active:scale-[0.93]"
              class:surface-flat={activeTag !== tag}
              style={activeTag === tag
                ? 'background: var(--brand); color: white; font-size: 12px; box-shadow: 0 3px 10px var(--glow); border: 1px solid transparent;'
                : 'color: var(--text-secondary); font-size: 12px;'}
              onclick={() => {
                activeTag = activeTag === tag ? null : tag;
              }}
            >
              {tag}
            </button>
          {/each}
        </div>

        {#if pagination.loading}
          <div style="display: grid; grid-template-columns: repeat(2, 1fr); gap: 16px;">
            {#each Array(4) as _}
              <div class="surface-flat rounded-[22px] overflow-hidden">
                <div
                  class="overflow-hidden rounded-t-[22px]"
                  style="aspect-ratio: 16/9; background: var(--border-theme);"
                ></div>
                <div class="p-3">
                  <div
                    class="mb-2 rounded"
                    style="width: 80%; height: 11px; background: var(--border-theme);"
                  ></div>
                  <div
                    class="rounded"
                    style="width: 60%; height: 9px; background: var(--border-theme);"
                  ></div>
                </div>
              </div>
            {/each}
          </div>
        {:else if pagination.error}
          {#if !authStore.isAuthenticated}
            <EmptyState
              icon={LogIn}
              title="登录后查看"
              description="登录后即可浏览附近的动态"
              actionText="立即登录"
              onAction={() => uiStore.openLoginModalWithRedirect({})}
            />
          {:else}
            <EmptyState
              icon={AlertTriangle}
              title={t('discover.loadFailed')}
              description={pagination.error}
              actionText={t('common.retry')}
              onAction={handleRetry}
            />
          {/if}
        {:else if pagination.items.length === 0 && hasLoadedOnce}
          <EmptyState
            icon={Image}
            title={t('discover.noMoments')}
            description={t('discover.noMomentsDesc')}
          />
        {:else}
          <div style="display: grid; grid-template-columns: repeat(2, 1fr); gap: 16px;">
            {#each pagination.items as moment (moment.moment_id)}
              <MomentCard {moment} />
            {/each}
          </div>
          <LoadMoreButton
            loading={pagination.loadingMore}
            hasMore={pagination.hasMore}
            onLoadMore={() => loadMoments(false)}
          />
        {/if}
      </div>
    </div>
  </div>
{:else}
  <!-- Desktop Discover — 2-column feed + 220px sidebar -->
  <div class="flex h-full flex-col">
    <!-- Sticky header -->
    <div
      class="flex-shrink-0 px-6 pt-5 pb-0"
      style="background-color: var(--surface-bg); border-bottom: 1px solid var(--border-theme);"
    >
      <!-- Top bar -->
      <div class="mb-2 flex items-center justify-between">
        <h1 style="font-size: 20px; color: var(--text-primary);">
          {t('discover.title')}
        </h1>
        <span style="font-size: 12px; color: var(--text-secondary);">
          {pagination.items.length} 条内容
        </span>
      </div>

      <!-- Search bar -->
      <div
        class="mb-2.5 flex items-center gap-3 rounded-[18px] px-4 py-2.5 input-surface"
        style="border: 1px solid var(--border-theme);"
      >
        <Search size={15} strokeWidth={1.5} color="var(--text-secondary)" class="flex-shrink-0" />
        <input
          type="text"
          bind:value={searchVal}
          placeholder="地点、人物、关键词…"
          class="flex-1 bg-transparent outline-none"
          style="font-size: 13px; color: var(--text-primary);"
        />
        {#if searchVal.length > 0}
          <button
            class="flex-shrink-0 outline-none transition-transform duration-150 active:scale-[0.85]"
            onclick={() => {
              searchVal = '';
            }}
          >
            <X size={13} color="var(--text-secondary)" strokeWidth={2} />
          </button>
        {/if}
      </div>

      <!-- Tab bar -->
      <div
        class="relative flex overflow-x-auto no-scrollbar"
        style="border-bottom: 1px solid var(--border-theme);"
      >
        {#each TABS as tab}
          <button
            class="relative flex-shrink-0 pb-3 pt-1 outline-none transition-transform duration-150 active:scale-[0.94]"
            style="padding-left: 18px; padding-right: 18px; color: {activeTab === tab
              ? 'var(--brand)'
              : 'var(--text-secondary)'}; font-weight: {activeTab === tab
              ? 500
              : 400}; font-size: 13px; transition: color 0.2s;"
            onclick={() => {
              activeTab = tab;
            }}
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
    </div>

    <!-- Body: main feed + right sidebar -->
    <div class="flex flex-1 min-h-0 overflow-hidden">
      <!-- Main feed -->
      <div class="flex-1 overflow-y-auto no-scrollbar">
        <div style="padding: 20px 24px 28px;">
          <!-- Tag chips -->
          <div class="mb-5 flex flex-wrap items-center gap-2">
            {#each TAGS as tag}
              <button
                class="flex-shrink-0 rounded-full px-3.5 py-1.5 outline-none transition-all duration-150 active:scale-[0.93]"
                class:surface-flat={activeTag !== tag}
                style={activeTag === tag
                  ? 'background: var(--brand); color: white; font-size: 12px; box-shadow: 0 3px 10px var(--glow); border: 1px solid transparent;'
                  : 'color: var(--text-secondary); font-size: 12px;'}
                onclick={() => {
                  activeTag = activeTag === tag ? null : tag;
                }}
              >
                {tag}
              </button>
            {/each}
          </div>

          {#if pagination.loading}
            <div
              style="display: grid; grid-template-columns: repeat(2, 1fr); gap: 16px; max-width: 800px;"
            >
              {#each Array(6) as _}
                <div class="surface-flat rounded-[22px] overflow-hidden">
                  <div
                    class="overflow-hidden rounded-t-[22px]"
                    style="aspect-ratio: 16/9; background: var(--border-theme);"
                  ></div>
                  <div class="p-3">
                    <div
                      class="mb-2 rounded"
                      style="width: 80%; height: 11px; background: var(--border-theme);"
                    ></div>
                    <div
                      class="rounded"
                      style="width: 60%; height: 9px; background: var(--border-theme);"
                    ></div>
                  </div>
                </div>
              {/each}
            </div>
          {:else if pagination.error}
            {#if !authStore.isAuthenticated}
              <EmptyState
                icon={LogIn}
                title="登录后查看"
                description="登录后即可浏览附近的动态"
                actionText="立即登录"
                onAction={() => uiStore.openLoginModalWithRedirect({})}
              />
            {:else}
              <EmptyState
                icon={AlertTriangle}
                title={t('discover.loadFailed')}
                description={pagination.error}
                actionText={t('common.retry')}
                onAction={handleRetry}
              />
            {/if}
          {:else if pagination.items.length === 0 && hasLoadedOnce}
            <EmptyState
              icon={Image}
              title={t('discover.noMoments')}
              description={t('discover.noMomentsDesc')}
            />
          {:else}
            <div
              style="display: grid; grid-template-columns: repeat(2, 1fr); gap: 16px; max-width: 800px;"
            >
              {#each pagination.items as moment (moment.moment_id)}
                <MomentCard {moment} />
              {/each}
            </div>
            <LoadMoreButton
              loading={pagination.loadingMore}
              hasMore={pagination.hasMore}
              onLoadMore={() => loadMoments(false)}
            />
          {/if}
        </div>
      </div>

      <!-- Right sidebar (220px) -->
      <div
        class="flex-shrink-0 overflow-y-auto no-scrollbar"
        style="width: 220px; border-left: 1px solid var(--border-theme); padding: 20px 16px;"
      >
        <!-- Trending topics -->
        <div class="mb-6">
          <div class="mb-3 flex items-center gap-1.5">
            <TrendingUp size={13} color="var(--brand)" strokeWidth={2} />
            <p
              class="uppercase"
              style="font-size: 10px; letter-spacing: 0.12em; color: var(--text-secondary);"
            >
              热门话题
            </p>
          </div>
          {#each ['#城市探索', '#周末去哪', '#隐藏咖啡馆', '#街头摄影', '#本地美食'] as tag, i}
            <button
              class="mb-1 flex w-full items-center justify-between rounded-[12px] px-3 py-2.5 outline-none"
              style="transition: background 0.15s;"
            >
              <span style="font-size: 13px; color: var(--text-primary);">{tag}</span>
              <span style="font-size: 11px; color: var(--text-secondary);"
                >{[89, 64, 52, 48, 41][i]}帖</span
              >
            </button>
          {/each}
        </div>

        <!-- Divider -->
        <div style="height: 1px; background: var(--border-theme); margin-bottom: 20px;"></div>

        <!-- Active users -->
        <div>
          <p
            class="mb-3 uppercase"
            style="font-size: 10px; letter-spacing: 0.12em; color: var(--text-secondary);"
          >
            活跃用户
          </p>
          {#if pagination.items.length > 0}
            {#each pagination.items.slice(0, 4) as item, i}
              <div class="mb-3.5 flex items-center gap-2.5">
                <div
                  class="flex-shrink-0 overflow-hidden rounded-full"
                  style="width: 32px; height: 32px; background: var(--border-theme);"
                >
                  {#if item.user.avatar_url}
                    <img
                      src={item.user.avatar_url}
                      alt={item.user.nickname}
                      class="h-full w-full object-cover"
                    />
                  {/if}
                </div>
                <div class="min-w-0 flex-1">
                  <p class="truncate" style="font-size: 12px; color: var(--text-primary);">
                    {item.user.nickname}
                  </p>
                  <p class="truncate" style="font-size: 10px; color: var(--text-secondary);">
                    {['徐州 · 云龙区', '徐州 · 鼓楼区', '徐州 · 泉山区'][i % 3]}
                  </p>
                </div>
                <button
                  class="flex-shrink-0 rounded-full px-2.5 py-1 outline-none"
                  style="background: var(--brand); font-size: 10px; color: white; box-shadow: 0 2px 6px var(--glow);"
                >
                  关注
                </button>
              </div>
            {/each}
          {/if}
        </div>
      </div>
    </div>
  </div>
{/if}
