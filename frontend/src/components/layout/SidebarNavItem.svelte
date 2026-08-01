<script lang="ts">
	import type { Component } from 'svelte';

	// 桌面侧栏导航项（迁移自旧版 DesktopLayout 内 SidebarNavItem）。
	// hover/active 指示条与背景填充均保留原语义；不使用 neu-hover-btn，
	// 避免矩形导航项的缩放/提亮——hover 反馈完全由下方背景 + 左侧指示条承载。

	let {
		item,
		isActive,
		onClick,
		collapsed = false
	}: {
		item: { icon: Component; label: string };
		isActive: boolean;
		onClick: () => void;
		collapsed?: boolean;
	} = $props();

	const Icon = $derived(item.icon);
	let hovered = $state(false);
	const showHover = $derived(hovered && !isActive);

	// 主题化 hover 背景 — --neu-hover（语义层 --interactive-hover）随
	// 主题（vibrant/forest）与明暗模式自动适配，与 active 的 accent-soft 同族
	const hoverBg = $derived(showHover ? 'var(--neu-hover)' : 'transparent');
</script>

<button
	type="button"
	class="relative flex outline-none text-left {collapsed ? 'h-10 w-10 self-center items-center justify-center rounded-xl' : 'w-full items-center gap-3 rounded-[10px]'}"
	style="padding:{collapsed ? 0 : '9px 12px'};background:{isActive ? (hovered ? 'var(--neu-active-hover)' : 'var(--neu-accent-soft)') : hoverBg};border:{isActive ? `1px solid ${hovered ? 'var(--neu-accent-primary)' : 'var(--neu-accent-soft-border)'}` : '1px solid transparent'};color:{isActive || showHover ? 'var(--neu-accent-primary)' : 'var(--neu-text-muted)'};box-shadow:none;transition:background 0.16s ease, color 0.16s ease, border-color 0.16s ease;"
	aria-label={item.label}
	title={collapsed ? item.label : undefined}
	onclick={onClick}
	onmouseenter={() => (hovered = true)}
	onmouseleave={() => (hovered = false)}
>
	<!-- 左侧指示条 — 仅在 hover 时显示，主题色，细 -->
	<span
		aria-hidden="true"
		style="position:absolute;left:{collapsed ? 2 : 4}px;top:50%;width:3px;height:{isActive ? (collapsed ? 18 : 16) : showHover ? (collapsed ? 14 : 16) : 0}px;margin-top:{isActive ? (collapsed ? -9 : -8) : showHover ? (collapsed ? -7 : -8) : 0}px;border-radius:2px;background:var(--neu-accent-primary);opacity:{isActive ? 1 : showHover ? 0.9 : 0};transition:height 0.18s ease, opacity 0.18s ease, margin-top 0.18s ease;pointer-events:none;"
	></span>
	<Icon
		style="width:18px;height:18px;flex-shrink:0;color:{isActive || showHover ? 'var(--neu-accent-primary)' : 'var(--neu-text-muted)'};transition:color 0.16s ease;"
		strokeWidth={isActive ? 2 : 1.6}
	/>
	<span
		class={collapsed ? 'sr-only' : ''}
		style={collapsed ? '' : 'font-size:14px;font-weight:500;letter-spacing:0.01em;'}
	>
		{item.label}
	</span>
</button>
