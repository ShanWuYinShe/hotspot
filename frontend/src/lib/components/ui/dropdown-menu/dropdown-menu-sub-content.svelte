<script lang="ts">
  import { DropdownMenu as DropdownMenuPrimitive, type WithoutChildrenOrChild } from 'bits-ui';
  import { cn } from '$lib/utils';
  import type { Snippet } from 'svelte';

  interface Props extends WithoutChildrenOrChild<DropdownMenuPrimitive.SubContentProps> {
    children?: Snippet;
    class?: string;
  }

  let { class: className, children, ...restProps }: Props = $props();
</script>

<DropdownMenuPrimitive.SubContent
  class={cn(
    'z-[var(--z-dropdown)] min-w-32 overflow-hidden p-1',
    'data-[state=open]:animate-in data-[state=closed]:animate-out data-[state=closed]:fade-out-0 data-[state=open]:fade-in-0 data-[state=closed]:zoom-out-95 data-[state=open]:zoom-in-95 data-[side=bottom]:slide-in-from-top-2 data-[side=left]:slide-in-from-right-2 data-[side=right]:slide-in-from-left-2 data-[side=top]:slide-in-from-bottom-2',
    className
  )}
  style="background: var(--surface-bg); box-shadow: var(--shadow-lg); border: 1px solid var(--border-theme); border-radius: var(--radius-md); color: var(--text-primary);"
  {...restProps}
>
  {#if children}
    {@render children()}
  {/if}
</DropdownMenuPrimitive.SubContent>
