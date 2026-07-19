<script lang="ts">
  import { goto } from '$app/navigation';
  import Lock from 'lucide-svelte/icons/lock';
  import Globe from 'lucide-svelte/icons/globe';
  import Users from 'lucide-svelte/icons/users';

  interface Group {
    id: string;
    name: string;
    banner_url?: string;
    member_count?: number;
    distance?: string;
    is_private?: boolean;
  }

  interface Props {
    groups?: Group[];
    loading?: boolean;
  }

  let { groups = [], loading = false }: Props = $props();
</script>

<div>
  <div class="flex items-center justify-between mb-3">
    <span class="eyebrow">附近群组</span>
    <button
      class="text-meta outline-none transition-all duration-150 hover:opacity-70 active:scale-[0.93]"
      style="color: var(--brand); background: none; border: none; cursor: pointer"
      onclick={() => void goto('/groups/nearby')}
      aria-label="查看全部群组"
    >
      更多 →
    </button>
  </div>

  <div class="flex gap-3 overflow-x-auto no-scrollbar h-scroll" style="padding-bottom: 4px;">
    {#if loading}
      {#each Array(3) as _}
        <div class="flex-shrink-0 w-[140px]">
          <div
            style="height: 72px; border-radius: var(--radius-lg) var(--radius-lg) 0 0; background: var(--skeleton-bg);"
          ></div>
          <div
            style="padding: 8px 10px; border-radius: 0 0 var(--radius-lg) var(--radius-lg); background: var(--surface-bg); border: 1px solid var(--border-theme); border-top: none;"
          >
            <div
              style="height: 12px; width: 80%; border-radius: var(--radius-xs); background: var(--skeleton-bg);"
            ></div>
          </div>
        </div>
      {/each}
    {:else if groups.length > 0}
	      {#each groups as group, i (group.id)}
        <button
          class="group-card card-hover flex-shrink-0 text-left"
          onclick={() => void goto(`/groups/${group.id}`)}
        >
          <!-- Banner -->
          <div
            style="position: relative; height: 72px; border-radius: var(--radius-lg) var(--radius-lg) 0 0; overflow: hidden; background: linear-gradient(135deg, var(--theme-accent-soft), var(--border-theme));"
          >
            {#if group.banner_url}
	              <img
	                src={group.banner_url}
	                alt={group.name}
	                loading={i === 0 ? 'eager' : 'lazy'}
	                decoding="async"
	                fetchpriority={i === 0 ? 'high' : 'auto'}
	                style="width: 100%; height: 100%; object-fit: cover;"
              />
            {/if}
            <div
              style="position: absolute; inset: 0; background: linear-gradient(to top, rgba(0,0,0,0.4) 0%, transparent 60%);"
            ></div>
            <!-- Privacy badge -->
            <div style="position: absolute; top: 6px; right: 6px;">
              {#if group.is_private}
                <div
                  style="width: 20px; height: 20px; border-radius: var(--radius-xs); background: rgba(0,0,0,0.4); backdrop-filter: blur(4px); display: flex; align-items: center; justify-content: center;"
                >
                  <Lock size={10} strokeWidth={2} color="var(--text-on-overlay)" />
                </div>
              {:else}
                <div
                  style="width: 20px; height: 20px; border-radius: var(--radius-xs); background: rgba(0,0,0,0.4); backdrop-filter: blur(4px); display: flex; align-items: center; justify-content: center;"
                >
                  <Globe size={10} strokeWidth={2} color="var(--text-on-overlay)" />
                </div>
              {/if}
            </div>
            <!-- Distance badge -->
            {#if group.distance}
              <div
                class="text-xs"
                style="position: absolute; bottom: 6px; left: 6px; color: var(--text-on-overlay); background: rgba(0,0,0,0.35); backdrop-filter: blur(4px); padding: 2px 6px; border-radius: var(--radius-xs)"
              >
                {group.distance}
              </div>
            {/if}
          </div>
          <!-- Info -->
          <div
            style="padding: 8px 10px; border-radius: 0 0 var(--radius-lg) var(--radius-lg); background: var(--surface-bg); border: 1px solid var(--border-theme); border-top: none;"
          >
            <p
              class="text-meta"
              style="font-weight: 500; color: var(--text-primary); white-space: nowrap; overflow: hidden; text-overflow: ellipsis"
            >
              {group.name}
            </p>
            {#if group.member_count}
              <div class="flex items-center gap-1" style="margin-top: 2px;">
                <Users size={10} strokeWidth={1.5} color="var(--text-secondary)" />
                <span class="text-xs" style="color: var(--text-secondary)"
                  >{group.member_count}人</span
                >
              </div>
            {/if}
          </div>
        </button>
      {/each}
    {:else}
      <p class="text-meta" style="color: var(--text-secondary); padding: 2px">
        附近暂无群组，点右上「更多」浏览全部。
      </p>
    {/if}
  </div>
</div>

<style>
  .group-card {
    width: 140px;
    background: none;
    border: none;
    padding: 0;
    cursor: pointer;
  }
</style>
