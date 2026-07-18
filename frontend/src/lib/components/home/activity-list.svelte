<script lang="ts">
  import { goto } from '$app/navigation';
  import ChevronRight from 'lucide-svelte/icons/chevron-right';
  import Coffee from 'lucide-svelte/icons/coffee';
  import Music from 'lucide-svelte/icons/music';
  import Users from 'lucide-svelte/icons/users';
  import Star from 'lucide-svelte/icons/star';
  import Flame from 'lucide-svelte/icons/flame';
  import Skeleton from '$lib/components/ui/skeleton/skeleton.svelte';

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
    onSelect?: (id: string) => void;
  }

  let { activities = [], loading = false, onSelect }: Props = $props();

  function handleSelect(id: string) {
    if (onSelect) onSelect(id);
    else void goto(`/discover?tag=${encodeURIComponent(id)}`);
  }
</script>

<div
  class="flex flex-col overflow-hidden"
  style="gap: 1px; background: var(--border-theme); border-radius: var(--radius-md); border: 1px solid var(--border-theme);"
>
  {#if loading}
    {#each Array(3) as _}
      <div class="flex items-center gap-3 px-4 py-3" style="background: var(--surface-bg);">
        <Skeleton class="flex-shrink-0 rounded-[var(--radius-xs)]" style="width: 34px; height: 34px;" />
        <div class="flex-1 space-y-1">
          <Skeleton class="rounded-[var(--radius-xs)]" style="height: 13px; width: 60%;" />
          <Skeleton class="rounded-[var(--radius-xs)]" style="height: 11px; width: 40%;" />
        </div>
      </div>
    {/each}
  {:else}
    {#each activities as a (a.id)}
      {@const Icon = ICON_MAP[a.icon] ?? Coffee}
      <button
        class="hover-row flex items-center gap-3 px-4 py-3 text-left transition-colors w-full"
        onclick={() => handleSelect(a.id)}
      >
        <div
          class="flex flex-shrink-0 items-center justify-center rounded-[var(--radius-xs)]"
          style="width: 34px; height: 34px; background: {a.bg};"
        >
          <Icon size={16} strokeWidth={1.75} color={a.color} />
        </div>
        <div class="min-w-0 flex-1">
          <p class="text-body" style="color: var(--text-primary)">{a.title}</p>
          <p class="text-caption" style="color: var(--text-secondary); margin-top: 1px">
            {a.sub}
          </p>
        </div>
        <ChevronRight size={13} strokeWidth={1.5} color="var(--text-secondary)" />
      </button>
    {/each}
  {/if}
</div>

<style>
  .hover-row {
    background-color: var(--surface-bg);
  }
  @media (hover: hover) {
    .hover-row:hover {
      background-color: var(--border-theme);
    }
  }
</style>
