<script lang="ts">
  import { browser } from '$app/environment';
  import { Dialog as DialogPrimitive, type WithoutChildrenOrChild } from 'bits-ui';
  import { cn } from '$lib/utils';
  import type { Snippet } from 'svelte';
  import X from 'lucide-svelte/icons/x';
  import Overlay from './dialog-overlay.svelte';

  interface Props extends WithoutChildrenOrChild<DialogPrimitive.ContentProps> {
    children?: Snippet;
    class?: string;
    hideClose?: boolean;
  }

  let {
    class: className,
    children,
    hideClose = false,
    ref = $bindable(null),
    onCloseAutoFocus,
    ...restProps
  }: Props = $props();

  function clearDialogRelatedSelection(content: HTMLElement | null) {
    if (!browser || !content) return;
    const selection = window.getSelection();
    if (!selection || selection.rangeCount === 0) return;

    const range = selection.getRangeAt(0);
    const commonAncestor = range.commonAncestorContainer;
    let shouldClear = content.contains(commonAncestor);

    if (!shouldClear) {
      const rects = Array.from(range.getClientRects());
      if (rects.length === 0) return;
      const contentRect = content.getBoundingClientRect();
      shouldClear = rects.some((rect) => {
        const separated =
          rect.right < contentRect.left ||
          rect.left > contentRect.right ||
          rect.bottom < contentRect.top ||
          rect.top > contentRect.bottom;
        return !separated;
      });
    }

    if (shouldClear) {
      selection.removeAllRanges();
    }
  }

  function handleCloseAutoFocus(event: Event) {
    clearDialogRelatedSelection(ref);
    onCloseAutoFocus?.(event);
  }
</script>

<DialogPrimitive.Portal>
  <Overlay />
  <DialogPrimitive.Content
    bind:ref
    onCloseAutoFocus={handleCloseAutoFocus}
    class={cn(
      'fixed z-[var(--z-modal)] grid w-full gap-4 p-6 select-none duration-200',
      'data-[state=open]:animate-in data-[state=open]:fade-in-0',
      'data-[state=closed]:animate-out data-[state=closed]:fade-out-0',
      'dialog-responsive-position',
      className
    )}
    style="background: var(--surface-bg); box-shadow: var(--shadow-lg); border: 1px solid var(--border-theme); color: var(--text-primary);"
    {...restProps}
  >
    {#if children}
      {@render children()}
    {/if}
    {#if !hideClose}
      <DialogPrimitive.Close
        class="absolute right-4 top-4 flex h-9 w-9 items-center justify-center rounded-[var(--radius-xs)] outline-none opacity-70 transition-opacity hover:opacity-100 cursor-pointer"
        style="background: var(--surface-bg); box-shadow: var(--shadow-md); border: 1px solid var(--border-theme);"
      >
        <X size={16} color="var(--text-secondary)" />
        <span class="sr-only">Close</span>
      </DialogPrimitive.Close>
    {/if}
  </DialogPrimitive.Content>
</DialogPrimitive.Portal>

<style>
  /* Mobile: slide up from bottom. Desktop: centered zoom */
  :global(.dialog-responsive-position) {
    left: 50%;
    top: 50%;
    transform: translate(-50%, -50%);
    max-width: 32rem;
    border-radius: var(--theme-radius-xl, 24px);
  }

  @media (max-width: 639px) {
    :global(.dialog-responsive-position) {
      left: 0;
      right: 0;
      top: auto;
      bottom: 0;
      transform: none;
      max-width: 100%;
      border-bottom-left-radius: 0;
      border-bottom-right-radius: 0;
    }

    :global(.dialog-responsive-position[data-state='open']) {
      animation: dialogSlideUp 0.42s cubic-bezier(0.34, 1.32, 0.46, 1);
    }
    :global(.dialog-responsive-position[data-state='closed']) {
      animation: dialogSlideDown 0.2s cubic-bezier(0.22, 1, 0.36, 1);
    }
  }

  @media (min-width: 640px) {
    :global(.dialog-responsive-position[data-state='open']) {
      animation: dialogZoomIn 0.28s cubic-bezier(0.34, 1.46, 0.5, 1);
    }
    :global(.dialog-responsive-position[data-state='closed']) {
      animation: dialogZoomOut 0.15s cubic-bezier(0.22, 1, 0.36, 1);
    }
  }

  @keyframes dialogSlideUp {
    from {
      transform: translateY(100%);
      opacity: 0.5;
    }
    to {
      transform: translateY(0);
      opacity: 1;
    }
  }
  @keyframes dialogSlideDown {
    from {
      transform: translateY(0);
      opacity: 1;
    }
    to {
      transform: translateY(100%);
      opacity: 0;
    }
  }
  @keyframes dialogZoomIn {
    from {
      transform: translate(-50%, -50%) scale(0.95);
      opacity: 0;
    }
    to {
      transform: translate(-50%, -50%) scale(1);
      opacity: 1;
    }
  }
  @keyframes dialogZoomOut {
    from {
      transform: translate(-50%, -50%) scale(1);
      opacity: 1;
    }
    to {
      transform: translate(-50%, -50%) scale(0.95);
      opacity: 0;
    }
  }
</style>
