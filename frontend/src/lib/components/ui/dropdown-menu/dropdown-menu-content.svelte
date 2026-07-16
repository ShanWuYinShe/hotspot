<script lang="ts">
  import { DropdownMenu as DropdownMenuPrimitive, type WithoutChildrenOrChild } from 'bits-ui';
  import { cn } from '$lib/utils';
  import type { Snippet } from 'svelte';

  interface Props extends WithoutChildrenOrChild<DropdownMenuPrimitive.ContentProps> {
    children?: Snippet;
    class?: string;
  }

  let { class: className, sideOffset = 4, side, align, children, ...restProps }: Props = $props();
</script>

<DropdownMenuPrimitive.Portal>
  <DropdownMenuPrimitive.Content
    {sideOffset}
    {side}
    {align}
    class={cn(
      'z-[var(--z-dropdown)] min-w-32 overflow-hidden',
      'p-2',
      'data-[state=open]:animate-bounce-in',
      'data-[state=closed]:animate-out data-[state=closed]:fade-out-0 data-[state=closed]:zoom-out-95',
      className
    )}
    style="background: var(--surface-bg); box-shadow: var(--shadow-lg); border: 1px solid var(--border-theme); border-radius: var(--radius-md); color: var(--text-primary);"
    {...restProps}
  >
    {#if children}
      {@render children()}
    {/if}
  </DropdownMenuPrimitive.Content>
</DropdownMenuPrimitive.Portal>
