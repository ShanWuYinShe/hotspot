<script lang="ts">
	import type { Snippet } from 'svelte';
	import { cn } from '@/lib/utils';

	let {
		children,
		active = false,
		onClick,
		size = 'md',
		className = ''
	}: {
		children: Snippet;
		active?: boolean;
		onClick?: () => void;
		size?: 'sm' | 'md';
		className?: string;
	} = $props();

	const sizeClasses = {
		sm: 'text-[10px]',
		md: 'text-xs'
	} as const;
</script>

{#if onClick}
	<button
		type="button"
		aria-pressed={active}
		class={cn(
			'rounded-full outline-none transition-colors px-3 py-1 cursor-pointer neu-surface-ghost focus-visible:ring-2 focus-visible:ring-[var(--neu-accent-primary)]',
			sizeClasses[size],
			active
				? 'bg-[var(--neu-accent-soft)] text-[var(--neu-accent-primary)] border border-[var(--neu-accent-soft-border)] font-semibold'
				: 'text-[var(--neu-text-muted)]',
			className
		)}
		onclick={onClick}
	>
		{@render children()}
	</button>
{:else}
	<span
		class={cn(
			'rounded-full outline-none transition-colors px-3 py-1',
			sizeClasses[size],
			active
				? 'bg-[var(--neu-accent-soft)] text-[var(--neu-accent-primary)] border border-[var(--neu-accent-soft-border)] font-semibold'
				: 'text-[var(--neu-text-muted)]',
			className
		)}
	>
		{@render children()}
	</span>
{/if}
