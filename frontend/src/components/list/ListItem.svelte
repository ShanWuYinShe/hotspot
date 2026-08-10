<script lang="ts">
	import type { Snippet } from 'svelte';
	import { cn } from '@/lib/utils';

	// 列表骨架组件（list）——列表只有两个公共组件：ListContainer（容器）+ ListItem（行）。
	// ListItem 是唯一交互行组件（对齐 Figma Make InteractiveListRow 契约）：
	//   - tone=joined：连接列表行（--surface-primary 表面，配合 ListContainer 使用）
	//   - tone=card  ：独立卡片行（rounded-md + 边框 + 凸起阴影）
	//   - hover：--interactive-hover-fill 背景填充；tap：轻微透明度按压反馈
	//   - pressed：选中态（accent 全背景 + on-accent 文字）；children 需按选中态
	//     切换文字类（text-text-on-accent 系列 + opacity-soft/80 派生次级文字，
	//     随明暗模式自动切换深浅；禁止用恒白 text-content-inverse——那是
	//     图片/遮罩上的反色 token，在 dark 浅色 accent 背景上对比度不足）
	//   - 高亮形状：joined 行 hover/focus/pressed 时背景带 rounded-sm 圆角
	//     （非高亮保持直角连接行视觉；圆角由组件统一应用，调用方不得叠加）
	//   - 可点击时提供 role=button / 键盘 Enter+Space / tabindex / aria-pressed
	//
	// 使用方式：列表的交互已全部在此定义，调用方只需向 children 填充内容。
	//
	// ┌──────────────── 列表容器 List ────────┐
	// │ ┌──── ListItem ────────────┐          │
	// │ │ 图标 / 主文本 / 操作      │          │
	// │ └──────────────────────────┘          │
	// └────────────────────────────────────────┘
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
		class?: string;
		tone?: 'joined' | 'card';
		onClick?: () => void;
		ariaLabel?: string;
		/** 选中态（accent 全背景 + 反色）。行自身渲染 aria-pressed/data-active。 */
		pressed?: boolean;
		style?: string;
	} = $props();

	let hovered = $state(false);
	let focused = $state(false);
	let pointerDown = $state(false);
	// 多边框防线（dev 提示）：joined 行调用方不得再叠卡片类（边框/圆角/阴影），
	// 卡片化请用 tone="card"（独立卡片容器）或 ListContainer bordered（容器级唯一外框）；
	// 静态防线见 eslint-rules/no-list-item-card-classes.mjs（ESLint 规则，已挂入 bun run lint / check / pre-commit）
	const checkRowCardStyles = () => {
		if (!import.meta.env.DEV) return;
		if (
			/\b(?:rounded|border|shadow|ring)[\w-]*/.test(className) ||
			/(?:border|box-shadow|outline)\s*:/.test(style ?? '')
		) {
			console.warn(
				'[ListItem] 调用方叠加了行级卡片样式（边框/圆角/阴影）——joined 行禁止，' +
					'卡片化请用 tone="card" 或 ListContainer bordered'
			);
		}
	};
	checkRowCardStyles();
	// hover 或 focus 时行处于「激活表面」：叠加 hover-fill 背景
	const surfaceActive = $derived(hovered || focused);
	// 对齐 Figma Make：可点击时 tap → opacity-tap 档（0.1s），无 scale
	const itemOpacity = $derived(onClick && pointerDown ? 'var(--cst-opacity-tap)' : 'var(--cst-opacity-full)');

	const isCard = $derived(tone === 'card');
	const cardShadow = 'var(--shadow-raised-md)';
	// joined 行高亮（hover/focus/pressed）时背景呈圆角矩形；非高亮保持直角
	// 连接行视觉。圆角由组件统一应用（token 档位 rounded-sm），调用方不得叠加。
	// card 行的圆角由 className（rounded-md）承载，inline 不覆盖。
	const itemStyle = $derived.by(() => {
		const parts: string[] = [];
		if (style) parts.push(style);
		if (isCard) {
			parts.push('background:var(--cst-surface-primary);');
			parts.push('border:var(--cst-border-width-hairline) solid var(--cst-border-default);');
			parts.push(
				`box-shadow:${surfaceActive ? `${cardShadow}, inset 0 0 0 1px var(--cst-accent-border)` : cardShadow};`
			);
		} else if (pressed || surfaceActive) {
			// 高亮圆角：只影响背景填充，不影响布局
			parts.push('border-radius:var(--cst-shape-sm);');
		}
		// 选中态：accent 全背景（inline style 优先，hover 不再叠加填充）
		if (pressed) {
			parts.push('background:var(--cst-accent-default);');
		} else if (surfaceActive) {
			// hover/focus 填充放在最后，确保覆盖行内背景（hover-fill 契约）
			parts.push('background:var(--cst-interactive-hover-fill);');
		}
		return parts.join('');
	});
</script>

<!-- svelte-ignore a11y_no_noninteractive_tabindex -->
<div
	role={onClick ? 'button' : undefined}
	aria-label={ariaLabel}
	aria-pressed={pressed ? 'true' : undefined}
	data-active={pressed ? 'true' : undefined}
	tabindex={onClick ? 0 : undefined}
	class={cn(
		'relative transition-[background-color,border-radius,box-shadow,opacity] duration-standard ease-out focus-visible:z-10 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-accent-default',
		onClick && 'cursor-pointer',
		isCard ? 'rounded-md border' : 'bg-surface-primary',
		className
	)}
	style={`${itemStyle}opacity:${itemOpacity};transition-duration:${pointerDown ? 'var(--cst-motion-instant)' : 'var(--cst-motion-standard)'};`}
	onmouseenter={() => (hovered = true)}
	onmouseleave={() => {
		hovered = false;
		pointerDown = false;
	}}
	onfocus={() => (focused = true)}
	onblur={() => {
		focused = false;
		pointerDown = false;
	}}
	onpointerdown={() => onClick && (pointerDown = true)}
	onpointerup={() => (pointerDown = false)}
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
