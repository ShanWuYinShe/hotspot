<script lang="ts">
  import { authStore } from '$lib/stores/auth.svelte';
  import { uiStore } from '$lib/stores/ui.svelte';
  import { mapAnchorStore } from '$lib/stores/map-anchor.svelte';
  import { t } from '$lib/i18n';
  import { usersService, feedService, groupsService } from '$lib/api/services';
  import { isSuccess } from '$lib/api/client';
  import type { UserItem, TrendingTagItem, GroupItem } from '$lib/types/api';
  import MapPin from 'lucide-svelte/icons/map-pin';
  import Users from 'lucide-svelte/icons/users';
  import Compass from 'lucide-svelte/icons/compass';
  import ChevronRight from 'lucide-svelte/icons/chevron-right';
  import Sparkles from 'lucide-svelte/icons/sparkles';
  import Star from 'lucide-svelte/icons/star';
  import Navigation2 from 'lucide-svelte/icons/navigation-2';
  import { goto } from '$app/navigation';

  import HeroStats from '$lib/components/home/hero-stats.svelte';
  import PeopleRadarStrip from '$lib/components/home/people-radar-strip.svelte';
  import ActivityList from '$lib/components/home/activity-list.svelte';
  import NearbyUsersFlatList from '$lib/components/home/nearby-users-flat-list.svelte';
  import GroupCarousel from '$lib/components/home/group-carousel.svelte';

  type HomeMode = 'guest' | 'explore' | 'authenticated';

  import { platformStore } from '$lib/stores/platform.svelte';

  let refreshTrigger = $state(0);
  let showExplore = $state(false);
  let lastMode = $state<HomeMode>('guest');
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

  const followingIds = $derived(
    new Set(nearbyUsers.filter((u) => u.relation?.is_following).map((u) => u.user.user_id))
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
    { icon: 'coffee', color: 'var(--color-semantic-warning, #b8843f)' },
    { icon: 'music', color: 'var(--brand)' },
    { icon: 'users', color: 'var(--color-map-group, #6d83b8)' },
    { icon: 'star', color: 'var(--color-semantic-hot, #c2603f)' }
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

  function handleFlatFollow(id: string) {
    const user = nearbyUsers.find((u) => u.user.user_id === id);
    if (user) handleFollowToggle(user);
  }

  function getAnchor() {
    return {
      latitude: mapAnchorStore.latitude,
      longitude: mapAnchorStore.longitude
    };
  }

  function getHomeMode(): HomeMode {
    if (authStore.isAuthenticated) return 'authenticated';
    if (showExplore) return 'explore';
    return 'guest';
  }

  function formatDistance(meters?: number): string {
    if (meters == null) return '';
    if (meters < 1000) return `${Math.round(meters)}m`;
    return `${(meters / 1000).toFixed(1)}km`;
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
      // ignore
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
      // non-critical, silently fail
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
      // non-critical, silently fail
    }
  }

  async function refreshHome(
    options: {
      refreshWidgets?: boolean;
    } = {}
  ) {
    const { refreshWidgets = authStore.isAuthenticated } = options;

    if (refreshWidgets && authStore.isAuthenticated) {
      refreshTrigger++;
    }

    if (refreshWidgets && authStore.isAuthenticated) {
      await loadNearbyUsers();
      await loadTrending();
      await loadNearbyGroups();
    } else if (refreshWidgets) {
      await loadTrending();
      await loadNearbyGroups();
    }
  }

  function refreshAll() {
    void refreshHome();
  }

  function handleGetStarted() {
    uiStore.openLoginModal(() => {
      showExplore = false;
      if (getHomeMode() === 'authenticated') {
        void refreshHome({ refreshWidgets: true });
      }
    });
  }

  function handleExplore() {
    if (showExplore) {
      void refreshHome({ refreshWidgets: false });
      return;
    }
    showExplore = true;
  }

  $effect(() => {
    const mode = getHomeMode();
    if (mode === lastMode) return;
    lastMode = mode;
    lastAnchorUpdatedAt = null;
    if (mode === 'guest') return;
    void refreshHome({ refreshWidgets: mode === 'authenticated' });
  });

  $effect(() => {
    const mode = getHomeMode();
    if (mode === 'guest') {
      if (locationDebounceTimer) {
        clearTimeout(locationDebounceTimer);
        locationDebounceTimer = null;
      }
      lastAnchorUpdatedAt = null;
      return;
    }
    const anchorUpdatedAt = mapAnchorStore.anchor.updatedAt;
    if (lastAnchorUpdatedAt === null) {
      lastAnchorUpdatedAt = anchorUpdatedAt;
      return;
    }
    if (anchorUpdatedAt === lastAnchorUpdatedAt) return;
    lastAnchorUpdatedAt = anchorUpdatedAt;
    if (locationDebounceTimer) clearTimeout(locationDebounceTimer);
    locationDebounceTimer = setTimeout(() => {
      void refreshHome({ refreshWidgets: mode === 'authenticated' });
    }, 500);
    return () => {
      if (locationDebounceTimer) {
        clearTimeout(locationDebounceTimer);
        locationDebounceTimer = null;
      }
    };
  });

  function handleFollowToggle(user: UserItem) {
    if (!authStore.isAuthenticated) {
      uiStore.openLoginModalWithRedirect({});
      return;
    }
    void goto(`/users/${user.user.user_id}`);
  }
</script>

<svelte:head>
  <title>{t('home.title')}</title>
</svelte:head>

{#if !authStore.isAuthenticated && !showExplore}
  <!-- Guest landing (Figma Make style) -->
  <div class="flex flex-col px-5 pt-4 pb-6">
    <!-- Sticky header -->
    <div
      class="sticky top-0 z-10 flex items-center justify-between px-1 py-3"
      style="background-color: var(--surface-bg); border-bottom: 1px solid var(--border-theme);"
    >
      <div>
        <h1 style="font-size: 22px; color: var(--text-primary);">{t('home.title')}</h1>
        <p style="font-size: 12px; color: var(--text-secondary); margin-top: 2px;">
          {t('home.discoverNearbyPeopleAndThings')}
        </p>
      </div>
      <div class="surface-flat flex cursor-pointer items-center gap-1.5 rounded-full py-1.5 px-3.5">
        <MapPin size={12} strokeWidth={1.5} color="var(--brand)" />
        <span style="font-size: 11px; color: var(--text-secondary);"
          >{mapAnchorStore.displayName}</span
        >
      </div>
    </div>

    <!-- Hero card -->
    <div class="mt-4">
      <div class="surface-hero flex flex-col items-center rounded-[24px] px-6 py-10 text-center">
        <div
          class="mb-4 flex items-center justify-center rounded-full"
          style="width: 56px; height: 56px; background: var(--brand); box-shadow: 0 6px 20px var(--glow);"
        >
          <Sparkles size={24} color="var(--text-on-accent)" />
        </div>
        <h2 style="font-size: 22px; font-weight: 700; color: var(--text-primary);">
          {t('home.discoverNearby')}
        </h2>
        <p
          style="font-size: 14px; color: var(--text-secondary); margin-top: 8px; max-width: 320px;"
        >
          {t('home.heroDescription')}
        </p>
        <div class="mt-6 flex gap-3">
          <button
            class="rounded-[22px] px-6 py-3.5 font-medium transition-transform duration-150 active:scale-[0.95]"
            style="background: var(--brand); color: var(--text-on-accent); box-shadow: 0 4px 14px var(--glow);"
            onclick={handleGetStarted}
          >
            {t('home.getStarted')}
          </button>
          <button
            class="rounded-[22px] px-6 py-3.5 font-medium transition-transform duration-150 active:scale-[0.97]"
            style="box-shadow: var(--shadow-sm); border: 1px solid var(--border-theme); color: var(--text-secondary); background: var(--surface-bg);"
            onclick={handleExplore}
          >
            {t('home.browseFirst')}
          </button>
        </div>
      </div>
    </div>

    <!-- Features grid -->
    <div class="mt-5">
      <p
        class="mb-3 uppercase"
        style="font-size: 10px; letter-spacing: 0.14em; color: var(--text-secondary);"
      >
        {t('home.coreFeatures')}
      </p>
      <div style="display: grid; grid-template-columns: repeat(2, 1fr); gap: 12px;">
        {#each [{ icon: MapPin, label: t('nav.nearby'), color: 'var(--brand)' }, { icon: Compass, label: t('nav.discover'), color: 'var(--text-tertiary)' }, { icon: Users, label: t('nav.messagesTab'), color: 'var(--brand)' }, { icon: Star, label: t('nav.me'), color: 'var(--neu-warning)' }] as feat}
          <div
            class="surface-flat flex flex-col items-center justify-center rounded-[22px] card-hover transition-transform duration-150 active:scale-[0.97]"
            style="aspect-ratio: 1.2;"
          >
            <div
              class="flex items-center justify-center rounded-[14px]"
              style="width: 36px; height: 36px; background: var(--surface-secondary);"
            >
              <feat.icon size={17} strokeWidth={1.75} color={feat.color} />
            </div>
            <span style="font-size: 12px; color: var(--text-primary); margin-top: 8px;"
              >{feat.label}</span
            >
          </div>
        {/each}
      </div>
    </div>
  </div>
{:else if !platformStore.isDesktop}
  <!-- Mobile / Tablet Home (Figma Make HomeMobile; tablet = centered 600 column) -->
  {@const contentMax = platformStore.isTablet ? '600px' : '680px'}
  <div class="flex flex-col">
    <!-- Sticky header -->
    <div
      class="sticky top-0 z-10 px-5 py-3"
      style="background-color: var(--surface-bg); border-bottom: 1px solid var(--border-theme);"
    >
      <div
        class="flex items-center justify-between"
        style="max-width: {contentMax}; margin-left: auto; margin-right: auto; width: 100%;"
      >
        <div>
          <h1 style="font-size: 22px; color: var(--text-primary);">{t('nav.nearby')}</h1>
          <p style="font-size: 12px; color: var(--text-secondary); margin-top: 2px;">
            {t('home.discoverNearbyPeopleAndThings')}
          </p>
        </div>
        <div
          class="surface-flat flex cursor-pointer items-center gap-1.5 rounded-full py-1.5 px-3.5"
          onclick={() => void goto('/location')}
        >
          <MapPin size={12} strokeWidth={1.5} color="var(--brand)" />
          <span style="font-size: 11px; color: var(--text-secondary);"
            >{mapAnchorStore.displayName}</span
          >
        </div>
      </div>
    </div>

    <div
      class="flex flex-col px-5 pt-4 pb-6"
      style="max-width: {contentMax}; margin-left: auto; margin-right: auto; width: 100%;"
    >
      <!-- Map Card (Figma Make HomeMobile) -->
      <div
        class="surface-hero relative mb-5 w-full overflow-hidden rounded-[24px]"
        style="aspect-ratio: 2.4/1;"
      >
        <div
          class="absolute inset-0"
          style="background: linear-gradient(135deg, var(--surface-secondary), var(--surface-bg));"
        ></div>
        <div
          class="absolute inset-0"
          style="background: linear-gradient(to bottom, transparent 30%, rgba(0,0,0,0.3) 100%);"
        ></div>
        <div class="absolute inset-0 flex items-center justify-center">
          <div
            class="flex items-center justify-center rounded-full"
            style="width: 44px; height: 44px; background: var(--surface-bg); opacity: 0.88; backdrop-filter: blur(8px); border: 1.5px solid rgba(255,255,255,0.35); box-shadow: 0 4px 16px rgba(0,0,0,0.18);"
          >
            <span
              class="rounded-full animate-pulse"
              style="width: 14px; height: 14px; background: var(--brand); box-shadow: 0 0 0 4px var(--glow);"
            ></span>
          </div>
        </div>
        <div class="absolute bottom-3 right-3 flex flex-col gap-2">
          {#each [Navigation2, Compass] as Icon}
            <button
              class="flex items-center justify-center rounded-xl outline-none"
              style="width: 34px; height: 34px; background: var(--surface-bg); opacity: 0.88; backdrop-filter: blur(8px); border: 1px solid rgba(255,255,255,0.25); box-shadow: 2px 2px 6px rgba(0,0,0,0.15);"
            >
              <Icon size={14} strokeWidth={1.5} color="var(--text-primary)" />
            </button>
          {/each}
        </div>
        <div
          class="absolute bottom-3 left-3 flex items-center gap-1 rounded-full px-2.5 py-1"
          style="background: var(--surface-bg); opacity: 0.88; backdrop-filter: blur(8px); border: 1px solid rgba(255,255,255,0.2);"
        >
          <span style="font-size: 11px; color: var(--brand); font-weight: 600;"
            >{nearbyUsers.length}</span
          >
          <span style="font-size: 10px; color: var(--text-secondary);">人附近</span>
        </div>
      </div>

      <!-- Nearby people -->
      {#if authStore.isAuthenticated && (loadingNearby || radarUsers.length > 0)}
        <div class="mb-5">
          <div class="mb-3 flex items-center justify-between">
            <span class="eyebrow">{t('sidebar.nearbyPeople')}</span>
            <a
              href="/users/nearby"
              class="flex items-center gap-0.5 outline-none"
              style="color: var(--brand); font-size: 12px;"
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
          <div class="mt-3">
            <NearbyUsersFlatList
              users={flatUsers.slice(0, 5)}
              loading={loadingNearby}
              {followingIds}
              onToggleFollow={handleFlatFollow}
            />
          </div>
        </div>
      {/if}

      <!-- Nearby activity -->
      {#if loadingNearby || activities.length > 0}
        <div class="mb-5">
          <div class="mb-3 flex items-center justify-between">
            <span class="eyebrow">{t('home.nearbyActivity')}</span>
            <button
              class="flex items-center gap-0.5 outline-none"
              style="color: var(--brand); font-size: 12px; background: none; border: none; cursor: pointer;"
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
      {#if carouselGroups.length > 0}
        <GroupCarousel groups={carouselGroups} loading={loadingNearby} />
      {/if}
    </div>
  </div>
{:else}
  <!-- Desktop Home (Figma Make HomeDesktop — 2-column) -->
  <div class="flex h-full flex-col">
    <!-- Sticky header -->
    <div
      class="sticky top-0 z-10 flex items-center justify-between px-6 py-4"
      style="background-color: var(--surface-bg); border-bottom: 1px solid var(--border-theme);"
    >
      <div>
        <h1 style="font-size: 24px; color: var(--text-primary);">
          {t('sidebar.home')}
        </h1>
        <p style="font-size: 12px; color: var(--text-secondary); margin-top: 2px;">
          {t('home.discoverNearbyPeopleAndThings')}
        </p>
      </div>
      <div class="flex items-center gap-2">
        <a
          href="/location"
          class="surface-flat flex items-center gap-1.5 rounded-full py-1.5 px-3.5 outline-none"
          style="text-decoration: none;"
        >
          <MapPin size={12} strokeWidth={1.5} color="var(--brand)" />
          <span style="font-size: 11px; color: var(--text-secondary);"
            >{mapAnchorStore.displayName}</span
          >
        </a>
        <button
          class="surface-flat flex items-center gap-1.5 rounded-full py-1.5 px-3.5 outline-none"
          onclick={refreshAll}
        >
          <span
            style="width: 7px; height: 7px; border-radius: 999px; background: var(--color-semantic-online, #66835d); box-shadow: 0 0 0 3px color-mix(in srgb, var(--color-semantic-online, #66835d) 22%, transparent);"
          ></span>
          <span style="font-size: 11px; color: var(--text-secondary);"
            >{onlineCount} {t('common.online')}</span
          >
        </button>
      </div>
    </div>

    <div class="flex min-h-0 flex-1">
      <!-- Left column -->
      <div
        class="flex flex-col gap-6 overflow-y-auto no-scrollbar p-6"
        style="width: 340px; flex-shrink: 0; border-right: 1px solid var(--border-theme);"
      >
        <HeroStats
          people={nearbyUsers.length}
          activity={activityCount}
          online={onlineCount}
          title={t('home.realtimeNearby')}
          rangeLabel={`${mapAnchorStore.radius}km ${t('home.rangeWithin')}`}
          loading={loadingNearby}
          onclick={() => void goto('/location')}
        />

        <!-- Nearby activity -->
        {#if loadingNearby || activities.length > 0}
          <div>
            <div class="mb-3 flex items-center justify-between">
              <span class="eyebrow">{t('home.nearbyActivity')}</span>
              <button
                class="flex items-center gap-0.5 outline-none"
                style="color: var(--brand); font-size: 12px; background: none; border: none; cursor: pointer;"
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
        {#if carouselGroups.length > 0}
          <GroupCarousel groups={carouselGroups} loading={loadingNearby} />
        {/if}
      </div>

      <!-- Right column -->
      <div class="flex flex-1 flex-col gap-6 overflow-y-auto no-scrollbar p-6">
        <!-- People radar -->
        {#if loadingNearby || radarUsers.length > 0}
          <div>
            <div class="mb-3 flex items-start justify-between">
              <div>
                <span class="eyebrow">{t('sidebar.nearbyPeople')}</span>
                <p style="font-size: 11px; color: var(--text-secondary); margin-top: 3px;">
                  {mapAnchorStore.radius}km {t('home.rangeWithin')}
                </p>
              </div>
              <a
                href="/users/nearby"
                class="flex items-center gap-0.5 outline-none"
                style="color: var(--brand); font-size: 12px;"
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
              <NearbyUsersFlatList
                users={flatUsers}
                loading={loadingNearby}
                {followingIds}
                onToggleFollow={handleFlatFollow}
              />
            </div>
          </div>
        {/if}

        <!-- Surrounding heat -->
        {#if activities.length > 0}
          <div>
            <div class="mb-3 flex items-center justify-between">
              <span class="eyebrow">{t('home.surroundingHeat')}</span>
            </div>
            <ActivityList {activities} loading={loadingNearby} onSelect={handleActivitySelect} />
          </div>
        {/if}
      </div>
    </div>
  </div>
{/if}
