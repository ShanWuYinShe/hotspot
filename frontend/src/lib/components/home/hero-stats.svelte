<script lang="ts">
  import Radar from 'lucide-svelte/icons/radar';
  import ChevronRight from 'lucide-svelte/icons/chevron-right';
  import StatSegments from '$lib/components/common/stat-segments.svelte';

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
    class="surface-hero w-full overflow-hidden rounded-[var(--radius-xl)] text-left transition-transform duration-150 active:scale-[0.99]"
    style="padding: 18px 20px;"
    {onclick}
  >
    <!-- Header row -->
    <div class="flex items-center justify-between">
      <div class="flex items-center gap-2.5">
        <div
          class="flex items-center justify-center rounded-full"
          style="width: 38px; height: 38px; background: var(--brand); box-shadow: var(--shadow-brand-btn);"
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
    <StatSegments layout="grid" {loading} {stats} class="mt-4" />
  </button>
</div>
