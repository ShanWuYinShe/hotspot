<script lang="ts">
  import { onMount } from 'svelte';
  import { goto } from '$app/navigation';
  import { t } from '$lib/i18n';
  import { platformStore } from '$lib/stores/platform.svelte';
  import { mapAnchorStore } from '$lib/stores/map-anchor.svelte';
  import { feedService } from '$lib/api/services';
  import { isSuccess } from '$lib/api/client';
  import type { UserItem, GroupItem, GeoPoint, TrendingTagItem } from '$lib/types/api';
  import HeroStats from '$lib/components/home/hero-stats.svelte';
  import PeopleRadarStrip from '$lib/components/home/people-radar-strip.svelte';
  import ActivityList from '$lib/components/home/activity-list.svelte';
  import GroupCarousel from '$lib/components/home/group-carousel.svelte';
  import NearbyUsersFlatList from '$lib/components/home/nearby-users-flat-list.svelte';

  function formatDistance(m: number): string {
    if (m < 1000) return `${Math.round(m)}m`;
    return `${(m / 1000).toFixed(1)}km`;
  }

  let loading = $state(true);
  let nearbyUsers: UserItem[] = $state([]);
  let nearbyGroups: GroupItem[] = $state([]);
  let trendingTags: TrendingTagItem[] = $state([]);
  let followingIds = $state(new Set<string>());

  function toReferencePoint(): GeoPoint {
    return {
      latitude: mapAnchorStore.latitude,
      longitude: mapAnchorStore.longitude
    };
  }

  async function loadData() {
    loading = true;
    const ref = toReferencePoint();

    const [usersRes, groupsRes, trendingRes] = await Promise.allSettled([
      feedService.getUsers({ reference_point: ref, radius: 5, pagination: { limit: 15 } }),
      feedService.getGroups({ reference_point: ref, radius: 5, pagination: { limit: 8 } }),
      feedService.getTrending()
    ]);

    if (usersRes.status === 'fulfilled' && isSuccess(usersRes.value)) {
      nearbyUsers = usersRes.value.data?.items ?? [];
    }
    if (groupsRes.status === 'fulfilled' && isSuccess(groupsRes.value)) {
      nearbyGroups = groupsRes.value.data?.items ?? [];
    }
    if (trendingRes.status === 'fulfilled' && isSuccess(trendingRes.value)) {
      trendingTags = trendingRes.value.data?.trending_tags ?? [];
    }
    loading = false;
  }

  function mapUsers() {
    return nearbyUsers.map((u) => ({
      id: u.user.user_id,
      avatar_url: u.user.avatar_url ?? '',
      nickname: u.user.nickname,
      distance: u.location?.distance != null ? formatDistance(u.location.distance) : undefined,
      online: u.presence_status === 'online',
      tag: u.tags?.[0]
    }));
  }

  function mapGroups() {
    return nearbyGroups.map((g) => ({
      id: g.group.group_id,
      name: g.group.name,
      banner_url: g.group.avatar_url,
      member_count: g.group.members_count,
      distance: g.location?.distance != null ? formatDistance(g.location.distance) : undefined,
      is_private: g.join_method === 'invite_only'
    }));
  }

  function toggleFollow(userId: string) {
    followingIds = followingIds.has(userId)
      ? new Set([...followingIds].filter((id) => id !== userId))
      : new Set([...followingIds, userId]);
  }

  onMount(() => {
    loadData();
  });

  const onlineCount = $derived(nearbyUsers.filter((u) => u.presence_status === 'online').length);
  const peopleCount = $derived(nearbyUsers.length);
  const activityCount = $derived(
    trendingTags.reduce((sum, t) => sum + t.usage_count, 0)
  );

  // Shared props
  const heroProps = $derived({
    people: peopleCount,
    activity: activityCount,
    online: onlineCount,
    radius: `${mapAnchorStore.radius}km`,
    location: mapAnchorStore.displayName,
    loading
  });
</script>

<svelte:head>
  <title>{t('dashboard.title')}</title>
</svelte:head>

{#if platformStore.isMobile}
  <!-- ═══ Mobile: sticky header + scrollable feed ═══ -->
  <div class="flex flex-col h-full">
    <div
      class="sticky top-0 z-10 flex items-center justify-between px-5 py-3"
      style="background: var(--surface-bg); border-bottom: 1px solid var(--border-theme);"
    >
      <div>
        <h1 style="font-size: 22px; color: var(--text-primary);">附近</h1>
        <p style="font-size: 12px; color: var(--text-secondary); margin-top: 2px;">
          发现周边有趣的人和事
        </p>
      </div>
      <button
        class="flex items-center gap-1.5 rounded-full px-3 py-1.5 surface-flat"
        onclick={() => void goto('/location')}
      >
        <span
          class="rounded-full"
          style="width: 6px; height: 6px; background: #22c55e; flex-shrink: 0;"
        ></span>
        <span style="font-size: 11px; color: var(--text-secondary);"
          >{mapAnchorStore.displayName}</span
        >
      </button>
    </div>

    <div class="flex flex-col gap-7 px-5 pt-5 pb-8 flex-1 overflow-y-auto no-scrollbar">
      <HeroStats {...heroProps} />

      {#if !loading}
        <PeopleRadarStrip users={mapUsers()} onViewAll={() => void goto('/users/nearby')} />
      {:else}
        <PeopleRadarStrip loading />
      {/if}

      <div>
        <div class="flex items-end justify-between mb-3">
          <span class="eyebrow">附近活动</span>
          <button
            class="flex items-center gap-0.5 rounded-full px-2 py-0.5 surface-ghost"
            style="color: var(--brand); font-size: 11px;"
          >
            更多 →
          </button>
        </div>
        <ActivityList activities={trendingTags.slice(0, 3).map((t, i) => ({ id: t.name, icon: ['coffee' as const, 'music' as const, 'flame' as const][i] ?? 'star', title: `#${t.name}`, sub: `${t.usage_count}条动态`, color: ['#F59E0B', '#8B5CF6', '#EF4444'][i] ?? 'var(--brand)', bg: ['rgba(245,158,11,0.1)', 'rgba(139,92,246,0.1)', 'rgba(239,68,68,0.1)'][i] ?? 'var(--surface-bg)' }))} {loading} />
      </div>

      {#if !loading}
        <GroupCarousel groups={mapGroups()} />
      {:else}
        <GroupCarousel loading />
      {/if}

      <div>
        <div class="flex items-end justify-between mb-3">
          <span class="eyebrow">人物推荐</span>
          <button
            class="flex items-center gap-0.5 rounded-full px-2 py-0.5 surface-ghost"
            style="color: var(--brand); font-size: 11px;"
            onclick={() => void goto('/users/nearby')}
          >
            查看全部 →
          </button>
        </div>
        <NearbyUsersFlatList
          users={mapUsers().slice(0, 5)}
          {loading}
          {followingIds}
          onToggleFollow={toggleFollow}
        />
      </div>
    </div>
  </div>
{:else if platformStore.isTablet}
  <!-- ═══ Tablet: centered column + right rail ═══ -->
  <div class="flex flex-col h-full">
    <div
      class="sticky top-0 z-10 flex items-center justify-between px-5 py-3"
      style="background: var(--surface-bg); border-bottom: 1px solid var(--border-theme);"
    >
      <div>
        <h1 style="font-size: 22px; color: var(--text-primary);">附近</h1>
        <p style="font-size: 12px; color: var(--text-secondary); margin-top: 2px;">
          发现周边有趣的人和事
        </p>
      </div>
      <button
        class="flex items-center gap-1.5 rounded-full px-3 py-1.5 surface-flat"
        onclick={() => void goto('/location')}
      >
        <span
          class="rounded-full"
          style="width: 6px; height: 6px; background: #22c55e; flex-shrink: 0;"
        ></span>
        <span style="font-size: 11px; color: var(--text-secondary);"
          >{mapAnchorStore.displayName}</span
        >
      </button>
    </div>

    <div
      class="flex gap-6 pt-5 pb-8 flex-1 overflow-hidden"
      style="max-width: 924px; margin: 0 auto; width: 100%;"
    >
      <div
        class="flex flex-col gap-7 px-5 flex-1 min-w-0 overflow-y-auto no-scrollbar"
        style="max-width: 600px;"
      >
        <HeroStats {...heroProps} />

        {#if !loading}
          <PeopleRadarStrip users={mapUsers()} onViewAll={() => void goto('/users/nearby')} />
        {:else}
          <PeopleRadarStrip loading />
        {/if}

        <div>
          <div class="flex items-end justify-between mb-3">
            <span class="eyebrow">附近活动</span>
            <button
              class="flex items-center gap-0.5 rounded-full px-2 py-0.5 surface-ghost"
              style="color: var(--brand); font-size: 11px;"
            >
              更多 →
            </button>
          </div>
          <ActivityList activities={trendingTags.slice(0, 3).map((t, i) => ({ id: t.name, icon: ['coffee' as const, 'music' as const, 'flame' as const][i] ?? 'star', title: `#${t.name}`, sub: `${t.usage_count}条动态`, color: ['#F59E0B', '#8B5CF6', '#EF4444'][i] ?? 'var(--brand)', bg: ['rgba(245,158,11,0.1)', 'rgba(139,92,246,0.1)', 'rgba(239,68,68,0.1)'][i] ?? 'var(--surface-bg)' }))} {loading} />
        </div>

        {#if !loading}
          <GroupCarousel groups={mapGroups()} />
        {:else}
          <GroupCarousel loading />
        {/if}

        <div>
          <div class="flex items-end justify-between mb-3">
            <span class="eyebrow">人物推荐</span>
            <button
              class="flex items-center gap-0.5 rounded-full px-2 py-0.5 surface-ghost"
              style="color: var(--brand); font-size: 11px;"
              onclick={() => void goto('/users/nearby')}
            >
              查看全部 →
            </button>
          </div>
          <NearbyUsersFlatList
            users={mapUsers().slice(0, 5)}
            {loading}
            {followingIds}
            onToggleFollow={toggleFollow}
          />
        </div>
      </div>

      <!-- Tablet right rail -->
      <aside class="hidden lg:block flex-shrink-0" style="width: 320px;"></aside>
    </div>
  </div>
{:else}
  <!-- ═══ Desktop: 2-column layout ═══ -->
  <div class="flex flex-col h-full">
    <div
      class="sticky top-0 z-10 flex items-center justify-between px-7 py-4"
      style="background: var(--surface-bg); border-bottom: 1px solid var(--border-theme);"
    >
      <div>
        <h1 style="font-size: 24px; color: var(--text-primary);">首页</h1>
        <p style="font-size: 12px; color: var(--text-secondary); margin-top: 2px;">
          发现周边有趣的人和事
        </p>
      </div>
      <div class="flex items-center gap-2">
        <div class="flex items-center gap-1.5 rounded-full px-3 py-1.5 surface-flat">
          <span class="rounded-full" style="width: 6px; height: 6px; background: #22c55e;"></span>
          <span style="font-size: 12px; color: var(--text-secondary);">{onlineCount} 在线</span>
        </div>
      </div>
    </div>

    <div class="flex gap-0 overflow-hidden" style="height: calc(100% - 65px);">
      <!-- Left column: hero + activity + groups (340px) -->
      <div
        class="flex flex-col gap-6 overflow-y-auto no-scrollbar p-6"
        style="width: 340px; flex-shrink: 0; border-right: 1px solid var(--border-theme);"
      >
        <HeroStats {...heroProps} />

        <div>
          <div class="flex items-end justify-between mb-3">
            <span class="eyebrow">附近活动</span>
            <button
              class="flex items-center gap-0.5 rounded-full px-2 py-0.5 surface-ghost"
              style="color: var(--brand); font-size: 11px;"
            >
              更多 →
            </button>
          </div>
          <ActivityList activities={trendingTags.slice(0, 3).map((t, i) => ({ id: t.name, icon: ['coffee' as const, 'music' as const, 'flame' as const][i] ?? 'star', title: `#${t.name}`, sub: `${t.usage_count}条动态`, color: ['#F59E0B', '#8B5CF6', '#EF4444'][i] ?? 'var(--brand)', bg: ['rgba(245,158,11,0.1)', 'rgba(139,92,246,0.1)', 'rgba(239,68,68,0.1)'][i] ?? 'var(--surface-bg)' }))} {loading} />
        </div>

        {#if !loading}
          <GroupCarousel groups={mapGroups()} />
        {:else}
          <GroupCarousel loading />
        {/if}
      </div>

      <!-- Right column: people radar + full user list -->
      <div class="flex-1 overflow-y-auto no-scrollbar p-6">
        <div class="flex items-end justify-between mb-5">
          <div>
            <span class="eyebrow">附近的人</span>
            <p style="font-size: 12px; color: var(--text-secondary); margin-top: 4px;">
              {mapAnchorStore.radius}km 范围内 · 实时更新
            </p>
          </div>
          <button
            class="flex items-center gap-0.5 rounded-full px-2 py-0.5 surface-ghost"
            style="color: var(--brand); font-size: 12px;"
            onclick={() => void goto('/users/nearby')}
          >
            查看全部
          </button>
        </div>

        {#if !loading}
          <PeopleRadarStrip users={mapUsers()} onViewAll={() => void goto('/users/nearby')} />
        {:else}
          <PeopleRadarStrip loading />
        {/if}

        <div class="mt-6">
          <NearbyUsersFlatList
            users={mapUsers()}
            {loading}
            {followingIds}
            onToggleFollow={toggleFollow}
          />
        </div>

        <div class="mt-7">
          <div class="flex items-end justify-between mb-3">
            <span class="eyebrow">周边热度</span>
          </div>
          <ActivityList activities={trendingTags.slice(0, 3).map((t, i) => ({ id: t.name, icon: ['coffee' as const, 'music' as const, 'flame' as const][i] ?? 'star', title: `#${t.name}`, sub: `${t.usage_count}条动态`, color: ['#F59E0B', '#8B5CF6', '#EF4444'][i] ?? 'var(--brand)', bg: ['rgba(245,158,11,0.1)', 'rgba(139,92,246,0.1)', 'rgba(239,68,68,0.1)'][i] ?? 'var(--surface-bg)' }))} {loading} />
        </div>
      </div>
    </div>
  </div>
{/if}
