<script lang="ts">
	import type { Snippet } from 'svelte';
	import { ChevronRight } from '@lucide/svelte';
	import { cn } from '@/lib/utils';

	// 详情栏通用小节包装。
	// variant:
	// - 'card'（默认）：带 surface-flat 边框容器与内衬 padding（保持向后兼容）
	// - 'bare'：去框化纯净排版（无外层边框与内阴影），直接融入侧栏背景，消除 Box-in-a-Box 嵌套杂音
	// - 'island'：现代 Bento 微岛浮动卡片（rounded-2xl 微阴影透光微岛），与主内容区无缝呼应、融为一体

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
		variant?: 'card' | 'bare' | 'island';
		class?: string;
		children: Snippet;
	} = $props();

	const containerClass = $derived.by(() => {
		if (variant === 'bare') return 'flex flex-col';
		if (variant === 'island') {
			return 'rounded-2xl border border-border-subtle/50 bg-surface-primary/80 backdrop-blur-xs p-4 shadow-xs hover:border-border-default/80 transition-all flex flex-col';
		}
		return 'surface-flat rounded-md p-4';
	});
</script>

<div class={cn(containerClass, className)}>
	<div class="flex items-end justify-between mb-3">
		<span class={variant === 'card' ? 'eyebrow' : 'text-xs font-semibold text-text-primary tracking-tight'}>{title}</span>
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
