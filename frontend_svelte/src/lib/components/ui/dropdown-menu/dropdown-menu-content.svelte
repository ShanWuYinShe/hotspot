<script lang="ts">
	import { DropdownMenu as DropdownMenuPrimitive, type WithoutChildrenOrChild } from 'bits-ui';
	import { cn } from '$lib/utils';
	import type { Snippet } from 'svelte';

	interface Props extends WithoutChildrenOrChild<DropdownMenuPrimitive.ContentProps> {
		children?: Snippet;
		class?: string;
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
			'p-2',
			'data-[state=open]:animate-bounce-in',
			'data-[state=closed]:animate-out data-[state=closed]:fade-out-0 data-[state=closed]:zoom-out-95',
			className
		)}
		style="background: var(--neu-bg); box-shadow: var(--neu-shadow-lg); border: 1px solid var(--neu-border); border-radius: var(--neu-radius-md); color: var(--neu-text);"
		{...restProps}
	>
		{#if children}
			{@render children()}
		{/if}
	</DropdownMenuPrimitive.Content>
</DropdownMenuPrimitive.Portal>
