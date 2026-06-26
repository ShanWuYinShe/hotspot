<script lang="ts">
  import type { Snippet } from 'svelte';

  interface Props {
    title: string;
    description?: string;
    actions?: Snippet;
  }

  let { title, description, actions }: Props = $props();
</script>

<!-- Full-bleed bordered header bar — same visual language as the app's destination pages
     (好友 / 关注 / 我的群组 …). Content is collapsed to the wide measure shell.
     Sticky only on desktop (≥xl): mobile/tablet use the sticky AdminSubNav instead, so
     pinning here would overlap it. -->
<div
  class="xl:sticky xl:top-0 xl:z-10 page-pad-x pt-3 pb-3"
  style="background-color: var(--surface-bg); border-bottom: 1px solid var(--border-theme);"
>
  <div class="page-shell page-shell--wide">
    <div class="flex items-center justify-between gap-4">
      <div class="min-w-0">
        <h1 class="text-h1" style="font-weight: 600; color: var(--text-primary)">{title}</h1>
        {#if description}
          <p class="text-meta" style="color: var(--text-secondary); margin-top: 2px">
            {description}
          </p>
        {/if}
      </div>
      {#if actions}
        <div class="flex shrink-0 items-center gap-2 self-start sm:self-center">
          {@render actions()}
        </div>
      {/if}
    </div>
  </div>
</div>
