<script lang="ts">
  import { dev } from '$app/environment';
  import { groupsService } from '$lib/api/services';
  import { isSuccess } from '$lib/api/client';
  import { mapAnchorStore } from '$lib/stores/map-anchor.svelte';
  import type { GroupItem } from '$lib/types/api';
  import ChevronRight from 'lucide-svelte/icons/chevron-right';

  interface Props {
    refreshTrigger?: number;
  }

  let { refreshTrigger = 0 }: Props = $props();

  let groups = $state.raw<GroupItem[]>([]);
  let loading = $state(false);
  let loadVersion = 0;

  async function loadGroups() {
    const version = ++loadVersion;
    loading = true;
    try {
      const response = await groupsService.search({
        radius: mapAnchorStore.radius * 1000,
        reference_point: {
          latitude: mapAnchorStore.latitude,
          longitude: mapAnchorStore.longitude
        },
        pagination: { limit: 6 }
      });
      if (version !== loadVersion) return;
      if (isSuccess(response) && response.data) {
        groups = response.data.items ?? [];
      }
    } catch (error) {
      if (version !== loadVersion) return;
      if (dev) console.error('Failed to load recommended groups:', error);
    } finally {
      if (version === loadVersion) loading = false;
    }
  }

  $effect(() => {
    refreshTrigger;
    mapAnchorStore.anchor.updatedAt;
    loadGroups();
  });
</script>

{#if groups.length > 0 || loading}
  <div>
    <div class="mb-3 flex items-center justify-between">
      <p
        class="uppercase"
        style="font-size: 10px; letter-spacing: 0.14em; color: var(--neu-text-muted);"
      >
        推荐群组
      </p>
      <a
        href="/groups/nearby"
        class="flex items-center gap-0.5 outline-none transition-transform duration-150 active:scale-[0.97]"
        style="color: var(--neu-accent-primary);"
      >
        <span style="font-size: 12px;">查看更多</span>
        <ChevronRight size={13} strokeWidth={1.5} />
      </a>
    </div>
    {#if loading}
      <div class="flex flex-col gap-2.5">
        {#each Array(3) as _}
          <div
            class="flex items-center gap-3 rounded-[18px] px-4 py-3"
            style="background: var(--neu-bg); box-shadow: var(--neu-shadow-lg); border: 1px solid var(--neu-border);"
          >
            <div
              class="rounded-[12px]"
              style="width: 36px; height: 36px; background: var(--neu-border);"
            ></div>
            <div class="flex-1">
              <div
                class="mb-1 rounded"
                style="width: 80px; height: 12px; background: var(--neu-border);"
              ></div>
              <div
                class="rounded"
                style="width: 60px; height: 10px; background: var(--neu-border);"
              ></div>
            </div>
          </div>
        {/each}
      </div>
    {:else}
      <div class="flex flex-col gap-2.5">
        {#each groups as group (group.group.group_id)}
          <a
            href="/groups/{group.group.group_id}"
            class="flex items-center gap-3 rounded-[18px] px-4 py-3 neu-row-hover transition-transform duration-150 active:scale-[0.98]"
            style="background: var(--neu-bg); box-shadow: var(--neu-shadow-lg); border: 1px solid var(--neu-border);"
          >
            <div
              class="flex flex-shrink-0 items-center justify-center rounded-[12px]"
              style="width: 36px; height: 36px; background: var(--neu-fallback-bg);"
            >
              <span style="font-size: 14px; font-weight: 600; color: var(--neu-fallback-text);">
                {group.group.name?.charAt(0) ?? 'G'}
              </span>
            </div>
            <div class="min-w-0 flex-1">
              <p style="font-size: 13px; color: var(--neu-text);">{group.group.name}</p>
              <p style="font-size: 11px; color: var(--neu-text-muted); margin-top: 1px;">
                {group.group.members_count ?? 0}人
                {#if group.location?.distance != null}· {Math.round(group.location.distance)}m{/if}
              </p>
            </div>
            <ChevronRight size={14} strokeWidth={1.5} color="var(--neu-text-muted)" />
          </a>
        {/each}
      </div>
    {/if}
  </div>
{/if}
