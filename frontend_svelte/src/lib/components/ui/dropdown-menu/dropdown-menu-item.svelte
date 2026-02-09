<script lang="ts">
	import { DropdownMenu as DropdownMenuPrimitive, type WithoutChildrenOrChild } from 'bits-ui';
	import { cn } from '$lib/utils';
	import type { Snippet } from 'svelte';

	interface Props extends WithoutChildrenOrChild<DropdownMenuPrimitive.ItemProps> {
		inset?: boolean;
		children?: Snippet;
		child?: Snippet<[{ props: Record<string, unknown> }]>;
		class?: string;
	}

	let { class: className, inset, disabled, children, child, ...restProps }: Props = $props();
</script>

<DropdownMenuPrimitive.Item
	class={cn(
		'relative flex cursor-pointer select-none items-center gap-2',
		'px-3 py-2.5 rounded-xl',
		'text-sm font-medium',
		'outline-none',
		'transition-all duration-200',
		'hover:bg-primary/10 hover:text-primary',
		'hover:scale-105 hover:-translate-x-1',
		'focus:bg-primary/10 focus:text-primary',
		'active:scale-95',
		'data-disabled:pointer-events-none data-disabled:opacity-50',
		'[&>svg]:size-4 [&>svg]:shrink-0',
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
