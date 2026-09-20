<script lang="ts">
	import type { Component } from 'svelte';
	import UnreadBadge from '@/components/shared/UnreadBadge.svelte';

	// ActionIconButton — 圆形图标按钮（对齐 Figma Make ActionIconButton 契约）。
	// variant: surface（凸起）/ overlay（图片覆盖层 blur）/ ghost（透明）。
	// tone: muted / accent / inverse 决定图标颜色；hover/tap 由 scale-btn 承载。
	// Figma 无 active 态、无 strokeWidth 变化；焦点环由 focus-ring 统一承载。
	// 主题差异（vibrant 浮起投影 / forest·inkwash 纸面压平）由 themes/*.css 的
	// .action-icon-btn 覆盖承载，本组件不按主题分支。
	// badge：可选未读数气泡，固定悬挂在按钮右上角（如顶栏「通知」铃铛）。

	let {
		icon: Icon,
		label,
		onClick,
		variant = 'surface',
		tone = 'muted',
		size = 36,
		disabled = false,
		type = 'button',
		badge = 0,
		class: className = ''
	}: {
		icon: Component;
		label: string;
		onClick?: () => void;
		variant?: 'surface' | 'overlay' | 'ghost';
		tone?: 'muted' | 'accent' | 'inverse';
		size?: 32 | 34 | 36 | 42;
		disabled?: boolean;
		type?: 'button' | 'submit' | 'reset';
		badge?: number;
		class?: string;
	} = $props();

	const iconColor = $derived(
		tone === 'inverse' ? 'var(--cst-content-inverse)' : tone === 'accent' ? 'var(--cst-accent-default)' : 'var(--cst-content-secondary)'
	);

	const style = $derived.by(() => {
		if (variant === 'overlay') {
			return `width:${size}px;height:${size}px;background:var(--cst-surface-overlay);backdrop-filter:blur(var(--cst-blur-md));border:var(--cst-border-width-hairline) solid var(--cst-border-inverse);`;
	} else if (variant === 'ghost') {
		// 不写 inline background：默认即透明，写死会以 inline 优先级压掉
		// header-action 等类的 hover 背景填充
		return `width:${size}px;height:${size}px;border:var(--cst-border-width-hairline) solid transparent;`;
	}
		return `width:${size}px;height:${size}px;background:var(--cst-surface-raised);border:var(--cst-border-width-hairline) solid var(--cst-border-subtle);box-shadow:var(--shadow-raised-sm);`;
	});
</script>

<button
	type={type}
	class="action-icon-btn hover-btn scale-btn hit-slop relative flex items-center justify-center rounded-full focus-ring shrink-0 disabled:pointer-events-none disabled:cursor-not-allowed disabled:opacity-muted {className}"
	style={style}
	aria-label={label}
	title={label}
	onclick={() => onClick?.()}
	{disabled}
>
	<Icon size={Math.round(size * 0.42)} color={iconColor} strokeWidth={1.75} />
	{#if badge > 0}
		<span class="absolute -right-1 -top-1 z-1">
			<UnreadBadge count={badge} />
		</span>
	{/if}
</button>
