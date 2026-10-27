<script lang="ts">
	import type { Component, Snippet } from 'svelte';
	import { ChevronRight } from '@lucide/svelte';
	import { cn } from '@/lib/utils';

	// 全站分区标题原语（R13 起从 home/ 提升为跨页唯一实现）。
	// 结构：可选图标 + 标题 + 可选计数 + 副标题，右侧统一「查看全部」文字链。
	// variant：
	// - section（默认）：页面正文分区标题（14px semibold + mb-3）
	// - rail：右侧 rail 卡片内小标题（11px semibold + mb-2.5，克制的侧栏语气）
	// - eyebrow：大写 eyebrow 风格（保留给表单/分组标签类分区，非必需）
	// 页面不再手写「标题 + 查看全部」行；需要新形态时扩展本组件 prop。

	let {
		title,
		subtitle,
		icon: Icon,
		count,
		action,
		onAction,
		variant = 'section',
		class: className = ''
	}: {
		title: string;
		subtitle?: string;
		icon?: Component;
		count?: number | string;
		action?: string;
		onAction?: () => void;
		variant?: 'section' | 'rail' | 'eyebrow';
		class?: string;
	} = $props();

	const titleCls = $derived(
		variant === 'rail'
			? 'text-xs font-semibold text-text-primary'
			: variant === 'eyebrow'
				? 'eyebrow'
				: 'text-md font-semibold text-text-primary'
	);
	const wrapperCls = $derived(
		cn(
			'flex min-w-0 items-center justify-between gap-3',
			variant === 'rail' ? 'mb-2.5' : 'mb-3',
			className
		)
	);
</script>

<div class={wrapperCls}>
	<div class="flex min-w-0 items-center gap-2">
		{#if Icon}
			<span class="flex shrink-0 items-center justify-center text-text-secondary" aria-hidden="true">
				<Icon class="size-4" strokeWidth={1.75} />
			</span>
		{/if}
		<div class="min-w-0">
			<div class="flex min-w-0 items-baseline gap-2">
				<h2 class={cn('truncate', titleCls)}>{title}</h2>
				{#if count !== undefined}
					<span class="shrink-0 text-xs font-medium tabular-nums text-text-secondary">{count}</span>
				{/if}
			</div>
			{#if subtitle}
				<p class="mt-1 text-xs text-text-secondary">{subtitle}</p>
			{/if}
		</div>
	</div>
	{#if action && onAction}
		<button
			type="button"
			class="link-btn hit-slop-44 -my-2.5 flex shrink-0 items-center gap-0.5 px-1 py-2.5 text-xs text-accent-default focus-ring"
			onclick={onAction}
		>
			<span>{action}</span>
			<ChevronRight class="size-3" strokeWidth={1.5} />
		</button>
	{/if}
</div>
