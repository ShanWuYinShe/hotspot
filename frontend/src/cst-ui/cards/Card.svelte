<script lang="ts">
	import type { Snippet } from 'svelte';
	import { cn } from '@/lib/utils';
	import { cardPress } from '@/lib/motion';

	// ═══ CSTUI Card ═══
	// 基础容器卡片组件，支持 surface / flat / elevated / island 变体。

	let {
		variant = 'surface',
		padding = 'md',
		border = true,
		hoverable = false,
		clickable = false,
		onClick,
		children,
		class: className = '',
		ariaLabel
	}: {
		variant?: 'surface' | 'flat' | 'elevated' | 'island';
		padding?: 'none' | 'sm' | 'md' | 'lg';
		border?: boolean;
		hoverable?: boolean;
		clickable?: boolean;
		onClick?: () => void;
		children?: Snippet;
		class?: string;
		ariaLabel?: string;
	} = $props();

	const variantClasses = {
		surface: 'bg-surface-primary',
		flat: 'bg-surface-subtle',
		elevated: 'bg-surface-primary shadow-raised-sm',
		island: 'bg-surface-island'
	};

	const paddingClasses = {
		none: 'p-0',
		sm: 'p-3',
		md: 'p-4',
		lg: 'p-6'
	};
</script>

<!-- svelte-ignore a11y_no_noninteractive_element_interactions -->
<!-- svelte-ignore a11y_no_noninteractive_tabindex -->
<div
	role={clickable ? 'button' : undefined}
	tabindex={clickable ? 0 : undefined}
	aria-label={ariaLabel}
	class={cn(
		'rounded-md transition-all duration-standard',
		variantClasses[variant],
		paddingClasses[padding],
		border && 'border border-border-default',
		hoverable && 'hover-btn',
		clickable && 'cursor-pointer active:scale-[0.99] focus-ring',
		className
	)}
	use:cardPress
	onclick={onClick}
	onkeydown={(e) => {
		if (clickable && (e.key === 'Enter' || e.key === ' ')) {
			e.preventDefault();
			onClick?.();
		}
	}}
>
	{#if children}
		{@render children()}
	{/if}
</div>
