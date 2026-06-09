<script lang="ts">
  import type { Snippet } from 'svelte';
  import type { HTMLAttributes } from 'svelte/elements';
  import { cn } from '$lib/utils';
  import { interactive } from '$lib/actions/interactive';

  type Variant = 'flat' | 'pressed' | 'convex';

  interface Props extends HTMLAttributes<HTMLDivElement> {
    variant?: Variant;
    children?: Snippet;
    class?: string;
  }

  let { variant = 'flat', class: className, children, ...rest }: Props = $props();

  const variantClass = $derived(
    variant === 'flat' ? 'neu-flat' : variant === 'pressed' ? 'neu-pressed' : 'neu-convex'
  );
</script>

<div
  use:interactive={{ type: 'card' }}
  class={cn('rounded-[32px] p-6 transition-all duration-300', variantClass, className)}
  {...rest}
>
  {#if children}
    {@render children()}
  {/if}
</div>
