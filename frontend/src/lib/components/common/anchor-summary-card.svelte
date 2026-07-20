<script lang="ts">
  import MapPin from 'lucide-svelte/icons/map-pin';
  import { mapAnchorStore } from '$lib/stores/map-anchor.svelte';
  import { t } from '$lib/i18n';
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
    label,
    description = null,
    subtle: _subtle = false,
    onAdjust,
    overrideName = null,
    actions
  }: AnchorSummaryCardProps = $props();

  const resolvedLabel = $derived(label ?? t('anchorSummary.label'));
  const displayName = $derived(overrideName ?? mapAnchorStore.displayName);
</script>

<div
  class="anchor-card card-surface rounded-[var(--radius-lg)] p-4"
>
  <div class="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
    <div class="min-w-0">
      <p class="text-xs uppercase" style="letter-spacing: 0.14em; color: var(--text-secondary)">
        {resolvedLabel}
      </p>
      <p class="text-body-lg" style="font-weight: 600; color: var(--text-primary); margin-top: 4px">
        {displayName || t('anchorSummary.notSet')}
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
        class="card-action"
        style="min-height:34px;padding:8px 16px;font-size:13px;font-weight:500;gap:6px;"
        onclick={onAdjust}
      >
        <MapPin size={15} strokeWidth={1.5} color="var(--brand)" />
        {t('anchorSummary.adjust')}
      </button>
    {/if}
  </div>
</div>

<style>
</style>
