<script lang="ts">
  import { authStore } from '$lib/stores/auth.svelte';
  import { uiStore } from '$lib/stores/ui.svelte';
  import { mapAnchorStore } from '$lib/stores/map-anchor.svelte';
  import { t } from '$lib/i18n';
  import { usersService, feedService } from '$lib/api/services';
  import { isSuccess } from '$lib/api/client';
  import type { UserItem, TrendingTagItem } from '$lib/types/api';
  import MapPin from 'lucide-svelte/icons/map-pin';
  import Users from 'lucide-svelte/icons/users';
  import Compass from 'lucide-svelte/icons/compass';
  import ChevronRight from 'lucide-svelte/icons/chevron-right';
  import Sparkles from 'lucide-svelte/icons/sparkles';
  import Star from 'lucide-svelte/icons/star';
  import TrendingUp from 'lucide-svelte/icons/trending-up';
  import Coffee from 'lucide-svelte/icons/coffee';
  import Music from 'lucide-svelte/icons/music';
  import Navigation2 from 'lucide-svelte/icons/navigation-2';
  import { goto } from '$app/navigation';

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
  let loadingNearby = $state(false);

  const onlineCount = $derived(nearbyUsers.filter((u) => u.presence_status === 'online').length);
  const activityCount = $derived(trendingTags.reduce((sum, t) => sum + t.usage_count, 0));

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
    } else if (refreshWidgets) {
      await loadTrending();
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
  <!-- Mobile / Tablet Home (Figma Make HomeMobile, single column, maxWidth 680) -->
  <div class="flex flex-col">
    <!-- Sticky header -->
    <div
      class="sticky top-0 z-10 flex items-center justify-between px-5 py-3"
      style="background-color: var(--surface-bg); border-bottom: 1px solid var(--border-theme);"
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

    <div
      class="flex flex-col px-5 pt-4 pb-6"
      style="max-width: 680px; margin-left: auto; margin-right: auto; width: 100%;"
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

      <!-- Nearby People -->
      {#if authStore.isAuthenticated}
        <div class="mb-3 flex items-center justify-between">
          <span
            class="uppercase"
            style="font-size: 10px; letter-spacing: 0.14em; color: var(--text-secondary);"
            >附近的人</span
          >
          <button
            class="flex items-center gap-0.5 outline-none"
            style="color: var(--brand);"
            onclick={() => void goto('/users/nearby')}
          >
            <span style="font-size: 12px;">{t('common.viewAll')}</span>
            <ChevronRight size={13} strokeWidth={1.5} />
          </button>
        </div>
        <div class="mb-5 flex flex-col gap-2.5">
          {#if loadingNearby}
            {#each Array(3) as _}
              <div
                class="surface-flat flex items-center gap-3.5 rounded-[22px] px-4 py-3.5"
                style="height: 68px;"
              >
                <div
                  class="rounded-full"
                  style="width: 44px; height: 44px; background: var(--border-theme);"
                ></div>
                <div class="flex-1">
                  <div
                    class="mb-1.5 rounded"
                    style="width: 60px; height: 12px; background: var(--border-theme);"
                  ></div>
                  <div
                    class="rounded"
                    style="width: 80px; height: 10px; background: var(--border-theme);"
                  ></div>
                </div>
              </div>
            {/each}
          {:else}
            {#each nearbyUsers.slice(0, 3) as user (user.user.user_id)}
              <div
                role="button"
                tabindex="0"
                class="surface-flat flex cursor-pointer items-center gap-3.5 rounded-[22px] px-4 py-3.5 text-left row-hover transition-transform duration-150 active:scale-[0.97]"
                aria-label={user.user.nickname}
                onclick={() => void goto(`/users/${user.user.user_id}`)}
              >
                <div class="relative flex-shrink-0">
                  <div
                    class="overflow-hidden rounded-full"
                    style="width: 44px; height: 44px; border: 1.5px solid var(--border-theme);"
                  >
                    <img
                      src={user.user.avatar_url || ''}
                      alt={user.user.nickname}
                      class="h-full w-full object-cover"
                    />
                  </div>
                  {#if user.presence_status === 'online'}
                    <span
                      class="absolute bottom-0 right-0 rounded-full"
                      style="width: 11px; height: 11px; background: #22c55e; border: 2px solid var(--surface-bg);"
                    ></span>
                  {/if}
                </div>
                <div class="min-w-0 flex-1">
                  <p style="font-size: 14px; color: var(--text-primary);">{user.user.nickname}</p>
                  <div class="mt-0.5 flex items-center gap-1.5">
                    {#if user.presence_status}
                      <span
                        style="font-size: 10px; color: {user.presence_status === 'online'
                          ? '#22c55e'
                          : 'var(--text-secondary)'};"
                      >
                        {user.presence_status === 'online'
                          ? t('common.online')
                          : t('common.offline')}
                      </span>
                      <span style="font-size: 10px; color: var(--text-secondary); opacity: 0.5;"
                        >·</span
                      >
                    {/if}
                    {#if user.tags && user.tags.length > 0}
                      <span style="font-size: 10px; color: var(--brand); opacity: 0.8;"
                        >{user.tags[0]}</span
                      >
                    {/if}
                  </div>
                </div>
                <div class="flex flex-col items-end gap-2">
                  {#if user.location?.distance != null}
                    <span style="font-size: 12px; color: var(--brand);"
                      >{formatDistance(user.location.distance)}</span
                    >
                  {/if}
                  <button
                    class="rounded-full px-3.5 py-1 outline-none transition-all active:scale-[0.93]"
                    class:surface-pressed={user.relation?.is_following}
                    class:surface-flat={!user.relation?.is_following}
                    style={user.relation?.is_following
                      ? 'color: var(--brand); font-size: 11px;'
                      : 'color: var(--text-secondary); font-size: 11px;'}
                    onclick={(e) => {
                      e.stopPropagation();
                      handleFollowToggle(user);
                    }}
                  >
                    {user.relation?.is_following ? t('common.following') : t('common.follow')}
                  </button>
                </div>
              </div>
            {/each}
          {/if}
        </div>
      {/if}

      <!-- Activities (Figma Make) -->
      <div class="mb-3 flex items-center justify-between">
        <span
          class="uppercase"
          style="font-size: 10px; letter-spacing: 0.14em; color: var(--text-secondary);"
          >附近活动</span
        >
        <button class="flex items-center gap-0.5 outline-none" style="color: var(--brand);">
          <span style="font-size: 12px;">更多</span>
          <ChevronRight size={13} strokeWidth={1.5} />
        </button>
      </div>
      <div
        class="mb-5"
        style="display: grid; grid-template-columns: repeat(2, 1fr); gap: 12px; width: 100%;"
      >
        {#each trendingTags.slice(0, 2) as t, i}
          {@const Icon = [Coffee, Music][i] ?? Star}
          {@const cfg = [
            { color: '#F59E0B', bg: 'rgba(245,158,11,0.1)' },
            { color: '#8B5CF6', bg: 'rgba(139,92,246,0.1)' }
          ][i] ?? { color: 'var(--brand)', bg: 'var(--surface-bg)' }}
          <button
            class="surface-flat flex flex-col justify-between rounded-[22px] cursor-pointer card-hover transition-transform duration-150 active:scale-[0.97]"
            style="aspect-ratio: 1/1; padding: 16px;"
          >
            <div
              class="flex items-center justify-center rounded-[12px]"
              style="width: 36px; height: 36px; background: {cfg.bg};"
            >
              <Icon size={17} strokeWidth={1.75} color={cfg.color} />
            </div>
            <div>
              <p style="font-size: 13px; color: var(--text-primary);">#{t.name}</p>
              <p style="font-size: 11px; color: var(--text-secondary); margin-top: 2px;">
                {t.usage_count}条动态
              </p>
            </div>
          </button>
        {/each}
      </div>
    </div>
  </div>
{:else}
  <!-- Desktop Home (Figma Make HomeDesktop — 3-column) -->
  <div class="flex h-full flex-col">
    <!-- Sticky header -->
    <div
      class="sticky top-0 z-10 flex items-center justify-between px-6 py-4"
      style="background-color: var(--surface-bg); border-bottom: 1px solid var(--border-theme);"
    >
      <div>
        <h1 style="font-size: 24px; color: var(--text-primary);">
          {t('nav.nearby')}
        </h1>
        <p style="font-size: 12px; color: var(--text-secondary); margin-top: 2px;">
          {t('home.discoverNearbyPeopleAndThings')}
        </p>
      </div>
      <div class="flex items-center gap-2">
        <div class="surface-flat flex items-center gap-1.5 rounded-full py-1.5 px-3.5">
          <MapPin size={12} strokeWidth={1.5} color="var(--brand)" />
          <span style="font-size: 11px; color: var(--text-secondary);"
            >{mapAnchorStore.displayName}</span
          >
        </div>
        <button
          class="flex items-center gap-1 rounded-full px-3 py-1.5"
          style="background: var(--brand); box-shadow: 0 4px 12px var(--glow);"
          onclick={refreshAll}
        >
          <TrendingUp size={11} strokeWidth={2} color="var(--text-on-accent)" />
          <span style="font-size: 11px; color: var(--text-on-accent);"
            >{nearbyUsers.length}{t('sidebar.nearbyPeople')}</span
          >
        </button>
      </div>
    </div>

    <div class="flex min-h-0 flex-1 gap-0">
      <!-- Left column (Figma Make HomeDesktop) -->
      <div
        class="flex flex-col gap-4 overflow-y-auto no-scrollbar p-5"
        style="width: 340px; flex-shrink: 0; border-right: 1px solid var(--border-theme);"
      >
        <!-- Map card -->
        <div
          class="surface-hero relative w-full overflow-hidden rounded-[24px]"
          style="height: 200px; opacity: 0.7;"
        >
          <div
            class="absolute inset-0"
            style="background: linear-gradient(135deg, var(--surface-secondary), var(--surface-bg));"
          ></div>
          <div
            class="absolute inset-0"
            style="background: linear-gradient(to bottom, transparent 40%, rgba(0,0,0,0.3) 100%);"
          ></div>
          <div class="absolute inset-0 flex items-center justify-center">
            <div
              class="flex items-center justify-center rounded-full"
              style="width: 48px; height: 48px; background: var(--surface-bg); opacity: 0.88; backdrop-filter: blur(10px); border: 1.5px solid rgba(255,255,255,0.35); box-shadow: 0 4px 16px rgba(0,0,0,0.18);"
            >
              <span
                class="rounded-full animate-pulse"
                style="width: 14px; height: 14px; background: var(--brand); box-shadow: 0 0 0 5px var(--glow);"
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

        <!-- Stats grid (3 cards) -->
        <div style="display: grid; grid-template-columns: repeat(3, 1fr); gap: 12px;">
          {#each [{ icon: Users, label: '附近', value: nearbyUsers.length, color: 'var(--brand)' }, { icon: Star, label: '活动', value: activityCount, color: '#F59E0B' }, { icon: TrendingUp, label: '在线', value: onlineCount, color: '#22c55e' }] as stat}
            <div
              class="surface-flat flex flex-col items-center justify-center rounded-[18px] py-3.5"
            >
              <stat.icon
                size={16}
                strokeWidth={1.75}
                color={stat.color}
                style="margin-bottom: 4px;"
              />
              <span style="font-size: 20px; font-weight: 700; color: var(--text-primary);"
                >{stat.value}</span
              >
              <span style="font-size: 10px; color: var(--text-secondary); margin-top: 1px;"
                >{stat.label}</span
              >
            </div>
          {/each}
        </div>

        <!-- Activities list -->
        <div>
          <p
            class="uppercase mb-3"
            style="font-size: 10px; letter-spacing: 0.14em; color: var(--text-secondary);"
          >
            附近活动
          </p>
          <div class="flex flex-col gap-2.5">
            {#each trendingTags.slice(0, 3) as t, i}
              {@const Icon = [Coffee, Music, Sparkles][i] ?? Star}
              {@const cfg = [
                { color: '#F59E0B', bg: 'rgba(245,158,11,0.1)' },
                { color: '#8B5CF6', bg: 'rgba(139,92,246,0.1)' },
                { color: '#06B6D4', bg: 'rgba(6,182,212,0.1)' }
              ][i] ?? { color: 'var(--brand)', bg: 'var(--surface-bg)' }}
              <div
                class="surface-flat flex cursor-pointer items-center gap-3 rounded-[18px] px-4 py-3 row-hover"
              >
                <div
                  class="flex items-center justify-center rounded-[12px]"
                  style="width: 36px; height: 36px; background: {cfg.bg};"
                >
                  <Icon size={17} strokeWidth={1.75} color={cfg.color} />
                </div>
                <div class="flex-1 min-w-0">
                  <p style="font-size: 13px; color: var(--text-primary);">#{t.name}</p>
                  <p style="font-size: 11px; color: var(--text-secondary); margin-top: 2px;">
                    {t.usage_count}条动态
                  </p>
                </div>
                <ChevronRight size={14} strokeWidth={1.5} color="var(--text-secondary)" />
              </div>
            {/each}
          </div>
        </div>
      </div>

      <!-- Main content column — people + feed -->
      <div class="flex-1 overflow-y-auto no-scrollbar p-5">
        <!-- Nearby People -->
        {#if authStore.isAuthenticated}
          <div class="mb-4 flex items-center justify-between">
            <p
              class="uppercase"
              style="font-size: 10px; letter-spacing: 0.14em; color: var(--text-secondary);"
            >
              {t('sidebar.nearbyPeople')}
            </p>
            <a
              href="/users/nearby"
              class="flex items-center gap-0.5 outline-none"
              style="color: var(--brand);"
            >
              <span style="font-size: 12px;">{t('common.viewAll')}</span>
              <ChevronRight size={13} strokeWidth={1.5} />
            </a>
          </div>
          <div class="mb-5 flex flex-col gap-3">
            {#if loadingNearby}
              {#each Array(4) as _}
                <div
                  class="surface-flat flex items-center gap-3.5 rounded-[22px] px-4 py-3.5"
                  style="height: 68px;"
                >
                  <div
                    class="rounded-full"
                    style="width: 44px; height: 44px; background: var(--border-theme);"
                  ></div>
                  <div class="flex-1">
                    <div
                      class="mb-1.5 rounded"
                      style="width: 60px; height: 12px; background: var(--border-theme);"
                    ></div>
                    <div
                      class="rounded"
                      style="width: 80px; height: 10px; background: var(--border-theme);"
                    ></div>
                  </div>
                </div>
              {/each}
            {:else}
              {#each nearbyUsers as user (user.user.user_id)}
                <div
                  role="button"
                  tabindex="0"
                  class="surface-flat flex cursor-pointer items-center gap-3.5 rounded-[22px] px-4 py-3.5 text-left row-hover transition-transform duration-150 active:scale-[0.97]"
                  aria-label={user.user.nickname}
                  onclick={() => void goto(`/users/${user.user.user_id}`)}
                >
                  <div class="relative flex-shrink-0">
                    <div
                      class="overflow-hidden rounded-full"
                      style="width: 44px; height: 44px; border: 1.5px solid var(--border-theme);"
                    >
                      <img
                        src={user.user.avatar_url || ''}
                        alt={user.user.nickname}
                        class="h-full w-full object-cover"
                      />
                    </div>
                    {#if user.presence_status === 'online'}
                      <span
                        class="absolute bottom-0 right-0 rounded-full"
                        style="width: 11px; height: 11px; background: #22c55e; border: 2px solid var(--surface-bg);"
                      ></span>
                    {/if}
                  </div>
                  <div class="min-w-0 flex-1">
                    <p style="font-size: 14px; color: var(--text-primary);">{user.user.nickname}</p>
                    <div class="mt-0.5 flex items-center gap-1.5">
                      {#if user.presence_status}
                        <span
                          style="font-size: 10px; color: {user.presence_status === 'online'
                            ? '#22c55e'
                            : 'var(--text-secondary)'};"
                        >
                          {user.presence_status === 'online'
                            ? t('common.online')
                            : t('common.offline')}
                        </span>
                        <span style="font-size: 10px; color: var(--text-secondary); opacity: 0.5;"
                          >·</span
                        >
                      {/if}
                      {#if user.tags && user.tags.length > 0}
                        <span style="font-size: 10px; color: var(--brand); opacity: 0.8;"
                          >{user.tags[0]}</span
                        >
                      {/if}
                    </div>
                  </div>
                  <div class="flex flex-col items-end gap-2">
                    {#if user.location?.distance != null}
                      <span style="font-size: 12px; color: var(--brand);"
                        >{formatDistance(user.location.distance)}</span
                      >
                    {/if}
                    <button
                      class="rounded-full px-3.5 py-1.5 outline-none transition-all active:scale-[0.93]"
                      class:surface-pressed={user.relation?.is_following}
                      class:surface-flat={!user.relation?.is_following}
                      style={user.relation?.is_following
                        ? 'color: var(--brand); font-size: 11px;'
                        : 'color: var(--text-secondary); font-size: 11px;'}
                      onclick={(e) => {
                        e.stopPropagation();
                        handleFollowToggle(user);
                      }}
                    >
                      {user.relation?.is_following ? t('common.following') : t('common.follow')}
                    </button>
                  </div>
                </div>
              {/each}
            {/if}
          </div>
        {/if}

        <!-- Banner card (Figma Make HomeDesktop) -->
        <div
          class="surface-hero relative mb-5 overflow-hidden rounded-[24px]"
          style="height: clamp(120px, 12vw, 180px);"
        >
          <div
            class="absolute inset-0"
            style="background: linear-gradient(135deg, var(--surface-secondary), var(--surface-bg));"
          ></div>
          <div
            class="absolute inset-0"
            style="background: linear-gradient(to right, rgba(99,102,241,0.75) 0%, transparent 60%);"
          ></div>
          <div class="relative flex h-full flex-col justify-center px-5">
            <h2 style="font-size: 16px; font-weight: 700; color: white;">发现身边精彩</h2>
            <p style="font-size: 12px; color: rgba(255,255,255,0.85); margin-top: 4px;">
              加入本地社区，探索附近的人和活动
            </p>
            <button
              class="mt-3 rounded-[20px] px-3.5 py-1.5 outline-none"
              style="background: white; color: var(--brand); font-size: 12px; font-weight: 600; box-shadow: 0 4px 12px rgba(0,0,0,0.2);"
              >立即探索</button
            >
          </div>
        </div>
      </div>

      <!-- Right context rail (Figma Make DesktopHome/Nearby v2) -->
      <div
        class="flex flex-col gap-4 overflow-y-auto no-scrollbar p-5"
        style="width: 320px; flex-shrink: 0; border-left: 1px solid var(--border-theme);"
      >
        <!-- Location context -->
        <div>
          <p
            class="uppercase mb-3"
            style="font-size: 10px; letter-spacing: 0.14em; color: var(--text-secondary);"
          >
            位置信息
          </p>
          <div class="surface-flat rounded-[18px] p-4">
            <div class="flex items-center gap-2 mb-2">
              <Navigation2 size={14} strokeWidth={1.5} color="var(--brand)" />
              <span style="font-size: 13px; color: var(--text-primary); font-weight: 500;"
                >{mapAnchorStore.displayName}</span
              >
            </div>
            <span style="font-size: 11px; color: var(--text-secondary);">
              半径 {mapAnchorStore.radius}km · {nearbyUsers.length} 人附近
            </span>
          </div>
        </div>

        <!-- Suggested people -->
        {#if nearbyUsers.length > 0}
          <div>
            <div class="mb-3 flex items-center justify-between">
              <p
                class="uppercase"
                style="font-size: 10px; letter-spacing: 0.14em; color: var(--text-secondary);"
              >
                推荐关注
              </p>
              <a
                href="/users/nearby"
                class="flex items-center gap-0.5 outline-none"
                style="color: var(--brand); font-size: 12px;"
              >
                {t('common.viewAll')}
                <ChevronRight size={13} strokeWidth={1.5} />
              </a>
            </div>
            <div class="flex flex-col gap-2">
              {#each nearbyUsers.slice(0, 4) as user (user.user.user_id)}
                <a
                  href="/users/{user.user.user_id}"
                  class="surface-flat flex items-center gap-3 rounded-[14px] px-3 py-2.5 row-hover"
                  style="text-decoration: none;"
                >
                  <div class="relative flex-shrink-0">
                    <div
                      class="overflow-hidden rounded-full"
                      style="width: 36px; height: 36px; border: 1px solid var(--border-theme);"
                    >
                      <img
                        src={user.user.avatar_url || ''}
                        alt={user.user.nickname}
                        class="h-full w-full object-cover"
                      />
                    </div>
                    {#if user.presence_status === 'online'}
                      <span
                        class="absolute bottom-0 right-0 rounded-full"
                        style="width: 9px; height: 9px; background: #22c55e; border: 2px solid var(--surface-bg);"
                      ></span>
                    {/if}
                  </div>
                  <div class="min-w-0 flex-1">
                    <p style="font-size: 12px; color: var(--text-primary);">{user.user.nickname}</p>
                    {#if user.location?.distance != null}
                      <span style="font-size: 10px; color: var(--brand);"
                        >{formatDistance(user.location.distance)}</span
                      >
                    {/if}
                  </div>
                </a>
              {/each}
            </div>
          </div>
        {/if}
      </div>
    </div>
  </div>
{/if}
