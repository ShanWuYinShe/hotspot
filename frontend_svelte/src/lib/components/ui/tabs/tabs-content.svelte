<script lang="ts">
	import { Tabs as TabsPrimitive } from 'bits-ui';
	import { cn } from '$lib/utils';
	import type { Snippet } from 'svelte';

	interface Props {
		value: string;
		children?: Snippet;
		class?: string;
	}

	let { class: className, value, children, ...restProps }: Props = $props();
</script>

<TabsPrimitive.Content
	class={cn(
		'mt-2 ring-offset-background focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2',
		className
	)}
	{value}
	{...restProps}
>
	{#if children}
		{@render children()}
	{/if}
</TabsPrimitive.Content>
