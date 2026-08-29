<script lang="ts">
	import type { Snippet } from 'svelte';
	import type { HTMLButtonAttributes } from 'svelte/elements';
	import { cn } from '@/lib/utils';

	// Button — 全站统一按钮原语（components/ui）。
	// variant：primary（accent 实底）/ secondary（面板描边）/ outline（透明描边）/
	//          ghost（透明）/ danger（状态色实底）。
	// size：sm=32 / md=40 / lg=44 高度，字号与内边距对齐既有页面按钮。
	// loading：显示统一 spinner + aria-busy，并阻止重复提交。
	// 焦点环由 focus-ring 统一提供；hover/tap 由 hover-btn 承载。
	// 调用方 className 经 tailwind-merge 合并，可覆盖圆角/宽度等布局类。

	type ButtonVariant = 'primary' | 'secondary' | 'outline' | 'ghost' | 'danger';
	type ButtonSize = 'sm' | 'md' | 'lg';

	// hover 反馈走组件级 token（全局 button hover 填充只兜底透明裸按钮）：
	// primary = accent 光晕抬档；secondary/outline = 描边加深；ghost = 淡底填充。
	const VARIANTS: Record<ButtonVariant, string> = {
		primary: 'bg-accent-default text-text-on-accent shadow-glow-sm hover:shadow-glow-md',
		secondary: 'bg-surface-primary text-text-primary border border-border-default shadow-raised-sm hover:border-border-strong',
		outline: 'bg-transparent text-text-primary border border-border-default hover:border-border-strong',
		ghost: 'bg-transparent text-text-primary hover:bg-interactive-hover',
		danger: 'bg-status-danger text-text-on-status'
	};

	const SIZES: Record<ButtonSize, string> = {
		sm: 'h-8 px-3 text-sm rounded-md',
		md: 'h-10 px-4 text-sm rounded-md',
		lg: 'h-11 px-5 text-sm rounded-md'
	};

	let {
		children,
		variant = 'primary',
		size = 'md',
		loading = false,
		class: className = '',
		type = 'button',
		disabled = false,
		...rest
	}: HTMLButtonAttributes & {
		children: Snippet;
		variant?: ButtonVariant;
		size?: ButtonSize;
		loading?: boolean;
	} = $props();
</script>

<button
	{type}
	{disabled}
	class={cn(
		'inline-flex shrink-0 select-none items-center justify-center gap-2 whitespace-nowrap font-medium hover-btn focus-ring transition-colors',
		VARIANTS[variant],
		SIZES[size],
		loading && 'pointer-events-none opacity-soft',
		className
	)}
	aria-busy={loading || undefined}
	{...rest}
>
	{#if loading}
		<span class="spinner size-4" aria-hidden="true"></span>
	{/if}
	<span class="inline-flex items-center gap-1.5 {loading ? 'opacity-emphasis' : ''}">{@render children()}</span>
</button>
