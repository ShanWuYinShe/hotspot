<script lang="ts">
	import { cn } from '$lib/utils';
	import type { Snippet } from 'svelte';
	import type { HTMLAttributes } from 'svelte/elements';

	interface Props extends HTMLAttributes<HTMLDivElement> {
		inset?: boolean;
		children?: Snippet;
		class?: string;
	}

	let { class: className, inset, children, ...restProps }: Props = $props();
</script>

<div
	class={cn('px-2 py-1.5 text-sm font-semibold', inset && 'pl-8', className)}
	{...restProps}
>
	{#if children}
		{@render children()}
	{/if}
</div>
