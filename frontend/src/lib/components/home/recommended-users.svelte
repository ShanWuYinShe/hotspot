<script lang="ts">
  import { dev } from '$app/environment';
  import { usersService } from '$lib/api/services';
  import { isSuccess } from '$lib/api/client';
  import { authStore } from '$lib/stores/auth.svelte';
  import { mapAnchorStore } from '$lib/stores/map-anchor.svelte';
  import type { UserItem } from '$lib/types/api';
  import Avatar from '$lib/components/common/avatar.svelte';
  import ChevronRight from 'lucide-svelte/icons/chevron-right';

  interface Props {
    refreshTrigger?: number;
  }

  let { refreshTrigger = 0 }: Props = $props();

  let users = $state.raw<UserItem[]>([]);
  let loading = $state(false);
  let loadVersion = 0;

  function formatDistance(meters?: number): string {
    if (meters == null) return '';
    if (meters < 1000) return `${Math.round(meters)}m`;
    return `${(meters / 1000).toFixed(1)}km`;
  }

  async function loadUsers() {
    const version = ++loadVersion;
    loading = true;
    try {
      const response = await usersService.search({
        radius: mapAnchorStore.radius * 1000,
        reference_point: {
          latitude: mapAnchorStore.latitude,
          longitude: mapAnchorStore.longitude
        },
        pagination: { limit: 6 }
      });
      if (version !== loadVersion) return;
      if (isSuccess(response) && response.data) {
        const currentUserId = authStore.currentUserId;
        const items = response.data.items ?? [];
        users = currentUserId ? items.filter((item) => item.user.user_id !== currentUserId) : items;
      }
    } catch (error) {
      if (version !== loadVersion) return;
      if (dev) console.error('Failed to load recommended users:', error);
    } finally {
      if (version === loadVersion) loading = false;
    }
  }

  $effect(() => {
    refreshTrigger;
    mapAnchorStore.anchor.updatedAt;
    loadUsers();
  });
</script>

{#if users.length > 0 || loading}
  <div>
    <div class="mb-3 flex items-center justify-between">
      <p
        class="uppercase"
        style="font-size: 10px; letter-spacing: 0.14em; color: var(--text-secondary);"
      >
        推荐用户
      </p>
      <a
        href="/users/nearby"
        class="flex items-center gap-0.5 outline-none"
        style="color: var(--brand);"
      >
        <span style="font-size: 12px;">查看更多</span>
        <ChevronRight size={13} strokeWidth={1.5} />
      </a>
    </div>
    {#if loading}
      <div class="flex flex-col gap-2.5">
        {#each Array(3) as _}
          <div
            class="flex items-center gap-3.5 rounded-[18px] px-4 py-3"
            style="background: var(--surface-bg); box-shadow: var(--shadow-lg); border: 1px solid var(--border-theme);"
          >
            <div
              class="rounded-full"
              style="width: 40px; height: 40px; background: var(--border-theme);"
            ></div>
            <div class="flex-1">
              <div
                class="mb-1 rounded"
                style="width: 60px; height: 12px; background: var(--border-theme);"
              ></div>
              <div
                class="rounded"
                style="width: 80px; height: 10px; background: var(--border-theme);"
              ></div>
            </div>
          </div>
        {/each}
      </div>
    {:else}
      <div class="flex flex-col gap-2.5">
        {#each users as user (user.user.user_id)}
          <a
            href="/users/{user.user.user_id}"
            class="flex items-center gap-3.5 rounded-[18px] px-4 py-3 row-hover transition-transform duration-150 active:scale-[0.98]"
            style="background: var(--surface-bg); box-shadow: var(--shadow-lg); border: 1px solid var(--border-theme);"
          >
            <div class="relative flex-shrink-0">
              <Avatar
                src={user.user.avatar_url}
                name={user.user.nickname}
                alt={user.user.nickname}
                size={40}
              />
              {#if user.presence_status === 'online'}
                <span
                  class="absolute bottom-0 right-0 rounded-full"
                  style="width: 9px; height: 9px; background: var(--semantic-online); border: 2px solid var(--surface-bg);"
                ></span>
              {/if}
            </div>
            <div class="min-w-0 flex-1">
              <p style="font-size: 13px; color: var(--text-primary);">{user.user.nickname}</p>
              <p style="font-size: 11px; color: var(--text-secondary); margin-top: 1px;">
                {#if user.tags && user.tags.length > 0}{user.tags[0]}{/if}
              </p>
            </div>
            {#if user.location?.distance != null}
              <span style="font-size: 12px; color: var(--brand);"
                >{formatDistance(user.location.distance)}</span
              >
            {/if}
            <ChevronRight size={14} strokeWidth={1.5} color="var(--text-secondary)" />
          </a>
        {/each}
      </div>
    {/if}
  </div>
{/if}
