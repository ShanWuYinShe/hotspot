<script lang="ts">
	import type { Component } from 'svelte';
	import { themeStore } from '@/stores/theme.svelte';

	// 桌面侧栏导航项（迁移自 React DesktopLayout.tsx 内 SidebarNavItem）。
	// hover/active 指示条与背景填充均保留原语义。

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

	const Icon = item.icon;
	let hovered = $state(false);
	const showHover = $derived(hovered && !isActive);

	// 极淡的背景填充 — 用 currentColor 触感的中性灰，避免和主题色冲突
	const hoverBg = $derived(
		showHover ? (themeStore.isDark ? 'rgba(255,255,255,0.045)' : 'rgba(0,0,0,0.035)') : 'transparent'
	);
</script>

<button
	type="button"
	class="neu-hover-btn relative flex outline-none text-left {collapsed ? 'h-10 w-10 self-center items-center justify-center rounded-xl' : 'w-full items-center gap-3 rounded-[10px]'}"
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
	/>
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
