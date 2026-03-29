<script lang="ts">
	import { Tabs as TabsPrimitive, type WithoutChildrenOrChild } from 'bits-ui';
	import { cn } from '$lib/utils';
	import type { Snippet } from 'svelte';

	interface Props extends WithoutChildrenOrChild<TabsPrimitive.ListProps> {
		children?: Snippet;
		class?: string;
	}

	let { class: className, children, ...restProps }: Props = $props();
</script>

<TabsPrimitive.List
	class={cn(
		'inline-flex min-h-10 items-center justify-center gap-1 rounded-[1.1rem] bg-muted/55 p-1 text-muted-foreground shadow-sm shadow-black/5 dark:shadow-black/15',
		className
	)}
	{...restProps}
>
	{#if children}
		{@render children()}
	{/if}
</TabsPrimitive.List>
