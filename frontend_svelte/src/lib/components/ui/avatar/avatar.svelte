<script lang="ts">
  import { Avatar as AvatarPrimitive, type WithoutChildrenOrChild } from 'bits-ui';
  import { cn } from '$lib/utils';
  import type { Snippet } from 'svelte';

  interface Props extends WithoutChildrenOrChild<AvatarPrimitive.RootProps> {
    children?: Snippet;
    class?: string;
    glow?: boolean;
  }

  let { class: className, children, glow = false, ...restProps }: Props = $props();
</script>

<AvatarPrimitive.Root
  class={cn(
    'relative flex h-10 w-10 shrink-0 overflow-hidden rounded-full',
    'transition-all duration-200',
    className
  )}
  style="border: 2px solid var(--neu-border); box-shadow: var(--neu-shadow-md);"
  {...restProps}
>
  {#if children}
    {@render children()}
  {/if}
</AvatarPrimitive.Root>
