<script lang="ts">
	import type { Component, Snippet } from 'svelte';
	import { cn } from '@/lib/utils';

	let {
		children,
		active = false,
		onClick,
		icon,
		iconSnippet,
		disabled = false,
		size = 'md',
		class: className = ''
	}: {
		children: Snippet;
		active?: boolean;
		onClick?: () => void;
		/** 可选前置图标（与文字同色，随激活态变色） */
		icon?: Component;
		/** 可选前置自定义内容（如彩色圆点/图钉 SVG），优先级高于 icon */
		iconSnippet?: Snippet;
		disabled?: boolean;
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
		disabled={disabled}
		class={cn(
			'rounded-full outline-none transition-colors px-3 py-1 cursor-pointer surface-ghost focus-visible:ring-2 focus-visible:ring-accent-default',
			sizeClasses[size],
			disabled && 'pointer-events-none opacity-subtle',
			active
				? 'bg-accent-soft text-accent-default border border-accent-border font-semibold'
				: 'text-text-secondary',
			className
		)}
		onclick={onClick}
	>
		{#if iconSnippet}
			{@render iconSnippet()}
		{:else if icon}
			<icon class="size-3.5" strokeWidth={active ? 2 : 1.5}></icon>
		{/if}
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
