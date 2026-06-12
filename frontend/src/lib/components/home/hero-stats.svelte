<script lang="ts">
  import Radar from 'lucide-svelte/icons/radar';
  import ChevronRight from 'lucide-svelte/icons/chevron-right';

  interface Props {
    people?: number;
    activity?: number;
    online?: number;
    title?: string;
    rangeLabel?: string;
    peopleLabel?: string;
    activityLabel?: string;
    onlineLabel?: string;
    loading?: boolean;
    onclick?: () => void;
  }

  let {
    people = 0,
    activity = 0,
    online = 0,
    title = '实时附近',
    rangeLabel = '持续更新',
    peopleLabel = '人附近',
    activityLabel = '活动中',
    onlineLabel = '在线',
    loading = false,
    onclick
  }: Props = $props();

  const stats = $derived([
    { label: peopleLabel, value: people, color: 'var(--brand)' },
    { label: activityLabel, value: activity, color: 'var(--semantic-warning)' },
    { label: onlineLabel, value: online, color: 'var(--semantic-online)' }
  ]);
</script>

<div class="ambient-glow">
  <button
    type="button"
    class="surface-hero w-full overflow-hidden rounded-[24px] text-left transition-transform duration-150 active:scale-[0.99]"
    style="padding: 18px 20px;"
    {onclick}
  >
    <!-- Header row -->
    <div class="flex items-center justify-between">
      <div class="flex items-center gap-2.5">
        <div
          class="flex items-center justify-center rounded-full"
          style="width: 38px; height: 38px; background: var(--brand); box-shadow: 0 4px 14px var(--glow);"
        >
          <Radar size={17} strokeWidth={2} color="var(--text-on-accent)" />
        </div>
        <div>
          <p style="font-size: 13px; font-weight: 500; color: var(--text-primary);">{title}</p>
          <p style="font-size: 11px; color: var(--text-secondary); margin-top: 1px;">
            {rangeLabel}
          </p>
        </div>
      </div>
      <ChevronRight size={16} strokeWidth={1.5} color="var(--text-secondary)" />
    </div>

    <!-- Stats grid -->
    <div
      class="mt-4 grid grid-cols-3 gap-px overflow-hidden rounded-[14px]"
      style="background: var(--border-theme);"
    >
      {#each stats as s (s.label)}
        <div class="flex flex-col items-center py-2.5" style="background: var(--surface-bg);">
          {#if loading}
            <div
              style="width: 28px; height: 19px; border-radius: 4px; background: var(--border-theme);"
            ></div>
          {:else}
            <span style="font-size: 19px; font-weight: 600; line-height: 1; color: {s.color};"
              >{s.value}</span
            >
          {/if}
          <span style="font-size: 10px; color: var(--text-secondary); margin-top: 4px;"
            >{s.label}</span
          >
        </div>
      {/each}
    </div>
  </button>
</div>
