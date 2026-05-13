<script lang="ts">
  import type { Snippet } from 'svelte';
  import ChevronLeft from 'lucide-svelte/icons/chevron-left';

  interface Props {
    title: string;
    description?: string;
    actions?: Snippet;
    backHref?: string;
    backLabel?: string;
  }

  let { title, description, actions, backHref, backLabel = '返回' }: Props = $props();
</script>

<div
  class="rounded-[22px] p-4 sm:p-5"
  style="background: var(--surface-bg); box-shadow: var(--shadow-lg); border: 1px solid var(--border-theme);"
>
  <div class="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between sm:gap-4">
    <div class="min-w-0">
      {#if backHref}
        <a
          href={backHref}
          class="mb-3 inline-flex items-center gap-1.5 outline-none"
          style="font-size: 13px; color: var(--text-secondary);"
        >
          <ChevronLeft size={16} strokeWidth={1.5} />
          <span>{backLabel}</span>
        </a>
      {/if}
      <h1 style="font-size: 20px; font-weight: 600; color: var(--text-primary);">{title}</h1>
      {#if description}
        <p
          style="font-size: 13px; color: var(--text-secondary); margin-top: 4px; line-height: 1.6;"
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
