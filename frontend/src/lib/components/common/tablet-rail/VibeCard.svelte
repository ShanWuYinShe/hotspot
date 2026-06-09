<script lang="ts">
  import { Sun, Cloud, Music } from 'lucide-svelte';
</script>

<div class="neu-surface-flat rounded-[18px] p-4">
  <div class="flex items-center justify-between mb-3">
    <span
      style="font-size: 10px; color: var(--theme-text-muted); text-transform: uppercase; letter-spacing: 0.08em;"
    >
      今日氛围
    </span>
    <Sun size={14} color="var(--theme-accent-primary)" strokeWidth={1.75} />
  </div>

  <div class="flex items-center gap-3 mb-3">
    <div class="flex items-center gap-1.5">
      <Cloud size={16} color="var(--theme-text-muted)" strokeWidth={1.5} />
      <span style="font-size: 13px; color: var(--theme-text);">多云 · 21°</span>
    </div>
    <span style="font-size: 10px; color: var(--theme-text-muted);">适合散步</span>
  </div>

  <div
    class="flex items-center gap-2.5 p-2 rounded-[12px]"
    style="background: var(--theme-border);"
  >
    <div
      class="flex items-center justify-center rounded-[8px] flex-shrink-0"
      style="width: 32px; height: 32px;
        background: var(--theme-accent-primary);
        opacity: 0.9;"
    >
      <Music size={14} color="white" strokeWidth={2} />
    </div>
    <div class="flex-1 min-w-0">
      <p class="truncate" style="font-size: 11px; color: var(--theme-text); line-height: 1.2;">
        漫长的告白
      </p>
      <p style="font-size: 9px; color: var(--theme-text-muted); margin-top: 1px;">
        陈鸿宇 · 单曲循环
      </p>
    </div>
  </div>
</div>
