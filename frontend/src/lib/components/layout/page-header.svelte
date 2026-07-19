<script lang="ts">
  import type { Snippet } from 'svelte';
  import ChevronLeft from 'lucide-svelte/icons/chevron-left';

  interface Props {
    title: string;
    description?: string;
    actions?: Snippet;
    backHref?: string;
    backLabel?: string;
    /** 标题字号档:default=20px(列表/钻取页)、hero=24px(门面页 home/discover/events) */
    size?: 'default' | 'hero';
  }

  let {
    title,
    description,
    actions,
    backHref,
    backLabel = '返回',
    size = 'default'
  }: Props = $props();
</script>

<div
  class="rounded-[var(--radius-lg)] p-4 sm:p-5"
  style="background: var(--surface-bg); border: 1px solid var(--border-theme);"
>
  <div class="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between sm:gap-4">
    <div class="min-w-0">
      {#if backHref}
        <a
          href={backHref}
          class="text-body mb-3 inline-flex items-center gap-1.5 outline-none"
          style="color: var(--text-secondary)"
        >
          <ChevronLeft size={16} strokeWidth={1.5} />
          <span>{backLabel}</span>
        </a>
      {/if}
      <h1
        style="font-size: {size === 'hero'
          ? '24px'
          : '20px'}; font-weight: 600; color: var(--text-primary);"
      >
        {title}
      </h1>
      {#if description}
        <p
          class="text-body"
          style="color: var(--text-secondary); margin-top: 4px; line-height: 1.6"
        >
          {description}
        </p>
      {/if}
    </div>
    {#if actions}
      <div class="flex shrink-0 items-center gap-2 self-start sm:self-auto">
        {@render actions()}
      </div>
    {/if}
  </div>
</div>
