<script lang="ts">
  interface StatSegment {
    value: string | number;
    label: string;
    color?: string;
  }

  interface Props {
    stats: StatSegment[];
    layout?: 'inline' | 'grid';
    loading?: boolean;
    class?: string;
  }

  let { stats, layout = 'inline', loading = false, class: className = '' }: Props = $props();
</script>

<div
  class="items-stretch gap-px overflow-hidden rounded-[var(--radius-md)] {layout === 'grid'
    ? 'grid'
    : 'inline-flex'} {className}"
  style="background: var(--border-theme);{layout === 'grid'
    ? ` grid-template-columns: repeat(${stats.length}, minmax(0, 1fr));`
    : ''}"
>
  {#each stats as s (s.label)}
    <div
      class="flex flex-col items-center justify-center {layout === 'grid' ? 'py-2.5' : 'px-4 py-2'}"
      style="background: var(--surface-bg);{layout === 'inline' ? ' min-width: 72px;' : ''}"
    >
      {#if loading}
        <div
          style="width: 28px; height: 19px; border-radius: var(--radius-xs); background: var(--border-theme);"
        ></div>
      {:else}
        <span
          class="text-h2"
          style="font-weight: 600; line-height: 1; color: {s.color ?? 'var(--text-primary)'}"
          >{s.value}</span
        >
      {/if}
      <span class="text-xs" style="color: var(--text-secondary); margin-top: 4px">{s.label}</span>
    </div>
  {/each}
</div>
