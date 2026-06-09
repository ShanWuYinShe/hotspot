<script lang="ts">
  import { ScrollArea as ScrollAreaPrimitive, type WithoutChildrenOrChild } from 'bits-ui';
  import { cn } from '$lib/utils';
  import type { Snippet } from 'svelte';
  import Scrollbar from './scroll-area-scrollbar.svelte';

  interface Props extends WithoutChildrenOrChild<ScrollAreaPrimitive.RootProps> {
    orientation?: 'vertical' | 'horizontal' | 'both';
    children?: Snippet;
    class?: string;
  }

  let { class: className, orientation = 'vertical', children, ...restProps }: Props = $props();
</script>

<ScrollAreaPrimitive.Root class={cn('relative overflow-hidden', className)} {...restProps}>
  <ScrollAreaPrimitive.Viewport class="h-full w-full rounded-[inherit]">
    {#if children}
      {@render children()}
    {/if}
  </ScrollAreaPrimitive.Viewport>
  {#if orientation === 'vertical' || orientation === 'both'}
    <Scrollbar orientation="vertical" />
  {/if}
  {#if orientation === 'horizontal' || orientation === 'both'}
    <Scrollbar orientation="horizontal" />
  {/if}
  <ScrollAreaPrimitive.Corner />
</ScrollAreaPrimitive.Root>
