<script lang="ts">
  import { page } from '$app/state';
  import { goto } from '$app/navigation';
  import { t } from '$lib/i18n';
  import { usersService } from '$lib/api/services';
  import { usePagination } from '$lib/hooks/use-pagination.svelte';
  import UserCard from '$lib/components/common/user-card.svelte';
  import ListSkeleton from '$lib/components/common/list-skeleton.svelte';
  import EmptyState from '$lib/components/common/empty-state.svelte';
  import LoadMoreButton from '$lib/components/common/load-more-button.svelte';
  import type { FriendItem, FollowingItem, FollowerItem } from '$lib/types/api';
  import { PAGE_SIZE } from '$lib/config/pagination';
  import RefreshCw from 'lucide-svelte/icons/refresh-cw';
  import Heart from 'lucide-svelte/icons/heart';
  import UserCheck from 'lucide-svelte/icons/user-check';
  import UserPlus from 'lucide-svelte/icons/user-plus';
  import AlertTriangle from 'lucide-svelte/icons/triangle-alert';

  type RelTab = 'friends' | 'following' | 'followers';

  interface TabMeta {
    label: string;
    subtitle: string;
    emptyIcon: typeof Heart;
    emptyTitle: string;
  }

  const TAB_ORDER: RelTab[] = ['friends', 'following', 'followers'];
  // $derived 以便 t() 在 locale 切换时响应式更新
  const TAB_META = $derived<Record<RelTab, TabMeta>>({
    friends: {
      label: t('relations.friends'),
      subtitle: t('relations.friendsSubtitle'),
      emptyIcon: Heart,
      emptyTitle: t('relations.noFriends')
    },
    following: {
      label: t('relations.following'),
      subtitle: t('relations.followingSubtitle'),
      emptyIcon: UserCheck,
      emptyTitle: t('relations.noFollowing')
    },
    followers: {
      label: t('relations.followers'),
      subtitle: t('relations.followersSubtitle'),
      emptyIcon: UserPlus,
      emptyTitle: t('relations.noFollowers')
    }
  });

  function parseTab(value: string | null): RelTab {
    return value === 'following' || value === 'followers' ? value : 'friends';
  }

  // 单一事实源 = URL 查询参数，支持深链与浏览器前进/后退。
  const activeTab = $derived(parseTab(page.url.searchParams.get('tab')));
  const activeMeta = $derived(TAB_META[activeTab]);

  const friendsPg = usePagination<FriendItem>({ uniqueKey: (item) => item.user.user_id });
  const followingPg = usePagination<FollowingItem>({ uniqueKey: (item) => item.user.user_id });
  const followersPg = usePagination<FollowerItem>({ uniqueKey: (item) => item.user.user_id });

  const activePg = $derived(
    activeTab === 'friends' ? friendsPg : activeTab === 'following' ? followingPg : followersPg
  );

  const loaded: Record<RelTab, boolean> = { friends: false, following: false, followers: false };

  async function loadTab(tab: RelTab, refresh = false) {
    if (tab === 'friends') {
      await friendsPg.load(
        (cursor) => usersService.getFriends({ pagination: { limit: PAGE_SIZE.relations, cursor } }),
        refresh
      );
    } else if (tab === 'following') {
      await followingPg.load(
        (cursor) => usersService.getFollowing({ pagination: { limit: PAGE_SIZE.relations, cursor } }),
        refresh
      );
    } else {
      await followersPg.load(
        (cursor) => usersService.getFollowers({ pagination: { limit: PAGE_SIZE.relations, cursor } }),
        refresh
      );
    }
    loaded[tab] = true;
  }

  function selectTab(tab: RelTab) {
    if (tab === activeTab) return;
    void goto(`/relations?tab=${tab}`, { replaceState: true, keepFocus: true, noScroll: true });
  }

  // 单一加载入口：挂载时拉取初始 tab，切到未加载的 tab 时按需拉取；
  // 回到已加载的 tab 保留缓存与滚动位置。URL（activeTab）是唯一触发源。
  $effect(() => {
    const tab = activeTab;
    if (!loaded[tab]) void loadTab(tab, true);
  });
</script>

<svelte:head>
  <title>{t('relations.browserTitle')}</title>
</svelte:head>

<div class="flex flex-col">
  <!-- Sticky header + tabs -->
  <div
    class="sticky top-0 z-10 page-pad-x pt-3 pb-3"
    style="background-color: var(--surface-bg); border-bottom: 1px solid var(--border-theme);"
  >
    <div class="page-shell">
      <div class="flex items-center justify-between">
        <div>
          <h1 class="page-title" style="color: var(--text-primary);">{t('relations.title')}</h1>
          <p class="text-meta" style="color: var(--text-secondary); margin-top: 2px">
            {activeMeta.subtitle}
          </p>
        </div>
        <button
          class="text-meta flex items-center gap-1.5 rounded-full px-3 py-1.5 outline-none"
          style="background: var(--surface-bg); box-shadow: var(--shadow-md); border: 1px solid var(--border-theme); color: var(--text-secondary)"
          onclick={() => loadTab(activeTab, true)}
          disabled={activePg.loading}
        >
          <RefreshCw size={13} class={activePg.loading ? 'animate-spin' : ''} />
          {t('common.refresh')}
        </button>
      </div>
      <div class="mt-3 flex items-center gap-2">
        {#each TAB_ORDER as tab (tab)}
          <button
            type="button"
            class="filter-chip"
            class:filter-chip--active={activeTab === tab}
            onclick={() => selectTab(tab)}
          >
            {TAB_META[tab].label}
          </button>
        {/each}
      </div>
    </div>
  </div>

  <div class="flex flex-col page-pad">
    <div class="page-shell flex flex-col gap-5">
      {#if activePg.loading}
        <ListSkeleton type="user" count={6} />
      {:else if activePg.error}
        <EmptyState
          icon={AlertTriangle}
          title={t('relations.loadFailed')}
          description={activePg.error}
          actionText={t('common.retry')}
          onAction={() => loadTab(activeTab, true)}
        />
      {:else if activePg.items.length === 0}
        <EmptyState icon={activeMeta.emptyIcon} title={activeMeta.emptyTitle} fill />
      {:else}
        <div class="density-grid density-grid--compact">
          {#each activePg.items as item (item.user.user_id)}
            <UserCard user={item.user} showBio showTags variant="square" />
          {/each}
        </div>
        <LoadMoreButton
          loading={activePg.loadingMore}
          hasMore={activePg.hasMore}
          onLoadMore={() => loadTab(activeTab, false)}
        />
      {/if}
    </div>
  </div>
</div>
