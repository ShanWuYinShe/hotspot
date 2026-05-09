<script lang="ts">
  import type { Snippet } from 'svelte';
  import type { HTMLInputAttributes } from 'svelte/elements';
  import { cn } from '$lib/utils';

  interface Props extends Omit<HTMLInputAttributes, 'value'> {
    icon?: Snippet;
    class?: string;
    value?: string;
  }

  let { icon, class: className, value = $bindable(''), ...rest }: Props = $props();
</script>

<div class="relative flex w-full items-center">
  {#if icon}
    <div class="pointer-events-none absolute left-4 text-neu-muted">
      {@render icon()}
    </div>
  {/if}
  <input
    class={cn(
      'w-full neu-input-base rounded-2xl py-3 px-4 text-neu outline-none transition-all duration-200',
      'placeholder:text-neu-muted',
      icon && 'pl-12',
      className
    )}
    bind:value
    {...rest}
  />
</div>
