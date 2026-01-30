<script lang="ts">
	import { DropdownMenu as DropdownMenuPrimitive } from 'bits-ui';
	import { cn } from '$lib/utils';
	import type { Snippet } from 'svelte';

	interface Props {
		sideOffset?: number;
		children?: Snippet;
		class?: string;
		side?: 'top' | 'right' | 'bottom' | 'left';
		align?: 'start' | 'center' | 'end';
	}

	let { class: className, sideOffset = 4, side, align, children, ...restProps }: Props = $props();
</script>

<DropdownMenuPrimitive.Portal>
	<DropdownMenuPrimitive.Content
		{sideOffset}
		{side}
		{align}
		class={cn(
		'z-50 min-w-32 overflow-hidden rounded-md border border-border/60 bg-popover/95 p-1 text-popover-foreground shadow-xl shadow-black/10 backdrop-blur-sm',
			'data-[state=open]:animate-in data-[state=closed]:animate-out data-[state=closed]:fade-out-0 data-[state=open]:fade-in-0 data-[state=closed]:zoom-out-95 data-[state=open]:zoom-in-95 data-[side=bottom]:slide-in-from-top-2 data-[side=left]:slide-in-from-right-2 data-[side=right]:slide-in-from-left-2 data-[side=top]:slide-in-from-bottom-2',
			className
		)}
		{...restProps}
	>
		{#if children}
			{@render children()}
		{/if}
	</DropdownMenuPrimitive.Content>
</DropdownMenuPrimitive.Portal>
