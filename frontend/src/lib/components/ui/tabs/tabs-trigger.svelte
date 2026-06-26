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
    'rounded-[var(--radius-sm)]',
    'disabled:pointer-events-none disabled:opacity-50',
    /* active 态：品牌色药丸（恢复被死属性 data-state-active-style 吞掉的意图）。
       text-on-accent 走在 bg-brand 之上；shadow-brand-btn 给操作一致的浮起。 */
    'data-[state=active]:bg-brand data-[state=active]:text-on-accent',
    'data-[state=active]:shadow-[var(--shadow-brand-btn)]',
    className
  )}
  style="color: var(--text-secondary);"
  {value}
  {disabled}
  {...restProps}
>
  {#if children}
    {@render children()}
  {/if}
</TabsPrimitive.Trigger>
