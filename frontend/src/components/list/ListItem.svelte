<script lang="ts">
	import type { Snippet } from 'svelte';
	import { cn } from '@/lib/utils';
	import { isInListContext } from './context';

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
	// JS hover 仅键鼠设备启用：触屏首击不闪现 hover 填充/返回后无残留高亮
	const canHover =
		typeof window !== 'undefined' &&
		typeof window.matchMedia === 'function' &&
		window.matchMedia('(hover: hover) and (pointer: fine)').matches;
	// 多边框防线（dev 提示）：joined 行调用方不得再叠卡片类（边框/圆角/阴影），
	// 卡片化请用 tone="card"（独立卡片容器）或 ListContainer bordered（容器级唯一外框）；
	// 静态防线见 eslint-rules/no-list-item-card-classes.mjs（ESLint 规则，已挂入 bun run lint / check / pre-commit）
	// dev 防线跟随 className/style 响应（原实现仅初始化时检测一次）
	$effect(() => {
		if (!import.meta.env.DEV) return;
		void className;
		void style;
		if (
			/\b(?:rounded|border|shadow|ring)[\w-]*/.test(className) ||
			/(?:border|box-shadow|outline)\s*:/.test(style ?? '')
		) {
			console.warn(
				'[ListItem] 调用方叠加了行级卡片样式（边框/圆角/阴影）——joined 行禁止，' +
					'卡片化请用 tone="card" 或 ListContainer bordered'
			);
		}
	});
	// 列表容器内自动包一层 role=listitem（axe aria-required-children：list 的
	// 直接子元素必须为 listitem，交互 button 语义保留在内层；listitem > button
	// 为合法 ARIA 组合）。裸用（独立卡片行等）不包裹，DOM 与既有契约一致。
	const inList = isInListContext();
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

{@render row()}

{#snippet row()}
	<!-- 交互语义在覆盖按钮上；根容器按上下文取语义角色：列表内=listitem
	     （aria-required-children），裸用=group（非静态，允许视觉反馈的指针事件） -->
	<div
		role={inList ? 'listitem' : 'group'}
		aria-label={ariaLabel}
		data-active={pressed ? 'true' : undefined}
		class={cn(
			'relative transition-[background-color,border-radius,box-shadow,opacity] duration-standard ease-standard',
			isCard ? 'rounded-md border' : 'bg-surface-primary',
			// 行内既有的操作按钮/链接/输入相对定位抬升到行覆盖按钮之上（axe
			// nested-interactive：button 语义不得嵌套交互子件）；后写的覆盖按钮
			// 与它们同层级时 DOM 顺序在先，不拦操作点击
			onClick &&
				'[&_button]:relative [&_button]:z-10 [&_a]:relative [&_a]:z-10 [&_input]:relative [&_input]:z-10 [&_select]:relative [&_select]:z-10',
			className
		)}
		style={`${itemStyle}opacity:${itemOpacity};transition-duration:${pointerDown ? 'var(--cst-motion-instant)' : 'var(--cst-motion-standard)'};`}
		onmouseenter={() => canHover && (hovered = true)}
		onmouseleave={() => {
			hovered = false;
			pointerDown = false;
		}}
		onpointerdown={() => onClick && (pointerDown = true)}
		onpointerup={() => (pointerDown = false)}
	>
		{#if onClick}
			<!-- 行点击覆盖层：真实 button 承载点击/键盘（Enter+Space 原生）/aria，
			     内容里的操作按钮经上方 auto-lift 类保持可点——根不再持 button 语义，
			     消除 nested-interactive（button 嵌套交互子件） -->
			<button
				type="button"
				class="absolute inset-0 z-10 cursor-pointer focus-visible:z-10 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-accent-default"
				data-cover
				aria-label={ariaLabel}
				aria-pressed={pressed ? 'true' : undefined}
				onclick={onClick}
				onfocus={() => (focused = true)}
				onblur={() => {
					focused = false;
					pointerDown = false;
				}}
			></button>
		{/if}
		{@render children()}
	</div>
{/snippet}
