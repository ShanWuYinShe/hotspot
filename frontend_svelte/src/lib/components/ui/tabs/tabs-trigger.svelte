<script lang="ts">
	import { Tabs as TabsPrimitive, type WithoutChildrenOrChild } from 'bits-ui';
	import { cn } from '$lib/utils';
	import type { Snippet } from 'svelte';

	interface Props extends WithoutChildrenOrChild<TabsPrimitive.TriggerProps> {
		children?: Snippet;
		class?: string;
	}

	let { class: className, value, disabled, children, ...restProps }: Props = $props();
</script>

<TabsPrimitive.Trigger
	class={cn(
		'inline-flex items-center justify-center whitespace-nowrap',
		'rounded-[0.95rem] px-3.5 py-1.5 sm:px-4',
		'border border-transparent',
		'text-sm font-medium leading-5',
		'transition-colors duration-200',
		'cursor-pointer',
		'hover:bg-primary/10 hover:text-primary',
		'focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary/40',
		'disabled:pointer-events-none disabled:opacity-50',
		'data-[state=active]:bg-primary data-[state=active]:text-primary-foreground',
		'data-[state=active]:shadow-sm',
		className
	)}
	{value}
	{disabled}
	{...restProps}
>
	{#if children}
		{@render children()}
	{/if}
</TabsPrimitive.Trigger>
