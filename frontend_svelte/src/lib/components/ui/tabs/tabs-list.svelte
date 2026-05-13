<script lang="ts">
  import { Tabs as TabsPrimitive, type WithoutChildrenOrChild } from 'bits-ui';
  import { cn } from '$lib/utils';
  import type { Snippet } from 'svelte';

  interface Props extends WithoutChildrenOrChild<TabsPrimitive.ListProps> {
    children?: Snippet;
    class?: string;
  }

  let { class: className, children, ...restProps }: Props = $props();
</script>

<TabsPrimitive.List
  class={cn('inline-flex min-h-10 items-center justify-center gap-1 p-1', className)}
  style="border-radius: var(--radius-md); background: var(--surface-bg); box-shadow: var(--shadow-inset); border: 1px solid var(--border-theme);"
  {...restProps}
>
  {#if children}
    {@render children()}
  {/if}
</TabsPrimitive.List>
