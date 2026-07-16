<script lang="ts">
  import { DropdownMenu as DropdownMenuPrimitive, type WithoutChildrenOrChild } from 'bits-ui';
  import { cn } from '$lib/utils';
  import type { Snippet } from 'svelte';
  import ChevronRight from 'lucide-svelte/icons/chevron-right';

  interface Props extends WithoutChildrenOrChild<DropdownMenuPrimitive.SubTriggerProps> {
    inset?: boolean;
    children?: Snippet;
    class?: string;
  }

  let { class: className, inset, children, ...restProps }: Props = $props();
</script>

<DropdownMenuPrimitive.SubTrigger
  class={cn(
    'flex cursor-default select-none items-center gap-2 rounded-[var(--radius-xs)] px-2 py-1.5 text-sm outline-none focus:bg-accent data-[state=open]:bg-accent [&>svg]:size-4 [&>svg]:shrink-0',
    inset && 'pl-8',
    className
  )}
  {...restProps}
>
  {#if children}
    {@render children()}
  {/if}
  <ChevronRight class="ml-auto" />
</DropdownMenuPrimitive.SubTrigger>
