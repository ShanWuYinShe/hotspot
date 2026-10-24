<script lang="ts">
	import type { Snippet } from 'svelte';
	import { ChevronRight } from '@lucide/svelte';
	import { cn } from '@/lib/utils';

	// 详情栏通用小节包装。
	// variant:
	// - 'card'（默认）：带 surface-flat 边框容器与内衬 padding（保持向后兼容）
	// - 'bare'：去框化纯净排版（无外层边框与内阴影），直接融入侧栏背景，消除 Box-in-a-Box 嵌套杂音

	let {
		title,
		action,
		onAction,
		variant = 'card',
		class: className = '',
		children
	}: {
		title: string;
		action?: string;
		onAction?: () => void;
		variant?: 'card' | 'bare';
		class?: string;
		children: Snippet;
	} = $props();
</script>

<div class={cn(variant === 'card' ? 'surface-flat rounded-md p-4' : 'flex flex-col', className)}>
	<div class="flex items-end justify-between mb-3">
		<span class={variant === 'bare' ? 'text-xs font-semibold text-text-primary tracking-tight' : 'eyebrow'}>{title}</span>
		{#if action && onAction}
			<button
				type="button"
				class="link-btn hit-slop flex items-center gap-0.5 focus-ring text-accent-default"
				onclick={onAction}
			>
				<span class="text-xs">{action}</span>
				<ChevronRight class="size-3" strokeWidth={1.5} />
			</button>
		{/if}
	</div>
	{@render children()}
</div>
