<script lang="ts">
	import { cn } from '$lib/utils';
	import type { Snippet } from 'svelte';
	import type { HTMLAttributes } from 'svelte/elements';

	interface Props extends HTMLAttributes<HTMLSpanElement> {
		children?: Snippet;
		class?: string;
	}

	let { class: className, children, ...restProps }: Props = $props();
</script>

<span class={cn('ml-auto text-xs tracking-widest opacity-60', className)} {...restProps}>
	{#if children}
		{@render children()}
	{/if}
</span>
