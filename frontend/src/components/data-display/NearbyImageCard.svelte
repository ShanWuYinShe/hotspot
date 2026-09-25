<script lang="ts">
	import type { Snippet } from 'svelte';
	import { cn } from '@/lib/utils';

	// 数据展示组件（data-display）。
	// NearbyImageCard — 图片优先的附近内容卡片统一原语。
	// 对齐 Figma Make data-display/NearbyImageCard 契约（Guidelines 标记为附近内容的唯一卡片实现）：
	//   - 图片 object-cover 填满卡片，底部 tray 承载元数据/操作（不用页面色块垫在图片下）
	//   - hover 与内容卡统一 card-hover 抬升语言（投影+上浮 / forest 纸面微染+边框），
	//     内部图片随之微妙缩放（image-hover-frame__image）
	//   - 底部 tray 用 --surface-card-menu + --border-card-menu，不用裸页面背景
	// 替代 PostCard / FeaturedGroupCard / GroupCard / UserCard 内联重复实现。
	//
	// ┌──────────────── 页面内容区 ────────────┐
	// │ ┌──────── NearbyImageCard ──────┐         │
	// │ │ ┌───────── image ─────────┐   │         │
	// │ │ │  (object-cover)         │   │         │
	// │ │ │  children overlay       │   │         │
	// │ │ └─────────────────────────┘   │         │
	// │ │ ┌── footer tray (card-menu) ─┐│         │
	// │ │ │ 元数据 / 操作              ││         │
	// │ │ └────────────────────────────┘│         │
	// │ └───────────────────────────────┘         │
	// └─────────────────────────────────────────────┘
	// 设备范围：桌面、平板和手机通用。

	let {
		imageSrc,
		imageAlt = '',
		onClick,
		onImageError,
		placeholder,
		footer,
		children,
		class: className = ''
	}: {
		imageSrc: string;
		imageAlt?: string;
		onClick?: () => void;
		onImageError?: () => void;
		/** 无图时的占位内容（渲染在 accent 渐变层上；缺省为空白渐变层） */
		placeholder?: Snippet;
		footer?: Snippet;
		children?: Snippet;
		class?: string;
	} = $props();

	// hover 与内容卡（SectionGroups/SectionMoments）统一走 card-hover 抬升语言
	// （投影+上浮，forest 纸面微染+边框），图片微缩放由 .card-hover:hover
	// .image-hover-frame__image 规则驱动（CSS :hover 单一事实源——JS 状态与
	// :hover 不同步会导致鼠标滑动时反馈闪烁/丢失）。键盘焦点由 focus-ring 承载。
	// 整卡按压 scale 由 --cst-card-tap-scale 驱动（vibrant 0.94 / forest 0.97，
	// 对齐 Figma useHoverConfig tap）；未按压时不写 inline transform，
	// 让 card-hover 的 hover translateY 生效。
	let pressed = $state(false);
	const cardTransform = $derived(pressed ? 'scale(var(--cst-card-tap-scale, 0.94))' : '');

	// 图片加载占位（审查 M4）：图片型信息流首刷在弱网下是「白卡跳动」。
	// 加载完成前显示 accent 渐变层兜底，onload 后渐显图片，避免空白闪现。
	let imageLoaded = $state(false);
	$effect(() => {
		// src 变化（翻页/复用）后回到占位态
		void imageSrc;
		imageLoaded = false;
	});

	// 行内子按钮（点赞/收藏/评论等）的按压由按钮自身的 hover-action/tap 承载，
	// 不带动整卡缩放——否则点按钮时整卡一起缩，看起来像误触了卡片。
	function pressedFromEvent(e: PointerEvent): boolean {
		// 覆盖按钮（.card-cover）= 卡片本体，按压带动整卡；行内操作按钮不带动
		const btn = (e.target as HTMLElement).closest('button');
		return !btn || btn.hasAttribute('data-cover');
	}
</script>

<!-- svelte-ignore a11y_no_noninteractive_tabindex -->
<article
	class={cn(
		'card-hover relative h-full w-full overflow-hidden border border-border-default',
		onClick &&
			// button 抬升必须排除覆盖按钮自身(:not([data-cover])):否则 [&_button]:relative
			// 与它的 absolute 同 specificity,且工具类生成顺序 .relative 在 .absolute 之后,
			// 覆盖按钮塌缩 0×0,整卡点击失效(线上实测 P0,同 ListItem);静态防线见
			// eslint-rules/no-cover-button-collapse.mjs
			'[&_button:not([data-cover])]:relative [&_button:not([data-cover])]:z-20 [&_a]:relative [&_a]:z-20',
		className
	)}
	style={cardTransform ? `transform:${cardTransform};transition:transform var(--cst-motion-instant) var(--cst-ease-standard);` : ''}
	onpointerdown={(e) => {
		if (pressedFromEvent(e)) pressed = true;
	}}
	onpointerup={() => (pressed = false)}
	onpointerleave={() => (pressed = false)}
>
	{#if imageSrc}
		<!-- 加载占位：onload 前垫 accent 渐变（pointer-events-none 不拦行内按钮） -->
		{#if !imageLoaded}
			<div class="surface-accent-gradient pointer-events-none absolute inset-0 z-[1] animate-pulse"></div>
		{/if}
		<img
			src={imageSrc}
			alt={imageAlt}
			loading="lazy"
			class="image-hover-frame__image absolute inset-0 h-full w-full object-cover will-change-transform transition-opacity duration-standard {imageLoaded ? 'opacity-100' : 'opacity-0'}"
			onload={() => (imageLoaded = true)}
			onerror={onImageError}
		/>
	{:else if placeholder}
		<!-- 装饰占位层：pointer-events-none 防止拦截 footer 行内按钮（关注等）的点击/悬停 -->
		<div class="surface-accent-gradient pointer-events-none absolute inset-0 z-[1] flex items-center justify-center">
			{@render placeholder()}
		</div>
	{/if}
	{#if imageSrc}
		<div class="pointer-events-none absolute inset-0" style="background:var(--cst-surface-media-scrim-card);"></div>
	{/if}
	{#if children}
		{@render children()}
	{/if}
	{#if footer}
		<div
			class="absolute inset-x-0 bottom-0 border-t border-border-card-menu bg-surface-card-menu"
		>
			{@render footer()}
		</div>
	{/if}
	{#if onClick}
		<!-- 卡片点击覆盖层（最后绘制，z-10 高于图片/遮罩/z-[1] 占位层/footer 非按钮区）：
		     真实 button 承载点击/键盘/aria，行内操作按钮经根上 auto-lift(z-20) 保持可点——
		     根不再持 button 语义，消除 nested-interactive（button 嵌套交互子件） -->
		<button
			type="button"
			class="absolute inset-0 z-10 cursor-pointer focus-ring"
			data-cover
			aria-label={imageAlt || undefined}
			onclick={onClick}
		></button>
	{/if}
</article>
