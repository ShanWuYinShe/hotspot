<script lang="ts">
  import { Flame } from 'lucide-svelte';

  const STREAK_DAYS = [
    { label: '一', done: true },
    { label: '二', done: true },
    { label: '三', done: true },
    { label: '四', done: true },
    { label: '五', done: true },
    { label: '六', done: true },
    { label: '日', done: false, today: true }
  ];
</script>

<div class="neu-surface-flat rounded-[18px] p-4">
  <div class="flex items-center justify-between mb-3">
    <span
      style="font-size: 10px; color: var(--theme-text-muted); text-transform: uppercase; letter-spacing: 0.08em;"
    >
      连续签到
    </span>
    <Flame size={14} color="var(--theme-accent-primary)" strokeWidth={1.75} />
  </div>

  <div class="flex items-baseline gap-1.5 mb-3.5">
    <span style="font-size: 32px; color: var(--theme-text); font-weight: 600; line-height: 1;"
      >27</span
    >
    <span style="font-size: 11px; color: var(--theme-text-muted);">天</span>
  </div>

  <div class="flex gap-1 mb-2.5">
    {#each STREAK_DAYS as d}
      <div class="flex-1 flex flex-col items-center gap-1">
        <span style="font-size: 9px; color: var(--theme-text-muted); opacity: {d.today ? 1 : 0.7};"
          >{d.label}</span
        >
        <div
          style="width: 100%; height: 22px; border-radius: 5px;
            background: {d.done
            ? 'var(--theme-accent-primary)'
            : d.today
              ? 'var(--theme-accent-soft)'
              : 'var(--theme-border)'};
            opacity: {d.done ? 0.88 : 1};
            border: {d.today ? '1.5px dashed var(--theme-accent-soft-border)' : 'none'};"
        ></div>
      </div>
    {/each}
  </div>

  <p style="font-size: 11px; color: var(--theme-text-muted); line-height: 1.5;">
    明天再来就满 28 天啦 🌱
  </p>
</div>
