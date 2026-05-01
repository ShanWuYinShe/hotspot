<script lang="ts">
	import type { Snippet } from 'svelte';
	import type { HTMLAttributes } from 'svelte/elements';
	import { cn } from '$lib/utils';

	type Variant = 'flat' | 'pressed' | 'convex';

	interface Props extends HTMLAttributes<HTMLDivElement> {
		variant?: Variant;
		children?: Snippet;
		class?: string;
	}

	let {
		variant = 'flat',
		class: className,
		children,
		...rest
	}: Props = $props();

	const variantClass = $derived(
		variant === 'flat'
			? 'neu-flat'
			: variant === 'pressed'
				? 'neu-pressed'
				: 'neu-convex'
	);
</script>

<div
	class={cn('rounded-[24px] p-4 transition-all duration-200', variantClass, className)}
	{...rest}
>
	{#if children}
		{@render children()}
	{/if}
</div>
