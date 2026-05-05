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
			'p-6',
			'select-none',
			'duration-200',
			'data-[state=open]:animate-in data-[state=open]:zoom-in-95 data-[state=open]:fade-in-0',
			'data-[state=closed]:animate-out data-[state=closed]:fade-out-0 data-[state=closed]:zoom-out-95',
			className
		)}
		style="background: var(--neu-bg); box-shadow: 8px 8px 20px var(--neu-shadow-dark), -8px -8px 18px var(--neu-shadow-light); border: 1px solid var(--neu-border); border-radius: 24px; color: var(--neu-text);"
		{...restProps}
	>
		{#if children}
			{@render children()}
		{/if}
		{#if !hideClose}
			<DialogPrimitive.Close
				class="absolute right-4 top-4 flex h-9 w-9 items-center justify-center rounded-sm outline-none opacity-70 transition-opacity hover:opacity-100 cursor-pointer"
				style="background: var(--neu-bg); box-shadow: 5px 5px 10px var(--neu-shadow-dark), -5px -5px 10px var(--neu-shadow-light); border: 1px solid var(--neu-border);"
			>
				<X size={16} color="var(--neu-text-muted)" />
				<span class="sr-only">Close</span>
			</DialogPrimitive.Close>
		{/if}
	</DialogPrimitive.Content>
</DialogPrimitive.Portal>
