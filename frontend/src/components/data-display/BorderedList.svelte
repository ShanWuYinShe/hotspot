<script lang="ts">
	import type { Snippet } from 'svelte';
	import { cn } from '@/lib/utils';

	// 数据展示组件（data-display）。
	// 列表容器（对齐 Figma Make ListContainers 契约）——页面中二选一：
	//   BorderedList  ：直接位于开放页面区域的集合（朋友/关注/粉丝/通知等）。
	//                   拥有唯一的圆角外框与 1px 行分隔线，行使用 --surface-primary 表面，
	//                   不得再给每行加第二层卡片边框。
	//   EmbeddedList  ：已被面板/Sheet/rail/分栏容纳的集合（会话列表、地图附近栏等）。
	//                   无外框，仅提供安静的行内分隔线，也不得再给行加卡片边框。
	//
	// ┌──────────────── 页面内容区 ────────────┐
	// │ 分组标题（列表外）                       │
	// │ ┌──── BorderedList / EmbeddedList ────┐ │
	// │ │ 行 1                                │ │
	// │ │ ──────────────────────────────────  │ │
	// │ │ 行 2                                │ │
	// │ └─────────────────────────────────────┘ │
	// └─────────────────────────────────────────┘
	// 设备范围：桌面、平板和手机通用。

	let {
		children,
		class: className = '',
		style: styleProp = '',
		ariaLabel
	}: {
		children: Snippet;
		class?: string;
		style?: string;
		ariaLabel?: string;
	} = $props();
</script>

<!-- BorderedList：唯一外框 + 行分隔（gap-px + 容器底色形成 1px 线） -->
<div
	class={cn('flex flex-col gap-px overflow-hidden rounded-control border', className)}
	style={`background:var(--border-default);border-color:var(--border-default);${styleProp}`}
	aria-label={ariaLabel}
>
	{@render children()}
</div>
