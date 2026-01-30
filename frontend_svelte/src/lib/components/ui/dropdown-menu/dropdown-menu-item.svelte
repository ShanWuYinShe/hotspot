<script lang="ts">
	import { DropdownMenu as DropdownMenuPrimitive } from 'bits-ui';
	import { cn } from '$lib/utils';
	import type { Snippet } from 'svelte';

	interface Props {
		inset?: boolean;
		children?: Snippet;
		child?: Snippet<[{ props: Record<string, unknown> }]>;
		class?: string;
		disabled?: boolean;
		onclick?: (event: MouseEvent) => void;
	}

	let { class: className, inset, disabled, children, child, ...restProps }: Props = $props();
</script>

<DropdownMenuPrimitive.Item
	class={cn(
		'relative flex cursor-default select-none items-center gap-2 rounded-sm px-2 py-1.5 text-sm outline-none transition-colors focus:bg-accent focus:text-accent-foreground data-disabled:pointer-events-none data-disabled:opacity-50 [&>svg]:size-4 [&>svg]:shrink-0',
		inset && 'pl-8',
		className
	)}
	{disabled}
	{...restProps}
>
	{#if child}
		{@render child({ props: {} })}
	{:else if children}
		{@render children()}
	{/if}
</DropdownMenuPrimitive.Item>
