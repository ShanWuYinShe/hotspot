<script lang="ts">
  import { authStore } from '$lib/stores/auth.svelte';
  import { mapAnchorStore } from '$lib/stores/map-anchor.svelte';
  import { toast } from '$lib/stores/toast.svelte';
  import { t } from '$lib/i18n';
  import { formatDistance } from '$lib/utils/format';
  import { usersService, feedService, groupsService } from '$lib/api/services';
  import { isSuccess } from '$lib/api/client';
  import type { UserItem, TrendingTagItem, GroupItem } from '$lib/types/api';
  import ChevronRight from 'lucide-svelte/icons/chevron-right';
  import Users from 'lucide-svelte/icons/users';
  import { goto } from '$app/navigation';

  import HeroStats from '$lib/components/home/hero-stats.svelte';
  import PeopleRadarStrip from '$lib/components/home/people-radar-strip.svelte';
  import ActivityList from '$lib/components/home/activity-list.svelte';
  import NearbyUsersFlatList from '$lib/components/home/nearby-users-flat-list.svelte';
  import GroupCarousel from '$lib/components/home/group-carousel.svelte';
  import LocationPickerModal from '$lib/components/common/location-picker-modal.svelte';
  import AnchorEntry from '$lib/components/common/anchor-entry.svelte';
  import EmptyState from '$lib/components/common/empty-state.svelte';
  import OnlineDot from '$lib/components/common/online-dot.svelte';
  import MapCard from '$lib/components/common/map-card.svelte';

  import { platformStore } from '$lib/stores/platform.svelte';

  let refreshTrigger = $state(0);
  let showLocationPicker = $state(false);
  let lastAuthState = $state<boolean | null>(null);
  let lastAnchorUpdatedAt = $state<number | null>(null);
  let locationDebounceTimer: ReturnType<typeof setTimeout> | null = null;

  // Nearby users
  let nearbyUsers = $state.raw<UserItem[]>([]);
  let trendingTags: TrendingTagItem[] = $state([]);
  let nearbyGroups = $state.raw<GroupItem[]>([]);
  let loadingNearby = $state(false);

  const onlineCount = $derived(nearbyUsers.filter((u) => u.presence_status === 'online').length);
  const activityCount = $derived(trendingTags.reduce((sum, t) => sum + t.usage_count, 0));

  const radarUsers = $derived(
    nearbyUsers.map((u) => ({
      id: u.user.user_id,
      avatar_url: u.user.avatar_url || '',
      nickname: u.user.nickname,
      distance: u.location?.distance != null ? formatDistance(u.location.distance) : undefined,
      online: u.presence_status === 'online'
    }))
  );

  const flatUsers = $derived(
    nearbyUsers.map((u) => ({
      id: u.user.user_id,
      avatar_url: u.user.avatar_url || '',
      nickname: u.user.nickname,
      distance: u.location?.distance != null ? formatDistance(u.location.distance) : undefined,
      online: u.presence_status === 'online',
      tag: u.tags?.[0]
    }))
  );

  const carouselGroups = $derived(
    nearbyGroups.map((g) => ({
      id: g.group.group_id,
      name: g.group.name,
      banner_url: g.group.avatar_url,
      member_count: g.group.members_count,
      distance: g.location?.distance != null ? formatDistance(g.location.distance) : undefined,
      is_private: g.has_password || g.join_method === 'invite_only'
    }))
  );

  const ACTIVITY_STYLES = [
    { icon: 'coffee', color: 'var(--semantic-warning)' },
    { icon: 'music', color: 'var(--brand)' },
    { icon: 'users', color: 'var(--map-group)' },
    { icon: 'star', color: 'var(--semantic-hot)' }
  ];

  const activities = $derived(
    trendingTags.slice(0, 4).map((tag, i) => {
      const style = ACTIVITY_STYLES[i % ACTIVITY_STYLES.length] ?? {
        icon: 'star',
        color: 'var(--brand)'
      };
      return {
        id: tag.name,
        icon: style.icon,
        title: `#${tag.name}`,
        sub: `${tag.usage_count}${t('home.posts')}`,
        color: style.color,
        bg: `color-mix(in srgb, ${style.color} 12%, transparent)`
      };
    })
  );

  function handleActivitySelect(id: string) {
    void goto(`/discover?tag=${encodeURIComponent(id)}`);
  }

  function getAnchor() {
    return {
      latitude: mapAnchorStore.latitude,
      longitude: mapAnchorStore.longitude
    };
  }

  async function loadNearbyUsers() {
    loadingNearby = true;
    try {
      const response = await usersService.search({
        radius: mapAnchorStore.radius * 1000,
        reference_point: getAnchor(),
        pagination: { limit: 10 }
      });
      if (isSuccess(response) && response.data) {
        const currentUserId = authStore.currentUserId;
        const items = response.data.items ?? [];
        nearbyUsers = currentUserId
          ? items.filter((item) => item.user.user_id !== currentUserId)
          : items;
      }
    } catch {
      toast.error('加载失败，请稍后重试');
    } finally {
      loadingNearby = false;
    }
  }

  async function loadTrending() {
    try {
      const result = await feedService.getTrending();
      if (isSuccess(result)) {
        trendingTags = result.data?.trending_tags ?? [];
      }
    } catch {
      toast.error('加载失败，请稍后重试');
    }
  }

  async function loadNearbyGroups() {
    try {
      const response = await groupsService.search({
        radius: mapAnchorStore.radius * 1000,
        reference_point: getAnchor(),
        sort_by: 'distance',
        pagination: { limit: 8 }
      });
      if (isSuccess(response) && response.data) {
        nearbyGroups = response.data.items ?? [];
      }
    } catch {
      toast.error('加载失败，请稍后重试');
    }
  }

  async function refreshHome(
    options: {
      refreshWidgets?: boolean;
    } = {}
  ) {
    const { refreshWidgets = true } = options;
    if (!refreshWidgets) return;

    if (authStore.isAuthenticated) {
      refreshTrigger++;
    }

    await loadNearbyUsers();
    await loadTrending();
    await loadNearbyGroups();
  }

  function refreshAll() {
    void refreshHome();
  }

  $effect(() => {
    const isAuthed = authStore.isAuthenticated;
    if (isAuthed === lastAuthState) return;
    lastAuthState = isAuthed;
    lastAnchorUpdatedAt = null;
    void refreshHome({ refreshWidgets: true });
  });

  $effect(() => {
    const anchorUpdatedAt = mapAnchorStore.anchor.updatedAt;
    if (lastAnchorUpdatedAt === null) {
      lastAnchorUpdatedAt = anchorUpdatedAt;
      return;
    }
    if (anchorUpdatedAt === lastAnchorUpdatedAt) return;
    lastAnchorUpdatedAt = anchorUpdatedAt;
    if (locationDebounceTimer) clearTimeout(locationDebounceTimer);
    locationDebounceTimer = setTimeout(() => {
      void refreshHome({ refreshWidgets: true });
    }, 500);
    return () => {
      if (locationDebounceTimer) {
        clearTimeout(locationDebounceTimer);
        locationDebounceTimer = null;
      }
    };
  });
</script>

<svelte:head>
  <title>{t('home.title')}</title>
</svelte:head>

{#if !platformStore.useMasterDetail}
  <!-- Narrow (mobile + portrait/small tablet): centered single column
       宽屏改用双栏工作区编排（见 {:else}）。信息架构级差异，容器查询无法消解。 -->
  <div class="flex flex-col">
    <!-- Sticky header -->
    <div
      class="sticky top-0 z-10 page-pad-x pt-3 pb-3"
      style="background-color: var(--surface-bg); border-bottom: 1px solid var(--border-theme);"
    >
      <div class="page-shell page-shell--narrow flex items-center justify-between">
        <div>
          <h1 class="page-title page-title--hero" style="color: var(--text-primary);">
            {t('sidebar.home')}
          </h1>
          <p class="text-meta" style="color: var(--text-secondary); margin-top: 2px">
            {t('home.discoverNearbyPeopleAndThings')}
          </p>
        </div>
        <AnchorEntry variant="pill" onAdjust={() => (showLocationPicker = true)} />
      </div>
    </div>

    <div class="page-pad-x page-shell page-shell--narrow flex flex-col pt-4 pb-6">
      <!-- Map Card -->
      <MapCard class="mb-5" />

      <!-- Nearby people -->
      <div class="mb-5">
        <div class="mb-3 flex items-center justify-between">
          <span class="eyebrow">{t('sidebar.nearbyPeople')}</span>
          <a
            href="/users/nearby"
            class="text-meta flex items-center gap-0.5 outline-none"
            style="color: var(--brand)"
          >
            {t('common.viewAll')}
            <ChevronRight size={13} strokeWidth={1.5} />
          </a>
        </div>
        {#if loadingNearby || radarUsers.length > 0}
          <PeopleRadarStrip
            users={radarUsers}
            loading={loadingNearby}
            showHeader={false}
            onViewAll={() => void goto('/users/nearby')}
          />
          <div class="mt-3">
            <NearbyUsersFlatList users={flatUsers.slice(0, 5)} loading={loadingNearby} />
          </div>
        {:else}
          <p class="text-meta" style="color: var(--text-secondary)">
            附近暂无其他用户，可稍后再来或调整浏览位置。
          </p>
        {/if}
      </div>

      <!-- Nearby activity -->
      {#if loadingNearby || activities.length > 0}
        <div class="mb-5">
          <div class="mb-3 flex items-center justify-between">
            <span class="eyebrow">{t('home.nearbyActivity')}</span>
            <button
              class="text-meta flex items-center gap-0.5 outline-none"
              style="color: var(--brand); background: none; border: none; cursor: pointer"
              onclick={() => void goto('/discover')}
            >
              {t('common.viewAll')}
              <ChevronRight size={13} strokeWidth={1.5} />
            </button>
          </div>
          <ActivityList {activities} loading={loadingNearby} onSelect={handleActivitySelect} />
        </div>
      {/if}

      <!-- Nearby groups -->
      <GroupCarousel groups={carouselGroups} loading={loadingNearby} />
    </div>
  </div>
{:else}
  <!-- Wide (landscape/large tablet + desktop): 2-column -->
  <div class="flex h-full flex-col">
    <!-- Sticky header -->
    <div
      class="sticky top-0 z-10 page-pad-x pt-3 pb-3"
      style="background-color: var(--surface-bg); border-bottom: 1px solid var(--border-theme);"
    >
      <div class="page-shell page-shell--wide flex items-center justify-between">
        <div>
          <h1 class="page-title page-title--hero" style="color: var(--text-primary);">
            {t('sidebar.home')}
          </h1>
          <p class="text-meta" style="color: var(--text-secondary); margin-top: 2px">
            {t('home.discoverNearbyPeopleAndThings')}
          </p>
        </div>
        <div class="flex items-center gap-2">
          <button
            class="surface-flat flex items-center gap-1.5 rounded-full py-1.5 px-3.5 outline-none"
            onclick={refreshAll}
          >
            <OnlineDot variant="glow" size="sm" />
            <span class="text-caption" style="color: var(--text-secondary)"
              >{onlineCount} {t('common.online')}</span
            >
          </button>
        </div>
      </div>
    </div>

    <div class="flex min-h-0 flex-1">
      <!-- Left column — overview: 概览/次路径（实时统计 + 群组） -->
      <div
        class="flex flex-col gap-6 overflow-y-auto no-scrollbar p-6"
        style="width: 340px; flex-shrink: 0; border-right: 1px solid var(--border-theme);"
      >
        <HeroStats
          people={nearbyUsers.length}
          activity={activityCount}
          online={onlineCount}
          title={t('home.realtimeNearby')}
          rangeLabel={t('home.rangeWithin')}
          loading={loadingNearby}
          onclick={() => (showLocationPicker = true)}
        />

        <!-- Nearby groups -->
        <GroupCarousel groups={carouselGroups} loading={loadingNearby} />
      </div>

      <!-- Right column — main feed: 主路径（附近的人 + 动态） -->
      <div class="flex flex-1 flex-col gap-6 overflow-y-auto no-scrollbar p-6">
        <!-- People radar -->
        {#if loadingNearby || radarUsers.length > 0}
          <div>
            <div class="mb-3 flex items-start justify-between">
              <div>
                <span class="eyebrow">{t('sidebar.nearbyPeople')}</span>
                <p class="text-caption" style="color: var(--text-secondary); margin-top: 3px">
                  {t('home.rangeWithin')}
                </p>
              </div>
              <a
                href="/users/nearby"
                class="text-meta flex items-center gap-0.5 outline-none"
                style="color: var(--brand)"
              >
                {t('common.viewAll')}
                <ChevronRight size={13} strokeWidth={1.5} />
              </a>
            </div>
            <PeopleRadarStrip
              users={radarUsers}
              loading={loadingNearby}
              showHeader={false}
              onViewAll={() => void goto('/users/nearby')}
            />
            <div class="mt-4">
              <NearbyUsersFlatList users={flatUsers} loading={loadingNearby} />
            </div>
          </div>
        {:else}
          <div class="flex flex-1 items-center justify-center">
            <EmptyState
              icon={Users}
              title="附近暂无其他用户"
              description="换个浏览位置或稍后再来看看。"
              actionText={t('common.viewAll')}
              onAction={() => void goto('/users/nearby')}
              class="w-full"
            />
          </div>
        {/if}

        <!-- Nearby activity (主路径右栏：与"人"组成主消费流) -->
        {#if loadingNearby || activities.length > 0}
          <div>
            <div class="mb-3 flex items-center justify-between">
              <span class="eyebrow">{t('home.nearbyActivity')}</span>
              <button
                class="text-meta flex items-center gap-0.5 outline-none"
                style="color: var(--brand); background: none; border: none; cursor: pointer"
                onclick={() => void goto('/discover')}
              >
                {t('common.viewAll')}
                <ChevronRight size={13} strokeWidth={1.5} />
              </button>
            </div>
            <ActivityList {activities} loading={loadingNearby} onSelect={handleActivitySelect} />
          </div>
        {/if}
      </div>
    </div>
  </div>
{/if}

<LocationPickerModal bind:open={showLocationPicker} />
