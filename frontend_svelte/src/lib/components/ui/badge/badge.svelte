<script lang="ts">
	import type { Snippet } from 'svelte';
	import type { HTMLAttributes } from 'svelte/elements';
	import { cn } from '$lib/utils';
	import { badgeVariants, type Variant } from './index';

	interface Props extends HTMLAttributes<HTMLElement> {
		variant?: Variant;
		children?: Snippet;
		class?: string;
	}

	let { variant = 'default', class: className, children, ...restProps }: Props = $props();
</script>

<span class={cn(badgeVariants({ variant }), className)} {...restProps}>
	{#if children}
		{@render children()}
	{/if}
</span>
