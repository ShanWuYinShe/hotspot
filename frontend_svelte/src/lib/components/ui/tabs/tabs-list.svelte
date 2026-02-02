<script lang="ts">
	import { Tabs as TabsPrimitive } from 'bits-ui';
	import { cn } from '$lib/utils';
	import type { Snippet } from 'svelte';

	interface Props {
		children?: Snippet;
		class?: string;
	}

	let { class: className, children, ...restProps }: Props = $props();
</script>

<TabsPrimitive.List
	class={cn(
		'inline-flex h-10 items-center justify-center gap-1 rounded-xl bg-muted/60 p-1 text-muted-foreground shadow-sm shadow-black/5',
		className
	)}
	{...restProps}
>
	{#if children}
		{@render children()}
	{/if}
</TabsPrimitive.List>
