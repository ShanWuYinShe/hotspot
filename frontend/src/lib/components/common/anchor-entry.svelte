<script lang="ts">
  import Navigation2 from 'lucide-svelte/icons/navigation-2';
  import MapPin from 'lucide-svelte/icons/map-pin';
  import ChevronRight from 'lucide-svelte/icons/chevron-right';
  import { mapAnchorStore } from '$lib/stores/map-anchor.svelte';
  import { t } from '$lib/i18n';
  import LocationPickerModal from '$lib/components/common/location-picker-modal.svelte';

  interface Props {
    /** `card` = full sidebar block, `pill` = compact header chip */
    variant?: 'card' | 'pill';
    /** Override the eyebrow label (defaults to the canonical browse-anchor label) */
    label?: string;
    /** When provided, the host owns the picker; otherwise this component mounts its own */
    onAdjust?: () => void;
  }

  let { variant = 'pill', label, onAdjust }: Props = $props();

  let internalOpen = $state(false);
  const resolvedLabel = $derived(label ?? t('sidebar.browseAnchor'));

  function handleClick() {
    if (onAdjust) onAdjust();
    else internalOpen = true;
  }
</script>

{#if variant === 'card'}
  <button
    type="button"
    class="flex w-full items-center gap-3 rounded-[var(--radius-md)] text-left outline-none transition-transform duration-150 active:scale-[0.98]"
    style="padding: 10px 12px; background: var(--surface-bg); border: 1px solid var(--border-theme); box-shadow: 5px 5px 12px var(--shadow-dark), -5px -5px 12px var(--shadow-light);"
    onclick={handleClick}
  >
    <div
      class="flex flex-shrink-0 items-center justify-center rounded-full"
      style="width: 34px; height: 34px; background: var(--brand); box-shadow: var(--shadow-md);"
    >
      <Navigation2 size={14} strokeWidth={2} color="var(--text-on-accent)" />
    </div>
    <div class="min-w-0 flex-1">
      <p
        style="font-size: 10px; line-height: 1.4; color: var(--text-secondary); letter-spacing: 0.06em;"
      >
        {resolvedLabel}
      </p>
      <div class="flex items-center gap-1.5">
        <p class="truncate" style="font-size: 13px; line-height: 1.4; color: var(--text-primary);">
          {mapAnchorStore.displayName}
        </p>
        <span
          class="flex-shrink-0"
          style="font-size: 10px; line-height: 1.5; padding: 0 6px; border-radius: var(--radius-pill); background: color-mix(in srgb, var(--brand) 12%, transparent); color: var(--brand);"
        >
          {mapAnchorStore.radius}km
        </span>
      </div>
    </div>
    <ChevronRight size={14} strokeWidth={1.5} color="var(--text-secondary)" class="flex-shrink-0" />
  </button>
{:else}
  <button
    type="button"
    class="surface-flat flex max-w-[55vw] items-center gap-1.5 rounded-full px-3.5 py-1.5 outline-none transition-transform duration-150 active:scale-[0.94]"
    onclick={handleClick}
    aria-label={resolvedLabel}
  >
    <MapPin size={12} strokeWidth={1.5} color="var(--brand)" class="flex-shrink-0" />
    <span class="truncate" style="font-size: 11px; color: var(--text-secondary);">
      {mapAnchorStore.displayName}
    </span>
    <span class="flex-shrink-0" style="font-size: 11px; color: var(--brand);">
      · {mapAnchorStore.radius}km
    </span>
  </button>
{/if}

{#if !onAdjust}
  <LocationPickerModal bind:open={internalOpen} />
{/if}
