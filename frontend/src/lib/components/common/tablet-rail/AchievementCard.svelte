<script lang="ts">
  import { Trophy } from 'lucide-svelte';

  const progress = 0.72;
</script>

<div class="neu-surface-flat rounded-[18px] p-4">
  <div class="flex items-center justify-between mb-2.5">
    <span
      style="font-size: 10px; color: var(--theme-text-muted); text-transform: uppercase; letter-spacing: 0.08em;"
    >
      下一枚徽章
    </span>
    <Trophy size={14} color="var(--theme-accent-primary)" strokeWidth={1.75} />
  </div>

  <div class="flex items-center gap-2.5 mb-3">
    <div
      class="flex items-center justify-center rounded-[12px] flex-shrink-0"
      style="width: 38px; height: 38px;
        background: var(--theme-accent-soft);
        border: 1px solid var(--theme-accent-soft-border);
        font-size: 20px;"
    >
      🌟
    </div>
    <div class="flex-1 min-w-0">
      <p style="font-size: 13px; color: var(--theme-text); line-height: 1.3;">城市漫游者</p>
      <p style="font-size: 11px; color: var(--theme-text-muted); margin-top: 1px;">
        已探索 27 / 38 个地点
      </p>
    </div>
  </div>

  <div
    style="height: 5px; border-radius: 999px; background: var(--theme-border); overflow: hidden;"
  >
    <div
      style="width: {progress * 100}%; height: 100%;
        background: var(--theme-accent-primary);
        border-radius: 999px;"
    ></div>
  </div>

  <p style="font-size: 10px; color: var(--theme-text-muted); margin-top: 6px;">
    再去 11 个新地方就解锁
  </p>
</div>
