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
      style="font-size: 12px; color: var(--brand); background: none; border: none; cursor: pointer;"
      onclick={() => void goto('/groups/nearby')}
    >
      更多 →
    </button>
  </div>

  <div class="flex gap-3 overflow-x-auto no-scrollbar" style="padding-bottom: 4px;">
    {#if loading}
      {#each Array(3) as _}
        <div class="flex-shrink-0" style="width: 140px;">
          <div
            style="height: 72px; border-radius: 14px 14px 0 0; background: var(--border-theme);"
          ></div>
          <div
            style="padding: 8px 10px; border-radius: 0 0 14px 14px; background: var(--surface-bg); border: 1px solid var(--border-theme); border-top: none;"
          >
            <div
              style="height: 12px; width: 80%; border-radius: 3px; background: var(--border-theme);"
            ></div>
          </div>
        </div>
      {/each}
    {:else}
      {#each groups as group (group.id)}
        <button
          class="flex-shrink-0 text-left"
          style="width: 140px; background: none; border: none; padding: 0; cursor: pointer;"
          onclick={() => void goto(`/groups/${group.id}`)}
        >
          <!-- Banner -->
          <div
            style="position: relative; height: 72px; border-radius: 14px 14px 0 0; overflow: hidden; background: linear-gradient(135deg, var(--neu-accent-soft), var(--border-theme));"
          >
            {#if group.banner_url}
              <img
                src={group.banner_url}
                alt={group.name}
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
                  style="width: 20px; height: 20px; border-radius: 6px; background: rgba(0,0,0,0.4); backdrop-filter: blur(4px); display: flex; align-items: center; justify-content: center;"
                >
                  <Lock size={10} strokeWidth={2} color="white" />
                </div>
              {:else}
                <div
                  style="width: 20px; height: 20px; border-radius: 6px; background: rgba(0,0,0,0.4); backdrop-filter: blur(4px); display: flex; align-items: center; justify-content: center;"
                >
                  <Globe size={10} strokeWidth={2} color="white" />
                </div>
              {/if}
            </div>
            <!-- Distance badge -->
            {#if group.distance}
              <div
                style="position: absolute; bottom: 6px; left: 6px; font-size: 9px; color: white; background: rgba(0,0,0,0.35); backdrop-filter: blur(4px); padding: 2px 6px; border-radius: 6px;"
              >
                {group.distance}
              </div>
            {/if}
          </div>
          <!-- Info -->
          <div
            style="padding: 8px 10px; border-radius: 0 0 14px 14px; background: var(--surface-bg); border: 1px solid var(--border-theme); border-top: none;"
          >
            <p
              style="font-size: 12px; font-weight: 500; color: var(--text-primary); white-space: nowrap; overflow: hidden; text-overflow: ellipsis;"
            >
              {group.name}
            </p>
            {#if group.member_count}
              <div class="flex items-center gap-1" style="margin-top: 2px;">
                <Users size={10} strokeWidth={1.5} color="var(--text-secondary)" />
                <span style="font-size: 10px; color: var(--text-secondary);"
                  >{group.member_count}人</span
                >
              </div>
            {/if}
          </div>
        </button>
      {/each}
    {/if}
  </div>
</div>
