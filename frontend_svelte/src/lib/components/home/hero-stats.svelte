<script lang="ts">
  import Radar from 'lucide-svelte/icons/radar';
  import MapPin from 'lucide-svelte/icons/map-pin';

  interface Props {
    people?: number;
    activity?: number;
    online?: number;
    radius?: string;
    location?: string;
    loading?: boolean;
  }

  let {
    people = 0,
    activity = 0,
    online = 0,
    radius = '5km',
    location = '',
    loading = false
  }: Props = $props();
</script>

<div class="ambient-glow">
  <div class="surface-hero rounded-[24px]" style="padding: 18px 20px;">
    <!-- Header row -->
    <div class="flex items-center gap-3 mb-4">
      <div
        class="flex items-center justify-center rounded-full"
        style="width: 38px; height: 38px; background: var(--brand); box-shadow: 0 4px 14px var(--glow);"
      >
        <Radar size={18} strokeWidth={2} color="var(--text-on-accent)" />
      </div>
      <div class="flex-1">
        <div class="flex items-center gap-1.5">
          <div
            style="width: 6px; height: 6px; border-radius: 999px; background: #22c55e; box-shadow: 0 0 0 3px rgba(34,197,94,0.18);"
          ></div>
          <span style="font-size: 12px; color: var(--text-secondary);">{radius} 范围</span>
        </div>
      </div>
      {#if location}
        <div class="flex items-center gap-1" style="font-size: 12px; color: var(--brand);">
          <MapPin size={12} strokeWidth={2} />
          <span>{location}</span>
        </div>
      {/if}
    </div>

    <!-- Stats grid -->
    {#if loading}
      <div
        class="grid grid-cols-3 gap-1 rounded-[14px] overflow-hidden"
        style="background: var(--border-theme);"
      >
        {#each [0, 1, 2] as _}
          <div class="flex flex-col items-center py-3" style="background: var(--surface-bg);">
            <div
              style="width: 32px; height: 18px; border-radius: 4px; background: var(--border-theme);"
            ></div>
            <div
              style="width: 24px; height: 10px; border-radius: 3px; background: var(--border-theme); margin-top: 4px;"
            ></div>
          </div>
        {/each}
      </div>
    {:else}
      <div
        class="grid grid-cols-3 gap-1 rounded-[14px] overflow-hidden"
        style="background: var(--border-theme);"
      >
        <div class="flex flex-col items-center py-3" style="background: var(--surface-bg);">
          <span style="font-size: 22px; font-weight: 700; color: var(--brand);">{people}</span>
          <span
            style="font-size: 9px; letter-spacing: 0.08em; text-transform: uppercase; color: var(--text-secondary); margin-top: 2px;"
            >附近的人</span
          >
        </div>
        <div class="flex flex-col items-center py-3" style="background: var(--surface-bg);">
          <span style="font-size: 22px; font-weight: 700; color: #f59e0b;">{activity}</span>
          <span
            style="font-size: 9px; letter-spacing: 0.08em; text-transform: uppercase; color: var(--text-secondary); margin-top: 2px;"
            >活动</span
          >
        </div>
        <div class="flex flex-col items-center py-3" style="background: var(--surface-bg);">
          <span style="font-size: 22px; font-weight: 700; color: #22c55e;">{online}</span>
          <span
            style="font-size: 9px; letter-spacing: 0.08em; text-transform: uppercase; color: var(--text-secondary); margin-top: 2px;"
            >在线</span
          >
        </div>
      </div>
    {/if}
  </div>
</div>
