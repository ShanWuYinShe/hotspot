<script lang="ts">
  import { groupsService } from '$lib/api/services';
  import { isSuccess } from '$lib/api/client';
  import { mapAnchorStore } from '$lib/stores/map-anchor.svelte';
  import { useWidget } from '$lib/hooks/use-widget.svelte';
  import type { GroupItem } from '$lib/types/api';
  import Avatar from '$lib/components/common/avatar.svelte';
  import ChevronRight from 'lucide-svelte/icons/chevron-right';

  interface Props {
    refreshTrigger?: number;
  }

  let { refreshTrigger = 0 }: Props = $props();

  const { data: groups, loading, reload } = useWidget<GroupItem[]>(async () => {
    const response = await groupsService.search({
      radius: mapAnchorStore.radius * 1000,
      reference_point: {
        latitude: mapAnchorStore.latitude,
        longitude: mapAnchorStore.longitude
      },
      pagination: { limit: 6 }
    });
    if (!isSuccess(response) || !response.data) throw new Error('Failed to load recommended groups');
    return response.data.items ?? [];
  }, { locationAware: true });

  $effect(() => {
    refreshTrigger;
    reload();
  });
</script>

{#if (groups?.length ?? 0) > 0 || loading}
  <div>
    <div class="mb-3 flex items-center justify-between">
      <p
        class="uppercase"
        style="font-size: 10px; letter-spacing: 0.14em; color: var(--text-secondary);"
      >
        推荐群组
      </p>
      <a
        href="/groups/nearby"
        class="flex items-center gap-0.5 outline-none transition-transform duration-150 active:scale-[0.97]"
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
            class="flex items-center gap-3 rounded-[var(--radius-md)] px-4 py-3"
            style="background: var(--surface-bg); box-shadow: var(--shadow-lg); border: 1px solid var(--border-theme);"
          >
            <div
              class="rounded-full"
              style="width: 40px; height: 40px; background: var(--border-theme);"
            ></div>
            <div class="flex-1">
              <div
                class="mb-1 rounded"
                style="width: 80px; height: 12px; background: var(--border-theme);"
              ></div>
              <div
                class="rounded"
                style="width: 60px; height: 10px; background: var(--border-theme);"
              ></div>
            </div>
          </div>
        {/each}
      </div>
    {:else}
      <div class="flex flex-col gap-2.5">
        {#each groups ?? [] as group (group.group.group_id)}
          <a
            href="/groups/{group.group.group_id}"
            class="flex items-center gap-3 rounded-[var(--radius-md)] px-4 py-3 row-hover transition-transform duration-150 active:scale-[0.98]"
            style="background: var(--surface-bg); box-shadow: var(--shadow-lg); border: 1px solid var(--border-theme);"
          >
            <div class="flex-shrink-0">
              <Avatar
                src={group.group.avatar_url}
                name={group.group.name}
                alt={group.group.name}
                size={40}
                type="group"
              />
            </div>
            <div class="min-w-0 flex-1">
              <p style="font-size: 13px; color: var(--text-primary);">{group.group.name}</p>
              <p style="font-size: 11px; color: var(--text-secondary); margin-top: 1px;">
                {group.group.members_count ?? 0}人
                {#if group.location?.distance != null}· {Math.round(group.location.distance)}m{/if}
              </p>
            </div>
            <ChevronRight size={14} strokeWidth={1.5} color="var(--text-secondary)" />
          </a>
        {/each}
      </div>
    {/if}
  </div>
{/if}
