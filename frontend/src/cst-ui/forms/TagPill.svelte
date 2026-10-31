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
		sm: 'h-6 px-2.5 text-xxs',
		md: 'h-7 px-3 text-xs'
	} as const;
</script>

{#if onClick}
	<button
		type="button"
		aria-pressed={active}
		disabled={disabled}
		// 注意不用 surface-ghost：它声明 border: solid transparent 且层叠在后，
		// 会覆盖下面的 border-border-default。底色/边框由激活态显式承载。
		class={cn(
			'hit-slop inline-flex items-center justify-center gap-1 rounded-full outline-none leading-none transition-colors cursor-pointer focus-visible:ring-2 focus-visible:ring-accent-default',
			sizeClasses[size],
			disabled && 'pointer-events-none opacity-subtle',
			active
				? 'bg-accent-soft text-text-accent-on-soft border border-accent-border font-semibold active:bg-interactive-active'
				: 'border border-border-default bg-surface-primary text-text-secondary hover:bg-accent-soft',
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
			'inline-flex items-center justify-center gap-1 rounded-full focus-ring leading-none transition-colors',
			sizeClasses[size],
			active
				? 'bg-accent-soft text-text-accent-on-soft border border-accent-border font-semibold'
				: 'border border-border-default bg-surface-primary text-text-secondary',
			className
		)}
	>
		{@render children()}
	</span>
{/if}
