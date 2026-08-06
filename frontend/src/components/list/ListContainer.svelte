<script lang="ts">
	import type { Snippet } from 'svelte';
	import { cn } from '@/lib/utils';

	// 列表骨架组件（list）——列表只有两个公共组件：ListContainer（容器）+ ListItem（行）。
	// ListContainer 按所在区域二选一（对齐 Figma Make ListContainers 契约）：
	//   bordered=false（默认）：嵌入面板/Sheet/rail/分栏的集合（会话列表、地图附近栏等）。
	//                          无外框，仅提供安静的行内分隔线（divide-y）。
	//   bordered=true          ：直接位于开放页面区域的集合（朋友/关注/粉丝/通知等）。
	//                          拥有唯一的圆角外框与 1px 行分隔，行使用 --surface-primary 表面。
	// 两种形态都不得再给每行加第二层卡片边框。
	//
	// ┌──────────────── 页面内容区 ────────────┐
	// │ 分组标题（列表外）                       │
	// │ ┌──── ListContainer（bordered） ──────┐ │
	// │ │ ListItem 1                          │ │
	// │ │ ──────────────────────────────────  │ │
	// │ │ ListItem 2                          │ │
	// │ └─────────────────────────────────────┘ │
	// └─────────────────────────────────────────┘
	// 设备范围：桌面、平板和手机通用。

	let {
		children,
		bordered = false,
		class: className = '',
		style: styleProp = '',
		ariaLabel
	}: {
		children: Snippet;
		/** 有边框形态（开放页面区域）还是无边框形态（面板/Sheet 内部）。 */
		bordered?: boolean;
		class?: string;
		style?: string;
		ariaLabel?: string;
	} = $props();
</script>

{#if bordered}
	<!-- bordered：唯一外框 + 行分隔（gap-px + 容器底色形成 1px 线） -->
	<div
		class={cn('flex flex-col gap-px overflow-hidden rounded-control border', className)}
		style={`background:var(--cst-border-default);border-color:var(--cst-border-default);${styleProp}`}
		aria-label={ariaLabel}
	>
		{@render children()}
	</div>
{:else}
	<!-- embedded：无外框，仅行内分隔线 -->
	<div class={cn('flex flex-col divide-y divide-border-default', className)} aria-label={ariaLabel}>
		{@render children()}
	</div>
{/if}
