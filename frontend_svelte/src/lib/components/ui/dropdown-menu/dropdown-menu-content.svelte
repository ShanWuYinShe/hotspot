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
			'z-50 min-w-32 overflow-hidden',
			'rounded-2xl p-2',
			'border-2 border-primary/20',
			'bg-white/95 dark:bg-card/90',
			'backdrop-blur-xl',
			'text-popover-foreground',
			'shadow-xl shadow-black/15',
			'dark:shadow-2xl dark:shadow-black/30',
			'data-[state=open]:animate-bounce-in',
			'data-[state=closed]:animate-out data-[state=closed]:fade-out-0 data-[state=closed]:zoom-out-95',
			className
		)}
		{...restProps}
	>
		{#if children}
			{@render children()}
		{/if}
	</DropdownMenuPrimitive.Content>
</DropdownMenuPrimitive.Portal>
