<script lang="ts">
  import MapPin from 'lucide-svelte/icons/map-pin';
  import { mapAnchorStore } from '$lib/stores/map-anchor.svelte';
  import type { Snippet } from 'svelte';

  interface AnchorSummaryCardProps {
    label?: string;
    description?: string | null;
    subtle?: boolean;
    onAdjust?: () => void;
    overrideName?: string | null;
    actions?: Snippet;
  }

  let {
    label = '当前锚点',
    description = null,
    subtle: _subtle = false,
    onAdjust,
    overrideName = null,
    actions
  }: AnchorSummaryCardProps = $props();

  const displayName = $derived(overrideName ?? mapAnchorStore.displayName);
</script>

<div
  class="rounded-[var(--radius-lg)] p-4"
  style="background: var(--surface-bg); box-shadow: var(--shadow-lg); border: 1px solid var(--border-theme);"
>
  <div class="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
    <div class="min-w-0">
      <p class="text-xs uppercase" style="letter-spacing: 0.14em; color: var(--text-secondary)">
        {label}
      </p>
      <p class="text-body-lg" style="font-weight: 600; color: var(--text-primary); margin-top: 4px">
        {displayName || '未设置锚点'}
      </p>
      {#if description}
        <p
          class="text-meta"
          style="color: var(--text-secondary); margin-top: 2px; line-height: 1.5"
        >
          {description}
        </p>
      {/if}
    </div>

    {#if actions}
      <div class="flex flex-col items-stretch gap-2 sm:flex-row sm:items-center sm:justify-end">
        {@render actions()}
      </div>
    {:else if onAdjust}
      <button
        class="text-body flex items-center gap-2 rounded-full px-4 py-2 outline-none"
        style="background: var(--surface-bg); box-shadow: var(--shadow-md); border: 1px solid var(--border-theme); color: var(--text-primary)"
        onclick={onAdjust}
      >
        <MapPin size={15} strokeWidth={1.5} color="var(--brand)" />
        调整位置
      </button>
    {/if}
  </div>
</div>
