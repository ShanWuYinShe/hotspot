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
		'overflow-hidden rounded-[1.25rem]',
		'transition-all duration-180 ease-in-out',
		'focus-visible:outline-none focus-visible:ring-4 focus-visible:ring-primary/25 focus-visible:ring-offset-2',
		'disabled:pointer-events-none disabled:opacity-50',
		'cursor-pointer',
		variant === 'default' &&
			'border border-border bg-card text-foreground shadow-[8px_8px_18px_var(--shadow-dark),-8px_-8px_18px_var(--shadow-light)] hover:bg-accent/75 active:shadow-[inset_5px_5px_12px_var(--shadow-dark),inset_-5px_-5px_12px_var(--shadow-light)]',
		variant === 'primary' &&
			'border border-primary/80 bg-primary text-primary-foreground shadow-[8px_8px_18px_var(--shadow-dark),-8px_-8px_18px_var(--shadow-light),0_6px_16px_var(--glow-primary)] hover:brightness-[1.03] active:shadow-[inset_5px_5px_12px_rgba(0,0,0,0.16),inset_-5px_-5px_12px_rgba(255,255,255,0.12)]',
		variant === 'secondary' &&
			'border border-border bg-secondary text-secondary-foreground shadow-[8px_8px_18px_var(--shadow-dark),-8px_-8px_18px_var(--shadow-light)] hover:bg-secondary/88 active:shadow-[inset_5px_5px_12px_var(--shadow-dark),inset_-5px_-5px_12px_var(--shadow-light)]',
		variant === 'ghost' &&
			'border-transparent bg-transparent text-foreground shadow-none hover:bg-muted/70 hover:text-foreground active:bg-muted/90',
		variant === 'icon' && 'rounded-full border border-border/60 bg-card shadow-[8px_8px_18px_var(--shadow-dark),-8px_-8px_18px_var(--shadow-light)]',
		size === 'default' && 'h-11 px-6 py-3 text-sm',
		size === 'sm' && 'h-9 px-4 py-2 text-xs',
		size === 'lg' && 'h-13 px-8 py-4 text-base rounded-[1.5rem]',
		size === 'icon' && 'h-11 w-11 min-h-[44px] min-w-[44px]',
		className
	)}
	{@attach feedbackAction}
	{...restProps}
>
	{#if children}
		{@render children()}
	{/if}
</button>
