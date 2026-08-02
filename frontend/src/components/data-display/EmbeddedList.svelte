<script lang="ts">
	import type { Snippet } from 'svelte';
	import { cn } from '@/lib/utils';

	// 数据展示组件（data-display）。
	// EmbeddedList — 已被面板/Sheet/rail/分栏容纳的集合（会话列表、地图附近栏等）。
	// 无外框，仅提供安静的行内分隔线（divide-y）。
	// 对齐 Figma Make ListContainers 契约，不得再给每行加第二层卡片边框。

	let {
		children,
		class: className = ''
	}: {
		children: Snippet;
		class?: string;
	} = $props();
</script>

<div class={cn('flex flex-col divide-y divide-[var(--neu-border)]', className)}>
	{@render children()}
</div>
