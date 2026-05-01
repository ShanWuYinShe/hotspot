<script lang="ts">
	import type { Snippet } from 'svelte';
	import type { HTMLButtonAttributes } from 'svelte/elements';
	import { cn } from '$lib/utils';

	type Variant = 'flat' | 'pressed' | 'icon' | 'glow';
	type Size = 'sm' | 'md' | 'lg' | 'icon';

	interface Props extends HTMLButtonAttributes {
		variant?: Variant;
		size?: Size;
		active?: boolean;
		children?: Snippet;
		class?: string;
	}

	let {
		variant = 'flat',
		size = 'md',
		active = false,
		class: className,
		children,
		...rest
	}: Props = $props();

	const variantClass = $derived(
		active
			? 'neu-pressed text-neu-accent'
			: variant === 'flat'
				? 'neu-flat'
				: variant === 'pressed'
					? 'neu-pressed'
					: variant === 'glow'
						? 'neu-button-glow'
						: 'neu-flat text-neu-accent'
	);

	const sizeClass = $derived(
		size === 'sm'
			? 'px-4 py-2 text-sm'
			: size === 'md'
				? 'px-8 py-3.5'
				: size === 'lg'
					? 'px-10 py-4 text-lg'
					: 'p-4 rounded-full aspect-square'
	);
</script>

<button
	class={cn(
		'relative flex items-center justify-center rounded-[22px] transition-all duration-200 outline-none select-none',
		'text-neu font-medium',
		sizeClass,
		variantClass,
		className
	)}
	{...rest}
>
	{#if children}
		{@render children()}
	{/if}
</button>
