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

	const shouldUseCircular = $derived(circular || variant === 'icon');
	const feedbackAction = $derived(shouldUseCircular ? circularFeedback : rippleFeedback);
</script>

<button
	class={cn(
		'relative inline-flex items-center justify-center gap-2',
		'whitespace-nowrap font-semibold',
		'overflow-hidden rounded-[22px]',
		'transition-all duration-180 ease-in-out',
		'focus-visible:outline-none',
		'disabled:pointer-events-none disabled:opacity-50',
		'cursor-pointer',
		className
	)}
	style="background: {variant === 'ghost' ? 'transparent' : 'var(--neu-bg)'}; color: var(--neu-text); border: 1px solid {variant === 'ghost' ? 'transparent' : 'var(--neu-border)'}; box-shadow: {variant === 'ghost' ? 'none' : variant === 'primary' ? '6px 6px 14px var(--neu-shadow-dark), -6px -6px 14px var(--neu-shadow-light), 0 4px 12px var(--neu-glow)' : '6px 6px 14px var(--neu-shadow-dark), -6px -6px 14px var(--neu-shadow-light)'}; {variant === 'primary' ? 'color: var(--neu-active-text);' : ''}"
	{@attach feedbackAction}
	{...restProps}
>
	{#if children}
		{@render children()}
	{/if}
</button>
