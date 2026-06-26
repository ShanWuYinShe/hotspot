<script lang="ts">
  import type { Snippet } from 'svelte';
  import type { HTMLButtonAttributes } from 'svelte/elements';
  import { cn } from '$lib/utils';
  import { circularFeedback, rippleFeedback } from '$lib/attachments';
  import { buttonVariants, type Variant, type Size } from './index';

  interface Props extends HTMLButtonAttributes {
    variant?: Variant;
    size?: Size;
    /** 触摸波纹反馈：'circular'(圆形按钮) | 'ripple'(矩形按钮) | 不设=无波纹。
     *  统一吸收原 ui/interactive/ripple-button 的能力，消除两套并行按钮实现。 */
    feedback?: 'circular' | 'ripple';
    children?: Snippet;
    class?: string;
  }

  let {
    variant = 'default',
    size = 'default',
    feedback = undefined,
    class: className,
    children,
    ...restProps
  }: Props = $props();

  const feedbackAction = $derived(
    feedback === 'circular' ? circularFeedback : feedback === 'ripple' ? rippleFeedback : undefined
  );
</script>

<button
  class={cn(buttonVariants({ variant, size }), className)}
  {@attach feedbackAction}
  {...restProps}
>
  {#if children}
    {@render children()}
  {/if}
</button>
