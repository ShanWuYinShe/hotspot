<script lang="ts">
  import { goto } from '$app/navigation';
  import Avatar from '$lib/components/common/avatar.svelte';

  interface NearbyUser {
    id: string;
    avatar_url: string;
    nickname: string;
    distance?: string;
    online?: boolean;
    tag?: string;
  }

  interface Props {
    users?: NearbyUser[];
    loading?: boolean;
  }

  let { users = [], loading = false }: Props = $props();
</script>

<div class="flex flex-col overflow-hidden rounded-[var(--radius-md)]">
  {#if loading}
    {#each Array(4) as _}
      <div class="flex items-center gap-3 px-4 py-3 border-b border-[var(--border-theme)] last:border-b-0" style="background: var(--surface-bg);">
        <div
          class="flex-shrink-0 rounded-full"
          style="width: 38px; height: 38px; background: var(--border-theme);"
        ></div>
        <div class="flex-1">
          <div
            style="height: 13px; width: 50%; border-radius: var(--radius-xs); background: var(--border-theme);"
          ></div>
          <div
            style="height: 10px; width: 30%; border-radius: var(--radius-xs); background: var(--border-theme); margin-top: 4px;"
          ></div>
        </div>
      </div>
    {/each}
  {:else}
    {#each users as user (user.id)}
      <!-- svelte-ignore a11y_no_static_element_interactions -->
      <div
        class="surface-row flex items-center gap-3 px-4 py-3 text-left cursor-pointer border-b border-[var(--border-theme)] last:border-b-0"
        role="button"
        tabindex="0"
        onclick={() => void goto(`/users/${user.id}`)}
        onkeydown={(e) => {
          if (e.key === 'Enter' || e.key === ' ') {
            e.preventDefault();
            void goto(`/users/${user.id}`);
          }
        }}
      >
        <Avatar
          class="flex-shrink-0"
          src={user.avatar_url}
          name={user.nickname}
          alt={user.nickname}
          size={40}
          online={user.online}
          onlineSize="sm"
        />
        <div class="min-w-0 flex-1">
          <p class="text-body-lg truncate" style="color: var(--text-primary)">{user.nickname}</p>
          <div class="flex items-center gap-1.5" style="margin-top: 2px;">
            {#if user.tag}
              <span class="text-xs" style="color: var(--brand)">#{user.tag}</span>
              <span class="text-xs" style="color: var(--text-secondary); opacity: 0.5">·</span>
            {/if}
            {#if user.distance}
              <span class="text-xs" style="color: var(--brand)">{user.distance}</span>
            {/if}
          </div>
        </div>
      </div>
    {/each}
  {/if}
</div>
