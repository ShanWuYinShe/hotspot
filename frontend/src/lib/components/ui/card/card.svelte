<script lang="ts">
  import type { Snippet } from 'svelte';
  import type { HTMLAttributes } from 'svelte/elements';
  import { cn } from '$lib/utils';
  import { cardVariants, type CardVariant, type CardGlow } from './index';

  interface Props extends HTMLAttributes<HTMLDivElement> {
    children?: Snippet;
    class?: string;
    variant?: CardVariant;
    glow?: CardGlow;
    interactive?: boolean;
  }

  let {
    class: className,
    children,
    variant = 'default',
    glow = 'none',
    interactive = false,
    ...restProps
  }: Props = $props();
</script>

<div class={cn(cardVariants({ variant, glow, interactive }), className)} {...restProps}>
  {#if children}
    {@render children()}
  {/if}
</div>
