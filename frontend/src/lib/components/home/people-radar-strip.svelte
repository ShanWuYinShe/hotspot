<script lang="ts">
  import ChevronRight from 'lucide-svelte/icons/chevron-right';
  import Avatar from '$lib/components/common/avatar.svelte';

  interface NearbyUser {
    id: string;
    avatar_url: string;
    nickname: string;
    distance?: string;
    online?: boolean;
  }

  interface Props {
    users?: NearbyUser[];
    loading?: boolean;
    onViewAll?: () => void;
    showHeader?: boolean;
  }

  let { users = [], loading = false, onViewAll, showHeader = true }: Props = $props();
</script>

<div>
  {#if showHeader}
    <div class="flex items-center justify-between mb-3">
      <span class="eyebrow">附近的人</span>
      <button
        class="flex items-center gap-0.5"
        style="font-size: 12px; color: var(--brand); background: none; border: none; cursor: pointer;"
        onclick={onViewAll}
      >
        更多
        <ChevronRight size={14} strokeWidth={1.5} />
      </button>
    </div>
  {/if}

  <div class="flex gap-3 overflow-x-auto no-scrollbar" style="padding-bottom: 4px;">
    {#if loading}
      {#each Array(5) as _}
        <div class="flex flex-col items-center gap-1.5 flex-shrink-0">
          <div
            style="width: 56px; height: 56px; border-radius: 9999px; border: 1.5px solid var(--border-theme); background: var(--border-theme);"
          ></div>
        </div>
      {/each}
    {:else}
      {#each users as user (user.id)}
        <div
          class="flex flex-col items-center gap-1 flex-shrink-0"
          style="cursor: pointer; width: 56px;"
        >
          <div style="position: relative; width: 56px; height: 56px;">
            <Avatar src={user.avatar_url} name={user.nickname} alt={user.nickname} size={48} />
            {#if user.online}
              <div
                style="position: absolute; bottom: 1px; right: 1px; width: 11px; height: 11px; border-radius: 9999px; background: var(--semantic-online); border: 2px solid var(--surface-bg);"
              ></div>
            {/if}
          </div>
          <span
            style="font-size: 12px; color: var(--text-primary); max-width: 56px; overflow: hidden; text-overflow: ellipsis; white-space: nowrap;"
            >{user.nickname}</span
          >
          {#if user.distance}
            <span style="font-size: 10px; color: var(--brand); margin-top: -2px;"
              >{user.distance}</span
            >
          {/if}
        </div>
      {/each}

      <!-- More card -->
      <div class="flex flex-col items-center gap-1 flex-shrink-0" style="width: 56px;">
        <button
          class="flex items-center justify-center"
          style="width: 56px; height: 56px; border-radius: 9999px; border: 1.5px dashed var(--border-active); background: transparent; cursor: pointer;"
          onclick={onViewAll}
        >
          <ChevronRight size={18} strokeWidth={1.5} color="var(--text-secondary)" />
        </button>
      </div>
    {/if}
  </div>
</div>
