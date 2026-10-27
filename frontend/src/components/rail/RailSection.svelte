<script lang="ts">
	import type { Component, Snippet } from 'svelte';
	import SectionHead from '@/components/home/SectionHead.svelte';
	import { cn } from '@/lib/utils';

	// 详情栏通用小节包装（R13：卡头统一走 SectionHead variant="rail"）。
	// variant:
	// - 'card'（默认）：带 surface-flat 边框容器与内衬 padding（保持向后兼容）
	// - 'bare'：去框化纯净排版（无外层边框与内阴影），直接融入侧栏背景，消除 Box-in-a-Box 嵌套杂音
	// - 'island'：现代 Bento 微岛浮动卡片（rounded-2xl 微阴影透光微岛），与主内容区无缝呼应、融为一体
	// 卡头不再手写「标题 + 动作」行：统一由 SectionHead 承载（12px semibold + 可选 icon +
	// 右侧 accent 文字链），全站 rail 小节语言一致。

	let {
		title,
		subtitle,
		icon,
		action,
		onAction,
		variant = 'card',
		class: className = '',
		children
	}: {
		title: string;
		/** 可选副标题（rail 内一句说明，用于「更像入口」的卡片） */
		subtitle?: string;
		icon?: Component;
		action?: string;
		onAction?: () => void;
		variant?: 'card' | 'bare' | 'island';
		class?: string;
		children: Snippet;
	} = $props();

	const containerClass = $derived.by(() => {
		if (variant === 'bare') return 'flex flex-col';
		if (variant === 'island') {
			return 'rounded-2xl bg-surface-primary/70 backdrop-blur-xs p-3.5 shadow-xs hover:shadow-raised-sm transition-all flex flex-col';
		}
		return 'surface-flat rounded-md p-4';
	});
</script>

<div class={cn(containerClass, className)}>
	<SectionHead {title} {subtitle} {icon} {action} {onAction} variant="rail" />
	{@render children()}
</div>
