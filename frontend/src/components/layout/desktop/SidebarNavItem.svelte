<script lang="ts">
	import type { Component } from 'svelte';
	import UnreadBadge from '@/components/shared/UnreadBadge.svelte';

	// 桌面侧栏导航项（R16 无障碍整改：三态「一眼可辨」且不只靠色相）。
	// token 见 styles/theme/semantic.css「桌面侧栏导航三态」；改动须同时满足：
	// - 普通：透明底 + 次级文字/图标（1.5 描边）
	// - hover（未选中）：中性「正文色混画布」底 + 图标/文字升 text-primary + 图标微放大 1.1。
	//   中性灰以**明度差**表达 hover（浅色 1.25:1 / 深色 1.36:1），不抢 accent 色相；
	//   无左侧指示条、无 accent 元素——色弱/色盲/灰度下也不会把 hover 读成「选中」。
	// - 选中：accent 混画布底（比 hover 更强）+ 左侧 3×20px 指示条 + accent 图标（2 描边）
	//   + semibold + 1px accent 内缘描边——颜色、形状、字重四重信号；灰度下靠指示条、
	//   内缘描边与字重可分，不是「只有颜色不同」。
	// - 选中 + hover：底色再加深一档（保留既有「选中行 hover 仍有可感知增量」契约）。
	// - 行高固定 40px；图标列固定 40px 居中，折叠时文字随侧栏宽度同步收起
	//   （max-width/opacity/margin 过渡与 aside 宽度过渡同时长，避免图标位移跳变）。
	// - badge：可选未读数气泡（展开态随行尾显示；折叠态退化为图标右上角圆点，
	//   保持 40px 图标列不因数字撑宽）。
	// - 动效：底色/描边/文字 150ms（--cst-motion-rapid ≤170ms）、图标缩放 150ms，
	//   按压 100ms（tap-97）；prefers-reduced-motion 由 utilities.css 全局规则瞬时化。

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

	// 三态底色（选中 > 选中 hover > 普通 hover > 透明）
	const fill = $derived(
		isActive
			? showHover
				? 'var(--cst-surface-navigation-active-hover)'
				: 'var(--cst-surface-navigation-active)'
			: showHover
				? 'var(--cst-surface-navigation-hover)'
				: 'transparent'
	);
	// 文字：普通 back 到次级；hover/选中升主文字（明度信号）
	const labelColor = $derived(isActive || showHover ? 'var(--cst-text-primary)' : 'var(--cst-text-secondary)');
	// 图标：选中 accent 色，hover 主文字，普通次级（灰度下靠明度与描边粗细）
	const iconColor = $derived(
		isActive ? 'var(--cst-accent-default)' : showHover ? 'var(--cst-text-primary)' : 'var(--cst-text-secondary)'
	);
	// 选中的内缘描边（1px，画在行内部）：灰度下的「形状」信号之二，不是外环；
	// 与 focus-ring 的外描边（outline offset 2）不重叠，键盘焦点仍可辨。
	const borderRing = $derived(isActive ? 'inset 0 0 0 1px var(--cst-accent-border)' : 'none');
	// 左侧指示条只属于选中态（hover 不再显示，避免「两项同时选中」的误读）
	const indicatorH = $derived(isActive ? 20 : 0);
	// hover 时图标微放大 1.1（长在元素本体上的位移反馈，不改变布局）
	const iconScale = $derived(showHover && !isActive ? 1.1 : 1);
</script>

<button
	type="button"
	class="tap-97 relative flex h-10 w-full items-center rounded-sm focus-ring text-left {collapsed
		? 'justify-center'
		: ''}"
	style="background:{fill};box-shadow:{borderRing};color:{labelColor};transition:background var(--cst-motion-rapid) var(--cst-ease-standard), box-shadow var(--cst-motion-rapid) var(--cst-ease-standard), color var(--cst-motion-rapid) var(--cst-ease-standard), transform var(--cst-motion-instant) var(--cst-ease-standard);"
	aria-current={isActive ? 'page' : undefined}
	data-active={isActive ? 'true' : undefined}
	aria-label={item.label}
	title={collapsed ? item.label : undefined}
	onclick={onClick}
	onmouseenter={() => (hovered = true)}
	onmouseleave={() => (hovered = false)}
>
	<!-- 左侧指示条 — 只属于选中态：accent 色、3×20px 圆头（灰度下最直观的形状信号） -->
	<span
		aria-hidden="true"
		class="rounded-full {isActive ? 'opacity-hover' : 'opacity-none'}"
		style="position:absolute;left:4px;top:50%;width:3px;height:{indicatorH}px;margin-top:{indicatorH
			? -10
			: 0}px;background:var(--cst-navigation-active-indicator);transition:height var(--cst-motion-snappy) var(--cst-ease-standard), opacity var(--cst-motion-snappy) var(--cst-ease-standard), margin-top var(--cst-motion-snappy) var(--cst-ease-standard);pointer-events:none;"
	></span>
	<span
		class="relative flex size-10 shrink-0 items-center justify-center"
		style="transform:scale({iconScale});transition:transform var(--cst-motion-rapid) var(--cst-ease-standard);"
	>
		<Icon
			class="size-4.5"
			style="color:{iconColor};transition:color var(--cst-motion-rapid) var(--cst-ease-standard);"
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
		class="overflow-hidden whitespace-nowrap text-md transition-[max-width,opacity,margin] duration-standard ease-standard {collapsed
			? 'ml-0 max-w-0 opacity-none'
			: 'ml-1.5 max-w-full opacity-full'} {isActive ? 'font-semibold' : 'font-normal'}"
	>
		{item.label}
	</span>
	{#if !collapsed}
		<span class="ml-auto mr-2.5">
			<UnreadBadge count={badge} />
		</span>
	{/if}
</button>
