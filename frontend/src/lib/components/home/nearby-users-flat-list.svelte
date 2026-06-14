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

<div
  class="flex flex-col overflow-hidden"
  style="gap: 1px; background: var(--border-theme); border-radius: 16px; border: 1px solid var(--border-theme);"
>
  {#if loading}
    {#each Array(4) as _}
      <div class="flex items-center gap-3 px-4 py-3" style="background: var(--surface-bg);">
        <div
          class="flex-shrink-0 rounded-full"
          style="width: 38px; height: 38px; background: var(--border-theme);"
        ></div>
        <div class="flex-1">
          <div
            style="height: 13px; width: 50%; border-radius: 3px; background: var(--border-theme);"
          ></div>
          <div
            style="height: 10px; width: 30%; border-radius: 3px; background: var(--border-theme); margin-top: 4px;"
          ></div>
        </div>
      </div>
    {/each}
  {:else}
    {#each users as user (user.id)}
      <!-- svelte-ignore a11y_no_static_element_interactions -->
      <div
        class="flex items-center gap-3 px-4 py-3 text-left transition-colors cursor-pointer"
        style="background: var(--surface-bg);"
        role="button"
        tabindex="0"
        onmouseenter={(e) => (e.currentTarget.style.background = 'var(--border-theme)')}
        onmouseleave={(e) => (e.currentTarget.style.background = 'var(--surface-bg)')}
        onclick={() => void goto(`/users/${user.id}`)}
        onkeydown={(e) => {
          if (e.key === 'Enter' || e.key === ' ') {
            e.preventDefault();
            void goto(`/users/${user.id}`);
          }
        }}
      >
        <div class="relative flex-shrink-0">
          <Avatar src={user.avatar_url} name={user.nickname} alt={user.nickname} size={38} />
          {#if user.online}
            <span
              class="absolute bottom-0 right-0 rounded-full"
              style="width: 9px; height: 9px; background: var(--semantic-online); border: 2px solid var(--surface-bg);"
            ></span>
          {/if}
        </div>
        <div class="min-w-0 flex-1">
          <p style="font-size: 13px; color: var(--text-primary);">{user.nickname}</p>
          <div class="flex items-center gap-1.5" style="margin-top: 2px;">
            {#if user.tag}
              <span style="font-size: 10px; color: var(--brand);">#{user.tag}</span>
              <span style="font-size: 10px; color: var(--text-secondary); opacity: 0.5;">·</span>
            {/if}
            {#if user.distance}
              <span style="font-size: 10px; color: var(--text-secondary);">{user.distance}</span>
            {/if}
          </div>
        </div>
      </div>
    {/each}
  {/if}
</div>
