<script lang="ts">
  import { Tabs as TabsPrimitive, type WithoutChildrenOrChild } from 'bits-ui';
  import { cn } from '$lib/utils';
  import type { Snippet } from 'svelte';

  interface Props extends WithoutChildrenOrChild<TabsPrimitive.TriggerProps> {
    children?: Snippet;
    class?: string;
  }

  let { class: className, value, disabled, children, ...restProps }: Props = $props();
</script>

<TabsPrimitive.Trigger
  class={cn(
    'inline-flex items-center justify-center whitespace-nowrap',
    'px-3.5 py-1.5 sm:px-4',
    'text-sm font-medium leading-5',
    'transition-all duration-200',
    'cursor-pointer outline-none',
    'disabled:pointer-events-none disabled:opacity-50',
    'data-[state=active]:neu-active-text',
    className
  )}
  style="border-radius: 14px; color: var(--neu-text-muted);"
  data-state-active-style="background: var(--neu-accent-primary); box-shadow: 0 2px 8px var(--neu-glow);"
  {value}
  {disabled}
  {...restProps}
>
  {#if children}
    {@render children()}
  {/if}
</TabsPrimitive.Trigger>
