<script lang="ts">
  import { UserPlus } from 'lucide-svelte';

  const SUGGESTED = [
    { name: 'Nori', handle: '@nori.k', color: '#f87171', mutual: 8 },
    { name: 'Mei', handle: '@meimei', color: '#60a5fa', mutual: 5 },
    { name: 'Kai', handle: '@kai.dev', color: '#34d399', mutual: 3 }
  ];

  const followed = $state<Set<string>>(new Set());
</script>

<div class="neu-surface-flat rounded-[18px] p-4">
  <div class="flex items-center justify-between mb-3">
    <span
      style="font-size: 10px; color: var(--neu-text-muted); text-transform: uppercase; letter-spacing: 0.08em;"
    >
      你可能认识
    </span>
    <UserPlus size={14} color="var(--neu-text-muted)" strokeWidth={1.75} />
  </div>

  <div class="flex flex-col gap-2.5">
    {#each SUGGESTED as s}
      <div class="flex items-center gap-2.5">
        <div
          class="rounded-full flex items-center justify-center flex-shrink-0"
          style="width: 32px; height: 32px; background: {s.color}; font-size: 12px; color: white; font-weight: 600;"
        >
          {s.name[0]}
        </div>
        <div class="flex-1 min-w-0">
          <p style="font-size: 12px; color: var(--neu-text); line-height: 1.2;">{s.name}</p>
          <p style="font-size: 10px; color: var(--neu-text-muted); margin-top: 1px;">
            {s.mutual} 位共同好友
          </p>
        </div>
        <button
          class="px-2.5 py-1 rounded-full"
          style="background: {followed.has(s.name)
            ? 'var(--neu-accent-primary)'
            : 'var(--neu-accent-soft)'};
            border: 1px solid {followed.has(s.name)
            ? 'var(--neu-accent-primary)'
            : 'var(--neu-accent-soft-border)'};
            color: {followed.has(s.name) ? 'white' : 'var(--neu-accent-primary)'};
            font-size: 10px; font-weight: 500; cursor: pointer; outline: none;"
          onclick={() => {
            const next = new Set(followed);
            if (next.has(s.name)) next.delete(s.name);
            else next.add(s.name);
            followed.clear();
            for (const n of next) followed.add(n);
          }}
        >
          {followed.has(s.name) ? '已关注' : '关注'}
        </button>
      </div>
    {/each}
  </div>
</div>
