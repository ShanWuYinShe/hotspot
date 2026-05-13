<script lang="ts">
  import { circularFeedback, rippleFeedback } from '$lib/attachments';
  import { cn } from '$lib/utils';
  import type { Snippet } from 'svelte';
  import type { HTMLButtonAttributes } from 'svelte/elements';

  interface Props extends HTMLButtonAttributes {
    variant?: 'default' | 'ghost' | 'icon' | 'primary' | 'secondary';
    size?: 'default' | 'sm' | 'lg' | 'icon';
    circular?: boolean;
    children?: Snippet;
  }

  let {
    variant = 'default',
    size = 'default',
    circular = false,
    class: className,
    children,
    ...restProps
  }: Props = $props();

  const shouldUseCircular = $derived(circular || variant === 'icon');
  const feedbackAction = $derived(shouldUseCircular ? circularFeedback : rippleFeedback);
</script>

<button
  class={cn(
    'relative inline-flex items-center justify-center gap-2',
    'whitespace-nowrap font-semibold',
    'overflow-hidden rounded-[28px]',
    'transition-all duration-180 ease-in-out',
    'focus-visible:outline-none focus-visible:ring-[2px] focus-visible:ring-[var(--glow)] focus-visible:ring-offset-2 focus-visible:ring-offset-[var(--surface-bg)]',
    'disabled:pointer-events-none disabled:opacity-50',
    'cursor-pointer',
    className
  )}
  style="background: {variant === 'ghost'
    ? 'transparent'
    : 'var(--surface-bg)'}; color: var(--text-primary); border: 1px solid {variant === 'ghost'
    ? 'transparent'
    : 'var(--border-theme)'}; box-shadow: {variant === 'ghost'
    ? 'none'
    : variant === 'primary'
      ? '8px 8px 20px var(--shadow-dark), -8px -8px 20px var(--shadow-light), 0 4px 16px var(--glow)'
      : '8px 8px 20px var(--shadow-dark), -8px -8px 20px var(--shadow-light)'}; {variant ===
  'primary'
    ? 'color: var(--text-on-accent);'
    : ''}"
  {@attach feedbackAction}
  {...restProps}
>
  {#if children}
    {@render children()}
  {/if}
</button>
