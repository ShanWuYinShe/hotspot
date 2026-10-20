<script lang="ts">
	import type { Component } from 'svelte';
	import UnreadBadge from '@/components/shared/UnreadBadge.svelte';

	// 桌面侧栏导航项：
	// - 选中态：accent-soft 淡底 + accent 图标 + 常驻左侧指示条（克制的「当前位」表达，
	//   替代旧实心 accent 块——大面积实底在窄侧栏里过重）
	// - hover（未选中）：surface-navigation-hover 背景 + 主题文字色 + 左侧细指示条
	// - hover（选中）：底色加深一档（interactive-active）——选中行 hover 仍有可感知增量
	// - 行高固定 40px；图标列固定 40px 居中，折叠时文字随侧栏宽度同步收起
	//   （max-width/opacity/margin 过渡与 aside 宽度过渡同时长，避免图标位移跳变）
	// - badge：可选未读数气泡（展开态随行尾显示；折叠态退化为图标右上角圆点，
	//   保持 40px 图标列不因数字撑宽）

	let {
		item,
		isActive,
		onClick,
		collapsed = false,
		badge = 0
	}: {
		item: { icon: Component; label: string };
		isActive: boolean;
		onClick: () => void;
		collapsed?: boolean;
		badge?: number;
	} = $props();

	const Icon = $derived(item.icon);
	// JS hover 仅在键鼠（fine pointer）设备启用：宽屏触屏点按后不让 hover 预览粘滞
	const canHover =
		typeof window !== 'undefined' &&
		typeof window.matchMedia === 'function' &&
		window.matchMedia('(hover: hover) and (pointer: fine)').matches;
	let hovered = $state(false);
	const showHover = $derived(canHover && hovered);

	// 未选中 hover = 淡底预览；选中 hover = accent-soft 加深为 interactive-active（两态差异）
	const hoverBg = $derived(
		isActive
			? showHover
				? 'var(--cst-interactive-active)'
				: 'var(--cst-accent-soft)'
			: showHover
				? 'var(--cst-surface-navigation-hover)'
				: 'transparent'
	);
	// 指示条**只属于选中态**（2026-10-20 用户截图：hover 项与选中项都有指示条 + accent 文字，
	// 视觉上像"两项同时选中"）。hover 只保留底色预览，选中才有 accent 指示条与 accent 图标。
	const indicatorH = $derived(isActive ? 16 : 0);
</script>

<button
	type="button"
	class="tap-97 relative flex h-10 w-full items-center rounded-sm focus-ring text-left {collapsed ? 'justify-center' : ''}"
	style="background:{hoverBg};color:{isActive || showHover ? 'var(--cst-text-primary)' : 'var(--cst-text-secondary)'};transition:background var(--cst-motion-rapid) var(--cst-ease-standard), color var(--cst-motion-rapid) var(--cst-ease-standard), transform var(--cst-motion-instant) var(--cst-ease-standard);"
	aria-current={isActive ? 'page' : undefined}
	data-active={isActive ? 'true' : undefined}
	aria-label={item.label}
	title={collapsed ? item.label : undefined}
	onclick={onClick}
	onmouseenter={() => (hovered = true)}
	onmouseleave={() => (hovered = false)}
>
	<!-- 左侧指示条 — hover 预览与选中常驻共用：accent 色、3px 圆头 -->
	<span
		aria-hidden="true"
		class="rounded-full {isActive || showHover ? 'opacity-hover' : 'opacity-none'}"
		style="position:absolute;left:4px;top:50%;width:3px;height:{indicatorH}px;margin-top:{indicatorH ? -8 : 0}px;background:var(--cst-accent-default);transition:height var(--cst-motion-snappy) var(--cst-ease-standard), opacity var(--cst-motion-snappy) var(--cst-ease-standard), margin-top var(--cst-motion-snappy) var(--cst-ease-standard);pointer-events:none;"
	></span>
	<span class="relative flex size-10 shrink-0 items-center justify-center">
		<Icon
			class="size-4.5"
			style="color:{isActive ? 'var(--cst-accent-default)' : 'var(--cst-text-secondary)'};transition:color var(--cst-motion-rapid) var(--cst-ease-standard);"
			strokeWidth={isActive ? 2 : 1.5}
		/>
		<!-- 折叠态：数字放不下，退化为图标右上角小圆点（accent 实底，与展开态徽标同源） -->
		{#if collapsed && badge > 0}
			<span
				aria-hidden="true"
				class="absolute right-1.5 top-1.5 size-2 rounded-full bg-accent-default"
			></span>
		{/if}
	</span>
	<span
		class="overflow-hidden whitespace-nowrap text-md transition-[max-width,opacity,margin] duration-standard ease-standard {collapsed ? 'ml-0 max-w-0 opacity-none' : 'ml-1.5 max-w-full opacity-full'} {isActive ? 'font-medium' : 'font-normal'}"
	>
		{item.label}
	</span>
	{#if !collapsed}
		<span class="ml-auto mr-2.5">
			<UnreadBadge count={badge} />
		</span>
	{/if}
</button>
