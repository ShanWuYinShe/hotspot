<script lang="ts">
  import type { HTMLAttributes } from 'svelte/elements';
  import { cn } from '$lib/utils';

  interface Props extends HTMLAttributes<HTMLDivElement> {
    class?: string;
  }

  let { class: className, ...restProps }: Props = $props();
</script>

<div class={cn('shimmer', className)} style="border-radius: 12px;" {...restProps}></div>

<style>
  :global(.shimmer) {
    background: linear-gradient(
      90deg,
      var(--border-theme) 25%,
      color-mix(in srgb, var(--text-primary) 8%, transparent) 50%,
      var(--border-theme) 75%
    );
    background-size: 200% 100%;
    animation: shimmer 1.5s ease-in-out infinite;
  }
  @keyframes shimmer {
    0% {
      background-position: 200% 0;
    }
    100% {
      background-position: -200% 0;
    }
  }
</style>
