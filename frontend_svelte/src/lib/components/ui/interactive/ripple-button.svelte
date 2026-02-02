<script lang="ts">
	import { circularFeedback, rippleFeedback } from '$lib/attachments';
	import { cn } from '$lib/utils';
	import type { Snippet } from 'svelte';
	import type { HTMLButtonAttributes } from 'svelte/elements';

	interface Props extends HTMLButtonAttributes {
		variant?: 'default' | 'ghost' | 'icon' | 'primary' | 'secondary';
		size?: 'default' | 'sm' | 'lg' | 'icon';
		circular?: boolean;
		children?: Snippet;
	}

	let {
		variant = 'default',
		size = 'default',
		circular = false,
		class: className,
		children,
		...restProps
	}: Props = $props();

	// 自动判断：icon 变体默认使用圆形反馈（使用 $derived 确保响应式）
	const shouldUseCircular = $derived(circular || variant === 'icon');
	
	// 根据条件选择action（使用 $derived 确保响应式）
	const feedbackAction = $derived(shouldUseCircular ? circularFeedback : rippleFeedback);
</script>

<button
	class={cn(
		'relative inline-flex items-center justify-center gap-2',
		'whitespace-nowrap font-semibold',
		'rounded-xl overflow-hidden',
		'transition-all duration-150 ease-in-out',
		'focus-visible:outline-none focus-visible:ring-4 focus-visible:ring-primary/40 focus-visible:ring-offset-2',
		'disabled:pointer-events-none disabled:opacity-50',
		'cursor-pointer',
		// Variants
		variant === 'default' &&
			'bg-background border-2 border-border text-foreground hover:bg-accent hover:border-accent',
		variant === 'primary' &&
			'bg-primary border-2 border-primary text-primary-foreground shadow-md hover:shadow-lg',
		variant === 'secondary' &&
			'bg-secondary border-2 border-secondary text-secondary-foreground hover:bg-secondary/80',
		variant === 'ghost' &&
			'border-transparent text-foreground hover:bg-primary/10 hover:text-primary',
		variant === 'icon' && 'rounded-full border-transparent',
		// Sizes
		size === 'default' && 'h-11 px-6 py-3 text-sm',
		size === 'sm' && 'h-9 px-4 py-2 text-xs',
		size === 'lg' && 'h-13 px-8 py-4 text-base rounded-2xl',
		size === 'icon' && 'h-11 w-11 min-h-[44px] min-w-[44px]',
		className
	)}
	use:feedbackAction
	{...restProps}
>
	{#if children}
		{@render children()}
	{/if}
</button>
