<script lang="ts">
  import { page } from '$app/state';
  import { authStore } from '$lib/stores/auth.svelte';
  import { uiStore } from '$lib/stores/ui.svelte';
  import { usersService } from '$lib/api/services';
  import { navigateToNearbyBackTarget } from '../../nearby-back';
  import { mapAnchorStore } from '$lib/stores/map-anchor.svelte';
  import { usePagination } from '$lib/hooks/use-pagination.svelte';
  import { useLocationAwareLoad } from '$lib/hooks/use-location-aware-load.svelte';
  import { presenceStore } from '$lib/stores/presence.svelte';
  import UserCard from '$lib/components/common/user-card.svelte';
  import EmptyState from '$lib/components/common/empty-state.svelte';
  import LoadMoreButton from '$lib/components/common/load-more-button.svelte';
  import type { UserItem } from '$lib/types/api';
  import ArrowLeft from 'lucide-svelte/icons/arrow-left';
  import MapIcon from 'lucide-svelte/icons/map';
  import Search from 'lucide-svelte/icons/search';
  import Users from 'lucide-svelte/icons/users';
  import AlertTriangle from 'lucide-svelte/icons/triangle-alert';
  import LogIn from 'lucide-svelte/icons/log-in';

  const SORT_OPTIONS = ['距离', '在线', '共同好友'] as const;
  const FILTER_TAGS = ['全部', '摄影', '旅行', '美食', '户外', '城市', '咖啡', '音乐', '运动'];

  const pagination = usePagination<UserItem>({
    uniqueKey: (item) => item.user.user_id
  });

  let hasLoadedOnce = $state(false);
  let searchText = $state('');
  let activeSort = $state<string>('距离');
  let activeTag = $state<string>('全部');

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
    class="sticky top-0 z-10 px-5 pt-3 pb-0"
    style="background-color: var(--surface-bg); border-bottom: 1px solid var(--border-theme);"
  >
    <div class="mb-3 flex items-center gap-3">
      <button
        class="surface-flat flex items-center justify-center rounded-full outline-none"
        style="width: 36px; height: 36px;"
        onclick={handleBack}
        aria-label="返回"
      >
        <ArrowLeft size={15} strokeWidth={1.5} color="var(--text-secondary)" />
      </button>
      <div class="flex-1">
        <h1 style="font-size: 18px; font-weight: 600; color: var(--text-primary);">附近的人</h1>
        <p style="font-size: 11px; color: var(--text-secondary); margin-top: 1px;">
          {onlineCount} 人在线 · {pagination.items.length} 人附近
        </p>
      </div>
      <button
        class="surface-flat flex items-center justify-center rounded-full outline-none"
        style="width: 36px; height: 36px;"
        aria-label="地图"
      >
        <MapIcon size={15} strokeWidth={1.5} color="var(--brand)" />
      </button>
    </div>

    <!-- Search -->
    <div class="input-surface mb-3 flex items-center gap-3 rounded-[18px] px-4 py-2.5">
      <Search size={14} strokeWidth={1.5} color="var(--text-secondary)" class="flex-shrink-0" />
      <input
        type="text"
        bind:value={searchText}
        placeholder="搜索附近用户"
        class="flex-1 bg-transparent outline-none"
        style="font-size: 13px; color: var(--text-primary);"
      />
    </div>

    <!-- Sort tabs + Filter tags -->
    <div class="no-scrollbar mb-3 flex gap-2 overflow-x-auto">
      {#each SORT_OPTIONS as s (s)}
        <button
          class="flex-shrink-0 rounded-full px-3.5 py-1.5 outline-none transition-all"
          class:surface-flat={activeSort !== s}
          style={activeSort === s
            ? 'background: var(--brand); color: white; font-size: 12px; box-shadow: 0 3px 10px var(--glow);'
            : 'color: var(--text-secondary); font-size: 12px;'}
          onclick={() => (activeSort = s)}
        >
          {s}
        </button>
      {/each}
      <div style="width: 1px; height: 28px; background: var(--border-theme); margin: auto;"></div>
      {#each FILTER_TAGS as t (t)}
        <button
          class="flex-shrink-0 rounded-full px-3 py-1.5 outline-none transition-all"
          style={activeTag === t && t !== '全部'
            ? 'background: rgba(99,102,241,0.12); color: var(--brand); font-size: 12px; border: 1px solid rgba(99,102,241,0.25);'
            : 'font-size: 12px; color: var(--text-secondary); background: transparent;'}
          onclick={() => (activeTag = t)}
        >
          {t}
        </button>
      {/each}
    </div>
  </div>

  <!-- List -->
  <div class="w-full px-5 pt-3 pb-6">
    {#if pagination.loading}
      <div
        class="flex flex-col gap-2.5"
        style="max-width: 640px; margin-left: auto; margin-right: auto;"
      >
        {#each Array(6) as _, i (i)}
          <div
            class="animate-pulse rounded-[22px]"
            style="height: 72px; background: var(--border-theme);"
          ></div>
        {/each}
      </div>
    {:else if pagination.error}
      {#if !authStore.isAuthenticated}
        <EmptyState
          icon={LogIn}
          title="登录后查看"
          description="登录后即可浏览附近的人"
          actionText="立即登录"
          onAction={() => uiStore.openLoginModalWithRedirect({})}
        />
      {:else}
        <EmptyState
          icon={AlertTriangle}
          title="加载失败"
          description={pagination.error}
          actionText="重试"
          onAction={handleRetry}
        />
      {/if}
    {:else if pagination.items.length === 0 && hasLoadedOnce}
      <EmptyState
        icon={Users}
        title="附近暂无其他用户"
        description="换个浏览位置或稍后再来看看。"
      />
    {:else}
      <div
        class="flex flex-col gap-2.5"
        style="max-width: 640px; margin-left: auto; margin-right: auto;"
      >
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
