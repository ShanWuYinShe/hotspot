<script lang="ts">
  import { goto } from '$app/navigation';
  import ChevronRight from 'lucide-svelte/icons/chevron-right';
  import Coffee from 'lucide-svelte/icons/coffee';
  import Music from 'lucide-svelte/icons/music';
  import Users from 'lucide-svelte/icons/users';
  import Star from 'lucide-svelte/icons/star';
  import Flame from 'lucide-svelte/icons/flame';

  interface Activity {
    id: string;
    icon: string;
    title: string;
    sub: string;
    color: string;
    bg: string;
  }

  const ICON_MAP: Record<string, typeof Coffee> = {
    coffee: Coffee,
    music: Music,
    users: Users,
    star: Star,
    flame: Flame
  };

  interface Props {
    activities?: Activity[];
    loading?: boolean;
  }

  let { activities = [], loading = false }: Props = $props();
</script>

<div
  class="flex flex-col overflow-hidden"
  style="gap: 1px; background: var(--border-theme); border-radius: 16px; border: 1px solid var(--border-theme);"
>
  {#if loading}
    {#each Array(3) as _}
      <div class="flex items-center gap-3 px-4 py-3" style="background: var(--surface-bg);">
        <div
          class="flex flex-shrink-0 items-center justify-center rounded-[10px]"
          style="width: 34px; height: 34px; background: var(--border-theme);"
        ></div>
        <div class="flex-1">
          <div
            style="height: 13px; width: 60%; border-radius: 3px; background: var(--border-theme);"
          ></div>
          <div
            style="height: 11px; width: 40%; border-radius: 3px; background: var(--border-theme); margin-top: 4px;"
          ></div>
        </div>
      </div>
    {/each}
  {:else}
    {#each activities as a (a.id)}
      {@const Icon = ICON_MAP[a.icon] ?? Coffee}
      <button
        class="flex items-center gap-3 px-4 py-3 text-left transition-colors"
        style="background: var(--surface-bg);"
        onmouseenter={(e) => (e.currentTarget.style.background = 'var(--border-theme)')}
        onmouseleave={(e) => (e.currentTarget.style.background = 'var(--surface-bg)')}
        onclick={() => void goto(`/events/${a.id}`)}
      >
        <div
          class="flex flex-shrink-0 items-center justify-center rounded-[10px]"
          style="width: 34px; height: 34px; background: {a.bg};"
        >
          <Icon size={16} strokeWidth={1.75} color={a.color} />
        </div>
        <div class="min-w-0 flex-1">
          <p style="font-size: 13px; color: var(--text-primary);">{a.title}</p>
          <p style="font-size: 11px; color: var(--text-secondary); margin-top: 1px;">
            {a.sub}
          </p>
        </div>
        <ChevronRight size={13} strokeWidth={1.5} color="var(--text-secondary)" />
      </button>
    {/each}
  {/if}
</div>
