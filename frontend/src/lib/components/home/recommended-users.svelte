<script lang="ts">
  import { usersService } from '$lib/api/services';
  import { isSuccess } from '$lib/api/client';
  import { formatDistance } from '$lib/utils/format';
  import { authStore } from '$lib/stores/auth.svelte';
  import { mapAnchorStore } from '$lib/stores/map-anchor.svelte';
  import { useWidget } from '$lib/hooks/use-widget.svelte';
  import type { UserItem } from '$lib/types/api';
  import Avatar from '$lib/components/common/avatar.svelte';
  import ChevronRight from 'lucide-svelte/icons/chevron-right';

  interface Props {
    refreshTrigger?: number;
  }

  let { refreshTrigger = 0 }: Props = $props();

  const {
    data: users,
    loading,
    reload
  } = useWidget<UserItem[]>(
    async () => {
      const response = await usersService.search({
        radius: mapAnchorStore.radius * 1000,
        reference_point: {
          latitude: mapAnchorStore.latitude,
          longitude: mapAnchorStore.longitude
        },
        pagination: { limit: 6 }
      });
      if (!isSuccess(response) || !response.data)
        throw new Error('Failed to load recommended users');
      const currentUserId = authStore.currentUserId;
      const items = response.data.items ?? [];
      return currentUserId ? items.filter((item) => item.user.user_id !== currentUserId) : items;
    },
    { locationAware: true }
  );

  $effect(() => {
    refreshTrigger;
    reload();
  });
</script>

{#if (users?.length ?? 0) > 0 || loading}
  <div>
    <div class="mb-3 flex items-center justify-between">
      <p class="text-xs uppercase" style="letter-spacing: 0.14em; color: var(--text-secondary)">
        推荐用户
      </p>
      <a
        href="/users/nearby"
        class="flex items-center gap-0.5 outline-none"
        style="color: var(--brand);"
      >
        <span class="text-meta">查看更多</span>
        <ChevronRight size={13} strokeWidth={1.5} />
      </a>
    </div>
    {#if loading}
      <div class="flex flex-col gap-2.5">
        {#each Array(3) as _}
          <div
            class="flex items-center gap-3.5 rounded-[var(--radius-md)] px-4 py-3"
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
        {#each users ?? [] as user (user.user.user_id)}
          <a
            href="/users/{user.user.user_id}"
            class="flex items-center gap-3.5 rounded-[var(--radius-md)] px-4 py-3 row-hover transition-transform duration-150 active:scale-[0.98]"
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
              <p class="text-body" style="color: var(--text-primary)">{user.user.nickname}</p>
              <p class="text-caption" style="color: var(--text-secondary); margin-top: 1px">
                {#if user.tags && user.tags.length > 0}{user.tags[0]}{/if}
              </p>
            </div>
            {#if user.location?.distance != null}
              <span class="text-meta" style="color: var(--brand)"
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
