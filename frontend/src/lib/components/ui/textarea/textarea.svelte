<script lang="ts">
  import type { HTMLTextareaAttributes } from 'svelte/elements';
  import { cn } from '$lib/utils';
  import { useKeyboardFocus } from '$lib/attachments';

  interface Props extends HTMLTextareaAttributes {
    class?: string;
    value?: string;
  }

  let { class: className, value = $bindable(''), ...restProps }: Props = $props();
</script>

<textarea
  {@attach useKeyboardFocus()}
  class={cn(
    'input-base flex min-h-[60px] w-full select-text outline-none disabled:cursor-not-allowed disabled:opacity-50 md:text-sm',
    className
  )}
  style="background: var(--surface-bg); border: 1px solid var(--border-theme); border-radius: var(--radius-lg); color: var(--text-primary);"
  bind:value
  {...restProps}
></textarea>
