<script lang="ts">
  import { Eye, ChevronRight } from 'lucide-svelte';

  const VISITORS = [
    { name: 'Yuki', color: '#f87171', min: 4 },
    { name: 'Lin', color: '#60a5fa', min: 12 },
    { name: 'Atlas', color: '#a78bfa', min: 23 },
    { name: 'Niko', color: '#34d399', min: 41 },
    { name: 'Sora', color: '#fbbf24', min: 68 }
  ] as const;
</script>

<button class="neu-surface-flat rounded-[18px] p-4 text-left w-full">
  <div class="flex items-center justify-between mb-3">
    <span
      style="font-size: 10px; color: var(--theme-text-muted); text-transform: uppercase; letter-spacing: 0.08em;"
    >
      今天看过你
    </span>
    <Eye size={14} color="var(--theme-text-muted)" strokeWidth={1.75} />
  </div>

  <div class="flex items-center mb-2.5">
    <div class="flex" style="margin-left: 0;">
      {#each VISITORS.slice(0, 4) as v, i}
        <div
          class="rounded-full flex items-center justify-center"
          style="width: 30px; height: 30px;
            margin-left: {i === 0 ? 0 : -8}px;
            border: 2px solid var(--theme-bg);
            z-index: {10 - i};
            background: {v.color};
            font-size: 11px; color: white; font-weight: 600;"
        >
          {v.name[0]}
        </div>
      {/each}
    </div>
    <div class="ml-auto flex items-baseline gap-1">
      <span style="font-size: 22px; color: var(--theme-text); font-weight: 600; line-height: 1;"
        >18</span
      >
      <span style="font-size: 11px; color: var(--theme-text-muted);">位</span>
    </div>
  </div>

  <div class="flex items-center justify-between">
    <p style="font-size: 11px; color: var(--theme-text-muted);">
      {VISITORS[0].name} 刚刚 · 还有 {VISITORS.length - 1} 位
    </p>
    <ChevronRight size={12} color="var(--theme-text-muted)" strokeWidth={1.5} />
  </div>
</button>
