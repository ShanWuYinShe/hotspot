<script lang="ts">
  import { Dialog as DialogPrimitive, type WithoutChildrenOrChild } from 'bits-ui';
  import { cn } from '$lib/utils';
  import type { Snippet } from 'svelte';

  interface Props extends WithoutChildrenOrChild<DialogPrimitive.DescriptionProps> {
    children?: Snippet;
    class?: string;
  }

  let { class: className, children, ...restProps }: Props = $props();
</script>

<DialogPrimitive.Description
  class={cn('text-sm', className)}
  style="color: var(--text-secondary);"
  {...restProps}
>
  {#if children}
    {@render children()}
  {/if}
</DialogPrimitive.Description>
