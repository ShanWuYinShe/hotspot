<script lang="ts">
  import { Sparkles } from 'lucide-svelte';

  const MOODS = [
    { emoji: '🌤', label: '舒服' },
    { emoji: '🌊', label: '平静' },
    { emoji: '🔥', label: '热血' },
    { emoji: '🌙', label: '想独处' },
    { emoji: '🍵', label: '慢一点' }
  ];

  let activeMood = $state<string | null>(null);
</script>

<div class="neu-surface-flat rounded-[18px] p-4">
  <div class="flex items-center justify-between mb-3">
    <span
      style="font-size: 10px; color: var(--neu-text-muted); text-transform: uppercase; letter-spacing: 0.08em;"
    >
      今日心情
    </span>
    <Sparkles size={14} color="var(--neu-accent-primary)" strokeWidth={1.75} />
  </div>

  <p style="font-size: 13px; color: var(--neu-text); margin-bottom: 12px; line-height: 1.5;">
    今天的你是怎样的？
  </p>

  <div class="flex gap-1.5 flex-wrap">
    {#each MOODS as m}
      <button
        class="flex items-center gap-1 px-2.5 py-1.5 rounded-full neu-surface-ghost"
        style="border: 1px solid {activeMood === m.label
          ? 'var(--neu-accent-primary)'
          : 'var(--neu-border)'}; background: {activeMood === m.label
          ? 'var(--neu-accent-soft)'
          : ''}; cursor: pointer; outline: none;"
        onclick={() => (activeMood = activeMood === m.label ? null : m.label)}
      >
        <span style="font-size: 14px;">{m.emoji}</span>
        <span style="font-size: 11px; color: var(--neu-text-muted);">{m.label}</span>
      </button>
    {/each}
  </div>
</div>
