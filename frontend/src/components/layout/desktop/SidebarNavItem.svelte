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

	const hoverBg = $derived(showHover ? 'var(--cst-surface-navigation-hover)' : 'transparent');
</script>

<button
	type="button"
	class="tap-985 relative flex h-10 outline-none {collapsed ? 'w-10 self-center items-center justify-center rounded-sm' : 'w-full items-center gap-3 rounded-sm px-3 text-left'}"
	style="background:{isActive ? 'var(--cst-accent-default)' : hoverBg};color:{isActive ? 'var(--cst-content-on-accent)' : showHover ? 'var(--cst-text-primary)' : 'var(--cst-text-secondary)'};box-shadow:{isActive ? 'var(--cst-elevation-navigation-active)' : 'none'};transition:background 0.16s var(--cst-ease-standard), color 0.16s var(--cst-ease-standard), box-shadow 0.2s var(--cst-ease-standard), transform 0.10s var(--cst-ease-standard);"
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
		style="position:absolute;left:4px;top:50%;width:3px;height:{showHover ? 16 : 0}px;margin-top:{showHover ? -8 : 0}px;border-radius:2px;background:var(--cst-accent-default);opacity:{showHover ? 0.9 : 0};transition:height 0.18s ease, opacity 0.18s ease, margin-top 0.18s var(--cst-ease-standard);pointer-events:none;"
	></span>
	<Icon
		style="width:17px;height:17px;shrink:0;color:{isActive ? 'var(--cst-content-on-accent)' : showHover ? 'var(--cst-accent-default)' : 'var(--cst-text-secondary)'};transition:color 0.16s var(--cst-ease-standard);"
		strokeWidth={isActive ? 2 : 1.6}
	/>
	<span
		class={collapsed ? 'sr-only' : `text-md ${isActive ? 'font-medium' : 'font-normal'}`}
	>
		{item.label}
	</span>
</button>
