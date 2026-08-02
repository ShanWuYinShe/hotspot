<script lang="ts">
	import type { Snippet } from 'svelte';
	import { cn } from '@/lib/utils';

	// 导航组件（navigation）。
	// InteractiveListRow — 通用交互列表行（对齐 Figma Make InteractiveListRow 契约）：
	//   - tone=joined：连接列表行（--neu-bg 表面，配合 BorderedList/EmbeddedList）
	//   - tone=card  ：独立卡片行（rounded-relaxed + 边框 + 凸起阴影）
	//   - hover：--interactive-hover-fill 背景填充；tap：轻微透明度按压反馈
	//   - 可点击时提供 role=button / 键盘 Enter+Space / tabindex / aria-pressed
	//
	// ┌──────────────── 列表容器 ────────────────┐
	// │ ┌──── InteractiveListRow ─────┐          │
	// │ │ 图标 / 主文本 / 操作          │          │
	// │ └──────────────────────────────┘          │
	// └────────────────────────────────────────────┘
	// 设备范围：桌面、平板和手机通用。

	let {
		children,
		class: className = '',
		tone = 'joined',
		onClick,
		ariaLabel,
		pressed = false,
		style
	}: {
		children: Snippet;
		className?: string;
		tone?: 'joined' | 'card';
		onClick?: () => void;
		ariaLabel?: string;
		pressed?: boolean;
		style?: string;
	} = $props();

	let hovered = $state(false);
	let focused = $state(false);
	const activeSurface = $derived(hovered || focused);

	const isCard = tone === 'card';
	const baseCardShadow = '5px 5px 12px var(--neu-shadow-dark), -5px -5px 12px var(--neu-shadow-light)';
	const rowStyle = $derived.by(() => {
		const parts: string[] = [];
		if (style) parts.push(style);
		if (isCard) {
			parts.push('background:var(--neu-bg);');
			parts.push('border:1px solid var(--neu-border);');
			parts.push(
				`box-shadow:${activeSurface ? `${baseCardShadow}, inset 0 0 0 1px var(--neu-accent-soft-border)` : baseCardShadow};`
			);
		}
		// hover/focus 填充放在最后，确保覆盖行内背景（Figma Make hover-fill 契约）
		if (activeSurface) parts.push('background:var(--interactive-hover-fill);');
		return parts.join('');
	});
</script>

<div
	role={onClick ? 'button' : undefined}
	aria-label={ariaLabel}
	aria-pressed={pressed ? 'true' : undefined}
	data-active={pressed ? 'true' : undefined}
	tabindex={onClick ? 0 : undefined}
	class={cn(
		'relative transition-[background-color,box-shadow,opacity] duration-200 ease-out focus-visible:z-10 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-[var(--neu-accent-primary)]',
		onClick && 'cursor-pointer',
		isCard ? 'rounded-relaxed border' : 'bg-[var(--neu-bg)]',
		className
	)}
	style={rowStyle}
	onmouseenter={() => (hovered = true)}
	onmouseleave={() => (hovered = false)}
	onfocus={() => (focused = true)}
	onblur={() => (focused = false)}
	onclick={onClick}
	onkeydown={
		onClick
			? (e) => {
					// 仅行自身聚焦时响应，避免行内嵌套按钮的键盘事件冒泡触发行导航
					if (e.target !== e.currentTarget) return;
					if (e.key === 'Enter' || e.key === ' ') {
						e.preventDefault();
						onClick();
					}
				}
			: undefined
	}
>
	{@render children()}
</div>
