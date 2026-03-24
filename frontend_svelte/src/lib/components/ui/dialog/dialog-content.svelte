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
			'fixed left-[50%] top-[50%] z-50',
				'grid w-full max-w-lg translate-x-[-50%] translate-y-[-50%] gap-4',
			'rounded-3xl border-2 border-primary/20',
			'bg-white/95 dark:bg-card/90',
			'backdrop-blur-xl',
			'p-6',
				'select-none shadow-2xl shadow-black/20',
			'dark:shadow-2xl dark:shadow-black/40',
			'ring-4 ring-primary/10',
			'duration-300',
			'data-[state=open]:animate-bounce-in',
			'data-[state=closed]:animate-out data-[state=closed]:fade-out-0 data-[state=closed]:zoom-out-95',
			className
		)}
		{...restProps}
	>
		{#if children}
			{@render children()}
		{/if}
		{#if !hideClose}
			<DialogPrimitive.Close
				class="absolute right-4 top-4 rounded-2xl h-9 w-9 inline-flex items-center justify-center border-2 border-transparent opacity-70 transition-all hover:opacity-100 hover:bg-muted/50 hover:border-primary/20 focus:outline-none focus:ring-4 focus:ring-primary/40 focus:ring-offset-2 disabled:pointer-events-none cursor-pointer"
			>
				<X class="h-4 w-4" />
				<span class="sr-only">Close</span>
			</DialogPrimitive.Close>
		{/if}
	</DialogPrimitive.Content>
</DialogPrimitive.Portal>
