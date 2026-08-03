<script lang="ts">
	import type { Component } from 'svelte';

	// 桌面侧栏导航项（对齐 Figma Make SidebarNavItem 契约）：
	// - 选中态：实心 accent-primary 背景 + content-on-accent 文字 + elevation-navigation-active 光晕
	// - hover（未选中）：surface-navigation-hover 背景 + 主题文字色 + 左侧细指示条
	// - 行高固定 40px；折叠态仅图标居中

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

	const hoverBg = $derived(showHover ? 'var(--surface-navigation-hover)' : 'transparent');
</script>

<button
	type="button"
	class="neu-tap-985 relative flex h-10 outline-none {collapsed ? 'w-10 self-center items-center justify-center rounded-tight' : 'w-full items-center gap-3 rounded-tight px-3 text-left'}"
	style="background:{isActive ? 'var(--neu-accent-primary)' : hoverBg};color:{isActive ? 'var(--content-on-accent)' : showHover ? 'var(--neu-text)' : 'var(--neu-text-muted)'};box-shadow:{isActive ? 'var(--elevation-navigation-active)' : 'none'};transition:background 0.16s ease, color 0.16s ease, box-shadow 0.2s ease;"
	aria-current={isActive ? 'page' : undefined}
	data-active={isActive ? 'true' : undefined}
	aria-label={item.label}
	title={collapsed ? item.label : undefined}
	onclick={onClick}
	onmouseenter={() => (hovered = true)}
	onmouseleave={() => (hovered = false)}
>
	<!-- 左侧指示条 — 仅在 hover 时显示，主题色，细（选中态由实心填充表达） -->
	<span
		aria-hidden="true"
		style="position:absolute;left:4px;top:50%;width:3px;height:{showHover ? 16 : 0}px;margin-top:{showHover ? -8 : 0}px;border-radius:2px;background:var(--neu-accent-primary);opacity:{showHover ? 0.9 : 0};transition:height 0.18s ease, opacity 0.18s ease, margin-top 0.18s ease;pointer-events:none;"
	></span>
	<Icon
		style="width:17px;height:17px;flex-shrink:0;color:{isActive ? 'var(--content-on-accent)' : showHover ? 'var(--neu-accent-primary)' : 'var(--neu-text-muted)'};transition:color 0.16s ease;"
		strokeWidth={isActive ? 2 : 1.6}
	/>
	<span
		class={collapsed ? 'sr-only' : ''}
		style={collapsed ? '' : `font-size:14px;font-weight:${isActive ? 500 : 400};letter-spacing:0.01em;`}
	>
		{item.label}
	</span>
</button>
