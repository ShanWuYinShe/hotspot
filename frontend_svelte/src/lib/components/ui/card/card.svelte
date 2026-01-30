<script lang="ts">
	import type { Snippet } from 'svelte';
	import type { HTMLAttributes } from 'svelte/elements';
	import { cn } from '$lib/utils';

	interface Props extends HTMLAttributes<HTMLDivElement> {
		children?: Snippet;
		class?: string;
	}

	let { class: className, children, ...restProps }: Props = $props();
</script>

<div
	class={cn(
		'rounded-xl border border-border/50 bg-card text-card-foreground shadow-sm shadow-black/5',
		className
	)}
	{...restProps}
>
	{#if children}
		{@render children()}
	{/if}
</div>
