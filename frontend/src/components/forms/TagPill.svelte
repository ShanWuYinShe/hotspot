<script lang="ts">
	import type { Snippet } from 'svelte';
	import { cn } from '@/lib/utils';

	let {
		children,
		active = false,
		onClick,
		size = 'md',
		class: className = ''
	}: {
		children: Snippet;
		active?: boolean;
		onClick?: () => void;
		size?: 'sm' | 'md';
		class?: string;
	} = $props();

	const sizeClasses = {
		sm: 'text-xxs',
		md: 'text-xs'
	} as const;
</script>

{#if onClick}
	<button
		type="button"
		aria-pressed={active}
		class={cn(
			'rounded-full outline-none transition-colors px-3 py-1 cursor-pointer surface-ghost focus-visible:ring-2 focus-visible:ring-accent-default',
			sizeClasses[size],
			active
				? 'bg-accent-soft text-accent-default border border-accent-border font-semibold'
				: 'text-text-secondary',
			className
		)}
		onclick={onClick}
	>
		{@render children()}
	</button>
{:else}
	<span
		class={cn(
			'rounded-full focus-ring transition-colors px-3 py-1',
			sizeClasses[size],
			active
				? 'bg-accent-soft text-accent-default border border-accent-border font-semibold'
				: 'text-text-secondary',
			className
		)}
	>
		{@render children()}
	</span>
{/if}
